package rendering

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"math"
	"os"
	"strings"
	"testing"

	"device-studio/internal/composition"
	"device-studio/internal/presentations"
)

func TestVideoDurationUsesExactWideDeltaAndStrictTenSecondLimit(t *testing.T) {
	base := VideoManifest{
		Inspection:   VideoInspection{TimeBase: composition.Rational{Num: 1, Den: 24}},
		TrimStartPTS: 0,
		TrimEndPTS:   240,
		PlaybackRate: composition.Rational{Num: 1, Den: 1},
		OutputRate:   composition.Rational{Num: 24, Den: 1},
	}
	if err := validateVideoDuration(base); err != nil {
		t.Fatalf("exact ten second clip rejected: %v", err)
	}
	count, err := plannedVideoFrameCount(base)
	if err != nil || count != 240 {
		t.Fatalf("planned count=(%d,%v), want (240,nil)", count, err)
	}
	base.TrimEndPTS++
	if err := validateVideoDuration(base); err == nil {
		t.Fatal("clip longer than ten seconds was accepted")
	}
	base.TrimStartPTS = math.MinInt64
	base.TrimEndPTS = math.MaxInt64
	base.Inspection.TimeBase = composition.Rational{Num: 1, Den: 1}
	if err := validateVideoDuration(base); err == nil {
		t.Fatal("wide PTS delta overflowed and was accepted")
	}
}

func TestVideoManifestUsesNonCollidingAcceptedInspectionManifestName(t *testing.T) {
	manifest := validManualLandscapeVideoManifest()
	if err := ValidateVideoManifest(manifest); err != nil {
		t.Fatalf("video manifest with staged accepted inspection alias rejected: %v", err)
	}
	manifest.Inspection.Manifest.Name = InspectionManifestName
	if err := ValidateVideoManifest(manifest); err == nil {
		t.Fatal("video manifest reused the worker's inspection-manifest entrypoint name")
	}
}

func TestPlannedOutputPTSUsesActualStreamTimeBase(t *testing.T) {
	m := VideoManifest{OutputRate: composition.Rational{Num: 24, Den: 1}}
	pts, err := plannedOutputPTS(m, composition.Rational{Num: 1, Den: 12288}, 24)
	if err != nil {
		t.Fatal(err)
	}
	if pts[0] != 0 || pts[1] != 512 || pts[23] != 11776 {
		t.Fatalf("stream PTS = first:%d second:%d last:%d, want 0,512,11776", pts[0], pts[1], pts[23])
	}
	if _, err := plannedOutputPTS(m, composition.Rational{Num: 1, Den: 1000}, 24); err == nil {
		t.Fatal("nonintegral frame cadence in stream time base was accepted")
	}
}

func TestExpandRGB24WritesOpaqueRGBAWithoutChannelShift(t *testing.T) {
	dst := make([]byte, 8)
	if err := expandRGB24(dst, []byte{0x12, 0x34, 0x56, 0xaa, 0xbb, 0xcc}); err != nil {
		t.Fatal(err)
	}
	want := []byte{0x12, 0x34, 0x56, 0xff, 0xaa, 0xbb, 0xcc, 0xff}
	for i := range want {
		if dst[i] != want[i] {
			t.Fatalf("RGBA byte %d = %02x, want %02x", i, dst[i], want[i])
		}
	}
	if err := expandRGB24(dst, []byte{1, 2, 3}); err == nil {
		t.Fatal("mismatched packed source and destination lengths were accepted")
	}
}

func TestResizeVideoFrameContainsArbitraryAspectWithNeutralMatte(t *testing.T) {
	src := image.NewRGBA(image.Rect(9, -4, 11, -2))
	src.SetRGBA(9, -4, color.RGBA{R: 0, A: 255})
	src.SetRGBA(10, -4, color.RGBA{G: 255, A: 255})
	src.SetRGBA(9, -3, color.RGBA{B: 255, A: 255})
	src.SetRGBA(10, -3, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	got, err := resizeVideoFrame(context.Background(), src, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	pix := got.RGBA64At(0, 0)
	if pix.R < 16000 || pix.R > 16800 || pix.G < 32000 || pix.G > 33500 || pix.B < 32000 || pix.B > 33500 || pix.A != 65535 {
		t.Fatalf("centered downsample = %#v, want opaque 1/4,1/2,1/2 channel average", pix)
	}
	portrait, err := resizeVideoFrame(context.Background(), src, 1, 3)
	if err != nil {
		t.Fatalf("contain-fit landscape source into portrait output: %v", err)
	}
	wantR, wantG, wantB, wantA := videoLetterboxMatte.RGBA()
	if got := portrait.RGBA64At(0, 0); uint32(got.R) != wantR || uint32(got.G) != wantG || uint32(got.B) != wantB || uint32(got.A) != wantA {
		t.Fatalf("letterbox matte = %#v, want opaque neutral matte", got)
	}
	if got := portrait.RGBA64At(0, 1); got.A != 65535 || got.R == 0 && got.G == 0 && got.B == 0 {
		t.Fatalf("contained source pixel = %#v, want opaque image content", got)
	}
	if got := portrait.RGBA64At(0, 2); uint32(got.R) != wantR || uint32(got.G) != wantG || uint32(got.B) != wantB || uint32(got.A) != wantA {
		t.Fatalf("bottom letterbox matte = %#v, want opaque neutral matte", got)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := resizeVideoFrame(ctx, src, 1, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled resize error = %v", err)
	}
}

func TestVideoFileLimitNeverWritesPastCap(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "video-bound-")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	w := videoFileLimit{file: f, limit: 4}
	if n, err := w.Write([]byte("1234")); err != nil || n != 4 {
		t.Fatalf("first bounded write=(%d,%v)", n, err)
	}
	if n, err := w.Write([]byte("5")); err == nil || n != 0 {
		t.Fatalf("overflow write=(%d,%v), want rejected before writing", n, err)
	}
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}
	info, err := f.Stat()
	if err != nil || info.Size() != 4 {
		t.Fatalf("bounded file size=(%v,%v), want 4", info, err)
	}
}

func TestVideoManifestAcceptsPortraitRecordingInsideLandscapePlate(t *testing.T) {
	m := validManualLandscapeVideoManifest()
	m.Inspection.Width, m.Inspection.Height = 430, 880
	var err error
	layout, err := presentations.BuildLayoutForVideoRaster(image.Rect(0, 0, 430, 880), *m.Presentation)
	if err != nil {
		t.Fatal(err)
	}
	m.PresentationLayout = &layout
	if err := validateVideoManifest(m); err != nil {
		t.Fatalf("valid portrait recording inside landscape composition rejected: %v", err)
	}
}

func TestVideoManifestRequiresExplicitManualColorsWhenSampleIsSkipped(t *testing.T) {
	m := validManualLandscapeVideoManifest()
	if err := validateVideoManifest(m); err != nil {
		t.Fatalf("explicit manual presentation rejected: %v", err)
	}
	m.Presentation.Browser.DesktopColor = nil
	if err := validateVideoManifest(m); err == nil {
		t.Fatal("final video without accepted sample or manual fill was accepted")
	}
}

func TestVideoSampleManifestUsesAutomaticPhoneAndBaseRevision(t *testing.T) {
	m := validManualLandscapeVideoManifest()
	m.Mode = VideoModeSample
	m.Presentation = &presentations.Config{Target: image.Pt(430, 880), Phone: &presentations.PhoneConfig{ProfileVersion: presentations.PhoneProfileVersionAuto, Appearance: presentations.AppearanceAuto}}
	m.PresentationLayout = nil
	m.Width, m.Height = 1920, 1080
	m.Scene.SurfaceWidth, m.Scene.SurfaceHeight = 430, 880
	m.ShotRevision, m.BaseShotRevision = 4, 4
	m.BaseRecipeSHA256 = digest([]byte("canonical-base-recipe"))
	m.RecipeSHA256 = digest([]byte("complete-shot-recipe"))
	if err := validateVideoManifest(m); err != nil {
		t.Fatalf("AUTO sample on base revision rejected: %v", err)
	}
	m.ShotRevision = 5
	if err := validateVideoManifest(m); err == nil {
		t.Fatal("sample job on a different revision than its base was accepted")
	}
}

func TestVideoBrowserSampleUsesPinnedManualLayoutOnSameRevision(t *testing.T) {
	m := validManualLandscapeVideoManifest()
	m.Mode = VideoModeSample
	m.BaseRecipeSHA256 = digest([]byte("canonical-browser-base"))
	m.RecipeSHA256 = digest([]byte("complete-browser-recipe"))
	if err := validateVideoManifest(m); err != nil {
		t.Fatalf("browser trim-start sample with explicit layout rejected: %v", err)
	}
	if videoResolvedFlags(m.Presentation) != ([4]bool{}) {
		t.Fatal("browser sample incorrectly claims phone-derived colors")
	}
	legacyPhone := &presentations.Config{Phone: &presentations.PhoneConfig{ProfileVersion: presentations.PhoneProfileVersion}}
	if videoResolvedFlags(legacyPhone) != ([4]bool{}) {
		t.Fatal("legacy phone profile defaults were mislabeled as sampled colors")
	}
	m.PresentationLayout = nil
	if err := validateVideoManifest(m); err == nil {
		t.Fatal("browser sample without a pinned manual layout was accepted")
	}
}

func TestReadAndVerifySampleReceiptRejectsMissingBinding(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	manifest := VideoManifest{Mode: VideoModeFinal}
	var gotErr error
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				gotErr = errors.New("readAndVerifySampleReceipt panicked for a missing sample binding")
			}
		}()
		_, gotErr = readAndVerifySampleReceipt(context.Background(), root, manifest)
	}()
	if gotErr == nil || gotErr.Error() != "final video has no accepted sample binding" {
		t.Fatalf("missing sample binding error = %v, want a clear validation error", gotErr)
	}
}

func TestVideoSamplePlannerUsesPinnedClockAtNonzeroTrim(t *testing.T) {
	m := validManualLandscapeVideoManifest()
	m.Inspection.FrameCount = 4
	m.Inspection.SourceEndPTS = 40
	m.Inspection.LastFrameDurationTicks = 10
	m.Inspection.TimeBase = composition.Rational{Num: 1, Den: 24}
	m.TrimStartPTS, m.TrimEndPTS = 15, 35
	m.PlaybackRate = composition.Rational{Num: 2, Den: 1}
	m.OutputRate = composition.Rational{Num: 24, Den: 1}
	clock, err := buildInspectionClock([]int64{0, 10, 20, 30})
	if err != nil {
		t.Fatal(err)
	}
	ordinal, pts, err := PlanVideoSample(context.Background(), m, clock)
	if err != nil {
		t.Fatal(err)
	}
	if ordinal != 1 || pts != 10 {
		t.Fatalf("sample source=(ordinal %d, PTS %d), want latest frame at trim start=(1,10)", ordinal, pts)
	}
	if _, _, err = PlanVideoSample(context.Background(), m, append(clock, 0)); err == nil {
		t.Fatal("sample planner accepted a clock with altered size")
	}

	changed := m.Inspection
	changed.SelectedStream++
	if VideoInspectionDigest(changed) == VideoInspectionDigest(m.Inspection) {
		t.Fatal("inspection identity digest omitted selected stream")
	}
}

func TestVideoSamplePlannerUsesAcceptedPhoneInspectionClock(t *testing.T) {
	clockPath := "testdata/retained/phone-source-clock.v1.bin"
	clockBytes, err := os.ReadFile(clockPath)
	if err != nil {
		t.Fatal(err)
	}
	m := validManualLandscapeVideoManifest()
	m.Inspection.Width, m.Inspection.Height = 430, 880
	m.Inspection.FrameCount = 488
	m.Inspection.TimeBase = composition.Rational{Num: 1, Den: 15360}
	m.Inspection.SourceEndPTS = 249856
	m.Inspection.LastFrameDurationTicks = 512
	m.TrimStartPTS, m.TrimEndPTS = 15360, 46080
	m.PlaybackRate, m.OutputRate = composition.Rational{Num: 2, Den: 1}, composition.Rational{Num: 30, Den: 1}
	ordinal, pts, err := PlanVideoSample(context.Background(), m, clockBytes)
	if err != nil {
		t.Fatal(err)
	}
	if ordinal != 30 || pts != 15360 {
		t.Fatalf("accepted phone trim-start sample=(%d,%d), want (ordinal 30, PTS 15360)", ordinal, pts)
	}
}

func TestVideoSampleBindingSeparatesFullRecipeAndBaseRecipeDigests(t *testing.T) {
	m := validManualLandscapeVideoManifest()
	m.Mode = VideoModeFinal
	m.ShotRevision, m.BaseShotRevision = 4, 3
	m.BaseRecipeSHA256 = digest([]byte("canonical-base-recipe"))
	m.RecipeSHA256 = digest([]byte("complete-final-recipe"))
	fill := presentations.RGB{R: 24, G: 28, B: 38}
	ink := presentations.RGB{R: 255, G: 255, B: 255}
	m.Presentation = &presentations.Config{Target: image.Pt(430, 880), Phone: &presentations.PhoneConfig{
		ProfileVersion: presentations.PhoneProfileVersionAuto, Appearance: presentations.AppearanceAuto,
		TopFill: &fill, BottomFill: &fill, StatusInk: &ink, HomeInk: &ink,
	}}
	layout, err := presentations.BuildLayoutForVideoRaster(image.Rect(0, 0, m.Inspection.Width, m.Inspection.Height), *m.Presentation)
	if err != nil {
		t.Fatal(err)
	}
	m.PresentationLayout = &layout
	m.Sample = &VideoSampleBinding{
		JobID: "sample-job", AttemptID: "sample-attempt", ReceiptSHA256: digest([]byte("receipt")),
		ManifestSHA256: digest([]byte("sample-manifest")), InspectionSHA256: VideoInspectionDigest(m.Inspection),
		RecipeSHA256: digest([]byte("complete-sample-recipe")), SourceSHA256: m.Inspection.Source.SHA256,
		ClockSHA256: m.Inspection.Clock.SHA256, TimingSHA256: VideoTimingDigest(m), ShotRevision: 3, BaseShotRevision: 3,
		BaseRecipeVersion: m.BaseRecipeVersion, BaseRecipeSHA256: m.BaseRecipeSHA256, ResolvedTopFill: fill, ResolvedBottomFill: fill,
		ResolvedStatusInk: &ink, ResolvedHomeInk: &ink, TopFillSourceResolved: true, BottomFillSourceResolved: true,
		StatusInkSourceResolved: true, HomeInkSourceResolved: true, FirstFrameRGBSHA256: digest([]byte("pixels")),
		PixelProfile: "srgb-rgb24-v1", Width: m.Width, Height: m.Height,
		ReceiptArtifact: VideoBinding{Name: "sample-receipt.json", SHA256: digest([]byte("receipt")), SizeBytes: 20},
	}
	if m.Sample.RecipeSHA256 == m.BaseRecipeSHA256 {
		t.Fatal("test setup must use distinct full-recipe and base-recipe digests")
	}
	if err = validateVideoManifest(m); err != nil {
		t.Fatalf("sampled final with distinct complete/base recipe digests rejected: %v", err)
	}
	m.Sample.BaseRecipeSHA256 = digest([]byte("wrong-base"))
	if err = validateVideoManifest(m); err == nil {
		t.Fatal("sample binding with a different base recipe digest was accepted")
	}
}

func validManualLandscapeVideoManifest() VideoManifest {
	d := strings.Repeat("a", 64)
	plate := VideoBinding{Name: "plate.png", SHA256: d, SizeBytes: 16}
	active := VideoBinding{Name: "active.png", SHA256: d, SizeBytes: 16}
	fill := presentations.RGB{R: 24, G: 28, B: 38}
	presentation := &presentations.Config{Target: image.Pt(430, 880), Browser: &presentations.BrowserConfig{DesktopColor: &fill}}
	layout, err := presentations.BuildLayoutForVideoRaster(image.Rect(0, 0, 1440, 800), *presentation)
	if err != nil {
		panic(err)
	}
	return VideoManifest{
		Version: videoManifestVersion, ProtocolVersion: videoProtocolVersion, JobID: "video-job", AttemptID: "attempt-1", InputFingerprint: d, WorkerSHA256: d,
		Mode: VideoModeFinal, VideoProfile: VideoProfileID, CompositorProfile: VideoCompositorID,
		Inspection:   VideoInspection{Source: VideoBinding{Name: "source.media", SHA256: d, SizeBytes: 100}, Manifest: VideoBinding{Name: AcceptedInspectionManifestName, SHA256: d, SizeBytes: 100}, Receipt: VideoBinding{Name: InspectionReceiptName, SHA256: d, SizeBytes: 100}, Clock: VideoBinding{Name: InspectionClockName, SHA256: d, SizeBytes: 100}, ProfileID: inspectionProfileID, ProfileDigest: d, ProtocolVersion: inspectionProtocol, TimeBase: composition.Rational{Num: 1, Den: 24}, SourceEndPTS: 240, LastFrameDurationTicks: 10, EOFPolicy: "last_frame_duration", FrameCount: 230, Width: 1440, Height: 800, ColorPolicy: inspectionColorPolicy},
		TrimStartPTS: 0, TrimEndPTS: 240, PlaybackRate: composition.Rational{Num: 1, Den: 1}, OutputRate: composition.Rational{Num: 24, Den: 1}, Width: 1920, Height: 1080,
		Scene:              VideoScene{Plate: plate, ActiveMask: active, SurfaceWidth: 430, SurfaceHeight: 880, Quad: [4]Corner{{X: 10, Y: 10}, {X: 20, Y: 10}, {X: 20, Y: 30}, {X: 10, Y: 30}}, Samples: 1},
		Presentation:       presentation,
		PresentationLayout: &layout,
		ShotRevision:       3, BaseShotRevision: 3, BaseRecipeVersion: "device-studio-presentation-v2", BaseRecipeSHA256: d, RecipeSHA256: d,
	}
}

func TestVerifyVideoArtifactsBindsNativeDimensionsAndMP4Prefix(t *testing.T) {
	manifest := validManualLandscapeVideoManifest()
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	output := []byte{0, 0, 0, 16, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm', 0, 0, 0, 0, 0, 0, 0, 0}
	frames, err := plannedVideoFrameCount(manifest)
	if err != nil {
		t.Fatal(err)
	}
	pts, err := plannedOutputPTS(manifest, composition.Rational{Num: 1, Den: 12288}, int(frames))
	if err != nil {
		t.Fatal(err)
	}
	outputHash := sha256.Sum256(output)
	d := strings.Repeat("a", 64)
	receipt := VideoReceipt{
		Version: videoManifestVersion, ProtocolVersion: videoProtocolVersion, JobID: manifest.JobID, AttemptID: manifest.AttemptID,
		InputFingerprint: manifest.InputFingerprint, ShotRevision: manifest.ShotRevision, BaseShotRevision: manifest.BaseShotRevision,
		BaseRecipeVersion: manifest.BaseRecipeVersion, BaseRecipeSHA256: manifest.BaseRecipeSHA256, RecipeSHA256: manifest.RecipeSHA256,
		ManifestSHA256: digest(manifestBytes), InspectionSHA256: VideoInspectionDigest(manifest.Inspection), WorkerSHA256: manifest.WorkerSHA256, Mode: VideoModeFinal,
		OutputName: VideoOutputName, OutputSHA256: hex.EncodeToString(outputHash[:]), OutputSizeBytes: int64(len(output)),
		OutputWidth: manifest.Width, OutputHeight: manifest.Height, FirstFrameRGBSHA256: d, PixelProfile: "srgb-rgb24-v1",
		PresentationLayout: manifest.PresentationLayout, Native: VideoNativeFacts{
			Container: "mov,mp4,m4a,3gp,3g2,mj2", Codec: "h264", PixelFormat: "yuv420p", Width: manifest.Width, Height: manifest.Height,
			FrameRate: manifest.OutputRate, TimeBase: composition.Rational{Num: 1, Den: 12288}, FrameCount: uint32(frames),
			FirstPTS: pts[0], LastPTS: pts[len(pts)-1], ColorPrimaries: "bt709", ColorTransfer: "bt709", ColorMatrix: "bt709", ColorRange: "tv", FullDecodeVerified: true,
		},
	}
	receiptBytes, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = VerifyVideoArtifacts(context.Background(), manifestBytes, receiptBytes, bytes.NewReader(output), int64(len(output))); err != nil {
		t.Fatalf("valid bounded MP4 receipt rejected: %v", err)
	}

	wrongDimensions := receipt
	wrongDimensions.Native.Width++
	wrongBytes, _ := json.Marshal(wrongDimensions)
	if _, err = VerifyVideoArtifacts(context.Background(), manifestBytes, wrongBytes, bytes.NewReader(output), int64(len(output))); err == nil {
		t.Fatal("receipt with native raster dimensions different from manifest was accepted")
	}
	if _, err = VerifyVideoArtifacts(context.Background(), manifestBytes, receiptBytes, bytes.NewReader([]byte("not an mp4 payload at all")), int64(len(output))); err == nil {
		t.Fatal("self-consistent receipt accepted an output without an MP4 ftyp header")
	}
}

func TestVideoPixelDigestIncludesDimensionsAndProfile(t *testing.T) {
	pixels := []byte{1, 2, 3, 4, 5, 6}
	want := videoPixelDigestForTest(2, 1, "srgb-rgb24-v1", pixels)
	if got := canonicalVideoPixelDigest(2, 1, "srgb-rgb24-v1", pixels); got != want {
		t.Fatalf("canonical pixel digest=%s want=%s", got, want)
	}
	if canonicalVideoPixelDigest(1, 2, "srgb-rgb24-v1", pixels) == want || canonicalVideoPixelDigest(2, 1, "other", pixels) == want {
		t.Fatal("canonical pixel digest omitted its raster or pixel profile")
	}
}

func videoPixelDigestForTest(width, height int, profile string, pixels []byte) string {
	// Exercise the canonical header representation independently from helper
	// call sites while keeping the expected byte form explicit.
	data := make([]byte, 12+len(profile)+len(pixels))
	data[0], data[1], data[2], data[3] = byte(width>>24), byte(width>>16), byte(width>>8), byte(width)
	data[4], data[5], data[6], data[7] = byte(height>>24), byte(height>>16), byte(height>>8), byte(height)
	data[8], data[9], data[10], data[11] = byte(len(profile)>>24), byte(len(profile)>>16), byte(len(profile)>>8), byte(len(profile))
	copy(data[12:], profile)
	copy(data[12+len(profile):], pixels)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
