package programs

import (
	"context"
	"fmt"
	"strings"

	_ "embed"
)

//go:embed schema.sql
var schema string

// Schema returns the programs domain schema for the central database.
func Schema() string { return schema }

// EnsureCompatibility upgrades the small SQLite schema in place for fields
// added after the initial scenario database was created. The scenario uses
// SQLite in every supported deployment, and keeping this additive migration
// here lets operators restart an existing installation without data loss.
func EnsureCompatibility(ctx context.Context, db SQLExecutor) error {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(programs)")
	if err != nil {
		return fmt.Errorf("inspect programs schema: %w", err)
	}
	defer rows.Close()
	found := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull, primaryKey int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return fmt.Errorf("scan programs schema: %w", err)
		}
		found[name] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read programs schema: %w", err)
	}
	columns := []struct{ name, definition string }{
		{"agent_bytes", "INTEGER NOT NULL DEFAULT 0"},
		{"wall_time_millis", "INTEGER NOT NULL DEFAULT 0"},
		{"cpu_time_millis", "INTEGER NOT NULL DEFAULT 0"},
		{"library_version", "TEXT NOT NULL DEFAULT ''"},
		{"failure_cause", "TEXT NOT NULL DEFAULT ''"},
	}
	for _, column := range columns {
		if found[column.name] {
			continue
		}
		if _, err := db.ExecContext(ctx, "ALTER TABLE programs ADD COLUMN "+column.name+" "+column.definition); err != nil {
			return fmt.Errorf("add programs.%s: %w", column.name, err)
		}
	}
	if err := reclassifyLegacyFailureShapes(ctx, db); err != nil {
		return err
	}
	return purgeNonCapabilityUnresolvedAttempts(ctx, db)
}

// purgeNonCapabilityUnresolvedAttempts removes ledger rows that only a resolver
// defect could have written.
//
// The pre-fix resolver treated lambda parameters, comprehension variables, loop
// targets, function names, and withheld builtins as unresolved capability
// requests, so names like `row`, `x`, `i`, `main`, `round`, and `KeyError`
// entered the unresolved-attempt ledger. That ledger is the Act denominator's
// feedback signal — "what did an agent try to invoke and could not" — so those
// rows are not merely noise, they are a false answer to the question the
// denominator asks. They are deleted rather than retained because no consumer
// can distinguish them from real attempts.
func purgeNonCapabilityUnresolvedAttempts(ctx context.Context, db SQLExecutor) error {
	rows, err := db.QueryContext(ctx, `SELECT DISTINCT attempted_name FROM unresolved_binding_attempts`)
	if err != nil {
		// The table is created by the bindings schema; a missing table here is
		// a startup ordering condition, not a failure worth blocking boot for.
		return nil
	}
	var doomed []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return fmt.Errorf("scan unresolved attempt: %w", err)
		}
		if !looksLikeCapabilityName(name) {
			doomed = append(doomed, name)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("read unresolved attempts: %w", err)
	}
	rows.Close()
	for _, name := range doomed {
		if _, err := db.ExecContext(ctx, `DELETE FROM unresolved_binding_attempts WHERE attempted_name = ?`, name); err != nil {
			return fmt.Errorf("purge unresolved attempt %q: %w", name, err)
		}
	}
	return nil
}

// pythonVocabulary mirrors the kernel's _PYTHON_VOCABULARY. A name here is a
// language-level miss, never a governed-capability request.
var pythonVocabulary = map[string]struct{}{
	"abs": {}, "all": {}, "any": {}, "ascii": {}, "bin": {}, "bool": {}, "bytearray": {}, "bytes": {},
	"callable": {}, "chr": {}, "classmethod": {}, "compile": {}, "complex": {}, "delattr": {}, "dict": {},
	"dir": {}, "divmod": {}, "enumerate": {}, "eval": {}, "exec": {}, "filter": {}, "float": {},
	"format": {}, "frozenset": {}, "getattr": {}, "globals": {}, "hasattr": {}, "hash": {}, "help": {},
	"hex": {}, "input": {}, "int": {}, "isinstance": {}, "issubclass": {}, "iter": {}, "len": {},
	"list": {}, "locals": {}, "map": {}, "max": {}, "memoryview": {}, "min": {}, "next": {}, "object": {},
	"oct": {}, "open": {}, "ord": {}, "pow": {}, "print": {}, "property": {}, "range": {}, "repr": {},
	"reversed": {}, "round": {}, "set": {}, "setattr": {}, "slice": {}, "sorted": {}, "staticmethod": {},
	"str": {}, "sum": {}, "super": {}, "tuple": {}, "type": {}, "vars": {}, "zip": {},
	"self": {}, "cls": {}, "args": {}, "kwargs": {}, "exc": {}, "err": {}, "idx": {}, "key": {},
	"val": {}, "row": {}, "item": {}, "text": {}, "data": {}, "result": {}, "name": {}, "main": {},
	"paid": {}, "value": {}, "rows": {}, "items": {}, "handle1": {}, "handle2": {}, "handle_one": {},
	"handle_two": {}, "prior_result": {}, "data_store": {}, "left": {}, "right": {}, "scenario": {},
	"left_handle": {}, "right_handle": {},
}

func looksLikeCapabilityName(name string) bool {
	if strings.HasPrefix(name, "__") || len(name) < 3 {
		return false
	}
	if _, ok := pythonVocabulary[name]; ok {
		return false
	}
	// Every builtin exception name ends in Error, Warning, or Exit and is part
	// of Python's vocabulary rather than the binding namespace.
	if strings.HasSuffix(name, "Error") || strings.HasSuffix(name, "Warning") || strings.HasSuffix(name, "Exception") {
		return false
	}
	return true
}

// legacyShapeCauses maps each pre-vocabulary failure_shape to the cause it is
// derivable to. A shape absent from this map, or a row whose detail does not
// match a rule, becomes `unclassified` rather than being deleted or guessed.
var legacyShapeCauses = map[string]string{
	"syntaxerror":         "kernel_syntax",
	"indentationerror":    "kernel_syntax",
	"modulenotfounderror": "kernel_runtime",
	"importerror":         "kernel_runtime",
	"nameerror":           "kernel_runtime",
	"typeerror":           "kernel_runtime",
	"valueerror":          "kernel_runtime",
	"keyerror":            "kernel_runtime",
	"attributeerror":      "kernel_runtime",
	"indexerror":          "kernel_runtime",
	"zerodivisionerror":   "kernel_runtime",
	"memoryerror":         "kernel_runtime",
	"remotedisconnected":  "bridge_transport",
	"deadline_exceeded":   "deadline_exceeded",
}

// detailCauseRules recover a precise cause from the stored failure detail. They
// run before the shape map because a `runtimeerror` row can be any of four
// unrelated causes, which is exactly why the exception-class taxonomy was
// replaced.
var detailCauseRules = []struct {
	substring string
	cause     string
}{
	{"is unreachable", "unreachable_scenario"},
	{"no proto field matches", "unknown_field"},
	{"has no determinable primary response field", "ambiguous_response"},
	{"requires an explicit grant", "refused_no_grant"},
	{"is not run-eligible", "refused_not_run_eligible"},
	{"inference_spend_exceeded", "inference_spend_exceeded"},
	{"delegated_run_spend_exceeded", "delegated_run_spend_exceeded"},
	{"does not resolve to a governed binding", "unresolved_name"},
	{"binding bridge unavailable", "bridge_transport"},
	{"remote status", "bridge_transport"},
	{"protocol error", "bridge_transport"},
}

// reclassifyLegacyFailureShapes rewrites corpus rows written before the closed
// cause vocabulary existed. Without it the corpus serves two taxonomies at once
// and `programs mine` returns both `runtimeerror` and `runtime error` beside
// real causes, which is the outcome the single-vocabulary decision rejects.
// Rows are never deleted: an underivable row is marked `unclassified` so the
// count stays honest.
func reclassifyLegacyFailureShapes(ctx context.Context, db SQLExecutor) error {
	known := map[string]struct{}{
		"unresolved_name": {}, "unknown_field": {}, "ambiguous_response": {}, "unreachable_scenario": {},
		"refused_no_grant": {}, "refused_not_run_eligible": {}, "inference_spend_exceeded": {},
		"delegated_run_spend_exceeded": {}, "deadline_exceeded": {}, "kernel_syntax": {},
		"kernel_runtime": {}, "bridge_transport": {}, "unclassified": {},
	}
	rows, err := db.QueryContext(ctx, `SELECT id, failure_shape, COALESCE(failure_detail, '') FROM programs WHERE status = 'failed' AND failure_shape != ''`)
	if err != nil {
		return fmt.Errorf("read legacy failure shapes: %w", err)
	}
	type update struct{ id, cause string }
	var pending []update
	for rows.Next() {
		var id, shape, detail string
		if err := rows.Scan(&id, &shape, &detail); err != nil {
			rows.Close()
			return fmt.Errorf("scan legacy failure shape: %w", err)
		}
		if _, ok := known[shape]; ok {
			continue
		}
		cause := ""
		for _, rule := range detailCauseRules {
			if strings.Contains(detail, rule.substring) {
				cause = rule.cause
				break
			}
		}
		if cause == "" {
			cause = legacyShapeCauses[strings.ToLower(strings.TrimSpace(shape))]
		}
		if cause == "" {
			cause = "unclassified"
		}
		pending = append(pending, update{id: id, cause: cause})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("read legacy failure shapes: %w", err)
	}
	rows.Close()
	for _, item := range pending {
		if _, err := db.ExecContext(ctx, `UPDATE programs SET failure_shape = ?, failure_cause = ? WHERE id = ?`, item.cause, item.cause, item.id); err != nil {
			return fmt.Errorf("reclassify program %s: %w", item.id, err)
		}
	}
	return nil
}
