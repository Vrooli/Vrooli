package physicalruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"

	"device-studio/internal/devices"
	"device-studio/internal/library"
	"device-studio/internal/motions"
	"device-studio/internal/physical"
	"device-studio/internal/renderjobs"
	"device-studio/internal/scenes"
	"device-studio/internal/shots"
)

const (
	reviewedSmartDisplayV7StarterID = "generic-smart-display-concept-v7-reviewed"
	smartDisplayV7ReviewCertificate = "assets/reviewed-starters/smart-display-v7/qualification-certificate-v1.json"
	smartDisplayV7ReviewOutput      = "assets/reviewed-starters/smart-display-v7/native-output.png"
	smartDisplayV7CertificateSHA256 = "910e5132d6b5c233593670a7e81da673eba6dc400116d5e4fcbd69e787b99760"
	smartDisplayV7PackageSHA256     = "7139e503c471e724789f61733712ff3f016ed33e841605f197574b338c3e1bd2"
)

type reviewedCertificateArtifact struct {
	Ref physical.ArtifactRef `json:"ref"`
}

// This compact certificate retains the exact owner records from the accepted
// package but loads source payloads from their separately pinned embedded bytes.
type reviewedStarterCertificate struct {
	Version         int                                    `json:"version"`
	PackageSHA      string                                 `json:"package_sha256"`
	Origin          qualificationPackageOrigin             `json:"origin"`
	Candidate       devices.DeviceRevision                 `json:"candidate"`
	SourceBundles   []library.BundleRevision               `json:"source_bundle_revisions"`
	SourceArtifacts []reviewedCertificateArtifact          `json:"source_artifacts"`
	Scene           scenes.Revision                        `json:"scene_revision"`
	Motion          motions.Revision                       `json:"motion_revision"`
	Preset          shots.PhysicalPresetRevision           `json:"preset_revision"`
	Review          devices.QualificationReview            `json:"review"`
	Render          renderjobs.QualificationRenderEvidence `json:"render_receipt"`
}

type embeddedReviewedStarter struct {
	ID             string
	BootstrapID    string
	DeviceID       string
	Certificate    string
	CertificateSHA string
	PackageSHA     string
}

var embeddedReviewedStarters = map[string]embeddedReviewedStarter{
	"generic-round-watch-body-v4-floating-reviewed": {
		ID: "generic-round-watch-body-v4-floating-reviewed", BootstrapID: "generic-round-watch-body-v4-floating",
		DeviceID: "generic-round-watch-body-candidate-v4", Certificate: "assets/reviewed-starters/watch-body-v4/qualification-certificate-v1.json",
		CertificateSHA: "908fb39458d946143ce23ebe69fb79a5a208b60a41838402b42e6b292ed6cfb6", PackageSHA: "86835937b293de8cfc1a77e2326c218203872ffe2aa59d401557527d723dc09d",
	},
	"generic-television-concept-v3-reviewed": {
		ID: "generic-television-concept-v3-reviewed", BootstrapID: "generic-television-concept-v3",
		DeviceID: "generic-television-candidate-v3", Certificate: "assets/reviewed-starters/tv-v3/qualification-certificate-v1.json",
		CertificateSHA: "ef5cefd6c2892bbf839b2525699cb5f2262ab55bf3b6ed4e9ed60c817ca8a712", PackageSHA: "f3394a8ae1422ddc7f3cf4312ab4f6fd9104f8ee6e6ea514c27b70f6d22df4a6",
	},
	"generic-digital-frame-concept-v4-glassless-reviewed": {
		ID: "generic-digital-frame-concept-v4-glassless-reviewed", BootstrapID: "generic-digital-frame-concept-v4-glassless",
		DeviceID: "innerscene-digital-frame-tabletop-candidate-v4-glassless", Certificate: "assets/reviewed-starters/frame-desk-v4/qualification-certificate-v1.json",
		CertificateSHA: "14ff7b720c7bbc8d3df3e81edf54759cf5c02a939acf3d3269604feb6d43c7cf", PackageSHA: "180e435f2802374c7ae450542880fc3ad8a3734cf3ba2ab609cafdf504067e01",
	},
	"generic-digital-frame-wall-v5-mounted-reviewed": {
		ID: "generic-digital-frame-wall-v5-mounted-reviewed", BootstrapID: "generic-digital-frame-wall-v5-mounted",
		DeviceID: "innerscene-digital-frame-wall-candidate-v5-mounted", Certificate: "assets/reviewed-starters/frame-wall-v5/qualification-certificate-v1.json",
		CertificateSHA: "e0aae615e2c951ab58df113db5e978d900001ebd0c69d0cf1f9a30d7f13cd4a0", PackageSHA: "a7ea0516d175b10300b3ef40001c1e04dc9bdec4688121cb65e27fa81ff7c316",
	},
	"generic-book-fold-v3-left-screen-vflip-static-reviewed": {
		ID: "generic-book-fold-v3-left-screen-vflip-static-reviewed", BootstrapID: "generic-book-fold-v3-left-screen-vflip-static",
		DeviceID: "generic-dual-rigid-book-fold-concept-v3-left-screen-vflip", Certificate: "assets/reviewed-starters/fold-v3-static/qualification-certificate-v1.json",
		CertificateSHA: "9195730179385db4fa6b05d838edd81f366a2c973350762d1b3bb69f4250f524", PackageSHA: "3a1f0669aace859cbcd92d96070f356fbc6fcd46dffeb48e3a3538a139b4acaa",
	},
	"laptop-screen-vflip-reviewed": {
		ID: "laptop-screen-vflip-reviewed", BootstrapID: "laptop-screen-vflip",
		DeviceID: "jack-baeten-macbook-pro-m3-16-screen-vflip-candidate-v1", Certificate: "assets/reviewed-starters/laptop/qualification-certificate-v1.json",
		CertificateSHA: "409dad0400a923a365a3a1e2b1b12da5580765b862cb7491373de034f4dc882a", PackageSHA: "f84ab0ee7d2988651aa742a6586963ddf769a84dd06944084559906f8c929c89",
	},
}

// installReviewedSmartDisplayV7 remains an explicit compatibility route for
// its factory initializer, which verifies imported qualified children itself.
func (o Owners) installReviewedSmartDisplayV7(ctx context.Context) (shots.PhysicalPresetRevision, error) {
	if o.Qualification == nil || o.Starter == nil || o.Presets == nil {
		return shots.PhysicalPresetRevision{}, errors.New("reviewed starter installer is unavailable")
	}
	state, err := o.InitializeReviewedSmartDisplayV7Candidate(ctx)
	if err != nil {
		return shots.PhysicalPresetRevision{}, err
	}
	packageBytes, err := loadEmbeddedSmartDisplayV7Review(o.Starter.fs)
	if err != nil {
		return shots.PhysicalPresetRevision{}, err
	}
	result, err := o.Qualification.ImportReviewPackage(ctx, packageBytes, state.Device.Revision)
	if err != nil {
		return shots.PhysicalPresetRevision{}, fmt.Errorf("apply embedded smart-display visual-setup review: %w", err)
	}
	return o.Presets.Get(ctx, result.PresetID, result.PresetRevision)
}

func (o Owners) installEmbeddedReviewedStarter(ctx context.Context, starterID string) (shots.PhysicalPresetRevision, error) {
	descriptor, ok := embeddedReviewedStarters[starterID]
	if !ok {
		return shots.PhysicalPresetRevision{}, errors.New("reviewed starter is not in the embedded allowlist")
	}
	if o.Qualification == nil || o.Starter == nil || o.Devices == nil || o.Presets == nil {
		return shots.PhysicalPresetRevision{}, errors.New("reviewed starter installer is unavailable")
	}
	packageBytes, certificate, err := loadEmbeddedReviewedStarter(o.Starter.fs, descriptor)
	if err != nil {
		return shots.PhysicalPresetRevision{}, err
	}
	if certificate.Candidate.DeviceID != descriptor.DeviceID || certificate.Candidate.Revision != 1 {
		return shots.PhysicalPresetRevision{}, errors.New("reviewed starter certificate does not match its pinned factory candidate")
	}
	candidate, err := o.Devices.GetRevision(ctx, descriptor.DeviceID, certificate.Candidate.Revision)
	if err != nil {
		if !errors.Is(err, devices.ErrRevisionNotFound) {
			return shots.PhysicalPresetRevision{}, fmt.Errorf("read exact reviewed starter factory candidate: %w", err)
		}
		if _, initErr := o.InitializePhysicalStarter(ctx, descriptor.BootstrapID); initErr != nil {
			return shots.PhysicalPresetRevision{}, fmt.Errorf("initialize reviewed starter factory inputs: %w", initErr)
		}
		candidate, err = o.Devices.GetRevision(ctx, descriptor.DeviceID, certificate.Candidate.Revision)
	}
	if err != nil || candidate.SHA256 != certificate.Candidate.SHA256 || !sameJSON(candidate.Candidate, certificate.Candidate.Candidate) {
		return shots.PhysicalPresetRevision{}, errors.New("exact reviewed starter factory candidate is missing or changed")
	}
	head, err := o.Devices.GetHead(ctx, descriptor.DeviceID)
	if err != nil || head.Revision < certificate.Candidate.Revision {
		return shots.PhysicalPresetRevision{}, errors.New("reviewed starter candidate head is unavailable")
	}
	result, err := o.Qualification.ImportReviewPackage(ctx, packageBytes, head.Revision)
	if err != nil {
		return shots.PhysicalPresetRevision{}, fmt.Errorf("apply retained scoped native review: %w", err)
	}
	return o.Presets.Get(ctx, result.PresetID, result.PresetRevision)
}

func loadEmbeddedSmartDisplayV7Review(assets fs.FS) ([]byte, error) {
	descriptor := embeddedReviewedStarter{
		Certificate: smartDisplayV7ReviewCertificate, CertificateSHA: smartDisplayV7CertificateSHA256,
		PackageSHA: smartDisplayV7PackageSHA256,
	}
	data, _, err := loadEmbeddedReviewedStarter(assets, descriptor)
	return data, err
}

func loadEmbeddedReviewedStarter(assets fs.FS, descriptor embeddedReviewedStarter) ([]byte, reviewedStarterCertificate, error) {
	if assets == nil {
		return nil, reviewedStarterCertificate{}, errors.New("reviewed starter assets are unavailable")
	}
	certificateBytes, err := fs.ReadFile(assets, descriptor.Certificate)
	if err != nil || len(certificateBytes) == 0 || len(certificateBytes) > 1<<20 {
		return nil, reviewedStarterCertificate{}, errors.New("review certificate is missing or exceeds its limit")
	}
	if digestHex(certificateBytes) != descriptor.CertificateSHA {
		return nil, reviewedStarterCertificate{}, errors.New("review certificate differs from its pinned application bytes")
	}
	decoder := json.NewDecoder(bytes.NewReader(certificateBytes))
	decoder.DisallowUnknownFields()
	var certificate reviewedStarterCertificate
	if err := decoder.Decode(&certificate); err != nil {
		return nil, reviewedStarterCertificate{}, fmt.Errorf("decode review certificate: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, reviewedStarterCertificate{}, errors.New("review certificate has trailing JSON")
	}
	if certificate.Version != 2 || certificate.PackageSHA != descriptor.PackageSHA || len(certificate.SourceArtifacts) != 6 {
		return nil, reviewedStarterCertificate{}, errors.New("review certificate has unsupported or incomplete scope")
	}
	outputPath := descriptor.Certificate[:len(descriptor.Certificate)-len("qualification-certificate-v1.json")] + "native-output.png"
	output, err := fs.ReadFile(assets, outputPath)
	if err != nil || len(output) == 0 || digestHex(output) != certificate.Render.OutputSHA256 {
		return nil, reviewedStarterCertificate{}, errors.New("embedded review output is missing or differs from its receipt")
	}
	artifacts := make([]qualificationPackageArtifact, 0, len(certificate.SourceArtifacts))
	seen := make(map[string]bool, len(certificate.SourceArtifacts))
	for _, item := range certificate.SourceArtifacts {
		key := qualificationArtifactKey(item.Ref)
		if seen[key] {
			return nil, reviewedStarterCertificate{}, errors.New("review certificate contains a duplicate source artifact")
		}
		seen[key] = true
		path, ok := reviewedStarterSourcePath(item.Ref)
		if !ok {
			return nil, reviewedStarterCertificate{}, errors.New("review certificate references an artifact outside its explicit source allowlist")
		}
		data, readErr := fs.ReadFile(assets, path)
		if readErr != nil || int64(len(data)) != item.Ref.SizeBytes || digestHex(data) != item.Ref.SHA256 {
			return nil, reviewedStarterCertificate{}, fmt.Errorf("embedded reviewed source %s is missing or changed", item.Ref.ArtifactID)
		}
		artifacts = append(artifacts, qualificationPackageArtifact{Ref: item.Ref, Bytes: data})
	}
	pkg := qualificationPackage{
		Version: certificate.Version, PackageSHA: certificate.PackageSHA, Origin: certificate.Origin,
		Candidate: certificate.Candidate, SourceBundles: certificate.SourceBundles, SourceArtifacts: artifacts,
		Scene: certificate.Scene, Motion: certificate.Motion, Preset: certificate.Preset,
		Review: certificate.Review, Render: certificate.Render, Output: output,
	}
	sealed, err := sealQualificationPackage(pkg)
	if err != nil {
		return nil, reviewedStarterCertificate{}, err
	}
	opened, err := openQualificationPackage(sealed)
	if err != nil || opened.PackageSHA != certificate.PackageSHA {
		return nil, reviewedStarterCertificate{}, errors.New("reconstructed package differs from its retained package digest")
	}
	return sealed, certificate, nil
}

func reviewedStarterSourcePath(ref physical.ArtifactRef) (string, bool) {
	if ref.BundleRevision != 1 {
		return "", false
	}
	if ref.BundleID == "device-studio-laptop-derive16-candidate-v1" && ref.ArtifactID == "laptop-study-room-no-cup-glb" {
		return "assets/laptop-study-environment/laptop-study-environment-no-cup.glb", true
	}
	if ref.BundleID == "device-studio-starter-phone-study-environment" {
		files := map[string]string{
			"travertine-color": "assets/phone-table/textures/travertine-color.jpg", "travertine-height": "assets/phone-table/textures/travertine-height.jpg",
			"travertine-normal-gl": "assets/phone-table/textures/travertine-normal-gl.jpg", "travertine-roughness": "assets/phone-table/textures/travertine-roughness.jpg",
		}
		path, ok := files[ref.ArtifactID]
		return path, ok
	}
	deviceFiles := map[string]map[string]string{
		"device-studio-generic-smart-display-concept-v7":                       {"vrooli-generic-smart-display-concept-v7": "assets/smart-display-v7/generic-smart-display-v7.glb"},
		"device-studio-generic-round-watch-body-v4":                            {"vrooli-generic-round-watch-body-concept-v4": "assets/roundwatch-body-v4/generic-roundwatch-body-v4.glb"},
		"device-studio-generic-television-thin-bezel-v3":                       {"vrooli-generic-television-thin-bezel-v3": "assets/television-v3/television-thin-bezel-v3.glb"},
		"device-studio-innerscene-frame-digital-neutral-tabletop-v4-glassless": {"innerscene-frame-digital-neutral-tabletop-v4-glassless": "assets/digital-frame/innerscene-frame-tabletop-v4-glassless.glb"},
		"device-studio-innerscene-frame-wall-v5-mounted":                       {"innerscene-frame-digital-neutral-wall-v5-mounted": "assets/digital-frame/innerscene-frame-wall-v5-mounted.glb"},
		"device-studio-starter-generic-book-fold-concept-v3-left-screen-vflip": {"vrooli-generic-book-fold-concept-v3-left-screen-vflip": "assets/foldable-v3/foldable-concept-v3.glb"},
		"device-studio-laptop-screen-vflip-candidate-v1":                       {"jack-baeten-laptop-neutral-screen-vflip-glb": "assets/laptop/laptop-neutral-rig-screen-vflip-v1.glb"},
	}
	path, ok := deviceFiles[ref.BundleID][ref.ArtifactID]
	return path, ok
}

func reviewedSmartDisplayV7SourcePath(ref physical.ArtifactRef) (string, bool) {
	return reviewedStarterSourcePath(ref)
}
