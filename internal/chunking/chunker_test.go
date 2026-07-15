package chunking

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplit(t *testing.T) {
	tests := []struct {
		name      string
		input     []byte
		chunkSize int
		expected  []Chunk
		wantErr   bool
	}{
		{
			name:      "exact multiple",
			input:     []byte("123456"),
			chunkSize: 2,
			expected: []Chunk{
				{Index: 0, Data: []byte("12")},
				{Index: 1, Data: []byte("34")},
				{Index: 2, Data: []byte("56")},
			},
		},
		{
			name:      "last chunk smaller",
			input:     []byte("12345"),
			chunkSize: 2,
			expected: []Chunk{
				{Index: 0, Data: []byte("12")},
				{Index: 1, Data: []byte("34")},
				{Index: 2, Data: []byte("5")},
			},
		},
		{
			name:      "zero byte input",
			input:     []byte(""),
			chunkSize: 2,
			expected: []Chunk{
				{Index: 0, Data: []byte("")},
			},
		},
		{
			name:      "chunk size larger than input",
			input:     []byte("123"),
			chunkSize: 10,
			expected: []Chunk{
				{Index: 0, Data: []byte("123")},
			},
		},
		{
			name:      "invalid chunk size",
			input:     []byte("123"),
			chunkSize: 0,
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := bytes.NewReader(tc.input)
			chunksCh, errsCh := Split(r, tc.chunkSize)

			var actualChunks []Chunk
			for chunk := range chunksCh {
				actualChunks = append(actualChunks, chunk)
			}

			err := <-errsCh

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, actualChunks)
			}
		})
	}
}

// errorReader injects an error
type errorReader struct{}

func (e errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrClosedPipe
}

func TestSplit_Error(t *testing.T) {
	r := errorReader{}
	chunksCh, errsCh := Split(r, 2)

	var actualChunks []Chunk
	for chunk := range chunksCh {
		actualChunks = append(actualChunks, chunk)
	}

	err := <-errsCh
	assert.Error(t, err)
	assert.Equal(t, io.ErrClosedPipe, err)
	assert.Empty(t, actualChunks)
}
