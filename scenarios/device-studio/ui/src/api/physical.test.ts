import { beforeEach, describe, expect, it, vi } from "vitest";
import { exportQualificationPackage, getBundleRevisionMetadata, getReviewedPhysicalStarter, getPhysicalShotPresetRevision, importQualificationPackage, initializePhysicalStarter, listDeviceCandidates, listMotionCandidates, listPhysicalShotPresets, listSceneCandidates, physicalStarterChoices, resolvePhysicalShotPreset, reviewDeviceCandidate } from "./physical";

const clients = vi.hoisted(() => ({
  starter: { initializeReviewedStarter: vi.fn() },
  devices: { getDeviceRevision: vi.fn(), listDeviceCandidates: vi.fn(), reviewDeviceCandidate: vi.fn(), exportQualificationPackage: vi.fn(), importQualificationPackage: vi.fn() },
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

describe("initializePhysicalStarter", () => {
  beforeEach(() => vi.clearAllMocks());
  it("binds the requested candidate to the exact returned preset and preserves phone default", async () => {
    clients.starter.initializeReviewedStarter.mockResolvedValue({ candidateOnly: true, selectedStarterId: "laptop-screen-upright-180", initializedPreset: { id: "laptop-upright-still", revision: 3, sha256: "a".repeat(64) } });
    const result = await initializePhysicalStarter("laptop-screen-upright-180");
    expect(clients.starter.initializeReviewedStarter).toHaveBeenCalledWith({ starterId: "laptop-screen-upright-180" });
    expect(result.preset).toEqual({ id: "laptop-upright-still", revision: 3, sha256: "a".repeat(64) });
    clients.starter.initializeReviewedStarter.mockResolvedValue({ candidateOnly: true, phoneStudyPreset: { id: "phone-study-hero", revision: 1, sha256: "b".repeat(64) } });
    await expect(initializePhysicalStarter("phone-default")).resolves.toMatchObject({ preset: { id: "phone-study-hero", revision: 1 } });
    expect(clients.starter.initializeReviewedStarter).toHaveBeenLastCalledWith({ starterId: "" });
  });

  it("accepts the explicitly selected embedded reviewed smart-display starter", async () => {
    clients.starter.initializeReviewedStarter.mockResolvedValue({ candidateOnly: false, selectedStarterId: "generic-smart-display-concept-v7-reviewed", initializedPreset: { id: "generic-smart-display-tabletop-still-v7", revision: 2, sha256: "a".repeat(64) } });
    const result = await initializePhysicalStarter("generic-smart-display-concept-v7-reviewed");
    expect(clients.starter.initializeReviewedStarter).toHaveBeenCalledWith({ starterId: "generic-smart-display-concept-v7-reviewed" });
    expect(result.preset).toMatchObject({ id: "generic-smart-display-tabletop-still-v7", revision: 2 });
    expect(result.refs.candidateOnly).toBe(false);
  });
  it.each([
    "generic-round-watch-body-v4-floating-reviewed",
    "generic-television-concept-v3-reviewed",
    "generic-digital-frame-concept-v4-glassless-reviewed",
    "generic-digital-frame-wall-v5-mounted-reviewed",
    "generic-book-fold-v3-left-screen-vflip-static-reviewed",
    "laptop-screen-vflip-reviewed",
  ])("accepts the explicitly selected scoped reviewed starter %s", async (starterId) => {
    clients.starter.initializeReviewedStarter.mockResolvedValue({ candidateOnly: false, selectedStarterId: starterId, initializedPreset: { id: "reviewed-preset", revision: 2, sha256: "a".repeat(64) } });
    const result = await initializePhysicalStarter(starterId);
    expect(result.refs.candidateOnly).toBe(false);
    expect(result.preset.revision).toBe(2);
  });
  it("rejects mismatched starter IDs and incomplete preset references", async () => {
    clients.starter.initializeReviewedStarter.mockResolvedValue({ candidateOnly: true, selectedStarterId: "generic-book-fold", initializedPreset: { id: "preset", revision: 1, sha256: "a".repeat(64) } });
    await expect(initializePhysicalStarter("laptop-screen-upright-180")).rejects.toThrow("different physical starter");
    clients.starter.initializeReviewedStarter.mockResolvedValue({ candidateOnly: true, selectedStarterId: "laptop-screen-upright-180", initializedPreset: { id: "preset", revision: 1, sha256: "bad" } });
    await expect(initializePhysicalStarter("laptop-screen-upright-180")).rejects.toThrow("exact candidate-only preset reference");
  });
});

describe("physical starter choice labels", () => {
  it("distinguishes historical rectangular watch sources from the new circular candidate and simulation-only vehicle hosts", () => {
    const labels = new Map(physicalStarterChoices.map(({ id, label }) => [id, label]));
    expect(labels.get("generic-round-watch")).toContain("Rectangular");
    expect(labels.get("generic-round-watch-v2")).toContain("circular smartwatch");
    expect(labels.get("generic-round-watch-body-v4-floating")).toContain("floating, no strap");
    expect(labels.get("generic-smart-display-concept-v7-reviewed")).toContain("visual setup only");
    expect(labels.get("generic-vehicle-dashboard-carplay-sim")).toContain("simulated screen");
    expect(labels.get("generic-vehicle-dashboard-android-auto-sim")).toContain("simulated screen");
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

  it("submits an explicit review only with exact project, candidate digest, and observed head", async () => {
    const review = { scope: "physical_geometry" } as never;
    clients.devices.reviewDeviceCandidate.mockResolvedValue({ revision: { deviceId: "device-1", revision: 3 } });
    await reviewDeviceCandidate({ projectId: "project-1", deviceId: "device-1", candidateRevision: 2, candidateSha256: "a".repeat(64), expectedHead: 2, review });
    expect(clients.devices.reviewDeviceCandidate).toHaveBeenCalledWith({ projectId: "project-1", deviceId: "device-1", candidateRevision: 2, candidateSha256: "a".repeat(64), expectedHead: 2, review });
    await expect(reviewDeviceCandidate({ projectId: "../project", deviceId: "device-1", candidateRevision: 2, candidateSha256: "a".repeat(64), expectedHead: 2, review })).rejects.toThrow(/project, exact candidate/);
    expect(clients.devices.reviewDeviceCandidate).toHaveBeenCalledTimes(1);
  });

  it("exports and imports bounded qualification packages through the owner API", async () => {
    const bytes = new Uint8Array([1, 2, 3]);
    clients.devices.exportQualificationPackage.mockResolvedValue({ packageJson: bytes });
    await expect(exportQualificationPackage({ projectId: "project-1", deviceId: "device-1", qualifiedRevision: 3 })).resolves.toEqual(bytes);
    expect(clients.devices.exportQualificationPackage).toHaveBeenCalledWith({ projectId: "project-1", deviceId: "device-1", qualifiedRevision: 3 });
    clients.devices.importQualificationPackage.mockResolvedValue({ revision: { revision: 4 } });
    await expect(importQualificationPackage(bytes, 2)).resolves.toMatchObject({ revision: { revision: 4 } });
    expect(clients.devices.importQualificationPackage).toHaveBeenCalledWith({ packageJson: bytes, expectedHead: 2 });
    await expect(importQualificationPackage(new Uint8Array(), 2)).rejects.toThrow("bounded qualification package");
    await expect(exportQualificationPackage({ projectId: "../project", deviceId: "device-1", qualifiedRevision: 3 })).rejects.toThrow("project and exact qualified");
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

  it("recognizes only a transferred visual review whose immutable parent setup matches the qualified child", async () => {
    const review = {
      scope: "visual_setup", result: "passed", candidateRevision: 1, candidateSha256: "d".repeat(64),
      presetId: "setup-1", presetRevision: 1, presetSha256: "e".repeat(64),
      sceneId: "scene-1", sceneRevision: 3, sceneSha256: "b".repeat(64),
      motionId: "motion-1", motionRevision: 4, motionSha256: "c".repeat(64),
      displayIds: ["display-1"], supportIds: ["device-support"], allowedMotionIds: ["static"],
      sourceBundleId: "bundle-1", sourceBundleRevision: 1, sourceManifestSha256: "1".repeat(64), sourceReviewSha256: "2".repeat(64),
      sourceArtifactId: "model.glb", sourceArtifactSha256: "3".repeat(64), sourceAllowedUses: ["render"], dimensionStates: ["source_geometry_envelope"],
    };
    const geometry = { bundleId: "bundle-1", bundleRevision: 1, manifestSha256: "1".repeat(64), reviewSha256: "2".repeat(64), artifactId: "model.glb", sha256: "3".repeat(64) };
    const parentSetup = {
      device: { id: "device-1", revision: 1, sha256: "d".repeat(64) }, scene: { id: "scene-1", revision: 3, sha256: "b".repeat(64) },
      motion: { id: "motion-1", revision: 4, sha256: "c".repeat(64) }, displayId: "display-1", deviceSupportId: "device-support", sceneSupportId: "table", cameraId: "camera-1",
    };
    const childSetup = { ...parentSetup, device: { id: "device-1", revision: 2, sha256: "a".repeat(64) } };
    const child = { deviceId: "device-1", revision: 2, sha256: "a".repeat(64), state: "qualified", candidate: { geometry, definition: { qualification: "qualified", qualificationReviews: [review] } } };
    const parent = { deviceId: "device-1", revision: 1, sha256: "d".repeat(64), state: "candidate", candidate: { geometry, definition: { qualification: "unqualified" } } };
    const scene = { sceneId: "scene-1", revision: 3, sha256: "b".repeat(64), state: "candidate", candidate: { sceneId: "scene-1" } };
    const motion = { motionId: "motion-1", revision: 4, sha256: "c".repeat(64), state: "candidate", candidate: { motionId: "motion-1" } };
    const childPreset = { presetId: "setup-1", revision: 2, sha256: "f".repeat(64), candidateOnly: true, setup: childSetup };
    const parentPreset = { presetId: "setup-1", revision: 1, sha256: "e".repeat(64), setup: parentSetup };
    clients.presets.getPhysicalShotPreset.mockResolvedValue({ revision: childPreset });
    clients.devices.getDeviceRevision.mockImplementation(({ revision }) => Promise.resolve({ revision: revision === 2 ? child : parent }));
    clients.presets.getPhysicalShotPreset.mockImplementation(({ revision }) => Promise.resolve({ revision: revision === 2 ? childPreset : parentPreset }));
    clients.scenes.getSceneRevision.mockResolvedValue({ revision: scene });
    clients.motions.getMotionRevision.mockResolvedValue({ revision: motion });
    await expect(resolvePhysicalShotPreset("setup-1", 2)).resolves.toMatchObject({ inheritedVisualSetupReview: review });

    clients.presets.getPhysicalShotPreset.mockImplementation(({ revision }) => Promise.resolve({ revision: revision === 2 ? childPreset : { ...parentPreset, setup: { ...parentSetup, cameraId: "different-camera" } } }));
    await expect(resolvePhysicalShotPreset("setup-1", 2)).resolves.toMatchObject({ inheritedVisualSetupReview: undefined });
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
