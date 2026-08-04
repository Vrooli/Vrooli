package capabilityregistry

import (
	"context"
	"strings"
	"testing"
)

type checkerFunc func(context.Context) (Status, string)

func (f checkerFunc) Check(ctx context.Context) (Status, string) { return f(ctx) }

func validRegistry() *Registry {
	return New([]Def{{ID: "audio-tools", Name: "Audio Tools", Description: "speech", DependencyKind: DependencyScenario, DependencySlug: "audio-tools"}}, map[string]Checker{
		"audio-tools": checkerFunc(func(context.Context) (Status, string) { return StatusAvailable, "healthy" }),
	}, 0)
}

func TestValidateRejectsMalformedDefinitions(t *testing.T) {
	cases := []struct {
		name string
		def  Def
		want string
	}{
		{"id", Def{Description: "x", DependencyKind: DependencyScenario, DependencySlug: "x"}, "no id"},
		{"description", Def{ID: "x", DependencyKind: DependencyScenario, DependencySlug: "x"}, "no description"},
		{"kind", Def{ID: "x", Description: "x", DependencySlug: "x"}, "invalid dependency kind"},
		{"slug", Def{ID: "x", Description: "x", DependencyKind: DependencyScenario}, "no dependency slug"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := New([]Def{tc.def}, nil, 0).Validate(); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate() error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestDescribeContainsDefinitionsAndStates(t *testing.T) {
	data, err := validRegistry().Describe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"audio-tools"`) || !strings.Contains(string(data), `"available"`) {
		t.Fatalf("Describe() = %s", data)
	}
}

func TestValidateRequiresReachabilityChecker(t *testing.T) {
	reg := New([]Def{{ID: "audio-tools", Description: "speech", DependencyKind: DependencyScenario, DependencySlug: "audio-tools"}}, nil, 0)
	if err := reg.Validate(); err == nil || !strings.Contains(err.Error(), "reachability checker") {
		t.Fatalf("Validate() error = %v, want missing reachability checker", err)
	}
}

func TestValidateStatesRequiresRecoveryAction(t *testing.T) {
	reg := validRegistry()
	if err := reg.ValidateStates([]State{{Def: Def{ID: "audio-tools"}, Status: StatusUnavailable}}); err == nil || !strings.Contains(err.Error(), "operator action") {
		t.Fatalf("ValidateStates() error = %v, want missing operator action", err)
	}
}
