package download

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/google/uuid"
	"project-atlas/internal/hashing"
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

// DownloadCurrentVersion streams the current version of the object.
func (s *Service) DownloadCurrentVersion(ctx context.Context, objectID uuid.UUID, w io.Writer) error {
	obj, err := s.repo.GetObject(ctx, objectID)
	if err != nil {
		return err
	}
	if obj.CurrentVersionID == nil {
		return fmt.Errorf("object has no current version")
	}

	return s.DownloadVersion(ctx, *obj.CurrentVersionID, w)
}

// DownloadVersion streams a specific version.
func (s *Service) DownloadVersion(ctx context.Context, versionID uuid.UUID, w io.Writer) error {
	// 1. Fetch chunk list
	chunks, err := s.repo.GetVersionChunks(ctx, versionID)
	if err != nil {
		return fmt.Errorf("get version chunks: %w", err)
	}

	// 2. Stream chunks
	for _, chunk := range chunks {
		rc, err := s.store.Read(chunk.Hash)
		if err != nil {
			return fmt.Errorf("read chunk %s: %w", chunk.Hash, err)
		}

		// Read the chunk fully into memory to verify hash before writing?
		// "verify each chunk's hash before sending it (integrity verification)"
		// If we stream directly to w, we can't verify before sending bytes.
		// Since chunk is 4MB, we can buffer it in memory, verify, and then write.
		// This is safe since 4MB is well within memory limits.
		buf, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return fmt.Errorf("read all chunk %s: %w", chunk.Hash, err)
		}

		actualHash := hashing.ComputeHash(buf)
		if actualHash != chunk.Hash {
			slog.Error("chunk integrity failure", "expected_hash", chunk.Hash, "actual_hash", actualHash, "version_id", versionID)
			return storage.ErrChunkCorrupted
		}

		if _, err := w.Write(buf); err != nil {
			return fmt.Errorf("write response: %w", err)
		}
	}

	return nil
}
