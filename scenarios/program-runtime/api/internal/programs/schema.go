package programs

import (
	"strings"

	_ "embed"
)

//go:embed schema.sql
var schema string

// Schema returns the programs domain schema for the central database.
func Schema() string { return schema }

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
