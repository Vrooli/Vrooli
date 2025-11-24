package engine

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

func TestBrowserlessCapabilitiesValidateAndEnforceMatrix(t *testing.T) {
	eng, err := NewBrowserlessEngineWithFallback(logrus.New(), true)
	if err != nil {
		t.Fatalf("failed to build engine: %v", err)
	}

	caps, err := eng.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("capabilities returned error: %v", err)
	}
	if err := caps.Validate(); err != nil {
		t.Fatalf("capabilities failed validation: %v", err)
	}

	// Should reject HAR/video/tracing as currently unsupported.
	req := contracts.CapabilityRequirement{NeedsHAR: true, NeedsVideo: true, NeedsTracing: true}
	gap := caps.CheckCompatibility(req)
	if gap.Satisfied() || len(gap.Missing) == 0 {
		t.Fatalf("expected missing capabilities for HAR/video/tracing, got %+v", gap)
	}

	// Parallel tabs/uploads/downloads are allowed in browserless defaults.
	req = contracts.CapabilityRequirement{
		NeedsParallelTabs: true,
		NeedsFileUploads:  true,
		NeedsDownloads:    true,
		MinViewportWidth:  1280,
		MinViewportHeight: 720,
	}
	gap = caps.CheckCompatibility(req)
	if !gap.Satisfied() {
		t.Fatalf("expected requirements satisfied, got gap %+v", gap)
	}
}
