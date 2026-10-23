package rendering

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"device-studio/internal/composition"
	"device-studio/internal/presentations"
)

func validPhysicalRecordingManifest(t *testing.T) PhysicalRenderManifest {
	t.Helper()
	m := validPhysicalManifest(t)
	m.Version = PhysicalRecordingManifestVersion
	m.Output.ProfileID = PhysicalRecordingMotionOutput
	m.Output.ProfileVersion = 1
	m.Output.MotionProfile = PhysicalRecordingMotionProfile
	m.Output.Rate = composition.Rational{Num: 24, Den: 1}
	m.Output.VideoToolProfileID = PhysicalMotionVideoProfileID
	m.Output.VideoToolArtifactID = PhysicalMotionVideoArtifactID
	m.Profile.Samples = PhysicalMotionSamples
	m.Profile.WallTimeSeconds = PhysicalRecordingWallTimeSeconds
	m.Profile.DigestSHA256, _ = PhysicalRenderProfileDigest(m.Profile)
	fill := presentations.RGB{R: 30, G: 40, B: 50}
	config := presentations.Config{Target: image.Pt(1206, 2622), Browser: &presentations.BrowserConfig{DesktopColor: &fill}}
	layout, err := presentations.BuildLayoutForVideoRaster(image.Rect(0, 0, 1440, 800), config)
	if err != nil {
		t.Fatal(err)
	}
	configBytes, _ := json.Marshal(config)
	layoutBytes, _ := json.Marshal(layout)
	d := testDigest('a')
	inspection := VideoInspection{
		Source:    VideoBinding{Name: "source.media", SHA256: testDigest('6'), SizeBytes: 4096},
		Manifest:  VideoBinding{Name: AcceptedInspectionManifestName, SHA256: d, SizeBytes: 128},
		Receipt:   VideoBinding{Name: InspectionReceiptName, SHA256: testDigest('b'), SizeBytes: 256},
		Clock:     VideoBinding{Name: InspectionClockName, SHA256: testDigest('c'), SizeBytes: 4096},
		ProfileID: inspectionProfileID, ProfileDigest: testDigest('d'), ProtocolVersion: inspectionProtocol,
		TimeBase: composition.Rational{Num: 1, Den: 24}, StreamStartPTS: 0, SourceEndPTS: 240, LastFrameDurationTicks: 10,
		EOFPolicy: "last_frame_duration", FrameCount: 240, Width: 1440, Height: 800, ColorPolicy: inspectionColorPolicy,
	}
	m.Screen.Raster = ImageBinding{}
	m.Screen.RasterSHA256 = ""
	m.Screen.OriginalMediaSHA256 = inspection.Source.SHA256
	m.Screen.Width, m.Screen.Height = config.Target.X, config.Target.Y
	m.Screen.PresentationProfile, m.Screen.PresentationVersion = layout.ProfileID, layout.ProfileVersion
	m.Screen.ConfigSHA256 = digest(configBytes)
	m.Screen.LayoutSHA256 = digest(layoutBytes)
	m.Screen.Recording = &PhysicalRecordingInput{Inspection: inspection, TrimStartPTS: 0, TrimEndPTS: 2, PlaybackRate: composition.Rational{Num: 1, Den: 1}, AudioPolicy: PhysicalRecordingAudioUnspecified, SourcePrefixFrameCount: 2, Presentation: config, PresentationLayout: layout}
	m.Output.FrameCount = 2
	m.Output.Samples = m.Output.Samples[:2]
	return m
}

func TestPhysicalRecordingManifestUsesBoundedV4ProfileAndPreservesLegacy(t *testing.T) {
	m := validPhysicalRecordingManifest(t)
	if err := ValidatePhysicalRenderManifest(m); err != nil {
		t.Fatalf("valid physical recording rejected: %v", err)
	}
	if _, _, err := CanonicalPhysicalRenderManifest(m); err != nil {
		t.Fatalf("recording manifest did not canonicalize: %v", err)
	}
	legacy := validPhysicalManifest(t)
	if _, _, err := CanonicalPhysicalRenderManifest(legacy); err != nil {
		t.Fatalf("legacy v2 motion changed: %v", err)
	}
	for name, edit := range map[string]func(*PhysicalRenderManifest){
		"passthrough rejected": func(m *PhysicalRenderManifest) { m.Screen.Recording.AudioPolicy = PhysicalRecordingAudioPassthrough },
		"prefix outside source clock": func(m *PhysicalRenderManifest) {
			m.Screen.Recording.SourcePrefixFrameCount = m.Screen.Recording.Inspection.FrameCount + 1
		},
		"oversized prefix": func(m *PhysicalRenderManifest) {
			m.Screen.Recording.SourcePrefixFrameCount = PhysicalRecordingMaxSourcePrefix + 1
		},
		"raster cannot be both static and live": func(m *PhysicalRenderManifest) {
			m.Screen.Raster = ImageBinding{Name: "screen.png", SHA256: testDigest('7'), SizeBytes: 4}
		},
		"wrong protocol version": func(m *PhysicalRenderManifest) { m.Version = PhysicalMotionManifestVersion },
		"wrong output kind":      func(m *PhysicalRenderManifest) { m.Output.Kind = "still_png" },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := validPhysicalRecordingManifest(t)
			edit(&candidate)
			candidate.Profile.DigestSHA256, _ = PhysicalRenderProfileDigest(candidate.Profile)
			if err := ValidatePhysicalRenderManifest(candidate); err == nil {
				t.Fatal("invalid recording closure passed")
			}
		})
	}
}

func TestPhysicalRecordingDiagnosticMotionIsLowSampleAndKeepsSavedClock(t *testing.T) {
	m := validPhysicalRecordingManifest(t)
	m.Output.ProfileID = PhysicalDiagnosticMotionOutput
	m.Output.MotionProfile = PhysicalDiagnosticMotionProfile
	m.Profile.Samples = PhysicalDiagnosticMotionSamples
	m.Profile.WallTimeSeconds = PhysicalDiagnosticMotionWallTimeSeconds
	m.Profile.DigestSHA256, _ = PhysicalRenderProfileDigest(m.Profile)
	if err := ValidatePhysicalRenderManifest(m); err != nil {
		t.Fatalf("valid low-sample recording diagnostic rejected: %v", err)
	}
	if m.Output.FrameCount != 2 || m.Output.Samples[1].RequestedMotionTick != (composition.Rational{Num: 1, Den: 1}) || m.Screen.Recording.TrimEndPTS != 2 {
		t.Fatalf("diagnostic changed the exact source/motion clock: output=%+v recording=%+v", m.Output, m.Screen.Recording)
	}
	m.Profile.Samples = PhysicalMotionSamples
	m.Profile.DigestSHA256, _ = PhysicalRenderProfileDigest(m.Profile)
	if err := ValidatePhysicalRenderManifest(m); err == nil {
		t.Fatal("diagnostic silently accepted the delivery-quality sample count")
	}
}

func TestDerivePhysicalRecordingScheduleUsesVerifiedClockAndExactPrefix(t *testing.T) {
	recording, rate, count, manifestBytes, receiptBytes, clockBytes := verifiedPhysicalRecordingInputs(t, []int64{0, 1, 2}, 1, composition.Rational{Num: 1, Den: 30}, 0, 2, 24, 2)
	facts, err := DerivePhysicalRecordingSchedule(context.Background(), recording, rate, count, manifestBytes, receiptBytes, clockBytes)
	if err != nil || len(facts.Samples) != 2 || facts.RequiredSourcePrefixFrameCount != 2 {
		t.Fatalf("valid accepted schedule = %+v, %v", facts, err)
	}
	recording.SourcePrefixFrameCount = 1
	if _, err := DerivePhysicalRecordingSchedule(context.Background(), recording, rate, count, manifestBytes, receiptBytes, clockBytes); err == nil {
		t.Fatal("incorrect predeclared source prefix passed")
	}
	recording.SourcePrefixFrameCount = 0
	recording.AudioPolicy = PhysicalRecordingAudioPassthrough
	if _, err := DerivePhysicalRecordingSchedule(context.Background(), recording, rate, count, manifestBytes, receiptBytes, clockBytes); err == nil {
		t.Fatal("unsupported passthrough audio passed schedule preflight")
	}
	recording.AudioPolicy = PhysicalRecordingAudioSilent
	recording.TrimEndPTS = 4
	if _, err := DerivePhysicalRecordingSchedule(context.Background(), recording, rate, count, manifestBytes, receiptBytes, clockBytes); err == nil {
		t.Fatal("trim beyond accepted source EOF passed schedule preflight")
	}
	recording.TrimEndPTS = 2
	if _, err := DerivePhysicalRecordingSchedule(context.Background(), recording, rate, count+1, manifestBytes, receiptBytes, clockBytes); err == nil {
		t.Fatal("caller output count inconsistent with rational schedule passed")
	}
	changedClock := append([]byte(nil), clockBytes...)
	changedClock[len(changedClock)-1]++
	if _, err := DerivePhysicalRecordingSchedule(context.Background(), recording, rate, count, manifestBytes, receiptBytes, changedClock); err == nil {
		t.Fatal("clock bytes not matching accepted receipt passed")
	}
}

func TestDerivePhysicalRecordingScheduleRejectsHugeDurationBeforePlanning(t *testing.T) {
	// The inspected source is exactly at its permitted 600-second ceiling, which
	// would plan 14,400 output frames at 24 fps absent the physical profile guard.
	recording, rate, _, manifestBytes, receiptBytes, clockBytes := verifiedPhysicalRecordingInputs(t, []int64{0, 1, 2}, 17_998, composition.Rational{Num: 1, Den: 30}, 0, 18_000, 24, 1)
	if _, err := DerivePhysicalRecordingSchedule(context.Background(), recording, rate, 1, manifestBytes, receiptBytes, clockBytes); err == nil || !strings.Contains(err.Error(), "bounded output profile") {
		t.Fatalf("huge actual duration with tiny declared output count was not rejected before planning: %v", err)
	}
}

func TestDerivePhysicalRecordingScheduleEnforces300And3600Boundaries(t *testing.T) {
	for _, rate := range []int64{24, 30} {
		for _, tc := range []struct {
			name       string
			frames     int
			wantPrefix uint32
			lastTicks  int64
			wantErr    bool
		}{
			{name: "exact prefix ceiling", frames: 3600, wantPrefix: 3600, lastTicks: map[bool]int64{true: 21, false: 14}[rate == 24]},
			{name: "prefix ceiling plus one", frames: 3601, lastTicks: map[bool]int64{true: 20, false: 13}[rate == 24], wantErr: true},
		} {
			t.Run(fmt.Sprintf("%dfps/%s", rate, tc.name), func(t *testing.T) {
				pts := make([]int64, tc.frames)
				for i := range pts {
					pts[i] = int64(i)
				}
				timeBase := composition.Rational{Num: 10, Den: 3613}
				if rate == 24 {
					timeBase = composition.Rational{Num: 1, Den: 362}
				}
				recording, outputRate, count, manifestBytes, receiptBytes, clockBytes := verifiedPhysicalRecordingInputs(t, pts, tc.lastTicks, timeBase, 0, 10*int64(timeBase.Den)/timeBase.Num, rate, uint32(rate*10))
				facts, err := DerivePhysicalRecordingSchedule(context.Background(), recording, outputRate, count, manifestBytes, receiptBytes, clockBytes)
				if tc.wantErr {
					if err == nil {
						t.Fatal("schedule above required-prefix ceiling passed")
					}
					return
				}
				if err != nil || len(facts.Samples) != int(rate*10) || facts.RequiredSourcePrefixFrameCount != tc.wantPrefix {
					t.Fatalf("ten-second %dfps ceiling schedule = %d frames, prefix %d, err %v", rate, len(facts.Samples), facts.RequiredSourcePrefixFrameCount, err)
				}
			})
		}
	}
	if _, err := DerivePhysicalRecordingSchedule(context.Background(), PhysicalRecordingInput{}, composition.Rational{Num: 30, Den: 1}, 301, nil, nil, nil); err == nil {
		t.Fatal("output schedule above 300 frames passed")
	}
}

func verifiedPhysicalRecordingInputs(t *testing.T, pts []int64, lastDuration int64, timeBase composition.Rational, trimStart, trimEnd, outputRate int64, outputCount uint32) (PhysicalRecordingInput, composition.Rational, uint32, []byte, []byte, []byte) {
	t.Helper()
	manifestBytes, receiptBytes, _ := validInspectionArtifacts(t)
	var inspectionManifest InspectionManifest
	var receipt InspectionReceipt
	if err := json.Unmarshal(manifestBytes, &inspectionManifest); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(receiptBytes, &receipt); err != nil {
		t.Fatal(err)
	}
	clockBytes, err := buildInspectionClock(pts)
	if err != nil {
		t.Fatal(err)
	}
	receipt.TimeBaseNum, receipt.TimeBaseDen = timeBase.Num, timeBase.Den
	receipt.FrameCount = uint32(len(pts))
	start, end := pts[0], pts[len(pts)-1]+lastDuration
	receipt.StreamStartPTS, receipt.SourceEndPTS = &start, &end
	receipt.LastFrameDurationTicks = lastDuration
	durationNum, durationDen, ok := exactDuration(timeBase.Num, timeBase.Den, start, end)
	if !ok {
		t.Fatal("fixture duration is not representable")
	}
	receipt.DurationNumerator, receipt.DurationDenominator = durationNum, durationDen
	receipt.ClockArtifactSizeBytes, receipt.ClockSHA256 = int64(len(clockBytes)), digest(clockBytes)
	receiptBytes, err = json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	base := validPhysicalRecordingManifest(t)
	recording := *base.Screen.Recording
	recording.Inspection.Source.SHA256, recording.Inspection.Source.SizeBytes = inspectionManifest.Source.SHA256, inspectionManifest.Source.SizeBytes
	recording.Inspection.Manifest.SizeBytes, recording.Inspection.Manifest.SHA256 = int64(len(manifestBytes)), digest(manifestBytes)
	recording.Inspection.Receipt.SizeBytes, recording.Inspection.Receipt.SHA256 = int64(len(receiptBytes)), digest(receiptBytes)
	recording.Inspection.Clock.SizeBytes, recording.Inspection.Clock.SHA256 = int64(len(clockBytes)), digest(clockBytes)
	recording.Inspection.TimeBase, recording.Inspection.StreamStartPTS, recording.Inspection.SourceEndPTS = timeBase, start, end
	recording.Inspection.LastFrameDurationTicks, recording.Inspection.FrameCount = lastDuration, uint32(len(pts))
	recording.Inspection.ProfileID, recording.Inspection.ProfileDigest, recording.Inspection.ProtocolVersion = receipt.ProfileID, receipt.ProfileDigest, receipt.ProtocolVersion
	recording.Inspection.SelectedStream = *receipt.VideoStreamIndex
	recording.Inspection.Width, recording.Inspection.Height = int(receipt.DisplayWidth), int(receipt.DisplayHeight)
	recording.Inspection.ColorPrimaries, recording.Inspection.ColorTransfer = receipt.ColorPrimaries, receipt.ColorTransfer
	recording.Inspection.ColorMatrix, recording.Inspection.ColorRange, recording.Inspection.ColorPolicy = receipt.ColorMatrix, receipt.ColorRange, receipt.ColorPolicy
	recording.TrimStartPTS, recording.TrimEndPTS = trimStart, trimEnd
	recording.PlaybackRate, recording.AudioPolicy, recording.SourcePrefixFrameCount = composition.Rational{Num: 1, Den: 1}, PhysicalRecordingAudioUnspecified, 0
	return recording, composition.Rational{Num: outputRate, Den: 1}, outputCount, manifestBytes, receiptBytes, clockBytes
}

func TestPhysicalRecordingOutputCeilingIsTenSecondsAt24And30FPS(t *testing.T) {
	for _, rate := range []int64{24, 30} {
		m := validPhysicalRecordingManifest(t)
		m.Output.Rate = composition.Rational{Num: rate, Den: 1}
		count := int(rate * 10)
		m.Output.FrameCount = uint32(count)
		m.Output.Samples = make([]PhysicalMotionSample, count)
		for i := 0; i < count; i++ {
			m.Output.Samples[i] = physicalScheduleSample(t, uint32(i), composition.Rational{Num: int64(i), Den: rate}, m.Output.Timebase)
		}
		if err := validatePhysicalOutput(m.Output); err != nil {
			t.Fatalf("%dfps ten-second schedule rejected: %v", rate, err)
		}
		m.Output.FrameCount++
		if err := validatePhysicalOutput(m.Output); err == nil {
			t.Fatalf("%dfps schedule above ten seconds passed", rate)
		}
	}
	if !strings.Contains(PhysicalRecordingMotionOutput, "recording") {
		t.Fatal("recording output profile identity is not distinct")
	}
}

func TestPhysicalRecordingRGB24AndSourceReceiptFactsBindExactSchedule(t *testing.T) {
	img, err := physicalRGB24Image([]byte{1, 2, 3, 4, 5, 6}, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got := img.NRGBAAt(0, 0); got != (color.NRGBA{R: 1, G: 2, B: 3, A: 255}) {
		t.Fatalf("first RGB pixel = %#v", got)
	}
	if got := img.NRGBAAt(1, 0); got != (color.NRGBA{R: 4, G: 5, B: 6, A: 255}) {
		t.Fatalf("second RGB pixel = %#v", got)
	}
	if _, err := physicalRGB24Image([]byte{1, 2, 3}, 2, 1); err == nil {
		t.Fatal("truncated RGB24 input was accepted")
	}
	r := validPhysicalRecordingManifest(t).Screen.Recording
	plan := []composition.FrameSample{{SourceIndex: 0, SourcePTS: 10}, {SourceIndex: 1, SourcePTS: 20}, {SourceIndex: 1, SourcePTS: 20}}
	r.SourcePrefixFrameCount = 2
	facts := []PhysicalRecordingFrameFact{{OutputOrdinal: 0, SourceOrdinal: 0, SourcePTS: 10, ScreenSHA256: testDigest('1')}, {OutputOrdinal: 1, SourceOrdinal: 1, SourcePTS: 20, ScreenSHA256: testDigest('2')}, {OutputOrdinal: 2, SourceOrdinal: 1, SourcePTS: 20, ScreenSHA256: testDigest('3')}}
	if err := validatePhysicalRecordingSchedule(r, plan, facts); err != nil {
		t.Fatalf("valid repeated-source schedule rejected: %v", err)
	}
	facts[2].SourcePTS++
	if err := validatePhysicalRecordingSchedule(r, plan, facts); err == nil {
		t.Fatal("tampered source PTS passed the output receipt schedule")
	}
	facts[2].SourcePTS = 20
	facts[1].OutputOrdinal = 2
	if err := validatePhysicalRecordingSchedule(r, plan, facts); err == nil {
		t.Fatal("reordered output facts passed")
	}
}

func TestPhysicalRecordingComposedScreenPNGUsesSupportedTextureEncoding(t *testing.T) {
	fill := presentations.RGB{R: 24, G: 36, B: 48}
	config := presentations.Config{Target: image.Pt(1440, 900), Browser: &presentations.BrowserConfig{
		Window:      presentations.BasisRect{X: 800, Y: 800, Width: 8400, Height: 8000},
		ShowToolbar: true, DesktopColor: &fill,
	}}
	source := image.NewNRGBA(image.Rect(0, 0, 32, 18))
	for i := 0; i < len(source.Pix); i += 4 {
		source.Pix[i], source.Pix[i+1], source.Pix[i+2], source.Pix[i+3] = 80, 100, 120, 255
	}
	composed, err := presentations.Compose(context.Background(), source, config)
	if err != nil {
		t.Fatal(err)
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, composed.Raster); err != nil {
		t.Fatal(err)
	}
	data := encoded.Bytes()
	if data[25] != 2 {
		t.Fatalf("opaque Go composition encoded PNG color type %d, expected RGB type 2", data[25])
	}
	if !isPhysicalScreenTexturePNG(data) {
		t.Fatal("actual composed screen PNG was rejected by the recording texture contract")
	}
	if isPhysicalRGBA8PNG(data) {
		t.Fatal("RGB screen texture was mistaken for the strict RGBA render-output contract")
	}
}

func TestPhysicalStillReceiptRejectsRecordingFrameFacts(t *testing.T) {
	output := PhysicalOutputReceipt{SourceFrameFacts: []PhysicalRecordingFrameFact{{
		OutputOrdinal: 0, SourceOrdinal: 0, SourcePTS: 12, ScreenSHA256: testDigest('a'),
	}}}
	if err := validatePhysicalOutputKindFacts("still_png", output); err == nil {
		t.Fatal("still output accepted recording source-frame facts")
	}
	if err := validatePhysicalOutputKindFacts("still_png", PhysicalOutputReceipt{}); err != nil {
		t.Fatalf("ordinary still receipt rejected: %v", err)
	}
}

func TestPhysicalRecordingFramePipeRoundTripAndRejectsTruncation(t *testing.T) {
	readEnd, writeEnd, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stream := &physicalRecordingBlenderStream{cmd: &exec.Cmd{}, inW: writeEnd, expected: 3}
	frame := []byte("bounded PNG payload")
	if err := stream.send(1, frame); err != nil {
		t.Fatal(err)
	}
	if err := stream.inW.Close(); err != nil {
		t.Fatal(err)
	}
	stream.inW = nil
	ordinal, got, err := readPhysicalFrame(readEnd)
	if err != nil || ordinal != 1 || string(got) != string(frame) {
		t.Fatalf("framed pipe round trip = (%d, %q, %v)", ordinal, got, err)
	}
	_ = readEnd.Close()

	var oversized [12]byte
	binary.BigEndian.PutUint32(oversized[:4], 1)
	binary.BigEndian.PutUint64(oversized[4:], uint64(physicalMotionFrameMax+1))
	for name, input := range map[string][]byte{
		"short header":    make([]byte, 11),
		"short payload":   append(append([]byte(nil), oversized[:4]...), 0, 0, 0, 0, 0, 0, 0, 3, 'p'),
		"oversized frame": oversized[:],
	} {
		t.Run(name, func(t *testing.T) {
			_, _, err := readPhysicalFrame(strings.NewReader(string(input)))
			if err == nil || err == io.EOF && name != "short header" {
				t.Fatalf("malformed frame accepted: %v", err)
			}
		})
	}
}

func TestPhysicalRecordingAbortKillsAndReapsChild(t *testing.T) {
	if os.Getenv("GO_WANT_PHYSICAL_RECORDING_ABORT_HELPER") == "1" {
		select {}
	}
	readEnd, outWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	inReader, inWriter, err := os.Pipe()
	if err != nil {
		_ = readEnd.Close()
		_ = outWriter.Close()
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestPhysicalRecordingAbortKillsAndReapsChild$")
	cmd.Env = append(os.Environ(), "GO_WANT_PHYSICAL_RECORDING_ABORT_HELPER=1")
	cmd.Stdin = inReader
	cmd.Stdout = outWriter
	if err := cmd.Start(); err != nil {
		_ = readEnd.Close()
		_ = outWriter.Close()
		_ = inReader.Close()
		_ = inWriter.Close()
		t.Fatal(err)
	}
	_ = outWriter.Close()
	_ = inReader.Close()
	stream := &physicalRecordingBlenderStream{ctx: context.Background(), cmd: cmd, outR: readEnd, inW: inWriter}
	stream.abort()
	if !stream.waited || cmd.ProcessState == nil {
		t.Fatal("recording Blender child was not synchronously reaped")
	}
	if stream.cmd != nil && stream.cmd.ProcessState == nil {
		t.Fatal("aborted recording child has no terminal process state")
	}
	stream.abort() // cleanup remains safe if a caller's defer also runs.
}
