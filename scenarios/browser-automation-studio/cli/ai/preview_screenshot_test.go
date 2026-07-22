package ai

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestPreviewScreenshotFlagsFromContext_UsesProductionFlagParser(t *testing.T) {
	command := previewScreenshotCommand(nil)
	rc, err := cliapp.NewTestRunContextFromArgs(command.Args, []string{
		"--url", "https://example.test", "--width", "390", "--height", "844", "--device-scale-factor", "2",
	}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	flags, err := previewScreenshotFlagsFromContext(rc)
	if err != nil {
		t.Fatal(err)
	}
	request, err := buildPreviewScreenshotRequest(flags)
	if err != nil {
		t.Fatal(err)
	}
	if request.GetViewport().GetWidth() != 390 || request.GetViewport().GetHeight() != 844 || request.GetViewport().GetDeviceScaleFactor() != 2 {
		t.Fatalf("viewport = %+v, want 390x844 at scale 2", request.GetViewport())
	}
}

func TestPreviewScreenshotFlagsFromContext_RejectsInvalidNumbers(t *testing.T) {
	command := previewScreenshotCommand(nil)
	for _, args := range [][]string{
		{"--url", "https://example.test", "--width", "nope"},
		{"--url", "https://example.test", "--height", "nope"},
		{"--url", "https://example.test", "--device-scale-factor", "nope"},
	} {
		rc, err := cliapp.NewTestRunContextFromArgs(command.Args, args, nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := previewScreenshotFlagsFromContext(rc); err == nil {
			t.Fatalf("previewScreenshotFlagsFromContext(%v) succeeded, want parse error", args)
		}
	}
}

func TestBuildPreviewScreenshotRequest_MapsNestedViewportFlags(t *testing.T) {
	request, err := buildPreviewScreenshotRequest(previewScreenshotFlags{url: "https://example.test", width: 390, height: 844})
	if err != nil {
		t.Fatal(err)
	}
	if request.GetViewport().GetWidth() != 390 || request.GetViewport().GetHeight() != 844 {
		t.Fatalf("viewport = %+v, want 390x844", request.GetViewport())
	}
}

func TestBuildPreviewScreenshotRequest_ResolvesDeviceAndRejectsConflict(t *testing.T) {
	request, err := buildPreviewScreenshotRequest(previewScreenshotFlags{url: "https://example.test", device: "mobile"})
	if err != nil {
		t.Fatal(err)
	}
	if request.GetViewport().GetWidth() != 390 || request.GetViewport().GetHeight() != 844 {
		t.Fatalf("mobile viewport = %+v", request.GetViewport())
	}
	if _, err := buildPreviewScreenshotRequest(previewScreenshotFlags{url: "https://example.test", device: "mobile", width: 390}); err == nil {
		t.Fatal("expected conflict error")
	}
}

func TestBuildPreviewScreenshotRequest_MapsDeviceScaleFactor(t *testing.T) {
	request, err := buildPreviewScreenshotRequest(previewScreenshotFlags{url: "https://example.test", deviceScale: 2, hasDeviceScale: true})
	if err != nil {
		t.Fatal(err)
	}
	if request.GetViewport().GetDeviceScaleFactor() != 2 {
		t.Fatalf("device scale factor = %v, want 2", request.GetViewport().GetDeviceScaleFactor())
	}
	if _, err := buildPreviewScreenshotRequest(previewScreenshotFlags{url: "https://example.test", deviceScale: 4.5, hasDeviceScale: true}); err == nil {
		t.Fatal("expected invalid scale factor error")
	}
}
