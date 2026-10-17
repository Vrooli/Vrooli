
-- Stream of Consciousness Analyzer schema
-- All statements are idempotent (safe to re-run)

CREATE TABLE IF NOT EXISTS schemes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL DEFAULT 'Untitled',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS information (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scheme_id UUID NOT NULL REFERENCES schemes(id) ON DELETE CASCADE,
    type TEXT NOT NULL DEFAULT 'text',
    content TEXT NOT NULL DEFAULT '',
    canvas_x DOUBLE PRECISION NOT NULL DEFAULT 0,
    canvas_y DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS thoughts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scheme_id UUID REFERENCES schemes(id) ON DELETE SET NULL,
    title TEXT NOT NULL DEFAULT '',
    body TEXT NOT NULL DEFAULT '',
    canvas_x DOUBLE PRECISION NOT NULL DEFAULT 0,
    canvas_y DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS thought_edges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES thoughts(id) ON DELETE CASCADE,
    target_id UUID NOT NULL REFERENCES thoughts(id) ON DELETE CASCADE,
    label TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(source_id, target_id)
);

CREATE INDEX IF NOT EXISTS idx_information_scheme ON information(scheme_id);
CREATE INDEX IF NOT EXISTS idx_thoughts_scheme ON thoughts(scheme_id);
CREATE INDEX IF NOT EXISTS idx_thought_edges_source ON thought_edges(source_id);
CREATE INDEX IF NOT EXISTS idx_thought_edges_target ON thought_edges(target_id);
