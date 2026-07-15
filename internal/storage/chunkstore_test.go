package storage

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalChunkStore(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewLocalChunkStore(tmpDir)

	hash := "abcd1234efgh"
	data := []byte("hello world")

	// Not exists initially
	exists, err := store.Exists(hash)
	require.NoError(t, err)
	assert.False(t, exists)

	// Write
	err = store.Write(hash, data)
	require.NoError(t, err)

	// Exists now
	exists, err = store.Exists(hash)
	require.NoError(t, err)
	assert.True(t, exists)

	// Verify sharding
	expectedPath := filepath.Join(tmpDir, "ab", "cd", "abcd1234efgh.chunk")
	_, err = os.Stat(expectedPath)
	require.NoError(t, err)

	// Read
	rc, err := store.Read(hash)
	require.NoError(t, err)
	defer rc.Close()
	
	readData, err := io.ReadAll(rc)
	rc.Close()
	require.NoError(t, err)
	assert.Equal(t, data, readData)

	// Delete
	err = store.Delete(hash)
	require.NoError(t, err)

	// Not exists anymore
	exists, err = store.Exists(hash)
	require.NoError(t, err)
	assert.False(t, exists)
	
	// Read deleted
	_, err = store.Read(hash)
	assert.ErrorIs(t, err, ErrChunkNotFound)
}
