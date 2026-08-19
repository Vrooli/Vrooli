CREATE TABLE IF NOT EXISTS rails (
    name TEXT PRIMARY KEY CHECK (length(trim(name)) > 0),
    enabled INTEGER NOT NULL CHECK (enabled IN (0, 1))
);

INSERT OR IGNORE INTO rails(name, enabled) VALUES('manual', 1);
