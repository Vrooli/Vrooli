package validate

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestGroupName(t *testing.T) {
	if GroupName != "validate" {
		t.Fatalf("group name = %q", GroupName)
	}
}

func TestHasContextualFileRootPickUsesASTReceiverAndContext(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		want   bool
	}{
		{name: "typed parameter Pick", source: `package p
import fr "github.com/vrooli/api-core/filerouting"
func use(ctx context.Context, roots *fr.RoutedRoots) { roots.Pick(ctx, storage.ClassData) }`, want: true},
		{name: "aliased import and routed receiver field", source: `package p
import routed "github.com/vrooli/api-core/filerouting"
type blobStore struct { renamedRoots *routed.RoutedRoots }
func (s *blobStore) use(ctx context.Context) { s.renamedRoots . PickRequired ( ctx , storage.ClassData ) }`, want: true},
		{name: "constructor-derived local", source: `package p
import routed "github.com/vrooli/api-core/filerouting"
func use(ctx context.Context) { roots := routed.New(paths); roots.PickRequired(ctx, storage.ClassData) }`, want: true},
		{name: "context-free selector", source: `package p
import fr "github.com/vrooli/api-core/filerouting"
func use(roots *fr.RoutedRoots) { roots.Pick(storage.ClassData) }`, want: false},
		{name: "nil context", source: `package p
import fr "github.com/vrooli/api-core/filerouting"
func use(roots *fr.RoutedRoots) { roots.PickRequired(nil, storage.ClassData) }`, want: false},
		{name: "unrelated selector beside routed type", source: `package p
import fr "github.com/vrooli/api-core/filerouting"
type wrapper struct { roots *fr.RoutedRoots }
func use(picker *other.Picker, ctx context.Context) { picker.Pick(ctx, storage.ClassData) }`, want: false},
		{name: "comment does not count", source: `package p
import fr "github.com/vrooli/api-core/filerouting"
type wrapper struct { roots *fr.RoutedRoots }
// roots.PickRequired(ctx, storage.ClassData)
func use(ctx context.Context) { _ = ctx }`, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasContextualFileRootPick([]sourceFile{{dir: "api", content: tc.source}}); got != tc.want {
				t.Fatalf("hasContextualFileRootPick(%q) = %t, want %t", tc.source, got, tc.want)
			}
		})
	}
}

func TestHasContextualFileRootPickScopesTypesAndConstructorResults(t *testing.T) {
	files := []sourceFile{
		{dir: "api/one", content: `package one
import fr "github.com/vrooli/api-core/filerouting"
type store struct { roots *fr.RoutedRoots }
`},
		{dir: "api/two", content: `package two
type store struct { roots *other.RoutedRoots }
func (s *store) use(ctx context.Context) { s.roots.Pick(ctx, storage.ClassData) }
`},
	}
	if hasContextualFileRootPick(files) {
		t.Fatal("same type name in another package must not donate routed fields")
	}

	for _, source := range []string{
		`func use(ctx context.Context) { roots, other := fr.New(paths), customRoot{}; other.Pick(ctx, storage.ClassData) }`,
		`func use(ctx context.Context) { roots, other := customRoot{}, fr.New(paths); roots.Pick(ctx, storage.ClassData) }`,
	} {
		if hasContextualFileRootPick([]sourceFile{{dir: "api", content: `package p
import fr "github.com/vrooli/api-core/filerouting"
` + source}}) {
			t.Fatalf("unrelated constructor-result variable was accepted: %s", source)
		}
	}
}

func TestRegisterKeepsAPIFreeIsolationProofLocal(t *testing.T) {
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:           "storage-manager-test",
		Version:        "test",
		Description:    "test",
		DefaultAPIBase: "http://127.0.0.1:1",
		AllowAnonymous: true,
	})
	if err != nil {
		t.Fatalf("NewStandardScenarioApp() error: %v", err)
	}
	manifest := []byte(`{"name":"storage-manager","groups":[{"name":"validate","description":"Validate storage","commands":[{"name":"scenario","description":"Validate a scenario","binding":{"kind":"connect-rpc","service":"ScenarioValidationService","method":"ValidateScenario"},"governance":{"effect":"read","run_eligible":true,"permissions":["network:internal"]}}]}]}`)
	group, err := Register(core, manifest)
	if err != nil {
		t.Fatalf("Register() error: %v", err)
	}
	if group.NeedsAPI {
		t.Fatal("validate group must not force API preflight for local commands")
	}
	needsAPI := map[string]bool{}
	for _, command := range group.Subcommands {
		needsAPI[command.Name] = command.NeedsAPI
	}
	if !needsAPI["scenario"] || !needsAPI["resource"] || !needsAPI["tool"] || !needsAPI["safeguard"] || !needsAPI["fleet"] {
		t.Fatalf("API-backed commands lost their API requirement: %#v", needsAPI)
	}
	if needsAPI["prove-isolation"] {
		t.Fatal("prove-isolation must not require the API")
	}
}
