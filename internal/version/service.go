package version

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"project-atlas/internal/metadata"
	"project-atlas/internal/storage"
)

type Service struct {
	repo  metadata.Repository
	store storage.ChunkStore
}

func NewService(repo metadata.Repository, store storage.ChunkStore) *Service {
	return &Service{
		repo:  repo,
		store: store,
	}
}

func (s *Service) ListVersions(ctx context.Context, objectID uuid.UUID) ([]metadata.Version, error) {
	return s.repo.ListVersions(ctx, objectID)
}

func (s *Service) GetVersion(ctx context.Context, versionID uuid.UUID) (metadata.Version, error) {
	return s.repo.GetVersion(ctx, versionID)
}

func (s *Service) DeleteVersion(ctx context.Context, objectID uuid.UUID, versionID uuid.UUID) error {
	// First check if it's the current version
	obj, err := s.repo.GetObject(ctx, objectID)
	if err != nil {
		return err
	}

	if obj.CurrentVersionID != nil && *obj.CurrentVersionID == versionID {
		return fmt.Errorf("CANNOT_DELETE_CURRENT_VERSION") // Let API layer map this to error
	}

	chunksToDelete, err := s.repo.GetChunksToDeleteAfterVersionDelete(ctx, versionID)
	if err != nil {
		return fmt.Errorf("delete version in db: %w", err)
	}

	for _, hash := range chunksToDelete {
		if err := s.store.Delete(hash); err != nil {
			slog.Error("failed to delete chunk from disk after version delete", "hash", hash, "error", err)
		}
	}
	slog.Info("version deleted", "object_id", objectID, "version_id", versionID)
	return nil
}

func (s *Service) DeleteObject(ctx context.Context, objectID uuid.UUID) error {
	chunksToDelete, err := s.repo.GetChunksToDeleteAfterWholeObjectDelete(ctx, objectID)
	if err != nil {
		return fmt.Errorf("delete object in db: %w", err)
	}

	for _, hash := range chunksToDelete {
		if err := s.store.Delete(hash); err != nil {
			slog.Error("failed to delete chunk from disk after object delete", "hash", hash, "error", err)
		}
	}
	slog.Info("object deleted", "object_id", objectID)
	return nil
}

func (s *Service) Rollback(ctx context.Context, objectID uuid.UUID, targetVersionID uuid.UUID) (metadata.Version, error) {
	// Validate target version
	targetVer, err := s.repo.GetVersion(ctx, targetVersionID)
	if err != nil {
		return metadata.Version{}, fmt.Errorf("get target version: %w", err)
	}

	if targetVer.ObjectID != objectID {
		return metadata.Version{}, fmt.Errorf("INVALID_ROLLBACK_TARGET")
	}

	if targetVer.Status != "committed" {
		return metadata.Version{}, fmt.Errorf("INVALID_ROLLBACK_TARGET")
	}

	chunks, err := s.repo.GetVersionChunks(ctx, targetVersionID)
	if err != nil {
		return metadata.Version{}, fmt.Errorf("get target version chunks: %w", err)
	}

	meta := metadata.VersionMeta{
		SizeBytes:    targetVer.SizeBytes,
		ContentType:  targetVer.ContentType,
		IsRollbackOf: &targetVersionID,
	}

	newVer, err := s.repo.CreateVersion(ctx, objectID, meta)
	if err != nil {
		return metadata.Version{}, fmt.Errorf("create version: %w", err)
	}

	if err := s.repo.CommitVersion(ctx, newVer.ID, chunks); err != nil {
		return metadata.Version{}, fmt.Errorf("commit rollback version: %w", err)
	}

	slog.Info("rollback performed", "object_id", objectID, "target_version", targetVersionID, "new_version", newVer.ID)

	committedVersion, err := s.repo.GetVersion(ctx, newVer.ID)
	if err != nil {
		return metadata.Version{}, fmt.Errorf("get committed rollback version: %w", err)
	}

	return committedVersion, nil
}
