# Project Atlas

Project Atlas is a distributed object storage platform. Phase 1 & 2 implement a single-node HTTP service with content-addressable storage, chunk deduplication, and object versioning.

## Features

- **Chunking**: Fixed-size 4MB chunking of uploads.
- **Deduplication**: Content-addressable storage (SHA-256) ensures identical chunks are stored only once.
- **Versioning**: Uploading to an existing object creates a new immutable version without duplicating unchanged chunks.
- **Rollback**: Old versions can be restored as the current version without touching the chunk store.
- **Integrity**: Chunk hashes are verified upon download.

## Setup

1. Copy `.env.example` to `.env`.
2. Start PostgreSQL: `docker-compose up -d`
3. Install `golang-migrate/migrate`.
4. Run migrations: `make migrate-up`

## Running

Start the server on port 8080:
```bash
make run
```

## Running Tests

Run unit tests:
```bash
make test
```

Run integration tests (requires PostgreSQL running):
```bash
make test-integration
```

## API Contract

- `POST /api/v1/objects`: Upload a file.
- `GET /api/v1/objects/{id}`: Get object metadata.
- `GET /api/v1/objects/{id}/download`: Download current version.
- `DELETE /api/v1/objects/{id}`: Delete object and all versions.
- `POST /api/v1/objects/{id}/versions`: Upload new version.
- `GET /api/v1/objects/{id}/versions`: List version history.
- `GET /api/v1/objects/{id}/versions/{versionId}`: Get version metadata.
- `GET /api/v1/objects/{id}/versions/{versionId}/download`: Download specific version.
- `POST /api/v1/objects/{id}/versions/{versionId}/rollback`: Rollback to version.
- `DELETE /api/v1/objects/{id}/versions/{versionId}`: Delete specific version.

## Architecture & Schema

The storage model divides logical files into physical chunks. The `chunks` table tracks reference counts. The `object_versions` table tracks linear version history, while `objects` provides a stable identity. The on-disk layout shards chunks using the first four characters of their SHA-256 hash (e.g. `xx/yy/xxyyzz.chunk`).

## Known Limitations

- **Disk / DB consistency edge cases**: If the process crashes mid-upload, the object version remains in `uploading` status forever. Partial disk writes before a DB failure are harmless as they will be ignored, but can leave orphaned chunks on disk. A failure deleting from disk during a `DELETE` operation doesn't roll back DB accounting.
- **Down Migration**: `make migrate-down` will lose version history beyond v1.

## Future Improvements

- Content-defined chunking (CDC) / rolling hashes.
- Parallel uploads.
- Resumable/multipart uploads.
- Version retention/pruning policies and legal hold.
