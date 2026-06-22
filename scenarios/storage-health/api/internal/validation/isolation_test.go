package validation

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// isoCtx builds an AnalyzerContext rooted at a temp scenario whose api/main.go
// holds the given source, with the given engines/language.
func isoCtx(t *testing.T, mainGo string, engines []Engine, lang string) AnalyzerContext {
	t.Helper()
	scenarioDir := filepath.Join(t.TempDir(), "scenarios", "demo")
	writeFile(t, filepath.Join(scenarioDir, "api", "main.go"), mainGo)
	return AnalyzerContext{
		Scenario:    "demo",
		ScenarioDir: scenarioDir,
		APIDir:      filepath.Join(scenarioDir, "api"),
		Language:    lang,
		Engines:     engines,
	}
}

const isoWiredMain = `package main

import (
	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
)

func main() {
	db, _ := database.Open(nil, database.Config{})
	_ = database.EnsureSchemas(nil, db.Primary())
	mux := newMux()
	devrouting.Register(mux, db)
	_ = apihttp.TestModeMiddleware(mux)
}
`

func TestRoutedSeams_AllWiredIsClean(t *testing.T) {
	ac := isoCtx(t, isoWiredMain, []Engine{EngineSQLite}, "go")
	if !(isoRoutedSeams{}).Applies(ac) {
		t.Fatal("routed-seams analyzer should apply to a Go SQLite scenario")
	}
	got, _ := (isoRoutedSeams{}).Analyze(context.Background(), ac)
	if len(got) != 0 {
		t.Fatalf("expected no finding when all 4 seams wired, got %+v", got)
	}
}

func TestRoutedSeams_MissingSeamFlagged(t *testing.T) {
	// Same as wired, but drop devrouting.Register.
	missing := strings.Replace(isoWiredMain, "\tdevrouting.Register(mux, db)\n", "", 1)
	// also drop the now-unused import to keep it realistic
	missing = strings.Replace(missing, "\t\"github.com/vrooli/api-core/devrouting\"\n", "", 1)
	ac := isoCtx(t, missing, []Engine{EngineSQLite}, "go")
	got, _ := (isoRoutedSeams{}).Analyze(context.Background(), ac)
	if len(got) != 1 || got[0].Code != "ROUTED_SEAMS_UNWIRED" {
		t.Fatalf("expected ROUTED_SEAMS_UNWIRED, got %+v", got)
	}
	if got[0].Severity != SeverityError {
		t.Fatalf("ROUTED_SEAMS_UNWIRED severity = %v, want ERROR", got[0].Severity)
	}
	if !strings.Contains(got[0].Message, "devrouting.Register") {
		t.Fatalf("message should name the missing seam devrouting.Register; got: %s", got[0].Message)
	}
}

func TestRoutedSeams_RawSQLDBFixtureFailsClosed(t *testing.T) {
	// The deliberately non-cooperating shape: a raw *sql.DB with none of the
	// routed seams. Must produce the loud L2 fail-closed finding.
	rawMain := `package main

import "database/sql"

func main() {
	db, _ := sql.Open("sqlite", "file:app.db")
	_ = db
}
`
	ac := isoCtx(t, rawMain, []Engine{EngineSQLite}, "go")
	got, _ := (isoRoutedSeams{}).Analyze(context.Background(), ac)
	if len(got) != 1 || got[0].Code != "ROUTED_SEAMS_UNWIRED" {
		t.Fatalf("expected ROUTED_SEAMS_UNWIRED for raw sql.DB, got %+v", got)
	}
	// All four seams missing → message should mention real-data risk loudly.
	if !strings.Contains(got[0].Message, "REAL database") {
		t.Fatalf("expected loud real-data-risk message; got: %s", got[0].Message)
	}
}

func TestRoutedSeams_AliasedImportRecognized(t *testing.T) {
	aliased := `package main

import (
	httpx "github.com/vrooli/api-core/apihttp"
	db "github.com/vrooli/api-core/database"
	dr "github.com/vrooli/api-core/devrouting"
)

func main() {
	h, _ := db.Open(nil, db.Config{})
	_ = db.EnsureSchemas(nil, h.Primary())
	mux := newMux()
	dr.Register(mux, h)
	_ = httpx.TestModeMiddleware(mux)
}
`
	ac := isoCtx(t, aliased, []Engine{EngineSQLite}, "go")
	got, _ := (isoRoutedSeams{}).Analyze(context.Background(), ac)
	if len(got) != 0 {
		t.Fatalf("aliased imports of all 4 seams should be clean, got %+v", got)
	}
}

func TestRoutedSeams_NonRelationalDoesNotApply(t *testing.T) {
	ac := isoCtx(t, "package main\nfunc main() {}\n", []Engine{EngineRedis}, "go")
	if (isoRoutedSeams{}).Applies(ac) {
		t.Fatal("routed-seams must not apply to a Redis-only (non-relational) scenario")
	}
}

func TestUnverified_NonGoWithEngineFlagged(t *testing.T) {
	ac := AnalyzerContext{Scenario: "ts-demo", APIDir: "/x/api", Language: "typescript", Engines: []Engine{EnginePostgres}}
	if !(isoUnverified{}).Applies(ac) {
		t.Fatal("unverified analyzer should apply to a non-Go scenario with an engine")
	}
	got, _ := (isoUnverified{}).Analyze(context.Background(), ac)
	if len(got) != 1 || got[0].Code != "STORAGE_ISOLATION_UNVERIFIED" {
		t.Fatalf("expected STORAGE_ISOLATION_UNVERIFIED, got %+v", got)
	}
	if got[0].Severity != SeverityWarning {
		t.Fatalf("unverified severity = %v, want WARNING (advisory-visible)", got[0].Severity)
	}
}

func TestUnverified_GoOrStatelessDoesNotApply(t *testing.T) {
	goCtx := AnalyzerContext{APIDir: "/x/api", Language: "go", Engines: []Engine{EnginePostgres}}
	if (isoUnverified{}).Applies(goCtx) {
		t.Fatal("unverified must not apply to a Go scenario (routed seams cover it)")
	}
	statelessTS := AnalyzerContext{APIDir: "/x/api", Language: "typescript", Engines: nil}
	if (isoUnverified{}).Applies(statelessTS) {
		t.Fatal("unverified must not apply to a stateless non-Go scenario")
	}
}

func TestNamespaceHardcoded_QdrantConstFlagged(t *testing.T) {
	src := `package notes

import "github.com/vrooli/ai-go/search"

const NotesCollection = "swarm-manager_notes_embeddings"

func ensure(c *aisearch.Client) error {
	return c.EnsureCollection(NotesCollection)
}
`
	got := isoScanNamespaceFile(src, "api/internal/notes/qdrant.go")
	if len(got) == 0 || got[0].Code != "STORAGE_NAMESPACE_HARDCODED" {
		t.Fatalf("expected STORAGE_NAMESPACE_HARDCODED for hardcoded qdrant collection, got %+v", got)
	}
}

func TestNamespaceHardcoded_RedisPrefixFlagged(t *testing.T) {
	src := `package auth

import "github.com/redis/go-redis/v9"

const sessionPrefix = "lpbs:auth:session:"

func sessionKey(rdb *redis.Client, id string) string { return sessionPrefix + id }
`
	got := isoScanNamespaceFile(src, "api/internal/auth/redis.go")
	if len(got) == 0 {
		t.Fatalf("expected a Redis namespace finding, got none")
	}
}

func TestNamespaceHardcoded_HelperAdoptedClean(t *testing.T) {
	src := `package auth

import "github.com/vrooli/api-core/storage"

func sessionKey(id string) (string, error) { return storage.RedisKey("auth", "session", id) }
`
	if got := isoScanNamespaceFile(src, "api/internal/auth/redis.go"); len(got) != 0 {
		t.Fatalf("storage.RedisKey adopter must be clean, got %+v", got)
	}
}

func TestNamespaceHardcoded_HostPortNotFlagged(t *testing.T) {
	src := `package cache

import "github.com/redis/go-redis/v9"

func client() *redis.Client { return redis.NewClient(&redis.Options{Addr: "localhost:6379"}) }
`
	if got := isoScanNamespaceFile(src, "api/internal/cache/redis.go"); len(got) != 0 {
		t.Fatalf("host:port literal must not be flagged, got %+v", got)
	}
}

// TestIsolation_DogfoodStorageHealthClean asserts storage-health itself — which
// wires all four routed seams in its api/main.go and uses SQLite only — produces
// zero isolation findings.
func TestIsolation_DogfoodStorageHealthClean(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", "..", "..", ".."))
	scenarioDir := filepath.Join(repoRoot, "scenarios", "storage-health")
	ac := AnalyzerContext{
		Scenario:    "storage-health",
		ScenarioDir: scenarioDir,
		APIDir:      filepath.Join(scenarioDir, "api"),
		Language:    "go",
		Engines:     []Engine{EngineSQLite},
	}
	if got, _ := (isoRoutedSeams{}).Analyze(context.Background(), ac); len(got) != 0 {
		t.Fatalf("storage-health must pass routed-seams clean, got %+v", got)
	}
	if (isoNamespace{}).Applies(ac) {
		t.Fatal("namespace analyzer should not apply to storage-health (no qdrant/redis)")
	}
	if (isoUnverified{}).Applies(ac) {
		t.Fatal("unverified must not apply to storage-health (Go)")
	}
}
