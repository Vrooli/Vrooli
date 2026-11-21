package coverage

import "strings"

// sqliteSidecarSuffixes are the files SQLite keeps beside a database while it is
// open: the write-ahead log, the shared-memory index and the rollback journal.
// None of them is data of its own. They are only meaningful with their database,
// and the database's own backup (a SQLite snapshot) already takes them into
// account, so offering one as a separate item to protect is a candidate-list
// defect, not a choice for the operator.
var sqliteSidecarSuffixes = []string{"-wal", "-shm", "-journal"}

// FoldSQLiteSidecars removes every suggestion that is a SQLite sidecar of a
// database the coverage picture already knows about (another suggestion or a
// registered target) and lists the sidecar suffixes on that database's
// suggestion. A sidecar whose database is unknown stays, because dropping it
// would hide the only item that leads to the data.
func FoldSQLiteSidecars(in []Suggestion, registered []CatalogTarget) []Suggestion {
	known := make(map[string]struct{}, len(in)+len(registered))
	for _, s := range in {
		if s.Locator != "" {
			known[s.Locator] = struct{}{}
		}
	}
	for _, t := range registered {
		if t.Locator != "" {
			known[t.Locator] = struct{}{}
		}
	}
	folded := map[string][]string{}
	kept := make([]Suggestion, 0, len(in))
	for _, s := range in {
		if base, suffix, ok := sqliteSidecarOf(s.Locator); ok {
			if _, has := known[base]; has {
				folded[base] = append(folded[base], suffix)
				continue
			}
		}
		kept = append(kept, s)
	}
	for i := range kept {
		if suffixes := folded[kept[i].Locator]; len(suffixes) > 0 {
			kept[i].Sidecars = append([]string(nil), suffixes...)
		}
	}
	return kept
}

func sqliteSidecarOf(locator string) (base, suffix string, ok bool) {
	for _, suffix := range sqliteSidecarSuffixes {
		if strings.HasSuffix(locator, suffix) && len(locator) > len(suffix) {
			return strings.TrimSuffix(locator, suffix), suffix, true
		}
	}
	return "", "", false
}
