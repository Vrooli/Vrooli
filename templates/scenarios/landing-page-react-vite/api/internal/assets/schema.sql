-- Assets domain schema.
--
-- Uploaded media (logos, favicons, OG images, general) with per-file metadata.
-- The bytes live on disk under UPLOAD_DIR; this table records the catalog.
-- Forward-only declarative DDL.
CREATE TABLE IF NOT EXISTS assets (
    id SERIAL PRIMARY KEY,
    filename TEXT NOT NULL,
    original_filename TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    storage_path TEXT NOT NULL,
    thumbnail_path TEXT,
    alt_text TEXT,
    category TEXT DEFAULT 'general' CHECK (category IN ('logo', 'favicon', 'og_image', 'general')),
    uploaded_by TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_assets_category ON assets(category);
CREATE INDEX IF NOT EXISTS idx_assets_created ON assets(created_at);
