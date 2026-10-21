package rendering

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

const (
	physicalProfileName   = "blender-profile.json"
	physicalAdapterMax    = 1 << 20
	physicalCommandOutMax = 64 << 10
	physicalCommandErrMax = 16 << 10
)

//go:embed physical_blender.py
var physicalBlenderAdapter []byte

// PhysicalAdapterBytes returns a copy of the exact reviewed adapter bytes so
// the immutable job closure can pin and stage them after native claim.
func PhysicalAdapterBytes() []byte { return bytes.Clone(physicalBlenderAdapter) }

func IsPhysicalWorkerInvocation(args []string) bool {
	return len(args) > 0 && args[0] == "physical-render-worker"
}

// RunPhysicalWorkerCommand accepts only the private fixed manifest form.
func RunPhysicalWorkerCommand(args []string) error {
	if len(args) != 3 || args[0] != "physical-render-worker" || args[1] != "--manifest" {
		return errors.New("physical-render-worker requires exactly --manifest <absolute attempt manifest>")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return RunPhysicalManifest(ctx, args[2])
}

// RunPhysicalManifest runs one fixed still render inside the caller's owned
// native operation. It owns no queue, lease, retry, or process supervisor.
func RunPhysicalManifest(ctx context.Context, manifestPath string) (retErr error) {
	if ctx == nil || ctx.Err() != nil || runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return errors.New("physical worker requires a live context on Linux/amd64")
	}
	if !filepath.IsAbs(manifestPath) || filepath.Clean(manifestPath) != manifestPath || filepath.Base(manifestPath) != PhysicalManifestName {
		return errors.New("physical worker manifest must use its fixed filename in an absolute attempt root")
	}
	workspace := filepath.Dir(manifestPath)
	if err := validateWorkspacePath(workspace); err != nil {
		return err
	}
	root, err := os.OpenRoot(workspace)
	if err != nil {
		return fmt.Errorf("open physical attempt root: %w", err)
	}
	defer root.Close()
	manifestBytes, err := readBoundedRegular(ctx, root, PhysicalManifestName, physicalManifestMax, 0)
	if err != nil {
		return fmt.Errorf("read physical manifest: %w", err)
	}
	manifest, err := decodeCanonicalPhysicalRenderManifest(manifestBytes)
	if err != nil {
		return err
	}
	if manifest.Profile.AdapterSHA256 != digest(physicalBlenderAdapter) {
		return errors.New("embedded physical adapter differs from the immutable profile")
	}
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve physical worker executable: %w", err)
	}
	workerBytes, err := readRegularPath(executable, 512<<20)
	if err != nil || digest(workerBytes) != manifest.WorkerSHA256 {
		return errors.New("physical worker executable differs from immutable job closure")
	}
	if err := validatePhysicalProfile(manifest.Profile, manifest.Output); err != nil {
		return err
	}
	if err := ensurePhysicalOutputAbsent(root, manifest.Output.OutputName); err != nil {
		return err
	}
	if err := ensurePhysicalOutputAbsent(root, PhysicalReceiptName); err != nil {
		return err
	}
	if err := ensurePhysicalOutputAbsent(root, physicalProfileName); err != nil {
		return err
	}
	if err := stagePhysicalAdapter(root); err != nil {
		return err
	}
	profileBefore, profile, err := ResolvePhysicalBlenderProfile(ctx)
	if err != nil {
		return err
	}
	if err := matchPhysicalToolProfile(manifest.Profile, profile); err != nil {
		return err
	}
	if err := writePhysicalRoot(root, physicalProfileName, profileBefore); err != nil {
		return err
	}
	if err := stagePhysicalAssets(ctx, root, manifest); err != nil {
		return err
	}
	deviceBytes, err := readPhysicalAsset(root, manifest, "device_geometry")
	if err != nil {
		return err
	}
	device, err := decodeDevice(manifest.Device.CanonicalJSON)
	if err != nil {
		return err
	}
	mapping, err := ResolvePhysicalScreenGeometry(deviceBytes, device, manifest.Screen.DisplayID)
	if err != nil || mapping.BlenderUVLayer != PhysicalBlenderUV0LayerName || mapping.PrimitiveIndex != manifest.Screen.PrimitiveIndex || mapping.MaterialIndex != manifest.Screen.MaterialSlot {
		return errors.New("physical screen mapping differs from pinned GLB and candidate")
	}
	if manifest.Version == PhysicalArticulatedManifestVersion {
		if _, err := physicalArticulationFromManifest(manifest, deviceBytes); err != nil {
			return fmt.Errorf("validate articulated physical input before Blender: %w", err)
		}
	}
	if err := stagePhysicalAdapterInput(ctx, root, manifest, mapping); err != nil {
		return err
	}
	var motion *physicalMotionExecution
	if manifest.Output.Kind == "motion_mp4" {
		var result physicalMotionExecution
		var runErr error
		if manifest.Screen.Recording != nil {
			result, runErr = runPhysicalRecordingMotion(ctx, root, workspace, profile.GetBlender().GetPath(), manifest)
		} else {
			result, runErr = runPhysicalMotion(ctx, root, workspace, profile.GetBlender().GetPath(), manifest)
		}
		if runErr != nil {
			return runErr
		}
		motion = &result
	} else if err := runPhysicalBlender(ctx, workspace, profile.GetBlender().GetPath(), manifest.Profile.WallTimeSeconds); err != nil {
		return err
	}
	if err := verifyPhysicalToolUnchanged(ctx, profile, profile.GetBlender().GetPath()); err != nil {
		return err
	}
	receipt, err := buildPhysicalReceipt(root, manifest, manifestBytes, profileBefore, mapping, motion)
	if err != nil {
		return err
	}
	receiptBytes, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	if len(receiptBytes) > 2<<20 {
		return errors.New("physical render receipt exceeds its limit")
	}
	return writePhysicalRoot(root, PhysicalReceiptName, append(receiptBytes, '\n'))
}

func stagePhysicalAdapter(root *os.Root) error {
	if len(physicalBlenderAdapter) == 0 || len(physicalBlenderAdapter) > physicalAdapterMax {
		return errors.New("embedded physical Blender adapter is empty or oversized")
	}
	if existing, err := readBoundedRegular(context.Background(), root, PhysicalAdapterName, physicalAdapterMax, 0); err == nil {
		if digest(existing) != digest(physicalBlenderAdapter) {
			return errors.New("staged physical adapter differs from embedded adapter")
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return writePhysicalRoot(root, PhysicalAdapterName, physicalBlenderAdapter)
}

func stagePhysicalAdapterInput(ctx context.Context, root *os.Root, manifest PhysicalRenderManifest, mapping PhysicalScreenGeometryBinding) error {
	canonical, _, err := CanonicalPhysicalRenderManifest(manifest)
	if err != nil {
		return err
	}
	var value map[string]json.RawMessage
	if err := json.Unmarshal(canonical, &value); err != nil {
		return err
	}
	value["device"] = json.RawMessage(manifest.Device.CanonicalJSON)
	value["scene"] = json.RawMessage(manifest.Scene.CanonicalJSON)
	value["motion"] = json.RawMessage(manifest.Motion.CanonicalJSON)
	encodedMapping, err := json.Marshal(mapping)
	if err != nil {
		return err
	}
	value["screen_geometry"] = encodedMapping
	var sceneInventory *PhysicalGLBInventory
	for _, asset := range manifest.Assets {
		if asset.Role != "scene_geometry" {
			continue
		}
		sceneBytes, readErr := readBoundedRegular(ctx, root, asset.Name, physicalGLBMaxBytes, asset.Reference.SizeBytes)
		if readErr != nil || digest(sceneBytes) != asset.Reference.SHA256 {
			return errors.New("scene geometry changed before adapter input was frozen")
		}
		inventory, inspectErr := InspectPhysicalGLB(sceneBytes)
		if inspectErr != nil {
			return inspectErr
		}
		sceneInventory = &inventory
		break
	}
	if sceneInventory == nil {
		return errors.New("scene geometry inventory is missing")
	}
	encodedInventory, err := json.Marshal(sceneInventory)
	if err != nil {
		return err
	}
	value["scene_inventory"] = encodedInventory
	data, err := json.Marshal(value)
	if err != nil || len(data) > physicalManifestMax {
		return errors.New("physical adapter input exceeds its bound")
	}
	return writePhysicalRoot(root, PhysicalAdapterInputName, data)
}

func stagePhysicalAssets(ctx context.Context, root *os.Root, manifest PhysicalRenderManifest) error {
	var total int64
	for _, asset := range manifest.Assets {
		if err := ctx.Err(); err != nil {
			return err
		}
		maximum := int64(physicalInputBytesMax)
		if asset.Role == "device_geometry" || asset.Role == "scene_geometry" {
			maximum = physicalGLBMaxBytes
		}
		data, err := readBoundedRegular(ctx, root, asset.Name, maximum, asset.Reference.SizeBytes)
		if err != nil || digest(data) != asset.Reference.SHA256 {
			return fmt.Errorf("staged asset %q differs from pinned reference", asset.Name)
		}
		total += int64(len(data))
		if total > physicalInputBytesMax {
			return errors.New("physical staged asset bytes exceed aggregate bound")
		}
		if asset.Role == "device_geometry" || asset.Role == "scene_geometry" {
			prefix := "device"
			if asset.Role == "scene_geometry" {
				prefix = "scene"
			}
			prepared, _, err := NormalizePhysicalGLBNames(data, prefix)
			if err != nil {
				return err
			}
			if err := writePhysicalRoot(root, prefix+"-normalized.glb", prepared); err != nil {
				return err
			}
		}
	}
	if manifest.Screen.Recording != nil {
		binding := manifest.Screen.Recording.Inspection.Source
		source, err := root.Open(binding.Name)
		if err != nil {
			return errors.New("staged physical recording source is missing")
		}
		err = verifyInspectionSourceUnchanged(ctx, source, InspectionSource{Name: binding.Name, SHA256: binding.SHA256, SizeBytes: binding.SizeBytes})
		closeErr := source.Close()
		if err != nil || closeErr != nil {
			return errors.New("staged physical recording source differs from its accepted inspection binding")
		}
		if total+binding.SizeBytes > physicalInputBytesMax {
			return errors.New("physical staged inputs exceed aggregate byte bound")
		}
	} else {
		screen, err := readBoundedRegular(ctx, root, manifest.Screen.Raster.Name, physicalScreenBytesMax, manifest.Screen.Raster.SizeBytes)
		if err != nil || digest(screen) != manifest.Screen.Raster.SHA256 {
			return errors.New("staged screen raster differs from its immutable binding")
		}
		if total+int64(len(screen)) > physicalInputBytesMax {
			return errors.New("physical staged inputs exceed aggregate byte bound")
		}
	}
	return nil
}

func runPhysicalBlender(parent context.Context, workspace, blender string, wallTimeSeconds uint32) error {
	if wallTimeSeconds == 0 || wallTimeSeconds > PhysicalStillWallTimeSeconds {
		return errors.New("physical still wall-time limit is outside the closed profile")
	}
	if !filepath.IsAbs(blender) || filepath.Clean(blender) != blender {
		return errors.New("managed Blender path is not absolute and clean")
	}
	info, err := os.Lstat(blender)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 {
		return errors.New("managed Blender executable is not a regular file")
	}
	binary, err := readRegularPath(blender, 512<<20)
	if err != nil || digest(binary) != PhysicalBlenderBinarySHA {
		return errors.New("managed Blender executable digest differs from profile")
	}
	ctx, cancel := context.WithTimeout(parent, time.Duration(wallTimeSeconds)*time.Second)
	defer cancel()
	if err := ensurePhysicalBlenderDirectories(workspace); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, blender, physicalBlenderArgs(workspace)...)
	cmd.Dir = workspace
	cmd.Env = physicalBlenderEnvironment(workspace)
	var stdout, stderr physicalBlenderLogBuffer
	stdout.limit, stderr.limit = physicalCommandOutMax, physicalCommandErrMax
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return physicalBlenderFailure(err, ctx.Err(), stderr.data.String())
	}
	return nil
}

func physicalBlenderArgs(workspace string) []string {
	return []string{"--background", "--factory-startup", "--disable-autoexec", "--python-use-system-env", "--threads", fmt.Sprint(PhysicalOperationThreads), "--python-exit-code", "17", "--python", filepath.Join(workspace, PhysicalAdapterName), "--", filepath.Join(workspace, PhysicalAdapterInputName)}
}

func ensurePhysicalBlenderDirectories(workspace string) error {
	for _, name := range []string{"home", "tmp", "config", "scripts", "data", "pycache"} {
		if err := os.MkdirAll(filepath.Join(workspace, name), 0o700); err != nil {
			return err
		}
	}
	return nil
}

func physicalBlenderFailure(runErr, contextErr error, stderr string) error {
	diagnostic := truncateUTF8Tail(strings.TrimSpace(stderr), 2048)
	if contextErr != nil {
		return fmt.Errorf("physical Blender operation ended: %w: %s", contextErr, diagnostic)
	}
	return fmt.Errorf("physical Blender render failed: %w: %s", runErr, diagnostic)
}

// truncateUTF8Tail preserves the most recent part of a native diagnostic while
// keeping the result within a byte bound and on a UTF-8 rune boundary.
func truncateUTF8Tail(value string, maximum int) string {
	if maximum <= 0 {
		return ""
	}
	value = strings.ToValidUTF8(value, "\uFFFD")
	if len(value) <= maximum {
		return value
	}
	start := len(value) - maximum
	for start < len(value) && !utf8.RuneStart(value[start]) {
		start++
	}
	return value[start:]
}

func physicalBlenderEnvironment(workspace string) []string {
	return []string{"HOME=" + filepath.Join(workspace, "home"), "TMPDIR=" + filepath.Join(workspace, "tmp"), "BLENDER_USER_CONFIG=" + filepath.Join(workspace, "config"), "BLENDER_USER_SCRIPTS=" + filepath.Join(workspace, "scripts"), "BLENDER_USER_DATAFILES=" + filepath.Join(workspace, "data"), "PYTHONPYCACHEPREFIX=" + filepath.Join(workspace, "pycache"), "PYTHONDONTWRITEBYTECODE=1", "LANG=C", "LC_ALL=C", "OMP_NUM_THREADS=2"}
}

// physicalBlenderLogBuffer bounds retained diagnostics without closing os/exec's
// copy path when Blender produces verbose Cycles logs.
type physicalBlenderLogBuffer struct {
	data  bytes.Buffer
	limit int
}

func (b *physicalBlenderLogBuffer) Write(p []byte) (int, error) {
	if b.limit <= 0 {
		return len(p), nil
	}
	if len(p) >= b.limit {
		b.data.Reset()
		_, _ = b.data.Write(p[len(p)-b.limit:])
		return len(p), nil
	}
	if len(p) > b.limit-b.data.Len() {
		drop := len(p) - (b.limit - b.data.Len())
		kept := bytes.Clone(b.data.Bytes()[drop:])
		b.data.Reset()
		_, _ = b.data.Write(kept)
	}
	_, _ = b.data.Write(p)
	return len(p), nil
}

func verifyPhysicalToolUnchanged(ctx context.Context, before *cliv1.BlenderToolProfile, blenderPath string) error {
	_, profile, err := ResolvePhysicalBlenderProfile(ctx)
	if err != nil || !samePhysicalProfileObservation(before, profile) {
		return errors.New("managed Blender profile changed during physical render")
	}
	if profile.GetBlender().GetPath() != blenderPath {
		return errors.New("managed Blender executable path changed during physical render")
	}
	return nil
}

func samePhysicalProfileObservation(a, b *cliv1.BlenderToolProfile) bool {
	if a == nil || b == nil || a.GetSchemaVersion() != b.GetSchemaVersion() || a.GetProfileId() != b.GetProfileId() || a.GetState() != b.GetState() || a.GetArtifactIdentity() != b.GetArtifactIdentity() || a.GetExpectedTreeSha256() != b.GetExpectedTreeSha256() || a.GetOs() != b.GetOs() || a.GetArchitecture() != b.GetArchitecture() {
		return false
	}
	x, y := a.GetBlender(), b.GetBlender()
	return x != nil && y != nil && x.GetPath() == y.GetPath() && x.GetSha256() == y.GetSha256() && x.GetSizeBytes() == y.GetSizeBytes() && x.GetVersion() == y.GetVersion() && x.GetArchiveMemberPath() == y.GetArchiveMemberPath()
}

func matchPhysicalToolProfile(expected PhysicalRenderProfile, profile *cliv1.BlenderToolProfile) error {
	if err := validatePhysicalBlenderProfile(profile); err != nil {
		return err
	}
	if profile.GetBlender().GetPath() == "" || profile.GetArtifactIdentity() != expected.BlenderArtifactID || profile.GetExpectedTreeSha256() != expected.BlenderTreeSHA256 || profile.GetBlender().GetSha256() != expected.BlenderBinarySHA256 || profile.GetBlender().GetVersion() != expected.BlenderVersion {
		return errors.New("resolved Blender profile differs from immutable render profile")
	}
	return nil
}

func buildPhysicalReceipt(root *os.Root, manifest PhysicalRenderManifest, manifestBytes, profileBytes []byte, mapping PhysicalScreenGeometryBinding, motion *physicalMotionExecution) (PhysicalRenderReceipt, error) {
	device, err := decodeDevice(manifest.Device.CanonicalJSON)
	if err != nil {
		return PhysicalRenderReceipt{}, err
	}
	scene, err := decodeScene(manifest.Scene.CanonicalJSON)
	if err != nil {
		return PhysicalRenderReceipt{}, err
	}
	inputs := make([]PhysicalInputReceipt, 0, len(manifest.Assets))
	for _, asset := range manifest.Assets {
		data, err := readBoundedRegular(context.Background(), root, asset.Name, physicalInputBytesMax, asset.Reference.SizeBytes)
		if err != nil {
			return PhysicalRenderReceipt{}, err
		}
		preparedName, prepared := asset.Name, data
		input := PhysicalInputReceipt{Role: asset.Role, BindingID: asset.BindingID, Name: asset.Name, SourceSHA256: asset.Reference.SHA256, SourceSizeBytes: asset.Reference.SizeBytes}
		if asset.Role == "device_geometry" || asset.Role == "scene_geometry" {
			prefix := "device"
			if asset.Role == "scene_geometry" {
				prefix = "scene"
			}
			preparedData, inventory, err := NormalizePhysicalGLBNames(data, prefix)
			if err != nil {
				return PhysicalRenderReceipt{}, err
			}
			prepared = preparedData
			preparedName = prefix + "-normalized.glb"
			invBytes, _ := json.Marshal(inventory)
			input.InventorySHA256 = digest(invBytes)
			if prefix == "device" {
				input.Parts = physicalPartMappings(prefix, device.Parts, inventory)
			} else {
				input.Parts = physicalScenePartMappings(scene, inventory)
			}
			input.Primitives = physicalPrimitiveMappings(prefix, inventory)
		}
		input.PreparedName = preparedName
		input.PreparedSHA256 = digest(prepared)
		input.PreparedSizeBytes = int64(len(prepared))
		inputs = append(inputs, input)
	}
	screenName, screenSHA, screenSize := manifest.Screen.Raster.Name, manifest.Screen.Raster.SHA256, manifest.Screen.Raster.SizeBytes
	if manifest.Screen.Recording != nil {
		source := manifest.Screen.Recording.Inspection.Source
		screenName, screenSHA, screenSize = source.Name, source.SHA256, source.SizeBytes
	}
	screenReceipt := PhysicalScreenReceipt{DisplayID: manifest.Screen.DisplayID, OriginalMediaID: manifest.Screen.OriginalMediaID, OriginalMediaSHA256: manifest.Screen.OriginalMediaSHA256, Name: screenName, SHA256: screenSHA, SizeBytes: screenSize, Width: manifest.Screen.Width, Height: manifest.Screen.Height, PresentationProfile: manifest.Screen.PresentationProfile, PresentationVersion: manifest.Screen.PresentationVersion, ConfigSHA256: manifest.Screen.ConfigSHA256, LayoutSHA256: manifest.Screen.LayoutSHA256, Mapping: mapping}
	output := PhysicalOutputReceipt{Kind: manifest.Output.Kind, Name: manifest.Output.OutputName, Width: manifest.Output.Width, Height: manifest.Output.Height, PixelFormat: manifest.Output.PixelFormat}
	receipt := PhysicalRenderReceipt{Version: manifest.Version, Status: "succeeded", JobID: manifest.JobID, AttemptID: manifest.AttemptID, InputFingerprint: manifest.InputFingerprint, WorkerSHA256: manifest.WorkerSHA256, ManifestSHA256: digest(manifestBytes), BlenderProfileSHA256: digest(profileBytes), Profile: manifest.Profile, Inputs: inputs, Screen: screenReceipt, Output: output, CompletedAt: time.Now().UTC()}
	if manifest.Version == PhysicalArticulatedManifestVersion {
		deviceBytes, err := readPhysicalAsset(root, manifest, "device_geometry")
		if err != nil {
			return PhysicalRenderReceipt{}, err
		}
		articulation, err := physicalArticulationFromManifest(manifest, deviceBytes)
		if err != nil {
			return PhysicalRenderReceipt{}, err
		}
		receipt.Articulation = &articulation
	}
	if manifest.Output.Kind == "still_png" {
		outputBytes, err := readBoundedRegular(context.Background(), root, manifest.Output.OutputName, physicalOutputBytesMax, 0)
		if err != nil {
			return PhysicalRenderReceipt{}, fmt.Errorf("read physical output: %w", err)
		}
		decoded, format, err := image.Decode(bytes.NewReader(outputBytes))
		if err != nil || format != "png" || decoded.Bounds().Dx() != int(manifest.Output.Width) || decoded.Bounds().Dy() != int(manifest.Output.Height) || !isPhysicalRGBA8PNG(outputBytes) {
			return PhysicalRenderReceipt{}, errors.New("physical output failed full RGBA8 PNG decode or dimensions")
		}
		cfg, _, _ := image.DecodeConfig(bytes.NewReader(outputBytes))
		receipt.Output.SHA256, receipt.Output.SizeBytes = digest(outputBytes), int64(len(outputBytes))
		receipt.Output.Width, receipt.Output.Height = uint32(cfg.Width), uint32(cfg.Height)
	} else {
		if motion == nil || motion.outputSize <= 0 || !physicalDigestPattern.MatchString(motion.outputSHA256) || !motion.native.FullDecodeVerified {
			return PhysicalRenderReceipt{}, errors.New("physical motion has no verified encoded output facts")
		}
		receipt.VideoToolProfileSHA256 = digest(motion.profileBytes)
		receipt.VideoToolArtifactID = manifest.Output.VideoToolArtifactID
		receipt.Output.SHA256, receipt.Output.SizeBytes = motion.outputSHA256, motion.outputSize
		receipt.Output.FrameCount = motion.native.FrameCount
		receipt.Output.RateNumerator, receipt.Output.RateDenominator = uint32(motion.native.FrameRate.Num), uint32(motion.native.FrameRate.Den)
		receipt.Output.FrameRGBDigests = append([]PhysicalFrameRGBDigest(nil), motion.frameRGBDigests...)
		receipt.Output.SourceFrameFacts = append([]PhysicalRecordingFrameFact(nil), motion.sourceFrameFacts...)
		receipt.Output.NativeFacts = physicalMotionNativeFactsJSON(motion.native)
	}
	return receipt, nil
}

func writePhysicalRoot(root *os.Root, name string, data []byte) error {
	if !isSafePhysicalName(name) || len(data) == 0 {
		return errors.New("physical staged file name or bytes are invalid")
	}
	f, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return nil
}

func readRegularPath(path string, max int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > max {
		return nil, errors.New("physical tool file exceeds bounds or is not regular")
	}
	b, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil || int64(len(b)) != info.Size() || int64(len(b)) > max {
		return nil, errors.New("physical tool file size changed while reading")
	}
	return b, nil
}

func validatePhysicalScreenImage(data []byte, width, height int) error {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || format != "png" || cfg.Width != width || cfg.Height != height {
		return errors.New("physical screen image is invalid")
	}
	return nil
}
