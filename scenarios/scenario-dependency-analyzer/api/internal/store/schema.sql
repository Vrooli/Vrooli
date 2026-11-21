CREATE TABLE IF NOT EXISTS scenario_dependencies (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))), 2) || '-' || substr('89ab', abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))), 2) || '-' || lower(hex(randomblob(6)))),
    scenario_name TEXT NOT NULL,
    dependency_type TEXT NOT NULL CHECK (dependency_type IN ('resource', 'scenario', 'shared_workflow', 'cli_tool')),
    dependency_name TEXT NOT NULL,
    required INTEGER NOT NULL DEFAULT 1,
    purpose TEXT,
    access_method TEXT,
    configuration TEXT,
    discovered_at TEXT DEFAULT (datetime('now')),
    last_verified TEXT DEFAULT (datetime('now')),
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_scenario_dependencies_scenario_name ON scenario_dependencies (scenario_name);
CREATE INDEX IF NOT EXISTS idx_scenario_dependencies_type ON scenario_dependencies (dependency_type);
CREATE INDEX IF NOT EXISTS idx_scenario_dependencies_name ON scenario_dependencies (dependency_name);
CREATE INDEX IF NOT EXISTS idx_scenario_dependencies_required ON scenario_dependencies (required);

CREATE TABLE IF NOT EXISTS optimization_recommendations (
    id TEXT PRIMARY KEY,
    scenario_name TEXT NOT NULL,
    recommendation_type TEXT NOT NULL CHECK (
        recommendation_type IN (
            'resource_swap',
            'dependency_reduction',
            'merger_opportunity',
            'shared_workflow_adoption',
            'cost_optimization',
            'performance_improvement'
        )
    ),
    title TEXT NOT NULL,
    description TEXT,
    current_state TEXT NOT NULL,
    recommended_state TEXT NOT NULL,
    estimated_impact TEXT,
    confidence_score REAL CHECK (confidence_score >= 0.0 AND confidence_score <= 1.0),
    priority TEXT DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    status TEXT DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'applied', 'rejected', 'expired')),
    applied_at TEXT,
    expires_at TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_optimization_recommendations_scenario ON optimization_recommendations (scenario_name);
CREATE INDEX IF NOT EXISTS idx_optimization_recommendations_type ON optimization_recommendations (recommendation_type);
CREATE INDEX IF NOT EXISTS idx_optimization_recommendations_status ON optimization_recommendations (status);
CREATE INDEX IF NOT EXISTS idx_optimization_recommendations_priority ON optimization_recommendations (priority);

CREATE TABLE IF NOT EXISTS scenario_metadata (
    scenario_name TEXT PRIMARY KEY,
    display_name TEXT,
    description TEXT,
    version TEXT,
    tags TEXT,
    ports TEXT,
    service_config TEXT,
    file_path TEXT,
    last_scanned TEXT DEFAULT (datetime('now')),
    last_modified TEXT,
    is_active INTEGER DEFAULT 1,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_scenario_metadata_active ON scenario_metadata (is_active);
CREATE INDEX IF NOT EXISTS idx_scenario_metadata_last_scanned ON scenario_metadata (last_scanned);

CREATE TABLE IF NOT EXISTS interface_graph_cache (
    fleet_signature TEXT PRIMARY KEY,
    graph_json TEXT NOT NULL,
    computed_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_interface_graph_cache_computed_at ON interface_graph_cache (computed_at);

-- graph_edges is the unified, evidence-tagged cross-scenario dependency graph.
-- It merges proto_import ∪ go_import ∪ declared ∪ vrooli_cli ∪ resource, deduped
-- by (from_scenario, to_node) keeping the highest-confidence source and the union
-- of evidence. It is the single source of truth read by graph generation and
-- centrality (populated by the ingest sweeper, NOT by per-scenario analyze).
CREATE TABLE IF NOT EXISTS graph_edges (
    from_scenario   TEXT NOT NULL,
    to_node         TEXT NOT NULL,
    kind            TEXT NOT NULL CHECK (kind IN ('scenario', 'resource')),
    evidence_source TEXT NOT NULL CHECK (evidence_source IN ('proto_import', 'go_import', 'declared', 'vrooli_cli', 'resource')),
    confidence      REAL NOT NULL DEFAULT 0,
    required        INTEGER NOT NULL DEFAULT 0,
    evidence_json   TEXT NOT NULL DEFAULT '[]',
    stale           INTEGER NOT NULL DEFAULT 0,
    last_verified   TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (from_scenario, to_node)
);

CREATE INDEX IF NOT EXISTS idx_graph_edges_to_node ON graph_edges (to_node);
CREATE INDEX IF NOT EXISTS idx_graph_edges_kind ON graph_edges (kind);
CREATE INDEX IF NOT EXISTS idx_graph_edges_source ON graph_edges (evidence_source);
CREATE INDEX IF NOT EXISTS idx_graph_edges_from ON graph_edges (from_scenario);

-- graph_ingest_state records the scenario tree digest of the last successful
-- ingest, so the freshness-gated sweeper can skip scenarios whose source tree
-- has not changed since they were last ingested.
CREATE TABLE IF NOT EXISTS graph_ingest_state (
    scenario          TEXT PRIMARY KEY,
    digest            TEXT NOT NULL,
    last_ingested_at  TEXT NOT NULL DEFAULT (datetime('now'))
);
