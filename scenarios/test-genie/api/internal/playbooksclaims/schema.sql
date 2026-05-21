-- playbooks_claims: per-scenario single-slot lease for an in-flight
-- test-genie playbooks run. Acquire/heartbeat/release lifecycle is owned
-- by internal/playbooksclaims. Declarative current state — never edited
-- in place to "migrate" data; greenfield only.
CREATE TABLE IF NOT EXISTS playbooks_claims (
    scenario_name TEXT PRIMARY KEY,
    run_id        TEXT NOT NULL,
    mode          TEXT NOT NULL,
    started_by    TEXT NOT NULL,
    acquired_at   INTEGER NOT NULL,
    heartbeat_at  INTEGER NOT NULL,
    expires_at    INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS playbooks_claims_expires_at_idx
    ON playbooks_claims (expires_at);
