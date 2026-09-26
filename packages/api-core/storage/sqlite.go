package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// SQLite bootstrap: the single seam through which every Go scenario opens its
// SQLite database.
//
// # Why this exists
//
// Before this package owned the concern, 60+ scenarios each carried a private
// copy of the same two functions — one resolving a path, one wrapping it in a
// pragma string. Two things went wrong with that arrangement, and both were
// structural rather than accidental.
//
// First, the copies drifted. Four distinct pragma implementations existed at
// once, including one still using the pre-"_pragma" DSN grammar that modernc
// silently ignores, so that scenario had never actually run in WAL mode. A
// tuning change had to be applied 60+ times or not at all.
//
// Second — and far worse — every copy resolved its path from a GENERIC
// environment variable (SQLITE_PATH, falling back to SQLITE_DB) at HIGHER
// precedence than its own identity. A generic name in an inherited environment
// is a cross-scenario write. The supervisor scenario declared SQLITE_PATH in its
// own manifest and restarted sick scenarios by exec'ing the CLI; each restarted
// child inherited the supervisor's environment and opened the SUPERVISOR's
// database instead of its own. Twelve scenarios were observed sharing one 9.35 GB
// file with a single writer lock. No unit test could see it, because the defect
// lives in process environment inheritance rather than in any code path a test
// exercises.
//
// # The rule
//
// A scenario NAMES ITSELF; it does not read a path from the environment. The
// resolved path is a pure function of the scenario's own compile-time slug and
// its variant-aware namespace, so an inherited environment cannot redirect it —
// there is no per-scenario path input left to inherit.
//
//	dsn, err := storage.SQLiteDSN(storage.SQLiteConfig{Scenario: "storage-manager"})
//	// -> file:<data-root>/vrooli/storage-manager/storage-manager.db?_pragma=...
//
// Shadow isolation is preserved and is the reason resolution is not simply a
// hardcoded path: SQLiteConfig.Scenario is the compile-time FALLBACK, and the
// effective scope comes from ScenarioNamespace, which prefers the
// lifecycle-injected VROOLI_STORAGE_NAMESPACE. That variable is scenario-AGNOSTIC
// by construction ("<scenario>" or "<scenario>_<variant>"), so a child that
// inherits it inherits a value that still resolves to the child's own identity.
// That is precisely the property the generic path variables lacked.
//
// # Tests and fixtures
//
// A test that needs an explicit path calls SQLiteDSNAt, which takes the path as
// a function ARGUMENT rather than as ambient process state. An override that
// exists in the environment is an override that gets inherited; an override
// passed as an argument cannot leak into a child process.

// Canonical SQLite tuning defaults. These are the values the fleet converged on
// and are applied whenever the corresponding SQLiteTuning field is left zero.
const (
	// DefaultSQLiteBusyTimeout bounds how long a writer waits for the
	// single-writer lock before returning SQLITE_BUSY.
	DefaultSQLiteBusyTimeout = 10 * time.Second
	// DefaultSQLiteCacheSizeKiB is the page-cache budget. SQLite reads a
	// NEGATIVE cache_size as KiB rather than as a page count; this package
	// applies the sign, so callers state a positive size.
	DefaultSQLiteCacheSizeKiB = 2000
	// DefaultSQLiteSynchronous is the durability mode. NORMAL is the correct
	// pairing with WAL: it survives application crashes and trades only
	// power-loss durability of the most recent commits for a large write-cost
	// reduction.
	DefaultSQLiteSynchronous = "NORMAL"
	// defaultSQLiteDirPerm is the permission applied to a created parent dir.
	defaultSQLiteDirPerm os.FileMode = 0o755
	// sqliteDSNScheme prefixes a file-backed SQLite DSN.
	sqliteDSNScheme = "file:"
	// envScenarioDataDir carries the lifecycle-assigned data directory for the
	// instance being launched. It is read only behind the identity guard in
	// lifecycleDataDir; see that function for why.
	envScenarioDataDir = "SCENARIO_DATA_DIR"
	// envScenarioName is the lifecycle's other name for the running scenario.
	// It is injected alongside VROOLI_SCENARIO from the same slug.
	envScenarioName = "SCENARIO_NAME"
)

// SQLiteTuning carries the optional, per-scenario deviations from the canonical
// pragma set. Every field is optional; a zero value yields the canonical DSN,
// which is what almost every scenario wants.
//
// Fields are typed rather than free-form pragma strings on purpose: a string
// escape hatch here would recreate, one abstraction level up, exactly the
// divergence this package was built to eliminate.
type SQLiteTuning struct {
	// BusyTimeout overrides DefaultSQLiteBusyTimeout. Sub-millisecond values
	// are rejected, because the pragma's unit is milliseconds and a value that
	// truncates to 0 disables waiting entirely — silently turning contention
	// into immediate SQLITE_BUSY errors.
	BusyTimeout time.Duration
	// CacheSizeKiB overrides DefaultSQLiteCacheSizeKiB. State it POSITIVE; the
	// negative sign SQLite uses to mean "KiB, not pages" is applied here.
	CacheSizeKiB int
	// Synchronous overrides DefaultSQLiteSynchronous ("OFF", "NORMAL", "FULL",
	// or "EXTRA").
	Synchronous string
	// PageSizeBytes emits a page_size pragma. It takes effect only on a
	// database that has not been created yet, or after a VACUUM. Zero omits it.
	PageSizeBytes int
	// MMapSizeBytes emits an mmap_size pragma, letting SQLite read through a
	// memory map instead of the page cache. It suits large read-mostly
	// databases. Zero omits it.
	MMapSizeBytes int64
	// TimeFormat sets the driver's _time_format parameter. This is a modernc
	// driver parameter rather than a SQLite pragma. Empty omits it.
	TimeFormat string
	// TxLock sets the driver's _txlock parameter ("deferred", "immediate", or
	// "exclusive"). "immediate" makes BeginTx take the reserved lock up front,
	// which suits a read-then-write transaction that would otherwise upgrade
	// mid-flight and risk SQLITE_BUSY under contention. Empty omits it, leaving
	// the driver default (deferred).
	TxLock string
}

// SQLiteConfig describes which database a scenario is opening.
//
// Scenario is the only required field. The rest exist so a scenario with a
// genuine reason to differ can state that reason in typed form instead of
// hand-rolling a DSN.
type SQLiteConfig struct {
	// Scenario is the compile-time scenario slug — the scenario naming itself.
	// It is the FALLBACK identity: the lifecycle-injected
	// VROOLI_STORAGE_NAMESPACE wins when present, so a shadow variant scopes to
	// "<scenario>_<variant>" and never shares live's file. Required.
	Scenario string
	// Filename is the database file name within the class root. It defaults to
	// "<Scenario>.db". It must be a bare file name; a path separator is
	// rejected so a file name can never escape its scenario's directory.
	Filename string
	// Class selects the storage class root. It defaults to ClassData, which is
	// correct for a primary application database. A rebuildable index or cache
	// may choose ClassCache so retention treats it as regenerable.
	Class Class
	// AppID is forwarded to the resolver and defaults to "vrooli".
	AppID string
	// Profile is forwarded to the resolver and defaults to ProfileAuto.
	Profile Profile
	// RootOverride forces the class roots under one directory. It is a test and
	// container seam; production callers leave it empty.
	RootOverride string
	// EnvGet reads environment variables during namespace and profile
	// resolution. It defaults to os.Getenv and exists so a test can drive
	// resolution without mutating real process state.
	EnvGet func(key string) string
	// UserHomeDir resolves the operator home from which the runtime home
	// derives. It defaults to os.UserHomeDir.
	UserHomeDir func() (string, error)
	// DirPerm is the permission applied when the parent directory is created.
	// It defaults to 0o755.
	DirPerm os.FileMode
	// Tuning carries optional deviations from the canonical pragma set.
	Tuning SQLiteTuning
}

// SQLitePath resolves the absolute, variant-aware path of a scenario's SQLite
// database WITHOUT creating anything or building a DSN.
//
// Use it when the path itself is the answer — a storage census, a backup
// target, a diagnostic that reports where a scenario's data lives. Use
// SQLiteDSN when the goal is to open the database.
func SQLitePath(cfg SQLiteConfig) (string, error) {
	scenario := strings.TrimSpace(cfg.Scenario)
	if scenario == "" {
		return "", &Error{
			Kind:    ErrInvalidInput,
			Message: "SQLiteConfig.Scenario is required: a scenario must name itself rather than read its database path from the environment",
		}
	}

	filename, err := sqliteFilename(scenario, cfg.Filename)
	if err != nil {
		return "", err
	}

	class := cfg.Class
	if class == "" {
		class = ClassData
	}

	resolver, err := NewResolver(ResolverConfig{
		AppID:       cfg.AppID,
		Profile:     cfg.Profile,
		EnvGet:      cfg.EnvGet,
		UserHomeDir: cfg.UserHomeDir,
	})
	if err != nil {
		return "", fmt.Errorf("create storage resolver for %s: %w", scenario, err)
	}

	scenarioID, err := ownNamespace(scenario, cfg.EnvGet)
	if err != nil {
		return "", fmt.Errorf("resolve %s storage namespace: %w", scenario, err)
	}

	// A live instance keeps the data directory the lifecycle assigned it. This
	// is what makes consolidation a pure refactor rather than a fleet-wide data
	// migration: every existing database stays exactly where it already is.
	//
	// An explicit storage-root override outranks it. That override is how a
	// test harness leases a scenario an isolated storage tree, and honouring
	// the assigned data directory over it would put a scenario under test back
	// on its production database.
	if class == ClassData && cfg.RootOverride == "" && scenarioID.IsLive() && !hasRootOverrideEnv(cfg.EnvGet) {
		if dir := lifecycleDataDir(scenario, cfg.EnvGet); dir != "" {
			return filepath.Join(dir, filename), nil
		}
	}

	path, err := resolver.Path(
		Options{ScenarioID: scenarioID.Root(), RootOverride: cfg.RootOverride},
		class,
		filename,
	)
	if err != nil {
		return "", fmt.Errorf("resolve %s database path: %w", scenario, err)
	}
	return path, nil
}

// hasRootOverrideEnv reports whether the environment redirects the storage
// class roots wholesale. These variables are scenario-AGNOSTIC — they name a
// tree, not one scenario's file — so every scenario beneath such a root still
// resolves to its own separate database, and inheriting one is safe.
func hasRootOverrideEnv(envGet func(string) string) bool {
	if envGet == nil {
		envGet = os.Getenv
	}
	for _, key := range []string{envStorageRoot, envDataRoot} {
		if strings.TrimSpace(envGet(key)) != "" {
			return true
		}
	}
	return false
}

// ownNamespace resolves the variant-aware namespace, but only accepts one that
// demonstrably belongs to THIS scenario.
//
// The variant-aware namespace is what keeps a shadow's database separate from
// live's, and the lifecycle supplies it as VROOLI_STORAGE_NAMESPACE
// ("<scenario>" or "<scenario>_<variant>"). That value is safe to inherit only
// in the sense that it cannot alias two scenarios onto ONE file — but an
// inherited one still names the wrong scenario, and would place this
// scenario's database inside the supervisor's directory.
//
// So the resolved root is accepted only when it is this scenario's own: either
// the bare slug, or the slug followed by a variant suffix. Anything else means
// the environment belongs to a different process, and the only identity
// actually evidenced is this scenario's own live identity — which is what it
// falls back to. Startup is deliberately NOT failed here: a supervisor
// restarting a sick scenario is a routine event, and refusing to start would
// turn a misplaced-directory problem into an outage.
func ownNamespace(scenario string, envGet func(string) string) (Namespace, error) {
	ns, err := ResolveNamespace(NamespaceConfig{EnvGet: envGet, FallbackScenario: scenario})
	if err != nil {
		// The one hard error ResolveNamespace raises is a non-live variant with
		// no injected root. If that environment does not belong to this
		// scenario either, it is another artefact of inheritance rather than a
		// statement about this process, so fall back to this scenario's own
		// live identity.
		// Fall back ONLY when the environment explicitly names a DIFFERENT
		// scenario, which is positive evidence of inheritance. With no name at
		// all there is no such evidence, and a non-live variant missing its
		// root stays a hard error — that is the guard which stops a shadow from
		// writing into live.
		if environmentNamesOtherScenario(scenario, envGet) {
			if own, ownErr := ResolveNamespace(NamespaceConfig{Root: scenario, Variant: defaultVariant}); ownErr == nil {
				return own, nil
			}
		}
		return Namespace{}, err
	}
	if isOwnNamespaceRoot(scenario, ns.Root()) {
		return ns, nil
	}
	return ResolveNamespace(NamespaceConfig{Root: scenario, Variant: defaultVariant})
}

// isOwnNamespaceRoot reports whether root is "<scenario>" or
// "<scenario>_<variant>". The separator check matters: without it, scenario
// "web" would claim the root of "web-console".
func isOwnNamespaceRoot(scenario, root string) bool {
	if root == scenario {
		return true
	}
	return strings.HasPrefix(root, scenario+"_")
}

// environmentNamesOtherScenario reports whether the process identity carried in
// the environment names a scenario that is NOT this one — positive evidence
// that the environment was inherited rather than injected for this process.
func environmentNamesOtherScenario(scenario string, envGet func(string) string) bool {
	if envGet == nil {
		envGet = os.Getenv
	}
	for _, key := range []string{envScenarioName, EnvScenario} {
		if v := strings.TrimSpace(envGet(key)); v != "" {
			return v != scenario
		}
	}
	return false
}

// lifecycleDataDir returns the lifecycle-assigned data directory for scenario,
// or "" when the environment does not credibly belong to this scenario.
//
// SCENARIO_DATA_DIR names a directory rather than a file, so on its own it
// cannot collide two scenarios on one database the way SQLITE_PATH did. It is
// still per-scenario state, and a supervisor that exec's the CLI leaks its own
// value into the child — so it is honoured ONLY when the process identity
// agrees that this is the scenario the lifecycle launched.
//
// That guard is reliable because the two facts come from ONE source: the
// lifecycle sets SCENARIO_NAME, VROOLI_SCENARIO, and SCENARIO_DATA_DIR together
// from the same slug (internal/ports). So the two cases separate cleanly:
//
//   - Launched by the lifecycle: the injected name equals this scenario's own
//     slug, the directory is genuinely this scenario's, and the guard passes.
//   - Inherited from a supervisor: the injected name is the SUPERVISOR's slug,
//     it does not equal this scenario's own, and the guard rejects the
//     directory. Resolution falls through to the class-rooted path, so the
//     scenario gets its own private database rather than a sibling's.
//
// Non-live variants never reach here. A shadow resolves through the class roots
// under its "<scenario>_<variant>" namespace, because the data directory the
// lifecycle assigns is not variant-scoped and would strand a shadow's writes in
// live's directory — the one outcome variant isolation exists to prevent.
func lifecycleDataDir(scenario string, envGet func(string) string) string {
	if envGet == nil {
		envGet = os.Getenv
	}
	running := strings.TrimSpace(envGet(envScenarioName))
	if running == "" {
		running = strings.TrimSpace(envGet(EnvScenario))
	}
	if running == "" || running != scenario {
		return ""
	}
	dir := strings.TrimSpace(envGet(envScenarioDataDir))
	if dir == "" || !filepath.IsAbs(dir) {
		return ""
	}
	return dir
}

// SQLiteDSN resolves a scenario's database path, creates its parent directory,
// and returns the DSN to hand to database.Open or sql.Open.
//
// This is the call almost every scenario makes, and for the common case it
// takes exactly one field:
//
//	dsn, err := storage.SQLiteDSN(storage.SQLiteConfig{Scenario: "notification-hub"})
func SQLiteDSN(cfg SQLiteConfig) (string, error) {
	path, err := SQLitePath(cfg)
	if err != nil {
		return "", err
	}
	return sqliteDSNAt(path, cfg.Tuning, cfg.DirPerm)
}

// SQLiteDSNAt builds a DSN for an EXPLICIT database path, creating the parent
// directory if it does not exist.
//
// The path arrives as a function argument, which is the whole point: tests,
// migrations, and one-off diagnostics need to name a file directly, and passing
// it as an argument keeps that need from becoming ambient process state that a
// child process could inherit. Production scenario code calls SQLiteDSN instead
// and never states a path at all.
func SQLiteDSNAt(path string, tuning SQLiteTuning) (string, error) {
	return sqliteDSNAt(path, tuning, 0)
}

// SQLiteReadOnlyDSNAt builds a DSN that opens an existing database for
// INSPECTION and cannot modify it.
//
// It is a separate constructor rather than a flag on SQLiteTuning because the
// intent is different in kind: this is for reading a database you do not own —
// a storage census, a backup audit, a diagnostic — and the guarantees are what
// make that safe. It creates nothing: no parent directory is made, and a
// missing file is an error rather than a new empty database. Tuning is not
// accepted, because none of it applies to a connection that never writes.
//
// # Why the locking mode depends on the WAL
//
// Every mode here refuses writes (mode=ro plus the query_only pragma). What
// varies is how the reader coordinates with a live writer, and picking one
// mode for both cases gets one of them wrong:
//
//   - No "-wal" file beside the database: immutable=1. The reader takes no
//     lock and creates no journal, so it cannot block or be blocked. This is
//     the cheapest correct read of a quiescent database.
//
//   - A "-wal" file exists: nolock=1 instead. immutable=1 promises SQLite the
//     file cannot change, which lets it skip the WAL entirely — so a reader
//     would silently miss every committed transaction still in the WAL and
//     report a stale view as authoritative. That is worse than failing.
//
// The check is a stat of the sidecar file rather than a query, because there is
// no connection yet to ask.
func SQLiteReadOnlyDSNAt(path string) (string, error) {
	trimmed, err := validateSQLitePath(path)
	if err != nil {
		return "", err
	}
	const readOnlyPragmas = "&_pragma=query_only(1)&_pragma=busy_timeout(10000)&_pragma=temp_store(MEMORY)"
	if _, statErr := os.Stat(trimmed + "-wal"); statErr == nil {
		// A live database with pending WAL content.
		return sqliteDSNScheme + trimmed + "?mode=ro&nolock=1" + readOnlyPragmas, nil
	}
	return sqliteDSNScheme + trimmed + "?mode=ro&immutable=1" + readOnlyPragmas, nil
}

// validateSQLitePath checks that a caller-supplied path can be used in a DSN.
func validateSQLitePath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", &Error{Kind: ErrInvalidInput, Message: "sqlite database path must not be empty"}
	}
	// Reject a DSN masquerading as a path. Accepting it would mean silently
	// discarding the caller's tuning, which is the kind of quiet inconsistency
	// this package exists to remove.
	if strings.HasPrefix(trimmed, sqliteDSNScheme) {
		return "", &Error{
			Kind:    ErrInvalidInput,
			Message: "sqlite path must be a filesystem path, not an assembled DSN; pass the path and let this package apply the pragmas",
			Details: trimmed,
		}
	}
	// "?" begins the DSN query string and "#" a fragment, so a path containing
	// either would silently truncate — the connection would open a DIFFERENT
	// file, or none, rather than fail. Refuse instead of guessing.
	if strings.ContainsAny(trimmed, "?#") {
		return "", &Error{
			Kind:    ErrInvalidInput,
			Message: "sqlite path must not contain '?' or '#': both terminate the path in a DSN, so the connection would silently address a different file",
			Details: trimmed,
		}
	}
	return trimmed, nil
}

func sqliteDSNAt(path string, tuning SQLiteTuning, dirPerm os.FileMode) (string, error) {
	trimmed, err := validateSQLitePath(path)
	if err != nil {
		return "", err
	}

	params, err := sqliteDSNParams(tuning)
	if err != nil {
		return "", err
	}

	if dirPerm == 0 {
		dirPerm = defaultSQLiteDirPerm
	}
	if dir := filepath.Dir(trimmed); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, dirPerm); err != nil {
			return "", &Error{Kind: ErrIO, Message: "prepare sqlite directory", Details: dir, Err: err}
		}
	}

	return sqliteDSNScheme + trimmed + "?" + params, nil
}

// sqliteDSNParams renders the DSN query string.
//
// The emission ORDER is fixed and deliberate. It reproduces, byte for byte, the
// string the fleet had converged on before consolidation, so migrating a
// scenario onto this package changes no bytes in the common case and a
// difference in a connection string is therefore real signal rather than noise.
func sqliteDSNParams(tuning SQLiteTuning) (string, error) {
	busy := tuning.BusyTimeout
	if busy == 0 {
		busy = DefaultSQLiteBusyTimeout
	}
	if busy < 0 {
		return "", &Error{Kind: ErrInvalidInput, Message: "sqlite busy timeout must not be negative", Details: busy.String()}
	}
	busyMillis := busy.Milliseconds()
	if busyMillis == 0 {
		// A sub-millisecond timeout truncates to 0, which SQLite reads as
		// "never wait". Refuse rather than silently disable lock waiting.
		return "", &Error{Kind: ErrInvalidInput, Message: "sqlite busy timeout truncates to 0ms; state at least 1ms", Details: busy.String()}
	}

	cache := tuning.CacheSizeKiB
	if cache == 0 {
		cache = DefaultSQLiteCacheSizeKiB
	}
	if cache < 0 {
		return "", &Error{Kind: ErrInvalidInput, Message: "sqlite cache size must be stated as a positive KiB value", Details: strconv.Itoa(tuning.CacheSizeKiB)}
	}

	sync := strings.TrimSpace(tuning.Synchronous)
	if sync == "" {
		sync = DefaultSQLiteSynchronous
	}
	switch strings.ToUpper(sync) {
	case "OFF", "NORMAL", "FULL", "EXTRA":
		sync = strings.ToUpper(sync)
	default:
		return "", &Error{Kind: ErrInvalidInput, Message: "unknown sqlite synchronous mode", Details: tuning.Synchronous}
	}

	params := make([]string, 0, 9)
	pragma := func(body string) { params = append(params, "_pragma="+body) }

	pragma("foreign_keys(ON)")
	pragma("journal_mode(WAL)")
	pragma("busy_timeout(" + strconv.FormatInt(busyMillis, 10) + ")")
	// SQLite reads a negative cache_size as KiB rather than as a page count.
	pragma("cache_size(-" + strconv.Itoa(cache) + ")")
	if tuning.PageSizeBytes != 0 {
		if tuning.PageSizeBytes < 0 {
			return "", &Error{Kind: ErrInvalidInput, Message: "sqlite page size must be positive", Details: strconv.Itoa(tuning.PageSizeBytes)}
		}
		pragma("page_size(" + strconv.Itoa(tuning.PageSizeBytes) + ")")
	}
	pragma("synchronous(" + sync + ")")
	pragma("temp_store(MEMORY)")
	if tuning.MMapSizeBytes != 0 {
		if tuning.MMapSizeBytes < 0 {
			return "", &Error{Kind: ErrInvalidInput, Message: "sqlite mmap size must not be negative", Details: strconv.FormatInt(tuning.MMapSizeBytes, 10)}
		}
		pragma("mmap_size(" + strconv.FormatInt(tuning.MMapSizeBytes, 10) + ")")
	}
	// Driver parameters, not pragmas. They are emitted last so the pragma run
	// stays contiguous and byte-comparable with the pre-consolidation string.
	if tf := strings.TrimSpace(tuning.TimeFormat); tf != "" {
		params = append(params, "_time_format="+tf)
	}
	if lock := strings.TrimSpace(tuning.TxLock); lock != "" {
		switch strings.ToLower(lock) {
		case "deferred", "immediate", "exclusive":
			params = append(params, "_txlock="+strings.ToLower(lock))
		default:
			return "", &Error{Kind: ErrInvalidInput, Message: "unknown sqlite txlock mode", Details: tuning.TxLock}
		}
	}

	return strings.Join(params, "&"), nil
}

// sqliteFilename validates the database file name and applies the
// "<scenario>.db" default.
func sqliteFilename(scenario, requested string) (string, error) {
	name := strings.TrimSpace(requested)
	if name == "" {
		return scenario + ".db", nil
	}
	// A bare file name only. A separator or a parent reference would let a
	// scenario address a sibling's directory, which is the class of defect this
	// package exists to close.
	if strings.ContainsRune(name, '/') || strings.ContainsRune(name, '\\') || name == "." || name == ".." {
		return "", &Error{
			Kind:    ErrInvalidInput,
			Message: "sqlite filename must be a bare file name within the scenario's storage class",
			Details: requested,
		}
	}
	return name, nil
}
