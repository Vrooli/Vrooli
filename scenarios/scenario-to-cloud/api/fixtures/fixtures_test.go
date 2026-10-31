package fixtures

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var catalogRoot = filepath.Join("..", "..", "fixtures")

func mustCatalog(t *testing.T) *Catalog {
	t.Helper()
	cat, err := Load(catalogRoot)
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	return cat
}

// TestCatalogDeclaresEveryCertifiedShape [REQ:STC-P0-040] proves the catalog
// holds the workload shapes the support policy names and both coexistence
// journeys.
func TestCatalogDeclaresEveryCertifiedShape(t *testing.T) {
	cat := mustCatalog(t)
	for _, id := range []string{"stateless-web", "headless-api", "sql-uploads", "dependent-chain", "resource-heavy"} {
		if _, ok := cat.Workloads[id]; !ok {
			t.Errorf("workload fixture %s missing", id)
		}
	}
	for _, id := range []string{"two-deployments", "existing-full-install"} {
		if _, ok := cat.Coexistence[id]; !ok {
			t.Errorf("coexistence fixture %s missing", id)
		}
	}
	if cat.Workloads["resource-heavy"].Classification != "compatibility_only" {
		t.Errorf("resource-heavy must be compatibility_only per support policy")
	}
	if len(cat.Workloads["sql-uploads"].Declaration.PersistentData) != 2 {
		t.Errorf("sql-uploads must declare a SQL and a file binding")
	}
	if len(cat.Workloads["dependent-chain"].Declaration.Scenarios) == 0 {
		t.Errorf("dependent-chain must declare a scenario dependency")
	}
}

// TestOracleIsReproducibleFromSeed [REQ:STC-P0-040] (P02-A06): computing the
// oracle twice from the same seed yields the same expected state, and it
// matches the declared checksums.
func TestOracleIsReproducibleFromSeed(t *testing.T) {
	cat := mustCatalog(t)
	for id, w := range cat.Workloads {
		first, err := VerifyOracle(w)
		if err != nil {
			t.Errorf("%s: %v", id, err)
			continue
		}
		second, err := Compute(w)
		if err != nil {
			t.Errorf("%s: recompute: %v", id, err)
			continue
		}
		if first.OracleChecksum != second.OracleChecksum || first.SeedChecksum != second.SeedChecksum {
			t.Errorf("%s: oracle not deterministic: %s vs %s", id, first.OracleChecksum, second.OracleChecksum)
		}
		if first.ExpectedState.TotalRecords == 0 {
			t.Errorf("%s: seed must carry at least one record", id)
		}
	}
}

// TestOracleDetectsSeedDrift proves a single changed seed byte changes the
// oracle, so the checksum is a real oracle and not a constant.
func TestOracleDetectsSeedDrift(t *testing.T) {
	cat := mustCatalog(t)
	src := cat.Workloads["stateless-web"]
	dir := t.TempDir()
	for _, rel := range src.Seed.Files {
		data, err := os.ReadFile(filepath.Join(src.Dir, rel))
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		dst := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		data[len(data)-2] ^= 0x01
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	drifted := *src
	drifted.Dir = dir
	if _, err := VerifyOracle(&drifted); err == nil {
		t.Fatalf("changed seed bytes must fail oracle verification")
	}
}

// TestEveryFixtureDeclaresCleanupOwnership [REQ:STC-P0-040] (P02-A02/A03
// groundwork): every workload and coexistence fixture declares its lease
// scope, owned-ephemeral target ownership, cleanup owner and policy.
func TestEveryFixtureDeclaresCleanupOwnership(t *testing.T) {
	cat := mustCatalog(t)
	seen := map[string]string{}
	check := func(id string, o Ownership) {
		if err := o.Validate(); err != nil {
			t.Errorf("%s: %v", id, err)
		}
		if prev, dup := seen[o.LeaseScope]; dup {
			t.Errorf("%s and %s share lease scope %s; scopes must be unique per fixture", prev, id, o.LeaseScope)
		}
		seen[o.LeaseScope] = id
		if !strings.Contains(o.CleanupPolicy, "retain_failed_evidence") {
			t.Errorf("%s: cleanup policy must retain failed evidence, got %q", id, o.CleanupPolicy)
		}
		if o.NeighborInvariant == "" || o.WriteScopeInvariant == "" {
			t.Errorf("%s: neighbor and write-scope invariants must be stated", id)
		}
	}
	for id, w := range cat.Workloads {
		check(id, w.Ownership)
		if w.Lifecycle.Status != "declared_not_executable" {
			t.Errorf("%s: no fixture owner runtime exists at phase 2; lifecycle.status must say so, got %q", id, w.Lifecycle.Status)
		}
	}
	for id, c := range cat.Coexistence {
		check("coexistence/"+id, c.Ownership)
	}
}

func TestLoadRefusesFixtureWithoutOwnership(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "workloads", "bare")
	if err := os.MkdirAll(filepath.Join(dir, "seed"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "seed", "x.json"), []byte("[1]"), 0o644); err != nil {
		t.Fatal(err)
	}
	doc := `{"schema_version":1,"id":"bare","kind":"workload","shape":"s","classification":"certified",
"declaration":{"listeners":[{"name":"api","protocol":"http","role":"api","public":true,"path_prefix":"/"}]},
"seed":{"version":"1","files":["seed/x.json"],"records":1,"checksum":"sha256:0"},
"oracle":{"algorithm":"fixture-oracle/v1","checksum":"sha256:0"},
"ownership":{"lease_scope":"","target_ownership":"operator"},"lifecycle":{"status":"declared_not_executable"}}`
	if err := os.WriteFile(filepath.Join(dir, "fixture.json"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "lease_scope") {
		t.Fatalf("expected lease_scope refusal, got %v", err)
	}
}

// TestFixturesCarryNoProductNamesOrSecrets [REQ:STC-P0-043]: fixtures are
// generic and value-free.
func TestFixturesCarryNoProductNamesOrSecrets(t *testing.T) {
	banned := []string{"landing-page-business-suite", "lpbs", "vrooli.com", "138.197.95.182", "password=", "BEGIN OPENSSH PRIVATE KEY", "/home/"}
	err := filepath.Walk(catalogRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lower := strings.ToLower(string(data))
		for _, b := range banned {
			if strings.Contains(lower, strings.ToLower(b)) {
				t.Errorf("%s contains banned token %q", path, b)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
}

func TestCoexistenceJourneysReferenceKnownWorkloadsAndCases(t *testing.T) {
	cat := mustCatalog(t)
	two := cat.Coexistence["two-deployments"]
	roles := map[string]bool{}
	for _, p := range two.Participants {
		roles[p.Role] = true
	}
	if !roles["subject"] || !roles["survivor"] {
		t.Fatalf("two-deployments needs a subject and a survivor, got %v", roles)
	}
	existing := cat.Coexistence["existing-full-install"]
	if !strings.Contains(strings.ToLower(existing.Description), "unrelated") {
		t.Fatalf("existing-full-install must describe the unrelated workload it preserves")
	}
	for _, c := range cat.Coexistence {
		for _, id := range c.MatrixCases {
			if !strings.Contains(id, "-") {
				t.Errorf("%s: matrix case %q is not a case id", c.ID, id)
			}
		}
	}
}
