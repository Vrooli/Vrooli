CREATE TABLE IF NOT EXISTS learning_tasks (
  attempt_id TEXT PRIMARY KEY,
  task_id TEXT NOT NULL,
  attempt_number INTEGER NOT NULL,
  session_id TEXT NOT NULL,
  program_id TEXT NOT NULL,
  state TEXT NOT NULL CHECK(state IN ('prepared','running','completed','uncertain')),
  delivery TEXT NOT NULL CHECK(delivery IN ('waiting','pending','delivered','blocked')),
  next_attempt_at TEXT NOT NULL,
  record TEXT NOT NULL,
  UNIQUE(task_id, attempt_number)
);
CREATE INDEX IF NOT EXISTS learning_tasks_pending ON learning_tasks(delivery,next_attempt_at);
CREATE INDEX IF NOT EXISTS learning_tasks_program ON learning_tasks(program_id);
