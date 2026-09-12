package collectors

import (
	"context"
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestCPUCollectorCatalogResolvesEverySignal(t *testing.T) {
	data, err := NewCPUCollector().Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, signal := range cpuSignalCatalog {
		status, ok := data.Values[signal.Key+"_status"].(string)
		if !ok || status == "" {
			t.Errorf("signal %q has no resolved state", signal.Key)
			continue
		}
		if status == "measured" {
			if _, present := data.Values[signal.Key]; !present {
				t.Errorf("signal %q is measured without a value", signal.Key)
			}
		}
	}
}

func TestCPUCatalogMatchesMetricContract(t *testing.T) {
	raw, err := os.ReadFile("../../../docs/internal/METRIC_CONTRACT.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := string(raw)
	for _, signal := range cpuSignalCatalog {
		if !strings.Contains(doc, "| "+signal.Key+" | ") {
			t.Errorf("catalogued CPU signal %q is absent from METRIC_CONTRACT.md", signal.Key)
		}
		if signal.Unit == "" || signal.Tier < 1 || signal.Tier > 3 {
			t.Errorf("invalid metadata for %q", signal.Key)
		}
		for platform, backend := range map[string]string{"linux": signal.Linux, "darwin": signal.Darwin, "windows": signal.Windows, "unsupported": signal.Unsupported} {
			if backend == "" {
				t.Errorf("%s backend/refusal missing for %q", platform, signal.Key)
			}
		}
	}
}

func TestCPUCollectorHasNoPlatformBranch(t *testing.T) {
	raw, err := os.ReadFile("cpu.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "collectorOS") || strings.Contains(string(raw), "runtime.GOOS") {
		t.Fatal("cpu.go must remain platform-neutral")
	}
}

func TestCPUCatalogUsesMechanismRefusals(t *testing.T) {
	for _, s := range cpuSignalCatalog {
		for _, refusal := range []string{s.Darwin, s.Windows, s.Unsupported} {
			if strings.HasPrefix(refusal, "refuse: ") && strings.TrimSpace(strings.TrimPrefix(refusal, "refuse: ")) == runtime.GOOS {
				t.Errorf("%s refusal names only an operating system", s.Key)
			}
		}
	}
}
