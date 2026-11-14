package resources

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

)

func TestManagedUpdateCredentialBindingRequiresCurrentUnchangedAuthority(t *testing.T) {
	for _, mode := range []string{"matches", "different-file", "rotated-authority", "missing-authority", "missing-file", "replaced-file"} {
		t.Run(mode, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "credential")
			value := "private-fixture-value"
			fileValue := value + "\n"
			if mode == "different-file" {
				fileValue = "different-private-value\n"
			}
			if mode != "missing-file" {
				if err := os.WriteFile(path, []byte(fileValue), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			calls := 0
			resolve := func() (string, error) {
				calls++
				if mode == "missing-authority" {
					return "", errors.New(value)
				}
				if mode == "rotated-authority" && calls > 1 {
					return "rotated-private-value", nil
				}
				if mode == "replaced-file" && calls > 1 {
					if err := os.Rename(path, path+".old"); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(path, []byte(value+"\n"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				return value, nil
			}
			err := verifyManagedUpdateCredentialBinding(path, resolve)
			if mode == "matches" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil {
				t.Fatal("unstable or missing credential binding was acknowledged")
			}
			if strings.Contains(err.Error(), value) || strings.Contains(err.Error(), "rotated-private-value") {
				t.Fatal("credential value leaked into rejection diagnostics")
			}
		})
	}
}

func TestManagedUpdateCurrentHostLegacyObservationProbe(t *testing.T) {
	if os.Getenv("VROOLI_MANAGED_UPDATE_LEGACY_OBSERVE_PROBE") != "1" {
		t.Skip("set VROOLI_MANAGED_UPDATE_LEGACY_OBSERVE_PROBE=1 for non-serving-mutating current connector observation")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	c := NewController(projectRoot(t), home)
	sup, _, err := managedServiceSupervisorFor("cloudflared")
	if err != nil {
		t.Fatal(err)
	}
	before, running, err := sup.Status()
	if err != nil || !running {
		t.Fatalf("serving connector unavailable: %v", err)
	}
	record, err := os.ReadFile(sup.statePath)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	observation, err := c.ObserveManagedServiceLegacyLaunch(ctx, "cloudflared")
	if err != nil {
		t.Fatal(err)
	}
	after, running, err := sup.Status()
	if err != nil || !running || !reflect.DeepEqual(before, after) {
		t.Fatal("legacy observation changed serving state")
	}
	finalRecord, err := os.ReadFile(sup.statePath)
	if err != nil || !reflect.DeepEqual(record, finalRecord) {
		t.Fatal("legacy observation changed supervisor record bytes")
	}
	if !observation.ExactArgumentsVerified || !observation.DeclaredEnvironmentVerified || !observation.CredentialBindingsVerified || !observation.RequiresIsolatedQualification {
		t.Fatal("legacy observation omitted its evidence or qualification boundary")
	}
	if out := os.Getenv("VROOLI_MANAGED_UPDATE_LEGACY_OBSERVE_REPORT"); out != "" {
		data, err := json.MarshalIndent(observation, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := writeResourceOwnedFileDurable(out, append(data, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("exact native launch and current credential-file authority binding verified for PID%d; serving record unchanged; isolated rollback qualification still required", before.PID)
}
