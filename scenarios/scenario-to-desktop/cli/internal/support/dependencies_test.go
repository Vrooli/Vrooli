package support

import "testing"

func TestDependenciesWithoutScenarioAppFailClosed(t *testing.T) {
	deps := Dependencies{}
	if deps.ScenarioApp() != nil {
		t.Fatal("ScenarioApp() should be nil without configured core")
	}

	for _, call := range []struct {
		name string
		run  func() error
	}{
		{"Get", func() error { _, err := deps.Get("/status", nil); return err }},
		{"Request", func() error { _, err := deps.Request("POST", "/status", nil, nil); return err }},
		{"GetRoot", func() error { _, err := deps.GetRoot("/health", nil); return err }},
		{"RequestRoot", func() error { _, err := deps.RequestRoot("GET", "/health", nil, nil); return err }},
	} {
		t.Run(call.name, func(t *testing.T) {
			if err := call.run(); err == nil || err.Error() != "scenario app not configured" {
				t.Fatalf("expected configuration error, got %v", err)
			}
		})
	}
}
