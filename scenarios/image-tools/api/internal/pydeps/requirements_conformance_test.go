package pydeps

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// These tests are the lock↔governance conformance guard. They keep the pinned
// requirements.lock honest against (a) the in-repo ranged requirements.in — the
// always-on drift guard that makes the transformers<5 class of regression a test
// failure, not a runtime crash — and (b) the Scenario Dependency Analyzer
// governance store when it is present (best-effort: the untracked store is absent
// in clean CI clones, so that cross-check skips rather than false-fails).

type pin struct {
	name    string
	version string
	hashed  bool
}

type directDep struct {
	name       string
	constraint string
}

// normalize folds a Python distribution name to its PEP 503 canonical form so
// "Pillow", "huggingface_hub" and "huggingface-hub" all compare equal.
func normalize(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	n = strings.ReplaceAll(n, "_", "-")
	n = strings.ReplaceAll(n, ".", "-")
	return n
}

// parseRequirementsIn extracts the directly-declared deps and their version
// constraints from requirements.in (comments/blank lines ignored).
func parseRequirementsIn(t *testing.T) []directDep {
	t.Helper()
	var deps []directDep
	for _, raw := range strings.Split(string(InBytes()), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if i := strings.Index(line, "#"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		name, constraint := splitNameConstraint(line)
		if name == "" {
			continue
		}
		deps = append(deps, directDep{name: normalize(name), constraint: constraint})
	}
	if len(deps) == 0 {
		t.Fatal("parsed zero direct deps from requirements.in")
	}
	return deps
}

// splitNameConstraint splits "transformers>=4.49,<5" into ("transformers",
// ">=4.49,<5"). A bare "name" yields an empty constraint (any version).
func splitNameConstraint(spec string) (string, string) {
	idx := strings.IndexAny(spec, "<>=!~")
	if idx < 0 {
		return strings.TrimSpace(spec), ""
	}
	return strings.TrimSpace(spec[:idx]), strings.TrimSpace(spec[idx:])
}

// parseLock extracts the pinned (name==version) lines from requirements.lock,
// recording whether each pin carries at least one --hash (uv emits hashes inline
// or on continuation lines).
func parseLock(t *testing.T) map[string]pin {
	t.Helper()
	pins := map[string]pin{}
	lines := strings.Split(string(LockBytes()), "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "--") {
			continue
		}
		name, version, ok := strings.Cut(line, "==")
		if !ok {
			continue
		}
		name = normalize(name)
		// version may be followed by " \" (line continuation) and/or an inline hash.
		version = strings.TrimSpace(strings.TrimSuffix(version, "\\"))
		if j := strings.Index(version, " "); j >= 0 {
			version = version[:j]
		}
		hashed := strings.Contains(line, "--hash")
		// uv emits each pin's hashes on the immediately-following `--hash` lines;
		// consume them, stopping at the first non-hash line (the next pin/comment).
		for k := i + 1; k < len(lines); k++ {
			if strings.Contains(lines[k], "--hash") {
				hashed = true
				continue
			}
			break
		}
		pins[name] = pin{name: name, version: version, hashed: hashed}
	}
	if len(pins) == 0 {
		t.Fatal("parsed zero pins from requirements.lock")
	}
	return pins
}

func TestLockPinsEveryDirectDep(t *testing.T) {
	deps := parseRequirementsIn(t)
	pins := parseLock(t)
	for _, d := range deps {
		p, ok := pins[d.name]
		if !ok {
			t.Errorf("direct dep %q from requirements.in is not pinned in requirements.lock", d.name)
			continue
		}
		if !p.hashed {
			t.Errorf("pin %s==%s has no --hash (lock must be fully hashed)", p.name, p.version)
		}
		if !satisfies(t, p.version, d.constraint) {
			t.Errorf("locked %s==%s violates requirements.in constraint %q", p.name, p.version, d.constraint)
		}
	}
}

// TestTransformersCeilingHeld is the explicit guard for the exact regression that
// motivated this work: transformers 5.x removed CLIPTextModel.text_model and
// broke diffusers SD/InstructPix2Pix loading. The ceiling lives in the lock (via
// requirements.in), NOT in the shared SDA range, so this test — not the fleet
// governance — is what keeps it nailed down.
func TestTransformersCeilingHeld(t *testing.T) {
	pins := parseLock(t)
	p, ok := pins["transformers"]
	if !ok {
		t.Fatal("transformers must be pinned in the lock")
	}
	if !satisfies(t, p.version, ">=4.49,<5") {
		t.Fatalf("transformers==%s must be >=4.49,<5 (5.x breaks diffusers CLIP loading)", p.version)
	}
}

// TestLockMatchesSDAGovernance cross-checks each direct dep against the Scenario
// Dependency Analyzer approved-dependencies store: it must be approved and the
// locked version must satisfy the approved (security/license) range. The store is
// untracked local state, so a clean CI clone without it skips this check — the
// hermetic tests above remain the durable drift guard.
func TestLockMatchesSDAGovernance(t *testing.T) {
	store, ok := loadGovernance(t)
	if !ok {
		t.Skip("SDA approved-dependencies store not found; hermetic lock↔requirements.in checks cover drift")
	}
	deps := parseRequirementsIn(t)
	pins := parseLock(t)
	for _, d := range deps {
		rec, found := store[d.name]
		if !found {
			t.Errorf("direct dep %q is not governed in SDA (run `scenario-dependency-analyzer deps-approved approve pip/%s ...`)", d.name, d.name)
			continue
		}
		if rec.state != "approved" {
			t.Errorf("direct dep %q SDA state is %q, want approved", d.name, rec.state)
		}
		if p, pinned := pins[d.name]; pinned && !satisfies(t, p.version, rec.versionRange) {
			t.Errorf("locked %s==%s is outside SDA approved range %q", d.name, p.version, rec.versionRange)
		}
	}
}

type govRecord struct {
	state        string
	versionRange string
}

// loadGovernance walks up from the module dir to find the repo-root SDA store and
// returns the pip records keyed by normalized name.
func loadGovernance(t *testing.T) (map[string]govRecord, bool) {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		return nil, false
	}
	var storePath string
	for {
		candidate := filepath.Join(dir, ".vrooli", "dependencies", "approved-dependencies.json")
		if _, err := os.Stat(candidate); err == nil {
			storePath = candidate
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, false
		}
		dir = parent
	}
	data, err := os.ReadFile(storePath)
	if err != nil {
		return nil, false
	}
	var doc struct {
		Records []struct {
			Ecosystem    string `json:"ecosystem"`
			PackageName  string `json:"package_name"`
			VersionRange string `json:"version_range"`
			State        string `json:"state"`
		} `json:"records"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse SDA store %q: %v", storePath, err)
	}
	out := map[string]govRecord{}
	for _, r := range doc.Records {
		if r.Ecosystem != "pip" {
			continue
		}
		out[normalize(r.PackageName)] = govRecord{state: r.State, versionRange: r.VersionRange}
	}
	return out, true
}

// satisfies evaluates a PEP 440 subset (comma-separated >=, >, <=, <, == over
// dotted-numeric versions; "" or "*" = any). Sufficient for the simple bounds in
// requirements.in / SDA ranges; pre-release and local (+cuXXX) tags are stripped.
func satisfies(t *testing.T, version, constraint string) bool {
	t.Helper()
	constraint = strings.TrimSpace(constraint)
	if constraint == "" || constraint == "*" {
		return true
	}
	for _, clause := range strings.Split(constraint, ",") {
		clause = strings.TrimSpace(clause)
		if clause == "" {
			continue
		}
		op, bound := splitOp(clause)
		cmp := compareVersions(parseVersion(version), parseVersion(bound))
		ok := false
		switch op {
		case ">=":
			ok = cmp >= 0
		case ">":
			ok = cmp > 0
		case "<=":
			ok = cmp <= 0
		case "<":
			ok = cmp < 0
		case "==":
			ok = cmp == 0
		case "!=":
			ok = cmp != 0
		case "~=":
			ok = cmp >= 0 // compatible-release lower bound (sufficient for our use)
		default:
			t.Fatalf("unsupported version operator %q in constraint %q", op, constraint)
		}
		if !ok {
			return false
		}
	}
	return true
}

func splitOp(clause string) (string, string) {
	for _, op := range []string{">=", "<=", "==", "!=", "~=", ">", "<"} {
		if strings.HasPrefix(clause, op) {
			return op, strings.TrimSpace(strings.TrimPrefix(clause, op))
		}
	}
	return "==", clause
}

// parseVersion turns "4.57.6" / "2.12.1+cu130" / "1.27.0rc1" into a numeric
// component slice, stopping at the first non-numeric segment.
func parseVersion(v string) []int {
	v = strings.TrimSpace(v)
	if i := strings.IndexAny(v, "+"); i >= 0 {
		v = v[:i]
	}
	var out []int
	for _, part := range strings.Split(v, ".") {
		// trim any trailing non-digit (e.g. rc/post/dev suffix on the segment).
		j := 0
		for j < len(part) && part[j] >= '0' && part[j] <= '9' {
			j++
		}
		if j == 0 {
			break
		}
		n, err := strconv.Atoi(part[:j])
		if err != nil {
			break
		}
		out = append(out, n)
		if j < len(part) {
			break
		}
	}
	return out
}

func compareVersions(a, b []int) int {
	for i := 0; i < len(a) || i < len(b); i++ {
		var av, bv int
		if i < len(a) {
			av = a[i]
		}
		if i < len(b) {
			bv = b[i]
		}
		if av != bv {
			if av < bv {
				return -1
			}
			return 1
		}
	}
	return 0
}

func TestSatisfiesComparator(t *testing.T) {
	cases := []struct {
		version    string
		constraint string
		want       bool
	}{
		{"4.57.6", ">=4.49,<5", true},
		{"5.0.0", ">=4.49,<5", false},
		{"4.49.0", ">=4.49,<5", true},
		{"4.48.0", ">=4.49,<5", false},
		{"2.12.1+cu130", ">=2.4", true},
		{"0.38.0", ">=0.36,<0.40", true},
		{"0.40.0", ">=0.36,<0.40", false},
		{"1.27.0", "*", true},
		{"1.27.0", "", true},
		{"12.2.0", ">=9", true},
	}
	for _, c := range cases {
		if got := satisfies(t, c.version, c.constraint); got != c.want {
			t.Errorf("satisfies(%q,%q)=%v want %v", c.version, c.constraint, got, c.want)
		}
	}
}

func TestLockHash_StableHexOfEmbeddedLock(t *testing.T) {
	h := LockHash()
	if len(h) != 64 {
		t.Fatalf("LockHash must be a 64-char hex sha-256, got %d chars: %q", len(h), h)
	}
	for _, c := range h {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Fatalf("LockHash has a non-hex char %q", c)
		}
	}
	if LockHash() != h {
		t.Error("LockHash must be deterministic")
	}
	// It must hash the SAME bytes Materialize writes (the smoke cache + pyenv
	// sentinel rely on this identity).
	sum := sha256.Sum256(LockBytes())
	if hex.EncodeToString(sum[:]) != h {
		t.Error("LockHash must equal sha256(LockBytes())")
	}
}

func TestMaterializeIdempotent(t *testing.T) {
	dir := t.TempDir()
	p1, err := Materialize(dir)
	if err != nil {
		t.Fatal(err)
	}
	info1, err := os.Stat(p1)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(p1) != LockName {
		t.Fatalf("materialized name = %q, want %q", filepath.Base(p1), LockName)
	}
	p2, err := Materialize(dir)
	if err != nil {
		t.Fatal(err)
	}
	info2, err := os.Stat(p2)
	if err != nil {
		t.Fatal(err)
	}
	if !info1.ModTime().Equal(info2.ModTime()) {
		t.Errorf("idempotent Materialize rewrote an unchanged lock (mtime changed)")
	}
	got, err := os.ReadFile(p1)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(LockBytes()) {
		t.Errorf("materialized content != embedded lock")
	}
}

func TestMaterializeRewritesStaleLock(t *testing.T) {
	dir := t.TempDir()
	stale := filepath.Join(dir, LockName)
	if err := os.WriteFile(stale, []byte("stale==0.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Materialize(dir); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(stale)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(LockBytes()) {
		t.Errorf("Materialize did not overwrite a stale lock")
	}
}

func TestMaterializeRejectsUnwritableDir(t *testing.T) {
	// A path whose parent is a regular file cannot be MkdirAll'd.
	dir := t.TempDir()
	file := filepath.Join(dir, "afile")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Materialize(filepath.Join(file, "sub")); err == nil {
		t.Error("expected Materialize to fail when dir cannot be created")
	}
}

func TestInAndLockBytesAreCopies(t *testing.T) {
	a := LockBytes()
	if len(a) == 0 {
		t.Fatal("LockBytes empty")
	}
	a[0] ^= 0xff
	if LockBytes()[0] == a[0] {
		t.Error("LockBytes must return a defensive copy")
	}
	in := InBytes()
	if len(in) == 0 {
		t.Fatal("InBytes empty")
	}
	in[0] ^= 0xff
	if InBytes()[0] == in[0] {
		t.Error("InBytes must return a defensive copy")
	}
}
