CREATE TABLE IF NOT EXISTS projects (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  revision INTEGER NOT NULL CHECK (revision > 0)
);
CREATE INDEX IF NOT EXISTS idx_projects_created_at_id ON projects ((substr(created_at,1,19) || '.' || CASE WHEN substr(created_at,20,1)='Z' THEN '000000000' ELSE substr(substr(created_at,21,instr(substr(created_at,21),'Z')-1) || '000000000',1,9) END) DESC, id DESC);
