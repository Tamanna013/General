package chunking

import (
	"errors"
	"io"
)

// Chunk represents a single chunk of data with its 0-based index.
type Chunk struct {
	Index int
	Data  []byte
}

// Split reads from r and yields fixed-size chunks via a channel.
// It streams the file and does not load the entire file into memory at once.
// The channel approach allows the caller to process chunks concurrently with reading
// or handle them in a pipeline fashion.
func Split(r io.Reader, chunkSize int) (<-chan Chunk, <-chan error) {
	chunks := make(chan Chunk)
	errs := make(chan error, 1)

	go func() {
		defer close(chunks)
		defer close(errs)

		if chunkSize <= 0 {
			errs <- errors.New("chunk size must be greater than 0")
			return
		}

		index := 0
		for {
			buf := make([]byte, chunkSize)
			n, err := io.ReadFull(r, buf)
			if err != nil {
				if err == io.EOF {
					// Only yield an empty chunk if this is the very first read (0-byte file)
					if index == 0 {
						chunks <- Chunk{Index: index, Data: []byte{}}
					}
					return
				}
				if err == io.ErrUnexpectedEOF {
					// Last chunk is smaller than chunk size
					chunks <- Chunk{Index: index, Data: buf[:n]}
					return
				}
				// Other read errors
				errs <- err
				return
			}
			chunks <- Chunk{Index: index, Data: buf[:n]}
			index++
		}
	}()

	return chunks, errs
}
