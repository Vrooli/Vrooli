CREATE TABLE IF NOT EXISTS review_runs (id TEXT PRIMARY KEY, draft_id TEXT NOT NULL, outcome TEXT NOT NULL, created_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS review_verdicts (run_id TEXT NOT NULL REFERENCES review_runs(id), mode TEXT NOT NULL, passed INTEGER NOT NULL, evidence TEXT NOT NULL DEFAULT '', finding TEXT NOT NULL DEFAULT '', PRIMARY KEY(run_id, mode));
