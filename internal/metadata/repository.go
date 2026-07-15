package metadata

import (
	"context"
	"errors"
	"github.com/google/uuid"
)

var (
	ErrObjectNotFound  = errors.New("object not found")
	ErrVersionNotFound = errors.New("version not found")
	ErrConflict        = errors.New("conflict")
)

type Repository interface {
	CreateObject(ctx context.Context, name string) (Object, error)
	GetObject(ctx context.Context, id uuid.UUID) (Object, error)
	SoftDeleteObject(ctx context.Context, id uuid.UUID) error

	CreateVersion(ctx context.Context, objectID uuid.UUID, meta VersionMeta) (Version, error)
	CommitVersion(ctx context.Context, versionID uuid.UUID, chunks []ChunkRef) error
	GetVersion(ctx context.Context, versionID uuid.UUID) (Version, error)
	ListVersions(ctx context.Context, objectID uuid.UUID) ([]Version, error)
	GetVersionChunks(ctx context.Context, versionID uuid.UUID) ([]ChunkRef, error)
	DeleteVersion(ctx context.Context, versionID uuid.UUID) error
	
	// GetChunksToDeleteAfterWholeObjectDelete returns chunks that should be deleted from disk
	// after an object and all its versions are soft-deleted.
	GetChunksToDeleteAfterWholeObjectDelete(ctx context.Context, objectID uuid.UUID) ([]string, error)
	
	// GetChunksToDeleteAfterVersionDelete returns chunks that should be deleted from disk
	// after a specific version is deleted.
	GetChunksToDeleteAfterVersionDelete(ctx context.Context, versionID uuid.UUID) ([]string, error)
}
