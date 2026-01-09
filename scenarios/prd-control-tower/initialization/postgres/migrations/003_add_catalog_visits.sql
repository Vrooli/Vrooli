-- Add catalog visit tracking to support recently viewed sorting

CREATE TABLE IF NOT EXISTS catalog_visits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(20) NOT NULL CHECK (entity_type IN ('scenario', 'resource')),
    entity_name VARCHAR(255) NOT NULL,
    visit_count INTEGER NOT NULL DEFAULT 1,
    last_visited_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(entity_type, entity_name)
);

CREATE INDEX IF NOT EXISTS idx_catalog_visits_entity ON catalog_visits(entity_type, entity_name);
CREATE INDEX IF NOT EXISTS idx_catalog_visits_last_visited ON catalog_visits(last_visited_at DESC);
CREATE INDEX IF NOT EXISTS idx_catalog_visits_count ON catalog_visits(visit_count DESC);

-- Add catalog labels to support user-defined tagging and filtering

CREATE TABLE IF NOT EXISTS catalog_labels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(20) NOT NULL CHECK (entity_type IN ('scenario', 'resource')),
    entity_name VARCHAR(255) NOT NULL,
    label VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(entity_type, entity_name, label)
);

CREATE INDEX IF NOT EXISTS idx_catalog_labels_entity ON catalog_labels(entity_type, entity_name);
CREATE INDEX IF NOT EXISTS idx_catalog_labels_label ON catalog_labels(label);
