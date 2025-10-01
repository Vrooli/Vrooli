-- =============================================================================
-- ATTACHMENTS AND FILE STORAGE
-- =============================================================================
-- Stores metadata for files (photos, documents) associated with contacts
-- Actual files stored in MinIO, this tracks references and metadata

CREATE TABLE IF NOT EXISTS attachments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Association
    person_id UUID REFERENCES persons(id) ON DELETE CASCADE,
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,

    -- File information
    file_name TEXT NOT NULL,
    file_type TEXT NOT NULL, -- 'photo', 'document', 'avatar', 'resume', etc
    mime_type TEXT NOT NULL,
    file_size_bytes INTEGER NOT NULL,

    -- Storage location
    storage_backend TEXT NOT NULL DEFAULT 'minio', -- 'minio', 'filesystem', 'url'
    storage_path TEXT NOT NULL, -- MinIO object key or file path
    storage_bucket TEXT, -- MinIO bucket name

    -- Optional thumbnail for images
    thumbnail_path TEXT,
    thumbnail_bucket TEXT,

    -- Metadata
    description TEXT,
    tags TEXT[],
    is_primary BOOLEAN DEFAULT false, -- Primary photo/document for the person
    is_public BOOLEAN DEFAULT false, -- Whether this can be shared with other scenarios

    -- Privacy and consent
    consent_verified BOOLEAN DEFAULT false,
    consent_verified_at TIMESTAMP WITH TIME ZONE,

    -- Additional metadata
    metadata JSONB DEFAULT '{}',

    -- Soft delete
    deleted_at TIMESTAMP WITH TIME ZONE,

    -- Ensure only one primary photo per person
    CONSTRAINT unique_primary_photo UNIQUE (person_id, file_type, is_primary)
        DEFERRABLE INITIALLY DEFERRED,

    -- Either person or organization, not both
    CONSTRAINT attachment_owner CHECK (
        (person_id IS NOT NULL AND organization_id IS NULL) OR
        (person_id IS NULL AND organization_id IS NOT NULL)
    )
);

-- Indexes for efficient queries
CREATE INDEX idx_attachments_person ON attachments(person_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_attachments_organization ON attachments(organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_attachments_file_type ON attachments(file_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_attachments_tags ON attachments USING GIN(tags) WHERE deleted_at IS NULL;

-- View for easy access to person photos
CREATE VIEW person_photos AS
SELECT
    a.*,
    p.full_name as person_name
FROM attachments a
JOIN persons p ON a.person_id = p.id
WHERE a.file_type = 'photo'
  AND a.deleted_at IS NULL
  AND p.deleted_at IS NULL;

-- View for primary photos only
CREATE VIEW primary_photos AS
SELECT
    person_id,
    file_name,
    storage_path,
    storage_bucket,
    thumbnail_path
FROM attachments
WHERE file_type = 'photo'
  AND is_primary = true
  AND deleted_at IS NULL;

-- Function to ensure only one primary photo per person
CREATE OR REPLACE FUNCTION ensure_single_primary_photo()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.is_primary = true AND NEW.file_type = 'photo' THEN
        UPDATE attachments
        SET is_primary = false, updated_at = CURRENT_TIMESTAMP
        WHERE person_id = NEW.person_id
          AND file_type = 'photo'
          AND is_primary = true
          AND id != NEW.id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER ensure_single_primary_photo_trigger
    BEFORE INSERT OR UPDATE ON attachments
    FOR EACH ROW
    EXECUTE FUNCTION ensure_single_primary_photo();

-- Comments for documentation
COMMENT ON TABLE attachments IS 'File metadata for photos and documents associated with contacts, actual files in MinIO';
COMMENT ON COLUMN attachments.storage_backend IS 'Storage backend: minio (default), filesystem, or url for external links';
COMMENT ON COLUMN attachments.is_primary IS 'Primary photo/avatar for the person, only one allowed per person';
COMMENT ON COLUMN attachments.consent_verified IS 'Whether consent has been obtained to store and use this file';