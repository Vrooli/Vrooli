package scenarioenv

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

type fakeStore struct {
	instance scenarioruntime.Instance
	claims   []scenarioruntime.PortClaim
}

func (s *fakeStore) ListInstances(context.Context, scenarioruntime.InstanceFilter) ([]scenarioruntime.Instance, error) {
	if s.instance.InstanceID == "" {
		return nil, nil
	}
	return []scenarioruntime.Instance{s.instance}, nil
}

func (s *fakeStore) ListPortClaims(context.Context, scenarioruntime.PortClaimFilter) ([]scenarioruntime.PortClaim, error) {
	return s.claims, nil
}
func (s *fakeStore) Close() error { return nil }

func bindingManifest(policy string) scenario.ServiceManifest {
	return scenario.ServiceManifest{Dependencies: scenario.Dependencies{Scenarios: map[string]scenario.Dependency{
		"authority": {Bindings: []scenario.Binding{{EnvVar: "AUTHORITY_URL", Form: "http_base_url", Port: "api", WhenUnavailable: policy}}},
	}}}
}

func TestResolveProjectsRuntimePortAndHonorsUnavailablePolicy(t *testing.T) {
	store := &fakeStore{
		instance: scenarioruntime.Instance{InstanceID: "inst-1", Scenario: "authority", Variant: "live", Status: scenarioruntime.StatusRunning},
		claims:   []scenarioruntime.PortClaim{{InstanceID: "inst-1", PortName: "api", Port: 18444, Status: scenarioruntime.ClaimStatusBound}},
	}
	open := func(context.Context, string) (peerStore, error) { return store, nil }
	got, err := resolve(context.Background(), t.TempDir(), bindingManifest("fail"), map[string]string{}, open)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got["AUTHORITY_URL"] != "http://127.0.0.1:18444" {
		t.Fatalf("binding = %q", got["AUTHORITY_URL"])
	}

	empty := &fakeStore{}
	open = func(context.Context, string) (peerStore, error) { return empty, nil }
	got, err = resolve(context.Background(), t.TempDir(), bindingManifest("omit"), map[string]string{}, open)
	if err != nil || len(got) != 0 {
		t.Fatalf("omit = %#v, %v", got, err)
	}
	_, err = resolve(context.Background(), t.TempDir(), bindingManifest("fail"), map[string]string{}, open)
	if err == nil || !strings.Contains(err.Error(), "AUTHORITY_URL") {
		t.Fatalf("fail error = %v", err)
	}
}

func TestResolveRejectsResourcePeerCollision(t *testing.T) {
	_, err := resolve(context.Background(), t.TempDir(), bindingManifest("omit"), map[string]string{"AUTHORITY_URL": "resource"}, func(context.Context, string) (peerStore, error) {
		return &fakeStore{}, nil
	})
	if err == nil || !strings.Contains(err.Error(), "collision") {
		t.Fatalf("collision error = %v", err)
	}
}

func TestPeerRecordPermissionsAndStaleness(t *testing.T) {
	home := t.TempDir()
	record := PeerRecord{
		Scenario:  "authority",
		Instance:  "live",
		Tier:      1,
		OwnerPID:  os.Getpid(),
		StartedAt: time.Now().UTC(),
		Ports:     map[string]int{"api": 18444},
	}
	if err := Write(home, record); err != nil {
		t.Fatalf("Write: %v", err)
	}
	info, err := os.Stat(filepath.Join(home, ".vrooli", "peers", "authority.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
	if _, err := Read(home, "authority"); err != nil {
		t.Fatalf("Read live: %v", err)
	}
	record.OwnerPID = 1 << 30
	if err := Write(home, record); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(home, "authority"); !os.IsNotExist(err) {
		t.Fatalf("Read stale error = %v", err)
	}
	if err := Remove(home, "authority"); err != nil {
		t.Fatal(err)
	}
}
