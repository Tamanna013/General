CREATE TABLE objects (
    id            UUID PRIMARY KEY,
    name          TEXT NOT NULL,
    size_bytes    BIGINT NOT NULL,
    content_type  TEXT NOT NULL DEFAULT 'application/octet-stream',
    status        TEXT NOT NULL DEFAULT 'uploading', -- uploading | committed | deleted
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ NULL
);

CREATE TABLE chunks (
    hash          TEXT PRIMARY KEY,        -- hex-encoded SHA-256, also the on-disk filename
    size_bytes    INT NOT NULL,
    ref_count     INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- join table preserving chunk order per object
CREATE TABLE object_chunks (
    object_id     UUID NOT NULL REFERENCES objects(id) ON DELETE CASCADE,
    chunk_hash    TEXT NOT NULL REFERENCES chunks(hash),
    chunk_index   INT NOT NULL,             -- 0-based order within the object
    PRIMARY KEY (object_id, chunk_index)
);

CREATE INDEX idx_object_chunks_chunk_hash ON object_chunks(chunk_hash);
CREATE INDEX idx_objects_status ON objects(status) WHERE deleted_at IS NULL;
