package ensure

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/resources"
)

func TestManagedLifecycleLockKeyMatchesControlPlaneOwner(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		ownerPath := filepath.Join(root, "internal", "resources", "lifecycle_lock.go")
		if _, err := os.Stat(ownerPath); err == nil {
			owner, err := os.ReadFile(ownerPath)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(owner), `artifactlock.AcquireContext(ctx, "resource-lifecycle:"+resource)`) {
				t.Fatalf("managed-service lifecycle lock implementation changed; update the Ollama shared lock binding")
			}
			return
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Fatal("could not locate internal/resources/lifecycle_lock.go from test source")
		}
		root = parent
	}
}

func TestManagedOllamaIdentityBindsProcessRootAndLoopbackListener(t *testing.T) {
	root := t.TempDir()
	const token = "managed-owner-token"
	tokenHash := sha256.Sum256([]byte(token))
	state := resources.ManagedServiceState{
		PID:                42,
		Incarnation:        "boot:42-start",
		OwnershipTokenHash: hex.EncodeToString(tokenHash[:]),
	}
	env := map[string]string{
		managedServiceTokenEnv: token,
		"OLLAMA_MODELS":        root,
		"OLLAMA_HOST":          "127.0.0.1:21434",
	}
	identity, err := managedOllamaIdentityFromOwner(state, root, true, state.Incarnation, env)
	if err != nil {
		t.Fatal(err)
	}
	if identity.PID != state.PID || identity.ModelRoot != root || identity.BaseURL != "http://127.0.0.1:21434" {
		t.Fatalf("managed Ollama identity=%+v", identity)
	}

	wrongRoot := filepath.Join(t.TempDir(), "other-models")
	if _, err := managedOllamaIdentityFromOwner(state, root, true, state.Incarnation, map[string]string{
		managedServiceTokenEnv: token,
		"OLLAMA_MODELS":        wrongRoot,
		"OLLAMA_HOST":          "127.0.0.1:21434",
	}); err == nil {
		t.Fatal("accepted a managed process writing to a different model root")
	}
	for name, change := range map[string]func(map[string]string, *resources.ManagedServiceState){
		"dead process":      func(_ map[string]string, _ *resources.ManagedServiceState) {},
		"wrong incarnation": func(_ map[string]string, s *resources.ManagedServiceState) { s.Incarnation = "boot:other" },
		"wrong token": func(e map[string]string, _ *resources.ManagedServiceState) {
			e[managedServiceTokenEnv] = "foreign-token"
		},
		"remote listener": func(e map[string]string, _ *resources.ManagedServiceState) { e["OLLAMA_HOST"] = "192.0.2.4:21434" },
	} {
		candidateState := state
		candidateEnv := map[string]string{}
		for key, value := range env {
			candidateEnv[key] = value
		}
		change(candidateEnv, &candidateState)
		alive := name != "dead process"
		if _, err := managedOllamaIdentityFromOwner(candidateState, root, alive, state.Incarnation, candidateEnv); err == nil {
			t.Errorf("accepted %s", name)
		}
	}
}
