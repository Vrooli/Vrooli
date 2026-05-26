package repocontract

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadLiveContract(t *testing.T) {
	root := fixtureRoot(t)
	contract, err := LoadDefault(root)
	if err != nil {
		t.Fatalf("LoadDefault() error = %v", err)
	}

	if contract.Schema() != "schemas/repo-contract.schema.json" {
		t.Fatalf("Schema() = %q", contract.Schema())
	}
	if contract.Version() != "1.1.0" {
		t.Fatalf("Version() = %q", contract.Version())
	}
	if got := contract.Layout().ScenarioDir; got != "scenarios" {
		t.Fatalf("Layout().ScenarioDir = %q", got)
	}
	if got := contract.EnvironmentVariables()["repo_root"]; got != "VROOLI_ROOT" {
		t.Fatalf("repo_root env = %q", got)
	}
}

func TestLoadValidatesInputAndReadFailures(t *testing.T) {
	t.Run("empty path", func(t *testing.T) {
		_, err := Load(" ")
		assertErrorKind(t, err, ErrInvalidInput)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := Load("/definitely/missing/repo-contract.json")
		assertErrorKind(t, err, ErrNotFound)
	})

	t.Run("invalid json", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "repo-contract.json")
		if err := os.WriteFile(path, []byte("{not-json"), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
		_, err := Load(path)
		assertErrorKind(t, err, ErrInvalidContract)
	})
}

func TestLoadSupportsFilesystemReadSeam(t *testing.T) {
	oldReadFile := readFile
	readFile = func(string) ([]byte, error) {
		return nil, os.ErrPermission
	}
	t.Cleanup(func() { readFile = oldReadFile })

	_, err := Load("/tmp/fake.json")
	assertErrorKind(t, err, ErrNotFound)
}

func TestLoadRejectsInvalidFixtures(t *testing.T) {
	tests := []struct {
		name string
		file string
		kind ErrorKind
	}{
		{name: "unsupported version", file: "invalid-unsupported-version.json", kind: ErrUnsupportedVersion},
		{name: "absolute path", file: "invalid-absolute-path.json", kind: ErrInvalidContract},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(filepath.Join("testdata", tt.file))
			assertErrorKind(t, err, tt.kind)
		})
	}
}

func TestLoadDefaultValidatesRepoRoot(t *testing.T) {
	_, err := LoadDefault(" ")
	assertErrorKind(t, err, ErrInvalidInput)
}

func TestValidateContractDocHappyPath(t *testing.T) {
	if err := validateContractDoc(validContractDoc(t)); err != nil {
		t.Fatalf("validateContractDoc() error = %v", err)
	}
}

func TestValidateContractDocRejectsSemanticDrift(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*contractDoc)
		kind   ErrorKind
	}{
		{
			name: "unexpected schema",
			mutate: func(doc *contractDoc) {
				doc.Schema = "schemas/other.schema.json"
			},
			kind: ErrInvalidContract,
		},
		{
			name: "invalid platform mode",
			mutate: func(doc *contractDoc) {
				doc.Platform.Mode = "legacy"
			},
			kind: ErrInvalidContract,
		},
		{
			name: "legacy bash supported",
			mutate: func(doc *contractDoc) {
				doc.Platform.LegacyProjectBashSupported = true
			},
			kind: ErrInvalidContract,
		},
		{
			name: "invalid env var",
			mutate: func(doc *contractDoc) {
				doc.Environment.Variables = map[string]string{"repo_root": "vrooli_root"}
			},
			kind: ErrInvalidContract,
		},
		{
			name: "missing sandbox scopes",
			mutate: func(doc *contractDoc) {
				doc.Sandbox.FullRepoScopes = []string{"."}
			},
			kind: ErrInvalidContract,
		},
		{
			name: "empty profiles",
			mutate: func(doc *contractDoc) {
				doc.Profiles = nil
			},
			kind: ErrInvalidContract,
		},
		{
			name: "duplicate include",
			mutate: func(doc *contractDoc) {
				profile := doc.Profiles["mini_vrooli_bundle"]
				profile.Include = []string{"packages", "packages"}
				doc.Profiles["mini_vrooli_bundle"] = profile
			},
			kind: ErrInvalidContract,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := validContractDoc(t)
			tt.mutate(&doc)
			err := validateContractDoc(doc)
			assertErrorKind(t, err, tt.kind)
		})
	}
}

func TestValidateVersion(t *testing.T) {
	tests := []struct {
		version string
		wantErr ErrorKind
	}{
		{version: "1.2.3"},
		{version: "1.0", wantErr: ErrInvalidContract},
		{version: "v1.0.0", wantErr: ErrInvalidContract},
		{version: "2.0.0", wantErr: ErrUnsupportedVersion},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			err := validateVersion(tt.version)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validateVersion() error = %v", err)
				}
				return
			}
			assertErrorKind(t, err, tt.wantErr)
		})
	}
}

func TestValidateHelpers(t *testing.T) {
	t.Run("validate slash path", func(t *testing.T) {
		if err := validateSlashPath("field", "scenarios/test"); err != nil {
			t.Fatalf("validateSlashPath() error = %v", err)
		}
		assertErrorKind(t, validateSlashPath("field", `scenarios\test`), ErrInvalidContract)
		assertErrorKind(t, validateSlashPath("field", "/scenarios/test"), ErrInvalidContract)
		assertErrorKind(t, validateSlashPath("field", "scenarios/../test"), ErrInvalidContract)
	})

	t.Run("validate slash paths duplicate", func(t *testing.T) {
		assertErrorKind(t, validateSlashPaths("field", []string{"a", "a"}), ErrInvalidContract)
	})

	t.Run("validate env var name", func(t *testing.T) {
		if err := validateEnvVarName("field", "VROOLI_ROOT"); err != nil {
			t.Fatalf("validateEnvVarName() error = %v", err)
		}
		assertErrorKind(t, validateEnvVarName("field", "vrooli_root"), ErrInvalidContract)
		assertErrorKind(t, validateEnvVarName("field", "VROOLI-ROOT"), ErrInvalidContract)
	})

	t.Run("clean identifier", func(t *testing.T) {
		got, err := cleanIdentifier("  test-genie ")
		if err != nil || got != "test-genie" {
			t.Fatalf("cleanIdentifier() = %q, %v", got, err)
		}
		assertErrorKind(t, func() error {
			_, err := cleanIdentifier("../bad")
			return err
		}(), ErrInvalidInput)
	})
}

func TestDeepCopyContractDoc(t *testing.T) {
	doc := validContractDoc(t)
	copy := deepCopyContractDoc(doc)

	copy.Root.Markers.RequiredDirs[0] = "BROKEN"
	copy.Scenario.WellKnownPaths["service"] = "BROKEN"
	copy.Environment.Variables["repo_root"] = "BROKEN"
	profile := copy.Profiles["mini_vrooli_bundle"]
	profile.Include[0] = "BROKEN"
	copy.Profiles["mini_vrooli_bundle"] = profile

	if doc.Root.Markers.RequiredDirs[0] == "BROKEN" {
		t.Fatal("deepCopyContractDoc() did not isolate RequiredDirs")
	}
	if doc.Scenario.WellKnownPaths["service"] == "BROKEN" {
		t.Fatal("deepCopyContractDoc() did not isolate WellKnownPaths")
	}
	if doc.Environment.Variables["repo_root"] == "BROKEN" {
		t.Fatal("deepCopyContractDoc() did not isolate env vars")
	}
	if doc.Profiles["mini_vrooli_bundle"].Include[0] == "BROKEN" {
		t.Fatal("deepCopyContractDoc() did not isolate profile include")
	}
}

func TestErrorFormattingAndUnwrap(t *testing.T) {
	var nilErr *Error
	if got := nilErr.Error(); got != "<nil>" {
		t.Fatalf("nil Error(). = %q", got)
	}

	base := errors.New("boom")
	err := (&Error{Kind: ErrInvalidInput, Message: "bad", Details: "x", Err: base})
	if !strings.Contains(err.Error(), "repo-contract invalid_input: bad (x)") {
		t.Fatalf("Error() = %q", err.Error())
	}
	if !errors.Is(err, base) {
		t.Fatal("Unwrap() did not expose wrapped error")
	}
}

func assertErrorKind(t *testing.T, err error, want ErrorKind) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error kind %q, got nil", want)
	}
	var target *Error
	if !errors.As(err, &target) {
		t.Fatalf("error type = %T, want *Error", err)
	}
	if target.Kind != want {
		t.Fatalf("error kind = %q, want %q", target.Kind, want)
	}
}
