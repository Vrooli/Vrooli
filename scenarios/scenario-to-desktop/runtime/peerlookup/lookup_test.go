package peerlookup

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/fsx"
	"github.com/vrooli/vrooli/internal/peerrecord"
	runtimemanifest "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
)

func TestFeatureExternalRequiresExplicitCompatibleOwner(t *testing.T) {
	for _, tc := range []struct {
		name, version, constraint string
		declared                  bool
		port, pid                 int
		want                      bool
	}{
		{"compatible", "0.0.1", ">=0.0.1 <1", true, 24567, os.Getpid(), true},
		{"blank range", "0.0.1", "", true, 24567, os.Getpid(), true},
		{"nonproviding API peer", "0.0.1", "", false, 24567, os.Getpid(), false},
		{"out of range", "0.0.1", ">=1", true, 24567, os.Getpid(), false},
		{"invalid owner version", "bad", "", true, 24567, os.Getpid(), false},
		{"missing owner version", "", "", true, 24567, os.Getpid(), false},
		{"malformed trailing comma", "0.0.1", ">=0.0.1,", true, 24567, os.Getpid(), false},
		{"malformed leading comma", "0.0.1", ",>=0.0.1", true, 24567, os.Getpid(), false},
		{"malformed comma only", "0.0.1", ", ,", true, 24567, os.Getpid(), false},
		{"invalid port", "0.0.1", "", true, 0, os.Getpid(), false},
		{"stale peer", "0.0.1", "", true, 24567, 2147483647, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			record := peerrecord.Record{Scenario: "agent-manager", OwnerPID: tc.pid, StartedAt: time.Now(), Version: "9.9.9", Provides: []string{"api"}, Ports: map[string]int{"api": tc.port}}
			if tc.declared {
				record.Features = map[string]peerrecord.Feature{"usage.live": {OwnerVersion: tc.version, Port: "api"}}
			}
			if tc.pid == 2147483647 {
				if err := writeDeadPeerFixture(home, record); err != nil {
					t.Fatal(err)
				}
			} else if err := peerrecord.Write(home, record); err != nil {
				t.Fatal(err)
			}
			port, err := FeatureExternal(&runtimemanifest.Manifest{App: runtimemanifest.App{Scenario: "web-console"}}, home, "agent-manager", "usage.live", tc.constraint)
			if tc.want && (err != nil || port != tc.port) {
				t.Fatalf("port=%d error=%v", port, err)
			}
			if !tc.want && (err == nil || port != 0) {
				t.Fatalf("ineligible feature resolved port=%d error=%v", port, err)
			}
		})
	}
}

func TestFeatureExternalPreservesConsumerEdgeRange(t *testing.T) {
	home := t.TempDir()
	record := peerrecord.Record{Scenario: "agent-manager", OwnerPID: os.Getpid(), Version: "9.9.9", Ports: map[string]int{"api": 24567}, Features: map[string]peerrecord.Feature{"usage.live": {OwnerVersion: "0.0.1", Port: "api"}}}
	if err := peerrecord.Write(home, record); err != nil {
		t.Fatal(err)
	}
	bundle := &runtimemanifest.Manifest{Peers: []runtimemanifest.Peer{{Scenario: "agent-manager", VersionRange: ">=1"}}}
	if port, err := FeatureExternal(bundle, home, "agent-manager", "usage.live", ""); err == nil || port != 0 {
		t.Fatalf("edge range bypassed: port=%d err=%v", port, err)
	}
	bundle.ProgramBindingPeers = []runtimemanifest.Peer{{Scenario: "agent-manager", VersionRange: ">=0.0.1 <1"}}
	if port, err := FeatureExternal(bundle, home, "agent-manager", "usage.live", ""); err != nil || port != 24567 {
		t.Fatalf("program binding priority lost: port=%d err=%v", port, err)
	}
}

func TestLocalInstalledOwnerUsesVerifiedVersionAndExcludesSlices(t *testing.T) {
	bundle := &runtimemanifest.Manifest{
		App:      runtimemanifest.App{Scenario: "web-console"},
		Peers:    []runtimemanifest.Peer{{Scenario: "audio-tools", BundlePolicy: "discover", VersionRange: ">=0.0.1 <1.0.0"}},
		Services: []runtimemanifest.Service{{ID: "audio-tools--api", Type: "api-binary", ComponentID: "audio-tools", ComponentVersion: "0.0.1", Ports: &runtimemanifest.ServicePorts{Requested: []runtimemanifest.PortRequest{{Name: "api"}}}}},
	}
	if service, port, err := Local(bundle, "audio-tools", "API_PORT"); err != nil || service != "audio-tools--api" || port != "api" {
		t.Fatalf("installed owner discovery: %s/%s %v", service, port, err)
	}
	for _, version := range []string{"", "1.0.0", "bad"} {
		bundle.Services[0].ComponentVersion = version
		if _, _, err := Local(bundle, "audio-tools", "API_PORT"); err == nil {
			t.Fatalf("unverified/incompatible installed version %q selected", version)
		}
	}
	bundle.Services[0].ComponentVersion = "0.0.1"
	bundle.Services[0].ComponentSlice = "usage"
	if _, _, err := Local(bundle, "audio-tools", "API_PORT"); err == nil {
		t.Fatal("bare lookup selected a slice")
	}
}

// SM11 is a call-time observation, not a persisted selected-provider row.
// Each state/event pair below is replayed through the real registry and lookup
// owners; local topology is an independent axis. The expected table deliberately
// lists admission results instead of reproducing the production predicates.
func TestSM11LookupTopologyEventMatrix(t *testing.T) {
	type observation struct {
		name                               string
		version, featureVersion            string
		published, alive, surface, feature bool
		port                               int
	}
	states := []observation{
		{name: "absent"},
		{name: "full", version: "1.2.0", featureVersion: "1.2.0", published: true, alive: true, surface: true, feature: true, port: 24567},
		{name: "owner_out_of_range", version: "2.0.0", featureVersion: "2.0.0", published: true, alive: true, surface: true, feature: true, port: 24567},
		{name: "feature_out_of_range", version: "1.2.0", featureVersion: "2.0.0", published: true, alive: true, surface: true, feature: true, port: 24567},
		{name: "surface_absent", version: "1.2.0", featureVersion: "1.2.0", published: true, alive: true, feature: true, port: 24567},
		{name: "feature_absent", version: "1.2.0", published: true, alive: true, surface: true, port: 24567},
		{name: "owner_lost", version: "1.2.0", featureVersion: "1.2.0", published: true, surface: true, feature: true, port: 24567},
		{name: "route_moved", version: "1.2.0", featureVersion: "1.2.0", published: true, alive: true, surface: true, feature: true, port: 24568},
	}
	// Bare surface discovery and explicit feature discovery have distinct
	// contracts: a feature's owner version is authoritative for that feature.
	barePorts := []int{0, 24567, 0, 24567, 0, 24567, 0, 24568}
	featurePorts := []int{0, 24567, 0, 0, 24567, 0, 0, 24568}
	topologies := []string{"none", "embedded", "installed", "slice", "ambiguous", "own"}
	for _, topology := range topologies {
		for from := range states {
			// Every observed topology accepts every publication event, including
			// repeated publication and repeated removal. These are external facts.
			for to := range states {
				t.Run(topology+"/"+states[from].name+"/observe_"+states[to].name, func(t *testing.T) {
					home := t.TempDir()
					bundle := &runtimemanifest.Manifest{App: runtimemanifest.App{Scenario: "web-console"}, Peers: []runtimemanifest.Peer{{Scenario: "audio-tools", VersionRange: ">=1 <2", EmbeddedVersion: "1.2.0"}}}
					service := runtimemanifest.Service{ID: "audio-tools--api", Type: "api-binary", Ports: &runtimemanifest.ServicePorts{Requested: []runtimemanifest.PortRequest{{Name: "api"}}}}
					switch topology {
					case "embedded":
						bundle.Services = []runtimemanifest.Service{service}
					case "installed":
						service.ComponentID = "audio-tools"
						service.ComponentVersion = "1.2.0"
						bundle.Services = []runtimemanifest.Service{service}
					case "slice":
						service.ComponentID = "usage"
						service.ComponentSlice = "usage"
						service.ComponentVersion = "1.2.0"
						bundle.Services = []runtimemanifest.Service{service}
					case "ambiguous":
						bundle.Services = []runtimemanifest.Service{service, service}
					case "own":
						bundle.App.Scenario = "audio-tools"
						service.ID = "api"
						bundle.Services = []runtimemanifest.Service{service}
					}
					observe := func(index int) {
						state := states[index]
						if err := peerrecord.Remove(home, "audio-tools", os.Getpid()); err != nil {
							t.Fatal(err)
						}
						if state.published {
							pid := os.Getpid()
							if !state.alive {
								pid = 2147483647
							}
							record := peerrecord.Record{Scenario: "audio-tools", Version: state.version, OwnerPID: pid, Ports: map[string]int{"api": state.port}, Provides: []string{"ipc"}}
							if state.surface {
								record.Provides = []string{"api"}
							}
							if state.feature {
								record.Features = map[string]peerrecord.Feature{"audio.processing": {OwnerVersion: state.featureVersion, Port: "api"}}
							}
							if !state.alive {
								if err := writeDeadPeerFixture(home, record); err != nil {
									t.Fatal(err)
								}
							} else if err := peerrecord.Write(home, record); err != nil {
								t.Fatal(err)
							}
						}
						wantBare, wantFeature := barePorts[index], featurePorts[index]
						if topology == "own" {
							wantBare, wantFeature = 0, 0
						}
						got, _ := External(bundle, home, "audio-tools", "API_PORT")
						if got != wantBare {
							t.Fatalf("%s external=%d want%d", state.name, got, wantBare)
						}
						got, _ = FeatureExternal(bundle, home, "audio-tools", "audio.processing", ">=1 <2")
						if got != wantFeature {
							t.Fatalf("%s feature=%d want%d", state.name, got, wantFeature)
						}
						id, port, err := Local(bundle, "audio-tools", "API_PORT")
						wantLocal := topology == "embedded" || topology == "installed" || topology == "own"
						if wantLocal && (err != nil || id == "" || port != "api") {
							t.Fatalf("local=%s/%s error=%v", id, port, err)
						}
						if !wantLocal && err == nil {
							t.Fatalf("ineligible %s local selected %s", topology, id)
						}
					}
					observe(from)
					observe(to)
				})
			}
		}
	}
}

// Dead-owner rows represent expired persisted state, so tests seed them as
// files directly. Production publication requires a live, identifiable PID.
func writeDeadPeerFixture(home string, record peerrecord.Record) error {
	record.SchemaVersion = peerrecord.SchemaVersion
	record.Incarnation = "linux:expired:1"
	data, err := fsx.MarshalJSON(record)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(peerrecord.Path(home, record.Scenario)), 0o700); err != nil {
		return err
	}
	return os.WriteFile(peerrecord.Path(home, record.Scenario), data, 0o600)
}

func TestSM11LookupRangeAndRouteReplay(t *testing.T) {
	home := t.TempDir()
	bundle := &runtimemanifest.Manifest{App: runtimemanifest.App{Scenario: "web-console"}, Peers: []runtimemanifest.Peer{{Scenario: "audio-tools", VersionRange: ">=1 <2"}}}
	record := peerrecord.Record{Scenario: "audio-tools", Version: "1.2.0", OwnerPID: os.Getpid(), Ports: map[string]int{"api": 24567}, Features: map[string]peerrecord.Feature{"audio.processing": {OwnerVersion: "1.2.0", Port: "api"}}}
	if err := peerrecord.Write(home, record); err != nil {
		t.Fatal(err)
	}
	assert := func(label string, want int) {
		t.Helper()
		got, _ := FeatureExternal(bundle, home, "audio-tools", "audio.processing", "")
		if got != want {
			t.Fatalf("%s got%d want%d", label, got, want)
		}
	}
	assert("published", 24567)
	record.Features["audio.processing"] = peerrecord.Feature{OwnerVersion: "2.0.0", Port: "api"}
	if err := peerrecord.Write(home, record); err != nil {
		t.Fatal(err)
	}
	assert("same port changed owner version", 0)
	bundle.ProgramBindingPeers = []runtimemanifest.Peer{{Scenario: "audio-tools", VersionRange: ">=2 <3"}}
	assert("binding overrides ordinary edge", 24567)
	record.Ports["api"] = 24568
	if err := peerrecord.Write(home, record); err != nil {
		t.Fatal(err)
	}
	assert("route moved", 24568)
	delete(record.Features, "audio.processing")
	if err := peerrecord.Write(home, record); err != nil {
		t.Fatal(err)
	}
	assert("feature withdrawn at same route", 0)
	if err := peerrecord.Remove(home, "audio-tools", os.Getpid()); err != nil {
		t.Fatal(err)
	}
	assert("full app closed", 0)
}
