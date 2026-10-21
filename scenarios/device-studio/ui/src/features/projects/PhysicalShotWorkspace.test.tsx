import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import { MediaInspectionReceiptSchema, MediaReadiness, SourceMediaKind, SourceMediaSchema } from "@vrooli/proto-types/device-studio/v1/projects/projects_pb";
import { DeviceCandidateSchema, ListDeviceCandidatesResponseSchema } from "@vrooli/proto-types/device-studio/v1/devices/devices_pb";
import { ListSceneCandidatesResponseSchema, SceneCandidateSchema } from "@vrooli/proto-types/device-studio/v1/scenes/scenes_pb";
import { ListMotionCandidatesResponseSchema, MotionCandidateSchema } from "@vrooli/proto-types/device-studio/v1/motions/motions_pb";
import { CaptureArea, CompiledShotSchema, ComponentRevisionRefSchema, MotionMode, PhysicalMotionOutputBindingSchema, PhysicalRenderBackend, PhysicalShotBindingSchema, PhysicalStillOutputProfile, PresentationKind, ScreenFitMode, ScreenPresentationSchema, ShotIssueCode, ShotIssueSchema, ShotRevisionSchema, ShotSchema, ShotSnapshotSchema, VideoAudioPolicy, VideoTimingSchema } from "@vrooli/proto-types/device-studio/v1/shots/shots_pb";
import { renderWithProviders } from "../../test-utils";
import { i18n, setLocale } from "../../i18n";
import { strings } from "../../consts/strings";
import { getBundleRevisionMetadata, getReviewedPhysicalStarter, listDeviceCandidates, listMotionCandidates, listPhysicalShotPresets, listSceneCandidates, motionsClient, resolvePhysicalShotPreset } from "../../api/physical";
import { getSourceMediaContent, projectsClient, verifySourceMediaContent } from "../../api/projects";
import { shotsClient } from "../../api/shots";
import { NewPhysicalShot, PhysicalShotEditor } from "./PhysicalShotWorkspace";

vi.mock("../../api/physical", () => ({ getReviewedPhysicalStarter: vi.fn(), getBundleRevisionMetadata: vi.fn(), listDeviceCandidates: vi.fn(), listSceneCandidates: vi.fn(), motionsClient: { getMotionRevision: vi.fn() }, listMotionCandidates: vi.fn(), listPhysicalShotPresets: vi.fn(), resolvePhysicalShotPreset: vi.fn() }));
vi.mock("../../api/projects", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/projects")>();
  return { ...actual, projectsClient: { ...actual.projectsClient, listSourceMedia: vi.fn(), getSourceMediaClockPage: vi.fn() }, getSourceMediaContent: vi.fn(), verifySourceMediaContent: vi.fn((blob: Blob) => Promise.resolve(blob)) };
});
vi.mock("../../api/shots", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/shots")>();
  return { ...actual, shotsClient: { ...actual.shotsClient, createShot: vi.fn(), getShot: vi.fn(), saveShot: vi.fn(), compileShot: vi.fn() } };
});

const media = create(SourceMediaSchema, {
  id: "screen-source", projectId: "project-1", kind: SourceMediaKind.IMAGE, readiness: MediaReadiness.READY,
  originalFilename: "screen.png", mediaType: "image/png", sizeBytes: 4n, sha256: "a".repeat(64), width: 444, height: 965,
});
const videoMedia = create(SourceMediaSchema, {
  id: "video-source", projectId: "project-1", kind: SourceMediaKind.VIDEO, readiness: MediaReadiness.READY,
  originalFilename: "screen-capture.mp4", mediaType: "video/mp4", sizeBytes: 900n, sha256: "e".repeat(64),
  inspectionReceipt: create(MediaInspectionReceiptSchema, {
    id: "inspection-receipt", sha256: "f".repeat(64), classification: "accepted", sourceSha256: "e".repeat(64), sourceSizeBytes: 900n,
    jobId: "inspection-job", attemptId: "inspection-attempt", videoStreamIndex: 0,
    videoWidth: 444, videoHeight: 965, timeBaseNumerator: 1n, timeBaseDenominator: 24n,
    streamStartPts: 0n, sourceEndPts: 48n, lastFrameDurationTicks: 1n,
    clockArtifactRef: "clock-artifact", clockSha256: "a".repeat(64), clockArtifactSizeBytes: 256n,
  }),
});
const starter = {
  refs: {
    candidateOnly: true,
    phoneDevice: { id: "device-phone", revision: 1, sha256: "b".repeat(64) },
    phoneStudyScene: { id: "scene-phone", revision: 1, sha256: "c".repeat(64) },
    phoneStillMotion: { id: "motion-still", revision: 1, sha256: "d".repeat(64) },
    phoneStudyPlacement: {
      profileId: "device-studio-reviewed-phone-study-placement-v1", profileVersion: 1,
      displayId: "main_display", deviceSupportId: "device-support", sceneSupportId: "table", selectedCameraId: "camera",
      device: { id: "device-phone", revision: 1, sha256: "b".repeat(64) },
      scene: { id: "scene-phone", revision: 1, sha256: "c".repeat(64) },
      motion: { id: "motion-still", revision: 1, sha256: "d".repeat(64) },
      sceneSupportWorld: { translationM: { y: 0.75 }, rotationXyzw: { w: 1 }, scale: { x: 1, y: 1, z: 1 } },
      supportPivotM: { y: 0.0071073174476624 },
      deviceRootRest: { translationM: {}, rotationXyzw: { x: 0.704, y: -0.08, z: 0.08, w: 0.7 }, scale: { x: 1, y: 1, z: 1 } },
    },
  },
  device: { revision: 1, candidate: { definition: { model: "Generic phone", displays: [{ id: "main_display", coordinateMetric: { widthMm: 44.4, heightMm: 96.5 } }] }, supportFrames: [{ id: "device-support", interfaceId: "horizontal_surface" }] } },
  scene: { revision: 1, candidate: { sceneId: "phone-study", supportFrames: [{ id: "table", interfaceId: "horizontal_surface" }], cameraFrames: [{ id: "camera" }] } },
  motion: { revision: 1, candidate: { motionId: "phone-still" } },
};
const savedPreset = {
  preset: { presetId: "phone-study-hero", revision: 1, sha256: "f".repeat(64), name: "Phone hero on study table", candidateOnly: true, state: "candidate", setup: {
    device: { id: "device-phone", revision: 1, sha256: "b".repeat(64) },
    scene: { id: "scene-phone", revision: 1, sha256: "c".repeat(64) },
    motion: { id: "motion-still", revision: 1, sha256: "d".repeat(64) },
    displayId: "main_display", deviceSupportId: "device-support", sceneSupportId: "table", cameraId: "camera",
    sceneSupportWorld: { translationM: { y: 0.75 }, rotationXyzw: { w: 1 }, scale: { x: 1, y: 1, z: 1 } },
    supportPivotM: { y: 0.0071073174476624 },
    deviceRootRest: { translationM: {}, rotationXyzw: { x: 0.704, y: -0.08, z: 0.08, w: 0.7 }, scale: { x: 1, y: 1, z: 1 } },
    stillProfiles: [PhysicalStillOutputProfile.PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1],
    motionProfiles: [1],
  } },
  device: { deviceId: "device-phone", revision: 1, sha256: "b".repeat(64), state: "candidate", candidate: starter.device.candidate },
  scene: { sceneId: "scene-phone", revision: 1, sha256: "c".repeat(64), state: "candidate", candidate: starter.scene.candidate },
  motion: { motionId: "motion-still", revision: 1, sha256: "d".repeat(64), state: "candidate", candidate: { version: "motion-candidate/v1", motionId: "motion-still", timebaseTicksPerSecond: { num: 24n, den: 1n }, durationTicks: 24n, tracks: [{ target: { deviceRoot: true }, interpolation: "step", keys: [{ tick: 0n, transform: { translationM: {}, rotationXyzw: { w: 1 } } }, { tick: 24n, transform: { translationM: {}, rotationXyzw: { w: 1 } } }] }], requiredCapabilities: [] } },
};

describe("physical still setup", () => {
  beforeEach(async () => {
    await setLocale("en");
    vi.mocked(getReviewedPhysicalStarter).mockResolvedValue(starter as never);
    vi.mocked(listPhysicalShotPresets).mockResolvedValue({ presets: [savedPreset.preset] } as never);
    vi.mocked(resolvePhysicalShotPreset).mockResolvedValue(savedPreset as never);
    vi.mocked(motionsClient.getMotionRevision).mockImplementation(({ motionId, revision }) => Promise.resolve({ revision: { motionId, revision, sha256: "d".repeat(64), candidate: savedPreset.motion.candidate } } as never));
    vi.mocked(listDeviceCandidates).mockResolvedValue(create(ListDeviceCandidatesResponseSchema, { candidates: [], nextAfterDeviceId: "" }));
    vi.mocked(listSceneCandidates).mockResolvedValue(create(ListSceneCandidatesResponseSchema, { candidates: [], nextAfterSceneId: "" }));
    vi.mocked(listMotionCandidates).mockResolvedValue(create(ListMotionCandidatesResponseSchema, { candidates: [], nextAfterMotionId: "" }));
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [media], nextPageToken: "" } as never);
    vi.mocked(projectsClient.getSourceMediaClockPage).mockResolvedValue({ points: [{ ordinal: 0, pts: 0n }, { ordinal: 24, pts: 24n }, { ordinal: 48, pts: 48n }], nextPageToken: "", clockSha256: "a".repeat(64), timeBaseNumerator: 1n, timeBaseDenominator: 24n } as never);
    vi.mocked(getSourceMediaContent).mockResolvedValue(new Blob(["png!"], { type: "image/png" }));
    vi.mocked(verifySourceMediaContent).mockImplementation((blob) => Promise.resolve(blob));
    vi.mocked(shotsClient.createShot).mockResolvedValue({ shot: { id: "physical-shot" } } as never);
    vi.mocked(shotsClient.saveShot).mockResolvedValue({} as never);
    vi.mocked(shotsClient.compileShot).mockResolvedValue({} as never);
    vi.stubGlobal("URL", { ...URL, createObjectURL: vi.fn(() => "blob:screen"), revokeObjectURL: vi.fn() });
  });
  afterEach(() => { cleanup(); vi.clearAllMocks(); vi.restoreAllMocks(); vi.unstubAllGlobals(); });

  it("lets the user cancel physical shot setup before creating a shot", async () => {
    const user = userEvent.setup();
    const onCancel = vi.fn();
    renderWithProviders(<NewPhysicalShot projectId="project-1" onCreated={vi.fn()} onCancel={onCancel} />);
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.physical.cancel) }));
    expect(onCancel).toHaveBeenCalledTimes(1);
    expect(shotsClient.createShot).not.toHaveBeenCalled();
  });

  it("creates a candidate-only physical shot from exact reviewed revisions and verified screen bytes", async () => {
    const user = userEvent.setup();
    const onCreated = vi.fn();
    renderWithProviders(<NewPhysicalShot projectId="project-1" onCreated={onCreated} onCancel={vi.fn()} />);

    await user.click(await screen.findByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.selectOptions(screen.getByLabelText(i18n.t(strings.physical.sourceSelect)), "screen-source");
    expect(await screen.findByAltText(i18n.t(strings.physical.sourceAlt))).toHaveAttribute("src", "blob:screen");
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.create) }));

    await waitFor(() => expect(onCreated).toHaveBeenCalledWith("physical-shot"));
    const request = vi.mocked(shotsClient.createShot).mock.calls[0]?.[0];
    expect(request?.projectId).toBe("project-1");
    expect(request?.snapshot?.physicalBinding?.device).toMatchObject({ id: "device-phone", revision: 1, sha256: "b".repeat(64) });
    expect(request?.snapshot?.physicalBinding?.scene).toMatchObject({ id: "scene-phone", revision: 1, sha256: "c".repeat(64) });
    expect(request?.snapshot?.physicalBinding?.motion).toMatchObject({ id: "motion-still", revision: 1, sha256: "d".repeat(64) });
    expect(request?.snapshot?.physicalBinding?.backend).toBe(PhysicalRenderBackend.OPTIX);
    expect(request?.snapshot?.physicalBinding?.supportPivotM?.y).toBe(0.0071073174476624);
    expect(request?.snapshot?.physicalBinding?.deviceRootRest?.rotationXyzw?.y).toBeCloseTo(-0.08);
    expect(request?.snapshot?.presentation?.sourceMediaId).toBe("screen-source");
    expect(request?.snapshot?.physicalBinding?.motionOutput).toBeUndefined();
    expect(resolvePhysicalShotPreset).toHaveBeenCalledWith("phone-study-hero", 1);
    expect(verifySourceMediaContent).toHaveBeenCalledWith(expect.any(Blob), media);
  });

  it("creates 10-second motion from an exact candidate revision using only the setup-listed V2 profile", async () => {
    const user = userEvent.setup();
    const motion = create(MotionCandidateSchema, {
      version: "motion-candidate/v1", motionId: "phone-root-demo-10s",
      timebaseTicksPerSecond: { num: 24n, den: 1n }, durationTicks: 240n,
      tracks: [{ target: { deviceRoot: true }, interpolation: "linear", keys: [
        { tick: 0n, transform: { translationM: {}, rotationXyzw: { w: 1 } } },
        { tick: 240n, transform: { translationM: {}, rotationXyzw: { w: 1 } } },
      ] }], requiredCapabilities: [],
    });
    vi.mocked(resolvePhysicalShotPreset).mockResolvedValue({
      ...savedPreset, preset: { ...savedPreset.preset, setup: { ...savedPreset.preset.setup, motionProfiles: [2] } },
    } as never);
    vi.mocked(listMotionCandidates).mockResolvedValue(create(ListMotionCandidatesResponseSchema, { candidates: [{ motionId: motion.motionId, revision: 9, sha256: "9".repeat(64), state: "candidate", candidate: motion }], nextAfterMotionId: "" }));
    renderWithProviders(<NewPhysicalShot projectId="project-1" onCreated={vi.fn()} onCancel={vi.fn()} />);
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.selectOptions(screen.getByLabelText(i18n.t(strings.physical.sourceSelect)), "screen-source");
    await user.selectOptions(screen.getByLabelText(i18n.t(strings.physical.motion)), "motion");
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.selectOptions(await screen.findByLabelText(i18n.t(strings.physical.selectMotionCandidate)), JSON.stringify([motion.motionId, 9, "9".repeat(64)]));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.create) }));
    await waitFor(() => expect(shotsClient.createShot).toHaveBeenCalledTimes(1));
    const binding = vi.mocked(shotsClient.createShot).mock.calls[0]?.[0].snapshot?.physicalBinding;
    expect(binding?.device).toMatchObject({ id: "device-phone", revision: 1, sha256: "b".repeat(64) });
    expect(binding?.scene).toMatchObject({ id: "scene-phone", revision: 1, sha256: "c".repeat(64) });
    expect(binding?.motion).toMatchObject({ id: motion.motionId, revision: 9, sha256: "9".repeat(64) });
    expect(binding?.displayId).toBe("main_display");
    expect(binding?.deviceSupportId).toBe("device-support");
    expect(binding?.sceneSupportId).toBe("table");
    expect(binding?.cameraId).toBe("camera");
    expect(binding?.sceneSupportWorld?.translationM?.y).toBe(0.75);
    expect(binding?.supportPivotM?.y).toBe(0.0071073174476624);
    expect(binding?.deviceRootRest?.rotationXyzw?.y).toBeCloseTo(-0.08);
    expect(binding?.motionOutput).toMatchObject({ profileId: "device-studio-physical-motion-1080p-v2", profileVersion: 2, frameCount: 240, outputRateNumerator: 24 });
    expect(binding?.outputProfile).toBe(PhysicalStillOutputProfile.PHYSICAL_STILL_OUTPUT_PROFILE_UNSPECIFIED);
  });

  it("does not treat the V2 frame ceiling as permission to exceed ten seconds", async () => {
    const user = userEvent.setup();
    const motion = create(MotionCandidateSchema, {
      version: "motion-candidate/v1", motionId: "phone-root-demo-12s",
      timebaseTicksPerSecond: { num: 24n, den: 1n }, durationTicks: 288n,
      tracks: [{ target: { deviceRoot: true }, interpolation: "linear", keys: [
        { tick: 0n, transform: { translationM: {}, rotationXyzw: { w: 1 } } },
        { tick: 288n, transform: { translationM: {}, rotationXyzw: { w: 1 } } },
      ] }], requiredCapabilities: [],
    });
    vi.mocked(resolvePhysicalShotPreset).mockResolvedValue({ ...savedPreset, preset: { ...savedPreset.preset, setup: { ...savedPreset.preset.setup, motionProfiles: [2] } } } as never);
    vi.mocked(listMotionCandidates).mockResolvedValue(create(ListMotionCandidatesResponseSchema, { candidates: [{ motionId: motion.motionId, revision: 10, sha256: "8".repeat(64), state: "candidate", candidate: motion }], nextAfterMotionId: "" }));
    renderWithProviders(<NewPhysicalShot projectId="project-1" onCreated={vi.fn()} onCancel={vi.fn()} />);
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.selectOptions(screen.getByLabelText(i18n.t(strings.physical.sourceSelect)), "screen-source");
    await user.selectOptions(screen.getByLabelText(i18n.t(strings.physical.motion)), "motion");
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.selectOptions(await screen.findByLabelText(i18n.t(strings.physical.selectMotionCandidate)), JSON.stringify([motion.motionId, 10, "8".repeat(64)]));
    expect(screen.getByRole("alert")).toHaveTextContent(i18n.t(strings.physical.motionIncompatible));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    expect(screen.getByRole("button", { name: i18n.t(strings.physical.create) })).toBeDisabled();
    expect(shotsClient.createShot).not.toHaveBeenCalled();
  });

  it("creates from a selected generic saved setup rather than phone-specific preset fields", async () => {
    const user = userEvent.setup();
    const other = {
      ...savedPreset,
      device: { ...savedPreset.device, deviceId: "device-tablet", sha256: "8".repeat(64), candidate: { ...savedPreset.device.candidate, definition: { model: "Reference tablet", displays: [{ id: "tablet-screen", coordinateMetric: { widthMm: 44.4, heightMm: 96.5 } }] }, supportFrames: [{ id: "tablet-foot", interfaceId: "horizontal_surface" }] } },
      scene: { ...savedPreset.scene, sceneId: "scene-table", sha256: "9".repeat(64), candidate: { ...savedPreset.scene.candidate, sceneId: "scene-table", supportFrames: [{ id: "table-top", interfaceId: "horizontal_surface" }], cameraFrames: [{ id: "table-camera" }] } },
      motion: { ...savedPreset.motion, motionId: "motion-tablet-still", sha256: "7".repeat(64), candidate: { ...savedPreset.motion.candidate, motionId: "motion-tablet-still" } },
      preset: { ...savedPreset.preset, presetId: "tablet-study-setup", name: "Tablet study", setup: {
        ...savedPreset.preset.setup,
        device: { id: "device-tablet", revision: 1, sha256: "8".repeat(64) },
        scene: { id: "scene-table", revision: 1, sha256: "9".repeat(64) },
        motion: { id: "motion-tablet-still", revision: 1, sha256: "7".repeat(64) },
        displayId: "tablet-screen", deviceSupportId: "tablet-foot", sceneSupportId: "table-top", cameraId: "table-camera",
      } },
    };
    vi.mocked(listPhysicalShotPresets).mockResolvedValue({ presets: [savedPreset.preset, other.preset] } as never);
    vi.mocked(resolvePhysicalShotPreset).mockImplementation((id) => Promise.resolve((id === "tablet-study-setup" ? other : savedPreset) as never));
    renderWithProviders(<NewPhysicalShot projectId="project-1" onCreated={vi.fn()} onCancel={vi.fn()} />);
    await user.selectOptions(await screen.findByLabelText(i18n.t(strings.physical.selectSavedSetup)), "tablet-study-setup");
    await waitFor(() => expect(resolvePhysicalShotPreset).toHaveBeenCalledWith("tablet-study-setup", 1));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.selectOptions(screen.getByLabelText(i18n.t(strings.physical.sourceSelect)), "screen-source");
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.create) }));
    await waitFor(() => expect(shotsClient.createShot).toHaveBeenCalledTimes(1));
    expect(vi.mocked(shotsClient.createShot).mock.calls[0]?.[0].snapshot?.physicalBinding).toMatchObject({
      device: { id: "device-tablet", sha256: "8".repeat(64) }, scene: { id: "scene-table", sha256: "9".repeat(64) }, motion: { id: "motion-tablet-still", sha256: "7".repeat(64) }, displayId: "tablet-screen",
    });
  });

  it("keeps other catalog candidates selectable but blocks shot creation until a matching placement preset exists", async () => {
    const user = userEvent.setup();
    vi.mocked(listSceneCandidates).mockResolvedValue(create(ListSceneCandidatesResponseSchema, { candidates: [{ sceneId: "other-scene", revision: 2, sha256: "9".repeat(64), state: "candidate", candidate: create(SceneCandidateSchema, { sceneId: "other-scene" }) }], nextAfterSceneId: "" }));
    renderWithProviders(<NewPhysicalShot projectId="project-1" onCreated={vi.fn()} onCancel={vi.fn()} />);
    const scenePicker = await screen.findByLabelText(i18n.t(strings.physical.selectSceneCandidate));
    await user.selectOptions(scenePicker, JSON.stringify(["other-scene", 2, "9".repeat(64)]));
    expect(await screen.findByTestId("physical-placement-unavailable")).toHaveTextContent(i18n.t(strings.physical.candidatePlacementUnavailable));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.selectOptions(screen.getByLabelText(i18n.t(strings.physical.sourceSelect)), "screen-source");
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.create) }));
    expect(screen.getByRole("button", { name: i18n.t(strings.physical.create) })).toBeDisabled();
    expect(shotsClient.createShot).not.toHaveBeenCalled();
  });

  it("keeps exact device and motion choices visible while refusing an unjoined setup", async () => {
    const user = userEvent.setup();
    vi.mocked(listDeviceCandidates).mockResolvedValue(create(ListDeviceCandidatesResponseSchema, { candidates: [{ deviceId: "other-device", revision: 4, sha256: "8".repeat(64), state: "candidate", candidate: create(DeviceCandidateSchema, { definition: { model: "Other device" } }) }], nextAfterDeviceId: "" }));
    vi.mocked(listMotionCandidates).mockResolvedValue(create(ListMotionCandidatesResponseSchema, { candidates: [{ motionId: "other-motion", revision: 5, sha256: "7".repeat(64), state: "candidate", candidate: create(MotionCandidateSchema, { motionId: "other-motion" }) }], nextAfterMotionId: "" }));
    renderWithProviders(<NewPhysicalShot projectId="project-1" onCreated={vi.fn()} onCancel={vi.fn()} />);
    const steps = await screen.findByRole("navigation", { name: i18n.t(strings.physical.steps) });
    await user.click(within(steps).getByRole("button", { name: /2\. Device/ }));
    const devicePicker = await screen.findByLabelText(i18n.t(strings.physical.selectDeviceCandidate));
    await user.selectOptions(devicePicker, JSON.stringify(["other-device", 4, "8".repeat(64)]));
    expect(await screen.findByTestId("physical-placement-unavailable")).toHaveTextContent(i18n.t(strings.physical.candidatePlacementUnavailable));

    await user.click(within(steps).getByRole("button", { name: /4\. Motion/ }));
    const motionPicker = await screen.findByLabelText(i18n.t(strings.physical.selectMotionCandidate));
    await user.selectOptions(motionPicker, JSON.stringify(["other-motion", 5, "7".repeat(64)]));
    expect(screen.getByTestId("physical-placement-unavailable")).toBeInTheDocument();
    await user.click(within(steps).getByRole("button", { name: /5\. Export/ }));
    expect(screen.getByRole("button", { name: i18n.t(strings.physical.create) })).toBeDisabled();
  });

  it("shows render rights only when catalog metadata matches the exact pinned artifact and review", async () => {
    const artifactRef = { bundleId: "phone-bundle", bundleRevision: 4, manifestSha256: "1".repeat(64), reviewSha256: "2".repeat(64), artifactId: "phone-glb", role: "geometry_glb", sizeBytes: 128n, sha256: "3".repeat(64) };
    vi.mocked(resolvePhysicalShotPreset).mockResolvedValue({
      ...savedPreset,
      device: { ...savedPreset.device, candidate: { ...savedPreset.device.candidate, geometry: artifactRef } },
    } as never);
    vi.mocked(getBundleRevisionMetadata).mockResolvedValue({
      manifest: { bundleId: "phone-bundle", revision: 4, artifacts: [{ id: "phone-glb", role: "geometry_glb", sizeBytes: 128n, sha256: "3".repeat(64), licenseId: "AGPL-3.0-only", rights: ["render"] }] },
      manifestSha256: "1".repeat(64), reviewSha256: "2".repeat(64),
      review: { decision: "approved", manifestSha256: "1".repeat(64), allowedUses: ["render"] },
    } as never);
    renderWithProviders(<NewPhysicalShot projectId="project-1" onCreated={vi.fn()} onCancel={vi.fn()} />);
    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.physical.continue) }));
    expect(await screen.findByTestId("physical-library-review")).toHaveTextContent(i18n.t(strings.physical.rightsReviewed));
    expect(screen.getByTestId("physical-library-review")).toHaveTextContent("phone-glb");
    expect(getBundleRevisionMetadata).toHaveBeenCalledWith("phone-bundle", 4);
  });

  it("creates a silent phone recording from the accepted source inspection and exact one-second clock range", async () => {
    const user = userEvent.setup();
    const appOnlySource = create(SourceMediaSchema, {
      ...videoMedia,
      inspectionReceipt: create(MediaInspectionReceiptSchema, {
        id: "inspection-receipt", sha256: "f".repeat(64), classification: "accepted",
        sourceSha256: "e".repeat(64), sourceSizeBytes: 900n, jobId: "inspection-job", attemptId: "inspection-attempt",
        videoStreamIndex: 0, videoWidth: 1206, videoHeight: 2400, timeBaseNumerator: 1n, timeBaseDenominator: 24n,
        streamStartPts: 0n, sourceEndPts: 48n, lastFrameDurationTicks: 1n,
        clockArtifactRef: "clock-artifact", clockSha256: "a".repeat(64), clockArtifactSizeBytes: 256n,
      }),
    });
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [appOnlySource], nextPageToken: "" } as never);
    const onCreated = vi.fn();
    renderWithProviders(<NewPhysicalShot projectId="project-1" onCreated={onCreated} onCancel={vi.fn()} />);
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.selectOptions(screen.getByLabelText(i18n.t(strings.physical.sourceSelect)), "video-source");
    expect(await screen.findByText(i18n.t(strings.physical.appOnlyCapture))).toBeInTheDocument();
    expect(screen.getByText(i18n.t(strings.physical.derivedRenderRaster))).toBeInTheDocument();
    expect(await screen.findByTestId("shot-video-timing")).toBeInTheDocument();
    await user.selectOptions(screen.getByLabelText(i18n.t(strings.shot.videoTrimEnd)), "24");
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.create) }));
    await waitFor(() => expect(onCreated).toHaveBeenCalledWith("physical-shot"));
    const snapshot = vi.mocked(shotsClient.createShot).mock.calls[0]?.[0].snapshot;
    expect(snapshot?.videoTiming).toMatchObject({
      sourceMediaId: "video-source", sourceSha256: "e".repeat(64), sourceSizeBytes: 900n,
      inspectionJobId: "inspection-job", inspectionAttemptId: "inspection-attempt", inspectionReceiptId: "inspection-receipt",
      inspectionReceiptSha256: "f".repeat(64), clockArtifactId: "clock-artifact", clockSha256: "a".repeat(64),
      trimStartPts: 0n, trimEndPts: 24n, outputRateNum: 24, audioPolicy: VideoAudioPolicy.SILENT,
    });
    expect(snapshot?.presentation?.captureArea).toBe(CaptureArea.APP_ONLY);
    expect(snapshot?.presentation?.kind).toBe(PresentationKind.PHONE_SYSTEM);
    expect(snapshot?.physicalBinding?.screenWidthPixels).toBe(1206);
    expect(snapshot?.physicalBinding?.screenHeightPixels).toBe(2621);
    expect(snapshot?.physicalBinding?.motionOutput).toMatchObject({ profileId: "device-studio-physical-motion-1080p-v1", frameCount: 24, width: 1920, height: 1080 });
    expect(getSourceMediaContent).not.toHaveBeenCalled();
    expect(verifySourceMediaContent).not.toHaveBeenCalled();
  });

  it("allows an exact 10-second root-motion recording when its setup lists V2", async () => {
    const user = userEvent.setup();
    const tenSecondMotion = {
      version: "motion-candidate/v1", motionId: "motion-still",
      timebaseTicksPerSecond: { num: 24n, den: 1n }, durationTicks: 240n,
      tracks: [{ target: { deviceRoot: true }, interpolation: "linear", keys: [
        { tick: 0n, transform: { translationM: {}, rotationXyzw: { w: 1 } } },
        { tick: 240n, transform: { translationM: {}, rotationXyzw: { w: 1 } } },
      ] }], requiredCapabilities: [],
    };
    const setup = { ...savedPreset.preset.setup, motionProfiles: [2] };
    vi.mocked(resolvePhysicalShotPreset).mockResolvedValue({ ...savedPreset, preset: { ...savedPreset.preset, setup }, motion: { ...savedPreset.motion, candidate: tenSecondMotion } } as never);
    const recording = create(SourceMediaSchema, { ...videoMedia, inspectionReceipt: create(MediaInspectionReceiptSchema, { ...videoMedia.inspectionReceipt, sourceEndPts: 240n } as never) });
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [recording], nextPageToken: "" } as never);
    renderWithProviders(<NewPhysicalShot projectId="project-1" onCreated={vi.fn()} onCancel={vi.fn()} />);
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.selectOptions(screen.getByLabelText(i18n.t(strings.physical.sourceSelect)), "video-source");
    await user.selectOptions(await screen.findByLabelText(i18n.t(strings.shot.videoTrimEnd)), "240");
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.create) }));
    await waitFor(() => expect(shotsClient.createShot).toHaveBeenCalledTimes(1));
    const snapshot = vi.mocked(shotsClient.createShot).mock.calls[0]?.[0].snapshot;
    expect(snapshot?.videoTiming).toMatchObject({ trimStartPts: 0n, trimEndPts: 240n, clockSha256: "a".repeat(64), outputRateNum: 24 });
    expect(snapshot?.physicalBinding?.motionOutput).toMatchObject({ profileId: "device-studio-physical-motion-1080p-v2", profileVersion: 2, frameCount: 240 });
  });

  it("rejects an already-composed image whose raster does not match the reviewed display", async () => {
    const wrongAspect = create(SourceMediaSchema, { ...media, width: 444, height: 1000 });
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [wrongAspect], nextPageToken: "" } as never);
    const user = userEvent.setup();
    renderWithProviders(<NewPhysicalShot projectId="project-1" onCreated={vi.fn()} onCancel={vi.fn()} />);
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.selectOptions(screen.getByLabelText(i18n.t(strings.physical.sourceSelect)), wrongAspect.id);
    expect(await screen.findByRole("alert")).toHaveTextContent(i18n.t(strings.physical.aspectIssue));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    expect(screen.getByRole("button", { name: i18n.t(strings.physical.create) })).toBeDisabled();
  });

  it("does not offer a recording when its accepted receipt belongs to different source bytes", async () => {
    const stale = create(SourceMediaSchema, {
      ...videoMedia,
      sha256: "1".repeat(64),
    });
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [stale], nextPageToken: "" } as never);
    const user = userEvent.setup();
    renderWithProviders(<NewPhysicalShot projectId="project-1" onCreated={vi.fn()} onCancel={vi.fn()} />);
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    const sourceChoice = screen.getByLabelText(i18n.t(strings.physical.sourceSelect));
    expect(within(sourceChoice).queryByRole("option", { name: /screen-capture/ })).not.toBeInTheDocument();
  });

  it("reopens a saved physical-motion revision with pinned refs and its actual output profile", async () => {
    const user = userEvent.setup();
    const snapshot = create(ShotSnapshotSchema, {
      motionMode: MotionMode.FIXED,
      physicalBinding: create(PhysicalShotBindingSchema, {
        device: create(ComponentRevisionRefSchema, { id: "device-phone", revision: 2, sha256: "b".repeat(64) }),
        scene: create(ComponentRevisionRefSchema, { id: "scene-phone", revision: 3, sha256: "c".repeat(64) }),
        motion: create(ComponentRevisionRefSchema, { id: "motion-phone", revision: 4, sha256: "d".repeat(64) }),
        backend: PhysicalRenderBackend.OPTIX,
        motionOutput: create(PhysicalMotionOutputBindingSchema, { profileId: "device-studio-physical-motion-1080p-v1", profileVersion: 1, width: 1920, height: 1080, outputRateNumerator: 24, outputRateDenominator: 1, frameCount: 48 }),
      }),
      presentation: create(ScreenPresentationSchema, { sourceMediaId: "screen-source", captureArea: CaptureArea.ALREADY_COMPOSED, kind: PresentationKind.PLAIN, fitMode: ScreenFitMode.CONTAIN }),
    });
    const head = create(ShotRevisionSchema, { shotId: "physical-shot", revision: 5n, name: "Phone motion", snapshot, issues: [] });
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: create(ShotSchema, { id: "physical-shot", projectId: "project-1", name: "Phone motion", headRevision: 5n }), head } as never);
    renderWithProviders(<PhysicalShotEditor projectId="project-1" shotId="physical-shot" />);

    expect(await screen.findByTestId("physical-shot-editor")).toBeInTheDocument();
    const heading = await screen.findByRole("heading", { name: "Phone motion" });
    expect(heading.parentElement).toHaveTextContent("Physical motion · 48 frames at 24 fps");
    expect(screen.getByText(/Device · device-phone · revision 2/)).toBeInTheDocument();
    expect(screen.getByText(/Scene · scene-phone · revision 3/)).toBeInTheDocument();
    expect(screen.getByText(/Motion · motion-phone · revision 4/)).toBeInTheDocument();
    expect(screen.getByText(/1920×1080, 48 frames at 24 fps/)).toBeInTheDocument();
    expect(await screen.findByAltText(i18n.t(strings.physical.sourceAlt))).toBeInTheDocument();
    await user.click(screen.getByText(i18n.t(strings.physical.advanced)));
    expect(screen.getByText(/This revision reopens the exact motion binding/)).toBeInTheDocument();
  });

  it("reopens a recorded physical shot with its accepted clock closure and saves it without downloading the video", async () => {
    const user = userEvent.setup();
    const timing = create(VideoTimingSchema, {
      sourceMediaId: "video-source", sourceSha256: "e".repeat(64), sourceSizeBytes: 900n,
      inspectionJobId: "inspection-job", inspectionAttemptId: "inspection-attempt", inspectionReceiptId: "inspection-receipt",
      inspectionReceiptSha256: "f".repeat(64), clockArtifactId: "clock-artifact", clockSha256: "a".repeat(64), clockSizeBytes: 256n,
      selectedVideoStream: 0, timeBaseNum: 1n, timeBaseDen: 24n, streamStartPts: 0n, sourceEndPts: 48n,
      lastFrameDurationTicks: 1n, trimStartPts: 0n, trimEndPts: 24n, playbackRateNum: 1, playbackRateDen: 1,
      outputRateNum: 24, outputRateDen: 1, audioPolicy: VideoAudioPolicy.SILENT,
      videoCompositionProfileId: "device-studio-fixed-plate-video-v1", videoCompositionProfileVersion: 1,
    });
    const binding = create(PhysicalShotBindingSchema, {
      device: create(ComponentRevisionRefSchema, { id: "device-phone", revision: 1, sha256: "b".repeat(64) }),
      scene: create(ComponentRevisionRefSchema, { id: "scene-phone", revision: 1, sha256: "c".repeat(64) }),
      motion: create(ComponentRevisionRefSchema, { id: "motion-still", revision: 1, sha256: "d".repeat(64) }),
      backend: PhysicalRenderBackend.OPTIX, screenWidthPixels: 444, screenHeightPixels: 965,
      motionOutput: create(PhysicalMotionOutputBindingSchema, { profileId: "device-studio-physical-motion-1080p-v1", profileVersion: 1, width: 1920, height: 1080, outputRateNumerator: 24, outputRateDenominator: 1, frameCount: 24 }),
    });
    const snapshot = create(ShotSnapshotSchema, {
      motionMode: MotionMode.FIXED, physicalBinding: binding, videoTiming: timing,
      presentation: create(ScreenPresentationSchema, { sourceMediaId: "video-source", captureArea: CaptureArea.APP_ONLY, kind: PresentationKind.PHONE_SYSTEM, fitMode: ScreenFitMode.CONTAIN }),
    });
    const head = create(ShotRevisionSchema, { shotId: "physical-shot", revision: 2n, name: "Phone recording", snapshot, issues: [] });
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [videoMedia], nextPageToken: "" } as never);
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: create(ShotSchema, { id: "physical-shot", projectId: "project-1", name: head.name, headRevision: 2n }), head } as never);
    vi.mocked(shotsClient.saveShot).mockResolvedValue({ revision: create(ShotRevisionSchema, { ...head, revision: 3n }) } as never);
    renderWithProviders(<PhysicalShotEditor projectId="project-1" shotId="physical-shot" />);
    await screen.findByTestId("physical-shot-editor");
    expect(await screen.findByTestId("shot-video-timing")).toBeInTheDocument();
    expect(screen.getByLabelText(i18n.t(strings.physical.sourceAlt))).toHaveAttribute("src", expect.stringContaining("video-source"));
    expect(getSourceMediaContent).not.toHaveBeenCalled();
    await user.clear(screen.getByLabelText(i18n.t(strings.shot.nameLabel)));
    await user.type(screen.getByLabelText(i18n.t(strings.shot.nameLabel)), "Phone recording · renamed");
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.saveRevision) }));
    await waitFor(() => expect(shotsClient.saveShot).toHaveBeenCalled());
    expect(vi.mocked(shotsClient.saveShot).mock.calls[0]?.[0].snapshot?.videoTiming).toEqual(timing);
    expect(vi.mocked(shotsClient.saveShot).mock.calls[0]?.[0].snapshot?.presentation?.captureArea).toBe(CaptureArea.APP_ONLY);
  });

  it("updates an exact 10-second V2 recording schedule and preserves its display-derived raster", async () => {
    const user = userEvent.setup();
    const appOnlySource = create(SourceMediaSchema, {
      ...videoMedia,
      inspectionReceipt: create(MediaInspectionReceiptSchema, {
        ...videoMedia.inspectionReceipt!, videoWidth: 1206, videoHeight: 2400, sourceEndPts: 240n,
      }),
    });
    const timing = create(VideoTimingSchema, {
      sourceMediaId: "video-source", sourceSha256: "e".repeat(64), sourceSizeBytes: 900n,
      inspectionJobId: "inspection-job", inspectionAttemptId: "inspection-attempt", inspectionReceiptId: "inspection-receipt",
      inspectionReceiptSha256: "f".repeat(64), clockArtifactId: "clock-artifact", clockSha256: "a".repeat(64), clockSizeBytes: 256n,
      selectedVideoStream: 0, timeBaseNum: 1n, timeBaseDen: 24n, streamStartPts: 0n, sourceEndPts: 240n,
      lastFrameDurationTicks: 1n, trimStartPts: 0n, trimEndPts: 240n, playbackRateNum: 1, playbackRateDen: 1,
      outputRateNum: 24, outputRateDen: 1, audioPolicy: VideoAudioPolicy.SILENT,
      videoCompositionProfileId: "device-studio-fixed-plate-video-v1", videoCompositionProfileVersion: 1,
    });
    const binding = create(PhysicalShotBindingSchema, {
      device: create(ComponentRevisionRefSchema, { id: "device-phone", revision: 1, sha256: "b".repeat(64) }),
      scene: create(ComponentRevisionRefSchema, { id: "scene-phone", revision: 1, sha256: "c".repeat(64) }),
      motion: create(ComponentRevisionRefSchema, { id: "motion-still", revision: 1, sha256: "d".repeat(64) }),
      backend: PhysicalRenderBackend.OPTIX, screenWidthPixels: 1206, screenHeightPixels: 2621,
      motionOutput: create(PhysicalMotionOutputBindingSchema, { profileId: "device-studio-physical-motion-1080p-v2", profileVersion: 2, width: 1920, height: 1080, outputRateNumerator: 24, outputRateDenominator: 1, frameCount: 240 }),
    });
    const snapshot = create(ShotSnapshotSchema, {
      motionMode: MotionMode.FIXED, physicalBinding: binding, videoTiming: timing,
      presentation: create(ScreenPresentationSchema, { sourceMediaId: "video-source", captureArea: CaptureArea.APP_ONLY, kind: PresentationKind.PHONE_SYSTEM, fitMode: ScreenFitMode.CONTAIN }),
    });
    const head = create(ShotRevisionSchema, { shotId: "physical-shot", revision: 2n, name: "Phone recording", snapshot, issues: [] });
    const tenSecondMotion = create(MotionCandidateSchema, {
      version: "motion-candidate/v1", motionId: "motion-still", timebaseTicksPerSecond: { num: 24n, den: 1n }, durationTicks: 240n,
      tracks: [{ target: { deviceRoot: true }, interpolation: "linear", keys: [
        { tick: 0n, transform: { translationM: {}, rotationXyzw: { w: 1 } } },
        { tick: 240n, transform: { translationM: {}, rotationXyzw: { w: 1 } } },
      ] }], requiredCapabilities: [],
    });
    vi.mocked(motionsClient.getMotionRevision).mockResolvedValue({ revision: { motionId: "motion-still", revision: 1, sha256: "d".repeat(64), candidate: tenSecondMotion } } as never);
    vi.mocked(projectsClient.listSourceMedia).mockResolvedValue({ sourceMedia: [appOnlySource], nextPageToken: "" } as never);
    vi.mocked(projectsClient.getSourceMediaClockPage).mockResolvedValue({ points: [{ ordinal: 0, pts: 0n }, { ordinal: 24, pts: 24n }, { ordinal: 48, pts: 48n }], nextPageToken: "", clockSha256: "a".repeat(64), timeBaseNumerator: 1n, timeBaseDenominator: 24n } as never);
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: create(ShotSchema, { id: "physical-shot", projectId: "project-1", name: head.name, headRevision: 2n }), head } as never);
    renderWithProviders(<PhysicalShotEditor projectId="project-1" shotId="physical-shot" />);
    await screen.findByTestId("physical-shot-editor");
    await user.selectOptions(screen.getByLabelText(i18n.t(strings.shot.videoOutputRate)), "30");
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.saveRevision) }));
    await waitFor(() => expect(shotsClient.saveShot).toHaveBeenCalled());
    const saved = vi.mocked(shotsClient.saveShot).mock.calls[0]?.[0].snapshot;
    expect(saved?.videoTiming?.trimEndPts).toBe(240n);
    expect(saved?.videoTiming?.outputRateNum).toBe(30);
    expect(saved?.physicalBinding?.motionOutput).toMatchObject({ profileId: "device-studio-physical-motion-1080p-v2", frameCount: 300, outputRateNumerator: 30 });
    expect(saved?.physicalBinding?.screenWidthPixels).toBe(1206);
    expect(saved?.physicalBinding?.screenHeightPixels).toBe(2621);
  });

  it("does not enable creation until source bytes pass verification", async () => {
    vi.mocked(verifySourceMediaContent).mockImplementation(() => new Promise(() => undefined));
    const user = userEvent.setup();
    renderWithProviders(<NewPhysicalShot projectId="project-1" onCreated={vi.fn()} onCancel={vi.fn()} />);
    await user.click(await screen.findByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.selectOptions(screen.getByLabelText(i18n.t(strings.physical.sourceSelect)), "screen-source");
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.continue) }));
    expect(screen.getByRole("button", { name: i18n.t(strings.physical.create) })).toBeDisabled();
    expect(screen.getByText(i18n.t(strings.physical.verifyingSource))).toBeInTheDocument();
    expect(shotsClient.createShot).not.toHaveBeenCalled();
  });

  it("preserves saved presentation settings when editing a physical shot", async () => {
    const user = userEvent.setup();
    const binding = create(PhysicalShotBindingSchema, { device: create(ComponentRevisionRefSchema, { id: "device-phone", revision: 1, sha256: "b".repeat(64) }), backend: PhysicalRenderBackend.OPTIX });
    const presentation = create(ScreenPresentationSchema, { sourceMediaId: "screen-source", captureArea: CaptureArea.ALREADY_COMPOSED, kind: PresentationKind.PLAIN, fitMode: ScreenFitMode.COVER, viewportWidthPixels: 111, viewportHeightPixels: 222 });
    const snapshot = create(ShotSnapshotSchema, { motionMode: MotionMode.FIXED, physicalBinding: binding, presentation });
    const head = create(ShotRevisionSchema, { shotId: "physical-shot", revision: 2n, name: "Phone", snapshot, issues: [] });
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: create(ShotSchema, { id: "physical-shot", projectId: "project-1", name: "Phone", headRevision: 2n }), head } as never);
    vi.mocked(shotsClient.saveShot).mockResolvedValue({ revision: create(ShotRevisionSchema, { ...head, revision: 3n }) } as never);
    renderWithProviders(<PhysicalShotEditor projectId="project-1" shotId="physical-shot" />);
    await screen.findByTestId("physical-shot-editor");
    await user.clear(screen.getByLabelText(i18n.t(strings.shot.nameLabel)));
    await user.type(screen.getByLabelText(i18n.t(strings.shot.nameLabel)), "Renamed");
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.saveRevision) }));
    await waitFor(() => expect(shotsClient.saveShot).toHaveBeenCalled());
    const saved = vi.mocked(shotsClient.saveShot).mock.calls[0]?.[0].snapshot?.presentation;
    expect(saved?.sourceMediaId).toBe("screen-source");
    expect(saved?.fitMode).toBe(ScreenFitMode.COVER);
    expect(saved?.viewportWidthPixels).toBe(111);
    expect(saved?.viewportHeightPixels).toBe(222);
  });

  it("saves edited physical placement, exact sample tick, backend, and still profile", async () => {
    const user = userEvent.setup();
    const binding = create(PhysicalShotBindingSchema, {
      device: create(ComponentRevisionRefSchema, { id: "device-phone", revision: 1, sha256: "b".repeat(64) }),
      scene: create(ComponentRevisionRefSchema, { id: "scene-phone", revision: 1, sha256: "c".repeat(64) }),
      motion: create(ComponentRevisionRefSchema, { id: "motion-still", revision: 1, sha256: "d".repeat(64) }),
      backend: PhysicalRenderBackend.OPTIX,
      motionSampleTick: { numerator: 0n, denominator: 1n },
      sceneSupportWorld: { translationM: { x: 0, y: 0.75, z: 0 }, rotationXyzw: { w: 1 }, scale: { x: 1, y: 1, z: 1 } },
      deviceRootRest: { translationM: {}, rotationXyzw: { w: 1 }, scale: { x: 1, y: 1, z: 1 } },
    });
    const snapshot = create(ShotSnapshotSchema, {
      motionMode: MotionMode.FIXED,
      physicalBinding: binding,
      presentation: create(ScreenPresentationSchema, { sourceMediaId: "screen-source", captureArea: CaptureArea.ALREADY_COMPOSED, kind: PresentationKind.PLAIN, fitMode: ScreenFitMode.CONTAIN }),
    });
    const head = create(ShotRevisionSchema, { shotId: "physical-shot", revision: 2n, name: "Phone", snapshot, issues: [] });
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: create(ShotSchema, { id: "physical-shot", projectId: "project-1", name: "Phone", headRevision: 2n }), head } as never);
    renderWithProviders(<PhysicalShotEditor projectId="project-1" shotId="physical-shot" />);
    await screen.findByTestId("physical-shot-editor");

    await user.selectOptions(screen.getByLabelText(i18n.t(strings.physical.backend)), "CUDA");
    await user.selectOptions(screen.getByLabelText(i18n.t(strings.physical.outputProfile)), "3840");
    await user.click(screen.getByText(i18n.t(strings.physical.advanced)));
    fireEvent.change(within(screen.getByRole("group", { name: i18n.t(strings.physical.translationMeters) })).getAllByRole("spinbutton")[1]!, { target: { value: "0.751" } });
    fireEvent.change(screen.getByLabelText(i18n.t(strings.physical.tickNumerator)), { target: { value: "3" } });
    fireEvent.change(screen.getByLabelText(i18n.t(strings.physical.tickDenominator)), { target: { value: "2" } });
    const rotation = within(screen.getByRole("group", { name: i18n.t(strings.physical.deviceRotation) }));
    fireEvent.change(rotation.getByRole("spinbutton", { name: "x" }), { target: { value: "0.1" } });
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.saveRevision) }));

    await waitFor(() => expect(shotsClient.saveShot).toHaveBeenCalled());
    const saved = vi.mocked(shotsClient.saveShot).mock.calls[0]?.[0].snapshot?.physicalBinding;
    expect(saved?.backend).toBe(PhysicalRenderBackend.CUDA);
    expect(saved?.outputProfile).toBe(PhysicalStillOutputProfile.PHYSICAL_STILL_OUTPUT_PROFILE_3840X2160_V1);
    expect(saved?.sceneSupportWorld?.translationM?.y).toBe(0.751);
    expect(saved?.motionSampleTick).toMatchObject({ numerator: 3n, denominator: 2n });
    expect(saved?.deviceRootRest?.rotationXyzw?.x).toBe(0.1);
  });

  it("shows a recoverable error when the saved shot cannot be loaded", async () => {
    vi.mocked(shotsClient.getShot).mockRejectedValue(new Error("service unavailable"));
    renderWithProviders(<PhysicalShotEditor projectId="project-1" shotId="physical-shot" />);
    expect(await screen.findByRole("alert")).toHaveTextContent("service unavailable");
    expect(screen.getByRole("button", { name: i18n.t(strings.media.retry) })).toBeInTheDocument();
  });

  it("does not remain in loading state when a saved revision has no physical binding", async () => {
    const snapshot = create(ShotSnapshotSchema, { motionMode: MotionMode.FIXED, presentation: create(ScreenPresentationSchema, { sourceMediaId: "screen-source", captureArea: CaptureArea.ALREADY_COMPOSED, kind: PresentationKind.PLAIN, fitMode: ScreenFitMode.CONTAIN }) });
    const head = create(ShotRevisionSchema, { shotId: "physical-shot", revision: 1n, name: "Incomplete", snapshot, issues: [] });
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: create(ShotSchema, { id: "physical-shot", projectId: "project-1", name: "Incomplete", headRevision: 1n }), head } as never);
    renderWithProviders(<PhysicalShotEditor projectId="project-1" shotId="physical-shot" />);
    expect(await screen.findByRole("alert")).toHaveTextContent(i18n.t(strings.physical.invalidBinding));
    expect(screen.queryByText(i18n.t(strings.shot.loading))).not.toBeInTheDocument();
  });

  it("shows compiler repair issues when a saved candidate is not render eligible", async () => {
    const user = userEvent.setup();
    const snapshot = create(ShotSnapshotSchema, { motionMode: MotionMode.FIXED, physicalBinding: create(PhysicalShotBindingSchema, { backend: PhysicalRenderBackend.OPTIX }), presentation: create(ScreenPresentationSchema, { sourceMediaId: "screen-source", captureArea: CaptureArea.ALREADY_COMPOSED, kind: PresentationKind.PLAIN, fitMode: ScreenFitMode.CONTAIN }) });
    const head = create(ShotRevisionSchema, { shotId: "physical-shot", revision: 1n, name: "Needs repair", snapshot, issues: [] });
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: create(ShotSchema, { id: "physical-shot", projectId: "project-1", name: "Needs repair", headRevision: 1n }), head } as never);
    vi.mocked(shotsClient.compileShot).mockResolvedValue({ compiled: create(CompiledShotSchema, { revision: 1n, renderEligible: false, issues: [create(ShotIssueSchema, { code: ShotIssueCode.MALFORMED_INPUT, fieldPath: "physical_binding.device", summary: "Device geometry is unavailable", repair: "Select a supported candidate.", blocksRender: true })] }) } as never);
    renderWithProviders(<PhysicalShotEditor projectId="project-1" shotId="physical-shot" />);
    await screen.findByTestId("physical-shot-editor");
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.compile) }));
    expect(await screen.findByRole("alert")).toHaveTextContent("physical_binding.device: Device geometry is unavailable Select a supported candidate.");
    expect(screen.queryByText(i18n.t(strings.physical.compileReady))).not.toBeInTheDocument();
  });

  it("keeps the draft and offers reload or reapply when another editor advances the head", async () => {
    const user = userEvent.setup();
    const binding = create(PhysicalShotBindingSchema, { device: create(ComponentRevisionRefSchema, { id: "device-phone", revision: 1, sha256: "b".repeat(64) }), backend: PhysicalRenderBackend.OPTIX });
    const snapshot = create(ShotSnapshotSchema, { motionMode: MotionMode.FIXED, physicalBinding: binding, presentation: create(ScreenPresentationSchema, { sourceMediaId: "screen-source", captureArea: CaptureArea.ALREADY_COMPOSED, kind: PresentationKind.PLAIN, fitMode: ScreenFitMode.CONTAIN }) });
    const original = create(ShotRevisionSchema, { shotId: "physical-shot", revision: 2n, name: "Phone", snapshot, issues: [] });
    const latest = create(ShotRevisionSchema, { ...original, revision: 3n, name: "Phone from another editor" });
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: create(ShotSchema, { id: "physical-shot", projectId: "project-1", name: "Phone", headRevision: 2n }), head: original } as never);
    const rendered = renderWithProviders(<PhysicalShotEditor projectId="project-1" shotId="physical-shot" />);
    await screen.findByTestId("physical-shot-editor");
    const nameField = screen.getByLabelText(i18n.t(strings.shot.nameLabel));
    await user.clear(nameField);
    await user.type(nameField, "My retained draft");
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: create(ShotSchema, { id: "physical-shot", projectId: "project-1", name: latest.name, headRevision: 3n }), head: latest } as never);
    rendered.queryClient.setQueryData(["shot", "physical-shot"], { shot: create(ShotSchema, { id: "physical-shot", projectId: "project-1", name: latest.name, headRevision: 3n }), head: latest });
    expect(await screen.findByText(i18n.t(strings.physical.staleRevision))).toBeInTheDocument();
    expect(screen.getByRole("button", { name: i18n.t(strings.physical.reloadLatest) })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.reapplyDraft) }));
    await waitFor(() => expect(screen.getByLabelText(i18n.t(strings.shot.nameLabel))).toHaveValue("My retained draft"));
    expect(screen.queryByText(i18n.t(strings.physical.staleRevision))).not.toBeInTheDocument();
  });

  it("uses sequential mobile steps instead of stacking the whole editor", async () => {
    const mediaQuery = vi.spyOn(window, "matchMedia").mockImplementation((query) => ({
      matches: query === "(max-width: 800px)", media: query, onchange: null,
      addEventListener: vi.fn(), removeEventListener: vi.fn(), addListener: vi.fn(), removeListener: vi.fn(), dispatchEvent: vi.fn(() => false),
    } as MediaQueryList));
    const binding = create(PhysicalShotBindingSchema, { device: create(ComponentRevisionRefSchema, { id: "device-phone", revision: 1, sha256: "b".repeat(64) }), backend: PhysicalRenderBackend.OPTIX });
    const snapshot = create(ShotSnapshotSchema, { motionMode: MotionMode.FIXED, physicalBinding: binding, presentation: create(ScreenPresentationSchema, { sourceMediaId: "screen-source", captureArea: CaptureArea.ALREADY_COMPOSED, kind: PresentationKind.PLAIN, fitMode: ScreenFitMode.CONTAIN }) });
    const head = create(ShotRevisionSchema, { shotId: "physical-shot", revision: 2n, name: "Phone", snapshot, issues: [] });
    vi.mocked(shotsClient.getShot).mockResolvedValue({ shot: create(ShotSchema, { id: "physical-shot", projectId: "project-1", name: "Phone", headRevision: 2n }), head } as never);
    const user = userEvent.setup();
    renderWithProviders(<PhysicalShotEditor projectId="project-1" shotId="physical-shot" />);
    await screen.findByTestId("physical-shot-editor");
    const stepNavigation = screen.getByRole("navigation", { name: i18n.t(strings.physical.steps) });
    expect(within(stepNavigation).getAllByRole("button")).toHaveLength(4);
    for (const label of [strings.physical.screen, strings.physical.export, strings.physical.advanced, strings.physical.compile]) {
      expect(within(stepNavigation).getByRole("button", { name: new RegExp(i18n.t(label)) })).toBeInTheDocument();
    }
    expect(screen.getByTestId("physical-shot-editor").querySelector(".physical-shot__stage")).toBeInTheDocument();
    expect(screen.getByLabelText(i18n.t(strings.physical.sourceSelect))).toBeInTheDocument();
    await user.click(within(stepNavigation).getByRole("button", { name: `2. ${i18n.t(strings.physical.export)}` }));
    expect(screen.getByLabelText(i18n.t(strings.physical.backend))).toBeInTheDocument();
    expect(screen.queryByLabelText(i18n.t(strings.physical.sourceSelect))).not.toBeInTheDocument();
    expect(screen.getByTestId("physical-shot-editor").querySelector(".physical-shot__stage")).toBeInTheDocument();
    await user.click(within(stepNavigation).getByRole("button", { name: `3. ${i18n.t(strings.physical.advanced)}` }));
    expect(screen.getByText(i18n.t(strings.physical.translationMeters))).toBeInTheDocument();
    expect(screen.getAllByRole("spinbutton").length).toBeGreaterThan(0);
    await user.click(within(stepNavigation).getByRole("button", { name: `4. ${i18n.t(strings.physical.compile)}` }));
    expect(screen.getByRole("button", { name: i18n.t(strings.physical.saveRevision) })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: i18n.t(strings.physical.compile) })).toBeEnabled();
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.compile) }));
    await waitFor(() => expect(shotsClient.compileShot).toHaveBeenCalledWith({ shotId: "physical-shot", revision: 2n }));
    expect(screen.getByTestId("physical-shot-editor").querySelector(".physical-shot__stage")).toBeInTheDocument();
    await user.click(within(stepNavigation).getByRole("button", { name: `1. ${i18n.t(strings.physical.screen)}` }));
    await user.clear(screen.getByLabelText(i18n.t(strings.shot.nameLabel)));
    await user.type(screen.getByLabelText(i18n.t(strings.shot.nameLabel)), "Updated phone name");
    await user.click(within(stepNavigation).getByRole("button", { name: `4. ${i18n.t(strings.physical.compile)}` }));
    expect(screen.getByRole("button", { name: i18n.t(strings.physical.saveRevision) })).toBeEnabled();
    await user.click(screen.getByRole("button", { name: i18n.t(strings.physical.saveRevision) }));
    await waitFor(() => expect(shotsClient.saveShot).toHaveBeenCalled());
    mediaQuery.mockRestore();
  });
});
