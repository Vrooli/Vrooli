package handlers

import (
	"testing"

	exportsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/exports"
)

func TestApplyReplayConfigMutatesGeneratedReplaySpec(t *testing.T) {
	spec := &exportsv1.ReplaySpec{
		Decor:        &exportsv1.ReplayDecor{},
		CursorMotion: &exportsv1.ReplayCursorMotion{},
		Presentation: &exportsv1.ReplayPresentation{
			Canvas:       &exportsv1.ReplayDimensions{Width: 1920, Height: 1080},
			Viewport:     &exportsv1.ReplayDimensions{Width: 1280, Height: 720},
			BrowserFrame: &exportsv1.ReplayFrameRect{Radius: 24},
		},
	}
	applyReplayConfigToSpec(spec, map[string]any{
		"style": map[string]any{
			"cursorSpeedProfile": "fast",
			"cursorPathStyle":    "bezier",
			"browserScale":       0.8,
			"background":         map[string]any{"type": "solid", "color": "#123456"},
			"watermark":          map[string]any{"enabled": true, "assetId": "brand-mark", "opacity": float64(65)},
			"introCard":          map[string]any{"enabled": true, "title": "Start", "duration": float64(900)},
			"outroCard":          map[string]any{"enabled": true, "ctaText": "Continue", "ctaUrl": "https://example.com"},
		},
	})
	if spec.GetCursorMotion().GetSpeedProfile() != "fast" || spec.GetCursorMotion().GetPathStyle() != "bezier" {
		t.Fatalf("cursor motion settings not applied: %v", spec.GetCursorMotion())
	}
	if spec.GetDecor().GetBackground().AsMap()["color"] != "#123456" {
		t.Fatalf("background not retained as protobuf Struct: %v", spec.GetDecor().GetBackground())
	}
	if spec.GetPresentation().GetBrowserFrame().GetWidth() == 0 || spec.GetPresentation().GetBrowserFrame().GetHeight() == 0 {
		t.Fatalf("browser scale did not update generated presentation geometry: %v", spec.GetPresentation())
	}
	if spec.GetWatermark().GetAssetId() != "brand-mark" || spec.GetWatermark().GetOpacity() != 65 {
		t.Fatalf("watermark settings not mapped: %v", spec.GetWatermark())
	}
	if spec.GetIntroCard().GetTitle() != "Start" || spec.GetIntroCard().GetDurationMs() != 900 {
		t.Fatalf("intro settings not mapped: %v", spec.GetIntroCard())
	}
	if spec.GetOutroCard().GetCtaText() != "Continue" || spec.GetOutroCard().GetCtaUrl() != "https://example.com" {
		t.Fatalf("outro settings not mapped: %v", spec.GetOutroCard())
	}
}
