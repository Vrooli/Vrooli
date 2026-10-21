package rendering

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"device-studio/internal/composition"
	"device-studio/internal/presentations"
)

func TestCameraManifestValidatesPortraitAndLandscapeProfiles(t *testing.T) {
	for _, tc := range []struct {
		name           string
		width, height  int
		plateW, plateH int
		quad           [4]Corner
	}{
		{
			name:  "landscape",
			width: 1920, height: 1080, plateW: 1920, plateH: 1080,
			quad: [4]Corner{{X: 720, Y: 80}, {X: 1200, Y: 80}, {X: 1200, Y: 1040}, {X: 720, Y: 1040}},
		},
		{
			name:  "portrait",
			width: 1080, height: 1920, plateW: 1080, plateH: 1920,
			quad: [4]Corner{{X: 270, Y: 250}, {X: 810, Y: 250}, {X: 810, Y: 1670}, {X: 270, Y: 1670}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := cameraSampleManifest(tc.width, tc.height, tc.plateW, tc.plateH, tc.quad)
			if err := ValidateVideoManifest(m); err != nil {
				t.Fatalf("valid %s camera manifest rejected: %v", tc.name, err)
			}
		})
	}
}

func TestScreenEntryFitsThreeByTwoPhonePlateAndKeepsActiveViewportTarget(t *testing.T) {
	m := cameraSampleManifest(1920, 1080, 1536, 1024, [4]Corner{{512, 128}, {1024, 128}, {1024, 896}, {512, 896}})
	startPTS, endPTS := m.TrimStartPTS, m.TrimEndPTS
	if err := ValidateVideoManifest(m); err != nil {
		t.Fatalf("3:2 photographic phone plate should be contain-fitted: %v", err)
	}
	path, err := newVideoCameraPath(m, image.Rect(0, 0, 1536, 1024), nil)
	if err != nil {
		t.Fatalf("build fitted camera path: %v", err)
	}
	scale := 1080.0 / 1024.0
	fitOffset := (1920 - 1536*scale) / 2
	quad := [4]composition.Point{{X: 512*scale + fitOffset, Y: 128 * scale}, {X: 1024*scale + fitOffset, Y: 128 * scale}, {X: 1024*scale + fitOffset, Y: 896 * scale}, {X: 512*scale + fitOffset, Y: 896 * scale}}
	viewport, err := presentations.BuildVideoViewport(image.Rect(0, 0, m.Inspection.Width, m.Inspection.Height), *m.Presentation)
	if err != nil {
		t.Fatal(err)
	}
	wantFrom, err := composition.ProjectRasterRect(image.Pt(m.Scene.SurfaceWidth, m.Scene.SurfaceHeight), image.Pt(m.Width, m.Height), quad, image.Rect(0, 0, m.Scene.SurfaceWidth, m.Scene.SurfaceHeight), viewport)
	if err != nil {
		t.Fatal(err)
	}
	if path.from != wantFrom || path.viewport != viewport {
		t.Fatalf("initial camera mapping did not apply the contain transform to the same app viewport: from=%+v want=%+v viewport=%v want=%v", path.from, wantFrom, path.viewport, viewport)
	}
	if path.quad(path.duration-1) != path.target || path.target != containedViewportQuad(path.viewport, image.Pt(m.Width, m.Height)) {
		t.Fatalf("screen-entry endpoint no longer targets the active app viewport: target=%+v viewport=%v", path.target, path.viewport)
	}
	if m.TrimStartPTS != startPTS || m.TrimEndPTS != endPTS {
		t.Fatalf("plate fit changed source-clock trim: got %d..%d want %d..%d", m.TrimStartPTS, m.TrimEndPTS, startPTS, endPTS)
	}
}

func TestScreenEntryManifestAcceptsTwelveSecondsAtBothRatesAndRejectsOverrun(t *testing.T) {
	for _, rate := range []int64{24, 30} {
		frames := uint32(12 * rate)
		m := cameraSampleManifest(1920, 1080, 1920, 1080, [4]Corner{{720, 80}, {1200, 80}, {1200, 1040}, {720, 1040}})
		m.Inspection.TimeBase = composition.Rational{Num: 1, Den: rate}
		m.Inspection.SourceEndPTS = int64(frames + 1)
		m.TrimEndPTS = int64(frames)
		m.OutputRate.Num = rate
		m.Camera.DurationOutputFrames = frames
		if err := ValidateVideoManifest(m); err != nil {
			t.Fatalf("12-second screen-entry manifest at %d fps rejected: %v", rate, err)
		}
		path, err := newVideoCameraPath(m, image.Rect(0, 0, m.Scene.PlateWidth, m.Scene.PlateHeight), nil)
		if err != nil {
			t.Fatalf("build %d-frame camera path: %v", frames, err)
		}
		if len(path.frames) != int(frames) || path.quad(frames-1) != path.target {
			t.Fatalf("camera path endpoint not reached for %d-frame schedule", frames)
		}
	}

	m := cameraSampleManifest(1920, 1080, 1920, 1080, [4]Corner{{720, 80}, {1200, 80}, {1200, 1040}, {720, 1040}})
	m.TrimEndPTS = 289 // 12.04 seconds at 24 fps.
	m.Inspection.SourceEndPTS = 300
	if err := ValidateVideoManifest(m); err == nil || !strings.Contains(err.Error(), "12-second") {
		t.Fatalf("camera clip above 12 seconds error = %v", err)
	}
	m = cameraSampleManifest(1920, 1080, 1920, 1080, [4]Corner{{720, 80}, {1200, 80}, {1200, 1040}, {720, 1040}})
	m.Camera.DurationOutputFrames = 361
	if err := ValidateVideoManifest(m); err == nil {
		t.Fatal("camera transition above 360 frames was accepted")
	}
	m = cameraSampleManifest(1920, 1080, 1920, 1080, [4]Corner{{720, 80}, {1200, 80}, {1200, 1040}, {720, 1040}})
	m.Scene.PlateWidth++
	if _, err := newVideoCameraPath(m, image.Rect(0, 0, m.Scene.PlateWidth-1, m.Scene.PlateHeight), nil); err == nil {
		t.Fatal("camera plate raster differing from pinned source dimensions was accepted")
	}
}

func TestFixedPlateProfileKeepsTenSecondDurationLimit(t *testing.T) {
	m := validManualLandscapeVideoManifest()
	m.TrimEndPTS = 241 // 10.04 seconds at 24 fps.
	m.Inspection.SourceEndPTS = 300
	if err := ValidateVideoManifest(m); err == nil || !strings.Contains(err.Error(), "ten-second") {
		t.Fatalf("fixed-plate duration above ten seconds error = %v", err)
	}
}

func TestCameraPathAdvancesByOutputOrdinalWhenVFRSourceFrameRepeats(t *testing.T) {
	plan, err := composition.PlanFrames(context.Background(), composition.FrameClock{
		SourcePTS: []int64{0, 1}, SourceTimeBase: composition.Rational{Num: 1, Den: 1}, SourceEndPTS: 2,
		TrimStartPTS: 0, TrimEndPTS: 2, PlaybackRate: composition.Rational{Num: 1, Den: 1},
		OutputRate: composition.Rational{Num: 2, Den: 1}, Underflow: composition.RangeReject, EOF: composition.RangeReject,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan) != 4 || plan[0].SourceIndex != plan[1].SourceIndex {
		t.Fatalf("test clock did not repeat the first source frame: %+v", plan)
	}
	path := videoCameraPath{
		from:   [4]composition.Point{{5, 5}, {25, 5}, {25, 45}, {5, 45}},
		target: [4]composition.Point{{20, 10}, {40, 10}, {40, 50}, {20, 50}}, duration: 3,
	}
	first, next := path.quad(0), path.quad(1)
	if first == next || first != path.from {
		t.Fatalf("camera did not begin at exact frame zero and move by output ordinal: first=%+v next=%+v", first, next)
	}
	if got := path.quad(uint32(len(plan) - 1)); got != path.target {
		t.Fatalf("camera endpoint/hold quad=%+v, want target %+v", got, path.target)
	}
}

func TestCameraPreflightRejectsWarpThatCrossesOutputHorizon(t *testing.T) {
	raster := image.Pt(100, 100)
	from := [4]composition.Point{{40, 40}, {60, 40}, {60, 60}, {40, 60}}
	candidates := [][4]composition.Point{
		{{40, 0}, {60, 0}, {100, 100}, {0, 100}},
		{{45, 45}, {55, 45}, {100, 100}, {0, 100}},
		{{0, 0}, {100, 0}, {60, 100}, {40, 100}},
		{{0, 0}, {10, 0}, {80, 80}, {0, 80}},
		{{0, 0}, {100, 0}, {100, 10}, {0, 90}},
	}
	for _, to := range candidates {
		if err := composition.ValidateProjectiveWarp(raster, from, to); err != nil && strings.Contains(err.Error(), "horizon") {
			return
		}
	}
	t.Fatal("strongly skewed camera paths crossing the output horizon were all accepted or rejected for another reason")
}

func TestFixedVideoTimingDigestAndJSONRemainLegacyWhenCameraAbsent(t *testing.T) {
	m := validManualLandscapeVideoManifest()
	legacy := struct {
		Start, End       int64
		Playback, Output composition.Rational
		W, H             int
		Profile          string
	}{m.TrimStartPTS, m.TrimEndPTS, m.PlaybackRate, m.OutputRate, m.Width, m.Height, m.VideoProfile}
	legacyBytes, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if got := videoTimingDigest(m); got != digest(legacyBytes) {
		t.Fatalf("fixed timing digest changed: got %s want legacy %s", got, digest(legacyBytes))
	}
	encoded, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), `"camera"`) || strings.Contains(string(encoded), `"plate_width"`) {
		t.Fatalf("fixed manifest serialized camera-only fields: %s", encoded)
	}
}

func TestRetainedRun6FixedManifestRemainsCanonicalAndCameraFree(t *testing.T) {
	path := "testdata/retained/fixed-video-manifest.json"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(data)
	if got := hex.EncodeToString(hash[:]); got != "7998719b6cc2d945b51b67109e416bbb52c9d7b7969e162e623352aff6c6ac78" {
		t.Fatalf("retained fixed manifest fixture changed: %s", got)
	}
	manifest, err := decodeCanonicalVideoManifest(data)
	if err != nil {
		t.Fatalf("decode retained fixed manifest: %v", err)
	}
	if manifest.Camera != nil || manifest.Scene.PlateWidth != 0 || manifest.Scene.PlateHeight != 0 || manifest.VideoProfile != VideoProfileID {
		t.Fatal("retained fixed manifest acquired camera-only fields or profile")
	}
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(append(encoded, '\n'), data) {
		limit := min(len(encoded), len(data))
		at := 0
		for at < limit && encoded[at] == data[at] {
			at++
		}
		t.Fatalf("retained fixed manifest canonical bytes changed after decode/encode (%s): sizes %d/%d first difference at %d: %q / %q", filepath.Base(path), len(data), len(encoded), at, data[max(0, at-24):min(len(data), at+48)], encoded[max(0, at-24):min(len(encoded), at+48)])
	}
}

func cameraSampleManifest(width, height, plateWidth, plateHeight int, quad [4]Corner) VideoManifest {
	m := validManualLandscapeVideoManifest()
	m.Mode = VideoModeSample
	m.VideoProfile = VideoScreenEntryProfileID
	m.Camera = &Camera{ProfileID: ScreenEntryCameraProfileID, ProfileVersion: ScreenEntryCameraProfileVersion, DurationOutputFrames: 48, MatteRGB: &presentations.RGB{R: 31, G: 35, B: 42}}
	m.Inspection.Width, m.Inspection.Height = 430, 880
	m.Inspection.TimeBase = composition.Rational{Num: 1, Den: 24}
	m.Inspection.SourceEndPTS = 240
	m.Inspection.LastFrameDurationTicks = 1
	m.TrimEndPTS = 100
	m.OutputRate = composition.Rational{Num: 24, Den: 1}
	m.Width, m.Height = width, height
	m.Scene.PlateWidth, m.Scene.PlateHeight = plateWidth, plateHeight
	m.Scene.SurfaceWidth, m.Scene.SurfaceHeight = 430, 880
	m.Scene.Quad = quad
	m.Presentation = &presentations.Config{Target: image.Pt(430, 880), Phone: &presentations.PhoneConfig{ProfileVersion: presentations.PhoneProfileVersionAuto, Appearance: presentations.AppearanceAuto}}
	m.PresentationLayout = nil
	m.ShotRevision, m.BaseShotRevision = 3, 3
	return m
}
