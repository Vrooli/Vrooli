package rendering

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"device-studio/internal/composition"
	"device-studio/internal/presentations"
	"google.golang.org/protobuf/encoding/protojson"
)

const physicalRecordingScreenName = "physical-recording-screen.png"

// PhysicalRecordingScheduleFacts is derived from a verified inspection clock
// and the exact recording/output timing requested by the immutable manifest.
type PhysicalRecordingScheduleFacts struct {
	Samples                        []composition.FrameSample `json:"samples"`
	SourcePTS                      []int64                   `json:"source_pts"`
	RequiredSourcePrefixFrameCount uint32                    `json:"required_source_prefix_frame_count"`
}

// DerivePhysicalRecordingSchedule verifies the accepted inspection artifacts
// and derives the output schedule and minimum decoded source prefix. The caller
// sets SourcePrefixFrameCount from the result, then validates/canonicalizes the
// complete render manifest before claiming work.
func DerivePhysicalRecordingSchedule(ctx context.Context, recording PhysicalRecordingInput, outputRate composition.Rational, outputFrameCount uint32, inspectionManifest, inspectionReceipt, clockBytes []byte) (PhysicalRecordingScheduleFacts, error) {
	var facts PhysicalRecordingScheduleFacts
	i := recording.Inspection
	if ctx == nil || len(inspectionManifest) == 0 || len(inspectionManifest) > videoMaxManifestBytes || len(inspectionReceipt) == 0 || len(inspectionReceipt) > videoMaxReceiptBytes || len(clockBytes) == 0 || len(clockBytes) > videoMaxClockBytes ||
		int64(len(inspectionManifest)) != i.Manifest.SizeBytes || digest(inspectionManifest) != i.Manifest.SHA256 || int64(len(inspectionReceipt)) != i.Receipt.SizeBytes || digest(inspectionReceipt) != i.Receipt.SHA256 || int64(len(clockBytes)) != i.Clock.SizeBytes || digest(clockBytes) != i.Clock.SHA256 {
		return facts, errors.New("recording inspection artifacts differ from their immutable bindings")
	}
	if err := validateVideoInspection(i); err != nil {
		return facts, err
	}
	accepted, err := VerifyInspectionArtifacts(inspectionManifest, inspectionReceipt, clockBytes)
	if err != nil {
		return facts, fmt.Errorf("verify accepted inspection artifacts: %w", err)
	}
	if err := matchVideoInspection(i, accepted); err != nil {
		return facts, fmt.Errorf("recording schedule differs from accepted inspection facts: %w", err)
	}
	if i.Source.SHA256 != accepted.SourceSHA256 || i.Source.SizeBytes != accepted.SourceSizeBytes {
		return facts, errors.New("recording source binding differs from accepted inspection source")
	}
	if recording.TrimStartPTS < i.StreamStartPTS || recording.TrimEndPTS <= recording.TrimStartPTS || recording.TrimEndPTS > i.SourceEndPTS || recording.PlaybackRate.Num <= 0 || recording.PlaybackRate.Den <= 0 ||
		recording.AudioPolicy == PhysicalRecordingAudioPassthrough || (recording.AudioPolicy != PhysicalRecordingAudioUnspecified && recording.AudioPolicy != PhysicalRecordingAudioSilent) {
		return facts, errors.New("recording trim, playback rate, or audio policy is unsupported")
	}
	if outputFrameCount == 0 || outputFrameCount > PhysicalRecordingMaxFrames || (outputRate.Num != 24 && outputRate.Num != 30) || outputRate.Den != 1 || outputFrameCount > uint32(outputRate.Num)*10 {
		return facts, errors.New("recording output schedule is outside its bounded profile")
	}
	timing := VideoManifest{Inspection: i, TrimStartPTS: recording.TrimStartPTS, TrimEndPTS: recording.TrimEndPTS, PlaybackRate: recording.PlaybackRate, OutputRate: outputRate}
	if err := validateVideoDuration(timing); err != nil {
		return facts, fmt.Errorf("recording duration is outside its bounded output profile: %w", err)
	}
	_, plannedCount, err := videoDurationAndFrameCount(timing)
	if err != nil || plannedCount != int64(outputFrameCount) {
		return facts, errors.New("recording output count differs from exact rational duration")
	}
	pts, err := decodeInspectionClock(clockBytes, i.FrameCount)
	if err != nil {
		return facts, err
	}
	plan, err := composition.PlanFrames(ctx, composition.FrameClock{SourcePTS: pts, SourceTimeBase: i.TimeBase, SourceEndPTS: i.SourceEndPTS, TrimStartPTS: recording.TrimStartPTS, TrimEndPTS: recording.TrimEndPTS, PlaybackRate: recording.PlaybackRate, OutputRate: outputRate, Underflow: composition.RangeReject, EOF: composition.RangeReject})
	if err != nil || len(plan) != int(outputFrameCount) {
		return facts, errors.New("recording output count differs from the exact source-clock schedule")
	}
	for _, sample := range plan {
		if sample.SourceIndex < 0 || sample.SourceIndex >= len(pts) {
			return facts, errors.New("recording schedule selects an invalid source ordinal")
		}
		if uint32(sample.SourceIndex+1) > facts.RequiredSourcePrefixFrameCount {
			facts.RequiredSourcePrefixFrameCount = uint32(sample.SourceIndex + 1)
		}
	}
	if facts.RequiredSourcePrefixFrameCount == 0 || facts.RequiredSourcePrefixFrameCount > PhysicalRecordingMaxSourcePrefix ||
		recording.SourcePrefixFrameCount != 0 && recording.SourcePrefixFrameCount != facts.RequiredSourcePrefixFrameCount {
		return PhysicalRecordingScheduleFacts{}, errors.New("recording source prefix does not match the exact scheduled source frames")
	}
	facts.Samples, facts.SourcePTS = plan, pts
	return facts, nil
}

func runPhysicalRecordingMotion(ctx context.Context, root *os.Root, workspace, blender string, manifest PhysicalRenderManifest) (result physicalMotionExecution, retErr error) {
	r := manifest.Screen.Recording
	if r == nil || manifest.Version != PhysicalRecordingManifestVersion {
		return result, errors.New("recording motion requires its versioned source closure")
	}
	profile, err := ResolveInspectionProfile(ctx)
	if err != nil {
		return result, fmt.Errorf("resolve pinned FFmpeg video profile: %w", err)
	}
	pair, err := LocalToolPairFromProfile(profile)
	if err != nil || pair.ProfileID != manifest.Output.VideoToolProfileID || pair.ArtifactIdentity != manifest.Output.VideoToolArtifactID || profileArtifactIdentity(profile) != manifest.Output.VideoToolArtifactID {
		return result, errors.New("resolved FFmpeg pair differs from immutable recording profile")
	}
	profileBytes, err := (protojson.MarshalOptions{EmitUnpopulated: false}).Marshal(profile)
	if err != nil || len(profileBytes) == 0 || len(profileBytes) > inspectionMaxProfile {
		return result, errors.New("resolved FFmpeg profile observation is invalid")
	}
	if err := ensurePhysicalOutputAbsent(root, physicalVideoProfileName); err != nil {
		return result, err
	}
	if err := writePhysicalRoot(root, physicalVideoProfileName, profileBytes); err != nil {
		return result, err
	}
	deadline, cancel := context.WithTimeout(ctx, time.Duration(manifest.Profile.WallTimeSeconds)*time.Second)
	defer cancel()
	ffmpeg, err := openVerifiedTool(deadline, pair.FFmpegPath, pair.FFmpegSize, pair.FFmpegSHA256)
	if err != nil {
		return result, fmt.Errorf("verify pinned FFmpeg: %w", err)
	}
	defer ffmpeg.Close()
	ffprobe, err := openVerifiedTool(deadline, pair.FFprobePath, pair.FFprobeSize, pair.FFprobeSHA256)
	if err != nil {
		return result, fmt.Errorf("verify pinned FFprobe: %w", err)
	}
	defer ffprobe.Close()
	if err := ensurePhysicalOutputAbsent(root, physicalRecordingScreenName); err != nil {
		return result, err
	}
	defer func() {
		if err := root.Remove(physicalRecordingScreenName); err != nil && !errors.Is(err, os.ErrNotExist) {
			retErr = errors.Join(retErr, fmt.Errorf("remove recording screen scratch: %w", err))
		}
	}()
	if err := ensurePhysicalOutputAbsent(root, physicalMotionFrameName); err != nil {
		return result, err
	}
	defer func() {
		if err := root.Remove(physicalMotionFrameName); err != nil && !errors.Is(err, os.ErrNotExist) {
			retErr = errors.Join(retErr, fmt.Errorf("remove recording Blender frame scratch: %w", err))
		}
	}()
	video := VideoManifest{Width: int(manifest.Output.Width), Height: int(manifest.Output.Height), OutputRate: manifest.Output.Rate}
	encoder, err := startVideoEncoderNamed(deadline, root, ffmpeg, video, int(manifest.Output.FrameCount), manifest.Output.OutputName)
	if err != nil {
		return result, fmt.Errorf("start bounded recording encoder: %w", err)
	}
	defer encoder.abort()
	inspectionManifestBytes, err := readBoundedRegular(deadline, root, r.Inspection.Manifest.Name, videoMaxManifestBytes, r.Inspection.Manifest.SizeBytes)
	if err != nil || digest(inspectionManifestBytes) != r.Inspection.Manifest.SHA256 {
		return result, errors.New("accepted inspection manifest changed before recording render")
	}
	inspectionReceiptBytes, err := readBoundedRegular(deadline, root, r.Inspection.Receipt.Name, videoMaxReceiptBytes, r.Inspection.Receipt.SizeBytes)
	if err != nil || digest(inspectionReceiptBytes) != r.Inspection.Receipt.SHA256 {
		return result, errors.New("accepted inspection receipt changed before recording render")
	}
	clockBytes, err := readBoundedRegular(deadline, root, r.Inspection.Clock.Name, videoMaxClockBytes, r.Inspection.Clock.SizeBytes)
	if err != nil || digest(clockBytes) != r.Inspection.Clock.SHA256 {
		return result, errors.New("accepted inspection clock changed before recording render")
	}
	schedule, err := DerivePhysicalRecordingSchedule(deadline, *r, manifest.Output.Rate, manifest.Output.FrameCount, inspectionManifestBytes, inspectionReceiptBytes, clockBytes)
	if err != nil {
		return result, fmt.Errorf("derive accepted recording schedule: %w", err)
	}
	pts := schedule.SourcePTS
	plan := schedule.Samples
	maxPrefix := uint32(0)
	bySource := make([][]int, len(pts))
	for ordinal, sample := range plan {
		bySource[sample.SourceIndex] = append(bySource[sample.SourceIndex], ordinal)
		if uint32(sample.SourceIndex+1) > maxPrefix {
			maxPrefix = uint32(sample.SourceIndex + 1)
		}
	}
	if maxPrefix != schedule.RequiredSourcePrefixFrameCount || maxPrefix != r.SourcePrefixFrameCount {
		return result, errors.New("recording decoder prefix differs from the shared exact schedule")
	}
	source, err := root.Open(r.Inspection.Source.Name)
	if err != nil {
		return result, fmt.Errorf("open accepted recording source: %w", err)
	}
	defer source.Close()
	if err := verifyInspectionSourceUnchanged(deadline, source, InspectionSource{Name: r.Inspection.Source.Name, SHA256: r.Inspection.Source.SHA256, SizeBytes: r.Inspection.Source.SizeBytes}); err != nil {
		return result, err
	}
	stream := &physicalRecordingBlenderStream{ctx: deadline, workspace: workspace, blender: blender, expected: manifest.Output.FrameCount}
	defer stream.abort()
	facts := make([]PhysicalRecordingFrameFact, 0, len(plan))
	videoManifest := VideoManifest{Inspection: r.Inspection, TrimStartPTS: r.TrimStartPTS, TrimEndPTS: r.TrimEndPTS, PlaybackRate: r.PlaybackRate, OutputRate: manifest.Output.Rate, Width: int(manifest.Output.Width), Height: int(manifest.Output.Height)}
	if err := decodeVideoRGBPrefix(deadline, ffmpeg, source, videoManifest, pts, r.SourcePrefixFrameCount, func(sourceOrdinal int, sourcePTS int64, raw []byte) error {
		ordinals := bySource[sourceOrdinal]
		if len(ordinals) == 0 {
			return nil
		}
		app, err := physicalRGB24Image(raw, r.Inspection.Width, r.Inspection.Height)
		if err != nil {
			return err
		}
		composed, composeErr := presentations.Compose(deadline, app, r.Presentation)
		if composeErr != nil {
			return fmt.Errorf("compose recording screen frame: %w", composeErr)
		}
		var pngBytes bytes.Buffer
		if err := png.Encode(&pngBytes, composed.Raster); err != nil {
			return err
		}
		frame := pngBytes.Bytes()
		if len(frame) == 0 || len(frame) > physicalScreenBytesMax || !isPhysicalScreenTexturePNG(frame) {
			return errors.New("composed recording screen exceeds its one-frame PNG texture contract")
		}
		for _, outputOrdinal := range ordinals {
			if outputOrdinal == 0 {
				if err := writePhysicalRoot(root, physicalRecordingScreenName, frame); err != nil {
					return err
				}
				if err := stream.start(); err != nil {
					return err
				}
			} else if err := stream.send(outputOrdinal, frame); err != nil {
				return err
			}
			got, pngOutput, err := stream.receive()
			if err != nil {
				return err
			}
			if got != uint32(outputOrdinal) {
				return fmt.Errorf("Blender recording output ordinal %d, expected %d", got, outputOrdinal)
			}
			if !isPhysicalRGBA8PNG(pngOutput) {
				return errors.New("Blender recording output is not RGBA8 PNG")
			}
			pixels, err := physicalPNGToRGB(pngOutput, int(manifest.Output.Width), int(manifest.Output.Height))
			if err != nil {
				return err
			}
			if err := encoder.writeFrame(deadline, pixels); err != nil {
				return err
			}
			result.frameRGBDigests = append(result.frameRGBDigests, PhysicalFrameRGBDigest{OutputOrdinal: uint32(outputOrdinal), RGBSHA256: digest(pixels)})
			facts = append(facts, PhysicalRecordingFrameFact{OutputOrdinal: uint32(outputOrdinal), SourceOrdinal: uint32(sourceOrdinal), SourcePTS: sourcePTS, ScreenSHA256: digest(frame)})
		}
		return nil
	}); err != nil {
		return result, err
	}
	if uint32(len(facts)) != manifest.Output.FrameCount {
		return result, errors.New("recording source decoder did not produce every planned screen frame")
	}
	if err := stream.finish(); err != nil {
		return result, err
	}
	result.outputSize, result.outputSHA256, err = encoder.finish(deadline)
	if err != nil {
		return result, err
	}
	result.native, err = verifyEncodedVideoNamed(deadline, ffprobe, ffmpeg, root, video, int(manifest.Output.FrameCount), manifest.Output.OutputName)
	if err != nil {
		return result, fmt.Errorf("verify encoded physical recording: %w", err)
	}
	if err := verifyInspectionSourceUnchanged(deadline, source, InspectionSource{Name: r.Inspection.Source.Name, SHA256: r.Inspection.Source.SHA256, SizeBytes: r.Inspection.Source.SizeBytes}); err != nil {
		return result, err
	}
	if err := verifyInspectionToolUnchanged(deadline, pair); err != nil {
		return result, err
	}
	if err := validatePhysicalFrameRGBDigests(manifest.Output, result.frameRGBDigests); err != nil {
		return result, err
	}
	result.sourceFrameFacts = facts
	result.profileBytes = profileBytes
	return result, nil
}

type physicalRecordingBlenderStream struct {
	ctx                context.Context
	workspace, blender string
	expected           uint32
	cmd                *exec.Cmd
	outR               *os.File
	inW                *os.File
	waited             bool
	stderr             *physicalBlenderLogBuffer
}

func (s *physicalRecordingBlenderStream) start() error {
	if s.ctx == nil || s.expected == 0 || s.expected > PhysicalRecordingMaxFrames || !filepath.IsAbs(s.blender) || filepath.Clean(s.blender) != s.blender {
		return errors.New("recording Blender stream identity is invalid")
	}
	binary, err := readRegularPath(s.blender, 512<<20)
	if err != nil || digest(binary) != PhysicalBlenderBinarySHA {
		return errors.New("managed Blender executable differs from pinned profile")
	}
	outR, outW, err := os.Pipe()
	if err != nil {
		return err
	}
	inR, inW, err := os.Pipe()
	if err != nil {
		outR.Close()
		outW.Close()
		return err
	}
	if err := ensurePhysicalBlenderDirectories(s.workspace); err != nil {
		outR.Close()
		outW.Close()
		inR.Close()
		inW.Close()
		return err
	}
	cmd := exec.CommandContext(s.ctx, s.blender, physicalBlenderArgs(s.workspace)...)
	cmd.Dir = s.workspace
	cmd.Env = physicalBlenderEnvironment(s.workspace)
	cmd.ExtraFiles = []*os.File{outW, inR}
	var stdout, stderr physicalBlenderLogBuffer
	stdout.limit, stderr.limit = physicalCommandOutMax, physicalCommandErrMax
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Start(); err != nil {
		outR.Close()
		outW.Close()
		inR.Close()
		inW.Close()
		return err
	}
	outW.Close()
	inR.Close()
	s.cmd, s.outR, s.inW, s.stderr = cmd, outR, inW, &stderr
	return nil
}
func (s *physicalRecordingBlenderStream) send(ordinal int, pngBytes []byte) error {
	if s.cmd == nil || s.inW == nil || ordinal <= 0 || uint32(ordinal) >= s.expected || len(pngBytes) == 0 || len(pngBytes) > physicalScreenBytesMax {
		return errors.New("recording Blender input frame is outside its bound")
	}
	var header [12]byte
	binary.BigEndian.PutUint32(header[:4], uint32(ordinal))
	binary.BigEndian.PutUint64(header[4:], uint64(len(pngBytes)))
	for _, part := range [][]byte{header[:], pngBytes} {
		for len(part) > 0 {
			n, err := s.inW.Write(part)
			if err != nil {
				return err
			}
			if n == 0 {
				return io.ErrShortWrite
			}
			part = part[n:]
		}
	}
	return nil
}
func (s *physicalRecordingBlenderStream) receive() (uint32, []byte, error) {
	if s.outR == nil {
		return 0, nil, errors.New("recording Blender stream is not started")
	}
	return readPhysicalFrame(s.outR)
}
func (s *physicalRecordingBlenderStream) finish() error {
	if s.cmd == nil {
		return errors.New("recording Blender stream did not start")
	}
	if err := s.inW.Close(); err != nil {
		return err
	}
	s.inW = nil
	var extra [1]byte
	n, readErr := s.outR.Read(extra[:])
	if n != 0 || readErr != io.EOF {
		s.abort()
		return errors.New("recording Blender emitted trailing frame bytes")
	}
	err := s.cmd.Wait()
	s.waited = true
	s.outR.Close()
	s.outR = nil
	if err != nil {
		diagnostic := ""
		if s.stderr != nil {
			diagnostic = s.stderr.data.String()
		}
		return physicalBlenderFailure(err, s.ctx.Err(), diagnostic)
	}
	return nil
}

func physicalRGB24Image(raw []byte, width, height int) (*image.NRGBA, error) {
	if width <= 0 || height <= 0 || int64(width)*int64(height) > inspectionMaxPixels || int64(width)*int64(height)*3 != int64(len(raw)) {
		return nil, errors.New("decoded recording frame differs from its pinned RGB24 raster")
	}
	image := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		dst := image.Pix[y*image.Stride : y*image.Stride+width*4]
		src := raw[y*width*3 : (y+1)*width*3]
		for x := 0; x < width; x++ {
			dst[x*4], dst[x*4+1], dst[x*4+2], dst[x*4+3] = src[x*3], src[x*3+1], src[x*3+2], 255
		}
	}
	return image, nil
}

func validatePhysicalRecordingSchedule(recording *PhysicalRecordingInput, plan []composition.FrameSample, facts []PhysicalRecordingFrameFact) error {
	if recording == nil || len(plan) == 0 || len(plan) != len(facts) {
		return errors.New("recording source facts do not cover the planned output frames")
	}
	var maxPrefix uint32
	for i, fact := range facts {
		sample := plan[i]
		if sample.SourceIndex < 0 || sample.SourceIndex >= int(recording.Inspection.FrameCount) ||
			fact.OutputOrdinal != uint32(i) || fact.SourceOrdinal != uint32(sample.SourceIndex) || fact.SourcePTS != sample.SourcePTS || !physicalDigestPattern.MatchString(fact.ScreenSHA256) {
			return fmt.Errorf("recording source frame fact %d differs from the exact planned clock", i)
		}
		if uint32(sample.SourceIndex+1) > maxPrefix {
			maxPrefix = uint32(sample.SourceIndex + 1)
		}
	}
	if maxPrefix != recording.SourcePrefixFrameCount || maxPrefix > PhysicalRecordingMaxSourcePrefix {
		return errors.New("recording source decode prefix differs from the highest scheduled source ordinal")
	}
	return nil
}
func (s *physicalRecordingBlenderStream) abort() {
	if s.inW != nil {
		s.inW.Close()
		s.inW = nil
	}
	if s.outR != nil {
		s.outR.Close()
		s.outR = nil
	}
	if s.cmd != nil && !s.waited && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		_ = s.cmd.Wait()
		s.waited = true
	}
}
