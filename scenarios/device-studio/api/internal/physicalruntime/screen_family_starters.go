package physicalruntime

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"device-studio/internal/devices"
	"device-studio/internal/library"
	"device-studio/internal/motions"
	"device-studio/internal/physical"
	"device-studio/internal/rendering"
	"device-studio/internal/scenes"
	"device-studio/internal/shots"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	"google.golang.org/protobuf/proto"
)

const (
	smartDisplayAssetFile           = "assets/smart-display/generic-wedge-smart-display.glb"
	smartDisplayAssetSHA            = "1b1364a137fc99545080dca69104c4f4d2a6b8821ad4ebc709981da4df89830f"
	smartDisplayScreenV2SHA         = "0cd217c12146c40672dd9b0f1880e6a8b26df141ae113fd460c4b03914f35bff"
	smartDisplayManifest            = "scenarios/device-studio/api/internal/physicalruntime/assets/smart-display/asset-manifest.json"
	smartDisplayManifestSHA         = "f7f89637e53ba8e2b900466b008d5e52b11185552a6fe537c169f6f71cee7380"
	smartDisplayV3AssetFile         = "assets/smart-display-v3/generic-smart-display-v3.glb"
	smartDisplayV3AssetSHA          = "9890e82e4ded55f098fb2eb99548bb3d0331c1a599193ee2041c9773a338cf50"
	smartDisplayV3Manifest          = "scenarios/device-studio/api/internal/physicalruntime/assets/smart-display-v3/asset-manifest-v3.json"
	smartDisplayV3ManifestSHA       = "ab862e6be2d30fea81dd166ddbd75e79b74f1ad3ff8e193d3ecb44d03f8787f8"
	smartDisplayV4AssetFile         = "assets/smart-display-v4/generic-smart-display-v4.glb"
	smartDisplayV4AssetSHA          = "7bf1f3b0c835f5817c592b12693d5b3d1deb6d289e06e727270ce245ebd75333"
	smartDisplayV4Manifest          = "scenarios/device-studio/api/internal/physicalruntime/assets/smart-display-v4/asset-manifest-v4.json"
	smartDisplayV4ManifestSHA       = "896e28ebe8c000949b0a0b6d3fa680e66d3ffdcc8ab2ca0dea122c56080cbb70"
	smartDisplayV5AssetFile         = "assets/smart-display-v5/generic-smart-display-v5.glb"
	smartDisplayV5AssetSHA          = "3ee98ae0dd5ce505f0de566055fffe8134fa1cccdad9c4ababab40a5a1ec84c2"
	smartDisplayV5Manifest          = "scenarios/device-studio/api/internal/physicalruntime/assets/smart-display-v5/asset-manifest-v5.json"
	smartDisplayV5ManifestSHA       = "47feb89d04bb739d0048cc612ed07e6237a47065bf54b4839aed8ce27e27d5fd"
	smartDisplayV6AssetFile         = "assets/smart-display-v6/generic-smart-display-v6.glb"
	smartDisplayV6AssetSHA          = "0b77cc12cf175c5c2d8640e0b81b3b1d5ac216233db3fc1fff4ba2d8b4f428e4"
	smartDisplayV6Manifest          = "scenarios/device-studio/api/internal/physicalruntime/assets/smart-display-v6/asset-manifest-v6.json"
	smartDisplayV6ManifestSHA       = "bbc52f7fbb8ad09ecd80a8187a63bef60cc73303c8414dce8a8feb64316fb709"
	smartDisplayV7AssetFile         = "assets/smart-display-v7/generic-smart-display-v7.glb"
	smartDisplayV7AssetSHA          = "e540c0dea92fe9217d2ac3833f3e89f0b77346e4ad22744d2e758b74566d4dd5"
	smartDisplayV7Manifest          = "scenarios/device-studio/api/internal/physicalruntime/assets/smart-display-v7/asset-manifest-v7.json"
	smartDisplayV7ManifestSHA       = "0488c6e4868bf617bf1cba717a3eaed120e4a9691b81d8ee278c3b6cc31a8422"
	digitalFrameManifest            = "scenarios/device-studio/api/internal/physicalruntime/assets/digital-frame/asset-manifest-v2.json"
	digitalFrameManifestSHA         = "0bd8c2af563201c7244cfbd17aadfb5150c297dade066f78c32666f98c27decd"
	digitalFrameWallManifest        = "scenarios/device-studio/api/internal/physicalruntime/assets/digital-frame/asset-manifest-v3.json"
	digitalFrameWallManifestSHA     = "d7458aa8c2160fc9f3d06f06187fe15ed1ebf4c9359b963bd6955751878f6c10"
	digitalFrameV4Manifest          = "scenarios/device-studio/api/internal/physicalruntime/assets/digital-frame/asset-manifest-v4.json"
	digitalFrameV4ManifestSHA       = "0f1f189720a99964691e196b00e573c0cd231e5f41953443a56d2d45b5bea6f8"
	digitalFrameWallV5AssetFile     = "assets/digital-frame/innerscene-frame-wall-v5-mounted.glb"
	digitalFrameWallV5AssetSHA      = "e9254d84025b8b3b3a371d10fa3bba970f31242790cbfce8f94f11110a72289b"
	digitalFrameWallV5Manifest      = "scenarios/device-studio/api/internal/physicalruntime/assets/digital-frame/asset-manifest-v5-wall-mounted.json"
	digitalFrameWallV5ManifestSHA   = "9ed0f676973e8cccf9fdfbf73b42cad85b8a33e42ba92b0ec5e945b21d3101d5"
	vehicleStackAssetFile           = "assets/vehicle-center-stack/generic-center-stack.glb"
	vehicleStackAssetSHA            = "37fc41e26e5c5f616f78fe68dfbd88cbbce6bfb0038754a65a20783684239ce9"
	vehicleStackScreenV2SHA         = "5cf6f233909d16d21e15e448519e736c17c7dcae154f9900196582d39f244796"
	vehicleStackManifest            = "scenarios/device-studio/api/internal/physicalruntime/assets/vehicle-center-stack/asset-manifest.json"
	vehicleStackManifestSHA         = "6f123070d68bd9021dbfb07ad01b34eb612a2ae1d4970847efb2bd6d278d84d5"
	vehicleStackV3AssetFile         = "assets/vehicle-center-stack/generic-dashboard-cc0-neutral-v1.glb"
	vehicleStackV3AssetSHA          = "4f7dee2ca8790baab3c8f115f21e582af710e0ed2f16d976058a56814b431b26"
	vehicleStackV3Manifest          = "scenarios/device-studio/api/internal/physicalruntime/assets/vehicle-center-stack/asset-manifest-v3-cc0.json"
	vehicleStackV3ManifestSHA       = "f76fe107ad9525518ce1b769093bb52bab654930c69517206914c31e75662bd5"
	vehicleStudioSceneFile          = "assets/vehicle-studio/vehicle-neutral-studio-wall-v1.glb"
	vehicleStudioSceneSHA           = "389531d4e51bcea93cea6156e8096831d2ac000acb1b9831ac56df6bef2ef7e3"
	vehicleStudioSceneManifest      = "scenarios/device-studio/api/internal/physicalruntime/assets/vehicle-studio/asset-manifest-v1.json"
	vehicleStudioSceneManifestSHA   = "84e32e04a07b28db2a39fc4ff98696d06f3b0a85d6ac152a08292c6d24bb4ce4"
	vehicleStudioSceneV2File        = "assets/vehicle-studio/vehicle-neutral-studio-wall-v2.glb"
	vehicleStudioSceneV2SHA         = "d7ff2208f1ad185a1da19fb78c9c07536f36f28514771bb69f02e9282a6c4de1"
	vehicleStudioSceneV2Manifest    = "scenarios/device-studio/api/internal/physicalruntime/assets/vehicle-studio/asset-manifest-v2.json"
	vehicleStudioSceneV2ManifestSHA = "309cb1e350967fbf8afab3ecefe02d928dfd4b9e1f94e04925c711392740361e"
	roundWatchV2AssetFile           = "assets/roundwatch-v2/generic-roundwatch-v2.glb"
	roundWatchV2AssetSHA            = "a609b435f5f4485504f1071e35cf74e95ba56095e5420aaa242255267c87750e"
	roundWatchV2Manifest            = "scenarios/device-studio/api/internal/physicalruntime/assets/roundwatch-v2/asset-manifest-v2.json"
	roundWatchV2ManifestSHA         = "0ee878061e3f495ecc194889deacaac2885b3d5a78b48c1883adb26541d9e880"
	roundWatchV3AssetFile           = "assets/roundwatch-v3/generic-roundwatch-v3.glb"
	roundWatchV3AssetSHA            = "faa7f39f305c77d77078080599cecdf2df4c2af3d28a06c2763c2707c1a7188c"
	roundWatchV3Manifest            = "scenarios/device-studio/api/internal/physicalruntime/assets/roundwatch-v3/asset-manifest-v3.json"
	roundWatchV3ManifestSHA         = "d4333c2a6ccca78e4defbb9da41c681de7eb6dca6934853c87e1fc9d07451128"
	roundWatchBodyV4AssetFile       = "assets/roundwatch-body-v4/generic-roundwatch-body-v4.glb"
	roundWatchBodyV4AssetSHA        = "d9631384327008b7a59fb78a3f5f29a42983582f7bbb3c4427fa2edc8767e03e"
	roundWatchBodyV4Manifest        = "scenarios/device-studio/api/internal/physicalruntime/assets/roundwatch-body-v4/asset-manifest-body-v4.json"
	roundWatchBodyV4ManifestSHA     = "1e24656b39af089b7017bf175b39af040f161c28ae00ababeb67ab84d5f919ef"
	televisionV3AssetFile           = "assets/television-v3/television-thin-bezel-v3.glb"
	televisionV3AssetSHA            = "bdf42b10916cffcc1bbdf540a91b80236db0fc1be3ff5658954b8bed132a16c3"
	televisionV3Manifest            = "scenarios/device-studio/api/internal/physicalruntime/assets/television-v3/asset-manifest-v3.json"
	televisionV3ManifestSHA         = "020edcc90afbfbc76cb5c6d5a836524ca08e93e701baa0823144af3d78048f4d"
	familyWallSceneID               = "screen-family-candidate-wall-study-v1"
	familyWallSupportID             = "study-wall-vertical-candidate"
	familyWallSupportPartID         = "laptop-scene-part-0"
	familyWallCameraID              = "screen-family-wall-camera-full-envelope-v1"
	familyWallV5SupportID           = "study-wall-vertical-v5-mounted-candidate"
	familyWallRearClearanceM        = 0.002
	familyWallSupportFrontOffsetM   = 0.0025
)

type ScreenFamilyStarterState struct {
	Bundle library.BundleRevision
	Device devices.DeviceRevision
	Scene  scenes.Revision
	Motion motions.Revision
	Preset shots.PhysicalPresetRevision
}

type screenFamilySpec struct {
	assetID, file, sha, bundleID, deviceID, sceneID, presetID, presetName, reviewID                         string
	manifestPath, manifestSHA, nodeName, displayID, rootName, groupKind, kind, model, variant, description  string
	license, attribution, sourceLabel, sourceSHA, adaptation, profile                                       string
	sceneFile, sceneSHA, sceneManifestPath, sceneManifestSHA, sceneBundleID, sceneReviewID, sceneArtifactID string
	wall, screenVFlip, frontalCamera                                                                        bool
	displayTopology, activeShape                                                                            string
	primitiveIndex                                                                                          uint32
}

func smartDisplaySpec() screenFamilySpec {
	return screenFamilySpec{
		assetID: "vrooli-generic-wedge-smart-display-concept-v2-screen-vflip", file: smartDisplayAssetFile, sha: smartDisplayScreenV2SHA,
		bundleID: "device-studio-generic-wedge-smart-display-v2-screen-vflip", deviceID: "generic-wedge-smart-display-candidate-v2-screen-vflip",
		sceneID: "generic-wedge-smart-display-study-candidate-v2-screen-vflip", presetID: "generic-wedge-smart-display-tabletop-still-v2-screen-vflip",
		presetName: "Generic wedge smart display tabletop review · screen UV v2", reviewID: "device-studio-e4-generic-wedge-smart-display-candidate-v2-screen-vflip",
		manifestPath: smartDisplayManifest, manifestSHA: smartDisplayManifestSHA, nodeName: "MainDisplay", displayID: "main_display",
		rootName: "Generic Wedge Smart Display", groupKind: string(devices.GroupSmartDisplayFrame), kind: "generic_wedge_smart_display_concept",
		model: "Generic authored wedge smart-display concept", variant: "180 x 125 x 90 mm authored design envelope",
		description: "Unqualified Vrooli-authored wedge display candidate for tabletop composition studies. Nest Hub dimensions are comparison context only; no branded identity, electronics, or product fit is claimed.",
		license:     "AGPL-3.0-only", attribution: "Vrooli-authored generic wedge smart-display concept; no manufacturer geometry.",
		sourceLabel: "Deterministic Vrooli GLB derivation script; screen-only vertical UV correction.", sourceSHA: smartDisplayAssetSHA,
		adaptation: "Private TEXCOORD_0 accessor for MainDisplay primitive 0 flips V only; the source BIN prefix and all positions, normals, indices, materials, and other accessors remain unchanged.", profile: "tabletop", screenVFlip: true,
	}
}

func smartDisplayV3Spec() screenFamilySpec {
	return screenFamilySpec{
		assetID: "vrooli-generic-smart-display-concept-v3", file: smartDisplayV3AssetFile, sha: smartDisplayV3AssetSHA,
		bundleID: "device-studio-generic-smart-display-concept-v3", deviceID: "generic-smart-display-candidate-v3",
		sceneID: "generic-smart-display-study-candidate-v3", presetID: "generic-smart-display-tabletop-still-v3",
		presetName: "Generic smart-display tabletop review · authored v3", reviewID: "device-studio-e4-generic-smart-display-candidate-v3",
		manifestPath: smartDisplayV3Manifest, manifestSHA: smartDisplayV3ManifestSHA,
		nodeName: "MainDisplayActivePlane", displayID: "main_display", rootName: "GenericSmartDisplayV3",
		groupKind: string(devices.GroupSmartDisplayFrame), kind: "generic_smart_display_concept",
		model: "Generic authored smart-display concept v3", variant: "177.4 x 120.4 x 69.5 mm conceptual envelope; reference-inspired, not measured",
		description: "Unqualified Vrooli-authored smart-display concept with a rounded, tilted screen, speaker base, and contact pads. The Nest Hub reference informs comparison only; this is not product CAD or a claim about any physical unit.",
		license:     "AGPL-3.0-only", attribution: "Vrooli-authored generic smart-display v3; official Google reference imagery is comparison-only and not included.",
		sourceLabel: "Deterministic offline Vrooli-authored smart-display v3 generator; no external model source.",
		profile:     "tabletop_concept_v3",
	}
}

func digitalFrameSpec(wall bool) screenFamilySpec {
	profile, file, digest, id, deviceID, sceneID, presetID, name := "tabletop", "assets/digital-frame/innerscene-frame-tabletop-v2.glb",
		"bc6b22fb722672b0601dc2ea4424517e6bed589e3ebf40de9ecd0257d20fb4f4", "innerscene-frame-digital-neutral-tabletop-v2",
		"innerscene-digital-frame-tabletop-candidate-v2", "innerscene-digital-frame-tabletop-study-v2", "innerscene-digital-frame-tabletop-still-v2", "Innerscene-derived neutral digital frame tabletop review"
	manifestPath, manifestSHA := digitalFrameManifest, digitalFrameManifestSHA
	if wall {
		profile, file, digest, id, deviceID, sceneID, presetID, name = "wall", "assets/digital-frame/innerscene-frame-wall-v3.glb",
			"86c216e9bc574ac076bc78231107cb53b4c178bdea4121239df43ca8963e17bf", "innerscene-frame-digital-neutral-wall-v3",
			"innerscene-digital-frame-wall-candidate-v3", "innerscene-digital-frame-wall-study-v3", "innerscene-digital-frame-wall-still-v3", "Innerscene-derived neutral digital frame wall review"
		manifestPath, manifestSHA = digitalFrameWallManifest, digitalFrameWallManifestSHA
	}
	return screenFamilySpec{
		assetID: id, file: file, sha: digest, bundleID: "device-studio-" + id, deviceID: deviceID, sceneID: sceneID,
		presetID: presetID, presetName: name, reviewID: "device-studio-e4-" + id, manifestPath: manifestPath,
		manifestSHA: manifestSHA, nodeName: "Unprinted frame insert", displayID: "main_display", rootName: "Frame Stand",
		groupKind: string(devices.GroupSmartDisplayFrame), kind: "generic_digital_frame_concept", model: "Generic Innerscene-derived digital frame concept",
		variant: "neutral electronic display, " + profile + " support derivative", description: "Unqualified generic electronic-frame candidate adapted from the CC0 Innerscene picture-frame model. This is not an Innerscene electronic product or measured digital frame.",
		license: "CC0-1.0", attribution: "Innerscene Frame Stand - Picture Frame, CC0-1.0; Vrooli neutral display UV/material adaptation; wall cleat authored by Vrooli where applicable.",
		sourceLabel: "Innerscene Frame Stand - Picture Frame, source-preserving digital-frame derivative.", sourceSHA: "7c9f5cf3e63c7f01cc911b1db9eea2c95b5e1bdadd9b75c3076b10dca70cef10",
		adaptation: "Neutral screen UV/material derivative from the retained source; the tabletop v2 keeps the easel, while wall v3 adds a generic rear cleat and omits the easel support. See " + manifestPath + " SHA-256 " + manifestSHA + ".", profile: profile, wall: wall,
	}
}

func digitalFrameVisibleV4Spec(wall bool) screenFamilySpec {
	profile, file, digest, id, deviceID, sceneID, presetID, name := "tabletop", "assets/digital-frame/innerscene-frame-tabletop-v4-glassless.glb",
		"6aed89133cfe34d46ae0fb12badc9300bd0947ae87280c98e45e562eccd94af3", "innerscene-frame-digital-neutral-tabletop-v4-glassless",
		"innerscene-digital-frame-tabletop-candidate-v4-glassless", "innerscene-digital-frame-tabletop-study-v4-glassless", "innerscene-digital-frame-tabletop-still-v4-glassless", "Innerscene-derived visible-screen digital frame tabletop v4"
	if wall {
		profile, file, digest, id, deviceID, sceneID, presetID, name = "wall", "assets/digital-frame/innerscene-frame-wall-v4-glassless.glb",
			"a555090d9896b2cbb89ce55b62a78ba80f077bc7b184ff1efe757c74f3074203", "innerscene-frame-digital-neutral-wall-v4-glassless",
			"innerscene-digital-frame-wall-candidate-v4-glassless", "innerscene-digital-frame-wall-study-v4-glassless", "innerscene-digital-frame-wall-still-v4-glassless", "Innerscene-derived visible-screen digital frame wall v4"
	}
	return screenFamilySpec{
		assetID: id, file: file, sha: digest, bundleID: "device-studio-" + id, deviceID: deviceID, sceneID: sceneID,
		presetID: presetID, presetName: name, reviewID: "device-studio-e4-" + id,
		manifestPath: digitalFrameV4Manifest, manifestSHA: digitalFrameV4ManifestSHA,
		nodeName: "Unprinted frame insert", displayID: "main_display", rootName: "Frame Stand",
		groupKind: string(devices.GroupSmartDisplayFrame), kind: "generic_digital_frame_concept", model: "Generic Innerscene-derived digital frame concept v4",
		variant:     "neutral electronic display, " + profile + " support, source protective glazing omitted from active render hierarchy",
		description: "Unqualified generic electronic-frame candidate adapted from the CC0 Innerscene picture-frame model. The high-transmission protective glazing is omitted from the active render hierarchy for visible screen content; source mesh/material/image records and binary bytes are retained. This is not an Innerscene electronic product or measured digital frame.",
		license:     "CC0-1.0 source; AGPL-3.0-only Vrooli adaptation", attribution: "Innerscene Frame Stand - Picture Frame, CC0-1.0; Vrooli visible-screen derivative; wall cleat authored by Vrooli where applicable.",
		sourceLabel: "Innerscene Frame Stand - Picture Frame; screen-visible source-preserving digital-frame derivative.", sourceSHA: "7c9f5cf3e63c7f01cc911b1db9eea2c95b5e1bdadd9b75c3076b10dca70cef10",
		adaptation: "From the existing source-preserving tabletop-v2 or wall-v3 candidate, remove only the Clear protective picture glazing node from the active node hierarchy and remap remaining node indices. Retain glazing mesh/material/image records and all binary bytes. See " + digitalFrameV4Manifest + " SHA-256 " + digitalFrameV4ManifestSHA + ".", profile: profile + "_glassless_v4", wall: wall,
	}
}

func digitalFrameWallMountedV5Spec() screenFamilySpec {
	return screenFamilySpec{
		assetID: "innerscene-frame-digital-neutral-wall-v5-mounted",
		file:    digitalFrameWallV5AssetFile, sha: digitalFrameWallV5AssetSHA,
		bundleID:     "device-studio-innerscene-frame-wall-v5-mounted",
		deviceID:     "innerscene-digital-frame-wall-candidate-v5-mounted",
		sceneID:      "innerscene-digital-frame-wall-study-v5-mounted",
		presetID:     "innerscene-digital-frame-wall-still-v5-mounted",
		presetName:   "Innerscene-derived visible-screen digital frame wall v5, mounted",
		reviewID:     "device-studio-e4-innerscene-digital-frame-wall-v5-mounted",
		manifestPath: digitalFrameWallV5Manifest, manifestSHA: digitalFrameWallV5ManifestSHA,
		nodeName: "Unprinted frame insert", displayID: "main_display", rootName: "Frame Stand",
		groupKind: string(devices.GroupSmartDisplayFrame), kind: "generic_digital_frame_concept",
		model:       "Generic Innerscene-derived digital frame concept v5",
		variant:     "wall-supported source-frame derivative with Vrooli cleat seated at rear rails",
		description: "Unqualified generic electronic-frame candidate adapted from the CC0 Innerscene picture frame. The active protective glazing is omitted, and the Vrooli-authored wall cleat is placed immediately behind the frame rails. Wall clearance is derived from the pinned study-room GLB; no physical wall mount, fastener, or load rating is claimed.",
		license:     "CC0-1.0 source; AGPL-3.0-only Vrooli adaptation",
		attribution: "Innerscene Frame Stand - Picture Frame, CC0-1.0; Vrooli visible-screen and wall-cleat placement derivative.",
		sourceLabel: "Innerscene Frame Stand - Picture Frame; v5 source-preserving wall mount placement derivative.",
		sourceSHA:   "7c9f5cf3e63c7f01cc911b1db9eea2c95b5e1bdadd9b75c3076b10dca70cef10",
		adaptation:  "Only the Vrooli-authored wall-cleat node transform changes from v4; screen and all original GLB geometry/UV/material/BIN records remain unchanged. Whole-frame wall clearance is derived from the support plane and full source bounds. See " + digitalFrameWallV5Manifest + " SHA-256 " + digitalFrameWallV5ManifestSHA + ".",
		profile:     "wall_mounted_v5", wall: true,
	}
}

func vehicleStackSpec(profile string) screenFamilySpec {
	presetID, presetName := "generic-center-stack-carplay-sim-still-v1", "Generic center stack · CarPlay-style simulated screen"
	if profile == "android_auto_simulated" {
		presetID, presetName = "generic-center-stack-android-auto-sim-still-v1", "Generic center stack · Android Auto-style simulated screen"
	}
	return screenFamilySpec{
		assetID: "vrooli-generic-vehicle-center-stack-concept-v2-screen-vflip", file: vehicleStackAssetFile, sha: vehicleStackScreenV2SHA,
		bundleID: "device-studio-generic-vehicle-center-stack-v2-screen-vflip", deviceID: "generic-vehicle-center-stack-candidate-v2-screen-vflip",
		sceneID: "generic-vehicle-center-stack-study-candidate-v2-screen-vflip", presetID: presetID + "-screen-vflip-v2", presetName: presetName + " · screen UV v2",
		reviewID: "device-studio-e4-generic-vehicle-center-stack-candidate-v2-screen-vflip", manifestPath: vehicleStackManifest, manifestSHA: vehicleStackManifestSHA,
		nodeName: "CenterDisplay", displayID: "main_display", rootName: "Generic Vehicle Center Stack Concept",
		groupKind: string(devices.GroupVehicleHost), kind: "generic_vehicle_center_stack_concept", model: "Generic authored vehicle center-stack concept",
		variant:     "380 x 250 x 120 mm authored review geometry; simulated screen-use profiles",
		description: "Unqualified Vrooli-authored center-stack candidate with simulated screen-use profiles. The profile label does not provide Apple CarPlay or Android Auto functionality, vehicle compatibility, safety review, or an in-vehicle fit.",
		license:     "AGPL-3.0-only", attribution: "Vrooli-authored generic center-stack concept; no manufacturer geometry.",
		sourceLabel: "Deterministic Vrooli GLB derivation script; screen-only vertical UV correction.", sourceSHA: vehicleStackAssetSHA,
		adaptation: "Private TEXCOORD_0 accessor for CenterDisplay primitive 0 flips V only; the source BIN prefix and all positions, normals, indices, materials, and other accessors remain unchanged.", profile: "simulated_vehicle_screen_use", screenVFlip: true,
	}
}

func vehicleStackV3Spec(profile string) screenFamilySpec {
	presetID, presetName := "generic-dashboard-cc0-carplay-sim-still-v1", "Grab3D dashboard host · CarPlay-style simulated screen"
	if profile == "android_auto_simulated" {
		presetID, presetName = "generic-dashboard-cc0-android-auto-sim-still-v1", "Grab3D dashboard host · Android Auto-style simulated screen"
	}
	return screenFamilySpec{
		assetID: "grab3d-generic-dashboard-neutral-display-v1", file: vehicleStackV3AssetFile, sha: vehicleStackV3AssetSHA,
		bundleID: "device-studio-grab3d-dashboard-neutral-display-v1", deviceID: "grab3d-generic-dashboard-candidate-v1",
		sceneID: "grab3d-generic-dashboard-study-v1", presetID: presetID, presetName: presetName,
		reviewID: "device-studio-e4-grab3d-dashboard-neutral-display-v1", manifestPath: vehicleStackV3Manifest, manifestSHA: vehicleStackV3ManifestSHA,
		nodeName: "vehicle-display-assembly", displayID: "main_display", rootName: "car-dashboard", groupKind: string(devices.GroupVehicleHost),
		kind: "generic_vehicle_dashboard_host_concept", model: "Generic source-derived center-stack dashboard host", variant: "Grab3D CC0 dashboard; simulated screen-use only",
		description: "Unqualified generic dashboard host adapted from a CC0 source model. CarPlay-style and Android-Auto-style presentations are simulated screen layouts only; there is no integration, certification, vehicle fit, or compatibility claim.",
		license:     "CC0-1.0", attribution: "Grab3D Car Dashboard (CC0-1.0); neutral-screen adaptation by Vrooli.",
		sourceLabel: "Grab3D Car Dashboard CC0 source derivative; see retained source packet and manifest.", sourceSHA: "17a5c6a762c5eeeaa7941ee385f2bba0bec453fb3a9b208c4eb39d0d643a8baf",
		adaptation: "One clean, upper-left-origin replaceable screen plane remains; source-map overlays are removed and mint styling is neutralized. Geometry is a source-derived candidate, not product CAD.", profile: "simulated_vehicle_screen_use",
	}
}

func vehicleStudioSpec(profile string) screenFamilySpec {
	if profile != "carplay_simulated" && profile != "android_auto_simulated" {
		return screenFamilySpec{}
	}
	spec := vehicleStackV3Spec(profile)
	spec.deviceID = "grab3d-generic-dashboard-studio-candidate-v1"
	spec.sceneID = "grab3d-generic-dashboard-studio-wall-v1"
	spec.presetID = "grab3d-generic-dashboard-studio-" + profile + "-still-v1"
	spec.presetName = "Generic dashboard host · neutral studio " + profile + " candidate"
	spec.profile = "neutral_dashboard_studio"
	spec.wall = true
	spec.description = "Unqualified generic dashboard assembly shown against an authored neutral studio wall. The support is a visualization fixture only; there is no automobile interior, vehicle fit, installation, or product compatibility claim. Simulated screen profile is presentation content only."
	spec.sceneFile, spec.sceneSHA = vehicleStudioSceneFile, vehicleStudioSceneSHA
	spec.sceneManifestPath, spec.sceneManifestSHA = vehicleStudioSceneManifest, vehicleStudioSceneManifestSHA
	spec.sceneBundleID, spec.sceneReviewID, spec.sceneArtifactID = "device-studio-vehicle-neutral-studio-wall-v1", "device-studio-e5-vehicle-neutral-studio-wall-v1", "vehicle-neutral-studio-wall-v1"
	return spec
}

func vehicleStudioV2Spec(profile string) screenFamilySpec {
	spec := vehicleStudioSpec(profile)
	if spec.presetID == "" {
		return spec
	}
	spec.sceneID = "grab3d-generic-dashboard-studio-wall-v2"
	spec.presetID = "grab3d-generic-dashboard-studio-" + profile + "-still-v2"
	spec.presetName = "Generic dashboard host · neutral studio wall " + profile + " v2 candidate"
	spec.sceneFile, spec.sceneSHA = vehicleStudioSceneV2File, vehicleStudioSceneV2SHA
	spec.sceneManifestPath, spec.sceneManifestSHA = vehicleStudioSceneV2Manifest, vehicleStudioSceneV2ManifestSHA
	spec.sceneBundleID, spec.sceneReviewID, spec.sceneArtifactID = "device-studio-vehicle-neutral-studio-wall-v2", "device-studio-e5-vehicle-neutral-studio-wall-v2", "vehicle-neutral-studio-wall-v2"
	return spec
}

func roundWatchV2Spec() screenFamilySpec {
	return screenFamilySpec{
		assetID: "vrooli-generic-round-watch-concept-v2", file: roundWatchV2AssetFile, sha: roundWatchV2AssetSHA,
		bundleID: "device-studio-generic-round-watch-concept-v2", deviceID: "generic-round-watch-candidate-v2",
		sceneID: "generic-round-watch-study-v2", presetID: "generic-round-watch-still-v2", presetName: "Generic round smartwatch v2 · face-on candidate",
		reviewID: "device-studio-e4-generic-round-watch-v2", manifestPath: roundWatchV2Manifest, manifestSHA: roundWatchV2ManifestSHA,
		nodeName: "One clean circular replaceable display", displayID: "main_display", rootName: "GenericRoundWatchV2",
		groupKind: string(devices.GroupWatch), kind: "generic_round_smartwatch_style", model: "Generic round smartwatch-style concept v2",
		variant:     "47.4 mm circular case comparison envelope; authored geometry, not product CAD",
		description: "Unqualified Vrooli-authored generic round smartwatch concept. Samsung Galaxy Watch Ultra published body dimensions inform proportions only; geometry is not Samsung product geometry or identity. Screen, support, fit and motion remain candidate-only.",
		license:     "AGPL-3.0-only", attribution: "Vrooli-authored generic round-watch v2; official Samsung public references used for comparison only and are not included.",
		sourceLabel: "Deterministic stdlib Vrooli round-watch generator; Samsung published dimensions and product photos are comparison-only.",
		profile:     "round_watch_v2", displayTopology: "planar", activeShape: "circle", frontalCamera: true,
	}
}

func roundWatchV3Spec() screenFamilySpec {
	return screenFamilySpec{
		assetID: "vrooli-generic-round-watch-concept-v3", file: roundWatchV3AssetFile, sha: roundWatchV3AssetSHA,
		bundleID: "device-studio-generic-round-watch-concept-v3", deviceID: "generic-round-watch-candidate-v3",
		sceneID: "generic-round-watch-study-v3", presetID: "generic-round-watch-still-v3", presetName: "Generic round smartwatch v3 · planar strap candidate",
		reviewID: "device-studio-e5-generic-round-watch-v3", manifestPath: roundWatchV3Manifest, manifestSHA: roundWatchV3ManifestSHA,
		nodeName: "One clean circular replaceable display", displayID: "main_display", rootName: "GenericRoundWatchV2",
		groupKind: string(devices.GroupWatch), kind: "generic_round_smartwatch_style", model: "Generic round smartwatch-style concept v3",
		variant:     "47.4 mm circular case comparison envelope; authored geometry, not product CAD",
		description: "Unqualified Vrooli-authored generic round smartwatch concept v3. The v2 enclosure and display remain byte-identical; only planar normals on the two broad silicone strap faces are corrected. Samsung published dimensions/photos inform comparison only; this is not Samsung product geometry or identity.",
		license:     "AGPL-3.0-only", attribution: "Vrooli-authored generic round-watch v3, derived from v2; Samsung public references are comparison only.",
		sourceLabel: "Deterministic offline round-watch v3 normal derivative; see the versioned source manifest.", sourceSHA: roundWatchV2AssetSHA,
		adaptation: "Only the planar cap normals of the upper and lower silicone strap meshes are flattened; geometry, display, bevel/side normals, indices and material factors remain unchanged.",
		profile:    "round_watch_v3", displayTopology: "planar", activeShape: "circle", frontalCamera: true,
	}
}

func roundWatchBodyV4Spec() screenFamilySpec {
	return screenFamilySpec{
		assetID: "vrooli-generic-round-watch-body-concept-v4", file: roundWatchBodyV4AssetFile, sha: roundWatchBodyV4AssetSHA,
		bundleID: "device-studio-generic-round-watch-body-v4", deviceID: "generic-round-watch-body-candidate-v4",
		sceneID: "generic-round-watch-body-floating-study-v4", presetID: "generic-round-watch-body-floating-still-v4",
		presetName: "Generic round-watch body · floating studio review v4", reviewID: "device-studio-e5-generic-round-watch-body-v4",
		manifestPath: roundWatchBodyV4Manifest, manifestSHA: roundWatchBodyV4ManifestSHA,
		nodeName: "One clean circular replaceable display", displayID: "main_display", rootName: "GenericRoundWatchV2",
		groupKind: string(devices.GroupWatch), kind: "generic_round_smartwatch_style", model: "Generic floating round-watch body concept v4",
		variant:     "47.4 mm circular case comparison envelope; body-only authored geometry, not product CAD",
		description: "Unqualified generic round-watch body shown in a floating studio pose. The active asset omits the strap, lugs, inner contacts, and buckle; it represents no complete wearable assembly or physical attachment. Samsung published dimensions/photos inform comparison only; this is not Samsung geometry or identity.",
		license:     "AGPL-3.0-only", attribution: "Vrooli-authored generic round-watch body v4 derived from v3; Samsung public references are comparison only.",
		sourceLabel: "Deterministic offline watch-body v4 active-hierarchy derivative; see versioned source manifest.", sourceSHA: roundWatchV3AssetSHA,
		adaptation: "Active scene omits strap/lug/buckle nodes while preserving the circular display, case, crown, buttons, and source geometry/material/BIN records.",
		profile:    "floating_watch_body_v4", displayTopology: "planar", activeShape: "circle", frontalCamera: true,
	}
}

func televisionV3Spec() screenFamilySpec {
	return screenFamilySpec{
		assetID: "vrooli-generic-television-thin-bezel-v3", file: televisionV3AssetFile, sha: televisionV3AssetSHA,
		bundleID: "device-studio-generic-television-thin-bezel-v3", deviceID: "generic-television-candidate-v3",
		sceneID: "generic-television-study-v3", presetID: "generic-television-thin-bezel-still-v3", presetName: "Generic thin-bezel 43-inch television concept v3",
		reviewID: "device-studio-e4-generic-television-v3", manifestPath: televisionV3Manifest, manifestSHA: televisionV3ManifestSHA,
		nodeName: "display", displayID: "main_display", rootName: "housing", groupKind: string(devices.GroupMonitorTV),
		kind: "generic_television_concept", model: "Generic authored thin-bezel television concept v3", variant: "43-inch class comparison proportions; authored candidate, not product CAD",
		description: "Unqualified Vrooli-authored thin-bezel television v3 with rounded enclosure, separate front screen, two-foot stand and rear mounting concept. Samsung published 43-inch dimensions and photos inform comparison only; no Samsung identity, CAD or VESA compliance is claimed.",
		license:     "AGPL-3.0-only", attribution: "Vrooli-authored generic thin-bezel television v3; official Samsung references used for comparison only.",
		sourceLabel: "Deterministic finite generic-television v3 recipe; 43-inch-class published dimensions inform comparison only.",
		profile:     "television_thin_bezel_v3", primitiveIndex: 1, displayTopology: "planar", activeShape: "rect",
	}
}

func (o Owners) InitializeReviewedTelevisionV3Candidate(ctx context.Context) (ScreenFamilyStarterState, error) {
	return o.initializeScreenFamilyCandidate(ctx, televisionV3Spec())
}

func (o Owners) InitializeReviewedRoundWatchV2Candidate(ctx context.Context) (ScreenFamilyStarterState, error) {
	return o.initializeScreenFamilyCandidate(ctx, roundWatchV2Spec())
}

func (o Owners) InitializeReviewedRoundWatchV3Candidate(ctx context.Context) (ScreenFamilyStarterState, error) {
	return o.initializeScreenFamilyCandidate(ctx, roundWatchV3Spec())
}

func (o Owners) InitializeReviewedRoundWatchBodyV4Candidate(ctx context.Context) (ScreenFamilyStarterState, error) {
	return o.initializeScreenFamilyCandidate(ctx, roundWatchBodyV4Spec())
}

func (o Owners) InitializeReviewedVehicleDashboardCC0Profile(ctx context.Context, profile string) (ScreenFamilyStarterState, error) {
	if profile != "carplay_simulated" && profile != "android_auto_simulated" {
		return ScreenFamilyStarterState{}, errors.New("vehicle screen profile must be carplay_simulated or android_auto_simulated")
	}
	return o.initializeScreenFamilyCandidate(ctx, vehicleStackV3Spec(profile))
}

func (o Owners) InitializeReviewedVehicleDashboardStudioProfile(ctx context.Context, profile string) (ScreenFamilyStarterState, error) {
	if profile != "carplay_simulated" && profile != "android_auto_simulated" {
		return ScreenFamilyStarterState{}, errors.New("vehicle screen profile must be carplay_simulated or android_auto_simulated")
	}
	return o.initializeScreenFamilyCandidate(ctx, vehicleStudioSpec(profile))
}

func (o Owners) InitializeReviewedVehicleDashboardStudioV2Profile(ctx context.Context, profile string) (ScreenFamilyStarterState, error) {
	if profile != "carplay_simulated" && profile != "android_auto_simulated" {
		return ScreenFamilyStarterState{}, errors.New("vehicle screen profile must be carplay_simulated or android_auto_simulated")
	}
	return o.initializeScreenFamilyCandidate(ctx, vehicleStudioV2Spec(profile))
}

func (o Owners) InitializeReviewedSmartDisplayCandidate(ctx context.Context) (ScreenFamilyStarterState, error) {
	return o.initializeScreenFamilyCandidate(ctx, smartDisplaySpec())
}

func (o Owners) InitializeReviewedSmartDisplayV3Candidate(ctx context.Context) (ScreenFamilyStarterState, error) {
	return o.initializeScreenFamilyCandidate(ctx, smartDisplayV3Spec())
}

func smartDisplayV4Spec() screenFamilySpec {
	return screenFamilySpec{
		assetID: "vrooli-generic-smart-display-concept-v4", file: smartDisplayV4AssetFile, sha: smartDisplayV4AssetSHA,
		bundleID: "device-studio-generic-smart-display-concept-v4", deviceID: "generic-smart-display-candidate-v4",
		sceneID: "generic-smart-display-study-candidate-v4", presetID: "generic-smart-display-tabletop-still-v4",
		presetName: "Generic smart-display tabletop review · authored v4", reviewID: "device-studio-e4-generic-smart-display-candidate-v4",
		manifestPath: smartDisplayV4Manifest, manifestSHA: smartDisplayV4ManifestSHA,
		nodeName: "MainDisplayActivePlane", displayID: "main_display", rootName: "GenericSmartDisplayV3",
		groupKind: string(devices.GroupSmartDisplayFrame), kind: "generic_smart_display_concept",
		model: "Generic authored smart-display concept v4", variant: "177.4 x 120.4 x 69.5 mm conceptual envelope; reference-inspired, not measured",
		description: "Unqualified Vrooli-authored smart-display v4 candidate; smooths the rounded rear-shell bevel shading while preserving the v3 envelope and screen binding. The Nest Hub reference informs comparison only; this is not product CAD or a claim about a physical unit.",
		license:     "AGPL-3.0-only", attribution: "Vrooli-authored generic smart-display v4; official Google reference imagery is comparison-only and not included.",
		sourceLabel: "Deterministic source-preserving v4 shell-normal/material derivative of generic smart-display v3.", sourceSHA: smartDisplayV3AssetSHA,
		adaptation: "Area-weighted normals replace flat per-triangle normals only on mesh 4 Display rear shell; shell material specularity is reduced. Source geometry and screen plane remain unchanged. See " + smartDisplayV4Manifest + " SHA-256 " + smartDisplayV4ManifestSHA + ".",
		profile:    "tabletop_concept_v4",
	}
}

func smartDisplayV5Spec() screenFamilySpec {
	return screenFamilySpec{
		assetID: "vrooli-generic-smart-display-concept-v5", file: smartDisplayV5AssetFile, sha: smartDisplayV5AssetSHA,
		bundleID: "device-studio-generic-smart-display-concept-v5", deviceID: "generic-smart-display-candidate-v5",
		sceneID: "generic-smart-display-study-candidate-v5", presetID: "generic-smart-display-tabletop-still-v5",
		presetName: "Generic smart-display tabletop review · authored v5", reviewID: "device-studio-e4-generic-smart-display-candidate-v5",
		manifestPath: smartDisplayV5Manifest, manifestSHA: smartDisplayV5ManifestSHA,
		nodeName: "MainDisplayActivePlane", displayID: "main_display", rootName: "GenericSmartDisplayV3",
		groupKind: string(devices.GroupSmartDisplayFrame), kind: "generic_smart_display_concept",
		model: "Generic authored smart-display concept v5", variant: "177.4 x 120.4 x 69.5 mm conceptual envelope; higher-resolution rounded shell",
		description: "Unqualified Vrooli-authored smart-display v5 candidate. The rear-shell rounded contour uses 48 angular segments per corner, while screen binding and nominal envelope remain as in v3. Google dimensions are comparison context only; no branded identity, product CAD, or physical-device claim is made.",
		license:     "AGPL-3.0-only", attribution: "Vrooli-authored generic smart-display v5; official Google reference imagery is comparison-only and not included.",
		sourceLabel: "Deterministic source recipe derivative with higher rear-shell contour tessellation; screen plane unchanged.", sourceSHA: smartDisplayV3AssetSHA,
		adaptation: "Rebuild only the Display rear shell rounded-prism contour at 48 segments per corner; preserve all other component geometry, screen selector, active area, and envelope. Graphite polymer specularity follows v4. See " + smartDisplayV5Manifest + " SHA-256 " + smartDisplayV5ManifestSHA + ".",
		profile:    "tabletop_concept_v5",
	}
}

func (o Owners) InitializeReviewedSmartDisplayV4Candidate(ctx context.Context) (ScreenFamilyStarterState, error) {
	return o.initializeScreenFamilyCandidate(ctx, smartDisplayV4Spec())
}

func (o Owners) InitializeReviewedSmartDisplayV5Candidate(ctx context.Context) (ScreenFamilyStarterState, error) {
	return o.initializeScreenFamilyCandidate(ctx, smartDisplayV5Spec())
}

func smartDisplayV6Spec() screenFamilySpec {
	spec := smartDisplayV5Spec()
	spec.assetID, spec.file, spec.sha = "vrooli-generic-smart-display-concept-v6", smartDisplayV6AssetFile, smartDisplayV6AssetSHA
	spec.bundleID, spec.deviceID = "device-studio-generic-smart-display-concept-v6", "generic-smart-display-candidate-v6"
	spec.sceneID, spec.presetID = "generic-smart-display-study-candidate-v6", "generic-smart-display-tabletop-still-v6"
	spec.presetName, spec.reviewID = "Generic smart-display tabletop review · authored v6 continuous shell normals", "device-studio-e4-generic-smart-display-candidate-v6"
	spec.manifestPath, spec.manifestSHA = smartDisplayV6Manifest, smartDisplayV6ManifestSHA
	spec.model, spec.variant = "Generic authored smart-display concept v6", "48-segment rounded shell with area-weighted continuous normals"
	spec.description = "Unqualified Vrooli-authored smart-display v6 candidate. Preserves v5's rounded contour and applies area-weighted normals across coincident shell positions; separated positions retain hard edges. Screen binding and material remain unchanged."
	spec.sourceLabel, spec.sourceSHA = "Deterministic v5 contour plus v4 coincident-position shell-normal derivation.", smartDisplayV5AssetSHA
	spec.adaptation = "Preserve v5 rounded shell geometry/material and screen mapping; recompute only mesh 4 shell normals by area-weighted coincident-position groups. See " + smartDisplayV6Manifest + " SHA-256 " + smartDisplayV6ManifestSHA + "."
	spec.profile = "tabletop_concept_v6"
	return spec
}

func (o Owners) InitializeReviewedSmartDisplayV6Candidate(ctx context.Context) (ScreenFamilyStarterState, error) {
	return o.initializeScreenFamilyCandidate(ctx, smartDisplayV6Spec())
}

func smartDisplayV7Spec() screenFamilySpec {
	spec := smartDisplayV6Spec()
	spec.assetID, spec.file, spec.sha = "vrooli-generic-smart-display-concept-v7", smartDisplayV7AssetFile, smartDisplayV7AssetSHA
	spec.bundleID, spec.deviceID = "device-studio-generic-smart-display-concept-v7", "generic-smart-display-candidate-v7"
	spec.sceneID, spec.presetID = "generic-smart-display-study-candidate-v7", "generic-smart-display-tabletop-still-v7"
	spec.presetName, spec.reviewID = "Generic smart-display tabletop review · authored v7 coplanar-cap repair", "device-studio-e4-generic-smart-display-candidate-v7"
	spec.manifestPath, spec.manifestSHA = smartDisplayV7Manifest, smartDisplayV7ManifestSHA
	spec.model, spec.variant = "Generic authored smart-display concept v7", "48-segment rounded shell with coplanar front-cap removal"
	spec.description = "Unqualified Vrooli-authored smart-display v7 candidate. Removes only the 192 rear-shell front-cap triangles coplanar with the continuous bezel, retaining the shell sides/back and all display geometry, materials, and mappings. The change addresses evidenced z-fighting; the candidate still requires native appearance review."
	spec.sourceLabel, spec.sourceSHA = "Deterministic v6 shell derivative removing only the coplanar front cap.", smartDisplayV6AssetSHA
	spec.adaptation = "Preserve v6 shell vertex attributes, side/back triangles, screen plane, UVs, materials, nodes, and transforms; remove only the 192 front-cap triangles coincident with the bezel plane. See " + smartDisplayV7Manifest + " SHA-256 " + smartDisplayV7ManifestSHA + "."
	spec.profile = "tabletop_concept_v7"
	return spec
}

func (o Owners) InitializeReviewedSmartDisplayV7Candidate(ctx context.Context) (ScreenFamilyStarterState, error) {
	return o.initializeScreenFamilyCandidate(ctx, smartDisplayV7Spec())
}

func (o Owners) InitializeReviewedDigitalFrameCandidate(ctx context.Context) (ScreenFamilyStarterState, error) {
	return o.initializeScreenFamilyCandidate(ctx, digitalFrameSpec(false))
}

func (o Owners) InitializeReviewedDigitalFrameWallCandidate(ctx context.Context) (ScreenFamilyStarterState, error) {
	return o.initializeScreenFamilyCandidate(ctx, digitalFrameSpec(true))
}

func (o Owners) InitializeReviewedDigitalFrameV4Candidate(ctx context.Context) (ScreenFamilyStarterState, error) {
	return o.initializeScreenFamilyCandidate(ctx, digitalFrameVisibleV4Spec(false))
}

func (o Owners) InitializeReviewedDigitalFrameWallV4Candidate(ctx context.Context) (ScreenFamilyStarterState, error) {
	return o.initializeScreenFamilyCandidate(ctx, digitalFrameVisibleV4Spec(true))
}

func (o Owners) InitializeReviewedDigitalFrameWallV5Candidate(ctx context.Context) (ScreenFamilyStarterState, error) {
	return o.initializeScreenFamilyCandidate(ctx, digitalFrameWallMountedV5Spec())
}

func (o Owners) InitializeReviewedVehicleCenterStackCandidate(ctx context.Context) (ScreenFamilyStarterState, error) {
	return o.InitializeReviewedVehicleCenterStackProfile(ctx, "carplay_simulated")
}

func (o Owners) InitializeReviewedVehicleCenterStackProfile(ctx context.Context, profile string) (ScreenFamilyStarterState, error) {
	if profile != "carplay_simulated" && profile != "android_auto_simulated" {
		return ScreenFamilyStarterState{}, errors.New("vehicle screen profile must be carplay_simulated or android_auto_simulated")
	}
	return o.initializeScreenFamilyCandidate(ctx, vehicleStackSpec(profile))
}

func (o Owners) initializeScreenFamilyCandidate(ctx context.Context, spec screenFamilySpec) (ScreenFamilyStarterState, error) {
	if o.Starter == nil {
		return ScreenFamilyStarterState{}, errors.New("screen-family candidate initializer is unavailable")
	}
	pinned, err := o.Starter.pinRoute(ctx)
	if err != nil {
		return ScreenFamilyStarterState{}, err
	}
	base, err := o.Starter.initializePinned(pinned)
	if err != nil {
		return ScreenFamilyStarterState{}, err
	}
	room, err := o.Starter.initializeLaptopCandidatePinned(pinned)
	if err != nil {
		return ScreenFamilyStarterState{}, err
	}
	if base.Motion.MotionID != "phone-still-identity" || room.FramedScene.SceneID != laptopFramedSceneID {
		return ScreenFamilyStarterState{}, errors.New("screen-family candidate requires the pinned cup-free room and shared identity motion")
	}
	return o.Starter.initializeScreenFamilyPinned(pinned, room.FramedScene, base.Motion, spec)
}

func (s *starterInitializer) initializeScreenFamilyPinned(ctx context.Context, room scenes.Revision, still motions.Revision, spec screenFamilySpec) (ScreenFamilyStarterState, error) {
	data, err := fs.ReadFile(s.fs, spec.file)
	if err != nil {
		return ScreenFamilyStarterState{}, err
	}
	if spec.screenVFlip {
		if sha256Hex(data) != spec.sourceSHA {
			return ScreenFamilyStarterState{}, fmt.Errorf("source screen asset %q differs from its pinned parent", spec.assetID)
		}
		data, err = flipScreenUVV(data, spec.nodeName, 0)
		if err != nil {
			return ScreenFamilyStarterState{}, fmt.Errorf("derive screen-only UV view for %s: %w", spec.assetID, err)
		}
	}
	if sha256Hex(data) != spec.sha {
		return ScreenFamilyStarterState{}, fmt.Errorf("screen-family bytes differ from pinned candidate %q", spec.assetID)
	}
	file := starterFile{
		ID: spec.assetID, File: spec.file, MIME: "model/gltf-binary", SHA: spec.sha, Role: library.RoleGeometryGLB,
		License: spec.license, Attribution: spec.attribution, Data: data,
	}
	if spec.sourceSHA == "" {
		file.Provenance = library.ArtifactProvenance{Kind: library.ProvenanceAuthored, SourceLabel: spec.sourceLabel}
	} else {
		file.Provenance = library.ArtifactProvenance{Kind: library.ProvenanceAdapted, SourceLabel: spec.sourceLabel, SourceSHA256: spec.sourceSHA, Adaptation: spec.adaptation + " See " + spec.manifestPath + " SHA-256 " + spec.manifestSHA + "."}
	}
	manifest, sources, err := s.buildManifest(ctx, spec.bundleID, []starterFile{file})
	if err != nil {
		return ScreenFamilyStarterState{}, err
	}
	bundle, err := s.publish(ctx, manifest, sources, spec.reviewID)
	if err != nil {
		return ScreenFamilyStarterState{}, err
	}
	ref := artifactRef(bundle, spec.assetID)
	if ref.SHA256 != spec.sha || ref.SizeBytes != int64(len(data)) {
		return ScreenFamilyStarterState{}, errors.New("screen-family Library artifact differs from the pinned candidate bytes")
	}
	device, bounds, inventory, err := s.saveScreenFamilyDevice(ctx, spec, ref, data)
	if err != nil {
		return ScreenFamilyStarterState{}, err
	}
	scene, placement, err := s.saveScreenFamilyScene(ctx, room, device, spec, bounds)
	if err != nil {
		return ScreenFamilyStarterState{}, err
	}
	preset, err := s.ensureScreenFamilyPreset(ctx, device, scene, still, placement, spec)
	if err != nil {
		return ScreenFamilyStarterState{}, err
	}
	if device.State == devices.DeviceRevisionQualified {
		if err := s.verifyScreenFamilyQualifiedReplay(ctx, spec, device, scene, still, preset); err != nil {
			return ScreenFamilyStarterState{}, err
		}
	}
	_ = inventory
	return ScreenFamilyStarterState{Bundle: bundle, Device: device, Scene: scene, Motion: still, Preset: preset}, nil
}

func (s *starterInitializer) saveScreenFamilyDevice(ctx context.Context, spec screenFamilySpec, ref physical.ArtifactRef, data []byte) (devices.DeviceRevision, physicalBounds, rendering.PhysicalGLBInventory, error) {
	invalid := func(err error) (devices.DeviceRevision, physicalBounds, rendering.PhysicalGLBInventory, error) {
		return devices.DeviceRevision{}, physicalBounds{}, rendering.PhysicalGLBInventory{}, err
	}
	inventory, err := rendering.InspectPhysicalGLB(data)
	if err != nil {
		return invalid(fmt.Errorf("inspect %s candidate: %w", spec.assetID, err))
	}
	if len(inventory.Nodes) == 0 || len(inventory.Meshes) == 0 {
		return invalid(errors.New("screen-family candidate has no bounded core glTF geometry"))
	}
	nodes := make(map[uint32]rendering.PhysicalGLBNode, len(inventory.Nodes))
	nodesByName := make(map[string]rendering.PhysicalGLBNode, len(inventory.Nodes))
	for _, node := range inventory.Nodes {
		nodes[node.Index], nodesByName[node.Name] = node, node
	}
	var root rendering.PhysicalGLBNode
	rootOK := false
	for _, node := range inventory.Nodes {
		if node.Name == spec.rootName || node.Parent == nil {
			if !rootOK {
				root, rootOK = node, true
			}
		}
	}
	display, displayOK := nodesByName[spec.nodeName]
	if !rootOK || !displayOK || display.Mesh == nil {
		return invalid(fmt.Errorf("screen-family root %q or screen node %q is missing", spec.rootName, spec.nodeName))
	}
	bounds, err := physicalGLBWorldBounds(inventory)
	if err != nil || !bounds.valid() {
		return invalid(fmt.Errorf("derive %s GLB envelope: %w", spec.assetID, err))
	}
	partID := func(index uint32) string { return "screen-family-node-" + fmt.Sprint(index) }
	parts := make([]devices.SemanticPart, 0, len(inventory.Nodes))
	for _, node := range inventory.Nodes {
		role := "housing_component"
		if node.Index == root.Index {
			role = "device_root"
		} else if node.Index == display.Index {
			role = "display_surface"
		}
		parent := ""
		if node.Parent != nil {
			if _, ok := nodes[*node.Parent]; !ok {
				return invalid(errors.New("screen-family node has an unavailable parent"))
			}
			parent = partID(*node.Parent)
		}
		parts = append(parts, devices.SemanticPart{ID: partID(node.Index), Role: role, ParentID: parent, NodeIndex: node.Index, NodeName: node.Name, LocalFrame: node.Frame})
	}
	if display.Mesh == nil || int(*display.Mesh) >= len(inventory.Meshes) {
		return invalid(errors.New("screen-family display node has no mesh"))
	}
	mesh := inventory.Meshes[*display.Mesh]
	if len(mesh.Primitives) == 0 {
		return invalid(errors.New("screen-family display has no primitive"))
	}
	if int(spec.primitiveIndex) >= len(mesh.Primitives) {
		return invalid(fmt.Errorf("screen-family display primitive %d is missing", spec.primitiveIndex))
	}
	primitive := mesh.Primitives[spec.primitiveIndex]
	materialFound := false
	for _, candidate := range inventory.Materials {
		if candidate.Index == primitive.Material {
			materialFound = true
			break
		}
	}
	if !materialFound || (spec.sourceSHA == "" && (primitive.UVMin != [2]float64{} || primitive.UVMax != [2]float64{1, 1})) ||
		(spec.sourceSHA != "" && (primitive.UVMin != [2]float64{} || primitive.UVMax != [2]float64{1, 1})) {
		return invalid(errors.New("screen-family display primitive is not a bounded unit-UV surface"))
	}
	displayBounds, err := physicalGLBNodeMeshWorldBounds(inventory, display.Index)
	if err != nil {
		return invalid(fmt.Errorf("derive screen plane bounds: %w", err))
	}
	widthMM, heightMM := (displayBounds.max.X-displayBounds.min.X)*1000, (displayBounds.max.Y-displayBounds.min.Y)*1000
	if !positiveFinite(widthMM) || !positiveFinite(heightMM) {
		return invalid(errors.New("screen-family display metric must have finite positive spans"))
	}
	evidence := []devices.Evidence{
		{Source: spec.manifestPath + " SHA-256 " + spec.manifestSHA, Kind: "candidate_asset_manifest", Qualifier: "candidate-only source, rights, geometry, and use-scope record"},
		{Source: spec.sourceLabel, Kind: "source_and_design_basis", Qualifier: spec.attribution},
	}
	measurement := func(value float64, axis string) devices.Measurement {
		return devices.Measurement{Value: &value, Unit: "mm", Meaning: "GLB-derived full-source " + axis + " span", Configuration: spec.profile + " geometry profile", Method: "derived", Uncertainty: devices.Uncertainty{Status: "unknown", Explanation: "geometry bytes are not an independent physical measurement"}, Evidence: evidence}
	}
	displayEvidence := []devices.Evidence{{Source: spec.file + " SHA-256 " + spec.sha, Kind: "derived_display_geometry", Qualifier: "single explicit active screen primitive; no physical active-area or pixel calibration claim"}}
	definition := devices.Definition{
		SchemaVersion: "device-definition/v1", ID: spec.deviceID, Group: devices.Group(spec.groupKind), Kind: spec.kind,
		IdentityStatus: devices.IdentityConcept, Model: spec.model, Variant: spec.variant, Description: spec.description,
		Dimensions: map[string]devices.Dimensions{"source_geometry_envelope": {Width: measurement((bounds.max.X-bounds.min.X)*1000, "X"), Height: measurement((bounds.max.Y-bounds.min.Y)*1000, "Y"), Depth: measurement((bounds.max.Z-bounds.min.Z)*1000, "Z")}},
		Displays: []devices.DisplaySurface{{ID: spec.displayID, Topology: func() string {
			if spec.displayTopology != "" {
				return spec.displayTopology
			}
			return "planar"
		}(), CoordinateMetric: &devices.DisplayMetric{WidthMM: widthMM, HeightMM: heightMM, Evidence: displayEvidence}, PhysicalBoundary: unknownRegion(), ActiveArea: candidateScreenRegion(spec.activeShape, 0), AppSafeArea: candidateScreenRegion(spec.activeShape, .05), Evidence: displayEvidence}},
		SupportedPoses:  []devices.Pose{{Name: "candidate-" + spec.profile, Status: "candidate_only", Evidence: evidence}},
		Supports:        []devices.SupportCapability{{Kind: familySupportKind(spec), InterfaceID: familySupportInterface(spec), Status: "candidate_only", Evidence: evidence}},
		ModelCandidates: []devices.ModelCandidate{{ID: spec.assetID, Status: "candidate_only", SHA256: ref.SHA256, Author: familyAuthor(spec), License: spec.license, Attribution: spec.attribution, RightsNotes: spec.description}},
		Provenance:      evidence, Qualification: "unqualified",
	}
	localSupport := identityTRS()
	localSupport.Translation = physical.Vec3{X: 0, Y: bounds.min.Y - root.Frame.Translation.Y, Z: bounds.min.Z - root.Frame.Translation.Z}
	primitiveIndex, materialIndex := primitive.Index, primitive.Material
	candidate := devices.DeviceCandidate{
		Version: devices.DeviceCandidateVersion, Definition: definition, Geometry: ref, GeometryUnit: "m", CoordinateConvention: physical.CoordinateConvention,
		SourceToCanonical: identityTRS(), Parts: parts,
		DisplayBindings: []devices.DisplayPartBinding{{DisplayID: spec.displayID, PartID: partID(display.Index), LocalFrame: identityTRS(), ScreenPrimitiveIndex: &primitiveIndex, ScreenMaterialIndex: &materialIndex}},
		SupportFrames:   []devices.DeviceSupportFrame{{ID: familyDeviceSupportID(spec), InterfaceID: familySupportInterface(spec), PartID: partID(root.Index), LocalFrame: localSupport}},
	}
	resolved, err := rendering.ResolvePhysicalScreenGeometry(data, candidate, spec.displayID)
	if err != nil || resolved.NodeIndex != display.Index || resolved.PrimitiveIndex != primitive.Index || resolved.MaterialIndex != primitive.Material || resolved.UVMin != [2]float64{} || resolved.UVMax != [2]float64{1, 1} {
		return invalid(fmt.Errorf("preflight screen-family display selector: mapping=%+v err=%w", resolved, err))
	}
	revision, err := s.saveScreenFamilyCandidateIfUnchanged(ctx, candidate, spec)
	if err != nil {
		return invalid(err)
	}
	return revision, bounds, inventory, nil
}

// saveScreenFamilyCandidateIfUnchanged reopens an exact scoped qualification
// child only when revision 1 still matches the immutable factory candidate and
// the qualified head differs solely by its persisted review record.
func (s *starterInitializer) saveScreenFamilyCandidateIfUnchanged(ctx context.Context, candidate devices.DeviceCandidate, spec screenFamilySpec) (devices.DeviceRevision, error) {
	_, _, digest, err := devices.CanonicalizeDeviceCandidate(candidate)
	if err != nil {
		return devices.DeviceRevision{}, err
	}
	head, err := s.devices.GetHead(ctx, candidate.Definition.ID)
	if errors.Is(err, devices.ErrRevisionNotFound) {
		return s.devices.SaveCandidate(ctx, candidate, 0)
	}
	if err != nil {
		return devices.DeviceRevision{}, err
	}
	if head.Revision == 1 && head.State == devices.DeviceRevisionCandidate && head.SHA256 == digest {
		return head, nil
	}
	factory, err := s.devices.GetRevision(ctx, candidate.Definition.ID, 1)
	if err != nil || factory.SHA256 != digest || factory.State != devices.DeviceRevisionCandidate {
		return devices.DeviceRevision{}, fmt.Errorf("screen-family factory revision differs from its pinned candidate; refusing qualified-head reuse: %w", devices.ErrRevisionConflict)
	}
	if head.Revision <= factory.Revision || head.State != devices.DeviceRevisionQualified || head.Candidate.Definition.Qualification != "qualified" || len(head.Candidate.Definition.QualificationReviews) == 0 {
		return devices.DeviceRevision{}, fmt.Errorf("screen-family head is not an exact qualified child of its factory candidate: %w", devices.ErrRevisionConflict)
	}
	base := head.Candidate
	base.Definition.Qualification = "unqualified"
	base.Definition.QualificationReviews = nil
	_, _, baseDigest, err := devices.CanonicalizeDeviceCandidate(base)
	if err != nil || baseDigest != digest {
		return devices.DeviceRevision{}, fmt.Errorf("screen-family qualified head changed factory candidate bytes: %w", devices.ErrRevisionConflict)
	}
	review := head.Candidate.Definition.QualificationReviews[len(head.Candidate.Definition.QualificationReviews)-1]
	if review.Scope != "visual_setup" || review.Result != "passed" || review.CandidateRevision != factory.Revision || review.CandidateSHA256 != factory.SHA256 ||
		review.SourceBundleID != factory.Candidate.Geometry.BundleID || review.SourceBundleRevision != factory.Candidate.Geometry.BundleRevision ||
		review.SourceManifestSHA256 != factory.Candidate.Geometry.ManifestSHA256 || review.SourceReviewSHA256 != factory.Candidate.Geometry.ReviewSHA256 ||
		review.SourceArtifactID != factory.Candidate.Geometry.ArtifactID || review.SourceArtifactSHA256 != factory.Candidate.Geometry.SHA256 ||
		!sameStringSet(review.DimensionStates, []string{"source_geometry_envelope"}) || !sameStringSet(review.DisplayIDs, []string{spec.displayID}) ||
		!sameStringSet(review.AllowedMotionIDs, []string{"static"}) || !sameStringSet(review.SupportIDs, []string{familyDeviceSupportID(spec)}) ||
		review.PresetID != spec.presetID || review.PresetRevision == 0 || review.PresetSHA256 == "" || review.SceneID != spec.sceneID || review.SceneRevision == 0 || review.SceneSHA256 == "" ||
		review.MotionID == "" || review.MotionRevision == 0 || review.MotionSHA256 == "" || !review.RightsClaimsAcknowledged || review.Reviewer == "" || review.NativeRenderJobID == "" || review.NativeOutputSHA256 == "" || review.NativeOutputKind == "" {
		return devices.DeviceRevision{}, fmt.Errorf("screen-family qualified head lacks the exact visual setup review scope: %w", devices.ErrRevisionConflict)
	}
	bundle, err := s.library.GetRevision(ctx, factory.Candidate.Geometry.BundleID, factory.Candidate.Geometry.BundleRevision)
	if err != nil || bundle.ManifestSHA256 != factory.Candidate.Geometry.ManifestSHA256 || bundle.ReviewSHA256 != factory.Candidate.Geometry.ReviewSHA256 || bundle.Review.Decision != "approved" ||
		!sameStringSet(review.SourceAllowedUses, libraryUses(bundle.Review.AllowedUses)) {
		return devices.DeviceRevision{}, fmt.Errorf("screen-family qualified review source rights differ from the exact approved source bundle: %w", devices.ErrRevisionConflict)
	}
	artifactFound := false
	for _, artifact := range bundle.Manifest.Artifacts {
		if artifact.ID == factory.Candidate.Geometry.ArtifactID && artifact.Role == factory.Candidate.Geometry.Role && artifact.SHA256 == factory.Candidate.Geometry.SHA256 && artifact.SizeBytes == factory.Candidate.Geometry.SizeBytes &&
			hasLibraryUse(artifact.Rights, library.UseRender) {
			artifactFound = true
			break
		}
	}
	if !artifactFound || bundle.Review.ManifestSHA256 != bundle.ManifestSHA256 {
		return devices.DeviceRevision{}, fmt.Errorf("screen-family qualified review source artifact is absent or no longer approved for render: %w", devices.ErrRevisionConflict)
	}
	return head, nil
}

func (s *starterInitializer) verifyScreenFamilyQualifiedReplay(ctx context.Context, spec screenFamilySpec, device devices.DeviceRevision, scene scenes.Revision, motion motions.Revision, preset shots.PhysicalPresetRevision) error {
	review := device.Candidate.Definition.QualificationReviews[len(device.Candidate.Definition.QualificationReviews)-1]
	factory, err := s.devices.GetRevision(ctx, device.DeviceID, review.CandidateRevision)
	if err != nil || factory.State != devices.DeviceRevisionCandidate || factory.SHA256 != review.CandidateSHA256 {
		return fmt.Errorf("screen-family qualification no longer points to its immutable factory candidate: %w", devices.ErrRevisionConflict)
	}
	if review.SceneID != scene.SceneID || review.SceneRevision != scene.Revision || review.SceneSHA256 != scene.SHA256 || review.MotionID != motion.MotionID || review.MotionRevision != motion.Revision || review.MotionSHA256 != motion.SHA256 {
		return fmt.Errorf("screen-family visual review is bound to a different scene or motion revision: %w", devices.ErrRevisionConflict)
	}
	reviewedPreset, err := s.presets.Get(ctx, review.PresetID, review.PresetRevision)
	if err != nil || reviewedPreset.SHA256 != review.PresetSHA256 || reviewedPreset.Setup == nil || reviewedPreset.Setup.Device == nil || reviewedPreset.Setup.Scene == nil || reviewedPreset.Setup.Motion == nil ||
		reviewedPreset.Setup.Device.Id != factory.DeviceID || reviewedPreset.Setup.Device.Revision != factory.Revision || reviewedPreset.Setup.Device.Sha256 != factory.SHA256 ||
		reviewedPreset.Setup.Scene.Id != scene.SceneID || reviewedPreset.Setup.Scene.Revision != scene.Revision || reviewedPreset.Setup.Scene.Sha256 != scene.SHA256 ||
		reviewedPreset.Setup.Motion.Id != motion.MotionID || reviewedPreset.Setup.Motion.Revision != motion.Revision || reviewedPreset.Setup.Motion.Sha256 != motion.SHA256 {
		return fmt.Errorf("screen-family review preset does not pin the exact candidate, scene, and motion: %w", devices.ErrRevisionConflict)
	}
	if preset.Setup == nil || preset.Setup.Device == nil || preset.Setup.Device.Id != device.DeviceID || preset.Setup.Device.Revision != device.Revision || preset.Setup.Device.Sha256 != device.SHA256 || preset.Setup.Scene == nil || preset.Setup.Scene.Id != scene.SceneID || preset.Setup.Scene.Revision != scene.Revision || preset.Setup.Scene.Sha256 != scene.SHA256 || preset.Setup.Motion == nil || preset.Setup.Motion.Id != motion.MotionID || preset.Setup.Motion.Revision != motion.Revision || preset.Setup.Motion.Sha256 != motion.SHA256 || preset.PresetID != spec.presetID {
		return fmt.Errorf("screen-family refreshed preset does not pin the exact qualified child and reviewed setup: %w", devices.ErrRevisionConflict)
	}
	return nil
}

func familySupportKind(spec screenFamilySpec) string {
	if spec.wall {
		return "wall_mount"
	}
	return "tabletop"
}

func familySupportInterface(spec screenFamilySpec) string {
	if spec.wall {
		return "vertical_surface"
	}
	return "horizontal_surface"
}

func familyDeviceSupportID(spec screenFamilySpec) string { return spec.deviceID + "-support" }

func candidateScreenRegion(shape string, inset float64) devices.Region {
	if shape == "" {
		return unknownRegion()
	}
	return devices.Region{Status: "candidate_only", Shape: shape, Bounds: &devices.Rect{X: inset, Y: inset, Width: 1 - 2*inset, Height: 1 - 2*inset}}
}

func familyAuthor(spec screenFamilySpec) string {
	if spec.sourceSHA != "" && spec.license == "CC0-1.0" {
		if strings.Contains(spec.sourceLabel, "Grab3D") {
			return "Grab3D; adapted by Vrooli"
		}
		return "Innerscene; adapted by Vrooli"
	}
	return "Vrooli"
}

func (s *starterInitializer) saveScreenFamilyScene(ctx context.Context, room scenes.Revision, device devices.DeviceRevision, spec screenFamilySpec, bounds physicalBounds) (scenes.Revision, ReviewedIPadPlacement, error) {
	if spec.profile == "neutral_dashboard_studio" {
		return s.saveVehicleStudioScene(ctx, room, spec, bounds)
	}
	if room.SceneID != laptopFramedSceneID || len(room.Candidate.CameraFrames) != 1 || room.Candidate.CameraFrames[0].ID != laptopFramedCameraID {
		return scenes.Revision{}, ReviewedIPadPlacement{}, errors.New("screen-family scene requires the pinned cup-free framed study room")
	}
	candidate := room.Candidate
	candidate.SceneID = spec.sceneID
	candidate.CameraFrames = append([]scenes.SceneCameraFrame(nil), room.Candidate.CameraFrames...)
	camera := candidate.CameraFrames[0]
	camera.ID = spec.sceneID + "-camera-v1"
	camera.SensorFit = scenes.SensorFitHorizontal
	identity := identityTRS()
	placement := ReviewedIPadPlacement{SceneSupportID: laptopTableSupportID, SceneSupportWorld: physical.TRS{Translation: physical.Vec3{Y: .75}, Rotation: identity.Rotation, Scale: identity.Scale}, DeviceRootRest: identity}
	if spec.wall {
		wallSupportID := familyWallSupportID
		candidate.SceneID = familyWallSceneID
		camera.ID = familyWallCameraID
		if spec.profile == "wall_glassless_v4" || spec.profile == "wall_mounted_v5" {
			candidate.SceneID = spec.sceneID
			camera.ID = spec.sceneID + "-camera-v1"
		}
		if spec.profile == "wall_mounted_v5" {
			wallSupportID = familyWallV5SupportID
		}
		placement.SceneSupportWorld = physical.TRS{Translation: physical.Vec3{Y: 1.72, Z: -.46}, Rotation: identity.Rotation, Scale: identity.Scale}
		for _, part := range candidate.Parts {
			if part.ID == familyWallSupportPartID && part.NodeName == "RoomBackWall" {
				candidate.SupportFrames = append(append([]scenes.SceneSupportFrame(nil), candidate.SupportFrames...), scenes.SceneSupportFrame{
					ID: wallSupportID, InterfaceID: "vertical_surface", FramePartID: part.ID,
					LocalFrame:          physical.TRS{Translation: physical.Vec3{Z: .02}, Rotation: identity.Rotation, Scale: identity.Scale},
					ContactRegionPartID: part.ID, LoadDirectionLocal: &physical.Vec3{Z: 1},
				})
				break
			}
		}
		placement.SceneSupportID = wallSupportID
		if !hasSceneSupport(candidate, wallSupportID) {
			return scenes.Revision{}, ReviewedIPadPlacement{}, errors.New("study room has no pinned vertical wall support part")
		}
		placement.SceneSupportID = wallSupportID
	}
	placement.DeviceSupportID = familyDeviceSupportID(spec)
	placement.SelectedCameraID = camera.ID
	placement.DisplayID = spec.displayID
	placement.DeviceRootRest.Translation.Y = -bounds.min.Y
	// Canonicalize IEEE negative zero: the preset repository omits zero-valued
	// fields on read, so a later initializer replay must compare equal.
	if placement.DeviceRootRest.Translation.Y == 0 {
		placement.DeviceRootRest.Translation.Y = 0
	}
	if spec.profile == "floating_watch_body_v4" {
		// Keep a visible air gap above the study surface for the declared
		// floating-body pose; no strap or wearable support is represented.
		placement.DeviceRootRest.Translation.Y += .12
	}
	if spec.wall {
		placement.DeviceRootRest.Translation.Y = 0
		placement.DeviceRootRest.Translation.Z = 0
		if spec.profile == "wall_mounted_v5" {
			placement.DeviceRootRest.Translation.Z = familyWallRearClearanceM - familyWallSupportFrontOffsetM - bounds.min.Z
		}
	}
	if spec.frontalCamera {
		target := physical.Vec3{X: 0, Y: placement.SceneSupportWorld.Translation.Y + (bounds.min.Y+bounds.max.Y)/2, Z: placement.SceneSupportWorld.Translation.Z}
		camera.Frame = lookAtFrame(physical.Vec3{X: target.X, Y: target.Y + .045, Z: target.Z + .52}, target)
	}
	fitted, err := fitIPadPerspectiveCamera(bounds, identityTRS(), placement, camera, 16.0/9.0, .10)
	if err != nil {
		return scenes.Revision{}, ReviewedIPadPlacement{}, fmt.Errorf("fit %s camera to full candidate bounds: %w", spec.assetID, err)
	}
	camera.Frame = fitted
	target := physical.Vec3{X: (bounds.min.X + bounds.max.X) / 2, Y: placement.SceneSupportWorld.Translation.Y + (bounds.min.Y+bounds.max.Y)/2, Z: placement.SceneSupportWorld.Translation.Z + placement.DeviceRootRest.Translation.Z + (bounds.min.Z+bounds.max.Z)/2}
	distance, err := watchCameraAxisDistance(camera.Frame, target)
	if err != nil {
		return scenes.Revision{}, ReviewedIPadPlacement{}, err
	}
	camera.Focus = &scenes.FocusTarget{Mode: scenes.FocusDistance, DistanceM: &distance}
	candidate.CameraFrames[0] = camera
	scene, err := saveSceneCandidateIfUnchanged(ctx, s.scenes, candidate)
	return scene, placement, err
}

func hasSceneSupport(candidate scenes.SceneCandidate, id string) bool {
	for _, support := range candidate.SupportFrames {
		if support.ID == id {
			return true
		}
	}
	return false
}

func (s *starterInitializer) saveVehicleStudioScene(ctx context.Context, room scenes.Revision, spec screenFamilySpec, bounds physicalBounds) (scenes.Revision, ReviewedIPadPlacement, error) {
	if room.SceneID != laptopFramedSceneID || spec.sceneFile == "" || spec.sceneSHA == "" {
		return scenes.Revision{}, ReviewedIPadPlacement{}, errors.New("vehicle studio scene requires the pinned room source and versioned neutral backdrop")
	}
	data, err := fs.ReadFile(s.fs, spec.sceneFile)
	if err != nil {
		return scenes.Revision{}, ReviewedIPadPlacement{}, err
	}
	if sha256Hex(data) != spec.sceneSHA {
		return scenes.Revision{}, ReviewedIPadPlacement{}, errors.New("vehicle studio backdrop digest differs from pinned candidate")
	}
	file := starterFile{ID: spec.sceneArtifactID, File: spec.sceneFile, MIME: "model/gltf-binary", SHA: spec.sceneSHA, Role: library.RoleGeometryGLB, License: "AGPL-3.0-only", Attribution: "Vrooli-authored neutral dashboard studio wall derived from the Device Studio room candidate.", Data: data,
		Provenance: library.ArtifactProvenance{Kind: library.ProvenanceAdapted, SourceLabel: "Vrooli-authored cup-free study room; active scene reduced to the neutral wall backdrop.", SourceSHA256: laptopRoomSHA256, Adaptation: "Active glTF scene root list retains only RoomBackWall. Omitted table/window context records remain inactive; this is a generic studio backdrop, not an automobile interior."}}
	manifest, sources, err := s.buildManifest(ctx, spec.sceneBundleID, []starterFile{file})
	if err != nil {
		return scenes.Revision{}, ReviewedIPadPlacement{}, err
	}
	bundle, err := s.publish(ctx, manifest, sources, spec.sceneReviewID)
	if err != nil {
		return scenes.Revision{}, ReviewedIPadPlacement{}, err
	}
	ref := artifactRef(bundle, spec.sceneArtifactID)

	candidate := room.Candidate
	candidate.SceneID = spec.sceneID
	if len(candidate.Artifacts) < 1 || len(candidate.CameraFrames) != 1 {
		return scenes.Revision{}, ReviewedIPadPlacement{}, errors.New("vehicle studio source room has an unexpected artifact or camera inventory")
	}
	bindingID := spec.sceneArtifactID
	candidate.Artifacts = append([]scenes.SceneArtifactBinding(nil), room.Candidate.Artifacts...)
	geometryBound := false
	for i := range candidate.Artifacts {
		if candidate.Artifacts[i].Ref.Role == library.RoleGeometryGLB {
			candidate.Artifacts[i] = scenes.SceneArtifactBinding{ID: bindingID, Ref: ref, SourceUnit: "m", SourceToCanonical: ptrTRS(identityTRS())}
			geometryBound = true
			break
		}
	}
	if !geometryBound {
		return scenes.Revision{}, ReviewedIPadPlacement{}, errors.New("vehicle studio source room has no geometry artifact binding")
	}
	var oldWall *scenes.ScenePart
	for i := range room.Candidate.Parts {
		if room.Candidate.Parts[i].NodeIndex == 0 && room.Candidate.Parts[i].NodeName == "RoomBackWall" {
			oldWall = &room.Candidate.Parts[i]
			break
		}
	}
	if oldWall == nil {
		return scenes.Revision{}, ReviewedIPadPlacement{}, errors.New("pinned source room has no RoomBackWall semantic part")
	}
	partID := "vehicle-neutral-studio-wall-part-v1"
	candidate.Parts = []scenes.ScenePart{{ID: partID, Role: "background", ArtifactBindingID: bindingID, NodeIndex: 0, NodeName: "RoomBackWall", LocalFrame: oldWall.LocalFrame}}
	materialIDs := map[string]bool{}
	candidate.MaterialSlots = nil
	for _, slot := range room.Candidate.MaterialSlots {
		if slot.PartID == oldWall.ID {
			slot.PartID = partID
			candidate.MaterialSlots = append(candidate.MaterialSlots, slot)
			materialIDs[slot.MaterialID] = true
		}
	}
	candidate.Materials = nil
	for _, material := range room.Candidate.Materials {
		if materialIDs[material.ID] {
			candidate.Materials = append(candidate.Materials, material)
		}
	}
	candidate.SupportFrames = []scenes.SceneSupportFrame{{ID: "vehicle-neutral-studio-wall-support-v1", InterfaceID: "vertical_surface", FramePartID: partID, ContactRegionPartID: partID,
		LocalFrame: physical.TRS{Translation: physical.Vec3{Z: familyWallSupportFrontOffsetM}, Rotation: identityTRS().Rotation, Scale: identityTRS().Scale}, LoadDirectionLocal: &physical.Vec3{Z: 1}}}
	candidate.Regions = nil
	candidate.SceneID = spec.sceneID
	candidate.Environment = scenes.SceneEnvironment{Model: "CONSTANT_V1", LinearRGB: [3]float64{0.18, 0.19, 0.21}, Strength: 0.65}
	identity := identityTRS()
	placement := ReviewedIPadPlacement{SceneSupportID: "vehicle-neutral-studio-wall-support-v1", SceneSupportWorld: physical.TRS{Translation: physical.Vec3{Y: 1.72, Z: -0.46}, Rotation: identity.Rotation, Scale: identity.Scale},
		DeviceSupportID: familyDeviceSupportID(spec), DisplayID: spec.displayID, DeviceRootRest: identity, SelectedCameraID: spec.sceneID + "-camera-v1"}
	placement.DeviceRootRest.Translation.Y = -bounds.min.Y
	placement.DeviceRootRest.Translation.Z = familyWallRearClearanceM - familyWallSupportFrontOffsetM - bounds.min.Z
	// Protobuf persistence omits zero-valued coordinates, so canonicalize
	// negative zero before creating the fixed preset used on subsequent replay.
	if placement.DeviceRootRest.Translation.Y == 0 {
		placement.DeviceRootRest.Translation.Y = 0
	}
	if placement.DeviceRootRest.Translation.Z == 0 {
		placement.DeviceRootRest.Translation.Z = 0
	}
	camera := candidate.CameraFrames[0]
	camera.ID, camera.SensorFit = placement.SelectedCameraID, scenes.SensorFitHorizontal
	target := physical.Vec3{Y: placement.SceneSupportWorld.Translation.Y + (bounds.min.Y+bounds.max.Y)/2, Z: placement.SceneSupportWorld.Translation.Z + placement.DeviceRootRest.Translation.Z + (bounds.min.Z+bounds.max.Z)/2}
	camera.Frame = lookAtFrame(physical.Vec3{Y: target.Y + .04, Z: target.Z + .8}, target)
	fitted, err := fitIPadPerspectiveCamera(bounds, identityTRS(), placement, camera, 16.0/9.0, .10)
	if err != nil {
		return scenes.Revision{}, ReviewedIPadPlacement{}, fmt.Errorf("fit neutral dashboard studio camera: %w", err)
	}
	camera.Frame = fitted
	distance, err := watchCameraAxisDistance(camera.Frame, target)
	if err != nil {
		return scenes.Revision{}, ReviewedIPadPlacement{}, err
	}
	camera.Focus = &scenes.FocusTarget{Mode: scenes.FocusDistance, DistanceM: &distance}
	candidate.CameraFrames = []scenes.SceneCameraFrame{camera}
	scene, err := saveSceneCandidateIfUnchanged(ctx, s.scenes, candidate)
	return scene, placement, err
}

func (s *starterInitializer) ensureScreenFamilyPreset(ctx context.Context, device devices.DeviceRevision, scene scenes.Revision, motion motions.Revision, placement ReviewedIPadPlacement, spec screenFamilySpec) (shots.PhysicalPresetRevision, error) {
	setup := &shotsv1.PhysicalShotSetup{
		Device:    &shotsv1.ComponentRevisionRef{Id: device.DeviceID, Revision: device.Revision, Sha256: device.SHA256},
		Scene:     &shotsv1.ComponentRevisionRef{Id: scene.SceneID, Revision: scene.Revision, Sha256: scene.SHA256},
		Motion:    &shotsv1.ComponentRevisionRef{Id: motion.MotionID, Revision: motion.Revision, Sha256: motion.SHA256},
		DisplayId: placement.DisplayID, DeviceSupportId: placement.DeviceSupportID, SceneSupportId: placement.SceneSupportID, CameraId: placement.SelectedCameraID,
		SceneSupportWorld: starterTRS(placement.SceneSupportWorld), SupportPivotM: starterVec3(placement.SupportPivotM), DeviceRootRest: starterTRS(placement.DeviceRootRest),
		StillProfiles: []shotsv1.PhysicalStillOutputProfile{shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1},
	}
	if current, err := s.presets.Head(ctx, spec.presetID); err == nil {
		if current.Name == spec.presetName && proto.Equal(current.Setup, setup) {
			return current, nil
		}
		return shots.PhysicalPresetRevision{}, fmt.Errorf("screen-family preset %q differs from its fixed candidate; refusing implicit revision: %w", spec.presetID, shots.ErrPhysicalPresetConflict)
	} else if !errors.Is(err, shots.ErrPhysicalPresetNotFound) {
		return shots.PhysicalPresetRevision{}, err
	}
	return s.presets.Save(ctx, spec.presetID, 0, spec.presetName, setup)
}
