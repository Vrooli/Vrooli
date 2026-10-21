import { createClient } from "@connectrpc/connect";
import { DevicesService, type DeviceRevision } from "@vrooli/proto-types/device-studio/v1/devices/devices_pb";
import { ScenesService, type SceneRevision } from "@vrooli/proto-types/device-studio/v1/scenes/scenes_pb";
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
  return { preset, device, scene, motion };
}

export function listDeviceCandidates(afterDeviceId = "") {
  return devicesClient.listDeviceCandidates({ afterDeviceId, pageSize: 100 });
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
  { id: "generic-round-watch", label: "Generic watch · candidate only · unqualified" },
  { id: "generic-round-smartwatch-source", label: "Generic round smartwatch · source-derived · candidate only · unqualified" },
  { id: "generic-television-concept", label: "Generic television · authored concept · candidate only · unqualified" },
  { id: "generic-book-fold", label: "Generic book-fold concept · candidate only · unqualified" },
  { id: "generic-smart-display-concept", label: "Generic wedge smart display · authored concept · candidate only · unqualified" },
  { id: "generic-digital-frame-concept", label: "Generic digital frame · tabletop candidate · unqualified" },
  { id: "generic-digital-frame-wall", label: "Generic digital frame · wall candidate · unqualified" },
  { id: "generic-vehicle-center-stack-carplay-sim", label: "Generic center stack · CarPlay-style simulation · unqualified" },
  { id: "generic-vehicle-center-stack-android-auto-sim", label: "Generic center stack · Android Auto-style simulation · unqualified" },
] as const;

export async function initializePhysicalStarter(starterId: string) {
  const choice = physicalStarterChoices.find((item) => item.id === starterId);
  if (!choice) throw new Error("Choose a supported physical starter.");
  const refs = await physicalStarterClient.initializeReviewedStarter({ starterId: starterId === "phone-default" ? "" : starterId });
  const preset = starterId === "phone-default" ? refs.phoneStudyPreset : refs.initializedPreset;
  if (starterId !== "phone-default" && refs.selectedStarterId !== starterId) throw new Error("The server returned a different physical starter than requested.");
  if (!refs.candidateOnly || !preset?.id || !stableCatalogID(preset.id) || !Number.isSafeInteger(preset.revision) || preset.revision < 1 || !/^[a-f0-9]{64}$/i.test(preset.sha256)) {
    throw new Error("The physical starter did not return an exact candidate-only preset reference.");
  }
  return { refs, preset };
}
