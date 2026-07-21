package ai

import "testing"

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
