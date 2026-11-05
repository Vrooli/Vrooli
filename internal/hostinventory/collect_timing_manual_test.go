package hostinventory

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestLiveCollectStageTimingsOnThisHost times each stage of the full Collect
// pipeline on the real host. It is evidence capture, not an assertion: it is
// skipped unless VROOLI_HOSTINVENTORY_LIVE is set.
func TestLiveCollectStageTimingsOnThisHost(t *testing.T) {
	if os.Getenv("VROOLI_HOSTINVENTORY_LIVE") == "" {
		t.Skip("set VROOLI_HOSTINVENTORY_LIVE to capture live stage timings")
	}
	ctx := context.Background()
	c := SystemCollector().withDefaults()
	now := c.Clock.Now()
	snap := Snapshot{
		OS:              c.GOOS,
		Arch:            c.GOARCH,
		RuntimeTools:    map[string]Tool{},
		ProbeStatuses:   map[string]string{},
		FieldProvenance: map[string]Provenance{},
	}
	stages := []struct {
		name string
		run  func()
	}{
		{"platformFacts", func() { c.collectPlatformFacts(ctx, &snap, now) }},
		{"memory", func() { c.collectMemory(&snap, now) }},
		{"load", func() { c.collectLoad(&snap, now) }},
		{"devices", func() { c.collectDevices(ctx, &snap, now) }},
		{"nvidiaGPUs", func() { c.collectNvidiaGPUs(ctx, &snap, now) }},
		{"linkNvidiaDevices", func() { c.linkNvidiaDevices(ctx, &snap, now) }},
		{"darwinGPUs", func() { c.collectDarwinGPUs(ctx, &snap, now) }},
		{"windowsGPUs", func() { c.collectWindowsGPUs(ctx, &snap, now) }},
		{"dockerGPU", func() { c.collectDockerGPU(ctx, &snap, now) }},
		{"nvidiaNodeAccess", func() { c.collectNvidiaNodeAccess(&snap, now) }},
		{"rocm", func() { c.collectROCm(ctx, &snap, now) }},
		{"vulkanICDs", func() { c.collectVulkanICDs(&snap, now) }},
		{"appleSiliconGPU", func() { c.collectAppleSiliconGPU(&snap, now) }},
		{"appleToolchain", func() { c.collectAppleToolchain(ctx, &snap, now) }},
		{"androidToolchain", func() { c.collectAndroidToolchain(ctx, &snap, now) }},
		{"desktopToolchain", func() { c.collectDesktopToolchain(ctx, &snap, now) }},
	}
	total := time.Duration(0)
	for _, stage := range stages {
		started := time.Now()
		stage.run()
		elapsed := time.Since(started)
		total += elapsed
		t.Logf("%-18s %s", stage.name, elapsed)
	}
	t.Logf("%-18s %s", "TOTAL", total)
}
