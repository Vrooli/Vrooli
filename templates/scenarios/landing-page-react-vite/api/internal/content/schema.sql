-- Content domain schema.
--
-- Ordered, typed content blocks (hero, features, pricing, …) that compose a
-- variant's landing page. Foreign-keys the variant it belongs to; this schema
-- is applied after the variant schema. Forward-only declarative DDL.
CREATE TABLE IF NOT EXISTS content_sections (
    id SERIAL PRIMARY KEY,
    variant_id INTEGER REFERENCES variants(id) ON DELETE CASCADE,
    section_type VARCHAR(50) NOT NULL CHECK (section_type IN ('hero', 'features', 'pricing', 'cta', 'testimonials', 'faq', 'footer', 'video', 'downloads')),
    content JSONB NOT NULL,
    "order" INTEGER DEFAULT 0,
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_content_sections_variant ON content_sections(variant_id);
CREATE INDEX IF NOT EXISTS idx_content_sections_type ON content_sections(section_type);
CREATE INDEX IF NOT EXISTS idx_content_sections_order ON content_sections("order");
