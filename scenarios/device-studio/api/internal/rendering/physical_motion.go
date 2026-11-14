package rendering

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"io"
	"os"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
)

const (
	physicalVideoProfileName         = "physical-video-profile.json"
	physicalMotionFrameName          = "physical-motion-frame.png"
	physicalMotionFrameMax           = 64 << 20
	physicalMotionPreviewPrefix      = "physical-motion-preview-"
	physicalMotionInterruptedPreview = "physical-motion-preview-interrupted.png"
	physicalMotionCheckpoint         = "physical-motion-preview-current.png"
	physicalMotionCheckpointTemp     = "physical-motion-preview-current.tmp"
)

type physicalMotionExecution struct {
	profileBytes     []byte
	native           VideoNativeFacts
	outputSize       int64
	outputSHA256     string
	frameRGBDigests  []PhysicalFrameRGBDigest
	sourceFrameFacts []PhysicalRecordingFrameFact
}

func runPhysicalMotion(ctx context.Context, root *os.Root, workspace, blender string, manifest PhysicalRenderManifest) (result physicalMotionExecution, retErr error) {
	profile, err := ResolveInspectionProfile(ctx)
	if err != nil {
		return result, fmt.Errorf("resolve pinned FFmpeg video profile: %w", err)
	}
	pair, err := LocalToolPairFromProfile(profile)
	if err != nil || pair.ProfileID != manifest.Output.VideoToolProfileID || pair.ArtifactIdentity != manifest.Output.VideoToolArtifactID || profileArtifactIdentity(profile) != manifest.Output.VideoToolArtifactID {
		return result, errors.New("resolved FFmpeg pair differs from the immutable physical motion profile")
	}
	profileBytes, err := (protojson.MarshalOptions{EmitUnpopulated: false}).Marshal(profile)
	if err != nil || len(profileBytes) == 0 || len(profileBytes) > inspectionMaxProfile {
		return result, errors.New("resolved FFmpeg profile observation is invalid or oversized")
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
	if err := ensurePhysicalOutputAbsent(root, physicalMotionFrameName); err != nil {
		return result, err
	}
	previewNames := []string{
		physicalMotionPreviewPrefix + "first.png",
		physicalMotionPreviewPrefix + "middle.png",
		physicalMotionPreviewPrefix + "last.png",
		physicalMotionInterruptedPreview,
		physicalMotionCheckpoint,
	}
	for _, name := range previewNames {
		if err := ensurePhysicalOutputAbsent(root, name); err != nil {
			return result, err
		}
	}
	defer func() {
		if retErr == nil {
			for _, name := range previewNames {
				if err := root.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
					retErr = errors.Join(retErr, fmt.Errorf("remove successful physical motion preview: %w", err))
				}
			}
		}
		if err := root.Remove(physicalMotionFrameName); err != nil && !errors.Is(err, os.ErrNotExist) {
			retErr = errors.Join(retErr, fmt.Errorf("remove physical motion frame scratch: %w", err))
		}
	}()
	video := VideoManifest{Width: int(manifest.Output.Width), Height: int(manifest.Output.Height), OutputRate: manifest.Output.Rate}
	encoder, err := startVideoEncoderNamed(deadline, root, ffmpeg, video, int(manifest.Output.FrameCount), manifest.Output.OutputName)
	if err != nil {
		return result, fmt.Errorf("start bounded physical MP4 encoder: %w", err)
	}
	defer encoder.abort()
	var lastReceivedFrame []byte
	var lastCheckpointAt time.Time
	if err := runPhysicalBlenderFrameStream(deadline, root, workspace, blender, manifest, func(ordinal uint32, pngBytes []byte) error {
		if ordinal >= manifest.Output.FrameCount {
			return errors.New("Blender emitted an unplanned physical frame ordinal")
		}
		if !isPhysicalRGBA8PNG(pngBytes) {
			return errors.New("Blender motion frame is not an 8-bit RGBA PNG")
		}
		// Keep only the most recent complete frame in memory. Blender removes its
		// scratch file after sending a frame, so a cancellation between frames
		// cannot rely on that file still existing when the stream unwinds.
		lastReceivedFrame = append(lastReceivedFrame[:0], pngBytes...)
		frame, err := physicalPNGToRGB(pngBytes, int(manifest.Output.Width), int(manifest.Output.Height))
		if err != nil {
			return err
		}
		if lastCheckpointAt.IsZero() || time.Since(lastCheckpointAt) >= time.Second {
			if err := writePhysicalMotionCheckpoint(root, pngBytes, manifest.Output.Width, manifest.Output.Height); err != nil {
				return err
			}
			lastCheckpointAt = time.Now()
		}
		if err := encoder.writeFrame(deadline, frame); err != nil {
			return err
		}
		for index, target := range physicalMotionPreviewOrdinals(manifest.Output.FrameCount) {
			if ordinal == target {
				if err := writePhysicalRoot(root, previewNames[index], pngBytes); err != nil {
					return fmt.Errorf("retain bounded failed-attempt motion preview %s: %w", previewNames[index], err)
				}
			}
		}
		result.frameRGBDigests = append(result.frameRGBDigests, PhysicalFrameRGBDigest{OutputOrdinal: ordinal, RGBSHA256: digest(frame)})
		return nil
	}, manifest.Output.FrameCount); err != nil {
		// Blender writes each complete frame to the attempt-local scratch path
		// before sending it over the frame pipe. If cancellation or another error
		// interrupts the pipe between those steps, preserve one validated frame
		// for review before deferred scratch cleanup runs. This is bounded
		// evidence from a failed attempt, never a render output.
		var previewErr error
		if len(lastReceivedFrame) > 0 {
			previewErr = retainInterruptedPhysicalPNG(root, lastReceivedFrame, manifest.Output.Width, manifest.Output.Height)
		} else {
			previewErr = retainInterruptedPhysicalFrame(root, manifest.Output.Width, manifest.Output.Height)
		}
		if previewErr != nil && !errors.Is(previewErr, os.ErrNotExist) {
			return result, errors.Join(err, fmt.Errorf("retain interrupted physical motion frame: %w", previewErr))
		}
		return result, err
	}
	result.outputSize, result.outputSHA256, err = encoder.finish(deadline)
	if err != nil {
		return result, err
	}
	result.native, err = verifyEncodedVideoNamed(deadline, ffprobe, ffmpeg, root, video, int(manifest.Output.FrameCount), manifest.Output.OutputName)
	if err != nil {
		return result, fmt.Errorf("verify encoded physical motion MP4: %w", err)
	}
	if err := verifyInspectionToolUnchanged(deadline, pair); err != nil {
		return result, err
	}
	if err := validatePhysicalFrameRGBDigests(manifest.Output, result.frameRGBDigests); err != nil {
		return result, err
	}
	result.profileBytes = profileBytes
	return result, nil
}

func retainInterruptedPhysicalFrame(root *os.Root, width, height uint32) error {
	if root == nil || width == 0 || height == 0 {
		return errors.New("interrupted physical frame retention arguments are invalid")
	}
	file, err := root.Open(physicalMotionFrameName)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > physicalMotionFrameMax {
		return errors.New("interrupted physical frame scratch is not a bounded regular file")
	}
	data, err := io.ReadAll(io.LimitReader(file, physicalMotionFrameMax+1))
	if err != nil || int64(len(data)) != info.Size() || int64(len(data)) > physicalMotionFrameMax {
		return errors.New("interrupted physical frame scratch changed or exceeded its bound")
	}
	return retainInterruptedPhysicalPNG(root, data, width, height)
}

func retainInterruptedPhysicalPNG(root *os.Root, data []byte, width, height uint32) error {
	if root == nil || width == 0 || height == 0 || len(data) == 0 || int64(len(data)) > physicalMotionFrameMax {
		return errors.New("interrupted physical frame retention arguments are invalid")
	}
	if !isPhysicalRGBA8PNG(data) {
		return errors.New("interrupted physical frame scratch is not an RGBA8 PNG")
	}
	if _, err := physicalPNGToRGB(data, int(width), int(height)); err != nil {
		return fmt.Errorf("validate interrupted physical frame: %w", err)
	}
	return writePhysicalRoot(root, physicalMotionInterruptedPreview, data)
}

// writePhysicalMotionCheckpoint atomically replaces one validated frame in the
// private attempt workspace. The single bounded file survives worker loss and
// is removed after successful render completion; it is not published as output.
func writePhysicalMotionCheckpoint(root *os.Root, data []byte, width, height uint32) error {
	if root == nil || len(data) == 0 || int64(len(data)) > physicalMotionFrameMax || !isPhysicalRGBA8PNG(data) {
		return errors.New("physical motion checkpoint is not a bounded RGBA8 PNG")
	}
	if _, err := physicalPNGToRGB(data, int(width), int(height)); err != nil {
		return fmt.Errorf("validate physical motion checkpoint: %w", err)
	}
	if err := root.Remove(physicalMotionCheckpointTemp); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove stale physical motion checkpoint temporary: %w", err)
	}
	f, err := root.OpenFile(physicalMotionCheckpointTemp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create physical motion checkpoint temporary: %w", err)
	}
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		_ = root.Remove(physicalMotionCheckpointTemp)
		return err
	}
	if closeErr != nil {
		_ = root.Remove(physicalMotionCheckpointTemp)
		return closeErr
	}
	if err := root.Rename(physicalMotionCheckpointTemp, physicalMotionCheckpoint); err != nil {
		_ = root.Remove(physicalMotionCheckpointTemp)
		return fmt.Errorf("publish physical motion checkpoint atomically: %w", err)
	}
	return nil
}

func physicalMotionPreviewOrdinals(frameCount uint32) [3]uint32 {
	if frameCount <= 1 {
		return [3]uint32{0, 0, 0}
	}
	return [3]uint32{0, (frameCount - 1) / 2, frameCount - 1}
}

func runPhysicalBlenderFrameStream(ctx context.Context, root *os.Root, workspace, blender string, manifest PhysicalRenderManifest, onFrame func(uint32, []byte) error, expected uint32) error {
	if ctx == nil || onFrame == nil || expected == 0 || expected > PhysicalMotionMaxFramesV2 {
		return errors.New("physical motion frame stream arguments are invalid")
	}
	readEnd, writeEnd, err := os.Pipe()
	if err != nil {
		return err
	}
	defer readEnd.Close()
	defer writeEnd.Close()
	var stdout, stderr physicalBlenderLogBuffer
	stdout.limit, stderr.limit = physicalCommandOutMax, physicalCommandErrMax
	process, err := startPhysicalBlenderProcess(ctx, root, workspace, blender, manifest, []*os.File{writeEnd}, &stdout, &stderr)
	if err != nil {
		return err
	}
	_ = writeEnd.Close()
	for ordinal := uint32(0); ordinal < expected; ordinal++ {
		gotOrdinal, data, readErr := readPhysicalFrame(readEnd)
		if readErr != nil {
			waitErr := process.abort()
			if ctx.Err() != nil {
				return fmt.Errorf("physical Blender frame stream ended: %w", ctx.Err())
			}
			return fmt.Errorf("physical Blender frame stream ended before ordinal %d: %w (%v): %s", ordinal, readErr, waitErr, truncateUTF8Tail(stderr.data.String(), 1024))
		}
		if gotOrdinal != ordinal {
			waitErr := process.abort()
			return errors.Join(fmt.Errorf("physical Blender emitted frame ordinal %d, expected %d", gotOrdinal, ordinal), waitErr)
		}
		if err := onFrame(ordinal, data); err != nil {
			return errors.Join(fmt.Errorf("consume physical frame %d: %w", ordinal, err), process.abort())
		}
	}
	var extra [1]byte
	if n, readErr := readEnd.Read(extra[:]); n != 0 || readErr != io.EOF {
		waitErr := process.abort()
		if ctx.Err() != nil {
			return fmt.Errorf("physical Blender frame stream ended: %w", ctx.Err())
		}
		return errors.Join(errors.New("physical Blender emitted trailing frame protocol bytes"), waitErr)
	}
	return process.wait()
}

func readPhysicalFrame(r io.Reader) (uint32, []byte, error) {
	var header [12]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return 0, nil, err
	}
	ordinal := binary.BigEndian.Uint32(header[:4])
	size := binary.BigEndian.Uint64(header[4:])
	if size == 0 || size > physicalMotionFrameMax {
		return 0, nil, errors.New("physical motion PNG frame exceeds its byte bound")
	}
	data := make([]byte, int(size))
	if _, err := io.ReadFull(r, data); err != nil {
		return 0, nil, err
	}
	return ordinal, data, nil
}

func physicalPNGToRGB(data []byte, width, height int) ([]byte, error) {
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || format != "png" || config.Width != width || config.Height != height {
		return nil, errors.New("Blender motion frame PNG header differs from the planned raster")
	}
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil || format != "png" || img.Bounds().Dx() != width || img.Bounds().Dy() != height {
		return nil, errors.New("Blender motion frame is not a valid planned PNG raster")
	}
	pixels := make([]byte, width*height*3)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			if c.A != 255 {
				return nil, errors.New("Blender motion frame contains non-opaque pixels")
			}
			i := (y*width + x) * 3
			pixels[i], pixels[i+1], pixels[i+2] = c.R, c.G, c.B
		}
	}
	return pixels, nil
}

func verifyInspectionToolUnchanged(ctx context.Context, expected LocalToolPair) error {
	profile, err := ResolveInspectionProfile(ctx)
	if err != nil {
		return fmt.Errorf("recheck pinned FFmpeg profile: %w", err)
	}
	actual, err := LocalToolPairFromProfile(profile)
	if err != nil || actual != expected || profileArtifactIdentity(profile) != expected.ArtifactIdentity {
		return errors.New("managed FFmpeg pair changed during physical motion render")
	}
	return nil
}

func physicalMotionNativeFactsJSON(facts VideoNativeFacts) json.RawMessage {
	b, _ := json.Marshal(facts)
	return b
}
