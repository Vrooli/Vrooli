package installgateway

import (
	"context"
	"fmt"
	"runtime"
	"testing"
)

type fixtureRegistrySizer struct {
	sizes    map[string]artifactSize
	optional map[string]string
}

func (f fixtureRegistrySizer) Registry(_ context.Context, name, version, _ string) (artifactSize, error) {
	size, ok := f.sizes[name+"@"+version]
	if !ok {
		return artifactSize{}, fmt.Errorf("no fixture size for %s@%s", name, version)
	}
	return size, nil
}

func (f fixtureRegistrySizer) OptionalDependencies(context.Context, string, string) (map[string]string, error) {
	return f.optional, nil
}

func (f fixtureRegistrySizer) Tarball(context.Context, string, string) (artifactSize, error) {
	return artifactSize{}, fmt.Errorf("fixture tarball probe is not used")
}

func TestProbeResourceAgentGrowthSumsPlatformOptionalDependencies(t *testing.T) {
	arch := map[string]string{"amd64": "x64", "arm64": "arm64", "386": "ia32"}[runtime.GOARCH]
	if arch == "" {
		t.Skip("unsupported fixture architecture")
	}
	matching := "fixture-cli-" + runtime.GOOS + "-" + arch
	other := "fixture-cli-other-" + runtime.GOOS + "-" + arch
	platformVersion := "2.0.0-" + runtime.GOOS + "-" + arch
	fake := fixtureRegistrySizer{
		sizes: map[string]artifactSize{
			"fixture-cli@2.0.0":               {Unpacked: 10, Archive: 1},
			"@fixture/cli@" + platformVersion: {Unpacked: 20, Archive: 2},
			other + "@2.0.0":                  {Unpacked: 1000, Archive: 100},
			"fixture-cli-unrelated@2.0.0":     {Unpacked: 1000, Archive: 100},
		},
		optional: map[string]string{matching: "npm:@fixture/cli@" + platformVersion, "fixture-cli-unrelated": "2.0.0"},
	}
	previous := newRegistrySizer
	newRegistrySizer = func() (registrySizer, error) { return fake, nil }
	t.Cleanup(func() { newRegistrySizer = previous })

	total, err := probeResourceAgentGrowth(context.Background(), Resolution{PackageName: "fixture-cli", HostToolTargetVersion: "2.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	if want := int64(10 + 1 + 20 + 2); total != want {
		t.Fatalf("total = %d, want %d (only the platform-relevant optional dependency is counted)", total, want)
	}
}

func TestResolveRegistryPackage(t *testing.T) {
	for _, test := range []struct {
		name, spec, wantName, wantVersion string
		wantError                         bool
	}{
		{name: "plain version", spec: "1.2.3", wantName: "fixture-cli", wantVersion: "1.2.3"},
		{name: "scoped npm alias", spec: "npm:@openai/codex@0.171.0-linux-x64", wantName: "@openai/codex", wantVersion: "0.171.0-linux-x64"},
		{name: "unscoped npm alias", spec: "npm:fixture-cli@2.0.0", wantName: "fixture-cli", wantVersion: "2.0.0"},
		{name: "range alias rejected", spec: "npm:@openai/codex@^0.171.0", wantError: true},
		{name: "malformed alias rejected", spec: "npm:@openai/codex", wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			name, version, err := resolveRegistryPackage("fixture-cli", test.spec)
			if (err != nil) != test.wantError {
				t.Fatalf("error = %v, wantError=%t", err, test.wantError)
			}
			if err == nil && (name != test.wantName || version != test.wantVersion) {
				t.Fatalf("resolved = %s@%s, want %s@%s", name, version, test.wantName, test.wantVersion)
			}
		})
	}
}
