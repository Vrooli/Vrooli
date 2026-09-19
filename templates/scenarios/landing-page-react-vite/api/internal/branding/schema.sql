-- Branding domain schema.
--
-- Singleton row (id = 1) holding site-wide branding: logos, theme colors,
-- SEO defaults, and robots policy. Forward-only declarative DDL — safe to
-- re-run (CREATE TABLE IF NOT EXISTS).
CREATE TABLE IF NOT EXISTS site_branding (
    id INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    site_name TEXT NOT NULL DEFAULT 'My Landing',
    tagline TEXT,
    logo_url TEXT,
    logo_icon_url TEXT,
    favicon_url TEXT,
    apple_touch_icon_url TEXT,
    default_title TEXT,
    default_description TEXT,
    default_og_image_url TEXT,
    theme_primary_color TEXT,
    theme_background_color TEXT,
    canonical_base_url TEXT,
    google_site_verification TEXT,
    robots_txt TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
