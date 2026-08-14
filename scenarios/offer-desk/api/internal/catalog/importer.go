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
)

// EnsureMigrations keeps existing local SQLite databases readable after the
// catalog gained actual-account projection metadata. Schema bootstrap creates
// the column for new databases; this additive check handles brownfield ones.
type migrationDB interface {
	database.SchemaExecer
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func EnsureMigrations(ctx context.Context, db migrationDB) error {
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(nodes)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &typ, &notnull, &defaultValue, &pk); err != nil {
			rows.Close()
			return err
		}
		if name == "actual_account_id" {
			found = true
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if !found {
		if _, err = db.ExecContext(ctx, `ALTER TABLE nodes ADD COLUMN actual_account_id TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	for _, tableColumn := range []struct{ table, column, statement string }{
		{table: "triggers", column: "clauses_json", statement: `ALTER TABLE triggers ADD COLUMN clauses_json TEXT NOT NULL DEFAULT '[]'`},
		{table: "triggers", column: "composition", statement: `ALTER TABLE triggers ADD COLUMN composition INTEGER NOT NULL DEFAULT 1`},
		{table: "facts", column: "dimension", statement: `ALTER TABLE facts ADD COLUMN dimension TEXT NOT NULL DEFAULT ''`},
	} {
		rows, err := db.QueryContext(ctx, `PRAGMA table_info(`+tableColumn.table+`)`)
		if err != nil {
			return err
		}
		defer rows.Close()
		present := false
		for rows.Next() {
			var cid, notnull, pk int
			var name, typ string
			var defaultValue any
			if err := rows.Scan(&cid, &name, &typ, &notnull, &defaultValue, &pk); err != nil {
				rows.Close()
				return err
			}
			if name == tableColumn.column {
				present = true
			}
		}
		if err := rows.Close(); err != nil {
			return err
		}
		if !present {
			if _, err := db.ExecContext(ctx, tableColumn.statement); err != nil {
				return err
			}
		}
	}
	return nil
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
	lifecycleStatus   = regexp.MustCompile("(?im)^\\*\\*Status:\\*\\*\\s*`?([a-z-]+)`?\\s*$")
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
	match := lifecycleStatus.FindStringSubmatch(body)
	if len(match) != 2 {
		return offerspb.Status_IDEA
	}
	switch strings.ToLower(match[1]) {
	case "active":
		return offerspb.Status_ACTIVE
	case "shipped":
		return offerspb.Status_SHIPPED
	case "retired":
		return offerspb.Status_RETIRED
	case "trigger-met":
		return offerspb.Status_TRIGGER_MET
	default:
		// Candidate requires a trigger ID, and narrative source files do not
		// carry a machine trigger. Preserve the source as an idea and let the
		// finding explain the deferred lifecycle step.
		return offerspb.Status_IDEA
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
		name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		if match := regexp.MustCompile("(?m)^\\*\\*SKU ID:\\*\\*\\s*`?([^`\\s]+)").FindStringSubmatch(string(body)); len(match) == 2 {
			name = match[1]
		}
		node, createErr := s.CreateNode(ctx, kind, name, status, "", "")
		if createErr != nil {
			return nil, fmt.Errorf("import %s: create lifecycle node: %w", rel, createErr)
		}
		fileReport := ImportFileReport{Path: rel, Read: 1, Written: 1}
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
				if _, insertErr := s.db.ExecContext(ctx, `INSERT INTO migration_findings(id,node_id,source_file,reference,reason,created_at) VALUES(?,?,?,?,?,?)`, uuid.NewString(), node.Id, rel, target, "unresolvable internal reference", s.now().UTC().Format(time.RFC3339Nano)); insertErr != nil {
					return nil, insertErr
				}
			}
		}
		report.Files = append(report.Files, fileReport)
	}
	_ = actor // lifecycle writes are operator-owned; retained for audit seam expansion.
	return report, nil
}
