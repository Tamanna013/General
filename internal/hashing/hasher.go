package hashing

import (
	"crypto/sha256"
	"encoding/hex"
)

// ComputeHash computes the SHA-256 hash of a byte slice and returns it as a hex string.
func ComputeHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// PathForHash determines the sharded directory path for a given hash.
// For example, "ab12cd34..." -> "ab/12".
// Git uses a similar sharding strategy to avoid filesystem performance
// degradation from having too many files in a single directory.
func PathForHash(hash string) (string, string) {
	if len(hash) < 4 {
		// Fallback for malformed hashes, though this shouldn't happen with sha256
		return "", hash
	}
	dir1 := hash[0:2]
	dir2 := hash[2:4]
	return dir1 + "/" + dir2, hash
}
