package hashing

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestComputeHash(t *testing.T) {
	// sha256("")
	expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	assert.Equal(t, expected, ComputeHash([]byte("")))
}

func TestPathForHash(t *testing.T) {
	hash := "ab12cd34567890"
	dir, file := PathForHash(hash)
	assert.Equal(t, "ab/12", dir)
	assert.Equal(t, hash, file)

	shortHash := "ab"
	dir, file = PathForHash(shortHash)
	assert.Equal(t, "", dir)
	assert.Equal(t, "ab", file)
}
