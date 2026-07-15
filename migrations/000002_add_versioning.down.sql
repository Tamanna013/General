-- Best-effort down migration. Will lose version history beyond v1.

-- Re-add columns to objects
ALTER TABLE objects ADD COLUMN size_bytes BIGINT;
ALTER TABLE objects ADD COLUMN content_type TEXT;
ALTER TABLE objects ADD COLUMN status TEXT;

-- Restore data from the current version
UPDATE objects o
SET 
    size_bytes = ov.size_bytes,
    content_type = ov.content_type,
    status = ov.status
FROM object_versions ov
WHERE o.current_version_id = ov.id;

-- Make columns NOT NULL where applicable
ALTER TABLE objects ALTER COLUMN size_bytes SET NOT NULL;
ALTER TABLE objects ALTER COLUMN content_type SET NOT NULL;
ALTER TABLE objects ALTER COLUMN content_type SET DEFAULT 'application/octet-stream';
ALTER TABLE objects ALTER COLUMN status SET NOT NULL;
ALTER TABLE objects ALTER COLUMN status SET DEFAULT 'uploading';

-- Drop current_version_id
ALTER TABLE objects DROP CONSTRAINT objects_current_version_id_fkey;
ALTER TABLE objects DROP COLUMN current_version_id;

-- Restore object_id to object_chunks
ALTER TABLE object_chunks ADD COLUMN object_id UUID;

-- Update object_id in object_chunks based on version_id
UPDATE object_chunks oc
SET object_id = ov.object_id
FROM object_versions ov
WHERE oc.version_id = ov.id;

-- Make object_id NOT NULL and add foreign key
ALTER TABLE object_chunks ALTER COLUMN object_id SET NOT NULL;
ALTER TABLE object_chunks ADD CONSTRAINT object_chunks_object_id_fkey FOREIGN KEY (object_id) REFERENCES objects(id) ON DELETE CASCADE;

-- Drop version_id and change primary key
ALTER TABLE object_chunks DROP CONSTRAINT object_chunks_pkey;
ALTER TABLE object_chunks DROP COLUMN version_id;
ALTER TABLE object_chunks ADD PRIMARY KEY (object_id, chunk_index);

-- Drop object_versions table
DROP TABLE object_versions;
