package rendering

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"device-studio/internal/composition"
	"device-studio/internal/presentations"
)

// This is opt-in native protocol validation, not a realistic scene or an API
// job. Its source inspection is real and its plate is deliberately synthetic.
func TestParentNativeVideoSampleAndFiveSecondFinal(t *testing.T) { // [REQ:DS-PHOTO-REPLAY]
	evidence := os.Getenv("DEVICE_STUDIO_PARENT_VIDEO_EVIDENCE")
	if evidence == "" {
		t.Skip("set DEVICE_STUDIO_PARENT_VIDEO_EVIDENCE to a fresh absolute evidence directory")
	}
	if !filepath.IsAbs(evidence) {
		t.Fatal("evidence directory must be absolute")
	}
	if err := os.Mkdir(evidence, 0700); err != nil {
		t.Fatal(err)
	}
	caseName := os.Getenv("DEVICE_STUDIO_PARENT_VIDEO_CASE")
	if caseName != "" && caseName != "portrait" && caseName != "landscape" {
		t.Fatalf("unsupported parent video case %q", caseName)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 11*time.Minute)
	defer cancel()
	// Go runs package tests from api/internal/rendering.
	fixtures, err := filepath.Abs("../../../docs/internal/goal/evidence/E2V/integration/p4b-live-inspection/run-3-preserved/accepted-phone/3dfede96-8777-4bf7-b646-c79a1dc4a12b")
	if err != nil {
		t.Fatal(err)
	}
	read := func(name string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(fixtures, name))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	files := map[string][]byte{}
	for _, name := range []string{InspectionManifestName, InspectionReceiptName, InspectionClockName, InspectionSourceName, VideoProfileName} {
		files[name] = read(name)
	}
	// The video attempt stages the original immutable inspection manifest under
	// a distinct alias so it cannot collide with its own worker entrypoint.
	files[AcceptedInspectionManifestName] = files[InspectionManifestName]
	r, err := VerifyInspectionArtifacts(files[InspectionManifestName], files[InspectionReceiptName], files[InspectionClockName])
	if err != nil {
		t.Fatal(err)
	}
	if r.FrameCount != 488 || r.SourceSHA256 != "4bf3e1663ce60116e85e4172f359ac9ce7a2b39a99bf4fa5d8b2e0fe040db857" {
		t.Fatal("authentic source receipt differs")
	}
	bind := func(name string) VideoBinding {
		return VideoBinding{Name: name, SHA256: digest(files[name]), SizeBytes: int64(len(files[name]))}
	}
	plate := image.NewNRGBA(image.Rect(0, 0, 1920, 1080))
	background := color.NRGBA{R: 30, G: 40, B: 55, A: 255}
	draw.Draw(plate, plate.Bounds(), &image.Uniform{C: background}, image.Point{}, draw.Src)
	mask := image.NewGray(plate.Bounds())
	draw.Draw(mask, image.Rect(740, 100, 1170, 980), image.White, image.Point{}, draw.Src)
	for name, img := range map[string]image.Image{"plate.png": plate, "active.png": mask} {
		var encoded bytes.Buffer
		if err := png.Encode(&encoded, img); err != nil {
			t.Fatal(err)
		}
		files[name] = encoded.Bytes()
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	worker, err := os.Open(executable)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.New()
	_, hashErr := io.Copy(hash, worker)
	closeErr := worker.Close()
	if hashErr != nil || closeErr != nil {
		t.Fatalf("hash test worker: %v / %v", hashErr, closeErr)
	}
	baseSHA := digest([]byte("synthetic-parent-base-recipe-v1"))
	sampleRecipeSHA := digest([]byte("synthetic-parent-full-sample-recipe-v1"))
	m := VideoManifest{
		Version: 1, ProtocolVersion: 1, JobID: "parent-sample", AttemptID: "parent-sample-attempt",
		InputFingerprint: digest([]byte("parent-sample-input")), WorkerSHA256: hex.EncodeToString(hash.Sum(nil)),
		Mode: VideoModeSample, VideoProfile: VideoProfileID, CompositorProfile: VideoCompositorID,
		Inspection: VideoInspection{
			Source: bind(InspectionSourceName), Manifest: bind(AcceptedInspectionManifestName), Receipt: bind(InspectionReceiptName), Clock: bind(InspectionClockName),
			ProfileID: r.ProfileID, ProfileDigest: r.ProfileDigest, ProtocolVersion: r.ProtocolVersion,
			SelectedStream: uint32(*r.VideoStreamIndex), TimeBase: composition.Rational{Num: r.TimeBaseNum, Den: r.TimeBaseDen},
			StreamStartPTS: *r.StreamStartPTS, SourceEndPTS: *r.SourceEndPTS,
			LastFrameDurationTicks: r.LastFrameDurationTicks, EOFPolicy: r.EOFPolicy,
			FrameCount: r.FrameCount, Width: int(r.DisplayWidth), Height: int(r.DisplayHeight),
			ColorPrimaries: r.ColorPrimaries, ColorTransfer: r.ColorTransfer, ColorMatrix: r.ColorMatrix, ColorRange: r.ColorRange, ColorPolicy: r.ColorPolicy,
		},
		TrimStartPTS: 0, TrimEndPTS: 76800, PlaybackRate: composition.Rational{Num: 1, Den: 1}, OutputRate: composition.Rational{Num: 24, Den: 1},
		Width: 1920, Height: 1080,
		Scene: VideoScene{Plate: bind("plate.png"), ActiveMask: bind("active.png"), SurfaceWidth: 430, SurfaceHeight: 880,
			Quad: [4]Corner{{X: 740, Y: 100}, {X: 1170, Y: 100}, {X: 1170, Y: 980}, {X: 740, Y: 980}}, Samples: 2},
		Presentation: &presentations.Config{Target: image.Pt(430, 880), Phone: &presentations.PhoneConfig{
			ProfileVersion: presentations.PhoneProfileVersionAuto, Appearance: presentations.AppearanceAuto,
			ShowStatus: true, ShowHome: true, Clock: "9:41", BatteryPercent: 82, Cellular: true, WiFi: true,
		}},
		ShotRevision: 1, BaseShotRevision: 1, BaseRecipeVersion: "parent-native-test-v1", BaseRecipeSHA256: baseSHA, RecipeSHA256: sampleRecipeSHA,
	}
	run := func(name string, manifest VideoManifest) (VideoReceipt, []byte, time.Duration) {
		t.Helper()
		workspace := filepath.Join(evidence, name)
		if err := os.Mkdir(workspace, 0700); err != nil {
			t.Fatal(err)
		}
		for filename, data := range files {
			if err := os.WriteFile(filepath.Join(workspace, filename), data, 0600); err != nil {
				t.Fatal(err)
			}
		}
		manifestBytes, err := json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(workspace, VideoManifestName)
		if err := os.WriteFile(path, manifestBytes, 0600); err != nil {
			t.Fatal(err)
		}
		start := time.Now()
		if err := RunVideoManifest(ctx, path); err != nil {
			t.Fatalf("%s worker after %s: %v", name, time.Since(start), err)
		}
		elapsed := time.Since(start)
		receiptBytes, err := os.ReadFile(filepath.Join(workspace, VideoReceiptName))
		if err != nil {
			t.Fatal(err)
		}
		filename := VideoSampleName
		if manifest.Mode == VideoModeFinal {
			filename = VideoOutputName
		}
		output, err := os.Open(filepath.Join(workspace, filename))
		if err != nil {
			t.Fatal(err)
		}
		info, err := output.Stat()
		if err != nil {
			output.Close()
			t.Fatal(err)
		}
		receipt, verifyErr := VerifyVideoArtifacts(ctx, manifestBytes, receiptBytes, output, info.Size())
		closeErr := output.Close()
		if verifyErr != nil || closeErr != nil {
			t.Fatalf("%s artifact verification: %v / %v", name, verifyErr, closeErr)
		}
		t.Logf("%s elapsed=%s bytes=%d pixels=%s", name, elapsed, info.Size(), receipt.FirstFrameRGBSHA256)
		return receipt, receiptBytes, elapsed
	}
	if caseName == "portrait" {
		portraitPlate := image.NewNRGBA(image.Rect(0, 0, 2160, 3840))
		background := color.NRGBA{R: 30, G: 40, B: 55, A: 255}
		draw.Draw(portraitPlate, portraitPlate.Bounds(), &image.Uniform{C: background}, image.Point{}, draw.Src)
		activeBounds := image.Rect(650, 1040, 1510, 2800) // 860x1760, matching 430x880 at 2x.
		portraitMask := image.NewGray(portraitPlate.Bounds())
		draw.Draw(portraitMask, activeBounds, image.White, image.Point{}, draw.Src)
		for name, img := range map[string]image.Image{"plate.png": portraitPlate, "active.png": portraitMask} {
			var encoded bytes.Buffer
			if err := png.Encode(&encoded, img); err != nil {
				t.Fatal(err)
			}
			files[name] = encoded.Bytes()
		}
		m.Mode, m.JobID, m.AttemptID = VideoModeSample, "parent-portrait-sample", "parent-portrait-sample-attempt"
		m.InputFingerprint = digest([]byte("parent-portrait-sample-input"))
		m.TrimStartPTS, m.TrimEndPTS = 15360, 46080 // 1s trim start, 2s source at 2x playback.
		m.PlaybackRate, m.OutputRate = composition.Rational{Num: 2, Den: 1}, composition.Rational{Num: 30, Den: 1}
		m.Width, m.Height = 1080, 1920
		m.Scene.Plate, m.Scene.ActiveMask = bind("plate.png"), bind("active.png")
		m.Scene.Quad = [4]Corner{{X: 650, Y: 1040}, {X: 1510, Y: 1040}, {X: 1510, Y: 2800}, {X: 650, Y: 2800}}
		m.Presentation.Target = image.Pt(430, 880)
		m.ShotRevision, m.BaseShotRevision, m.Sample = 1, 1, nil
		m.BaseRecipeSHA256 = digest([]byte("parent-portrait-base-recipe-v1"))
		m.RecipeSHA256 = digest([]byte("parent-portrait-full-sample-recipe-v1"))
		sample, sampleBytes, sampleElapsed := run("portrait-sample", m)
		sampleImageFile, err := os.Open(filepath.Join(evidence, "portrait-sample", VideoSampleName))
		if err != nil {
			t.Fatal(err)
		}
		sampleImage, decodeErr := png.Decode(sampleImageFile)
		_ = sampleImageFile.Close()
		if decodeErr != nil {
			t.Fatal(decodeErr)
		}
		if actual := color.NRGBAModel.Convert(sampleImage.At(10, 10)); actual != background {
			t.Fatalf("portrait sample protected corner changed: %v", actual)
		}
		files["accepted-portrait-sample.json"] = sampleBytes
		m.Mode, m.JobID, m.AttemptID = VideoModeFinal, "parent-portrait-final", "parent-portrait-final-attempt"
		m.InputFingerprint = digest([]byte("parent-portrait-final-input"))
		m.ShotRevision, m.RecipeSHA256 = 2, digest([]byte("parent-portrait-sampled-recipe-v2"))
		m.Presentation.Phone.TopFill, m.Presentation.Phone.BottomFill = &sample.ResolvedTopFill, &sample.ResolvedBottomFill
		m.Presentation.Phone.StatusInk, m.Presentation.Phone.HomeInk = sample.ResolvedStatusInk, sample.ResolvedHomeInk
		layout, err := presentations.BuildLayoutForVideoRaster(image.Rect(0, 0, 430, 880), *m.Presentation)
		if err != nil {
			t.Fatal(err)
		}
		m.PresentationLayout = &layout
		m.Sample = &VideoSampleBinding{
			JobID: sample.JobID, AttemptID: sample.AttemptID, ReceiptSHA256: digest(sampleBytes), ManifestSHA256: sample.ManifestSHA256,
			RecipeSHA256: sample.RecipeSHA256, InspectionSHA256: VideoInspectionDigest(m.Inspection), SourceSHA256: r.SourceSHA256, ClockSHA256: r.ClockSHA256, TimingSHA256: VideoTimingDigest(m),
			ShotRevision: 1, BaseShotRevision: 1, BaseRecipeVersion: m.BaseRecipeVersion, BaseRecipeSHA256: m.BaseRecipeSHA256,
			ResolvedTopFill: sample.ResolvedTopFill, ResolvedBottomFill: sample.ResolvedBottomFill, ResolvedStatusInk: sample.ResolvedStatusInk, ResolvedHomeInk: sample.ResolvedHomeInk,
			TopFillSourceResolved: true, BottomFillSourceResolved: true, StatusInkSourceResolved: true, HomeInkSourceResolved: true,
			FirstFrameRGBSHA256: sample.FirstFrameRGBSHA256, PixelProfile: sample.PixelProfile, Width: 1080, Height: 1920, ReceiptArtifact: bind("accepted-portrait-sample.json"),
		}
		final, _, finalElapsed := run("portrait-final", m)
		if final.FirstFrameRGBSHA256 != sample.FirstFrameRGBSHA256 || final.Native.FrameCount != 30 || final.Native.FrameRate != m.OutputRate || final.Native.FirstPTS != 0 || !final.Native.FullDecodeVerified {
			t.Fatalf("portrait sample/final parity or output timing failed: %+v", final)
		}
		t.Logf("portrait sample elapsed=%s final elapsed=%s; 2160x3840 to1080x1920, 1s trim, 2x playback, 30fps", sampleElapsed, finalElapsed)
		return
	}
	sample, sampleBytes, sampleElapsed := run("sample", m)
	sampleFile, err := os.Open(filepath.Join(evidence, "sample", VideoSampleName))
	if err != nil {
		t.Fatal(err)
	}
	sampleImage, decodeErr := png.Decode(sampleFile)
	sampleFile.Close()
	if decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if actual := color.NRGBAModel.Convert(sampleImage.At(10, 10)); actual != background {
		t.Fatalf("protected background changed: %v", actual)
	}
	files["accepted-sample.json"] = sampleBytes
	m.Mode, m.JobID, m.AttemptID = VideoModeFinal, "parent-final", "parent-final-attempt"
	m.InputFingerprint = digest([]byte("parent-final-input"))
	m.ShotRevision, m.RecipeSHA256 = 2, digest([]byte("parent-saved-sample-recipe-v2"))
	p := m.Presentation.Phone
	p.TopFill, p.BottomFill = &sample.ResolvedTopFill, &sample.ResolvedBottomFill
	p.StatusInk, p.HomeInk = sample.ResolvedStatusInk, sample.ResolvedHomeInk
	layout, err := presentations.BuildLayoutForVideoRaster(image.Rect(0, 0, 430, 880), *m.Presentation)
	if err != nil {
		t.Fatal(err)
	}
	m.PresentationLayout = &layout
	m.Sample = &VideoSampleBinding{
		JobID: sample.JobID, AttemptID: sample.AttemptID, ReceiptSHA256: digest(sampleBytes), ManifestSHA256: sample.ManifestSHA256,
		RecipeSHA256: sample.RecipeSHA256, InspectionSHA256: VideoInspectionDigest(m.Inspection), SourceSHA256: r.SourceSHA256, ClockSHA256: r.ClockSHA256, TimingSHA256: VideoTimingDigest(m),
		ShotRevision: 1, BaseShotRevision: 1, BaseRecipeVersion: m.BaseRecipeVersion, BaseRecipeSHA256: baseSHA,
		ResolvedTopFill: sample.ResolvedTopFill, ResolvedBottomFill: sample.ResolvedBottomFill, ResolvedStatusInk: sample.ResolvedStatusInk, ResolvedHomeInk: sample.ResolvedHomeInk,
		TopFillSourceResolved: true, BottomFillSourceResolved: true, StatusInkSourceResolved: true, HomeInkSourceResolved: true,
		FirstFrameRGBSHA256: sample.FirstFrameRGBSHA256, PixelProfile: sample.PixelProfile, Width: 1920, Height: 1080, ReceiptArtifact: bind("accepted-sample.json"),
	}
	final, _, finalElapsed := run("final", m)
	if final.FirstFrameRGBSHA256 != sample.FirstFrameRGBSHA256 || final.Native.FrameCount != 120 || final.Native.FirstPTS != 0 || !final.Native.FullDecodeVerified {
		t.Fatalf("sample/final parity or output timing failed: %+v", final)
	}
	report, err := json.MarshalIndent(map[string]any{
		"kind": "authentic-source-synthetic-plate-native-code-path", "photographic_look_qualified": false, "owned_cgroup_qualified": false,
		"sample_elapsed_ms": sampleElapsed.Milliseconds(), "final_elapsed_ms": finalElapsed.Milliseconds(), "output_seconds": 5,
		"sample_revision": sample.ShotRevision, "final_revision": final.ShotRevision, "first_frame_rgb_sha256": final.FirstFrameRGBSHA256,
		"native": final.Native, "source_sha256": r.SourceSHA256, "worker_sha256": m.WorkerSHA256,
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evidence, "result.json"), append(report, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}
