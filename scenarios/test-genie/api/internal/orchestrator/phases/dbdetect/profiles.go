package dbdetect

import "strings"

// Matchers ---------------------------------------------------------------

// ManifestType matches a manifest observation whose Value equals the given
// resource type (case-insensitive).
func ManifestType(t string) Matcher {
	want := strings.ToLower(strings.TrimSpace(t))
	return func(o Observation) bool {
		return o.Collector == "manifest" && strings.EqualFold(o.Value, want)
	}
}

// ImportPrefixes matches a godeps observation whose Value starts with any of
// the given import-path prefixes.
func ImportPrefixes(prefixes ...string) Matcher {
	return func(o Observation) bool {
		if o.Collector != "godeps" {
			return false
		}
		for _, p := range prefixes {
			if strings.HasPrefix(o.Value, p) {
				return true
			}
		}
		return false
	}
}

// Tokens matches a source observation whose Value is exactly one of the given
// tokens. ProfileSource.Tokens must list the same set so the source collector
// knows what to scan for.
func Tokens(tokens ...string) Matcher {
	want := map[string]bool{}
	for _, t := range tokens {
		want[t] = true
	}
	return func(o Observation) bool {
		return o.Collector == "source" && want[o.Value]
	}
}

// SourceTokens returns the union of ProfileSource.Tokens across all profiles
// for source-collector sources. NewResolver passes the result to
// SetSourceTokens so the collector only scans for tokens the active profiles
// care about.
func SourceTokens(profiles []Profile) []string {
	var out []string
	for _, p := range profiles {
		for _, s := range p.Sources {
			if s.Collector != "source" {
				continue
			}
			out = append(out, s.Tokens...)
		}
	}
	return out
}

// Default profiles ------------------------------------------------------

// DefaultProfiles returns the canonical detection profiles for postgres,
// redis, and sqlite. The profile table is the single source of truth for
// what evidence counts for each DB.
func DefaultProfiles() []Profile {
	postgresTokens := []string{"POSTGRES_URL", "DriverPostgres"}
	// Source tokens follow the code, not history. Scenarios resolve SQLite
	// through the one owned seam in api-core/storage now, so the tokens that
	// evidence a SQLite dependency are the seam's entry points. The former
	// tokens — the generic SQLITE_PATH / SQLITE_DB pair and the per-scenario
	// sqliteDSN( helper — no longer appear in a migrated scenario, and leaving
	// them here would report a SQLite user as having no evidence.
	sqliteTokens := []string{
		`sql.Open("sqlite"`,
		`sqlx.Connect("sqlite"`,
		"storage.SQLiteDSN(",
		"storage.SQLiteDSNAt(",
		"storage.SQLitePath(",
		"DriverSQLite",
		"BAS_SQLITE_PATH",
	}
	return []Profile{
		{DB: "postgres", Sources: []ProfileSource{
			{Collector: "manifest", Match: ManifestType("postgres"), Priority: PriorityHigh, Label: "manifest:resource[type=postgres]"},
			{Collector: "godeps", Match: ImportPrefixes("github.com/lib/pq", "github.com/jackc/pgx"), Priority: PriorityMedium, Label: "godeps:postgres-driver"},
			{Collector: "source", Match: Tokens(postgresTokens...), Priority: PriorityLow, Label: "source:postgres-tokens", Tokens: postgresTokens},
		}},
		{DB: "redis", Sources: []ProfileSource{
			{Collector: "manifest", Match: ManifestType("redis"), Priority: PriorityHigh, Label: "manifest:resource[type=redis]"},
			{Collector: "godeps", Match: ImportPrefixes("github.com/redis/go-redis", "github.com/go-redis/redis"), Priority: PriorityMedium, Label: "godeps:redis-driver"},
		}},
		{DB: "sqlite", Sources: []ProfileSource{
			{Collector: "godeps", Match: ImportPrefixes("modernc.org/sqlite", "github.com/mattn/go-sqlite3"), Priority: PriorityHigh, Label: "godeps:sqlite-driver"},
			{Collector: "source", Match: Tokens(sqliteTokens...), Priority: PriorityMedium, Label: "source:sqlite-tokens", Tokens: sqliteTokens},
		}},
	}
}
