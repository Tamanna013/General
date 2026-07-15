package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr            string
	PostgresDSN         string
	StorageRoot         string
	ChunkSizeBytes      int
	MaxUploadSizeBytes  int64
}

func Load() (*Config, error) {
	httpAddr := os.Getenv("ATLAS_HTTP_ADDR")
	if httpAddr == "" {
		return nil, fmt.Errorf("ATLAS_HTTP_ADDR is required")
	}

	postgresDSN := os.Getenv("ATLAS_POSTGRES_DSN")
	if postgresDSN == "" {
		return nil, fmt.Errorf("ATLAS_POSTGRES_DSN is required")
	}

	storageRoot := os.Getenv("ATLAS_STORAGE_ROOT")
	if storageRoot == "" {
		return nil, fmt.Errorf("ATLAS_STORAGE_ROOT is required")
	}

	chunkSizeStr := os.Getenv("ATLAS_CHUNK_SIZE_BYTES")
	if chunkSizeStr == "" {
		return nil, fmt.Errorf("ATLAS_CHUNK_SIZE_BYTES is required")
	}
	chunkSize, err := strconv.Atoi(chunkSizeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid ATLAS_CHUNK_SIZE_BYTES: %w", err)
	}

	maxUploadSizeStr := os.Getenv("ATLAS_MAX_UPLOAD_SIZE_BYTES")
	if maxUploadSizeStr == "" {
		return nil, fmt.Errorf("ATLAS_MAX_UPLOAD_SIZE_BYTES is required")
	}
	maxUploadSize, err := strconv.ParseInt(maxUploadSizeStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid ATLAS_MAX_UPLOAD_SIZE_BYTES: %w", err)
	}

	return &Config{
		HTTPAddr:           httpAddr,
		PostgresDSN:        postgresDSN,
		StorageRoot:        storageRoot,
		ChunkSizeBytes:     chunkSize,
		MaxUploadSizeBytes: maxUploadSize,
	}, nil
}
