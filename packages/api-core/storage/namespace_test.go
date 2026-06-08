package storage

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

// envMap turns a fixture map into a NamespaceConfig.EnvGet seam so tests never
// touch the real process environment.
func envMap(m map[string]string) func(string) string {
	return func(key string) string { return m[key] }
}

func TestResolveNamespace_RootComposition(t *testing.T) {
	tests := []struct {
		name        string
		env         map[string]string
		wantRoot    string
		wantVariant string
		wantLive    bool
	}{
		{
			name:        "live from injected namespace",
			env:         map[string]string{EnvStorageNamespace: "swarm-manager", EnvVariant: "live"},
			wantRoot:    "swarm-manager",
			wantVariant: "live",
			wantLive:    true,
		},
		{
			name:        "shadow from injected namespace",
			env:         map[string]string{EnvStorageNamespace: "swarm-manager_shadow", EnvVariant: "shadow"},
			wantRoot:    "swarm-manager_shadow",
			wantVariant: "shadow",
			wantLive:    false,
		},
		{
			name:        "namespace root is authoritative even if variant env missing",
			env:         map[string]string{EnvStorageNamespace: "swarm-manager_shadow"},
			wantRoot:    "swarm-manager_shadow",
			wantVariant: "live", // advisory only; the root already folds the variant in
			wantLive:    true,
		},
		{
			name:        "live fallback to bare scenario slug",
			env:         map[string]string{EnvScenario: "agent-manager"},
			wantRoot:    "agent-manager",
			wantVariant: "live",
			wantLive:    true,
		},
		{
			name:        "explicit live variant with scenario fallback",
			env:         map[string]string{EnvScenario: "agent-manager", EnvVariant: "live"},
			wantRoot:    "agent-manager",
			wantVariant: "live",
			wantLive:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns, err := ResolveNamespace(NamespaceConfig{EnvGet: envMap(tt.env)})
			if err != nil {
				t.Fatalf("ResolveNamespace returned error: %v", err)
			}
			if ns.Root() != tt.wantRoot {
				t.Errorf("Root() = %q, want %q", ns.Root(), tt.wantRoot)
			}
			if ns.Variant() != tt.wantVariant {
				t.Errorf("Variant() = %q, want %q", ns.Variant(), tt.wantVariant)
			}
			if ns.IsLive() != tt.wantLive {
				t.Errorf("IsLive() = %v, want %v", ns.IsLive(), tt.wantLive)
			}
		})
	}
}

func TestResolveNamespace_ExplicitRootSeam(t *testing.T) {
	ns, err := ResolveNamespace(NamespaceConfig{Root: "swarm-manager_shadow", Variant: "shadow"})
	if err != nil {
		t.Fatalf("ResolveNamespace returned error: %v", err)
	}
	if ns.Root() != "swarm-manager_shadow" {
		t.Errorf("Root() = %q, want swarm-manager_shadow", ns.Root())
	}
	if ns.IsLive() {
		t.Error("IsLive() = true, want false for shadow root")
	}
	// Explicit Root must win over the environment entirely.
	ns2, err := ResolveNamespace(NamespaceConfig{
		Root:   "explicit",
		EnvGet: envMap(map[string]string{EnvStorageNamespace: "from-env"}),
	})
	if err != nil {
		t.Fatalf("ResolveNamespace returned error: %v", err)
	}
	if ns2.Root() != "explicit" {
		t.Errorf("explicit Root not honored: got %q", ns2.Root())
	}
}

func TestResolveNamespace_FailLoud(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
	}{
		{
			name: "non-live variant with no namespace root",
			env:  map[string]string{EnvScenario: "swarm-manager", EnvVariant: "shadow"},
		},
		{
			name: "no scenario identity at all",
			env:  map[string]string{},
		},
		{
			name: "invalid namespace root from env",
			env:  map[string]string{EnvStorageNamespace: "bad/root"},
		},
		{
			name: "invalid scenario fallback",
			env:  map[string]string{EnvScenario: "bad slug"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveNamespace(NamespaceConfig{EnvGet: envMap(tt.env)})
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			var serr *Error
			if !errors.As(err, &serr) || serr.Kind != ErrInvalidInput {
				t.Fatalf("expected *Error{Kind: invalid_input}, got %v", err)
			}
		})
	}
}

func TestResolveNamespace_FallbackScenario(t *testing.T) {
	// No env at all: the compile-time fallback slug stands in as a live root, so
	// a binary run outside the variant-aware lifecycle still resolves.
	ns, err := ResolveNamespace(NamespaceConfig{
		EnvGet:           envMap(map[string]string{}),
		FallbackScenario: "my-scenario",
	})
	if err != nil {
		t.Fatalf("ResolveNamespace with fallback returned error: %v", err)
	}
	if ns.Root() != "my-scenario" || !ns.IsLive() {
		t.Errorf("fallback Root()/IsLive() = %q/%v, want my-scenario/true", ns.Root(), ns.IsLive())
	}

	// An injected namespace root always wins over the fallback.
	ns2, err := ResolveNamespace(NamespaceConfig{
		EnvGet:           envMap(map[string]string{EnvStorageNamespace: "my-scenario_shadow", EnvVariant: "shadow"}),
		FallbackScenario: "my-scenario",
	})
	if err != nil {
		t.Fatalf("ResolveNamespace returned error: %v", err)
	}
	if ns2.Root() != "my-scenario_shadow" {
		t.Errorf("injected root must win over fallback: got %q", ns2.Root())
	}

	// VROOLI_SCENARIO (lifecycle-set bare slug) wins over the fallback too.
	ns3, _ := ResolveNamespace(NamespaceConfig{
		EnvGet:           envMap(map[string]string{EnvScenario: "from-env"}),
		FallbackScenario: "my-scenario",
	})
	if ns3.Root() != "from-env" {
		t.Errorf("VROOLI_SCENARIO must win over fallback: got %q", ns3.Root())
	}
}

func TestResolveNamespace_FallbackNeverAliasesShadow(t *testing.T) {
	// The one invariant the whole mechanism protects: a non-live variant with no
	// injected root must fail loud even when a fallback slug is supplied, rather
	// than silently writing shadow state into the live namespace.
	_, err := ResolveNamespace(NamespaceConfig{
		EnvGet:           envMap(map[string]string{EnvVariant: "shadow"}),
		FallbackScenario: "my-scenario",
	})
	if err == nil {
		t.Fatal("expected fail-loud error for non-live variant with fallback, got nil")
	}
	var serr *Error
	if !errors.As(err, &serr) || serr.Kind != ErrInvalidInput {
		t.Fatalf("expected *Error{Kind: invalid_input}, got %v", err)
	}
}

func TestScenarioNamespace(t *testing.T) {
	// Live in the lifecycle: resolves to the bare slug, so on-disk paths are
	// unchanged from pre-Baseline-Modes behavior.
	t.Run("live injected", func(t *testing.T) {
		t.Setenv(EnvStorageNamespace, "my-scenario")
		t.Setenv(EnvVariant, "live")
		got, err := ScenarioNamespace("my-scenario")
		if err != nil {
			t.Fatalf("ScenarioNamespace: %v", err)
		}
		if got != "my-scenario" {
			t.Errorf("ScenarioNamespace = %q, want my-scenario", got)
		}
	})

	// Shadow in the lifecycle: resolves to the variant-suffixed root, so the
	// resolver scopes SQLite/data to a distinct directory.
	t.Run("shadow injected", func(t *testing.T) {
		t.Setenv(EnvStorageNamespace, "my-scenario_shadow")
		t.Setenv(EnvVariant, "shadow")
		got, err := ScenarioNamespace("my-scenario")
		if err != nil {
			t.Fatalf("ScenarioNamespace: %v", err)
		}
		if got != "my-scenario_shadow" {
			t.Errorf("ScenarioNamespace = %q, want my-scenario_shadow", got)
		}
	})

	// Non-live with no injected root still fails loud through the helper.
	t.Run("shadow without root fails loud", func(t *testing.T) {
		t.Setenv(EnvVariant, "shadow")
		if _, err := ScenarioNamespace("my-scenario"); err == nil {
			t.Fatal("expected fail-loud error, got nil")
		}
	})
}

// TestScenarioNamespace_ResolverPathIsolation exercises the exact resolution the
// react-vite template's sqliteDSN() performs: ScenarioNamespace(slug) feeding
// Resolver.Path(Options{ScenarioID}, ClassData, ...). It is the end-to-end proof
// that a generated scenario's SQLite file auto-isolates under a shadow with zero
// per-scenario work — live and shadow resolve to distinct on-disk paths.
func TestScenarioNamespace_ResolverPathIsolation(t *testing.T) {
	// Hermetic: clear any inherited namespace env so the live case truly relies
	// on the compile-time fallback slug.
	t.Setenv(EnvStorageNamespace, "")
	t.Setenv(EnvScenario, "")
	t.Setenv(EnvVariant, "")

	tmp := t.TempDir()
	resolve := func() string {
		t.Helper()
		r, err := NewResolver(ResolverConfig{
			AppID:   "vrooli",
			Profile: ProfileAuto,
			EnvGet:  envMap(map[string]string{envStorageRoot: tmp}),
		})
		if err != nil {
			t.Fatalf("NewResolver: %v", err)
		}
		id, err := ScenarioNamespace("demo")
		if err != nil {
			t.Fatalf("ScenarioNamespace: %v", err)
		}
		p, err := r.Path(Options{ScenarioID: id}, ClassData, "demo.db")
		if err != nil {
			t.Fatalf("Path: %v", err)
		}
		return p
	}

	// Live: no namespace injected ⇒ fallback slug ⇒ path scoped to ".../demo/".
	livePath := resolve()
	if !strings.HasSuffix(livePath, filepath.Join("vrooli", "demo", "demo.db")) {
		t.Errorf("live path %q not scoped to the bare slug", livePath)
	}

	// Shadow: lifecycle-injected namespace ⇒ path scoped to ".../demo_shadow/".
	t.Setenv(EnvStorageNamespace, "demo_shadow")
	t.Setenv(EnvVariant, "shadow")
	shadowPath := resolve()
	if !strings.HasSuffix(shadowPath, filepath.Join("vrooli", "demo_shadow", "demo.db")) {
		t.Errorf("shadow path %q not scoped to demo_shadow", shadowPath)
	}

	if livePath == shadowPath {
		t.Fatalf("shadow and live SQLite paths must differ; both = %q", livePath)
	}
}

func TestNamespace_Collection(t *testing.T) {
	live, _ := ResolveNamespace(NamespaceConfig{Root: "swarm-manager"})
	shadow, _ := ResolveNamespace(NamespaceConfig{Root: "swarm-manager_shadow"})

	gotLive, err := live.Collection("backlog")
	if err != nil {
		t.Fatalf("live.Collection: %v", err)
	}
	if gotLive != "swarm-manager_backlog" {
		t.Errorf("live Collection = %q, want swarm-manager_backlog", gotLive)
	}

	gotShadow, err := shadow.Collection("backlog")
	if err != nil {
		t.Fatalf("shadow.Collection: %v", err)
	}
	if gotShadow != "swarm-manager_shadow_backlog" {
		t.Errorf("shadow Collection = %q, want swarm-manager_shadow_backlog", gotShadow)
	}

	// The whole point: live and shadow never collide.
	if gotLive == gotShadow {
		t.Error("live and shadow collections must differ")
	}
}

func TestNamespace_RedisPrefix(t *testing.T) {
	ns, _ := ResolveNamespace(NamespaceConfig{Root: "swarm-manager"})
	got, err := ns.RedisPrefix("idea")
	if err != nil {
		t.Fatalf("RedisPrefix: %v", err)
	}
	if got != "swarm-manager:idea:" {
		t.Errorf("RedisPrefix = %q, want swarm-manager:idea:", got)
	}

	shadow, _ := ResolveNamespace(NamespaceConfig{Root: "swarm-manager_shadow"})
	gotShadow, _ := shadow.RedisPrefix("idea")
	if gotShadow != "swarm-manager_shadow:idea:" {
		t.Errorf("shadow RedisPrefix = %q, want swarm-manager_shadow:idea:", gotShadow)
	}
}

func TestNamespace_RedisKey_MidStringSegments(t *testing.T) {
	ns, _ := ResolveNamespace(NamespaceConfig{Root: "swarm-manager"})

	// Reproduces swarm-manager's "swarm-manager:idea:<id>:research" shape — the
	// reason a flat prefix is insufficient (the dynamic token sits mid-string).
	got, err := ns.RedisKey("idea", "abc123", "research")
	if err != nil {
		t.Fatalf("RedisKey: %v", err)
	}
	if got != "swarm-manager:idea:abc123:research" {
		t.Errorf("RedisKey = %q, want swarm-manager:idea:abc123:research", got)
	}

	// agent-manager's "sandbox:run:<id>" shape (domain + single dynamic token).
	got2, _ := ns.RedisKey("sandbox", "run", "run-42")
	if got2 != "swarm-manager:sandbox:run:run-42" {
		t.Errorf("RedisKey = %q, want swarm-manager:sandbox:run:run-42", got2)
	}

	// No segments degrades to "<root>:<domain>".
	got3, _ := ns.RedisKey("idea")
	if got3 != "swarm-manager:idea" {
		t.Errorf("RedisKey with no segments = %q, want swarm-manager:idea", got3)
	}

	// A shadow key never aliases the live key.
	shadow, _ := ResolveNamespace(NamespaceConfig{Root: "swarm-manager_shadow"})
	gotShadow, _ := shadow.RedisKey("idea", "abc123", "research")
	if gotShadow == got {
		t.Error("shadow and live redis keys must differ")
	}
}

func TestNamespace_RedisKey_EmptySegmentRejected(t *testing.T) {
	ns, _ := ResolveNamespace(NamespaceConfig{Root: "swarm-manager"})
	if _, err := ns.RedisKey("idea", "ok", "  "); err == nil {
		t.Fatal("expected an error for a blank segment, got nil")
	}
}

func TestCleanDomain_Rejection(t *testing.T) {
	ns, _ := ResolveNamespace(NamespaceConfig{Root: "swarm-manager"})
	bad := []string{"", "  ", "with space", "has:colon", "path/sep", "back\\slash", "dot.domain"}
	for _, d := range bad {
		if _, err := ns.Collection(d); err == nil {
			t.Errorf("Collection(%q) expected error, got nil", d)
		}
		if _, err := ns.RedisPrefix(d); err == nil {
			t.Errorf("RedisPrefix(%q) expected error, got nil", d)
		}
	}

	good := []string{"backlog", "idea", "records", "sandbox-run", "v2_embeddings", "ABC123"}
	for _, d := range good {
		if _, err := ns.Collection(d); err != nil {
			t.Errorf("Collection(%q) unexpected error: %v", d, err)
		}
	}
}

func TestPackageLevelConvenience(t *testing.T) {
	// The package-level helpers read the real process env via os.Getenv; set it
	// for the duration of this test to exercise the zero-config adoption seam.
	t.Setenv(EnvStorageNamespace, "swarm-manager_shadow")
	t.Setenv(EnvVariant, "shadow")

	coll, err := Collection("backlog")
	if err != nil {
		t.Fatalf("Collection: %v", err)
	}
	if coll != "swarm-manager_shadow_backlog" {
		t.Errorf("Collection = %q, want swarm-manager_shadow_backlog", coll)
	}

	prefix, err := RedisPrefix("idea")
	if err != nil {
		t.Fatalf("RedisPrefix: %v", err)
	}
	if prefix != "swarm-manager_shadow:idea:" {
		t.Errorf("RedisPrefix = %q, want swarm-manager_shadow:idea:", prefix)
	}

	key, err := RedisKey("idea", "abc123", "research")
	if err != nil {
		t.Fatalf("RedisKey: %v", err)
	}
	if key != "swarm-manager_shadow:idea:abc123:research" {
		t.Errorf("RedisKey = %q, want swarm-manager_shadow:idea:abc123:research", key)
	}
}
