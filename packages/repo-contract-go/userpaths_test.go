package repocontract

import (
	"path/filepath"
	"testing"
)

func TestUserSecretPaths(t *testing.T) {
	home := filepath.Join(string(filepath.Separator), "tmp", "vrooli-home")

	root, err := VrooliUserRoot(home)
	if err != nil {
		t.Fatalf("VrooliUserRoot: %v", err)
	}
	if want := filepath.Join(home, ".vrooli"); root != want {
		t.Fatalf("VrooliUserRoot = %q, want %q", root, want)
	}

	plaintext, err := UserPlaintextSecretsPath(home)
	if err != nil {
		t.Fatalf("UserPlaintextSecretsPath: %v", err)
	}
	if want := filepath.Join(home, ".vrooli", "secrets.json"); plaintext != want {
		t.Fatalf("UserPlaintextSecretsPath = %q, want %q", plaintext, want)
	}

	encrypted, err := UserEncryptedSecretsPath(home)
	if err != nil {
		t.Fatalf("UserEncryptedSecretsPath: %v", err)
	}
	if want := filepath.Join(home, ".vrooli", "secrets.enc.json"); encrypted != want {
		t.Fatalf("UserEncryptedSecretsPath = %q, want %q", encrypted, want)
	}

	scenarioPath, err := UserScenarioPlaintextSecretsPath(home, "scenario-to-cloud")
	if err != nil {
		t.Fatalf("UserScenarioPlaintextSecretsPath: %v", err)
	}
	if want := filepath.Join(home, ".vrooli", "scenarios", "scenario-to-cloud", "secrets.json"); scenarioPath != want {
		t.Fatalf("UserScenarioPlaintextSecretsPath = %q, want %q", scenarioPath, want)
	}
}

func TestUserScenarioPlaintextSecretsPathRejectsInvalidScenarioID(t *testing.T) {
	if _, err := UserScenarioPlaintextSecretsPath("/tmp/home", "../bad"); err == nil {
		t.Fatal("expected invalid scenario id error")
	}
}
