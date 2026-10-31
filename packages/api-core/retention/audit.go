package retention

import (
	"context"
	"fmt"
	"sort"
)

// A budget bounds the table it names. Nothing bounds the tables nobody named,
// and that gap is not theoretical: autoheal declared ceilings for
// health_results, host_inventory_snapshots, and system_events while
// incident_observations grew unattended alongside them, inside the same file
// those ceilings were meant to protect. The declaration looked complete because
// every budget in it was correct.
//
// This is the check that makes the gap visible. It answers "what in this
// database is nobody watching?" — the question a manifest cannot answer about
// itself, because a manifest only knows what it declares.

// UnbudgetedTableFloor is how large an undeclared table must be before the audit
// reports it.
//
// A floor rather than a full listing, because a warning naming eleven
// bookkeeping tables of two pages each is one nobody reads twice, and the next
// real finding arrives in a channel already known to be noise. 64 MiB is small
// enough to catch a table on its way up and large enough that steady-state
// config and cursor tables never appear.
const UnbudgetedTableFloor = 64 << 20

// UnbudgetedTable is one table in a budgeted database that no budget covers.
type UnbudgetedTable struct {
	// Database is the absolute path of the file the table lives in.
	Database string
	// Table is the undeclared table.
	Table string
	// Bytes is what it currently occupies, including its indexes.
	Bytes int64
}

// String renders the finding as one operator-readable line.
func (u UnbudgetedTable) String() string {
	return fmt.Sprintf("table %q in %s holds %s and no retention budget covers it",
		u.Table, u.Database, FormatBytes(u.Bytes))
}

// AuditUnbudgetedTables reports tables in the component's budgeted SQLite
// databases that exceed UnbudgetedTableFloor and that no budget bounds.
//
// It measures rather than merely lists, so what it reports is ranked by the only
// thing that matters about an unbounded table: how much of the disk it has taken
// so far. Results are ordered largest first.
//
// It is advisory and never modifies anything. An undeclared table is a decision
// the component's author has not made yet — it may be intentionally unbounded,
// small forever, or an oversight — and this package's job is to make sure the
// decision is made deliberately rather than discovered by a full disk.
func (m *Manager) AuditUnbudgetedTables(ctx context.Context) ([]UnbudgetedTable, error) {
	if m.openDatabase == nil {
		return nil, nil
	}

	// Group the declared tables by the database file that holds them: the
	// question is per-file, because a file is what fills a disk.
	budgeted := make(map[string]map[string]bool)
	for _, spec := range m.specs {
		if spec.Target.Kind != TargetSQLiteTable {
			continue
		}
		path, ok := m.paths[spec.Budget.Name]
		if !ok {
			continue
		}
		if budgeted[path] == nil {
			budgeted[path] = make(map[string]bool)
		}
		budgeted[path][spec.Target.Table] = true
	}

	var found []UnbudgetedTable
	for path, declared := range budgeted {
		db, err := m.openDatabase(path)
		if err != nil {
			return nil, fmt.Errorf("audit %s: %w", path, err)
		}
		tables, err := unbudgetedTablesIn(ctx, db, path, declared)
		if err != nil {
			return nil, err
		}
		found = append(found, tables...)
	}

	sort.Slice(found, func(i, j int) bool {
		if found[i].Bytes != found[j].Bytes {
			return found[i].Bytes > found[j].Bytes
		}
		return found[i].Table < found[j].Table
	})
	return found, nil
}

// unbudgetedTablesIn measures every undeclared user table in one database.
func unbudgetedTablesIn(ctx context.Context, db Execer, path string, declared map[string]bool) ([]UnbudgetedTable, error) {
	// sqlite_% is SQLite's own bookkeeping (sqlite_sequence, sqlite_stat1) and
	// is not the component's data to budget.
	rows, err := db.QueryContext(ctx,
		`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		return nil, fmt.Errorf("audit %s: list tables: %w", path, err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("audit %s: read table name: %w", path, err)
		}
		if !declared[name] {
			names = append(names, name)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("audit %s: list tables: %w", path, err)
	}

	var found []UnbudgetedTable
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		// Same measurement a budgeted table gets — the table's own pages plus
		// those of every index on it — so the numbers are comparable.
		const sizeQuery = `SELECT COALESCE(SUM(pgsize), 0) FROM dbstat WHERE name = ?
		    OR name IN (SELECT name FROM sqlite_master WHERE type = 'index' AND tbl_name = ?)`
		var bytes int64
		if err := db.QueryRowContext(ctx, sizeQuery, name, name).Scan(&bytes); err != nil {
			return nil, fmt.Errorf("audit %s: measure %s: %w", path, name, err)
		}
		if bytes >= UnbudgetedTableFloor {
			found = append(found, UnbudgetedTable{Database: path, Table: name, Bytes: bytes})
		}
	}
	return found, nil
}
