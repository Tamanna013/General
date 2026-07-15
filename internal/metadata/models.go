package metadata

import (
	"time"
	"github.com/google/uuid"
)

type Object struct {
	ID               uuid.UUID  `json:"id"`
	Name             string     `json:"name"`
	CurrentVersionID *uuid.UUID `json:"current_version_id,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	DeletedAt        *time.Time `json:"deleted_at,omitempty"`
}

type Version struct {
	ID            uuid.UUID  `json:"id"`
	ObjectID      uuid.UUID  `json:"object_id"`
	VersionNumber int        `json:"version_number"`
	SizeBytes     int64      `json:"size_bytes"`
	ContentType   string     `json:"content_type"`
	Status        string     `json:"status"` // uploading | committed | deleted
	IsRollbackOf  *uuid.UUID `json:"is_rollback_of,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
}

type ChunkRef struct {
	Index int
	Hash  string
	Size  int
}

type VersionMeta struct {
	SizeBytes    int64
	ContentType  string
	IsRollbackOf *uuid.UUID
}
