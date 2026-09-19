CREATE TABLE IF NOT EXISTS goals (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  purpose TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  progress_method TEXT NOT NULL,
  progress_basis_points INTEGER NOT NULL DEFAULT 0,
  target_basis_points INTEGER NOT NULL DEFAULT 10000,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  revision INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_goals_status_updated ON goals(status, updated_at DESC);
CREATE TABLE IF NOT EXISTS milestones (
  id TEXT PRIMARY KEY,
  goal_id TEXT NOT NULL,
  title TEXT NOT NULL,
  criteria TEXT NOT NULL DEFAULT '',
  due_date TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  revision INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_milestones_goal_due ON milestones(goal_id, due_date, updated_at DESC);
CREATE TABLE IF NOT EXISTS milestone_work_links (
  milestone_id TEXT PRIMARY KEY,
  work_item_id TEXT NOT NULL,
  created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_milestone_work_links_work ON milestone_work_links(work_item_id);
CREATE TABLE IF NOT EXISTS milestone_prerequisites (
  milestone_id TEXT NOT NULL,
  prerequisite_milestone_id TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  PRIMARY KEY (milestone_id, prerequisite_milestone_id)
);
CREATE INDEX IF NOT EXISTS idx_milestone_prerequisites_prerequisite ON milestone_prerequisites(prerequisite_milestone_id);
