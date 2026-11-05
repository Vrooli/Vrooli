-- Migration: Add PII watchlist and scan allowlist tables
-- Apply with: psql -d secrets_manager -f migrations/003_pii_watchlist_allowlist.sql

BEGIN;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Stores encrypted personal values that should always be flagged when found
-- during a scan. Values are encrypted with AES-256-GCM using an env-var key.
CREATE TABLE IF NOT EXISTS pii_watchlist (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    label TEXT NOT NULL,
    encrypted_value BYTEA NOT NULL,
    value_type TEXT NOT NULL CHECK (value_type IN ('email', 'phone', 'path', 'ssn', 'custom')),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Stores path-glob × finding-type exemption rules. A rule with path_pattern
-- matching a file and excluded_types containing the finding type (or '*')
-- suppresses that finding.
CREATE TABLE IF NOT EXISTS scan_allowlist_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    path_pattern TEXT NOT NULL UNIQUE,
    excluded_types TEXT[] NOT NULL,
    description TEXT,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_pii_watchlist_value_type ON pii_watchlist(value_type);
CREATE INDEX IF NOT EXISTS idx_scan_allowlist_rules_enabled ON scan_allowlist_rules(enabled);

-- Seed default allowlist rules covering the most common false-positive surfaces.
INSERT INTO scan_allowlist_rules (path_pattern, excluded_types, description, enabled) VALUES
    ('*_test.go', ARRAY['pii_email', 'pii_phone_us', 'pii_ip_address'], 'Go test files — test fixtures commonly contain synthetic PII', true),
    ('testdata/**', ARRAY['*'], 'Standard Go testdata directory', true),
    ('**/fixtures/**', ARRAY['*'], 'Test fixtures', true),
    ('**/vendor/**', ARRAY['*'], 'Vendored dependencies', true),
    ('go.sum', ARRAY['*'], 'Go module checksum file', true),
    ('package-lock.json', ARRAY['*'], 'npm lockfile', true),
    ('yarn.lock', ARRAY['*'], 'yarn lockfile', true)
ON CONFLICT (path_pattern) DO NOTHING;

COMMENT ON TABLE pii_watchlist IS
    'User-defined personal values that should always be flagged during scans. Values are encrypted at rest with AES-256-GCM.';
COMMENT ON TABLE scan_allowlist_rules IS
    'Path-glob × finding-type exemption rules. A match suppresses findings for the listed types (or ''*'' for all).';

COMMIT;
