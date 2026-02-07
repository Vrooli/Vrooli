package storage

import "testing"

func TestResolveRejectsRelativeEnvRoots(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		env  map[string]string
	}{
		{name: "global", env: map[string]string{envStorageRoot: "relative/root"}},
		{name: "config", env: map[string]string{envConfigRoot: "relative/config"}},
		{name: "data", env: map[string]string{envDataRoot: "relative/data"}},
		{name: "cache", env: map[string]string{envCacheRoot: "relative/cache"}},
		{name: "logs", env: map[string]string{envLogsRoot: "relative/logs"}},
		{name: "state", env: map[string]string{envStateRoot: "relative/state"}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := mustResolver(t, ResolverConfig{AppID: "vrooli", EnvGet: mapEnv(tc.env)})
			_, err := r.Resolve(Options{ScenarioID: "demo"})
			if err == nil {
				t.Fatalf("expected relative path validation error")
			}
		})
	}
}

func TestResolveUnknownProfile(t *testing.T) {
	t.Parallel()

	r := mustResolver(t, ResolverConfig{AppID: "vrooli", Profile: Profile("unknown-profile"), EnvGet: mapEnv(nil)})
	_, err := r.Resolve(Options{ScenarioID: "demo"})
	if err == nil {
		t.Fatalf("expected unknown profile error")
	}
}

func TestPathPropagatesUnknownClassError(t *testing.T) {
	t.Parallel()

	r := mustResolver(t, ResolverConfig{AppID: "vrooli", EnvGet: mapEnv(nil)})
	_, err := r.Path(Options{ScenarioID: "demo", RootOverride: "/tmp/x"}, Class("unknown"), "a.txt")
	if err == nil {
		t.Fatalf("expected unknown class error")
	}
}
