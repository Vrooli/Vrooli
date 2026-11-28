package rendering

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vrooli/repo-contract-go/cliinvoke"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestResolveInspectionProfileUsesFixedOwnerCommandAndTypedProfile(t *testing.T) {
	cli := filepath.Join(t.TempDir(), "vrooli")
	if err := os.WriteFile(cli, []byte("test cli placeholder"), 0o600); err != nil {
		t.Fatal(err)
	}
	profile := validInspectionProfile()
	encoded, err := protojson.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	called := false
	got, err := resolveInspectionProfile(context.Background(), func(key string) string {
		if key == cliinvoke.BinaryEnvVar {
			return cli
		}
		return ""
	}, t.TempDir(), func(_ context.Context, path, command string, args []string) ([]byte, error) {
		called = true
		if path != cli || command != "host" || strings.Join(args, " ") != "renderer-profile ffmpeg-video --json" {
			t.Fatalf("unexpected control-plane invocation: %q %q %q", path, command, args)
		}
		return encoded, nil
	})
	if err != nil || !called || got.GetArtifactIdentity() != profile.GetArtifactIdentity() {
		t.Fatalf("profile resolution failed: called=%t got=%v err=%v", called, got, err)
	}
}

func TestProfileIdentityMatchesRetainedHostOwnerArtifact(t *testing.T) {
	data, err := os.ReadFile("testdata/retained/ffmpeg-profile.json")
	if err != nil {
		t.Fatal(err)
	}
	profile := new(cliv1.RendererToolProfile)
	if err := protojson.Unmarshal(data, profile); err != nil {
		t.Fatal(err)
	}
	const expected = "2ad074849ce8635ce03d623865a26fdb1636a9a44346d21d4af08c1b191a9789"
	if got := profileArtifactIdentity(profile); got != expected || profile.GetArtifactIdentity() != expected {
		t.Fatalf("profile identity drifted from host owner: calculated=%s supplied=%s", got, profile.GetArtifactIdentity())
	}
}

func TestResolveInspectionProfileFailsClosedForMissingOrInvalidProfile(t *testing.T) {
	ctx := context.Background()
	missingCalls := 0
	_, err := resolveInspectionProfile(ctx, func(string) string { return "" }, t.TempDir(), func(context.Context, string, string, []string) ([]byte, error) {
		missingCalls++
		return nil, nil
	})
	if err == nil || missingCalls != 0 {
		t.Fatalf("missing CLI must not invoke PATH or provider: calls=%d err=%v", missingCalls, err)
	}

	cli := filepath.Join(t.TempDir(), "vrooli")
	if err := os.WriteFile(cli, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, response := range map[string][]byte{
		"oversized":    bytesOf(profileStdoutLimit + 1),
		"invalid json": []byte(`{"schemaVersion":1,"unknown":"field"}`),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := resolveInspectionProfile(ctx, func(string) string { return cli }, t.TempDir(), func(context.Context, string, string, []string) ([]byte, error) { return response, nil })
			if err == nil {
				t.Fatal("invalid profile response unexpectedly accepted")
			}
		})
	}
	profile := validInspectionProfile()
	profile.State = cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_PARTIAL
	encoded, _ := protojson.Marshal(profile)
	if _, err := resolveInspectionProfile(ctx, func(string) string { return cli }, t.TempDir(), func(context.Context, string, string, []string) ([]byte, error) { return encoded, nil }); err == nil {
		t.Fatal("partial tool pair unexpectedly accepted")
	}
	if _, err := resolveInspectionProfile(ctx, func(string) string { return cli }, t.TempDir(), func(context.Context, string, string, []string) ([]byte, error) {
		return nil, errors.New("owner unavailable")
	}); err == nil {
		t.Fatal("owner invocation error unexpectedly accepted")
	}
}

func TestVerifyInspectionArtifactsAcceptsBoundedExactClock(t *testing.T) {
	manifestBytes, receiptBytes, clock := validInspectionArtifacts(t)
	receipt, err := VerifyInspectionArtifacts(manifestBytes, receiptBytes, clock)
	if err != nil || receipt.Classification != InspectionAccepted || receipt.FrameCount != 3 {
		t.Fatalf("valid inspection artifacts rejected: receipt=%+v err=%v", receipt, err)
	}
}

func TestLegacyInspectionReceiptV1RemainsByteStable(t *testing.T) {
	manifestBytes, receiptBytes, clock := validInspectionArtifacts(t)
	var receipt InspectionReceipt
	if err := json.Unmarshal(receiptBytes, &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Version != 1 || receipt.Audio != nil {
		t.Fatalf("legacy receipt was normalized: version=%d audio=%+v", receipt.Version, receipt.Audio)
	}
	encoded, err := json.Marshal(receipt)
	if err != nil || !bytes.Equal(encoded, receiptBytes) {
		t.Fatalf("legacy receipt bytes changed: equal=%t err=%v", bytes.Equal(encoded, receiptBytes), err)
	}
	if _, err := VerifyInspectionArtifacts(manifestBytes, receiptBytes, clock); err != nil {
		t.Fatalf("historical receipt v1 no longer verifies: %v", err)
	}
}

func TestInspectionReceiptV2DistinguishesZeroAndObservedAudio(t *testing.T) { // [REQ:DS-SCREEN-AUDIO]
	manifestBytes, receiptBytes, clock := validInspectionArtifacts(t)
	var base InspectionReceipt
	if err := json.Unmarshal(receiptBytes, &base); err != nil {
		t.Fatal(err)
	}

	zero := base
	zero.Version = inspectionReceiptVersion
	zero.Audio = &InspectionAudioFacts{Version: 1, Streams: make([]InspectionAudioStreamFact, 0)}
	zeroBytes, err := json.Marshal(zero)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := VerifyInspectionArtifacts(manifestBytes, zeroBytes, clock)
	if err != nil || verified.Audio == nil || verified.Audio.Streams == nil || len(verified.Audio.Streams) != 0 {
		t.Fatalf("known-zero audio facts not preserved: audio=%+v err=%v", verified.Audio, err)
	}

	observed := base
	observed.Version = inspectionReceiptVersion
	observed.StreamCount = 3
	observed.Audio = &InspectionAudioFacts{Version: 1, Streams: []InspectionAudioStreamFact{{OriginalStreamIndex: 2, Codec: "aac"}, {OriginalStreamIndex: 1, Codec: "opus"}}}
	observedBytes, err := json.Marshal(observed)
	if err != nil {
		t.Fatal(err)
	}
	verified, err = VerifyInspectionArtifacts(manifestBytes, observedBytes, clock)
	if err != nil || verified.Audio == nil || len(verified.Audio.Streams) != 2 {
		t.Fatalf("observed audio facts not accepted: audio=%+v err=%v", verified.Audio, err)
	}

	mutations := map[string]func(*InspectionReceipt){
		"missing facts":       func(r *InspectionReceipt) { r.Audio = nil },
		"wrong facts version": func(r *InspectionReceipt) { r.Audio.Version = 2 },
		"null stream list":    func(r *InspectionReceipt) { r.Audio.Streams = nil },
		"duplicate index":     func(r *InspectionReceipt) { r.Audio.Streams = []InspectionAudioStreamFact{{1, "aac"}, {1, "opus"}} },
		"video index":         func(r *InspectionReceipt) { r.Audio.Streams = []InspectionAudioStreamFact{{0, "aac"}} },
		"out of range":        func(r *InspectionReceipt) { r.Audio.Streams = []InspectionAudioStreamFact{{3, "aac"}} },
		"invalid codec":       func(r *InspectionReceipt) { r.Audio.Streams = []InspectionAudioStreamFact{{1, "AAC/evil"}} },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			candidate := observed
			candidate.Audio = &InspectionAudioFacts{Version: observed.Audio.Version, Streams: append([]InspectionAudioStreamFact(nil), observed.Audio.Streams...)}
			mutate(&candidate)
			encoded, err := json.Marshal(candidate)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := VerifyInspectionArtifacts(manifestBytes, encoded, clock); err == nil {
				t.Fatal("invalid audio facts unexpectedly verified")
			}
		})
	}

	rejected := InspectionReceipt{
		Version: inspectionReceiptVersion, JobID: base.JobID, AttemptID: base.AttemptID,
		InputFingerprint: base.InputFingerprint, WorkerSHA256: base.WorkerSHA256, ManifestSHA256: base.ManifestSHA256,
		SourceSHA256: base.SourceSHA256, SourceSizeBytes: base.SourceSizeBytes, ProfileID: base.ProfileID,
		ProfileDigest: base.ProfileDigest, ProtocolVersion: base.ProtocolVersion,
		Classification: InspectionRejected, IssueCode: "audio_stream_unsupported", IssueMessage: "malformed audio metadata",
	}
	rejectedBytes, err := json.Marshal(rejected)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyInspectionArtifacts(manifestBytes, rejectedBytes, nil); err != nil {
		t.Fatalf("rejected v2 without fabricated audio facts was rejected: %v", err)
	}
	rejected.Audio = &InspectionAudioFacts{Version: 1, Streams: make([]InspectionAudioStreamFact, 0)}
	rejectedBytes, _ = json.Marshal(rejected)
	if _, err := VerifyInspectionArtifacts(manifestBytes, rejectedBytes, nil); err == nil {
		t.Fatal("rejected inspection fabricated accepted audio facts")
	}
}

func TestInspectionAudioFactsPreserveOriginalIndexesAndRejectMalformedCodec(t *testing.T) { // [REQ:DS-SCREEN-AUDIO]
	document := inspectionProbeDocument{Streams: []inspectionStream{
		{Index: 0, CodecType: "video", CodecName: "h264"},
		{Index: 1, CodecType: "audio", CodecName: "opus"},
		{Index: 2, CodecType: "audio", CodecName: "aac"},
	}}
	facts, rejection := inspectionAudioFacts(document)
	if rejection != nil || facts.Version != 1 || len(facts.Streams) != 2 || facts.Streams[0].OriginalStreamIndex != 1 || facts.Streams[1].OriginalStreamIndex != 2 {
		t.Fatalf("audio stream facts did not preserve and order original indexes: facts=%+v rejection=%+v", facts, rejection)
	}
	document.Streams[1].CodecName = ""
	if facts, rejection := inspectionAudioFacts(document); facts != nil || rejection == nil || rejection.Code != "audio_stream_unsupported" {
		t.Fatalf("malformed audio codec was not a typed rejection: facts=%+v rejection=%+v", facts, rejection)
	}
}

func TestExecInspectionRunnerRejectsStdoutBeyondLimit(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("descriptor-bound native runner is Linux-only")
	}
	echoPath, err := exec.LookPath("echo")
	if err != nil {
		t.Skip("echo executable is unavailable")
	}
	tool, err := os.Open(echoPath)
	if err != nil {
		t.Fatal(err)
	}
	defer tool.Close()
	source, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	runner := execInspectionRunner{}
	for name, run := range map[string]func() ([]byte, error){
		"probe": func() ([]byte, error) {
			return runner.Probe(context.Background(), tool, []string{strings.Repeat("x", 32)}, source, 8)
		},
		"decode": func() ([]byte, error) {
			return runner.Decode(context.Background(), tool, []string{strings.Repeat("x", 32)}, source, 8)
		},
	} {
		t.Run(name, func(t *testing.T) {
			output, err := run()
			if err == nil || output != nil {
				t.Fatalf("runner returned output despite exceeding limit: bytes=%d err=%v", len(output), err)
			}
		})
	}
}

func TestInvokeInspectionProfileRejectsStdoutBeyondLimit(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("cliinvoke profile provider regression is qualified on Linux")
	}
	cli := filepath.Join(t.TempDir(), "profile-output")
	if err := os.WriteFile(cli, []byte("#!/bin/sh\nprintf '%70000s' x\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	output, err := invokeInspectionProfile(context.Background(), cli, "host", []string{"renderer-profile", "ffmpeg-video", "--json"})
	if err == nil || output != nil {
		t.Fatalf("cliinvoke accepted profile output beyond bound: bytes=%d err=%v", len(output), err)
	}
}

func TestVerifyInspectionArtifactsRejectsTamperedClockWithUpdatedDigest(t *testing.T) {
	manifestBytes, receiptBytes, clock := validInspectionArtifacts(t)
	var base InspectionReceipt
	if err := json.Unmarshal(receiptBytes, &base); err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func([]byte){
		"header": func(data []byte) { data[0] ^= 0xff },
		"ordinal": func(data []byte) {
			binary.BigEndian.PutUint32(data[inspectionClockHeader:inspectionClockHeader+4], 1)
		},
		"PTS order": func(data []byte) {
			secondPTS := inspectionClockHeader + inspectionClockRecord + 4
			binary.BigEndian.PutUint64(data[secondPTS:secondPTS+8], 0)
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			changedClock := bytes.Clone(clock)
			mutate(changedClock)
			changedReceipt := base
			changedReceipt.ClockArtifactSizeBytes = int64(len(changedClock))
			changedReceipt.ClockSHA256 = digest(changedClock)
			encoded, err := json.Marshal(changedReceipt)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := VerifyInspectionArtifacts(manifestBytes, encoded, changedClock); err == nil {
				t.Fatal("tampered source clock accepted after receipt digest was recomputed")
			}
		})
	}
}

func TestAuthenticFFprobeFixtureUsesExactFrameDurationTicks(t *testing.T) {
	data, err := os.ReadFile("testdata/retained/ffprobe-phone-frames.json")
	if err != nil {
		t.Fatal(err)
	}
	var document inspectionFrameDocument
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if len(document.Frames) != 488 {
		t.Fatalf("selected ffprobe fixture frame count=%d, want 488", len(document.Frames))
	}
	last := document.Frames[len(document.Frames)-1]
	pts, ptsOK := parseRawInt(last.PTS)
	duration, durationOK := parseRawInt(last.DurationTicks)
	if !ptsOK || !durationOK || pts != 249344 || duration != 512 {
		t.Fatalf("real ffprobe EOF fields were not parsed exactly: pts=%d duration=%d ptsOK=%t durationOK=%t", pts, duration, ptsOK, durationOK)
	}
}

func TestRejectedInspectionDoesNotPublishAfterSourceMutation(t *testing.T) {
	workspace := t.TempDir()
	root, err := os.OpenRoot(workspace)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	sourcePath := filepath.Join(workspace, InspectionSourceName)
	if err := os.WriteFile(sourcePath, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := os.WriteFile(sourcePath, []byte("mutated!"), 0o600); err != nil {
		t.Fatal(err)
	}
	binding := InspectionSource{Name: InspectionSourceName, SHA256: digest([]byte("original")), SizeBytes: int64(len("original"))}
	err = publishInspectionRejection(context.Background(), root, f, binding, InspectionReceipt{JobID: "job-1", AttemptID: "attempt-1"}, &inspectionRejection{Code: "container_unsupported", Message: "unsupported"})
	if err == nil {
		t.Fatal("rejected receipt published despite source mutation")
	}
	if _, err := root.Lstat(InspectionReceiptName); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("rejected receipt exists after source mutation: %v", err)
	}
}

func TestVerifyInspectionArtifactsRejectsFabricatedOrInconsistentFacts(t *testing.T) {
	manifestBytes, receiptBytes, clock := validInspectionArtifacts(t)
	var good InspectionReceipt
	if err := json.Unmarshal(receiptBytes, &good); err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func(*InspectionReceipt){
		"duration":       func(r *InspectionReceipt) { r.DurationNumerator++ },
		"raster bound":   func(r *InspectionReceipt) { r.CodedWidth = inspectionMaxDimension + 1 },
		"display raster": func(r *InspectionReceipt) { r.DisplayWidth++ },
		"stream index":   func(r *InspectionReceipt) { *r.VideoStreamIndex = r.StreamCount },
		"color claim":    func(r *InspectionReceipt) { r.ColorTransfer = "unspecified" },
		"worker binding": func(r *InspectionReceipt) { r.WorkerSHA256 = strings.Repeat("e", 64) },
		"EOF":            func(r *InspectionReceipt) { r.SourceEndPTS = int64Pointer(4) },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			receipt := good
			receipt.VideoStreamIndex = intPointer(int(*good.VideoStreamIndex))
			receipt.StreamStartPTS = int64Pointer(*good.StreamStartPTS)
			receipt.SourceEndPTS = int64Pointer(*good.SourceEndPTS)
			mutate(&receipt)
			encoded, err := json.Marshal(receipt)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := VerifyInspectionArtifacts(manifestBytes, encoded, clock); err == nil {
				t.Fatal("tampered accepted receipt unexpectedly verified")
			}
		})
	}
	if _, err := VerifyInspectionArtifacts(manifestBytes, receiptBytes, append(clock, 0)); err == nil {
		t.Fatal("clock with trailing byte unexpectedly verified")
	}
	if _, err := VerifyInspectionArtifacts(manifestBytes, receiptBytes, bytesOf(inspectionClockHeader+inspectionMaxFrames*inspectionClockRecord+1)); err == nil {
		t.Fatal("oversized clock was processed")
	}
}

func TestVerifyInspectionArtifactsRejectsAcceptedFactsOnSourceRejection(t *testing.T) {
	manifest := InspectionManifest{
		Version: inspectionVersion, JobID: "job-1", AttemptID: "attempt-1", InputFingerprint: strings.Repeat("a", 64), WorkerSHA256: strings.Repeat("d", 64),
		Source:    InspectionSource{Name: InspectionSourceName, SHA256: strings.Repeat("b", 64), SizeBytes: 1},
		ProfileID: inspectionProfileID, ProfileDigest: strings.Repeat("c", 64), ProtocolVersion: inspectionProtocol, Limits: DefaultInspectionLimits(),
	}
	manifestBytes, _ := json.Marshal(manifest)
	receipt := InspectionReceipt{
		Version: inspectionVersion, JobID: manifest.JobID, AttemptID: manifest.AttemptID, InputFingerprint: manifest.InputFingerprint,
		WorkerSHA256:   manifest.WorkerSHA256,
		ManifestSHA256: digest(manifestBytes), SourceSHA256: manifest.Source.SHA256, SourceSizeBytes: manifest.Source.SizeBytes,
		ProfileID: manifest.ProfileID, ProfileDigest: manifest.ProfileDigest, ProtocolVersion: inspectionProtocol,
		Classification: InspectionRejected, IssueCode: "unsupported", IssueMessage: "unsupported source",
	}
	receipt.Container = "mov"
	receiptBytes, _ := json.Marshal(receipt)
	if _, err := VerifyInspectionArtifacts(manifestBytes, receiptBytes, nil); err == nil {
		t.Fatal("rejected receipt containing fabricated accepted facts verified")
	}
}

func TestInspectionColorPolicyPreservesUnmarkedSDRAndRejectsHDR(t *testing.T) {
	stream := inspectionStream{ColorMatrix: "bt470bg", ColorRange: "tv"}
	if !supportedSDR(stream) {
		t.Fatal("unmarked phone SDR with a known BT.601 matrix should be accepted under the named default policy")
	}
	stream.ColorTransfer = "smpte2084"
	if supportedSDR(stream) {
		t.Fatal("explicit HDR transfer was accepted as SDR")
	}
	stream.ColorTransfer = ""
	stream.ColorMatrix = "reserved"
	if supportedSDR(stream) {
		t.Fatal("unknown matrix was accepted")
	}
}

func TestInspectionSourceBindingDetectsSameLengthMutationAfterUse(t *testing.T) {
	path := filepath.Join(t.TempDir(), InspectionSourceName)
	original := []byte("immutable video bytes")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(original)
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	binding := InspectionSource{Name: InspectionSourceName, SHA256: hex.EncodeToString(h[:]), SizeBytes: int64(len(original))}
	if err := verifyInspectionSourceUnchanged(context.Background(), f, binding); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("mutated video bytes!!"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyInspectionSourceUnchanged(context.Background(), f, binding); err == nil {
		t.Fatal("same-length source mutation after initial verification was not detected")
	}
}

func TestExecInspectionRunnerUsesPinnedToolAndSourceDescriptors(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("inspection protocol v1 uses Linux descriptor paths")
	}
	tool, err := os.Open("/bin/cat")
	if err != nil {
		t.Skipf("cat executable is unavailable: %v", err)
	}
	defer tool.Close()
	sourcePath := filepath.Join(t.TempDir(), InspectionSourceName)
	want := []byte("descriptor-bound input")
	if err := os.WriteFile(sourcePath, want, 0o600); err != nil {
		t.Fatal(err)
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	got, err := (execInspectionRunner{}).Probe(context.Background(), tool, []string{"/proc/self/fd/4"}, source, 128)
	if err != nil || string(got) != string(want) {
		t.Fatalf("descriptor-bound process failed: got=%q err=%v", got, err)
	}
}

func TestOpenVerifiedToolKeepsVerifiedFileIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tool")
	original := []byte("verified executable bytes")
	if err := os.WriteFile(path, original, 0o700); err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(original)
	tool, err := openVerifiedTool(context.Background(), path, int64(len(original)), hex.EncodeToString(h[:]))
	if err != nil {
		t.Fatal(err)
	}
	defer tool.Close()
	replacement := path + ".replacement"
	if err := os.WriteFile(replacement, []byte("other executable bytes"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, path); err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(tool)
	if err != nil || string(got) != string(original) {
		t.Fatalf("open verified identity changed after path replacement: got=%q err=%v", got, err)
	}
}

func TestInspectionWorkerPublishesVerifiedReceiptFromPinnedInputs(t *testing.T) {
	workspace := t.TempDir()
	manifestPath := filepath.Join(workspace, InspectionManifestName)
	profile := validInspectionProfile()
	for i, tool := range []*cliv1.RendererToolBinary{profile.Ffmpeg, profile.Ffprobe} {
		tool.Path = filepath.Join(workspace, []string{"ffmpeg", "ffprobe"}[i])
		content := []byte(tool.GetArchiveMemberPath())
		if err := os.WriteFile(tool.GetPath(), content, 0o700); err != nil {
			t.Fatal(err)
		}
		h := sha256.Sum256(content)
		tool.Sha256 = hex.EncodeToString(h[:])
		tool.SizeBytes = uint64(len(content))
	}
	profile.ArtifactIdentity = profileArtifactIdentity(profile)
	profileBytes, err := protojson.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, InspectionProfileName), profileBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	sourceBytes := []byte("small immutable video fixture")
	sourceHash := sha256.Sum256(sourceBytes)
	if err := os.WriteFile(filepath.Join(workspace, InspectionSourceName), sourceBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := InspectionManifest{
		Version: inspectionVersion, JobID: "job-fixture", AttemptID: "attempt-fixture", InputFingerprint: strings.Repeat("a", 64), WorkerSHA256: currentWorkerDigest(t),
		Source:    InspectionSource{Name: InspectionSourceName, SHA256: hex.EncodeToString(sourceHash[:]), SizeBytes: int64(len(sourceBytes))},
		ProfileID: profile.GetProfileId(), ProfileDigest: profile.GetArtifactIdentity(), ProtocolVersion: inspectionProtocol, Limits: DefaultInspectionLimits(),
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, manifestBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	runner := inspectionFixtureRunner{streamJSON: []byte(`{"streams":[{"index":0,"codec_type":"video","codec_name":"h264","width":64,"height":64,"sample_aspect_ratio":"1:1","time_base":"1/30","start_pts":0,"color_primaries":"bt709","color_transfer":"bt709","color_space":"bt709","color_range":"tv"},{"index":1,"codec_type":"audio","codec_name":"aac"}],"format":{"format_name":"mov,mp4,m4a,3gp,3g2,mj2"}}`)}
	if err := runInspectionManifest(context.Background(), manifestPath, runner); err != nil {
		t.Fatal(err)
	}
	receiptBytes, err := os.ReadFile(filepath.Join(workspace, InspectionReceiptName))
	if err != nil {
		t.Fatal(err)
	}
	clockBytes, err := os.ReadFile(filepath.Join(workspace, InspectionClockName))
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := VerifyInspectionArtifacts(manifestBytes, receiptBytes, clockBytes)
	if err != nil || receipt.Classification != InspectionAccepted || receipt.Version != inspectionReceiptVersion || receipt.FrameCount != 3 || receipt.Audio == nil || len(receipt.Audio.Streams) != 1 || receipt.Audio.Streams[0] != (InspectionAudioStreamFact{OriginalStreamIndex: 1, Codec: "aac"}) {
		t.Fatalf("published inspection did not verify: receipt=%+v err=%v", receipt, err)
	}
}

type inspectionFixtureRunner struct{ streamJSON []byte }

func (runner inspectionFixtureRunner) Probe(_ context.Context, _ *os.File, args []string, _ *os.File, _ int64) ([]byte, error) {
	if !inspectionArgsBounded(args, false) {
		return nil, errors.New("probe did not use the pinned descriptor-backed constrained input")
	}
	if containsArg(args, "-show_frames") {
		return []byte(`{"frames":[{"pts":0,"duration":1},{"pts":1,"duration":1},{"pts":2,"duration":1}]}`), nil
	}
	if len(runner.streamJSON) != 0 {
		return bytes.Clone(runner.streamJSON), nil
	}
	return []byte(`{"streams":[{"index":0,"codec_type":"video","codec_name":"h264","width":64,"height":64,"sample_aspect_ratio":"1:1","time_base":"1/30","start_pts":0,"color_primaries":"bt709","color_transfer":"bt709","color_space":"bt709","color_range":"tv"}],"format":{"format_name":"mov,mp4,m4a,3gp,3g2,mj2"}}`), nil
}

func (inspectionFixtureRunner) Decode(_ context.Context, _ *os.File, args []string, _ *os.File, _ int64) ([]byte, error) {
	if !inspectionArgsBounded(args, true) {
		return nil, errors.New("decode did not use the pinned descriptor-backed constrained input")
	}
	for i, arg := range args {
		if arg == "-stats_enc_pre" && i+1 < len(args) {
			if args[i+1] != "pipe:1" {
				return nil, errors.New("stats output is not the bounded pipe")
			}
			return []byte("0 0 1/30\n1 1 1/30\n2 2 1/30\n"), nil
		}
	}
	return nil, errors.New("stats output path missing")
}

func inspectionArgsBounded(args []string, decode bool) bool {
	if !containsSequence(args, "-protocol_whitelist", "file,pipe") || !containsSequence(args, "-format_whitelist", "mov,matroska,webm") || !containsSequence(args, "-max_streams", "8") {
		return false
	}
	input := argIndex(args, inspectionInputDescriptor)
	protocol := argIndex(args, "-protocol_whitelist")
	format := argIndex(args, "-format_whitelist")
	streams := argIndex(args, "-max_streams")
	if input < 0 || protocol < 0 || format < 0 || streams < 0 || protocol >= input || format >= input || streams >= input {
		return false
	}
	if decode {
		return containsSequence(args, "-i", inspectionInputDescriptor) && containsSequence(args, "-stats_enc_pre", "pipe:1")
	}
	return len(args) > 0 && args[len(args)-1] == inspectionInputDescriptor
}

func argIndex(args []string, target string) int {
	for i, arg := range args {
		if arg == target {
			return i
		}
	}
	return -1
}

func containsArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}

func containsSequence(args []string, first, second string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == first && args[i+1] == second {
			return true
		}
	}
	return false
}

func validInspectionArtifacts(t *testing.T) ([]byte, []byte, []byte) {
	t.Helper()
	manifest := InspectionManifest{
		Version: inspectionVersion, JobID: "job-1", AttemptID: "attempt-1", InputFingerprint: strings.Repeat("a", 64), WorkerSHA256: strings.Repeat("d", 64),
		Source:    InspectionSource{Name: InspectionSourceName, SHA256: strings.Repeat("b", 64), SizeBytes: 1},
		ProfileID: inspectionProfileID, ProfileDigest: strings.Repeat("c", 64), ProtocolVersion: inspectionProtocol, Limits: DefaultInspectionLimits(),
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	clock, err := buildInspectionClock([]int64{0, 1, 2})
	if err != nil {
		t.Fatal(err)
	}
	start, end, index := int64(0), int64(3), uint32(0)
	receipt := InspectionReceipt{
		Version: inspectionVersion, JobID: manifest.JobID, AttemptID: manifest.AttemptID, InputFingerprint: manifest.InputFingerprint,
		WorkerSHA256:   manifest.WorkerSHA256,
		ManifestSHA256: digest(manifestBytes), SourceSHA256: manifest.Source.SHA256, SourceSizeBytes: manifest.Source.SizeBytes,
		ProfileID: manifest.ProfileID, ProfileDigest: manifest.ProfileDigest, ProtocolVersion: inspectionProtocol,
		Classification: InspectionAccepted, Container: "mov,mp4,m4a", StreamCount: 1, VideoStreamIndex: &index,
		Codec: "h264", CodedWidth: 64, CodedHeight: 64, DisplayWidth: 64, DisplayHeight: 64,
		SampleAspectNum: 1, SampleAspectDen: 1, ColorPrimaries: "bt709", ColorTransfer: "bt709", ColorMatrix: "bt709", ColorRange: "tv",
		TimeBaseNum: 1, TimeBaseDen: 30, StreamStartPTS: &start, SourceEndPTS: &end, LastFrameDurationTicks: 1,
		ColorPolicy:       inspectionColorPolicy,
		DurationNumerator: 1, DurationDenominator: 10, FrameCount: 3, EOFPolicy: "last_frame_duration",
		ClockArtifactRef: InspectionClockName, ClockArtifactSizeBytes: int64(len(clock)), ClockSHA256: digest(clock),
	}
	receiptBytes, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	return manifestBytes, receiptBytes, clock
}

func validInspectionProfile() *cliv1.RendererToolProfile {
	tool := func(path, member, version, sum string) *cliv1.RendererToolBinary {
		return &cliv1.RendererToolBinary{Path: path, ArchiveMemberPath: member, Version: version, Sha256: sum, BuildConfigSha256: strings.Repeat("d", 64), SizeBytes: 10}
	}
	profile := &cliv1.RendererToolProfile{
		SchemaVersion: 1, ProfileId: inspectionProfileID,
		State:               cliv1.RendererToolProfileState_RENDERER_TOOL_PROFILE_STATE_ARTIFACT_PRESENT,
		EnvironmentIdentity: strings.Repeat("e", 64), Os: "linux", Architecture: "amd64",
		ArchiveSha256: strings.Repeat("f", 64), ArchiveSizeBytes: 20, LicenseId: "pending-qualification", LicenseSha256: strings.Repeat("1", 64),
		Ffmpeg:  tool("/opt/ffmpeg/ffmpeg", "bin/ffmpeg", "ffmpeg-test", strings.Repeat("2", 64)),
		Ffprobe: tool("/opt/ffmpeg/ffprobe", "bin/ffprobe", "ffprobe-test", strings.Repeat("3", 64)),
	}
	profile.ArtifactIdentity = profileArtifactIdentity(profile)
	return profile
}

func bytesOf(size int) []byte     { return make([]byte, size) }
func intPointer(v int) *uint32    { value := uint32(v); return &value }
func int64Pointer(v int64) *int64 { return &v }

func currentWorkerDigest(t *testing.T) string {
	t.Helper()
	file, err := os.Open("/proc/self/exe")
	if err != nil {
		t.Skipf("Linux executable identity is unavailable: %v", err)
	}
	defer file.Close()
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(h.Sum(nil))
}
