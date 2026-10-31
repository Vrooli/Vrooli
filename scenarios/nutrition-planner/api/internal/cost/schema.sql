CREATE TABLE IF NOT EXISTS price_observations (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  item_id TEXT NOT NULL,
  product_id TEXT NOT NULL DEFAULT '',
  package_label TEXT NOT NULL DEFAULT '',
  package_amount TEXT NOT NULL,
  package_unit TEXT NOT NULL,
  price_minor INTEGER NOT NULL,
  currency TEXT NOT NULL,
  currency_exponent INTEGER NOT NULL,
  retailer TEXT NOT NULL DEFAULT '',
  observed_at TEXT NOT NULL,
  valid_through TEXT NOT NULL DEFAULT '',
  available INTEGER NOT NULL DEFAULT 1,
  membership_required INTEGER NOT NULL DEFAULT 0,
  coupon_required INTEGER NOT NULL DEFAULT 0,
  minimum_buy TEXT NOT NULL DEFAULT '',
  source TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_price_observations_workspace_item ON price_observations(workspace_id, item_id, observed_at DESC);
