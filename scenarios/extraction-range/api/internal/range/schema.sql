CREATE TABLE IF NOT EXISTS extraction_range_scores (
    tool TEXT NOT NULL,
    level_id TEXT NOT NULL,
    family TEXT NOT NULL,
    correct INTEGER NOT NULL,
    submitted INTEGER NOT NULL,
    expected INTEGER NOT NULL,
    precision REAL NOT NULL,
    recall REAL NOT NULL,
    f1 REAL NOT NULL,
    passed INTEGER NOT NULL,
    PRIMARY KEY (tool, level_id)
);
