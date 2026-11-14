-- Variant domain schema.
--
-- A/B landing variants and their axis selections. The variants table also
-- carries the per-variant header presentation config and optional SEO overrides
-- (both JSONB), which the header/seo projections read and write in place.
-- Forward-only declarative DDL.
CREATE TABLE IF NOT EXISTS variants (
    id SERIAL PRIMARY KEY,
    slug VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    weight INTEGER DEFAULT 50 CHECK (weight >= 0 AND weight <= 100),
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'archived', 'deleted')),
    header_config JSONB DEFAULT '{}'::jsonb,
    seo_config JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    archived_at TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_variants_slug ON variants(slug);
CREATE INDEX IF NOT EXISTS idx_variants_status ON variants(status);

CREATE TABLE IF NOT EXISTS variant_axes (
    variant_id INTEGER REFERENCES variants(id) ON DELETE CASCADE,
    axis_id VARCHAR(100) NOT NULL,
    variant_value VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (variant_id, axis_id)
);
CREATE INDEX IF NOT EXISTS idx_variant_axes_axis ON variant_axes(axis_id);
