package shots

import (
	"testing"

	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	"google.golang.org/protobuf/proto"
)

func videoBaseFixture() *shotsv1.ShotSnapshot {
	return &shotsv1.ShotSnapshot{
		Scene: &shotsv1.PhotographicScene{PlateMediaId: "plate", DisplayWidthPixels: 1920, DisplayHeightPixels: 1080},
		Presentation: &shotsv1.ScreenPresentation{SourceMediaId: "video", Kind: shotsv1.PresentationKind_PRESENTATION_KIND_PHONE_SYSTEM, PhoneSystemChrome: &shotsv1.PhoneSystemChrome{TopFill: &shotsv1.PresentationRGB{Red: 12, Green: 13, Blue: 14}}},
		VideoTiming: &shotsv1.VideoTiming{SourceMediaId: "video", SourceSha256: "a", TrimStartPts: 10, TrimEndPts: 100, SampleBinding: &shotsv1.VideoSampleBinding{TopFillSourceResolved: false, BottomFillSourceResolved: true, StatusInkSourceResolved: true, HomeInkSourceResolved: true}},
	}
}

func TestVideoBaseRecipePreservesExplicitColorsAndPinsSceneSourceAndTrim(t *testing.T) {
	base := videoBaseFixture()
	first, err := videoBaseRecipeSHA(base)
	if err != nil {
		t.Fatal(err)
	}
	manual := proto.Clone(base).(*shotsv1.ShotSnapshot)
	manual.Presentation.PhoneSystemChrome.TopFill = &shotsv1.PresentationRGB{Red: 15, Green: 13, Blue: 14}
	manualSHA, err := videoBaseRecipeSHA(manual)
	if err != nil || manualSHA == first {
		t.Fatalf("changing an explicit manual fill did not change the base recipe: %q, %v", manualSHA, err)
	}
	derived := proto.Clone(base).(*shotsv1.ShotSnapshot)
	derived.Presentation.PhoneSystemChrome.BottomFill = &shotsv1.PresentationRGB{Red: 80, Green: 90, Blue: 100}
	derivedSHA, err := videoBaseRecipeSHA(derived)
	if err != nil || derivedSHA != first {
		t.Fatalf("source-resolved fill changed the base recipe: %q != %q, %v", derivedSHA, first, err)
	}
	for name, mutate := range map[string]func(*shotsv1.ShotSnapshot){
		"scene": func(s *shotsv1.ShotSnapshot) { s.Scene.DisplayWidthPixels++ },
		"source": func(s *shotsv1.ShotSnapshot) { s.VideoTiming.SourceSha256 = "b" },
		"trim": func(s *shotsv1.ShotSnapshot) { s.VideoTiming.TrimStartPts++ },
	} {
		t.Run(name, func(t *testing.T) {
			changed := proto.Clone(base).(*shotsv1.ShotSnapshot)
			mutate(changed)
			got, err := videoBaseRecipeSHA(changed)
			if err != nil || got == first {
				t.Fatalf("changing %s did not change the base recipe: %q, %v", name, got, err)
			}
		})
	}
}
