package credentialspec

import (
	"strings"
	"testing"
)

// ResolvedField has one job that matters: every caller must agree. A
// descriptor that resolved to "value" in the resolver and "" in the CLI would
// read and write two different store keys for one declaration, which is the
// shape of bug that loses a credential without any error.
func TestResolvedFieldIsTotal(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		field string
		want  string
	}{
		{name: "explicit field", field: "api-key", want: "api-key"},
		{name: "omitted field", field: "", want: DefaultField},
		{name: "whitespace-only field", field: "   ", want: DefaultField},
		{name: "padded field", field: "  api-key  ", want: "api-key"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if got := (Descriptor{Field: testCase.field}).ResolvedField(); got != testCase.want {
				t.Fatalf("ResolvedField() = %q, want %q", got, testCase.want)
			}
		})
	}
}

// Injectable decides whether a value enters a process environment at all, so
// whitespace must not be mistaken for a variable name.
func TestInjectableRequiresARealVariableName(t *testing.T) {
	if (Descriptor{Env: "  "}).Injectable() {
		t.Fatal("a whitespace-only env must not count as an injection target")
	}
	if !(Descriptor{Env: "OPENROUTER_API_KEY"}).Injectable() {
		t.Fatal("a named env must count as an injection target")
	}
}

func TestValidateRejectsOnlyWhatAnOperatorCannotFixAtRuntime(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		descriptors []Descriptor
		wantErr     string
	}{
		{
			name:        "no logical id",
			descriptors: []Descriptor{{Field: "api-key"}},
			wantErr:     "no logical_id",
		},
		{
			name:        "field carrying a path separator",
			descriptors: []Descriptor{{LogicalID: "vrooli/openrouter", Field: "a/b"}},
			wantErr:     "path separator",
		},
		{
			name: "two descriptors over one env",
			descriptors: []Descriptor{
				{LogicalID: "vrooli/openrouter", Field: "api-key", Env: "KEY"},
				{LogicalID: "vrooli/other", Field: "api-key", Env: "KEY"},
			},
			wantErr: "twice",
		},
		{
			// Reachable only because env is optional: with no env to collide
			// on, the store key is the last thing left that can conflict.
			name: "two descriptors over one store key",
			descriptors: []Descriptor{
				{LogicalID: "vrooli/openrouter", Field: "api-key"},
				{LogicalID: "vrooli/openrouter", Field: "api-key"},
			},
			wantErr: "twice",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			err := Declaration{Descriptors: testCase.descriptors}.Validate("resource openrouter")
			if err == nil {
				t.Fatalf("Validate() = nil, want an error mentioning %q", testCase.wantErr)
			}
			if !strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("error = %v, want it to mention %q", err, testCase.wantErr)
			}
		})
	}
}

// The default field is applied before the duplicate check, so an omitted field
// and an explicit "value" are the same store key and must collide.
func TestValidateTreatsAnOmittedFieldAsTheDefaultWhenDeduplicating(t *testing.T) {
	err := Declaration{Descriptors: []Descriptor{
		{LogicalID: "vrooli/openrouter"},
		{LogicalID: "vrooli/openrouter", Field: DefaultField},
	}}.Validate("resource openrouter")
	if err == nil {
		t.Fatal("an omitted field and an explicit default address one key and must collide")
	}
}

func TestValidateAcceptsAMixOfInjectedAndDirectlyResolvedCredentials(t *testing.T) {
	declaration := Declaration{Descriptors: []Descriptor{
		{LogicalID: "vrooli/postgres", Field: "password", Env: "POSTGRES_PASSWORD"},
		{LogicalID: "vrooli/tunnel-manager", Field: "cloudflare-api-token"},
	}}
	if err := declaration.Validate("scenario tunnel-manager"); err != nil {
		t.Fatalf("Validate() = %v, want a mixed declaration to be legal", err)
	}
	if injectable := declaration.Injectable(); len(injectable) != 1 || injectable[0].Env != "POSTGRES_PASSWORD" {
		t.Fatalf("Injectable() = %+v, want only the descriptor naming a variable", injectable)
	}
	if all := declaration.All(); len(all) != 2 {
		t.Fatalf("All() = %+v, want both descriptors — a directly resolved credential is still declared", all)
	}
}

func TestDescriptorProvisioningValidation(t *testing.T) {
	valid := Descriptor{LogicalID: "vrooli/test", Field: "derived", Provisioning: "derived", DerivedFrom: "token"}
	if err := (Declaration{Descriptors: []Descriptor{valid}}).Validate("test"); err != nil {
		t.Fatalf("derived descriptor rejected: %v", err)
	}
	for _, descriptor := range []Descriptor{
		{LogicalID: "vrooli/test", Field: "derived", Provisioning: "unknown"},
		{LogicalID: "vrooli/test", Field: "derived", Provisioning: "derived"},
	} {
		if err := (Declaration{Descriptors: []Descriptor{descriptor}}).Validate("test"); err == nil {
			t.Fatalf("descriptor %+v unexpectedly validated", descriptor)
		}
	}
}
