package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/peerrecord"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	runtimemanifest "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
	"google.golang.org/protobuf/proto"
)

func TestOfflineRendererProfileRunsBeforeRuntimeAppDataLookup(t *testing.T) {
	var output bytes.Buffer
	manifest := &bundleInfo{App: runtimemanifest.App{Name: "demo"}}
	handled, err := runOfflineToolProfile([]string{"host", "renderer-profile", "blender", "--json"}, t.TempDir(), manifest, &output)
	if err != nil || !handled {
		t.Fatalf("offline command = handled %t, err %v", handled, err)
	}
	if !strings.Contains(output.String(), `"profileId":"blender-5.2.2-linux-x64"`) || !strings.Contains(output.String(), `"state":"RENDERER_TOOL_PROFILE_STATE_MISSING"`) {
		t.Fatalf("offline profile JSON differs from generated proto contract: %s", output.String())
	}
}

func TestPeerScenarioPortFindsHomeRecordWithoutRepository(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	err := peerrecord.Write(home, peerrecord.Record{Scenario: "vrooli-memory", Version: "1.4.2", OwnerPID: os.Getpid(), Ports: map[string]int{"api": 48231}})
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	bundle := &bundleInfo{ProgramBindingPeers: []runtimemanifest.Peer{{Scenario: "vrooli-memory", VersionRange: ">=1.2.0 <2.0.0"}}}
	handled, err := runPeerScenarioPort([]string{"scenario", "port", "vrooli-memory", "API_PORT", "--json"}, bundle, &output)
	if !handled || err != nil {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	var got map[string]any
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["scenario"] != "vrooli-memory" || got["port_name"] != "API_PORT" || got["port"] != float64(48231) {
		t.Fatalf("peer port response = %#v", got)
	}
}

func TestEmbeddedPeerScenarioPortPrefersCompatibleFullPeerRecord(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := peerrecord.Write(home, peerrecord.Record{Scenario: "vrooli-memory", Version: "1.4.2", OwnerPID: os.Getpid(), Ports: map[string]int{"api": 48231}}); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	bundle := embeddedPeerFixture("1.3.0")
	handled, err := runPeerScenarioPort([]string{"scenario", "port", "vrooli-memory", "API_PORT"}, bundle, &output)
	if !handled || err != nil {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := strings.TrimSpace(output.String()); got != "48231" {
		t.Fatalf("external peer port = %q, want 48231", got)
	}
}

func TestEmbeddedPeerScenarioPortFallsBackWhenExternalRecordOutOfRange(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := peerrecord.Write(home, peerrecord.Record{Scenario: "vrooli-memory", Version: "2.1.0", OwnerPID: os.Getpid(), Ports: map[string]int{"api": 48231}}); err != nil {
		t.Fatal(err)
	}
	bundle := embeddedPeerFixture("1.3.0")
	handled, err := runPeerScenarioPort([]string{"scenario", "port", "vrooli-memory", "API_PORT"}, bundle, &bytes.Buffer{})
	if handled || err != nil {
		t.Fatalf("out-of-range external should fall through: handled=%t err=%v", handled, err)
	}
	args, err := translateScenarioPort(bundle, []string{"vrooli-memory", "API_PORT"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"port", "--service", "vrooli-memory--api", "--port-name", "http"}
	if !slices.Equal(args, want) {
		t.Fatalf("embedded translation = %#v, want %#v", args, want)
	}
}

func TestEmbeddedPeerScenarioPortRejectsOutOfRangeLocalCandidate(t *testing.T) {
	bundle := embeddedPeerFixture("2.1.0")
	if _, err := translateScenarioPort(bundle, []string{"vrooli-memory", "API_PORT"}); err == nil {
		t.Fatal("expected out-of-range local peer to fail closed")
	}
}

func TestEmbeddedPeerScenarioPortRequiresUniqueLocalService(t *testing.T) {
	bundle := embeddedPeerFixture("1.3.0")
	bundle.Services = append(bundle.Services, bundle.Services[1])
	if _, err := translateScenarioPort(bundle, []string{"vrooli-memory", "API_PORT"}); err == nil {
		t.Fatal("expected ambiguous local peer services to fail closed")
	}
}

func TestRootScenarioPortPreservesRuntimeTranslation(t *testing.T) {
	bundle := embeddedPeerFixture("1.3.0")
	var output bytes.Buffer
	handled, err := runPeerScenarioPort([]string{"scenario", "port", "browser-automation-studio", "API_PORT"}, bundle, &output)
	if handled || err != nil {
		t.Fatalf("root scenario should use runtime translation: handled=%t err=%v", handled, err)
	}
	args, err := translateScenarioPort(bundle, []string{"browser-automation-studio", "API_PORT"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"port", "--service", "api", "--port-name", "http"}
	if !slices.Equal(args, want) {
		t.Fatalf("root translation = %#v, want %#v", args, want)
	}
}

func embeddedPeerFixture(version string) *bundleInfo {
	return &bundleInfo{
		App:   runtimemanifest.App{Scenario: "browser-automation-studio"},
		Peers: []runtimemanifest.Peer{{Scenario: "vrooli-memory", VersionRange: ">=1.2.0 <2.0.0", EmbeddedVersion: version}},
		Services: []runtimemanifest.Service{
			{ID: "api", Type: "api-binary", Ports: &runtimemanifest.ServicePorts{Requested: []runtimemanifest.PortRequest{{Name: "http"}}}},
			{ID: "vrooli-memory--api", Type: "api-binary", Ports: &runtimemanifest.ServicePorts{Requested: []runtimemanifest.PortRequest{{Name: "http"}}}},
		},
	}
}

func TestPeerScenarioPortRejectsOutOfRangeRecordAndReportsStoppedJSON(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := peerrecord.Write(home, peerrecord.Record{Scenario: "vrooli-memory", Version: "2.1.0", OwnerPID: os.Getpid(), Ports: map[string]int{"api": 48231}}); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	bundle := &bundleInfo{ProgramBindingPeers: []runtimemanifest.Peer{{Scenario: "vrooli-memory", VersionRange: ">=1.2.0 <2.0.0"}}}
	handled, err := runPeerScenarioPort([]string{"scenario", "port", "vrooli-memory", "API_PORT", "--json"}, bundle, &output)
	if !handled || err != nil {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	var got map[string]any
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["scenario"] != "vrooli-memory" || got["port_name"] != "API_PORT" || got["error"] == "" {
		t.Fatalf("stopped peer response = %#v", got)
	}
}

func TestVersionSatisfiesRangeFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		version, constraint string
		want                bool
	}{
		{"1.4.2", ">=1.2.0 <2.0.0", true},
		{"2.1.0", ">=1.2.0 <2.0.0", false},
		{"not-a-version", ">=1.2.0", false},
		{"1.4.2", "invalid-range", false},
	} {
		if got := runtimemanifest.VersionSatisfiesRange(tc.version, tc.constraint); got != tc.want {
			t.Errorf("VersionSatisfiesRange(%q,%q)=%t want %t", tc.version, tc.constraint, got, tc.want)
		}
	}
}

func TestTranslateScenarioPortDoesNotIgnoreRequestedScenario(t *testing.T) {
	bundle := &bundleInfo{App: runtimemanifest.App{Scenario: "web-console"}, Services: []runtimemanifest.Service{{ID: "api", Type: "api-binary", Ports: &runtimemanifest.ServicePorts{Requested: []runtimemanifest.PortRequest{{Name: "http"}}}}}}
	if _, err := translateScenarioPort(bundle, []string{"other-app", "API_PORT"}); err == nil {
		t.Fatal("expected a mismatch for an unrelated scenario")
	}
}

func TestOfflineRendererProfileRejectsUnboundedArguments(t *testing.T) {
	var output bytes.Buffer
	handled, err := runOfflineToolProfile([]string{"host", "renderer-profile", "blender", "--json", "extra"}, t.TempDir(), &bundleInfo{}, &output)
	if !handled || err == nil || output.Len() != 0 {
		t.Fatalf("unexpected dispatch result handled=%t err=%v output=%q", handled, err, output.String())
	}
}

func TestOfflineRendererProfileResolverHasTwentySecondDeadline(t *testing.T) {
	var output bytes.Buffer
	var deadline time.Time
	resolve := func(ctx context.Context, _ string, _ string, _ []runtimemanifest.ToolProfilePackage) (proto.Message, error) {
		var ok bool
		deadline, ok = ctx.Deadline()
		if !ok {
			t.Fatal("profile resolver has no total deadline")
		}
		return &cliv1.BlenderToolProfile{}, nil
	}
	handled, err := runOfflineToolProfileWithResolver([]string{"host", "renderer-profile", "blender", "--json"}, t.TempDir(), &bundleInfo{}, &output, resolve)
	if err != nil || !handled {
		t.Fatalf("offline profile dispatch handled=%t err=%v", handled, err)
	}
	remaining := deadline.Sub(time.Now())
	if remaining > rendererProfileQueryTimeout || remaining <= 0 {
		t.Fatalf("unexpected total resolver deadline: %s", remaining)
	}
}
