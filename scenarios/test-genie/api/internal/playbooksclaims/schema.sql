-- playbooks_claims: per-target single-slot lease for an in-flight
-- test-genie playbooks run. Acquire/heartbeat/release lifecycle is owned
-- by internal/playbooksclaims. Declarative current state — never edited
-- in place to "migrate" data; typed target columns are upgraded by Migrate.
CREATE TABLE IF NOT EXISTS playbooks_claims (
    scenario_name TEXT NOT NULL,
    target_kind   TEXT NOT NULL DEFAULT 'scenario',
    target_id     TEXT NOT NULL DEFAULT '',
    run_id        TEXT NOT NULL,
    mode          TEXT NOT NULL,
    started_by    TEXT NOT NULL,
    acquired_at   INTEGER NOT NULL,
    heartbeat_at  INTEGER NOT NULL,
    expires_at    INTEGER NOT NULL,
    PRIMARY KEY (target_kind, target_id)
);

CREATE INDEX IF NOT EXISTS playbooks_claims_expires_at_idx
    ON playbooks_claims (expires_at);

CREATE INDEX IF NOT EXISTS playbooks_claims_scenario_idx
    ON playbooks_claims (scenario_name);
