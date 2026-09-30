// Package reconcile moves rows that one scenario wrote into another scenario's
// database back to the database that owns them.
//
// # Why this exists
//
// Scenarios used to resolve their SQLite path from a GENERIC environment
// variable, above their own identity. The supervisor scenario declared that
// variable in its manifest and restarted sick scenarios by exec'ing the CLI, so
// each restarted child inherited the supervisor's value and opened the
// SUPERVISOR's database. Twelve scenarios were observed sharing one 9.35 GB
// file: Test Genie recorded 148 suite executions and 2,680 phase rows into it,
// plan-manager wrote 11 plans, git-control-tower 151 audit entries.
//
// The resolution defect is fixed in api-core/storage. Those rows are real
// history — they feed cost estimation, plan status, and audit trails — so they
// must be MOVED, not dropped.
//
// # Safety posture
//
// This package copies. It never deletes, and it is separate from any deletion
// step on purpose: a reconciliation that both copies and deletes has one
// failure mode where it has done half of each.
//
//   - Idempotent. Rows are inserted by primary key with INSERT OR IGNORE, so
//     running twice moves nothing the second time.
//   - Additive. An existing row in the target always wins; this tool cannot
//     overwrite data the owning scenario wrote itself.
//   - Column-intersecting. The two databases may carry different schema
//     versions of the same table, so only columns present in BOTH are copied.
//     A column missing from the target is reported rather than silently lost.
//   - Verifiable. Every table reports its source count, the rows inserted, and
//     the target count before and after, so the caller can prove the move.
//
// Deletion from the source is a separate, explicit operation. Run it only after
// the copy is verified and the scenario that owns the rows has been restarted
// onto its own database — otherwise it keeps writing into the source and the
// counts move again.
package reconcile

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

// TableResult records what happened to one table.
type TableResult struct {
	// Table is the table name.
	Table string
	// SourceRows is the row count found in the source database.
	SourceRows int64
	// TargetRowsBefore and TargetRowsAfter bracket the copy.
	TargetRowsBefore int64
	TargetRowsAfter  int64
	// Inserted is TargetRowsAfter minus TargetRowsBefore. It is less than
	// SourceRows when the target already held some of the rows.
	Inserted int64
	// CopiedColumns are the columns present in both schemas.
	CopiedColumns []string
	// DroppedColumns are columns the source has and the target does not. They
	// are reported rather than silently discarded: a non-empty list means the
	// two databases are at different schema versions and the caller should
	// decide whether the loss is acceptable.
	DroppedColumns []string
	// Identical counts source rows that already exist verbatim in the target.
	// They are not moved and are not a problem: the copy has simply already
	// happened, or both databases recorded the same fact.
	Identical int64
	// Conflicted counts source rows that could be moved neither as new rows nor
	// as duplicates, because they violate a unique constraint against a
	// DIFFERENT target row. These are the only rows a reconciliation leaves
	// behind, and they need a human decision: the same natural key describes
	// two different things in the two databases.
	Conflicted int64
	// SurrogateKey names the single INTEGER PRIMARY KEY column when the table
	// has one. Such an id is a per-database counter with no meaning across
	// databases, so it is NOT copied: the row is re-inserted and the target
	// assigns a fresh id. See reconcileTable for why this matters.
	SurrogateKey string
	// Skipped is set when the table was not copied, with Reason saying why.
	Skipped bool
	Reason  string
}

// Result is the outcome of one reconciliation.
type Result struct {
	// DryRun reports whether anything was written.
	DryRun bool
	// Tables holds one entry per requested table, in request order.
	Tables []TableResult
}

// TotalInserted sums the rows moved across all tables.
func (r Result) TotalInserted() int64 {
	var total int64
	for _, t := range r.Tables {
		total += t.Inserted
	}
	return total
}

// Options configures one reconciliation.
type Options struct {
	// SourcePath is the database rows were wrongly written into. It is opened
	// read-only, so a live writer is never blocked by this tool.
	SourcePath string
	// Tables are the tables to move, in the order they should be copied.
	// Order matters when foreign keys are involved: a parent table must
	// precede its children.
	Tables []string
	// DryRun reports what would move without writing anything. It defaults to
	// FALSE at the struct level, so callers must be explicit; the command-line
	// entry point defaults it to true.
	DryRun bool
}

// Run copies rows from the source database into target, which must already be
// open and is the database that OWNS the rows.
//
// The source is attached read-only. Foreign keys are left to the target's own
// setting: rows are copied parent-first in the caller's stated order, and a
// violation should surface as an error rather than be suppressed here.
func Run(ctx context.Context, target *sql.DB, opts Options) (Result, error) {
	result := Result{DryRun: opts.DryRun}

	if strings.TrimSpace(opts.SourcePath) == "" {
		return result, fmt.Errorf("reconcile: source path is required")
	}
	if len(opts.Tables) == 0 {
		return result, fmt.Errorf("reconcile: at least one table is required")
	}

	// Read-only + immutable: this tool must never write to, lock, or journal
	// the source. The source is typically a live database with an active
	// writer, and blocking that writer would turn a data-tidying task into an
	// outage.
	attach := fmt.Sprintf("ATTACH DATABASE 'file:%s?mode=ro&immutable=1' AS reconcile_source", opts.SourcePath)
	if _, err := target.ExecContext(ctx, attach); err != nil {
		return result, fmt.Errorf("attach source %s: %w", opts.SourcePath, err)
	}
	defer func() { _, _ = target.ExecContext(context.WithoutCancel(ctx), "DETACH DATABASE reconcile_source") }()

	for _, table := range opts.Tables {
		tr, err := reconcileTable(ctx, target, table, opts.DryRun)
		if err != nil {
			return result, fmt.Errorf("table %s: %w", table, err)
		}
		result.Tables = append(result.Tables, tr)
	}
	return result, nil
}

func reconcileTable(ctx context.Context, db *sql.DB, table string, dryRun bool) (TableResult, error) {
	tr := TableResult{Table: table}

	quoted, err := quoteIdentifier(table)
	if err != nil {
		return tr, err
	}

	sourceCols, err := columnsOf(ctx, db, "reconcile_source", table)
	if err != nil {
		return tr, err
	}
	if len(sourceCols) == 0 {
		tr.Skipped, tr.Reason = true, "table does not exist in the source database"
		return tr, nil
	}
	targetCols, err := columnsOf(ctx, db, "main", table)
	if err != nil {
		return tr, err
	}
	if len(targetCols) == 0 {
		tr.Skipped, tr.Reason = true, "table does not exist in the target database"
		return tr, nil
	}

	targetSet := make(map[string]bool, len(targetCols))
	for _, c := range targetCols {
		targetSet[c] = true
	}
	for _, c := range sourceCols {
		if targetSet[c] {
			tr.CopiedColumns = append(tr.CopiedColumns, c)
		} else {
			tr.DroppedColumns = append(tr.DroppedColumns, c)
		}
	}
	if len(tr.CopiedColumns) == 0 {
		tr.Skipped, tr.Reason = true, "the two schemas share no columns"
		return tr, nil
	}

	if tr.SourceRows, err = countRows(ctx, db, "reconcile_source", quoted); err != nil {
		return tr, err
	}
	if tr.TargetRowsBefore, err = countRows(ctx, db, "main", quoted); err != nil {
		return tr, err
	}
	tr.TargetRowsAfter = tr.TargetRowsBefore

	if tr.SourceRows == 0 {
		tr.Skipped, tr.Reason = true, "no rows in the source"
		return tr, nil
	}

	surrogate, err := surrogateKeyColumn(ctx, db, "reconcile_source", table)
	if err != nil {
		return tr, err
	}
	tr.SurrogateKey = surrogate

	// Columns to copy. A surrogate key is excluded so the target assigns its
	// own; see the comment on the statement below.
	cols := make([]string, 0, len(tr.CopiedColumns))
	plain := make([]string, 0, len(tr.CopiedColumns))
	for _, c := range tr.CopiedColumns {
		if surrogate != "" && c == surrogate {
			continue
		}
		q, qErr := quoteIdentifier(c)
		if qErr != nil {
			return tr, qErr
		}
		cols = append(cols, q)
		plain = append(plain, c)
	}
	if len(cols) == 0 {
		tr.Skipped, tr.Reason = true, "the table has no columns to copy once its surrogate key is excluded"
		return tr, nil
	}
	list := strings.Join(cols, ", ")

	var stmt string
	if surrogate == "" {
		// A meaningful primary key: the same key denotes the same row in both
		// databases. INSERT OR IGNORE is then both idempotent and
		// non-destructive — a row the owning scenario wrote itself wins.
		//
		// Count how many source rows the target already holds under the same
		// key, so a dry run can report what is genuinely left to move.
		keyCols, keyErr := primaryKeyColumns(ctx, db, "reconcile_source", table)
		if keyErr != nil {
			return tr, keyErr
		}
		if len(keyCols) > 0 {
			conds := make([]string, 0, len(keyCols))
			for _, c := range keyCols {
				q, qErr := quoteIdentifier(c)
				if qErr != nil {
					return tr, qErr
				}
				conds = append(conds, fmt.Sprintf("t.%s IS s.%s", q, q))
			}
			countQ := fmt.Sprintf(
				"SELECT COUNT(*) FROM reconcile_source.%s AS s WHERE EXISTS (SELECT 1 FROM main.%s AS t WHERE %s)",
				quoted, quoted, strings.Join(conds, " AND "))
			if err := db.QueryRowContext(ctx, countQ).Scan(&tr.Identical); err != nil {
				return tr, fmt.Errorf("count existing rows: %w", err)
			}
		}
		stmt = fmt.Sprintf(
			"INSERT OR IGNORE INTO main.%s (%s) SELECT %s FROM reconcile_source.%s",
			quoted, list, list, quoted,
		)
	} else {
		// A surrogate INTEGER PRIMARY KEY is a per-database counter. Source row
		// 7 and target row 7 are DIFFERENT rows that merely share a number, so
		// INSERT OR IGNORE would silently discard every source row whose id
		// happened to be taken — which is exactly what it did to 149 audit-log
		// rows before this branch existed, while reporting success.
		//
		// Instead the id is dropped and the target assigns a fresh one.
		// Idempotency then has to come from the DATA rather than the key, so a
		// row is copied only when no row with identical values already exists.
		// "IS" is used rather than "=" so NULL matches NULL; with "=" every row
		// containing a NULL would be re-inserted on each run.
		conds := make([]string, 0, len(cols))
		for _, c := range cols {
			conds = append(conds, fmt.Sprintf("t.%s IS s.%s", c, c))
		}
		notExists := fmt.Sprintf(
			"SELECT %s FROM reconcile_source.%s AS s "+
				"WHERE NOT EXISTS (SELECT 1 FROM main.%s AS t WHERE %s)",
			prefixColumns("s", cols), quoted, quoted, strings.Join(conds, " AND "),
		)

		// Count the rows that are already present verbatim, so the report can
		// separate "already moved" from "could not be moved".
		var candidates int64
		countQ := fmt.Sprintf("SELECT COUNT(*) FROM (%s)", notExists)
		if err := db.QueryRowContext(ctx, countQ).Scan(&candidates); err != nil {
			return tr, fmt.Errorf("count movable rows: %w", err)
		}
		tr.Identical = tr.SourceRows - candidates

		// OR IGNORE handles the remaining case: a source row that is not a
		// duplicate but collides with a DIFFERENT target row on a unique
		// constraint — the same repository path recorded under two ids, say.
		// Such a row cannot be merged automatically, so it is skipped and
		// counted rather than aborting the whole move.
		stmt = "INSERT OR IGNORE INTO main." + quoted + " (" + list + ") " + notExists
	}
	if dryRun {
		return tr, nil
	}
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return tr, fmt.Errorf("copy rows: %w", err)
	}

	if tr.TargetRowsAfter, err = countRows(ctx, db, "main", quoted); err != nil {
		return tr, err
	}
	tr.Inserted = tr.TargetRowsAfter - tr.TargetRowsBefore
	tr.Conflicted = tr.SourceRows - tr.Identical - tr.Inserted
	if tr.Conflicted < 0 {
		tr.Conflicted = 0
	}
	return tr, nil
}

// primaryKeyColumns returns the table's primary key columns, in key order.
func primaryKeyColumns(ctx context.Context, db *sql.DB, schema, table string) ([]string, error) {
	rows, err := db.QueryContext(ctx,
		"SELECT name FROM pragma_table_info(?, ?) WHERE pk > 0 ORDER BY pk", table, schema)
	if err != nil {
		return nil, fmt.Errorf("inspect primary key of %s.%s: %w", schema, table, err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

// prefixColumns qualifies each already-quoted column with a table alias.
func prefixColumns(alias string, cols []string) string {
	out := make([]string, 0, len(cols))
	for _, c := range cols {
		out = append(out, alias+"."+c)
	}
	return strings.Join(out, ", ")
}

// surrogateKeyColumn returns the name of the table's single INTEGER PRIMARY KEY
// column, or "" when the table has no such key.
//
// SQLite makes a lone "INTEGER PRIMARY KEY" an alias for the rowid: the value
// is assigned by that database's own counter and means nothing anywhere else. A
// composite key, or a key of any other declared type, is treated as meaningful
// — a TEXT run id identifies the same run in either database.
func surrogateKeyColumn(ctx context.Context, db *sql.DB, schema, table string) (string, error) {
	rows, err := db.QueryContext(ctx,
		"SELECT name, type, pk FROM pragma_table_info(?, ?) WHERE pk > 0", table, schema)
	if err != nil {
		return "", fmt.Errorf("inspect primary key of %s.%s: %w", schema, table, err)
	}
	defer rows.Close()

	var (
		keys     []string
		keyTypes []string
	)
	for rows.Next() {
		var name, declType string
		var pk int
		if err := rows.Scan(&name, &declType, &pk); err != nil {
			return "", err
		}
		keys = append(keys, name)
		keyTypes = append(keyTypes, strings.ToUpper(strings.TrimSpace(declType)))
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(keys) != 1 || keyTypes[0] != "INTEGER" {
		return "", nil
	}
	return keys[0], nil
}

// columnsOf returns the column names of schema.table, or nil when the table
// does not exist. Names are returned sorted so a result is reproducible.
func columnsOf(ctx context.Context, db *sql.DB, schema, table string) ([]string, error) {
	// The two-ARGUMENT form, pragma_table_info(table, schema), is the one that
	// actually honours the schema. The schema-QUALIFIED form
	// (schema.pragma_table_info(table)) parses without error but resolves
	// against main, so it silently reports the wrong table's columns — which
	// would make the column intersection below wrong in exactly the case this
	// tool exists for: two databases at different schema versions.
	rows, err := db.QueryContext(ctx, "SELECT name FROM pragma_table_info(?, ?)", table, schema)
	if err != nil {
		return nil, fmt.Errorf("inspect %s.%s: %w", schema, table, err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

func countRows(ctx context.Context, db *sql.DB, schema, quotedTable string) (int64, error) {
	quotedSchema, err := quoteIdentifier(schema)
	if err != nil {
		return 0, err
	}
	var n int64
	q := fmt.Sprintf("SELECT COUNT(*) FROM %s.%s", quotedSchema, quotedTable)
	if err := db.QueryRowContext(ctx, q).Scan(&n); err != nil {
		return 0, fmt.Errorf("count %s.%s: %w", schema, quotedTable, err)
	}
	return n, nil
}

// quoteIdentifier renders a SQLite identifier safely.
//
// Table and column names cannot be bound as parameters, so they are
// interpolated — which makes validation the only thing standing between a
// caller-supplied name and injected SQL. Names come from an operator-supplied
// list, so this rejects anything that is not a plain identifier rather than
// trying to escape it.
func quoteIdentifier(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", fmt.Errorf("identifier must not be empty")
	}
	for _, r := range trimmed {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_':
		default:
			return "", fmt.Errorf("invalid identifier %q: only letters, digits and underscore are allowed", name)
		}
	}
	return `"` + trimmed + `"`, nil
}
