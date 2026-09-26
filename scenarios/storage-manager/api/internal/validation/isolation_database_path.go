package validation

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// isoDatabasePath emits DATABASE_PATH_FROM_ENVIRONMENT when a scenario reads its
// database location out of a GENERIC environment variable, or hand-rolls a
// SQLite DSN instead of resolving both through api-core/storage.
//
// # Why this analyzer exists
//
// It encodes a defect that was live in production and that no scenario's own
// test suite could have caught. Sixty-plus scenarios each resolved their SQLite
// path by reading SQLITE_PATH, falling back to SQLITE_DB, ABOVE their own
// identity. vrooli-autoheal declared SQLITE_PATH in its manifest and restarted
// sick scenarios by exec'ing the CLI; every restarted child inherited the value
// and opened vrooli-autoheal's database instead of its own. Twelve scenarios
// were observed sharing one 9.35 GB file behind a single writer lock, including
// Test Genie — whose run ledger recorded 146 runs into it — and plan-manager.
//
// Every layer behaved reasonably in isolation. That is exactly why it needs an
// analyzer rather than a code review convention: the defect is only visible
// when you look at two scenarios at once, and it reappears the moment someone
// adds a "convenient" override.
//
// # What counts as a violation
//
//  1. Reading a generic database-path variable. The name is the problem, not
//     the read: a variable that does not identify a scenario cannot be scoped
//     to one, so any process that exports it captures every process it starts.
//  2. Assembling a SQLite DSN by hand. A private pragma string drifts from the
//     fleet's, and the drift is silent — one scenario spent its whole life
//     using the pre-"_pragma" grammar the driver ignores, so it never ran in
//     WAL mode despite a comment saying it did.
//
// # What does not count
//
// A SCENARIO-SCOPED variable (BAS_SQLITE_PATH, PLAYBOOKS_SQLITE_DSN) is not
// flagged: its name carries the owner, so it cannot silently capture a sibling.
// Neither are the storage-root levers (VROOLI_STORAGE_ROOT, VROOLI_DATA_ROOT)
// or VROOLI_STORAGE_NAMESPACE — those redirect a whole tree rather than name
// one file, so every scenario beneath them still resolves to its own path,
// which is precisely what makes them safe to inherit and useful for test
// isolation.
type isoDatabasePath struct {
	// fileHandlesFileDSN records whether the file under analysis treats a
	// value as a "file:" SQLite DSN. It gates the conditional variables above
	// and is set per file by analyzeSource.
	fileHandlesFileDSN bool
}

func init() { register(&isoDatabasePath{}) }

func (isoDatabasePath) Name() string { return "isolation.database-path-from-environment" }

func (isoDatabasePath) Applies(ac AnalyzerContext) bool {
	return ac.IsGo() && ac.HasEngine(EngineSQLite)
}

// isoGenericDatabasePathEnvVars are the variable names that name a database
// without naming its owner. Each one is a channel through which one scenario's
// environment redirects another scenario's storage.
var isoGenericDatabasePathEnvVars = map[string]string{
	"SQLITE_PATH":          "the generic SQLite path variable that caused the cross-scenario database hijack",
	"SQLITE_DB":            "the generic SQLite path variable that caused the cross-scenario database hijack",
	"SQLITE_DATABASE_PATH": "a generic SQLite directory variable",
}

// isoSQLiteDSNEnvVars are variables that are only a hijack risk when the file
// treats them as a SQLite location.
//
// DATABASE_URL is the conventional PostgreSQL connection string and reading it
// for Postgres is correct: Postgres isolates per variant through the
// lifecycle-injected POSTGRES_DB, so the URL is not a cross-scenario path. It
// becomes this defect only when a scenario accepts a "file:" DSN through it,
// which is how three scenarios took a SQLite path from an inherited
// environment. Flagging every Postgres read would make this analyzer noise, and
// a noisy analyzer gets switched off — so the finding is gated on the file
// actually handling a "file:" DSN.
var isoSQLiteDSNEnvVars = map[string]string{
	"DATABASE_URL": "a generic database URL that this file accepts as a \"file:\" SQLite DSN, " +
		"so an inherited value redirects this scenario's database",
}

func (a isoDatabasePath) Analyze(_ context.Context, ac AnalyzerContext) ([]Finding, error) {
	var findings []Finding
	for _, gf := range CollectGoFiles(ac) {
		findings = append(findings, a.analyzeSource(ReadFile(gf.AbsPath), gf.RelPath)...)
	}
	return findings, nil
}

// analyzeSource is the pure detection core, shared by Analyze and the tests.
func (a isoDatabasePath) analyzeSource(source, relPath string) []Finding {
	// api-core/storage is the package that legitimately holds this knowledge,
	// and its own tests must be free to assert on the rejected variables.
	if hygieneIsExemptPath(relPath) || hygieneIsAPICorePath(relPath) {
		return nil
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, relPath, source, 0)
	if err != nil {
		return nil
	}
	a.fileHandlesFileDSN = strings.Contains(source, `"file:`)

	var findings []Finding
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if f := a.checkEnvRead(call, fset, relPath); f != nil {
			findings = append(findings, *f)
		}
		if f := a.checkHandRolledDSN(call, fset, relPath); f != nil {
			findings = append(findings, *f)
		}
		return true
	})

	// A raw DSN can also be built by concatenation rather than by a call, so
	// look for the connection grammar in any string literal in the file.
	findings = append(findings, a.checkPragmaLiterals(file, fset, relPath)...)

	// A Sprintf DSN matches both detectors — the call and its format literal —
	// so report each distinct site once.
	return isoDedupeFindings(findings)
}

// isoDedupeFindings collapses findings that share a code and a location,
// preserving order.
func isoDedupeFindings(in []Finding) []Finding {
	if len(in) < 2 {
		return in
	}
	seen := make(map[string]bool, len(in))
	out := in[:0]
	for _, f := range in {
		key := f.Code + "\x00" + f.Location
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, f)
	}
	return out
}

// checkEnvRead flags os.Getenv / os.LookupEnv of a generic database-path name.
func (a isoDatabasePath) checkEnvRead(call *ast.CallExpr, fset *token.FileSet, relPath string) *Finding {
	if !hygieneIsSelectorCall(call, "os", "Getenv") && !hygieneIsSelectorCall(call, "os", "LookupEnv") {
		return nil
	}
	if len(call.Args) == 0 {
		return nil
	}
	name, ok := isoStringLiteral(call.Args[0])
	if !ok {
		return nil
	}
	reason, generic := isoGenericDatabasePathEnvVars[name]
	if !generic {
		if r, conditional := isoSQLiteDSNEnvVars[name]; conditional && a.fileHandlesFileDSN {
			reason = r
		} else {
			return nil
		}
	}
	return &Finding{
		Code:     "DATABASE_PATH_FROM_ENVIRONMENT",
		Severity: SeverityError,
		Title:    "Database location read from a generic environment variable",
		Message: "This file reads " + name + " — " + reason + ". A variable that does not " +
			"identify a scenario cannot be scoped to one: any process that exports it " +
			"redirects the database of every scenario it goes on to start, and a " +
			"supervisor that restarts sick scenarios does exactly that. This is not " +
			"hypothetical — it put twelve scenarios, including Test Genie's run ledger, " +
			"into one 9.35 GB file behind a single writer lock, and no scenario's own " +
			"tests could see it because the defect lives in process environment " +
			"inheritance rather than in any code path a test exercises.",
		Location: hygieneLoc(relPath, fset, call.Pos()),
		Remediation: "Resolve the database from the scenario's own identity: " +
			"storage.SQLiteDSN(storage.SQLiteConfig{Scenario: \"<scenario>\"}), or " +
			"database.Config{Driver: database.DriverSQLite, Scenario: \"<scenario>\"}. " +
			"For an explicit path in a test or migration, pass it as an ARGUMENT to " +
			"storage.SQLiteDSNAt(path, tuning) rather than through the environment. " +
			"To isolate storage for a test run, set VROOLI_STORAGE_ROOT, which redirects " +
			"the whole class tree and stays scenario-agnostic.",
		Analyzer: a.Name(),
	}
}

// checkHandRolledDSN flags a fmt.Sprintf that assembles a "file:...?_pragma=" DSN.
func (a isoDatabasePath) checkHandRolledDSN(call *ast.CallExpr, fset *token.FileSet, relPath string) *Finding {
	if !hygieneIsSelectorCall(call, "fmt", "Sprintf") || len(call.Args) == 0 {
		return nil
	}
	format, ok := isoStringLiteral(call.Args[0])
	if !ok || !isoLooksLikeSQLiteDSN(format) {
		return nil
	}
	finding := a.dsnFinding(relPath, hygieneLoc(relPath, fset, call.Pos()))
	return &finding
}

// checkPragmaLiterals catches a DSN built by concatenation instead of Sprintf.
func (a isoDatabasePath) checkPragmaLiterals(file *ast.File, fset *token.FileSet, relPath string) []Finding {
	var findings []Finding
	seen := map[string]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		value, ok := isoStringLiteral(lit)
		if !ok || !isoLooksLikeSQLiteDSN(value) {
			return true
		}
		loc := hygieneLoc(relPath, fset, lit.Pos())
		if seen[loc] {
			return true
		}
		seen[loc] = true
		findings = append(findings, a.dsnFinding(relPath, loc))
		return true
	})
	return findings
}

func (a isoDatabasePath) dsnFinding(_ string, loc string) Finding {
	return Finding{
		Code:     "DATABASE_PATH_FROM_ENVIRONMENT",
		Severity: SeverityWarning,
		Title:    "Hand-rolled SQLite DSN",
		Message: "This file assembles a SQLite DSN itself instead of resolving it through " +
			"api-core/storage. Sixty-plus scenarios each carried a private copy of this " +
			"string and they drifted into four different implementations — including one " +
			"still written in the pre-\"_pragma\" grammar that the modernc driver ignores, " +
			"so that scenario had never actually run in WAL mode despite a comment saying " +
			"it did. A private DSN also means a tuning change must be applied everywhere " +
			"or nowhere.",
		Location: loc,
		Remediation: "Use storage.SQLiteDSN(storage.SQLiteConfig{Scenario: \"<scenario>\"}) for the " +
			"scenario's own database, or storage.SQLiteDSNAt(path, tuning) for an explicit " +
			"path. A genuine deviation belongs in storage.SQLiteTuning as a typed field " +
			"(PageSizeBytes, MMapSizeBytes, TxLock, ...), not in a private string.",
		Analyzer: a.Name(),
	}
}

// isoLooksLikeSQLiteDSN reports whether a string literal carries SQLite
// connection grammar, meaning a DSN is being assembled by hand here.
//
// It deliberately does NOT require a "file:" prefix on the same literal. The
// drifted copies concatenate rather than format — `"file:" + path + "?_journal_mode=WAL"` —
// so the scheme and the grammar land in two different literals and a
// prefix-anchored check misses exactly the cases most worth catching. Test
// files are already exempt from this analyzer, so a test asserting on a DSN
// fragment does not reach here.
func isoLooksLikeSQLiteDSN(s string) bool {
	// An in-memory DSN names no file, so nothing can be captured by it and no
	// tuning drift matters.
	if strings.Contains(s, ":memory:") {
		return false
	}
	for _, marker := range sqliteDSNMarkers {
		idx := strings.Index(s, marker)
		if idx < 0 {
			continue
		}
		// The marker must actually carry a value. A bare "_pragma=" is a
		// fragment used to talk ABOUT DSNs — in a parser, a matcher, or this
		// analyzer itself — not a DSN being assembled.
		if len(s) > idx+len(marker) {
			return true
		}
	}
	return false
}

// sqliteDSNMarkers are the connection-grammar tokens that identify a
// hand-assembled SQLite DSN. "_pragma" is the modern form; the others are the
// pre-"_pragma" grammar the modernc driver silently ignores, which is why they
// matter just as much — a scenario using them is not configured at all.
var sqliteDSNMarkers = []string{"_pragma=", "_journal_mode=", "_busy_timeout=", "_txlock="}

// isoStringLiteral extracts an unquoted Go string literal value.
func isoStringLiteral(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	value := lit.Value
	if len(value) >= 2 && (value[0] == '`' || value[0] == '"') {
		value = value[1 : len(value)-1]
	}
	return value, true
}
