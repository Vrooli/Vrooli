package platform

import (
	"strings"
	"testing"
)

// The healer must survive the saturation it exists to report on. Without these
// the autoheal service is scheduled and reclaimed like any other process, and
// on 2026-08-19 it stopped responding at exactly the moment it was needed.
func TestSystemdDefinitionProtectsTheHealerUnderPressure(t *testing.T) {
	def := renderSystemdDefinition(WatchdogDefinitionOptions{
		LoopBinary:   "/home/u/.vrooli/bin/vrooli-autoheal-loop",
		VrooliBinary: "/home/u/.vrooli/bin/vrooli",
		Home:         "/home/u",
		Root:         "/home/u/Vrooli",
	})

	for _, directive := range []string{"CPUWeight=400", "MemoryMin=128M", "OOMScoreAdjust=-500"} {
		if !strings.Contains(def, directive) {
			t.Errorf("unit is missing %s:\n%s", directive, def)
		}
	}

	// The directives belong to the service, not the install section.
	service := def[strings.Index(def, "[Service]"):]
	if idx := strings.Index(service, "[Install]"); idx >= 0 {
		service = service[:idx]
	}
	if !strings.Contains(service, "OOMScoreAdjust=-500") {
		t.Error("resource directives must sit in the [Service] section")
	}
}

// Protecting the healer must not change how it is started or stopped.
func TestSystemdDefinitionKeepsItsLifecycleContract(t *testing.T) {
	def := renderSystemdDefinition(WatchdogDefinitionOptions{
		LoopBinary: "/loop", VrooliBinary: "/vrooli", Home: "/home/u", Root: "/root",
	})
	for _, directive := range []string{"Restart=always", "RestartSec=15", "TimeoutStopSec=30", "ExecStart=/loop"} {
		if !strings.Contains(def, directive) {
			t.Errorf("unit lost %s", directive)
		}
	}
}
