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

// EnsureCompatibility adds fields introduced after the initial programs
// table was created. Existing records default to unverified attribution.
func EnsureCompatibility(ctx context.Context, db SQLExecutor) error {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(programs)")
	if err != nil {
		return fmt.Errorf("inspect programs schema: %w", err)
	}
	defer rows.Close()
	columns := make(map[string]bool)
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return fmt.Errorf("scan programs schema: %w", err)
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate programs schema: %w", err)
	}
	for _, name := range []string{"caller_identity_verified", "source_receipt_format", "stdout_receipt_format", "failure_detail_receipt_format"} {
		if !columns[name] {
			if _, err := db.ExecContext(ctx, "ALTER TABLE programs ADD COLUMN "+name+" INTEGER NOT NULL DEFAULT 0"); err != nil {
				return fmt.Errorf("add %s to programs: %w", name, err)
			}
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
