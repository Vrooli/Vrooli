package retrieval

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
)

type Document struct {
	ID           string
	SourceFileID string
	SourceHash   string
	Path         string
	Language     string
	Role         string
	Scope        string
	Authority    string
	Kind         string
	Title        string
	ExactText    string
	Body         string
	ContractText string
	Aliases      []string
	StartLine    int
	EndLine      int
}

// SourceRecord is the catalog half of one atomic ordinary-file refresh.
// Policy/model changes still use isolated shadow generations; this seam keeps
// a file's catalog hash and FTS rows transactionally aligned in the active
// generation.
type SourceRecord struct {
	ID, Path, Language, Role, Scope, Authority, Owner, Hash string
	Size, ModTimeUnixNano                                   int64
	Searchable                                              bool
}

type SQLiteIndex struct {
	db      *sql.DB
	planner QueryPlanner
}

func NewSQLiteIndex(db *sql.DB) *SQLiteIndex { return &SQLiteIndex{db: db} }

func (index *SQLiteIndex) Upsert(ctx context.Context, generation string, documents []Document) error {
	if index == nil || index.db == nil {
		return fmt.Errorf("lexical index requires database")
	}
	if strings.TrimSpace(generation) == "" {
		return fmt.Errorf("lexical index requires generation")
	}
	tx, err := index.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin lexical batch: %w", err)
	}
	defer tx.Rollback()
	var state string
	if err := tx.QueryRowContext(ctx, `SELECT state FROM code_facts_generations WHERE id=?`, generation).Scan(&state); err != nil {
		return fmt.Errorf("read lexical generation: %w", err)
	}
	if state != "shadow" {
		return fmt.Errorf("lexical generation %q is %s, want shadow", generation, state)
	}
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO code_facts_search_documents(
 generation_id,id,source_file_id,source_hash,path,language,role,scope,authority,kind,
 title,exact_text,split_text,path_text,body,contract_text,aliases,start_line,end_line
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(generation_id,id) DO UPDATE SET
 source_file_id=excluded.source_file_id,source_hash=excluded.source_hash,path=excluded.path,
 language=excluded.language,role=excluded.role,scope=excluded.scope,authority=excluded.authority,
 kind=excluded.kind,title=excluded.title,exact_text=excluded.exact_text,split_text=excluded.split_text,
 path_text=excluded.path_text,body=excluded.body,contract_text=excluded.contract_text,
 aliases=excluded.aliases,start_line=excluded.start_line,end_line=excluded.end_line`)
	if err != nil {
		return fmt.Errorf("prepare lexical batch: %w", err)
	}
	defer stmt.Close()
	for _, document := range documents {
		if err := validateDocument(document); err != nil {
			return err
		}
		exactText := document.ExactText
		if exactText == "" {
			exactText = document.Title
		}
		splitText := normalizeDocumentText(strings.Join([]string{document.Title, exactText, strings.Join(document.Aliases, " ")}, " "))
		pathText := normalizeDocumentText(document.Path)
		if _, err := stmt.ExecContext(ctx, generation, document.ID, document.SourceFileID, document.SourceHash,
			document.Path, document.Language, document.Role, document.Scope, document.Authority, document.Kind,
			document.Title, normalizeExact(exactText), splitText, pathText, document.Body,
			document.ContractText, strings.Join(document.Aliases, " "), document.StartLine, document.EndLine); err != nil {
			return fmt.Errorf("upsert lexical document %q: %w", document.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit lexical batch: %w", err)
	}
	return nil
}

func (index *SQLiteIndex) Delete(ctx context.Context, generation string, ids []string) error {
	if index == nil || index.db == nil {
		return fmt.Errorf("lexical index requires database")
	}
	tx, err := index.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx, `DELETE FROM code_facts_search_documents WHERE generation_id=? AND id=?`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, id := range ids {
		if _, err := stmt.ExecContext(ctx, generation, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ApplySourceChange atomically replaces or deletes one source and all of its
// lexical rows. A nil source represents deletion. The generation digest and
// freshness timestamp move in the same commit, so readers observe either the
// old complete file or the new complete file, never a mixed projection.
func (index *SQLiteIndex) ApplySourceChange(ctx context.Context, generation, sourceID string, source *SourceRecord, documents []Document) error {
	if index == nil || index.db == nil || strings.TrimSpace(generation) == "" || strings.TrimSpace(sourceID) == "" {
		return fmt.Errorf("atomic source refresh requires index, generation, and source id")
	}
	tx, err := index.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin atomic source refresh: %w", err)
	}
	defer tx.Rollback()
	var state string
	if err := tx.QueryRowContext(ctx, `SELECT state FROM code_facts_generations WHERE id=?`, generation).Scan(&state); err != nil {
		return fmt.Errorf("read refresh generation: %w", err)
	}
	if state != "active" && state != "shadow" {
		return fmt.Errorf("source refresh generation %q is %s", generation, state)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM code_facts_search_documents WHERE generation_id=? AND source_file_id=?`, generation, sourceID); err != nil {
		return fmt.Errorf("delete stale search documents: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM code_facts_source_files WHERE generation_id=? AND id=?`, generation, sourceID); err != nil {
		return fmt.Errorf("delete stale catalog source: %w", err)
	}
	if source != nil {
		searchable := 0
		if source.Searchable {
			searchable = 1
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO code_facts_source_files(
generation_id,id,path,language,role,scope,authority,owner,content_hash,size_bytes,mod_time_unix_nano,searchable
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, generation, source.ID, source.Path, source.Language, source.Role, source.Scope, source.Authority, source.Owner, source.Hash, source.Size, source.ModTimeUnixNano, searchable); err != nil {
			return fmt.Errorf("insert refreshed catalog source: %w", err)
		}
		for _, document := range documents {
			if err := validateDocument(document); err != nil {
				return err
			}
			exactText := document.ExactText
			if exactText == "" {
				exactText = document.Title
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO code_facts_search_documents(
generation_id,id,source_file_id,source_hash,path,language,role,scope,authority,kind,title,exact_text,split_text,path_text,body,contract_text,aliases,start_line,end_line
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, generation, document.ID, document.SourceFileID, document.SourceHash, document.Path, document.Language, document.Role, document.Scope, document.Authority, document.Kind, document.Title, normalizeExact(exactText), normalizeDocumentText(strings.Join([]string{document.Title, exactText, strings.Join(document.Aliases, " ")}, " ")), normalizeDocumentText(document.Path), document.Body, document.ContractText, strings.Join(document.Aliases, " "), document.StartLine, document.EndLine); err != nil {
				return fmt.Errorf("insert refreshed search document %q: %w", document.ID, err)
			}
		}
	}
	digest := sha256.New()
	rows, err := tx.QueryContext(ctx, `SELECT path,content_hash FROM code_facts_source_files WHERE generation_id=? ORDER BY path`, generation)
	if err != nil {
		return fmt.Errorf("read refreshed source digest: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var path, hash string
		if err := rows.Scan(&path, &hash); err != nil {
			rows.Close()
			return err
		}
		_, _ = digest.Write([]byte(path))
		_, _ = digest.Write([]byte{0})
		_, _ = digest.Write([]byte(hash))
		_, _ = digest.Write([]byte{0})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate refreshed source digest: %w", err)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE code_facts_generations SET source_digest=?,updated_at_unix=unixepoch() WHERE id=?`, "sha256:"+hex.EncodeToString(digest.Sum(nil)), generation); err != nil {
		return fmt.Errorf("update refreshed generation digest: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit atomic source refresh: %w", err)
	}
	return nil
}

func (index *SQLiteIndex) SearchLexical(ctx context.Context, query Query) ([]Candidate, error) {
	if index == nil || index.db == nil {
		return nil, fmt.Errorf("lexical index requires database")
	}
	var err error
	query, err = NormalizeQuery(query)
	if err != nil {
		return nil, err
	}
	if query.Limit <= 0 {
		query.Limit = 10
	}
	if query.Limit > 100 {
		query.Limit = 100
	}
	generation, err := index.resolveGeneration(ctx, query.Generation)
	if err != nil {
		return nil, err
	}
	plan := index.planner.Plan(query)
	results, err := index.exact(ctx, generation, query, plan.Regime)
	if err != nil {
		return nil, err
	}
	if plan.Regime == RegimeExact && len(results) > 0 {
		return results, nil
	}
	if len(results) < query.Limit {
		remaining, err := index.fts(ctx, generation, query, plan.Regime, query.Limit-len(results), resultIDs(results))
		if err != nil {
			return nil, err
		}
		results = append(results, remaining...)
	}
	if len(results) > query.Limit {
		results = results[:query.Limit]
	}
	return results, nil
}

func (index *SQLiteIndex) resolveGeneration(ctx context.Context, requested string) (string, error) {
	if requested == "" {
		var generation string
		if err := index.db.QueryRowContext(ctx, `SELECT id FROM code_facts_generations WHERE state='active'`).Scan(&generation); err != nil {
			return "", fmt.Errorf("resolve active lexical generation: %w", err)
		}
		return generation, nil
	}
	var state string
	if err := index.db.QueryRowContext(ctx, `SELECT state FROM code_facts_generations WHERE id=?`, requested).Scan(&state); err != nil {
		return "", fmt.Errorf("resolve lexical generation: %w", err)
	}
	if state != "active" && state != "shadow" {
		return "", fmt.Errorf("lexical generation %q is not queryable", requested)
	}
	return requested, nil
}

func (index *SQLiteIndex) Current(ctx context.Context, candidate Candidate) (bool, error) {
	if index == nil || index.db == nil {
		return false, fmt.Errorf("freshness fence requires database")
	}
	var count int
	err := index.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM code_facts_source_files f
JOIN code_facts_generations g ON g.id=f.generation_id
WHERE f.generation_id=? AND f.content_hash=? AND f.path=? AND g.state IN ('active','shadow')`,
		candidate.Generation, candidate.SourceHash, candidate.Path).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("apply retrieval freshness fence: %w", err)
	}
	return count == 1, nil
}

func (index *SQLiteIndex) exact(ctx context.Context, generation string, query Query, regime Regime) ([]Candidate, error) {
	normalized := normalizeExact(query.Text)
	roles := strings.Join(query.Roles, ",")
	families := strings.Join(query.Families, ",")
	languages := strings.Join(query.Languages, ",")
	identityRows, err := index.db.QueryContext(ctx, `
SELECT d.id,d.path,d.title,d.body,d.source_hash,d.generation_id,d.role,d.language,d.scope,d.authority,d.kind,d.start_line,d.end_line,1.0
FROM code_facts_search_documents d
JOIN code_facts_source_files f ON f.generation_id=d.generation_id AND f.id=d.source_file_id AND f.content_hash=d.source_hash
WHERE d.generation_id=? AND d.exact_text=?
  AND (?='' OR d.scope=?) AND (?='' OR instr(','||?||',',','||d.language||',')>0)
	AND (?='' OR instr(','||?||',',','||d.role||',')>0)
	AND (?='' OR instr(','||?||',',','||d.kind||',')>0)
	AND (?='' OR d.path LIKE ?||'%')
ORDER BY d.id LIMIT ?`, generation, normalized,
		query.Scope, query.Scope, languages, languages, roles, roles, families, families,
		query.Target, query.Target, query.Limit)
	if err != nil {
		return nil, fmt.Errorf("exact identity query: %w", err)
	}
	identities, err := scanCandidates(identityRows, regime, "exact identifier match", "exact")
	identityRows.Close()
	if err != nil {
		return nil, err
	}
	if len(identities) > 0 {
		applyRankingPolicy(identities, query)
		return identities, nil
	}
	rows, err := index.db.QueryContext(ctx, `
SELECT d.id,d.path,d.title,d.body,d.source_hash,d.generation_id,d.role,d.language,d.scope,d.authority,d.kind,d.start_line,d.end_line,
 CASE WHEN d.exact_text=? THEN 1.0 WHEN lower(d.path)=? THEN 0.95 WHEN lower(d.path) LIKE ? THEN 0.90 ELSE 0.85 END
FROM code_facts_search_documents d
JOIN code_facts_source_files f ON f.generation_id=d.generation_id AND f.id=d.source_file_id AND f.content_hash=d.source_hash
WHERE d.generation_id=? AND (d.exact_text=? OR lower(d.path)=? OR lower(d.path) LIKE ? OR d.exact_text LIKE ?)
  AND (?='' OR d.scope=?) AND (?='' OR instr(','||?||',',','||d.language||',')>0)
	AND (?='' OR instr(','||?||',',','||d.role||',')>0)
	AND (?='' OR instr(','||?||',',','||d.kind||',')>0)
	AND (?='' OR d.path LIKE ?||'%')
ORDER BY 14 DESC,d.id LIMIT ?`, normalized, normalized, "%"+normalized,
		generation, normalized, normalized, "%"+normalized, normalized+"%",
		query.Scope, query.Scope, languages, languages,
		roles, roles, families, families, query.Target, query.Target, query.Limit)
	if err != nil {
		return nil, fmt.Errorf("exact lexical query: %w", err)
	}
	defer rows.Close()
	candidates, err := scanCandidates(rows, regime, "exact identity or path match", "exact")
	applyRankingPolicy(candidates, query)
	return candidates, err
}

func (index *SQLiteIndex) fts(ctx context.Context, generation string, query Query, regime Regime, limit int, excluded map[string]struct{}) ([]Candidate, error) {
	match := safeMatch(query.Text)
	if match == "" || limit <= 0 {
		return nil, nil
	}
	roles := strings.Join(query.Roles, ",")
	families := strings.Join(query.Families, ",")
	languages := strings.Join(query.Languages, ",")
	rows, err := index.db.QueryContext(ctx, `
SELECT d.id,d.path,d.title,snippet(code_facts_search_fts,4,'','', ' … ', 64),d.source_hash,d.generation_id,d.role,d.language,d.scope,d.authority,d.kind,d.start_line,d.end_line,
 bm25(code_facts_search_fts,6.0,10.0,5.0,7.0,1.0,4.0,5.0)
FROM code_facts_search_fts
JOIN code_facts_search_documents d ON d.rowid=code_facts_search_fts.rowid
JOIN code_facts_source_files f ON f.generation_id=d.generation_id AND f.id=d.source_file_id AND f.content_hash=d.source_hash
WHERE code_facts_search_fts MATCH ? AND d.generation_id=?
  AND (?='' OR d.scope=?) AND (?='' OR instr(','||?||',',','||d.language||',')>0)
  AND (?='' OR instr(','||?||',',','||d.role||',')>0)
	AND (?='' OR instr(','||?||',',','||d.kind||',')>0)
	AND (?='' OR d.path LIKE ?||'%')
ORDER BY 14,d.id LIMIT ?`, match, generation, query.Scope, query.Scope,
		languages, languages, roles, roles, families, families,
		query.Target, query.Target, limit+len(excluded))
	if err != nil {
		return nil, fmt.Errorf("FTS lexical query: %w", err)
	}
	defer rows.Close()
	candidates, err := scanCandidates(rows, regime, "weighted BM25 over indexed evidence", "bm25")
	if err != nil {
		return nil, err
	}
	filtered := candidates[:0]
	for _, candidate := range candidates {
		if _, found := excluded[candidate.ID]; found {
			continue
		}
		if lexicalCoverage(candidate, query.Text) < 0.4 {
			continue
		}
		magnitude := math.Abs(candidate.Score)
		candidate.Score = magnitude / (1 + magnitude)
		candidate.ScoreFactors["bm25_normalized"] = candidate.Score
		filtered = append(filtered, candidate)
		if len(filtered) == limit {
			break
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].Score != filtered[j].Score {
			return filtered[i].Score > filtered[j].Score
		}
		return filtered[i].ID < filtered[j].ID
	})
	applyRankingPolicy(filtered, query)
	return filtered, nil
}

func applyRankingPolicy(candidates []Candidate, query Query) {
	for index := range candidates {
		candidate := &candidates[index]
		boost := 0.0
		if candidate.Authority == "authoritative" {
			candidate.ScoreFactors["authority_boost"] = 0.08
			boost += 0.08
		}
		switch candidate.Role {
		case "implementation", "contract":
			candidate.ScoreFactors["role_boost"] = 0.06
			boost += 0.06
		case "generated_alias", "fixture", "test":
			candidate.ScoreFactors["role_demote"] = -0.04
			boost -= 0.04
		}
		if query.Scope != "" && candidate.Scope == query.Scope {
			candidate.ScoreFactors["scope_boost"] = 0.04
			boost += 0.04
		}
		if strings.Contains(strings.ToLower(candidate.Path), normalizeExact(query.Text)) {
			candidate.ScoreFactors["path_boost"] = 0.08
			boost += 0.08
		}
		// Apply policy inside the remaining score headroom. Additive boosts made
		// every strong BM25 hit saturate at 1.0, erasing lexical ordering.
		if boost >= 0 {
			candidate.Score += boost * (1 - candidate.Score)
		} else {
			candidate.Score *= 1 + boost
		}
		if candidate.Score < 0 {
			candidate.Score = 0
		}
		candidate.Explanation += "; documented authority, role, scope, and path policy applied"
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		return candidates[i].ID < candidates[j].ID
	})
}

var lexicalStopWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "are": {}, "as": {}, "at": {}, "be": {}, "by": {}, "code": {}, "does": {},
	"file": {}, "files": {}, "for": {}, "from": {}, "handles": {}, "how": {}, "implementation": {}, "implemented": {}, "implements": {},
	"in": {}, "into": {}, "is": {}, "it": {}, "location": {}, "method": {}, "module": {}, "of": {}, "on": {}, "or": {},
	"owns": {}, "package": {}, "source": {}, "that": {}, "the": {}, "this": {}, "to": {}, "what": {}, "where": {}, "which": {}, "who": {},
}

func lexicalCoverage(candidate Candidate, query string) float64 {
	queryTokens := splitIdentifier(query)
	evidenceTokens := splitIdentifier(strings.Join([]string{candidate.Title, candidate.Text, candidate.Path}, " "))
	evidence := make(map[string]struct{}, len(evidenceTokens))
	for _, token := range evidenceTokens {
		evidence[token] = struct{}{}
	}
	seen := map[string]struct{}{}
	matched, total := 0, 0
	for _, token := range queryTokens {
		if _, ignored := lexicalStopWords[token]; ignored {
			continue
		}
		if _, duplicate := seen[token]; duplicate {
			continue
		}
		seen[token] = struct{}{}
		total++
		if _, found := evidence[token]; found {
			matched++
		}
	}
	if total == 0 {
		return 0
	}
	return float64(matched) / float64(total)
}

func scanCandidates(rows *sql.Rows, regime Regime, explanation, factor string) ([]Candidate, error) {
	var candidates []Candidate
	for rows.Next() {
		var candidate Candidate
		if err := rows.Scan(&candidate.ID, &candidate.Path, &candidate.Title, &candidate.Text, &candidate.SourceHash,
			&candidate.Generation, &candidate.Role, &candidate.Language, &candidate.Scope, &candidate.Authority,
			&candidate.Kind, &candidate.StartLine, &candidate.EndLine, &candidate.Score); err != nil {
			return nil, err
		}
		candidate.Regime = regime
		candidate.Explanation = explanation
		candidate.Evidence = "current_source_hash"
		candidate.Proof = "source_hash_verified"
		candidate.ScoreFactors = map[string]float64{factor: candidate.Score}
		candidate.RankEvidence = []RankEvidence{{Leg: "lexical", Rank: len(candidates) + 1, Score: candidate.Score}}
		candidates = append(candidates, candidate)
	}
	return candidates, rows.Err()
}

func safeMatch(text string) string {
	parts := splitIdentifier(text)
	unique := make(map[string]struct{}, len(parts))
	terms := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) < 2 {
			continue
		}
		if _, ignored := lexicalStopWords[part]; ignored {
			continue
		}
		if _, exists := unique[part]; exists {
			continue
		}
		unique[part] = struct{}{}
		terms = append(terms, `"`+strings.ReplaceAll(part, `"`, `""`)+`"*`)
		if len(terms) == 12 {
			break
		}
	}
	return strings.Join(terms, " OR ")
}

func validateDocument(document Document) error {
	if strings.TrimSpace(document.ID) == "" || strings.TrimSpace(document.SourceFileID) == "" || strings.TrimSpace(document.SourceHash) == "" || strings.TrimSpace(document.Path) == "" || strings.TrimSpace(document.Title) == "" {
		return errors.New("lexical document requires id, source file, source hash, path, and title")
	}
	if document.StartLine < 0 || document.EndLine < document.StartLine {
		return fmt.Errorf("lexical document %q has invalid source range", document.ID)
	}
	return nil
}

func resultIDs(candidates []Candidate) map[string]struct{} {
	ids := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		ids[candidate.ID] = struct{}{}
	}
	return ids
}

var _ LexicalSearcher = (*SQLiteIndex)(nil)
