package catalog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/database"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// EnsureMigrations keeps existing local SQLite databases readable after the
// catalog gained projection metadata and identity constraints. Duplicate data
// is never deleted here: the service remains readable and names catalog-merge
// as the only repair path.
type migrationDB interface {
	database.SchemaExecer
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func EnsureMigrations(ctx context.Context, db migrationDB) error {
	columnExists := func(table, column string) (bool, error) {
		rows, err := db.QueryContext(ctx, `PRAGMA table_info(`+table+`)`)
		if err != nil {
			return false, err
		}
		defer rows.Close()
		for rows.Next() {
			var cid, notnull, pk int
			var name, typ string
			var defaultValue any
			if err := rows.Scan(&cid, &name, &typ, &notnull, &defaultValue, &pk); err != nil {
				return false, err
			}
			if name == column {
				return true, nil
			}
		}
		return false, rows.Err()
	}
	found, err := columnExists("nodes", "actual_account_id")
	if err != nil {
		return err
	}
	if !found {
		if _, err = db.ExecContext(ctx, `ALTER TABLE nodes ADD COLUMN actual_account_id TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	for _, tableColumn := range []struct{ table, column, statement string }{
		{table: "edges", column: "intended_price_declared", statement: `ALTER TABLE edges ADD COLUMN intended_price_declared INTEGER NOT NULL DEFAULT 0`},
		{table: "triggers", column: "clauses_json", statement: `ALTER TABLE triggers ADD COLUMN clauses_json TEXT NOT NULL DEFAULT '[]'`},
		{table: "triggers", column: "composition", statement: `ALTER TABLE triggers ADD COLUMN composition INTEGER NOT NULL DEFAULT 1`},
		{table: "facts", column: "dimension", statement: `ALTER TABLE facts ADD COLUMN dimension TEXT NOT NULL DEFAULT ''`},
		{table: "catalog_audit", column: "related_node_id", statement: `ALTER TABLE catalog_audit ADD COLUMN related_node_id TEXT NOT NULL DEFAULT ''`},
	} {
		present, err := columnExists(tableColumn.table, tableColumn.column)
		if err != nil {
			return err
		}
		if !present {
			if _, err := db.ExecContext(ctx, tableColumn.statement); err != nil {
				return err
			}
		}
	}
	duplicate, err := duplicateGroup(ctx, db, `SELECT kind, name, COUNT(*) FROM nodes GROUP BY kind, name HAVING COUNT(*) > 1 ORDER BY kind, name LIMIT 1`)
	if err != nil {
		return err
	}
	if duplicate != "" {
		return fmt.Errorf("catalog identity migration refused: duplicate node identity %s; repair with offer-desk offers catalog-merge", duplicate)
	}
	duplicate, err = duplicateGroup(ctx, db, `SELECT from_id || ' -> ' || to_id || ' [' || kind || ']', '', COUNT(*) FROM edges GROUP BY from_id, to_id, kind HAVING COUNT(*) > 1 ORDER BY from_id, to_id, kind LIMIT 1`)
	if err != nil {
		return err
	}
	if duplicate != "" {
		return fmt.Errorf("catalog identity migration refused: duplicate edge identity %s; repair with offer-desk offers catalog-merge", duplicate)
	}
	if _, err := db.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS nodes_kind_name ON nodes(kind,name)`); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS edges_from_to_kind ON edges(from_id,to_id,kind)`); err != nil {
		return err
	}
	return nil
}

func duplicateGroup(ctx context.Context, db migrationDB, query string) (string, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	if !rows.Next() {
		return "", rows.Err()
	}
	var first, second string
	var count int
	if err := rows.Scan(&first, &second, &count); err != nil {
		return "", err
	}
	if second == "" {
		return fmt.Sprintf("%s (%d rows)", first, count), nil
	}
	return fmt.Sprintf("(%s,%s) (%d rows)", first, second, count), nil
}

func (s *Store) upsertImportedNode(ctx context.Context, kind offerspb.NodeKind, name string, status offerspb.Status, actor string) (*offerspb.Node, bool, error) {
	node := &offerspb.Node{}
	var storedKind, storedStatus int32
	var created string
	lookupErr := s.db.QueryRowContext(ctx, `SELECT id,kind,name,status,trigger_id,created_at,actual_account_id FROM nodes WHERE kind=? AND name=?`, int32(kind), name).Scan(&node.Id, &storedKind, &node.Name, &storedStatus, &node.TriggerId, &created, &node.ActualAccountId)
	if lookupErr == nil {
		node.Kind = offerspb.NodeKind(storedKind)
		node.Status = offerspb.Status(storedStatus)
		parsed, _ := time.Parse(time.RFC3339Nano, created)
		node.CreatedAt = timestamppb.New(parsed)
		if node.Status == status {
			return node, false, nil
		}
		if _, err := s.db.ExecContext(ctx, `UPDATE nodes SET status=? WHERE id=?`, int32(status), node.Id); err != nil {
			return nil, false, err
		}
		if actor == "" {
			actor = "operator"
		}
		if _, err := s.db.ExecContext(ctx, `INSERT INTO catalog_audit(id,node_id,actor,prior_status,next_status,reason,created_at) VALUES(?,?,?,?,?,?,?)`, uuid.NewString(), node.Id, actor, int32(node.Status), int32(status), "fixture import status upsert", s.now().UTC().Format(time.RFC3339Nano)); err != nil {
			return nil, false, err
		}
		node.Status = status
		return node, true, nil
	}
	if lookupErr != sql.ErrNoRows {
		return nil, false, lookupErr
	}
	createdNode, err := s.CreateNode(ctx, kind, name, status, "", "")
	return createdNode, true, err
}

// ImportFileReport is durable migration evidence for one source file. A
// source is never removed by this package; callers may retire it only after
// every file has read==written and no blocking finding remains.
type ImportFileReport struct {
	Path     string
	Read     int
	Written  int
	Findings int
}

type ImportReport struct {
	Files    []ImportFileReport
	Findings int
}

var (
	markdownReference = regexp.MustCompile(`\]\(([^)#]+)(?:#[^)]*)?\)`)
	statusBare        = regexp.MustCompile(`(?im)^\s*\*\*Status:\*\*\s*` + "`?" + `([a-z-]+)` + "`?" + `(?:\s*\([^\n]*\))?\s*$`)
	statusList        = regexp.MustCompile(`(?im)^\s*-\s+\*\*Status:\*\*\s*` + "`?" + `([a-z-]+)` + "`?" + `(?:\s*\([^\n]*\))?\s*$`)
	statusInline      = regexp.MustCompile(`(?im)^\s*\*\*Status:\s*` + "`?" + `([a-z-]+)` + "`?" + `[^\n]*\*\*\s*$`)
)

func fixtureRoot(root string) (string, error) {
	abs, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil {
		return "", err
	}
	parts := strings.Split(filepath.ToSlash(abs), "/")
	for _, part := range parts {
		if part == "docs" {
			// The source tree is intentionally outside this importer's authority.
			return "", errors.New("catalog importer accepts fixture roots only; live docs/monetization is forbidden")
		}
	}
	for _, part := range parts {
		if part == "testdata" {
			return abs, nil
		}
	}
	return "", errors.New("catalog importer requires a configured testdata fixture root")
}

func importedKind(rel string) offerspb.NodeKind {
	switch {
	case strings.Contains(filepath.ToSlash(rel), "/channels/"):
		return offerspb.NodeKind_CHANNEL
	case strings.Contains(filepath.ToSlash(rel), "/revenue-lines/"):
		return offerspb.NodeKind_REVENUE_LINE
	case strings.Contains(filepath.ToSlash(rel), "/deliverables/"):
		return offerspb.NodeKind_DELIVERABLE
	default:
		return offerspb.NodeKind_OFFER
	}
}

func importedStatus(body string) offerspb.Status {
	status, found, _ := parseImportedStatus(body)
	if !found {
		if statusMarker := regexp.MustCompile(`(?im)^\s*(?:-\s+)?\*\*Status:`); statusMarker.FindStringIndex(body) != nil {
			return offerspb.Status_STATUS_UNSPECIFIED
		}
		return offerspb.Status_IDEA
	}
	return status
}

func parseImportedStatus(body string) (offerspb.Status, bool, int) {
	match := statusBare.FindStringSubmatch(body)
	if len(match) != 2 {
		match = statusList.FindStringSubmatch(body)
	}
	if len(match) != 2 {
		match = statusInline.FindStringSubmatch(body)
	}
	if len(match) != 2 {
		return offerspb.Status_STATUS_UNSPECIFIED, false, 0
	}
	line := 1 + strings.Count(body[:strings.Index(body, match[0])], "\n")
	switch strings.ToLower(strings.TrimSpace(match[1])) {
	case "active":
		return offerspb.Status_ACTIVE, true, line
	case "candidate":
		return offerspb.Status_CANDIDATE, true, line
	case "north-star":
		return offerspb.Status_IDEA, true, line
	case "shipped":
		return offerspb.Status_SHIPPED, true, line
	case "retired":
		return offerspb.Status_RETIRED, true, line
	case "trigger-met":
		return offerspb.Status_TRIGGER_MET, true, line
	case "proposed":
		return offerspb.Status_PROPOSED, true, line
	default:
		return offerspb.Status_STATUS_UNSPECIFIED, false, line
	}
}

// ImportTree imports only lifecycle-bearing source records. Narrative prose is
// deliberately not copied into the catalog. Each markdown file becomes an
// idea node, and broken relative links become durable findings attached to it.
func (s *Store) ImportTree(ctx context.Context, root, actor string) (*ImportReport, error) {
	root, err := fixtureRoot(root)
	if err != nil {
		return nil, err
	}
	var paths []string
	err = filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.Mode().IsRegular() && strings.EqualFold(filepath.Ext(path), ".md") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	report := &ImportReport{}
	for _, path := range paths {
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, readErr
		}
		rel, _ := filepath.Rel(root, path)
		kind := importedKind(rel)
		status := importedStatus(string(body))
		if status == offerspb.Status_STATUS_UNSPECIFIED || status == offerspb.Status_CANDIDATE {
			// Fixture mode preserves the predecessor importer's permissive,
			// trigger-free fixture behavior. Operator mode uses the strict
			// parser in ImportCatalog below.
			status = offerspb.Status_IDEA
		}
		name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		if match := regexp.MustCompile("(?m)^\\*\\*SKU ID:\\*\\*\\s*`?([^`\\s]+)").FindStringSubmatch(string(body)); len(match) == 2 {
			name = match[1]
		}
		node, written, createErr := s.upsertImportedNode(ctx, kind, name, status, actor)
		if createErr != nil {
			return nil, fmt.Errorf("import %s: upsert lifecycle node: %w", rel, createErr)
		}
		fileReport := ImportFileReport{Path: rel, Read: 1}
		if written {
			fileReport.Written = 1
		}
		for _, ref := range markdownReference.FindAllStringSubmatch(string(body), -1) {
			target := strings.TrimSpace(ref[1])
			if target == "" || strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
				continue
			}
			candidate := filepath.Clean(filepath.Join(filepath.Dir(path), target))
			if filepath.Ext(candidate) == "" {
				candidate += ".md"
			}
			if _, statErr := os.Stat(candidate); statErr != nil {
				fileReport.Findings++
				report.Findings++
				if _, insertErr := s.db.ExecContext(ctx, `INSERT INTO migration_findings(id,node_id,source_file,reference,reason,created_at) SELECT ?,?,?,?,?,? WHERE NOT EXISTS (SELECT 1 FROM migration_findings WHERE node_id=? AND source_file=? AND reference=? AND reason=?)`, uuid.NewString(), node.Id, rel, target, "unresolvable internal reference", s.now().UTC().Format(time.RFC3339Nano), node.Id, rel, target, "unresolvable internal reference"); insertErr != nil {
					return nil, insertErr
				}
			}
		}
		report.Files = append(report.Files, fileReport)
	}
	return report, nil
}
