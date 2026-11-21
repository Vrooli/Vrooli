package hubidentity

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/vrooli/api-core/askreader"
	"github.com/vrooli/api-core/operatordecision"
	repocontract "github.com/vrooli/repo-contract-go"
)

func TestWebhookConfigWiresTheReaderAndTheRefusalNotice(t *testing.T) {
	cfg := WebhookConfig(operatordecision.WebhookConfig{Source: "storage-manager"})
	if cfg.Reader == nil || cfg.OnRefused == nil || cfg.Source != "storage-manager" {
		t.Fatalf("cfg = %+v, want a Reader and an OnRefused for storage-manager", cfg)
	}
	if _, ok := cfg.Reader.(*askreader.Reader); !ok {
		t.Fatalf("Reader is %T, want the scenario's own ask reader", cfg.Reader)
	}
}

// A Reader or OnRefused the caller set is kept, which is how a test substitutes a Hub.
func TestWebhookConfigKeepsWhatTheCallerSet(t *testing.T) {
	reader := stubReader{}
	called := false
	cfg := WebhookConfig(operatordecision.WebhookConfig{
		Source: "storage-manager", Reader: reader, OnRefused: func(_ context.Context, _ operatordecision.Refusal) { called = true },
	})
	if cfg.Reader != operatordecision.AskReader(reader) {
		t.Fatal("the caller's Reader was replaced")
	}
	cfg.OnRefused(context.Background(), operatordecision.Refusal{})
	if !called {
		t.Fatal("the caller's OnRefused was replaced")
	}
}

func TestWebhookConfigRequiresTheScenariosSource(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("a webhook with no Source reads back as no scenario and must not be built")
		}
	}()
	WebhookConfig(operatordecision.WebhookConfig{})
}

type stubReader struct{}

func (stubReader) ReadAsk(context.Context, string) (askreader.Ask, error) {
	return askreader.Ask{}, nil
}

// unwired are the decision consumers that build a resolution webhook without
// hubidentity.WebhookConfig, and why. Nothing is added here to make the test pass:
// a new consumer wires through hubidentity.
var unwired = map[string]string{
	"scenarios/treasury/": "owned by another effort: it refuses to run without an issued ask-reader (NewDecisionAskReader) and has no refusal notice yet",
}

var newWebhookCall = regexp.MustCompile(`operatordecision\.NewWebhook\(\s*(hubidentity\.WebhookConfig\()?`)

// Every scenario that mounts a resolution webhook gets the Hub readback and the
// operator notice from one place. A consumer that built operatordecision.NewWebhook
// on its own config could silently ship without either.
func TestEveryDecisionWebhookIsWiredToTheHub(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(currentFile))))
	contract, err := repocontract.LoadDefault(repoRoot)
	if err != nil {
		t.Fatalf("load repo contract: %v", err)
	}
	wired, exempt := 0, 0
	err = contract.WalkSource(repoRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "templates" {
				return fs.SkipDir
			}
			return nil
		}
		rel, relErr := filepath.Rel(repoRoot, path)
		if relErr != nil || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || !strings.HasPrefix(filepath.ToSlash(rel), "scenarios/") {
			return nil
		}
		source, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, match := range newWebhookCall.FindAllSubmatch(source, -1) {
			rel = filepath.ToSlash(rel)
			switch {
			case len(match[1]) > 0:
				wired++
			case exemption(rel) != "":
				exempt++
			default:
				t.Errorf("%s builds operatordecision.NewWebhook without hubidentity.WebhookConfig, so it neither confirms answers with the Hub nor tells the operator when one is refused; wrap its config", rel)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// The six consumers wired today. A scan that finds fewer is looking in the wrong place.
	if wired < 6 {
		t.Fatalf("found %d wired webhooks (%d exempt), want at least 6: the scan is not seeing the scenarios", wired, exempt)
	}
}

func exemption(rel string) string {
	for prefix, why := range unwired {
		if strings.HasPrefix(rel, prefix) {
			return why
		}
	}
	return ""
}
