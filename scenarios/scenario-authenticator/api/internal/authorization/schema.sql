-- Opaque principal scopes. The authenticator stores and emits these values but
-- never imports or interprets a relying party's scope catalog.
CREATE TABLE IF NOT EXISTS account_scopes (
  account_id TEXT NOT NULL,
  scope      TEXT NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY (account_id, scope)
);

