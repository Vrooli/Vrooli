CREATE TABLE IF NOT EXISTS shopping_checks (
  workspace_id TEXT NOT NULL,
  line_key TEXT NOT NULL,
  checked INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL,
  PRIMARY KEY(workspace_id, line_key)
);

CREATE TABLE IF NOT EXISTS shopping_have_this (
  workspace_id TEXT NOT NULL,
  line_key TEXT NOT NULL,
  asserted INTEGER NOT NULL DEFAULT 1,
  updated_at TEXT NOT NULL,
  PRIMARY KEY(workspace_id,line_key)
);

CREATE TABLE IF NOT EXISTS shopping_purchase_reviews (
  workspace_id TEXT NOT NULL,
  review_id TEXT NOT NULL,
  payload_hash TEXT NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY(workspace_id,review_id)
);

CREATE TABLE IF NOT EXISTS shopping_purchase_review_lines (
  workspace_id TEXT NOT NULL,
  review_id TEXT NOT NULL,
  line_key TEXT NOT NULL,
  item_id TEXT NOT NULL,
  amount TEXT NOT NULL,
  unit TEXT NOT NULL,
  price TEXT NOT NULL,
  omitted INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY(workspace_id,review_id,line_key),
  FOREIGN KEY(workspace_id,review_id) REFERENCES shopping_purchase_reviews(workspace_id,review_id)
);
