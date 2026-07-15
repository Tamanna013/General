package upload

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	
	"github.com/google/uuid"
	"project-atlas/internal/chunking"
	"project-atlas/internal/hashing"
	"project-atlas/internal/metadata"
	"project-atlas/internal/storage"
)

type Service struct {
	repo      metadata.Repository
	store     storage.ChunkStore
	chunkSize int
}

func NewService(repo metadata.Repository, store storage.ChunkStore, chunkSize int) *Service {
	return &Service{
		repo:      repo,
		store:     store,
		chunkSize: chunkSize,
	}
}

// UploadNewObject creates a new object and its first version from the reader.
func (s *Service) UploadNewObject(ctx context.Context, name string, contentType string, sizeBytes int64, r io.Reader) (metadata.Version, error) {
	obj, err := s.repo.CreateObject(ctx, name)
	if err != nil {
		return metadata.Version{}, fmt.Errorf("create object: %w", err)
	}

	return s.UploadVersion(ctx, obj.ID, contentType, sizeBytes, r)
}

// UploadVersion creates a new version for an existing object.
// Note: Chunk processing is currently sequential, which is a named optimization target for a later phase.
func (s *Service) UploadVersion(ctx context.Context, objectID uuid.UUID, contentType string, sizeBytes int64, r io.Reader) (metadata.Version, error) {
	meta := metadata.VersionMeta{
		SizeBytes:   sizeBytes,
		ContentType: contentType,
	}
	
	version, err := s.repo.CreateVersion(ctx, objectID, meta)
	if err != nil {
		return metadata.Version{}, fmt.Errorf("create version: %w", err)
	}

	slog.Info("version created", "object_id", objectID, "version_id", version.ID, "version_number", version.VersionNumber)

	chunksCh, errsCh := chunking.Split(r, s.chunkSize)
	
	var chunkRefs []metadata.ChunkRef

	for chunk := range chunksCh {
		hash := hashing.ComputeHash(chunk.Data)
		
		// Dedup check
		exists, err := s.store.Exists(hash)
		if err != nil {
			return metadata.Version{}, fmt.Errorf("check chunk exists: %w", err)
		}

		if !exists {
			if err := s.store.Write(hash, chunk.Data); err != nil {
				return metadata.Version{}, fmt.Errorf("write chunk: %w", err)
			}
			slog.Debug("chunk written to disk", "hash", hash)
		} else {
			slog.Debug("chunk deduplicated", "hash", hash)
		}

		chunkRefs = append(chunkRefs, metadata.ChunkRef{
			Index: chunk.Index,
			Hash:  hash,
			Size:  len(chunk.Data),
		})
	}

	if err := <-errsCh; err != nil {
		return metadata.Version{}, fmt.Errorf("chunking error: %w", err)
	}

	// Commit version (atomically links chunks and updates status)
	if err := s.repo.CommitVersion(ctx, version.ID, chunkRefs); err != nil {
		return metadata.Version{}, fmt.Errorf("commit version: %w", err)
	}

	// Fetch updated version to return
	committedVersion, err := s.repo.GetVersion(ctx, version.ID)
	if err != nil {
		return metadata.Version{}, fmt.Errorf("get committed version: %w", err)
	}

	return committedVersion, nil
}
