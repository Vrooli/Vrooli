import { createClient } from "@connectrpc/connect";
import { DevicesService, type DeviceRevision, type QualificationReview } from "@vrooli/proto-types/device-studio/v1/devices/devices_pb";
import { ScenesService, type ImportPreparedPhotographResponse, type SceneRevision } from "@vrooli/proto-types/device-studio/v1/scenes/scenes_pb";
import { MotionsService, type MotionRevision } from "@vrooli/proto-types/device-studio/v1/motions/motions_pb";
import { PhysicalStarterService, type InitializeReviewedStarterResponse } from "@vrooli/proto-types/device-studio/v1/physical/physical_pb";
import { LibraryCatalogService, type BundleRevisionMetadata } from "@vrooli/proto-types/device-studio/v1/library/library_pb";
import { ShotsService, type PhysicalShotPresetRevision } from "@vrooli/proto-types/device-studio/v1/shots/shots_pb";
import { transport } from "./client";

export const physicalStarterClient = createClient(PhysicalStarterService, transport);
export const devicesClient = createClient(DevicesService, transport);
export const scenesClient = createClient(ScenesService, transport);
export const motionsClient = createClient(MotionsService, transport);
export const libraryCatalogClient = createClient(LibraryCatalogService, transport);
export const physicalShotPresetsClient = createClient(ShotsService, transport);

export async function importPreparedPhotograph(request: {
  preparationId: string; expectedBundleSha256: string; licenseId: string; attribution: string; sourceLabel: string;
  allowLocalRender?: boolean; sourceUseRationale?: string;
}): Promise<ImportPreparedPhotographResponse> {
  if (!/^[0-9a-f]{64}$/.test(request.preparationId) || request.expectedBundleSha256 !== request.preparationId) {
    throw new Error("Use the exact lowercase SHA-256 preparation ID as its expected bundle hash.");
  }
  if (!request.licenseId.trim() || request.licenseId.length > 128 ||
      !request.attribution.trim() || request.attribution.length > 1024 ||
      !request.sourceLabel.trim() || request.sourceLabel.length > 512) {
    throw new Error("License, attribution, and source label are required within their size limits.");
  }
  const sourceUseRationale = request.sourceUseRationale?.trim() ?? "";
  if (sourceUseRationale.length > 1024 || (request.allowLocalRender && !sourceUseRationale) || (!request.allowLocalRender && sourceUseRationale)) {
    throw new Error("Local rendering requires a bounded source-use rationale; leave the rationale empty when local rendering is not authorized.");
  }
  const response = await scenesClient.importPreparedPhotograph({
    preparationId: request.preparationId,
    expectedBundleSha256: request.expectedBundleSha256,
    licenseId: request.licenseId.trim(),
    attribution: request.attribution.trim(),
    sourceLabel: request.sourceLabel.trim(),
    allowLocalRender: request.allowLocalRender ?? false,
    sourceUseRationale,
  });
  const revision = response.revision;
  if (!revision?.candidate || revision.state !== "candidate" || revision.candidate.sceneId !== revision.sceneId ||
      !response.libraryBundleId || response.libraryRevision < 1 || !/^[0-9a-f]{64}$/.test(response.libraryManifestSha256) ||
      response.scaleStatus !== "estimated_monocular_scale_not_measured" || !Number.isFinite(response.estimatedScaleM) || response.estimatedScaleM <= 0) {
    throw new Error("Prepared-photo import did not return a complete candidate-only scene and exact library revision.");
  }
  return response;
}

function stableCatalogID(value: string) {
  return /^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/.test(value);
}

export async function listPhysicalShotPresets(afterPresetId = "") {
  if (afterPresetId && !stableCatalogID(afterPresetId)) throw new Error("The physical setup cursor must be a stable ID.");
  return physicalShotPresetsClient.listPhysicalShotPresets({ afterPresetId, pageSize: 100 });
}

export async function getPhysicalShotPresetRevision(presetId: string, revision: number): Promise<PhysicalShotPresetRevision> {
  if (!stableCatalogID(presetId) || !Number.isSafeInteger(revision) || revision < 1) throw new Error("A stable preset ID and positive revision are required.");
  const response = await physicalShotPresetsClient.getPhysicalShotPreset({ presetId, revision });
  const preset = response.revision;
  if (!preset?.setup || preset.presetId !== presetId || preset.revision !== revision || !preset.candidateOnly) {
    throw new Error("The catalog did not return the requested candidate-only physical setup.");
  }
  return preset;
}

function stableJSON(value: unknown): string {
  if (Array.isArray(value)) return `[${value.map(stableJSON).join(",")}]`;
  if (value && typeof value === "object") {
    const record = value as Record<string, unknown>;
    return `{${Object.keys(record).sort().map((key) => `${JSON.stringify(key)}:${stableJSON(record[key])}`).join(",")}}`;
  }
  return JSON.stringify(value);
}

function sameSetupExceptDevice(left: PhysicalShotPresetRevision["setup"], right: PhysicalShotPresetRevision["setup"]): boolean {
  if (!left || !right) return false;
  const { device: _leftDevice, ...leftRest } = left;
  const { device: _rightDevice, ...rightRest } = right;
  return stableJSON(leftRest) === stableJSON(rightRest);
}

async function findInheritedVisualSetupReview(preset: PhysicalShotPresetRevision, device: DeviceRevision, scene: SceneRevision, motion: MotionRevision): Promise<QualificationReview | undefined> {
  const currentSetup = preset.setup;
  const candidate = device.candidate;
  if (!currentSetup || !candidate || device.state !== "qualified" || candidate.definition?.qualification !== "qualified") return undefined;
  const geometry = candidate.geometry;
  for (const review of candidate.definition.qualificationReviews ?? []) {
    if (review.scope !== "visual_setup" || review.result !== "passed" || review.candidateRevision + 1 !== device.revision || review.presetId !== preset.presetId || review.presetRevision + 1 !== preset.revision || review.allowedMotionIds.length !== 1 || review.allowedMotionIds[0] !== "static") continue;
    if (review.sceneId !== scene.sceneId || review.sceneRevision !== scene.revision || review.sceneSha256 !== scene.sha256 ||
      review.motionId !== motion.motionId || review.motionRevision !== motion.revision || review.motionSha256 !== motion.sha256 ||
      !currentSetup.displayId || !review.displayIds.includes(currentSetup.displayId) || !currentSetup.deviceSupportId || !review.supportIds.includes(currentSetup.deviceSupportId) ||
      !geometry || review.sourceBundleId !== geometry.bundleId || review.sourceBundleRevision !== geometry.bundleRevision || review.sourceManifestSha256 !== geometry.manifestSha256 || review.sourceReviewSha256 !== geometry.reviewSha256 || review.sourceArtifactId !== geometry.artifactId || review.sourceArtifactSha256 !== geometry.sha256 || !review.sourceAllowedUses.includes("render")) continue;

    const [parentDeviceResponse, parentPresetResponse] = await Promise.all([
      devicesClient.getDeviceRevision({ deviceId: device.deviceId, revision: review.candidateRevision }),
      physicalShotPresetsClient.getPhysicalShotPreset({ presetId: review.presetId, revision: review.presetRevision }),
    ]);
    const parentDevice = parentDeviceResponse.revision;
    const parentPreset = parentPresetResponse.revision;
    if (!parentDevice?.candidate || parentDevice.state !== "candidate" || parentDevice.sha256 !== review.candidateSha256 ||
      parentDevice.candidate.definition?.qualification !== "unqualified" || stableJSON(parentDevice.candidate.geometry) !== stableJSON(candidate.geometry) ||
      !parentPreset?.setup || parentPreset.sha256 !== review.presetSha256 || parentPreset.presetId !== preset.presetId ||
      parentPreset.setup.device?.id !== device.deviceId || parentPreset.setup.device.revision !== review.candidateRevision || parentPreset.setup.device.sha256 !== review.candidateSha256 ||
      parentPreset.setup.scene?.id !== review.sceneId || parentPreset.setup.scene.revision !== review.sceneRevision || parentPreset.setup.scene.sha256 !== review.sceneSha256 ||
      parentPreset.setup.motion?.id !== review.motionId || parentPreset.setup.motion.revision !== review.motionRevision || parentPreset.setup.motion.sha256 !== review.motionSha256 ||
      !sameSetupExceptDevice(parentPreset.setup, currentSetup) || currentSetup.device?.id !== device.deviceId || currentSetup.device.revision !== device.revision || currentSetup.device.sha256 !== device.sha256) continue;
    return review;
  }
  return undefined;
}

export async function resolvePhysicalShotPreset(presetId: string, revision: number) {
  const preset = await getPhysicalShotPresetRevision(presetId, revision);
  const setup = preset.setup;
  if (!setup?.device || !setup.scene || !setup.motion) throw new Error("The physical setup does not pin all three component revisions.");
  const [deviceResponse, sceneResponse, motionResponse] = await Promise.all([
    devicesClient.getDeviceRevision({ deviceId: setup.device.id, revision: setup.device.revision }),
    scenesClient.getSceneRevision({ sceneId: setup.scene.id, revision: setup.scene.revision }),
    motionsClient.getMotionRevision({ motionId: setup.motion.id, revision: setup.motion.revision }),
  ]);
  const device = deviceResponse.revision;
  const scene = sceneResponse.revision;
  const motion = motionResponse.revision;
  if (!device?.candidate || !scene?.candidate || !motion?.candidate ||
      device.deviceId !== setup.device.id || device.revision !== setup.device.revision || device.sha256 !== setup.device.sha256 ||
      scene.sceneId !== setup.scene.id || scene.revision !== setup.scene.revision || scene.sha256 !== setup.scene.sha256 ||
      motion.motionId !== setup.motion.id || motion.revision !== setup.motion.revision || motion.sha256 !== setup.motion.sha256) {
    throw new Error("The saved setup's component revisions do not match their pinned digests.");
  }
  const inheritedVisualSetupReview = await findInheritedVisualSetupReview(preset, device, scene, motion);
  return { preset, device, scene, motion, inheritedVisualSetupReview };
}

export function listDeviceCandidates(afterDeviceId = "") {
  return devicesClient.listDeviceCandidates({ afterDeviceId, pageSize: 100 });
}

export async function reviewDeviceCandidate(input: { projectId: string; deviceId: string; candidateRevision: number; candidateSha256: string; expectedHead: number; review: QualificationReview }) {
  if (!stableCatalogID(input.deviceId) || !stableCatalogID(input.projectId) || !Number.isSafeInteger(input.candidateRevision) || input.candidateRevision < 1 || !Number.isSafeInteger(input.expectedHead) || input.expectedHead < 0 || !/^[a-f0-9]{64}$/i.test(input.candidateSha256)) {
    throw new Error("A project, exact candidate revision/digest, and observed head are required for physical review.");
  }
  return devicesClient.reviewDeviceCandidate(input);
}

export async function exportQualificationPackage(input: { projectId: string; deviceId: string; qualifiedRevision: number }) {
  if (!stableCatalogID(input.projectId) || !stableCatalogID(input.deviceId) || !Number.isSafeInteger(input.qualifiedRevision) || input.qualifiedRevision < 1) {
    throw new Error("A project and exact qualified device revision are required to export a scoped review package.");
  }
  const response = await devicesClient.exportQualificationPackage(input);
  if (!response.packageJson.length || response.packageJson.length > 128 * 1024 * 1024) throw new Error("The qualification package is empty or exceeds the 128 MiB limit.");
  return response.packageJson;
}

export async function importQualificationPackage(packageJson: Uint8Array, expectedHead: number) {
  if (!packageJson.length || packageJson.length > 128 * 1024 * 1024 || !Number.isSafeInteger(expectedHead) || expectedHead < 0) {
    throw new Error("A bounded qualification package and observed device head are required.");
  }
  return devicesClient.importQualificationPackage({ packageJson, expectedHead });
}

export function listSceneCandidates(afterSceneId = "") {
  return scenesClient.listSceneCandidates({ afterSceneId, pageSize: 100 });
}

export function listMotionCandidates(afterMotionId = "") {
  return motionsClient.listMotionCandidates({ afterMotionId, pageSize: 100 });
}

export async function getBundleRevisionMetadata(bundleId: string, revision: number): Promise<BundleRevisionMetadata> {
  if (!bundleId || !Number.isSafeInteger(revision) || revision < 1) throw new Error("A bundle ID and positive revision are required.");
  const response = await libraryCatalogClient.getBundleRevision({ bundleId, revision });
  if (!response.revision?.manifest || response.revision.manifest.bundleId !== bundleId || response.revision.manifest.revision !== revision) {
    throw new Error("The catalog did not return the requested immutable bundle revision.");
  }
  return response.revision;
}

export type ReviewedPhysicalStarter = {
  refs: InitializeReviewedStarterResponse;
  device: DeviceRevision;
  scene: SceneRevision;
  motion: MotionRevision;
};

export async function getReviewedPhysicalStarter(): Promise<ReviewedPhysicalStarter> {
  const refs = await physicalStarterClient.initializeReviewedStarter({});
  const { phoneDevice, phoneStudyScene, phoneStillMotion } = refs;
  if (!refs.candidateOnly || !phoneDevice || !phoneStudyScene || !phoneStillMotion) {
    throw new Error("The reviewed physical starter did not return all candidate revision references.");
  }
  const [deviceResult, sceneResult, motionResult] = await Promise.all([
    devicesClient.getDeviceRevision({ deviceId: phoneDevice.id, revision: phoneDevice.revision }),
    scenesClient.getSceneRevision({ sceneId: phoneStudyScene.id, revision: phoneStudyScene.revision }),
    motionsClient.getMotionRevision({ motionId: phoneStillMotion.id, revision: phoneStillMotion.revision }),
  ]);
  const device = deviceResult.revision;
  const scene = sceneResult.revision;
  const motion = motionResult.revision;
  if (!device?.candidate || !scene?.candidate || !motion?.candidate ||
      device.sha256 !== phoneDevice.sha256 || scene.sha256 !== phoneStudyScene.sha256 || motion.sha256 !== phoneStillMotion.sha256) {
    throw new Error("The reviewed physical starter revision could not be verified against its pinned digest.");
  }
  return { refs, device, scene, motion };
}

export const physicalStarterChoices = [
  { id: "phone-default", label: "Phone study setup" },
  { id: "laptop-screen-upright-180", label: "Laptop screen upright · candidate only · unqualified" },
  { id: "laptop-screen-vflip", label: "Laptop screen vertical flip · candidate only · unqualified" },
  { id: "ipad-screen-uv-oriented-180", label: "iPad screen oriented · candidate only · unqualified" },
  { id: "monitor-screen-mapped-v2", label: "Generic ultrawide monitor · candidate only · unqualified" },
  { id: "generic-round-watch", label: "Rectangular smartwatch source · historical candidate · unqualified" },
  { id: "generic-round-smartwatch-source", label: "Square-faced source smartwatch · historical candidate · unqualified" },
  { id: "generic-round-watch-v2", label: "Generic circular smartwatch v2 · authored candidate · unqualified" },
  { id: "generic-round-watch-v3", label: "Generic circular smartwatch v3 · corrected strap cap shading · candidate only · unqualified" },
  { id: "generic-round-watch-body-v4-floating", label: "Generic round-watch body v4 · floating, no strap · candidate only · unqualified" },
  { id: "generic-round-watch-body-v4-floating-reviewed", label: "Reviewed round-watch body v4 · floating, no strap · static visual setup only" },
  { id: "generic-television-concept", label: "Generic television · authored concept · candidate only · unqualified" },
  { id: "generic-television-concept-v3", label: "Generic thin-bezel 43-inch television v3 · authored candidate · unqualified" },
  { id: "generic-television-concept-v3-reviewed", label: "Reviewed generic television v3 · static visual setup only" },
  { id: "generic-book-fold", label: "Generic book-fold concept · candidate only · unqualified" },
  { id: "generic-book-fold-v3-left-screen-vflip", label: "Generic book-fold concept v3 · left-screen UV review · candidate only" },
  { id: "generic-book-fold-v3-left-screen-vflip-static", label: "Generic book-fold concept v3 · static still review · candidate only" },
  { id: "generic-book-fold-v3-left-screen-vflip-static-reviewed", label: "Reviewed generic book-fold v3 · left-screen static visual setup only" },
  { id: "generic-smart-display-concept", label: "Generic wedge smart display · authored concept · candidate only · unqualified" },
  { id: "generic-smart-display-concept-v3", label: "Generic smart display v3 · original appearance candidate · unqualified" },
  { id: "generic-smart-display-concept-v4", label: "Generic smart display v4 · smoothed shell candidate · unqualified" },
  { id: "generic-smart-display-concept-v5", label: "Generic smart display v5 · higher-resolution rounded shell · candidate only · unqualified" },
  { id: "generic-smart-display-concept-v6", label: "Generic smart display v6 · continuous rounded-shell normals · candidate only" },
  { id: "generic-smart-display-concept-v7", label: "Generic smart display v7 · coplanar bezel-cap repair · candidate only · unqualified" },
  { id: "generic-smart-display-concept-v7-reviewed", label: "Reviewed generic smart display v7 · visual setup only · unknown dimensions retained" },
  { id: "generic-digital-frame-concept", label: "Generic digital frame · tabletop · original candidate · unqualified" },
  { id: "generic-digital-frame-wall", label: "Generic digital frame · wall · original candidate · unqualified" },
  { id: "generic-digital-frame-concept-v4-glassless", label: "Generic digital frame v4 · tabletop · protective glass omitted for screen visibility · candidate only · unqualified" },
  { id: "generic-digital-frame-concept-v4-glassless-reviewed", label: "Reviewed generic digital frame v4 · tabletop · static visual setup only · glass omitted" },
  { id: "generic-digital-frame-wall-v4-glassless", label: "Generic digital frame v4 · wall · protective glass omitted for screen visibility · candidate only · unqualified" },
  { id: "generic-digital-frame-wall-v5-mounted", label: "Generic digital frame v5 · wall cleat seated behind frame · candidate only · unqualified" },
  { id: "generic-digital-frame-wall-v5-mounted-reviewed", label: "Reviewed generic digital frame v5 · wall mounted · static visual setup only" },
  { id: "laptop-screen-vflip-reviewed", label: "Reviewed laptop screen vflip · static visual setup only · CC BY source attribution retained" },
  { id: "generic-vehicle-center-stack-carplay-sim", label: "Generic center stack · CarPlay-style simulation · unqualified" },
  { id: "generic-vehicle-center-stack-android-auto-sim", label: "Generic center stack · Android Auto-style simulation · unqualified" },
  { id: "generic-vehicle-dashboard-carplay-sim", label: "Grab3D generic dashboard · CarPlay-style simulated screen · unqualified" },
  { id: "generic-vehicle-dashboard-android-auto-sim", label: "Grab3D generic dashboard · Android Auto-style simulated screen · unqualified" },
  { id: "generic-vehicle-dashboard-carplay-studio-v1", label: "Generic dashboard host · neutral studio wall · CarPlay-style screen simulation · candidate only" },
  { id: "generic-vehicle-dashboard-android-auto-studio-v1", label: "Generic dashboard host · neutral studio wall · Android Auto-style screen simulation · candidate only" },
  { id: "generic-vehicle-dashboard-carplay-studio-v2", label: "Generic dashboard host · neutral studio wall v2 · CarPlay-style screen simulation · candidate only" },
  { id: "generic-vehicle-dashboard-android-auto-studio-v2", label: "Generic dashboard host · neutral studio wall v2 · Android Auto-style screen simulation · candidate only" },
] as const;

export async function initializePhysicalStarter(starterId: string) {
  const choice = physicalStarterChoices.find((item) => item.id === starterId);
  if (!choice) throw new Error("Choose a supported physical starter.");
  const refs = await physicalStarterClient.initializeReviewedStarter({ starterId: starterId === "phone-default" ? "" : starterId });
  const preset = starterId === "phone-default" ? refs.phoneStudyPreset : refs.initializedPreset;
  if (starterId !== "phone-default" && refs.selectedStarterId !== starterId) throw new Error("The server returned a different physical starter than requested.");
  const mayReturnScopedQualified = new Set([
    "generic-smart-display-concept-v7", "generic-smart-display-concept-v7-reviewed",
    "generic-round-watch-body-v4-floating-reviewed", "generic-television-concept-v3-reviewed",
    "generic-digital-frame-concept-v4-glassless-reviewed", "generic-digital-frame-wall-v5-mounted-reviewed",
    "generic-book-fold-v3-left-screen-vflip-static-reviewed", "laptop-screen-vflip-reviewed",
  ]).has(starterId);
  if ((!refs.candidateOnly && !mayReturnScopedQualified) || !preset?.id || !stableCatalogID(preset.id) || !Number.isSafeInteger(preset.revision) || preset.revision < 1 || !/^[a-f0-9]{64}$/i.test(preset.sha256)) {
    throw new Error("The physical starter did not return an exact candidate-only preset reference or explicitly reviewed preset reference.");
  }
  return { refs, preset };
}
