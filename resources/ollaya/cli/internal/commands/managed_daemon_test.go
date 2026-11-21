package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/resources"
)

func TestManagedOllayaIdentityBindsOwnerRootAndEndpoints(t *testing.T) {
	root := t.TempDir()
	const token = "ollaya-owner-token"
	tokenHash := sha256.Sum256([]byte(token))
	state := resources.ManagedServiceState{PID: 42, Incarnation: "boot:42-start", OwnershipTokenHash: hex.EncodeToString(tokenHash[:])}
	env := map[string]string{
		managedServiceTokenEnv: token,
		"OLLAYA_MODELS":        root,
		"OLLAYA_HOST":          "127.0.0.1:21435",
		"RESOURCE_PORT_HTTP":   "11435",
	}
	identity, err := managedOllayaIdentityFromOwner(state, root, true, state.Incarnation, env, "http://localhost:11435")
	if err != nil {
		t.Fatal(err)
	}
	if identity.PID != state.PID || identity.ModelRoot != root || identity.BaseURL != "http://127.0.0.1:21435" || identity.PublicURL != "http://localhost:11435" {
		t.Fatalf("managed Ollaya identity=%+v", identity)
	}

	wrongRoot := filepath.Join(t.TempDir(), "other-models")
	if _, err := managedOllayaIdentityFromOwner(state, root, true, state.Incarnation, map[string]string{
		managedServiceTokenEnv: token,
		"OLLAYA_MODELS":        wrongRoot,
		"OLLAYA_HOST":          "127.0.0.1:21435",
		"RESOURCE_PORT_HTTP":   "11435",
	}, "http://localhost:11435"); err == nil {
		t.Fatal("accepted a managed process writing to a different model root")
	}
	for name, change := range map[string]func(map[string]string, *resources.ManagedServiceState){
		"dead process":      func(_ map[string]string, _ *resources.ManagedServiceState) {},
		"wrong incarnation": func(_ map[string]string, s *resources.ManagedServiceState) { s.Incarnation = "boot:other" },
		"wrong token": func(e map[string]string, _ *resources.ManagedServiceState) {
			e[managedServiceTokenEnv] = "foreign-token"
		},
		"remote listener": func(e map[string]string, _ *resources.ManagedServiceState) {
			e["OLLAYA_HOST"] = "192.0.2.4:21435"
		},
		"wrong endpoint": func(_ map[string]string, _ *resources.ManagedServiceState) {},
	} {
		candidateState := state
		candidateEnv := map[string]string{}
		for key, value := range env {
			candidateEnv[key] = value
		}
		change(candidateEnv, &candidateState)
		alive := true
		if name == "dead process" {
			alive = false
		}
		requested := "http://localhost:11435"
		if name == "wrong endpoint" {
			requested = "http://127.0.0.1:11436"
		}
		if _, err := managedOllayaIdentityFromOwner(candidateState, root, alive, state.Incarnation, candidateEnv, requested); err == nil {
			t.Errorf("accepted %s", name)
		}
	}
}

func TestOllayaLifecycleLockKeyMatchesControlPlaneOwner(t *testing.T) {
	if !sameLoopbackRootURL("http://127.0.0.1:11435", "http://localhost:11435") {
		t.Fatal("did not accept the default IPv4 loopback alias for the managed Ollaya endpoint")
	}
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
				t.Fatalf("managed-service lifecycle lock implementation changed; update the Ollaya shared lock binding")
			}
			if ollayaManagedLifecycleLockKey != "resource-lifecycle:ollaya" {
				t.Fatalf("Ollaya lock key=%q; want resource owner key", ollayaManagedLifecycleLockKey)
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
