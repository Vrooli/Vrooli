package lifecycle

import (
	"context"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
)

func listenTestPort() (net.Listener, error) { return net.Listen("tcp", "127.0.0.1:0") }

func listenerPort(listener net.Listener) string {
	return strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
}

func readinessTestRunner(home string, now *time.Time, sleeps *[]time.Duration) *Runner {
	return &Runner{
		Home: home,
		deps: lifecycleDeps{
			now: func() time.Time { return *now },
			sleep: func(duration time.Duration) {
				*sleeps = append(*sleeps, duration)
				*now = now.Add(duration)
			},
			readScenarioRecords: process.ReadScenarioRecords,
		},
	}
}

func readinessTestItem() scenario.Scenario {
	return scenario.Scenario{
		Slug: "readiness-fixture",
		Manifest: scenario.ServiceManifest{
			Ports: map[string]scenario.Port{"api": {EnvVar: "API_PORT"}},
			Components: map[string]scenario.Component{
				"api": {Run: scenario.ComponentRun{Port: "api", Argv: []string{"true"}}},
			},
		},
	}
}

func TestDerivedReadinessUsesPortWithoutLeadingGraceSleep(t *testing.T) {
	item := readinessTestItem()
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	var sleeps []time.Duration
	runner := readinessTestRunner(t.TempDir(), &now, &sleeps)
	listener, err := listenTestPort()
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if err := runner.awaitScenarioReadiness(context.Background(), item, map[string]string{"API_PORT": listenerPort(listener)}); err != nil {
		t.Fatal(err)
	}
	if len(sleeps) != 0 {
		t.Fatalf("readiness sleeps = %v, want immediate success", sleeps)
	}
}

func TestSupervisedComponentsAreExcludedFromIndependentReadiness(t *testing.T) {
	item := readinessTestItem()
	item.Manifest.Components["sidecar"] = scenario.Component{Run: scenario.ComponentRun{
		Argv:         []string{"node", "sidecar.js"},
		SupervisedBy: "api",
	}}
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	var sleeps []time.Duration
	runner := readinessTestRunner(t.TempDir(), &now, &sleeps)
	listener, err := listenTestPort()
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if err := runner.awaitScenarioReadiness(context.Background(), item, map[string]string{"API_PORT": listenerPort(listener)}); err != nil {
		t.Fatal(err)
	}
}

func TestExplicitProcessReadinessOverridesDerivedPort(t *testing.T) {
	item := readinessTestItem()
	item.Manifest.Components["api"] = scenario.Component{Run: scenario.ComponentRun{
		Port:      "api",
		Readiness: &scenario.ComponentReadiness{Type: "process_alive", TimeoutMS: 500},
	}}
	home := t.TempDir()
	if err := process.WriteScenarioRecord(home, item.Slug, "start-api", process.Record{PID: os.Getpid(), Step: "start-api", Status: "running"}); err != nil {
		t.Fatal(err)
	}
	defer process.RemoveScenarioRecord(home, item.Slug, "start-api")
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	var sleeps []time.Duration
	runner := readinessTestRunner(home, &now, &sleeps)
	if err := runner.awaitScenarioReadiness(context.Background(), item, nil); err != nil {
		t.Fatal(err)
	}
	if len(sleeps) != 0 {
		t.Fatalf("explicit readiness sleeps = %v, want immediate process success", sleeps)
	}
}

func TestDerivedReadinessHonorsFailureCeilingWithInjectedClock(t *testing.T) {
	item := readinessTestItem()
	item.Manifest.Lifecycle.Health = &scenario.HealthConfig{StartupGracePeriod: 500}
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	var sleeps []time.Duration
	runner := readinessTestRunner(t.TempDir(), &now, &sleeps)
	err := runner.awaitScenarioReadiness(context.Background(), item, map[string]string{"API_PORT": "65530"})
	if err == nil {
		t.Fatal("expected readiness failure at the ceiling")
	}
	if len(sleeps) == 0 || now.Sub(time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)) < 500*time.Millisecond {
		t.Fatalf("readiness did not consume the failure ceiling: sleeps=%v now=%s", sleeps, now)
	}
}
