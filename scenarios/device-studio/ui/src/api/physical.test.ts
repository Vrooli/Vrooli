import { beforeEach, describe, expect, it, vi } from "vitest";
import { getBundleRevisionMetadata, getReviewedPhysicalStarter, getPhysicalShotPresetRevision, listDeviceCandidates, listMotionCandidates, listPhysicalShotPresets, listSceneCandidates, resolvePhysicalShotPreset } from "./physical";

const clients = vi.hoisted(() => ({
  starter: { initializeReviewedStarter: vi.fn() },
  devices: { getDeviceRevision: vi.fn(), listDeviceCandidates: vi.fn() },
  scenes: { getSceneRevision: vi.fn(), listSceneCandidates: vi.fn() },
  motions: { getMotionRevision: vi.fn(), listMotionCandidates: vi.fn() },
  library: { getBundleRevision: vi.fn() },
  presets: { listPhysicalShotPresets: vi.fn(), getPhysicalShotPreset: vi.fn() },
}));

vi.mock("@connectrpc/connect", () => {
  const sequence = [clients.starter, clients.devices, clients.scenes, clients.motions, clients.library, clients.presets];
  let index = 0;
  return { createClient: vi.fn(() => sequence[index++]) };
});

describe("getReviewedPhysicalStarter", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    clients.starter.initializeReviewedStarter.mockResolvedValue({
      candidateOnly: true,
      phoneDevice: { id: "device-1", revision: 2, sha256: "a".repeat(64) },
      phoneStudyScene: { id: "scene-1", revision: 3, sha256: "b".repeat(64) },
      phoneStillMotion: { id: "motion-1", revision: 4, sha256: "c".repeat(64) },
    });
    clients.devices.getDeviceRevision.mockResolvedValue({ revision: { candidate: { deviceId: "device-1" }, sha256: "a".repeat(64) } });
    clients.scenes.getSceneRevision.mockResolvedValue({ revision: { candidate: { sceneId: "scene-1" }, sha256: "b".repeat(64) } });
    clients.motions.getMotionRevision.mockResolvedValue({ revision: { candidate: { motionId: "motion-1" }, sha256: "c".repeat(64) } });
  });

  it("returns only the exact candidate revisions pinned by the starter", async () => {
    const result = await getReviewedPhysicalStarter();
    expect(result.refs.phoneDevice).toMatchObject({ id: "device-1", revision: 2 });
    expect(result.device.sha256).toBe("a".repeat(64));
    expect(result.scene.sha256).toBe("b".repeat(64));
    expect(result.motion.sha256).toBe("c".repeat(64));
    expect(clients.devices.getDeviceRevision).toHaveBeenCalledWith({ deviceId: "device-1", revision: 2 });
    expect(clients.scenes.getSceneRevision).toHaveBeenCalledWith({ sceneId: "scene-1", revision: 3 });
    expect(clients.motions.getMotionRevision).toHaveBeenCalledWith({ motionId: "motion-1", revision: 4 });
  });

  it("rejects incomplete starter refs and a revision whose digest changed", async () => {
    clients.starter.initializeReviewedStarter.mockResolvedValue({ candidateOnly: true, phoneDevice: { id: "device-1" } });
    await expect(getReviewedPhysicalStarter()).rejects.toThrow("did not return all candidate revision references");

    clients.starter.initializeReviewedStarter.mockResolvedValue({
      candidateOnly: true,
      phoneDevice: { id: "device-1", revision: 2, sha256: "a".repeat(64) },
      phoneStudyScene: { id: "scene-1", revision: 3, sha256: "b".repeat(64) },
      phoneStillMotion: { id: "motion-1", revision: 4, sha256: "c".repeat(64) },
    });
    clients.devices.getDeviceRevision.mockResolvedValue({ revision: { candidate: { deviceId: "device-1" }, sha256: "f".repeat(64) } });
    await expect(getReviewedPhysicalStarter()).rejects.toThrow("could not be verified against its pinned digest");
  });
});

describe("physical component catalog clients", () => {
  beforeEach(() => vi.clearAllMocks());

  it("requests bounded stable pages for each component family", async () => {
    clients.devices.listDeviceCandidates.mockResolvedValue({ candidates: [] });
    clients.scenes.listSceneCandidates.mockResolvedValue({ candidates: [] });
    clients.motions.listMotionCandidates.mockResolvedValue({ candidates: [] });
    await listDeviceCandidates("device-cursor");
    await listSceneCandidates("scene-cursor");
    await listMotionCandidates("motion-cursor");
    expect(clients.devices.listDeviceCandidates).toHaveBeenCalledWith({ afterDeviceId: "device-cursor", pageSize: 100 });
    expect(clients.scenes.listSceneCandidates).toHaveBeenCalledWith({ afterSceneId: "scene-cursor", pageSize: 100 });
    expect(clients.motions.listMotionCandidates).toHaveBeenCalledWith({ afterMotionId: "motion-cursor", pageSize: 100 });
  });

  it("requires exact immutable bundle metadata and rejects invalid lookup keys", async () => {
    clients.library.getBundleRevision.mockResolvedValue({ revision: { manifest: { bundleId: "bundle-1", revision: 2 } } });
    await expect(getBundleRevisionMetadata("bundle-1", 2)).resolves.toMatchObject({ manifest: { bundleId: "bundle-1", revision: 2 } });
    expect(clients.library.getBundleRevision).toHaveBeenCalledWith({ bundleId: "bundle-1", revision: 2 });
    await expect(getBundleRevisionMetadata("", 2)).rejects.toThrow("bundle ID and positive revision");
    await expect(getBundleRevisionMetadata("bundle-1", 0)).rejects.toThrow("bundle ID and positive revision");
    clients.library.getBundleRevision.mockResolvedValue({ revision: { manifest: { bundleId: "other", revision: 2 } } });
    await expect(getBundleRevisionMetadata("bundle-1", 2)).rejects.toThrow("did not return the requested immutable bundle revision");
  });

  it("lists bounded saved setup heads and resolves exact component revisions by digest", async () => {
    clients.presets.listPhysicalShotPresets.mockResolvedValue({ presets: [] });
    await listPhysicalShotPresets("preset-cursor");
    expect(clients.presets.listPhysicalShotPresets).toHaveBeenCalledWith({ afterPresetId: "preset-cursor", pageSize: 100 });
    const setup = {
      device: { id: "device-1", revision: 2, sha256: "a".repeat(64) },
      scene: { id: "scene-1", revision: 3, sha256: "b".repeat(64) },
      motion: { id: "motion-1", revision: 4, sha256: "c".repeat(64) },
    };
    clients.presets.getPhysicalShotPreset.mockResolvedValue({ revision: { presetId: "setup-1", revision: 5, candidateOnly: true, setup } });
    clients.devices.getDeviceRevision.mockResolvedValue({ revision: { deviceId: "device-1", revision: 2, candidate: { deviceId: "device-1" }, sha256: "a".repeat(64) } });
    clients.scenes.getSceneRevision.mockResolvedValue({ revision: { sceneId: "scene-1", revision: 3, candidate: { sceneId: "scene-1" }, sha256: "b".repeat(64) } });
    clients.motions.getMotionRevision.mockResolvedValue({ revision: { motionId: "motion-1", revision: 4, candidate: { motionId: "motion-1" }, sha256: "c".repeat(64) } });
    const result = await resolvePhysicalShotPreset("setup-1", 5);
    expect(result.preset.presetId).toBe("setup-1");
    expect(clients.presets.getPhysicalShotPreset).toHaveBeenCalledWith({ presetId: "setup-1", revision: 5 });
    expect(clients.devices.getDeviceRevision).toHaveBeenCalledWith({ deviceId: "device-1", revision: 2 });
  });

  it("rejects preset revisions that are not candidate-only or whose component digest changed", async () => {
    clients.presets.getPhysicalShotPreset.mockResolvedValue({ revision: { presetId: "setup-1", revision: 1, candidateOnly: false, setup: {} } });
    await expect(getPhysicalShotPresetRevision("setup-1", 1)).rejects.toThrow("candidate-only physical setup");
    clients.presets.getPhysicalShotPreset.mockResolvedValue({ revision: { presetId: "setup-1", revision: 1, candidateOnly: true, setup: {
      device: { id: "device-1", revision: 1, sha256: "a".repeat(64) }, scene: { id: "scene-1", revision: 1, sha256: "b".repeat(64) }, motion: { id: "motion-1", revision: 1, sha256: "c".repeat(64) },
    } } });
    clients.devices.getDeviceRevision.mockResolvedValue({ revision: { candidate: {}, sha256: "f".repeat(64) } });
    clients.scenes.getSceneRevision.mockResolvedValue({ revision: { candidate: {}, sha256: "b".repeat(64) } });
    clients.motions.getMotionRevision.mockResolvedValue({ revision: { candidate: {}, sha256: "c".repeat(64) } });
    await expect(resolvePhysicalShotPreset("setup-1", 1)).rejects.toThrow("pinned digests");
  });
});
