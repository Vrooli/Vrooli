package rendering

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"time"

	"device-studio/internal/composition"
	"device-studio/internal/devices"
	"device-studio/internal/physical"
	"device-studio/internal/scenes"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

const physicalOutputBytesMax = 64 << 20

type PhysicalRenderReceipt struct {
	Version                int                    `json:"version"`
	Status                 string                 `json:"status"`
	JobID                  string                 `json:"job_id"`
	AttemptID              string                 `json:"attempt_id"`
	InputFingerprint       string                 `json:"input_fingerprint"`
	WorkerSHA256           string                 `json:"worker_sha256"`
	ManifestSHA256         string                 `json:"manifest_sha256"`
	BlenderProfileSHA256   string                 `json:"blender_profile_sha256"`
	VideoToolProfileSHA256 string                 `json:"video_tool_profile_sha256,omitempty"`
	VideoToolArtifactID    string                 `json:"video_tool_artifact_identity,omitempty"`
	Profile                PhysicalRenderProfile  `json:"profile"`
	Inputs                 []PhysicalInputReceipt `json:"inputs"`
	Screen                 PhysicalScreenReceipt  `json:"screen"`
	Output                 PhysicalOutputReceipt  `json:"output"`
	Articulation           *PhysicalJointBinding  `json:"articulation,omitempty"`
	CompletedAt            time.Time              `json:"completed_at"`
}

type PhysicalInputReceipt struct {
	Role              string                     `json:"role"`
	BindingID         string                     `json:"binding_id"`
	Name              string                     `json:"name"`
	SourceSHA256      string                     `json:"source_sha256"`
	SourceSizeBytes   int64                      `json:"source_size_bytes"`
	PreparedName      string                     `json:"prepared_name"`
	PreparedSHA256    string                     `json:"prepared_sha256"`
	PreparedSizeBytes int64                      `json:"prepared_size_bytes"`
	InventorySHA256   string                     `json:"inventory_sha256,omitempty"`
	Parts             []PhysicalPartMapping      `json:"parts,omitempty"`
	Primitives        []PhysicalPrimitiveMapping `json:"primitives,omitempty"`
}

type PhysicalPartMapping struct {
	PartID      string      `json:"part_id"`
	Role        string      `json:"role"`
	NodeIndex   uint32      `json:"node_index"`
	NodeName    string      `json:"node_name"`
	ObjectName  string      `json:"object_name"`
	ParentIndex *uint32     `json:"parent_index,omitempty"`
	MeshIndex   *uint32     `json:"mesh_index,omitempty"`
	LocalFrame  physicalTRS `json:"local_frame"`
}

type physicalTRS struct {
	Translation [3]float64 `json:"translation_m"`
	Rotation    [4]float64 `json:"rotation_xyzw"`
	Scale       [3]float64 `json:"scale"`
}

type PhysicalPrimitiveMapping struct {
	MeshIndex      uint32     `json:"mesh_index"`
	PrimitiveIndex uint32     `json:"primitive_index"`
	MaterialIndex  uint32     `json:"material_index"`
	MaterialName   string     `json:"material_name"`
	Mode           uint32     `json:"mode"`
	UV0            bool       `json:"has_uv0"`
	UVMin          [2]float64 `json:"uv_min"`
	UVMax          [2]float64 `json:"uv_max"`
	VertexCount    uint32     `json:"vertex_count"`
	TriangleCount  uint32     `json:"triangle_count"`
}

type PhysicalScreenReceipt struct {
	DisplayID           string                        `json:"display_id"`
	OriginalMediaID     string                        `json:"original_media_id"`
	OriginalMediaSHA256 string                        `json:"original_media_sha256"`
	Name                string                        `json:"name"`
	SHA256              string                        `json:"sha256"`
	SizeBytes           int64                         `json:"size_bytes"`
	Width               int                           `json:"width"`
	Height              int                           `json:"height"`
	PresentationProfile string                        `json:"presentation_profile"`
	PresentationVersion string                        `json:"presentation_version"`
	ConfigSHA256        string                        `json:"config_sha256"`
	LayoutSHA256        string                        `json:"layout_sha256"`
	Mapping             PhysicalScreenGeometryBinding `json:"mapping"`
}

type PhysicalOutputReceipt struct {
	Kind             string                       `json:"kind"`
	Name             string                       `json:"name"`
	SHA256           string                       `json:"sha256"`
	SizeBytes        int64                        `json:"size_bytes"`
	Width            uint32                       `json:"width"`
	Height           uint32                       `json:"height"`
	PixelFormat      string                       `json:"pixel_format"`
	FrameCount       uint32                       `json:"frame_count,omitempty"`
	RateNumerator    uint32                       `json:"rate_numerator,omitempty"`
	RateDenominator  uint32                       `json:"rate_denominator,omitempty"`
	FrameRGBDigests  []PhysicalFrameRGBDigest     `json:"frame_rgb_digests,omitempty"`
	SourceFrameFacts []PhysicalRecordingFrameFact `json:"source_frame_facts,omitempty"`
	NativeFacts      json.RawMessage              `json:"native_facts,omitempty"`
}

// PhysicalFrameRGBDigest binds the exact RGB24 bytes handed to the encoder at
// one ordinal. The receipt's ManifestSHA256 binds that ordinal to the exact
// rational pose/output-time schedule without duplicating it here.
type PhysicalFrameRGBDigest struct {
	OutputOrdinal uint32 `json:"output_ordinal"`
	RGBSHA256     string `json:"rgb_sha256"`
}

// PhysicalRecordingFrameFact links each final output ordinal to the source
// inspection-clock sample and the exact composed screen PNG consumed by Blender.
type PhysicalRecordingFrameFact struct {
	OutputOrdinal uint32 `json:"output_ordinal"`
	SourceOrdinal uint32 `json:"source_ordinal"`
	SourcePTS     int64  `json:"source_pts"`
	ScreenSHA256  string `json:"screen_sha256"`
}

// VerifyPhysicalRenderReceipt independently binds the worker's fixed outputs
// to the immutable manifest. It does not spawn Blender or another renderer.
func VerifyPhysicalRenderReceipt(attemptRoot string, expected PhysicalRenderManifest) (PhysicalRenderReceipt, error) {
	var receipt PhysicalRenderReceipt
	if err := ValidatePhysicalRenderManifest(expected); err != nil {
		return receipt, fmt.Errorf("invalid expected physical manifest: %w", err)
	}
	if err := validateWorkspacePath(attemptRoot); err != nil {
		return receipt, err
	}
	root, err := os.OpenRoot(attemptRoot)
	if err != nil {
		return receipt, fmt.Errorf("open physical attempt root: %w", err)
	}
	defer root.Close()
	manifestBytes, err := readBoundedRegular(context.Background(), root, PhysicalManifestName, physicalManifestMax, 0)
	if err != nil {
		return receipt, fmt.Errorf("read physical render manifest: %w", err)
	}
	manifest, err := decodeCanonicalPhysicalRenderManifest(manifestBytes)
	if err != nil {
		return receipt, err
	}
	expectedBytes, _, err := CanonicalPhysicalRenderManifest(expected)
	if err != nil || !bytes.Equal(manifestBytes, expectedBytes) && !bytes.Equal(manifestBytes, append(bytes.Clone(expectedBytes), '\n')) {
		return receipt, errors.New("staged physical manifest differs from immutable job input")
	}
	receiptBytes, err := readBoundedRegular(context.Background(), root, PhysicalReceiptName, 2<<20, 0)
	if err != nil {
		return receipt, fmt.Errorf("read physical render receipt: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(receiptBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil || decoder.Decode(new(any)) != io.EOF {
		return PhysicalRenderReceipt{}, errors.New("physical render receipt is malformed")
	}
	canonical, err := json.Marshal(receipt)
	if err != nil || !bytes.Equal(receiptBytes, canonical) && !bytes.Equal(receiptBytes, append(canonical, '\n')) {
		return PhysicalRenderReceipt{}, errors.New("physical render receipt is not canonical JSON")
	}
	manifestSHA := digest(manifestBytes)
	if receipt.Version != manifest.Version || receipt.Status != "succeeded" || receipt.JobID != manifest.JobID || receipt.AttemptID != manifest.AttemptID ||
		receipt.InputFingerprint != manifest.InputFingerprint || receipt.WorkerSHA256 != manifest.WorkerSHA256 || receipt.ManifestSHA256 != manifestSHA ||
		receipt.Profile != manifest.Profile || len(receipt.Inputs) != len(manifest.Assets) || receipt.CompletedAt.IsZero() {
		return PhysicalRenderReceipt{}, errors.New("physical receipt identity or pinned profile does not match the manifest")
	}
	profileBytes, err := readBoundedRegular(context.Background(), root, physicalProfileName, physicalProfileOutputMax, 0)
	if err != nil || digest(profileBytes) != receipt.BlenderProfileSHA256 {
		return PhysicalRenderReceipt{}, errors.New("receipt does not bind its recorded managed Blender profile observation")
	}
	profile := new(cliv1.BlenderToolProfile)
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(profileBytes, profile); err != nil || matchPhysicalToolProfile(manifest.Profile, profile) != nil {
		return PhysicalRenderReceipt{}, errors.New("staged Blender profile observation differs from the pinned render profile")
	}
	if err := verifyPhysicalInputReceipts(root, manifest, receipt.Inputs); err != nil {
		return PhysicalRenderReceipt{}, err
	}
	if manifest.Version == PhysicalArticulatedManifestVersion {
		deviceBytes, readErr := readPhysicalAsset(root, manifest, "device_geometry")
		if readErr != nil {
			return PhysicalRenderReceipt{}, readErr
		}
		want, bindingErr := physicalArticulationFromManifest(manifest, deviceBytes)
		if bindingErr != nil || receipt.Articulation == nil || !jsonValuesEqual(*receipt.Articulation, want) {
			return PhysicalRenderReceipt{}, errors.New("physical receipt articulation differs from the immutable candidate and GLB")
		}
	} else if receipt.Articulation != nil {
		return PhysicalRenderReceipt{}, errors.New("legacy physical receipt unexpectedly contains articulation facts")
	}
	if err := verifyPhysicalScreenReceipt(root, manifest, receipt.Screen); err != nil {
		return PhysicalRenderReceipt{}, err
	}
	if manifest.Output.Kind == "still_png" {
		if validatePhysicalOutputKindFacts(manifest.Output.Kind, receipt.Output) != nil || receipt.VideoToolProfileSHA256 != "" || receipt.VideoToolArtifactID != "" {
			return PhysicalRenderReceipt{}, errors.New("still receipt unexpectedly includes video tool facts")
		}
		if err := verifyPhysicalPNGOutput(root, manifest, receipt.Output); err != nil {
			return PhysicalRenderReceipt{}, err
		}
	} else if err := verifyPhysicalVideoOutput(root, manifest, receipt); err != nil {
		return PhysicalRenderReceipt{}, err
	}
	return receipt, nil
}

func decodeCanonicalPhysicalRenderManifest(data []byte) (PhysicalRenderManifest, error) {
	var manifest PhysicalRenderManifest
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&manifest); err != nil {
		return manifest, fmt.Errorf("decode physical render manifest: %w", err)
	}
	if d.Decode(new(any)) != io.EOF {
		return manifest, errors.New("physical render manifest contains trailing data")
	}
	canonical, _, err := CanonicalPhysicalRenderManifest(manifest)
	if err != nil || !bytes.Equal(data, canonical) && !bytes.Equal(data, append(canonical, '\n')) {
		return manifest, errors.New("physical render manifest is not canonical")
	}
	return manifest, nil
}

func verifyPhysicalInputReceipts(root *os.Root, manifest PhysicalRenderManifest, receipts []PhysicalInputReceipt) error {
	ctx := context.Background()
	for i, asset := range manifest.Assets {
		receipt := receipts[i]
		ref := asset.Reference
		if receipt.Role != asset.Role || receipt.BindingID != asset.BindingID || receipt.Name != asset.Name || receipt.SourceSHA256 != ref.SHA256 || receipt.SourceSizeBytes != ref.SizeBytes {
			return fmt.Errorf("physical receipt asset %d does not match the pinned source reference", i)
		}
		maximum := int64(physicalInputBytesMax)
		if asset.Role == "device_geometry" || asset.Role == "scene_geometry" {
			maximum = 128 << 20
		}
		data, err := readBoundedRegular(ctx, root, asset.Name, maximum, ref.SizeBytes)
		if err != nil || sha256Bytes(data) != ref.SHA256 {
			return fmt.Errorf("physical staged asset %q changed or is invalid", asset.Name)
		}
		preparedName, prepared := asset.Name, data
		var inventory PhysicalGLBInventory
		if asset.Role == "device_geometry" || asset.Role == "scene_geometry" {
			prefix := "device"
			if asset.Role == "scene_geometry" {
				prefix = "scene"
			}
			if prefix == "device" {
				candidate, err := decodeDevice(manifest.Device.CanonicalJSON)
				if err != nil {
					return fmt.Errorf("decode device mapping candidate: %w", err)
				}
				if _, err := ValidateDevicePhysicalGLB(data, candidate, manifest.Screen); err != nil {
					return fmt.Errorf("validate original device GLB against candidate: %w", err)
				}
			} else {
				candidate, err := decodeScene(manifest.Scene.CanonicalJSON)
				if err != nil {
					return fmt.Errorf("decode scene mapping candidate: %w", err)
				}
				if _, err := ValidateScenePhysicalGLB(data, candidate); err != nil {
					return fmt.Errorf("validate original scene GLB against candidate: %w", err)
				}
			}
			preparedName = prefix + "-normalized.glb"
			prepared, inventory, err = NormalizePhysicalGLBNames(data, prefix)
			if err != nil {
				return fmt.Errorf("normalize pinned %s GLB: %w", prefix, err)
			}
			invBytes, _ := json.Marshal(inventory)
			if receipt.InventorySHA256 != digest(invBytes) {
				return fmt.Errorf("physical receipt %s GLB inventory digest differs", prefix)
			}
		} else if receipt.InventorySHA256 != "" || len(receipt.Parts) != 0 || len(receipt.Primitives) != 0 {
			return fmt.Errorf("physical texture asset %q unexpectedly contains geometry mappings", asset.Name)
		}
		if receipt.PreparedName != preparedName || receipt.PreparedSHA256 != sha256Bytes(prepared) || receipt.PreparedSizeBytes != int64(len(prepared)) {
			return fmt.Errorf("physical receipt asset %q prepared identity differs", asset.Name)
		}
		if preparedName != asset.Name {
			stored, err := readBoundedRegular(ctx, root, preparedName, 128<<20, int64(len(prepared)))
			if err != nil || !bytes.Equal(stored, prepared) {
				return fmt.Errorf("derived physical GLB %q differs from deterministic alias normalization", preparedName)
			}
		}
		if err := verifyPhysicalMappings(asset.Role, manifest, receipt, inventory); err != nil {
			return err
		}
	}
	return nil
}

func verifyPhysicalMappings(role string, manifest PhysicalRenderManifest, receipt PhysicalInputReceipt, inventory PhysicalGLBInventory) error {
	if role != "device_geometry" && role != "scene_geometry" {
		return nil
	}
	var expected []PhysicalPartMapping
	if role == "device_geometry" {
		candidate, err := decodeDevice(manifest.Device.CanonicalJSON)
		if err != nil {
			return err
		}
		expected = physicalPartMappings("device", candidate.Parts, inventory)
	} else {
		candidate, err := decodeScene(manifest.Scene.CanonicalJSON)
		if err != nil {
			return err
		}
		expected = physicalScenePartMappings(candidate, inventory)
	}
	prefix := "device"
	if role == "scene_geometry" {
		prefix = "scene"
	}
	if !jsonValuesEqual(expected, receipt.Parts) || !jsonValuesEqual(physicalPrimitiveMappings(prefix, inventory), receipt.Primitives) {
		return fmt.Errorf("physical receipt %s node/primitive mapping differs from canonical candidate and GLB", role)
	}
	return nil
}

func verifyPhysicalScreenReceipt(root *os.Root, manifest PhysicalRenderManifest, receipt PhysicalScreenReceipt) error {
	s := manifest.Screen
	if s.Recording != nil {
		b := s.Recording.Inspection.Source
		if receipt.DisplayID != s.DisplayID || receipt.OriginalMediaID != s.OriginalMediaID || receipt.OriginalMediaSHA256 != s.OriginalMediaSHA256 ||
			receipt.Name != b.Name || receipt.SHA256 != b.SHA256 || receipt.SizeBytes != b.SizeBytes || receipt.Width != s.Width || receipt.Height != s.Height ||
			receipt.PresentationProfile != s.PresentationProfile || receipt.PresentationVersion != s.PresentationVersion || receipt.ConfigSHA256 != s.ConfigSHA256 || receipt.LayoutSHA256 != s.LayoutSHA256 {
			return errors.New("physical recording screen receipt differs from the immutable source and presentation bindings")
		}
		file, err := root.Open(b.Name)
		if err != nil {
			return err
		}
		verifyErr := verifyInspectionSourceUnchanged(context.Background(), file, InspectionSource{Name: b.Name, SHA256: b.SHA256, SizeBytes: b.SizeBytes})
		closeErr := file.Close()
		if verifyErr != nil || closeErr != nil {
			return errors.New("physical recording source differs from the accepted inspection bytes")
		}
		deviceBytes, err := readPhysicalAsset(root, manifest, "device_geometry")
		if err != nil {
			return err
		}
		device, err := decodeDevice(manifest.Device.CanonicalJSON)
		if err != nil {
			return err
		}
		mapping, err := ResolvePhysicalScreenGeometry(deviceBytes, device, s.DisplayID)
		if err != nil || !jsonValuesEqual(mapping, receipt.Mapping) || mapping.PrimitiveIndex != s.PrimitiveIndex || mapping.MaterialIndex != s.MaterialSlot {
			return errors.New("physical recording screen mapping differs from its unique supported source primitive")
		}
		return nil
	}
	if receipt.DisplayID != s.DisplayID || receipt.OriginalMediaID != s.OriginalMediaID || receipt.OriginalMediaSHA256 != s.OriginalMediaSHA256 ||
		receipt.Name != s.Raster.Name || receipt.SHA256 != s.Raster.SHA256 || receipt.SizeBytes != s.Raster.SizeBytes || receipt.Width != s.Width || receipt.Height != s.Height ||
		receipt.PresentationProfile != s.PresentationProfile || receipt.PresentationVersion != s.PresentationVersion || receipt.ConfigSHA256 != s.ConfigSHA256 || receipt.LayoutSHA256 != s.LayoutSHA256 {
		return errors.New("physical screen receipt differs from the immutable screen binding")
	}
	data, err := readBoundedRegular(context.Background(), root, s.Raster.Name, physicalScreenBytesMax, s.Raster.SizeBytes)
	if err != nil || sha256Bytes(data) != s.Raster.SHA256 {
		return errors.New("physical screen raster changed or is invalid")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || format != "png" || config.Width != s.Width || config.Height != s.Height {
		return errors.New("physical screen raster is not the pinned PNG dimensions")
	}
	deviceBytes, err := readPhysicalAsset(root, manifest, "device_geometry")
	if err != nil {
		return err
	}
	device, err := decodeDevice(manifest.Device.CanonicalJSON)
	if err != nil {
		return err
	}
	mapping, err := ResolvePhysicalScreenGeometry(deviceBytes, device, s.DisplayID)
	if err != nil || !jsonValuesEqual(mapping, receipt.Mapping) || mapping.PrimitiveIndex != s.PrimitiveIndex || mapping.MaterialIndex != s.MaterialSlot {
		return errors.New("physical screen receipt mapping differs from the unique supported source primitive")
	}
	return nil
}

func verifyPhysicalPNGOutput(root *os.Root, manifest PhysicalRenderManifest, receipt PhysicalOutputReceipt) error {
	plan := manifest.Output
	if receipt.Kind != plan.Kind || receipt.Name != plan.OutputName || receipt.Width != plan.Width || receipt.Height != plan.Height || receipt.PixelFormat != plan.PixelFormat || receipt.SizeBytes <= 0 || receipt.SizeBytes > physicalOutputBytesMax || !physicalDigestPattern.MatchString(receipt.SHA256) {
		return errors.New("physical output receipt facts do not match the output plan")
	}
	data, err := readBoundedRegular(context.Background(), root, plan.OutputName, physicalOutputBytesMax, receipt.SizeBytes)
	if err != nil || sha256Bytes(data) != receipt.SHA256 {
		return errors.New("physical PNG output bytes differ from the receipt")
	}
	if plan.PixelFormat == "rgba8" && !isPhysicalRGBA8PNG(data) {
		return errors.New("physical PNG IHDR is not 8-bit RGBA")
	}
	decoded, format, err := image.Decode(bytes.NewReader(data))
	if err != nil || format != "png" || decoded.Bounds().Dx() != int(plan.Width) || decoded.Bounds().Dy() != int(plan.Height) {
		return errors.New("physical PNG output failed full decode or has the wrong raster")
	}
	return nil
}

func verifyPhysicalVideoOutput(root *os.Root, manifest PhysicalRenderManifest, receipt PhysicalRenderReceipt) error {
	plan, output := manifest.Output, receipt.Output
	if output.Kind != "motion_mp4" || output.Name != plan.OutputName || output.Width != plan.Width || output.Height != plan.Height || output.PixelFormat != plan.PixelFormat || output.FrameCount != plan.FrameCount || output.RateNumerator != uint32(plan.Rate.Num) || output.RateDenominator != uint32(plan.Rate.Den) || output.SizeBytes <= 0 || output.SizeBytes > videoMaxOutputBytes || !physicalDigestPattern.MatchString(output.SHA256) || receipt.VideoToolArtifactID != plan.VideoToolArtifactID || !physicalDigestPattern.MatchString(receipt.VideoToolProfileSHA256) {
		return errors.New("physical motion receipt facts do not match its immutable plan")
	}
	if err := validatePhysicalFrameRGBDigests(plan, output.FrameRGBDigests); err != nil {
		return err
	}
	if manifest.Screen.Recording != nil {
		if err := verifyPhysicalRecordingFrameFacts(root, manifest, output.SourceFrameFacts); err != nil {
			return err
		}
	} else if len(output.SourceFrameFacts) != 0 {
		return errors.New("static physical motion receipt unexpectedly includes recording source facts")
	}
	profileBytes, err := readBoundedRegular(context.Background(), root, physicalVideoProfileName, inspectionMaxProfile, 0)
	if err != nil || digest(profileBytes) != receipt.VideoToolProfileSHA256 {
		return errors.New("physical motion receipt does not bind its FFmpeg profile sidecar")
	}
	var profile cliv1.RendererToolProfile
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(profileBytes, &profile); err != nil {
		return errors.New("physical motion FFmpeg profile sidecar is malformed")
	}
	pair, err := LocalToolPairFromProfile(&profile)
	if err != nil || pair.ProfileID != plan.VideoToolProfileID || pair.ArtifactIdentity != plan.VideoToolArtifactID || profileArtifactIdentity(&profile) != plan.VideoToolArtifactID {
		return errors.New("physical motion receipt FFmpeg profile differs from its pinned pair")
	}
	var facts VideoNativeFacts
	decoder := json.NewDecoder(bytes.NewReader(output.NativeFacts))
	decoder.DisallowUnknownFields()
	if len(output.NativeFacts) == 0 || decoder.Decode(&facts) != nil || decoder.Decode(new(any)) != io.EOF || !facts.FullDecodeVerified || facts.Container != "mov,mp4,m4a,3gp,3g2,mj2" || facts.Codec != "h264" || facts.PixelFormat != "yuv420p" || facts.Width != int(plan.Width) || facts.Height != int(plan.Height) || facts.FrameCount != plan.FrameCount || facts.FrameRate != plan.Rate || facts.FirstPTS != 0 || facts.ColorPrimaries != "bt709" || facts.ColorTransfer != "bt709" || facts.ColorMatrix != "bt709" || facts.ColorRange != "tv" && facts.ColorRange != "mpeg" {
		return errors.New("physical motion native facts are incomplete or outside the pinned profile")
	}
	wantPTS, err := plannedOutputPTS(VideoManifest{Width: int(plan.Width), Height: int(plan.Height), OutputRate: plan.Rate}, facts.TimeBase, int(plan.FrameCount))
	if err != nil || len(wantPTS) != int(plan.FrameCount) || facts.LastPTS != wantPTS[len(wantPTS)-1] {
		return errors.New("physical motion native frame timestamps do not match the output clock")
	}
	f, err := root.Open(plan.OutputName)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() != output.SizeBytes || info.Size() > videoMaxOutputBytes {
		return errors.New("physical MP4 output size differs from receipt or exceeds its bound")
	}
	var prefix [12]byte
	if _, err := io.ReadFull(f, prefix[:]); err != nil || string(prefix[4:8]) != "ftyp" {
		return errors.New("physical motion output lacks an MP4 ftyp header")
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, videoMaxOutputBytes+1))
	if err != nil || n != output.SizeBytes || hex.EncodeToString(h.Sum(nil)) != output.SHA256 {
		return errors.New("physical MP4 bytes differ from the verified receipt hash")
	}
	return nil
}

func verifyPhysicalRecordingFrameFacts(root *os.Root, manifest PhysicalRenderManifest, facts []PhysicalRecordingFrameFact) error {
	recording := manifest.Screen.Recording
	if recording == nil || len(facts) != int(manifest.Output.FrameCount) {
		return errors.New("recording frame facts do not cover the output schedule")
	}
	b := recording.Inspection
	manifestBytes, err := readBoundedRegular(context.Background(), root, b.Manifest.Name, videoMaxManifestBytes, b.Manifest.SizeBytes)
	if err != nil || digest(manifestBytes) != b.Manifest.SHA256 {
		return errors.New("accepted inspection manifest changed")
	}
	receiptBytes, err := readBoundedRegular(context.Background(), root, b.Receipt.Name, videoMaxReceiptBytes, b.Receipt.SizeBytes)
	if err != nil || digest(receiptBytes) != b.Receipt.SHA256 {
		return errors.New("accepted inspection receipt changed")
	}
	clockBytes, err := readBoundedRegular(context.Background(), root, b.Clock.Name, videoMaxClockBytes, b.Clock.SizeBytes)
	if err != nil || digest(clockBytes) != b.Clock.SHA256 {
		return errors.New("accepted inspection clock changed")
	}
	accepted, err := VerifyInspectionArtifacts(manifestBytes, receiptBytes, clockBytes)
	if err != nil || matchVideoInspection(b, accepted) != nil {
		return errors.New("accepted source inspection closure no longer verifies")
	}
	pts, err := decodeInspectionClock(clockBytes, b.FrameCount)
	if err != nil {
		return err
	}
	plan, err := composition.PlanFrames(context.Background(), composition.FrameClock{SourcePTS: pts, SourceTimeBase: b.TimeBase, SourceEndPTS: b.SourceEndPTS, TrimStartPTS: recording.TrimStartPTS, TrimEndPTS: recording.TrimEndPTS, PlaybackRate: recording.PlaybackRate, OutputRate: manifest.Output.Rate, Underflow: composition.RangeReject, EOF: composition.RangeReject})
	if err != nil || len(plan) != len(facts) {
		return errors.New("recording output schedule differs from the accepted source clock")
	}
	return validatePhysicalRecordingSchedule(recording, plan, facts)
}

func validatePhysicalFrameRGBDigests(plan PhysicalOutputPlan, digests []PhysicalFrameRGBDigest) error {
	if plan.Kind != "motion_mp4" || len(digests) != int(plan.FrameCount) || len(plan.Samples) != int(plan.FrameCount) {
		return errors.New("physical motion receipt frame RGB digests do not cover the exact planned schedule")
	}
	for i, frame := range digests {
		if frame.OutputOrdinal != plan.Samples[i].OutputOrdinal || frame.OutputOrdinal != uint32(i) || !physicalDigestPattern.MatchString(frame.RGBSHA256) {
			return fmt.Errorf("physical motion receipt frame RGB digest %d is invalid or out of schedule order", i)
		}
	}
	return nil
}

func isPhysicalRGBA8PNG(data []byte) bool {
	return len(data) >= 29 && bytes.Equal(data[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10}) &&
		bytes.Equal(data[8:12], []byte{0, 0, 0, 13}) && bytes.Equal(data[12:16], []byte("IHDR")) &&
		data[24] == 8 && data[25] == 6 && data[26] == 0 && data[27] == 0 && data[28] == 0
}

// isPhysicalScreenTexturePNG accepts the two 8-bit PNG encodings Go's encoder
// may choose for a composed screen. Blender consumes texture Color, so RGB
// and RGBA have the same pinned sampling contract; rendered outputs stay RGBA8.
func isPhysicalScreenTexturePNG(data []byte) bool {
	return len(data) >= 29 && bytes.Equal(data[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10}) &&
		bytes.Equal(data[8:12], []byte{0, 0, 0, 13}) && bytes.Equal(data[12:16], []byte("IHDR")) &&
		data[24] == 8 && (data[25] == 2 || data[25] == 6) && data[26] == 0 && data[27] == 0 && data[28] == 0
}

func validatePhysicalOutputKindFacts(kind string, output PhysicalOutputReceipt) error {
	if kind == "still_png" && (len(output.FrameRGBDigests) != 0 || len(output.SourceFrameFacts) != 0) {
		return errors.New("still output unexpectedly includes motion frame facts")
	}
	return nil
}

func readPhysicalAsset(root *os.Root, manifest PhysicalRenderManifest, role string) ([]byte, error) {
	for _, asset := range manifest.Assets {
		if asset.Role == role {
			limit := int64(physicalInputBytesMax)
			if role == "device_geometry" || role == "scene_geometry" {
				limit = 128 << 20
			}
			return readBoundedRegular(context.Background(), root, asset.Name, limit, asset.Reference.SizeBytes)
		}
	}
	return nil, fmt.Errorf("physical manifest has no %s asset", role)
}

func physicalPartMappings(prefix string, parts []devices.SemanticPart, inventory PhysicalGLBInventory) []PhysicalPartMapping {
	byIndex := make(map[uint32]PhysicalGLBNode, len(inventory.Nodes))
	for _, node := range inventory.Nodes {
		byIndex[node.Index] = node
	}
	result := make([]PhysicalPartMapping, 0, len(parts))
	for _, part := range parts {
		node := byIndex[part.NodeIndex]
		result = append(result, PhysicalPartMapping{PartID: part.ID, Role: part.Role, NodeIndex: node.Index, NodeName: node.Name, ObjectName: physicalNodeAlias(prefix, node.Index), ParentIndex: node.Parent, MeshIndex: node.Mesh, LocalFrame: trsReceipt(node.Frame)})
	}
	return result
}

func physicalScenePartMappings(candidate scenes.SceneCandidate, inventory PhysicalGLBInventory) []PhysicalPartMapping {
	parts := make([]devices.SemanticPart, 0, len(candidate.Parts))
	for _, part := range candidate.Parts {
		parts = append(parts, devices.SemanticPart{ID: part.ID, Role: part.Role, NodeIndex: part.NodeIndex, NodeName: part.NodeName, LocalFrame: part.LocalFrame})
	}
	return physicalPartMappings("scene", parts, inventory)
}

func physicalPrimitiveMappings(prefix string, inventory PhysicalGLBInventory) []PhysicalPrimitiveMapping {
	result := make([]PhysicalPrimitiveMapping, 0)
	for _, mesh := range inventory.Meshes {
		for _, primitive := range mesh.Primitives {
			result = append(result, PhysicalPrimitiveMapping{
				MeshIndex: mesh.Index, PrimitiveIndex: primitive.Index, MaterialIndex: primitive.Material,
				MaterialName: physicalMaterialAlias(prefix, primitive.Material), Mode: primitive.Mode, UV0: primitive.UV0 != nil,
				UVMin: primitive.UVMin, UVMax: primitive.UVMax, VertexCount: primitive.VertexCount, TriangleCount: primitive.TriangleCount,
			})
		}
	}
	return result
}

func physicalNodeAlias(prefix string, index uint32) string {
	return fmt.Sprintf("DS_%s_NODE_%05d", prefix, index)
}

func physicalMaterialAlias(prefix string, index uint32) string {
	return fmt.Sprintf("DS_%s_MATERIAL_%05d", prefix, index)
}

func trsReceipt(value physical.TRS) physicalTRS {
	return physicalTRS{Translation: [3]float64{value.Translation.X, value.Translation.Y, value.Translation.Z}, Rotation: [4]float64{value.Rotation.X, value.Rotation.Y, value.Rotation.Z, value.Rotation.W}, Scale: [3]float64{value.Scale.X, value.Scale.Y, value.Scale.Z}}
}

func jsonValuesEqual(a, b any) bool {
	left, errLeft := json.Marshal(a)
	right, errRight := json.Marshal(b)
	return errLeft == nil && errRight == nil && bytes.Equal(left, right)
}

func ensurePhysicalOutputAbsent(root *os.Root, name string) error {
	_, err := root.Lstat(name)
	if err == nil {
		return fmt.Errorf("physical worker output %q already exists", filepath.Base(name))
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect reserved physical output: %w", err)
	}
	return nil
}
