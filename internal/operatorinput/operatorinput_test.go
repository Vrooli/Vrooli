package operatorinput

import (
	"os"
	"path/filepath"
	"testing"
)

func TestQueueResolvesDefaultsAndRejectsInvalidChoices(t *testing.T) {
	path := filepath.Join(t.TempDir(), "operator-input.json")
	old := queuePathFn
	queuePathFn = func() (string, error) { return path, nil }
	t.Cleanup(func() { queuePathFn = old })
	if err := Replace([]Request{{ID: "profile", Kind: KindChoice, Title: "Profile", Options: []string{"starter", "custom"}, Default: "starter", Required: true}}); err != nil {
		t.Fatal(err)
	}
	if err := Resolve(nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("queue path still exists: %v", err)
	}
	if err := Replace([]Request{{ID: "profile", Kind: KindChoice, Title: "Profile", Options: []string{"starter"}, Required: true}}); err != nil {
		t.Fatal(err)
	}
	if err := Resolve([]Answer{{RequestID: "profile", Value: "other"}}); err == nil {
		t.Fatal("invalid choice unexpectedly resolved")
	}
}

func TestRemoveCapabilityReconcilesMetadataWithoutAnswers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "operator-input.json")
	old := queuePathFn
	queuePathFn = func() (string, error) { return path, nil }
	t.Cleanup(func() { queuePathFn = old })
	if err := Replace([]Request{
		{ID: "demo:sink", CapabilityID: "demo", InputID: "sink", Kind: KindPath, Title: "Destination", Required: true},
		{ID: "other:choice", CapabilityID: "other", InputID: "choice", Kind: KindChoice, Title: "Choice", Options: []string{"a"}, Required: true},
	}); err != nil {
		t.Fatal(err)
	}
	if err := RemoveCapability("demo"); err != nil {
		t.Fatal(err)
	}
	queue, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(queue.Requests) != 1 || queue.Requests[0].CapabilityID != "other" {
		t.Fatalf("remaining queue = %#v", queue.Requests)
	}
}
