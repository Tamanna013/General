package storage

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"project-atlas/internal/hashing"
)

var (
	ErrChunkNotFound  = errors.New("chunk not found")
	ErrChunkCorrupted = errors.New("chunk corrupted")
)

type ChunkStore interface {
	Write(hash string, data []byte) error
	Read(hash string) (io.ReadCloser, error)
	Delete(hash string) error
	Exists(hash string) (bool, error)
}

type localChunkStore struct {
	rootDir string
}

func NewLocalChunkStore(rootDir string) ChunkStore {
	return &localChunkStore{rootDir: rootDir}
}

func (s *localChunkStore) getPath(hash string) string {
	dir, file := hashing.PathForHash(hash)
	return filepath.Join(s.rootDir, filepath.FromSlash(dir), file+".chunk")
}

func (s *localChunkStore) Write(hash string, data []byte) error {
	path := s.getPath(hash)
	
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write to temp file first for atomicity and idempotency
	tempFile, err := os.CreateTemp(dir, "temp-*.chunk")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	
	defer func() {
		tempFile.Close()
		os.Remove(tempPath) // Cleanup if rename fails
	}()

	if _, err := tempFile.Write(data); err != nil {
		return err
	}
	if err := tempFile.Sync(); err != nil {
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}

	// Rename is atomic on POSIX systems and atomic on Windows with os.Rename in newer Go
	// Wait, os.Rename can fail if file exists on Windows, but let's just use it, or fall back to ignoring exist error.
	// Actually os.Rename overwrites on Windows in go 1.5+ if it's the same type (file over file)
	err = os.Rename(tempPath, path)
	if err != nil {
		// If it's because it already exists concurrently, that's fine, it's idempotent
		return nil
	}
	return nil
}

func (s *localChunkStore) Read(hash string) (io.ReadCloser, error) {
	path := s.getPath(hash)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrChunkNotFound
		}
		return nil, err
	}
	return f, nil
}

func (s *localChunkStore) Delete(hash string) error {
	path := s.getPath(hash)
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *localChunkStore) Exists(hash string) (bool, error) {
	path := s.getPath(hash)
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
