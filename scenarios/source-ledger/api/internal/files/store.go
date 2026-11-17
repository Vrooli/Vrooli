package files

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// schema is a cache: deleting the database only costs one rebuild.
const schema = `
CREATE TABLE IF NOT EXISTS files (
  corpus TEXT NOT NULL, path TEXT NOT NULL, size INTEGER NOT NULL, mtime_ns INTEGER NOT NULL,
  ctime_ns INTEGER NOT NULL DEFAULT 0, truncated INTEGER NOT NULL DEFAULT 0, hash TEXT NOT NULL, topic TEXT NOT NULL, feed TEXT NOT NULL, title TEXT NOT NULL,
  PRIMARY KEY (corpus, path));
CREATE TABLE IF NOT EXISTS chunks (
  id INTEGER PRIMARY KEY, corpus TEXT NOT NULL, path TEXT NOT NULL, topic TEXT NOT NULL,
  feed TEXT NOT NULL, line_start INTEGER NOT NULL, line_end INTEGER NOT NULL,
  heading TEXT NOT NULL, hash TEXT NOT NULL);
CREATE INDEX IF NOT EXISTS chunks_by_path ON chunks(corpus, path);
CREATE INDEX IF NOT EXISTS chunks_by_hash ON chunks(hash);
CREATE VIRTUAL TABLE IF NOT EXISTS chunks_fts USING fts5(path, heading, text, tokenize='porter unicode61');
CREATE TABLE IF NOT EXISTS vectors (hash TEXT PRIMARY KEY, vec BLOB NOT NULL);
`

type store struct{ db *sql.DB }

func openStore(ctx context.Context, db *sql.DB) (*store, error) {
	if _, err := db.ExecContext(ctx, schema); err != nil {
		return nil, fmt.Errorf("files index schema: %w", err)
	}
	// Caches built before these columns gain them; their rows read as
	// changed once and are re-hashed (unchanged content only gets a touch).
	for _, column := range []string{"ctime_ns", "truncated"} {
		var has int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info('files') WHERE name = ?`, column).Scan(&has); err != nil {
			return nil, fmt.Errorf("files index schema: %w", err)
		}
		if has == 0 {
			if _, err := db.ExecContext(ctx, `ALTER TABLE files ADD COLUMN `+column+` INTEGER NOT NULL DEFAULT 0`); err != nil {
				return nil, fmt.Errorf("files index schema: %w", err)
			}
		}
	}
	return &store{db: db}, nil
}

type indexed struct {
	Size, MtimeNS, ChangedNS int64
	Hash                     string
}

// indexedUnder returns the index rows for the corpus under prefix ("" = all).
func (s *store) indexedUnder(ctx context.Context, corpus, prefix string) (map[string]indexed, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT path, size, mtime_ns, ctime_ns, hash FROM files WHERE corpus = ? AND (? = '' OR path >= ? AND path < ?)`, corpus, prefix, prefix, prefix+"\xff")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]indexed{}
	for rows.Next() {
		var p string
		var v indexed
		if err := rows.Scan(&p, &v.Size, &v.MtimeNS, &v.ChangedNS, &v.Hash); err != nil {
			return nil, err
		}
		out[p] = v
	}
	return out, rows.Err()
}

func deleteChunks(ctx context.Context, tx *sql.Tx, corpus, path string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM chunks_fts WHERE rowid IN (SELECT id FROM chunks WHERE corpus = ? AND path = ?)`, corpus, path); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `DELETE FROM chunks WHERE corpus = ? AND path = ?`, corpus, path)
	return err
}

func (s *store) remove(ctx context.Context, tx *sql.Tx, corpus, path string) error {
	if err := deleteChunks(ctx, tx, corpus, path); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `DELETE FROM files WHERE corpus = ? AND path = ?`, corpus, path)
	return err
}

// removeAll drops the given paths from the index in one transaction.
func (s *store) removeAll(ctx context.Context, corpus string, paths map[string]indexed) error {
	if len(paths) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for p := range paths {
		if err := s.remove(ctx, tx, corpus, p); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// touch records a new size/mtime for unchanged content.
func (s *store) touch(ctx context.Context, tx *sql.Tx, corpus string, e Entry) error {
	_, err := tx.ExecContext(ctx, `UPDATE files SET size = ?, mtime_ns = ?, ctime_ns = ?, truncated = ? WHERE corpus = ? AND path = ?`, e.Size, e.MtimeNS, e.ChangedNS, e.Truncated(), corpus, e.Path)
	return err
}

func (s *store) replace(ctx context.Context, tx *sql.Tx, corpus string, e Entry, hash string, chunks []Chunk) error {
	if err := deleteChunks(ctx, tx, corpus, e.Path); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO files (corpus, path, size, mtime_ns, ctime_ns, truncated, hash, topic, feed, title) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(corpus, path) DO UPDATE SET size = excluded.size, mtime_ns = excluded.mtime_ns, ctime_ns = excluded.ctime_ns, truncated = excluded.truncated, hash = excluded.hash, topic = excluded.topic, feed = excluded.feed, title = excluded.title`,
		corpus, e.Path, e.Size, e.MtimeNS, e.ChangedNS, e.Truncated(), hash, e.Topic, e.Feed, firstHeading(chunks, e.Path)); err != nil {
		return err
	}
	for _, c := range chunks {
		res, err := tx.ExecContext(ctx, `INSERT INTO chunks (corpus, path, topic, feed, line_start, line_end, heading, hash) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			corpus, e.Path, e.Topic, e.Feed, c.LineStart, c.LineEnd, c.Heading, chunkHash(c.Text))
		if err != nil {
			return err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO chunks_fts (rowid, path, heading, text) VALUES (?, ?, ?, ?)`, id, e.Path, c.Heading, c.Text); err != nil {
			return err
		}
	}
	return nil
}

type candidate struct {
	ID                 int64
	Path, Topic, Feed  string
	LineStart, LineEnd int
	Heading, Snippet   string
	Text, Hash         string
	Rank               float64
	MtimeNS            int64
	Truncated          bool
}

// search runs one FTS5 query. Column weights favour path and heading matches.
func (s *store) search(ctx context.Context, corpus, match, topic, feed string, limit int) ([]candidate, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT c.id, c.path, c.topic, c.feed, c.line_start, c.line_end, c.heading, c.hash,
       snippet(chunks_fts, 2, '', '', ' … ', 40), chunks_fts.text, bm25(chunks_fts, 2.0, 4.0, 1.0), f.mtime_ns, f.truncated
FROM chunks_fts JOIN chunks c ON c.id = chunks_fts.rowid JOIN files f ON f.corpus = c.corpus AND f.path = c.path
WHERE chunks_fts MATCH ? AND c.corpus = ? AND (? = '' OR c.topic = ?) AND (? = '' OR c.feed = ?)
ORDER BY bm25(chunks_fts, 2.0, 4.0, 1.0) LIMIT ?`, match, corpus, topic, topic, feed, feed, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.ID, &c.Path, &c.Topic, &c.Feed, &c.LineStart, &c.LineEnd, &c.Heading, &c.Hash, &c.Snippet, &c.Text, &c.Rank, &c.MtimeNS, &c.Truncated); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

type feedRow struct {
	Topic, Feed string
	Files       int
	Bytes       int64
	LastNS      int64
}

func (s *store) feeds(ctx context.Context, corpus, topic string) ([]feedRow, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT topic, feed, COUNT(*), SUM(size), MAX(mtime_ns) FROM files WHERE corpus = ? AND (? = '' OR topic = ?) GROUP BY topic, feed ORDER BY topic, feed`, corpus, topic, topic)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []feedRow
	for rows.Next() {
		var r feedRow
		if err := rows.Scan(&r.Topic, &r.Feed, &r.Files, &r.Bytes, &r.LastNS); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// coverage counts distinct chunks and embedded chunks per topic and feed.
func (s *store) coverage(ctx context.Context, corpus, topic string) (map[[2]string][2]int, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT c.topic, c.feed, COUNT(DISTINCT CASE WHEN v.hash IS NULL OR length(v.vec) > 0 THEN c.hash END),
       COUNT(DISTINCT CASE WHEN length(v.vec) > 0 THEN v.hash END)
FROM chunks c LEFT JOIN vectors v ON v.hash = c.hash
WHERE c.corpus = ? AND `+prose+` AND (? = '' OR c.topic = ?) GROUP BY c.topic, c.feed`, corpus, topic, topic)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[[2]string][2]int{}
	for rows.Next() {
		var t, f string
		var n, e int
		if err := rows.Scan(&t, &f, &n, &e); err != nil {
			return nil, err
		}
		out[[2]string{t, f}] = [2]int{n, e}
	}
	return out, rows.Err()
}

type recentRow struct {
	Path, Title string
	MtimeNS     int64
}

func (s *store) recent(ctx context.Context, corpus, topic, feed string, n int) ([]recentRow, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT path, title, mtime_ns FROM files WHERE corpus = ? AND topic = ? AND feed = ? ORDER BY mtime_ns DESC LIMIT ?`, corpus, topic, feed, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []recentRow
	for rows.Next() {
		var r recentRow
		if err := rows.Scan(&r.Path, &r.Title, &r.MtimeNS); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *store) vectors(ctx context.Context, hashes []string) (map[string][]float32, error) {
	out := map[string][]float32{}
	if len(hashes) == 0 {
		return out, nil
	}
	args := make([]any, len(hashes))
	for i, h := range hashes {
		args[i] = h
	}
	rows, err := s.db.QueryContext(ctx, `SELECT hash, vec FROM vectors WHERE hash IN (?`+strings.Repeat(",?", len(hashes)-1)+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var h string
		var blob []byte
		if err := rows.Scan(&h, &blob); err != nil {
			return nil, err
		}
		if len(blob) > 0 { // empty marks a chunk the model cannot embed
			out[h] = decodeVector(blob)
		}
	}
	return out, rows.Err()
}

// putVector stores a chunk's vector; an empty vector marks the chunk as one
// the model refused, so it never blocks the queue again.
func (s *store) putVector(ctx context.Context, hash string, vec []float64) error {
	_, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO vectors (hash, vec) VALUES (?, ?)`, hash, encodeVector(vec))
	return err
}

// unembedded returns chunk hashes (with text) that have no vector yet for
// the background embedder: a topic's entry files and curated knowledge
// first, run evidence last, prose before data, newest files first.
func (s *store) unembedded(ctx context.Context, corpus string, n int) ([][2]string, error) {
	return s.texts(ctx, `
SELECT c.hash, chunks_fts.text FROM chunks c JOIN chunks_fts ON chunks_fts.rowid = c.id JOIN files f ON f.corpus = c.corpus AND f.path = c.path
WHERE c.corpus = ? AND `+prose+` AND NOT EXISTS (SELECT 1 FROM vectors v WHERE v.hash = c.hash)
GROUP BY c.hash
ORDER BY CASE
  WHEN c.path LIKE '%/SKILL.md' OR c.path LIKE '%/MISSION.md' OR c.path LIKE '%README.md' OR c.path LIKE '%MAP.md' OR c.feed = 'knowledge' THEN 0
  WHEN c.feed = 'runs' THEN 2 ELSE 1 END,
  (c.path NOT LIKE '%.md') ASC, MAX(f.mtime_ns) DESC LIMIT ?`, corpus, n)
}

// prose limits a chunk query (alias c) to the files the semantic leg covers:
// prose and notes. Data files (JSON, JSONL, CSV) are found lexically only;
// measured on the research home, their vectors crowd the neighbourhood and
// lower paraphrase recall (owners/C/evidence/search-eval).
const prose = `(c.path LIKE '%.md' OR c.path LIKE '%.markdown' OR c.path LIKE '%.txt' OR c.path LIKE '%.yaml' OR c.path LIKE '%.yml')`

// textsByHash returns the text of the given hashes that still have no vector.
func (s *store) textsByHash(ctx context.Context, corpus string, hashes []string) ([][2]string, error) {
	if len(hashes) == 0 {
		return nil, nil
	}
	args := []any{corpus}
	for _, h := range hashes {
		args = append(args, h)
	}
	return s.texts(ctx, `
SELECT c.hash, chunks_fts.text FROM chunks c JOIN chunks_fts ON chunks_fts.rowid = c.id
WHERE c.corpus = ? AND c.hash IN (?`+strings.Repeat(",?", len(hashes)-1)+`)
  AND `+prose+` AND NOT EXISTS (SELECT 1 FROM vectors v WHERE v.hash = c.hash)
GROUP BY c.hash`, args...)
}

func (s *store) texts(ctx context.Context, query string, args ...any) ([][2]string, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out [][2]string
	for rows.Next() {
		var h, t string
		if err := rows.Scan(&h, &t); err != nil {
			return nil, err
		}
		out = append(out, [2]string{h, t})
	}
	return out, rows.Err()
}

// eachVector streams every stored vector (the in-memory index load).
func (s *store) eachVector(ctx context.Context, fn func(string, []float32)) error {
	rows, err := s.db.QueryContext(ctx, `SELECT v.hash, v.vec FROM vectors v WHERE length(v.vec) > 0 AND EXISTS (SELECT 1 FROM chunks c WHERE c.hash = v.hash AND `+prose+`)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var h string
		var blob []byte
		if err := rows.Scan(&h, &blob); err != nil {
			return err
		}
		fn(h, decodeVector(blob))
	}
	return rows.Err()
}

// hashesWhere returns the chunk hashes in a topic and/or feed.
func (s *store) hashesWhere(ctx context.Context, corpus, topic, feed string) (map[string]bool, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT DISTINCT hash FROM chunks WHERE corpus = ? AND (? = '' OR topic = ?) AND (? = '' OR feed = ?)`, corpus, topic, topic, feed, feed)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			return nil, err
		}
		out[h] = true
	}
	return out, rows.Err()
}

// byHashes returns the live chunks carrying the given hashes, with the
// chunk head as snippet (there are no matched words to centre on).
func (s *store) byHashes(ctx context.Context, corpus string, hashes []string, topic, feed string) ([]candidate, error) {
	if len(hashes) == 0 {
		return nil, nil
	}
	args := []any{corpus, topic, topic, feed, feed}
	for _, h := range hashes {
		args = append(args, h)
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT c.id, c.path, c.topic, c.feed, c.line_start, c.line_end, c.heading, c.hash, chunks_fts.text, f.mtime_ns, f.truncated
FROM chunks c JOIN chunks_fts ON chunks_fts.rowid = c.id JOIN files f ON f.corpus = c.corpus AND f.path = c.path
WHERE c.corpus = ? AND (? = '' OR c.topic = ?) AND (? = '' OR c.feed = ?) AND c.hash IN (?`+strings.Repeat(",?", len(hashes)-1)+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.ID, &c.Path, &c.Topic, &c.Feed, &c.LineStart, &c.LineEnd, &c.Heading, &c.Hash, &c.Text, &c.MtimeNS, &c.Truncated); err != nil {
			return nil, err
		}
		c.Snippet = headSnippet(c.Text)
		out = append(out, c)
	}
	return out, rows.Err()
}

func headSnippet(text string) string {
	fields := strings.Fields(text)
	out := strings.Join(fields, " ")
	if r := []rune(out); len(r) > 280 {
		out = string(r[:280]) + " …"
	}
	return out
}

func (s *store) chunkCount(ctx context.Context, corpus string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `
SELECT COUNT(DISTINCT c.hash) FROM chunks c
WHERE c.corpus = ? AND `+prose+` AND NOT EXISTS (SELECT 1 FROM vectors v WHERE v.hash = c.hash AND length(v.vec) = 0)`, corpus).Scan(&n)
	return n, err
}
