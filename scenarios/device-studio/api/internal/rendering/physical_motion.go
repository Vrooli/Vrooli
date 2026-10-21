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
	"os/exec"
	"path/filepath"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
)

const (
	physicalVideoProfileName = "physical-video-profile.json"
	physicalMotionFrameName  = "physical-motion-frame.png"
	physicalMotionFrameMax   = 64 << 20
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
	defer func() {
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
	if err := runPhysicalBlenderFrameStream(deadline, workspace, blender, func(ordinal uint32, pngBytes []byte) error {
		if ordinal >= manifest.Output.FrameCount {
			return errors.New("Blender emitted an unplanned physical frame ordinal")
		}
		if !isPhysicalRGBA8PNG(pngBytes) {
			return errors.New("Blender motion frame is not an 8-bit RGBA PNG")
		}
		frame, err := physicalPNGToRGB(pngBytes, int(manifest.Output.Width), int(manifest.Output.Height))
		if err != nil {
			return err
		}
		if err := encoder.writeFrame(deadline, frame); err != nil {
			return err
		}
		result.frameRGBDigests = append(result.frameRGBDigests, PhysicalFrameRGBDigest{OutputOrdinal: ordinal, RGBSHA256: digest(frame)})
		return nil
	}, manifest.Output.FrameCount); err != nil {
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

func runPhysicalBlenderFrameStream(ctx context.Context, workspace, blender string, onFrame func(uint32, []byte) error, expected uint32) error {
	if ctx == nil || onFrame == nil || expected == 0 || expected > PhysicalMotionMaxFramesV2 {
		return errors.New("physical motion frame stream arguments are invalid")
	}
	if !filepath.IsAbs(blender) || filepath.Clean(blender) != blender {
		return errors.New("managed Blender path is not absolute and clean")
	}
	binary, err := readRegularPath(blender, 512<<20)
	if err != nil || digest(binary) != PhysicalBlenderBinarySHA {
		return errors.New("managed Blender executable digest differs from profile")
	}
	readEnd, writeEnd, err := os.Pipe()
	if err != nil {
		return err
	}
	defer readEnd.Close()
	defer writeEnd.Close()
	if err := ensurePhysicalBlenderDirectories(workspace); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, blender, physicalBlenderArgs(workspace)...)
	cmd.Dir = workspace
	cmd.Env = physicalBlenderEnvironment(workspace)
	cmd.ExtraFiles = []*os.File{writeEnd}
	var stdout, stderr physicalBlenderLogBuffer
	stdout.limit, stderr.limit = physicalCommandOutMax, physicalCommandErrMax
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err = cmd.Start(); err != nil {
		return err
	}
	_ = writeEnd.Close()
	for ordinal := uint32(0); ordinal < expected; ordinal++ {
		gotOrdinal, data, readErr := readPhysicalFrame(readEnd)
		if readErr != nil {
			_ = cmd.Process.Kill()
			waitErr := cmd.Wait()
			if ctx.Err() != nil {
				return fmt.Errorf("physical Blender frame stream ended: %w", ctx.Err())
			}
			return fmt.Errorf("physical Blender frame stream ended before ordinal %d: %w (%v): %s", ordinal, readErr, waitErr, truncateUTF8Tail(stderr.data.String(), 1024))
		}
		if gotOrdinal != ordinal {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return fmt.Errorf("physical Blender emitted frame ordinal %d, expected %d", gotOrdinal, ordinal)
		}
		if err := onFrame(ordinal, data); err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return fmt.Errorf("consume physical frame %d: %w", ordinal, err)
		}
	}
	var extra [1]byte
	if n, readErr := readEnd.Read(extra[:]); n != 0 || readErr != io.EOF {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		if ctx.Err() != nil {
			return fmt.Errorf("physical Blender frame stream ended: %w", ctx.Err())
		}
		return errors.New("physical Blender emitted trailing frame protocol bytes")
	}
	if err := cmd.Wait(); err != nil {
		return physicalBlenderFailure(err, ctx.Err(), stderr.data.String())
	}
	return nil
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
