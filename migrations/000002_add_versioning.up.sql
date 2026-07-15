CREATE TABLE object_versions (
    id              UUID PRIMARY KEY,
    object_id       UUID NOT NULL REFERENCES objects(id) ON DELETE CASCADE,
    version_number  INT NOT NULL,             -- 1, 2, 3... monotonic per object
    size_bytes      BIGINT NOT NULL,
    content_type    TEXT NOT NULL DEFAULT 'application/octet-stream',
    status          TEXT NOT NULL DEFAULT 'uploading', -- uploading | committed | deleted
    is_rollback_of  UUID NULL REFERENCES object_versions(id), -- set if created via rollback
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ NULL,
    UNIQUE (object_id, version_number)
);

-- Copy existing object chunks over to reference versions instead of objects.
-- Wait, first we need to populate object_versions from objects!
INSERT INTO object_versions (
    id, object_id, version_number, size_bytes, content_type, status, created_at
)
SELECT 
    gen_random_uuid(), -- Need to generate a UUID for the version.
    id,
    1,
    size_bytes,
    content_type,
    status,
    created_at
FROM objects;

-- Now add version_id to object_chunks
ALTER TABLE object_chunks ADD COLUMN version_id UUID;

-- Update object_chunks to point to the newly created version
UPDATE object_chunks oc
SET version_id = ov.id
FROM object_versions ov
WHERE oc.object_id = ov.object_id;

-- Make version_id NOT NULL and add foreign key
ALTER TABLE object_chunks ALTER COLUMN version_id SET NOT NULL;
ALTER TABLE object_chunks ADD CONSTRAINT object_chunks_version_id_fkey FOREIGN KEY (version_id) REFERENCES object_versions(id) ON DELETE CASCADE;

-- Drop object_id and change primary key
ALTER TABLE object_chunks DROP CONSTRAINT object_chunks_pkey;
ALTER TABLE object_chunks DROP COLUMN object_id;
ALTER TABLE object_chunks ADD PRIMARY KEY (version_id, chunk_index);

-- Add current_version_id to objects
ALTER TABLE objects ADD COLUMN current_version_id UUID;

-- Backfill current_version_id
UPDATE objects o
SET current_version_id = ov.id
FROM object_versions ov
WHERE o.id = ov.object_id;

-- Add FK and drop old columns
ALTER TABLE objects ADD CONSTRAINT objects_current_version_id_fkey FOREIGN KEY (current_version_id) REFERENCES object_versions(id);
ALTER TABLE objects DROP COLUMN size_bytes;
ALTER TABLE objects DROP COLUMN content_type;
ALTER TABLE objects DROP COLUMN status;
