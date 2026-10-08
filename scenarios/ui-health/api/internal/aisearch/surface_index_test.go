package aisearch

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"testing"
)

type failingDiscovery struct {
	scenarios []string
	failing   map[string]bool
}

func (d failingDiscovery) ListScenarios(context.Context) ([]string, error) { return d.scenarios, nil }

func (d failingDiscovery) Discover(_ context.Context, scenario string) ([]SurfaceRecord, error) {
	if d.failing[scenario] {
		return nil, errors.New("react-component-library unavailable")
	}
	return []SurfaceRecord{{Scenario: scenario, FilePath: "ui/src/App.tsx"}}, nil
}

// When the component library is down every scenario fails the same way. A sync
// must write one summary line, not one line per scenario.
func TestLoadAllSummarizesDiscoveryFailuresInOneLine(t *testing.T) {
	var buf bytes.Buffer
	previous, flags := log.Writer(), log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() { log.SetOutput(previous); log.SetFlags(flags) })

	discovery := failingDiscovery{failing: map[string]bool{}}
	for i := 0; i < 40; i++ {
		name := fmt.Sprintf("scenario-%02d", i)
		discovery.scenarios = append(discovery.scenarios, name)
		if i%10 != 0 {
			discovery.failing[name] = true
		}
	}
	docs, err := newSurfaceSource(discovery).LoadAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 4 {
		t.Fatalf("docs = %d, want the 4 scenarios that discovered", len(docs))
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("log lines = %d, want 1:\n%s", len(lines), buf.String())
	}
	for _, fragment := range []string{"skipped 36 of 40 scenarios", "and 31 more", "react-component-library unavailable"} {
		if !strings.Contains(lines[0], fragment) {
			t.Fatalf("summary %q does not contain %q", lines[0], fragment)
		}
	}
}

func TestLoadAllLogsNothingWhenEveryScenarioDiscovers(t *testing.T) {
	var buf bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(previous) })

	if _, err := newSurfaceSource(failingDiscovery{scenarios: []string{"a", "b"}}).LoadAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Fatalf("unexpected log output: %q", buf.String())
	}
}
