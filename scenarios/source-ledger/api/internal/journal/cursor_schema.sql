-- The position's explicit integer primary key survives VACUUM with gaps intact.
CREATE TABLE IF NOT EXISTS journal_entry_positions (
  position INTEGER PRIMARY KEY,
  entry_id TEXT NOT NULL UNIQUE REFERENCES entries(id)
);
CREATE TABLE IF NOT EXISTS journal_high_water_mark (
  id INTEGER PRIMARY KEY CHECK (id=1),
  max_rowid INTEGER NOT NULL,
  recorded_at TEXT NOT NULL
);
CREATE TRIGGER IF NOT EXISTS entries_append_only_update
BEFORE UPDATE ON entries
BEGIN
  SELECT RAISE(ABORT, 'journal entries are append-only');
END;
CREATE TRIGGER IF NOT EXISTS entries_append_only_delete
BEFORE DELETE ON entries
BEGIN
  SELECT RAISE(ABORT, 'journal entries are append-only');
END;
CREATE TRIGGER IF NOT EXISTS journal_entry_positions_append_only_update
BEFORE UPDATE ON journal_entry_positions
BEGIN
  SELECT RAISE(ABORT, 'journal entry positions are append-only');
END;
CREATE TRIGGER IF NOT EXISTS journal_entry_positions_append_only_delete
BEFORE DELETE ON journal_entry_positions
BEGIN
  SELECT RAISE(ABORT, 'journal entry positions are append-only');
END;
CREATE TRIGGER IF NOT EXISTS journal_entry_positions_no_replace
BEFORE INSERT ON journal_entry_positions
WHEN EXISTS (SELECT 1 FROM journal_entry_positions WHERE entry_id=NEW.entry_id OR position=NEW.position)
BEGIN
  SELECT RAISE(ABORT, 'journal entry positions are append-only');
END;
CREATE TRIGGER IF NOT EXISTS entries_append_position
AFTER INSERT ON entries
BEGIN
  INSERT INTO journal_entry_positions(position,entry_id)
  SELECT COALESCE(MAX(position),0)+1,NEW.id FROM journal_entry_positions;
  INSERT INTO journal_high_water_mark(id,max_rowid,recorded_at)
  VALUES(1,(SELECT position FROM journal_entry_positions WHERE entry_id=NEW.id),strftime('%Y-%m-%dT%H:%M:%fZ','now'))
  ON CONFLICT(id) DO UPDATE SET max_rowid=MAX(journal_high_water_mark.max_rowid,excluded.max_rowid), recorded_at=excluded.recorded_at;
END;
