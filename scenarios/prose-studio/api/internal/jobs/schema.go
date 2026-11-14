package jobs

const schemaSQL = `
CREATE TABLE IF NOT EXISTS job_openings (id TEXT PRIMARY KEY, payload TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS job_facts (id TEXT PRIMARY KEY, payload TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS job_resumes (id TEXT PRIMARY KEY, opening_id TEXT NOT NULL DEFAULT '', payload TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS job_cover_letters (id TEXT PRIMARY KEY, opening_id TEXT NOT NULL DEFAULT '', payload TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE INDEX IF NOT EXISTS job_resumes_opening ON job_resumes(opening_id);
CREATE INDEX IF NOT EXISTS job_letters_opening ON job_cover_letters(opening_id);
`

func Schema() string { return schemaSQL }
