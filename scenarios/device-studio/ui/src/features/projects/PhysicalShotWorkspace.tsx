import { create } from "@bufbuild/protobuf";
import { useEffect, useMemo, useState } from "react";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import { useMediaQuery } from "@vrooli/react-component-library/useMediaQuery/1.1.1";
import { Card, CardContent, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";
import { MediaReadiness, SourceMediaKind, type SourceMedia } from "@vrooli/proto-types/device-studio/v1/projects/projects_pb";
import { CaptureArea, ComponentRevisionRefSchema, MotionMode, PhysicalMotionOutputBindingSchema, PhysicalMotionOutputProfile, PhysicalRenderBackend, PhysicalShotBindingSchema, PhysicalStillOutputProfile, PresentationAppearance, PresentationKind, RationalTimeSchema, ShotSnapshotSchema, VideoAudioPolicy, VideoEOFPolicy, VideoTimingSchema, type PhysicalShotBinding, type VideoTiming } from "@vrooli/proto-types/device-studio/v1/shots/shots_pb";
import { QuatXYZWSchema, TRSSchema, Vec3Schema, type ArtifactRef } from "@vrooli/proto-types/device-studio/v1/physical/physical_pb";
import type { MotionCandidate } from "@vrooli/proto-types/device-studio/v1/motions/motions_pb";
import { getBundleRevisionMetadata, initializePhysicalStarter, listDeviceCandidates, listMotionCandidates, listPhysicalShotPresets, listSceneCandidates, motionsClient, physicalStarterChoices, resolvePhysicalShotPreset } from "../../api/physical";
import { getSourceMediaContent, projectsClient, sourceMediaContentURL, verifySourceMediaContent } from "../../api/projects";
import { shotConflict, shotsClient } from "../../api/shots";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { RenderControls, type FinalRenderOutput } from "./RenderControls";
import { VideoTimingEditor } from "./VideoTimingEditor";
import { LibraryIntakePanel } from "./LibraryIntakePanel";
import { ScreenPresentationEditor, emptyScreenPresentationDraft, screenPresentationDraftFromProto, screenPresentationDraftInvalid, screenPresentationFromDraft, type ScreenPresentationDraft } from "./ScreenPresentationEditor";

const PAGE_SIZE = 100;
const PHYSICAL_PROFILE_ID = "device-studio-physical-cycles-v1";
const MOBILE_QUERY = "(max-width: 800px)";

function sourceIsReadyImage(media: SourceMedia): media is SourceMedia & { width: number; height: number } {
  return media.kind === SourceMediaKind.IMAGE && media.readiness === MediaReadiness.READY &&
    media.width !== undefined && media.height !== undefined && media.width > 0 && media.height > 0;
}

function sourceRaster(media: SourceMedia): { width: number; height: number } | undefined {
  if (sourceIsReadyImage(media)) return media;
  const receipt = media.inspectionReceipt;
  if (media.kind !== SourceMediaKind.VIDEO || media.readiness !== MediaReadiness.READY || receipt?.classification !== "accepted" ||
      receipt.sourceSha256 !== media.sha256 || receipt.sourceSizeBytes !== media.sizeBytes ||
      !receipt.videoWidth || !receipt.videoHeight || receipt.videoWidth <= 0 || receipt.videoHeight <= 0) return undefined;
  return { width: receipt.videoWidth, height: receipt.videoHeight };
}

function videoTimingFromMedia(source: SourceMedia): VideoTiming | undefined {
  const receipt = source.inspectionReceipt;
  if (source.kind !== SourceMediaKind.VIDEO || source.readiness !== MediaReadiness.READY || !receipt || receipt.classification !== "accepted" ||
      receipt.sourceSha256 !== source.sha256 || receipt.sourceSizeBytes !== source.sizeBytes ||
      receipt.timeBaseNumerator === undefined || receipt.timeBaseDenominator === undefined || receipt.streamStartPts === undefined || receipt.sourceEndPts === undefined ||
      receipt.lastFrameDurationTicks === undefined || receipt.lastFrameDurationTicks <= 0n || receipt.videoStreamIndex === undefined || !receipt.clockSha256 || !receipt.clockArtifactRef || receipt.clockArtifactSizeBytes <= 0n) return undefined;
  return create(VideoTimingSchema, {
    sourceMediaId: source.id, sourceSha256: source.sha256, sourceSizeBytes: source.sizeBytes,
    inspectionJobId: receipt.jobId, inspectionAttemptId: receipt.attemptId, inspectionReceiptId: receipt.id,
    inspectionReceiptSha256: receipt.sha256, clockArtifactId: receipt.clockArtifactRef, clockSha256: receipt.clockSha256,
    clockSizeBytes: receipt.clockArtifactSizeBytes, selectedVideoStream: receipt.videoStreamIndex,
    timeBaseNum: receipt.timeBaseNumerator, timeBaseDen: receipt.timeBaseDenominator,
    streamStartPts: receipt.streamStartPts, sourceEndPts: receipt.sourceEndPts,
    lastFrameDurationTicks: receipt.lastFrameDurationTicks, trimStartPts: receipt.streamStartPts,
    trimEndPts: receipt.sourceEndPts, playbackRateNum: 1, playbackRateDen: 1,
    outputRateNum: 24, outputRateDen: 1, eofPolicy: VideoEOFPolicy.VIDEO_EOF_POLICY_REJECT,
    audioPolicy: VideoAudioPolicy.SILENT,
    videoCompositionProfileId: "device-studio-fixed-plate-video-v1", videoCompositionProfileVersion: 1,
  });
}

function plannedFrames(timing?: VideoTiming): number | undefined {
  if (!timing || timing.trimEndPts <= timing.trimStartPts || timing.timeBaseNum <= 0n || timing.timeBaseDen <= 0n || timing.playbackRateNum <= 0 || timing.playbackRateDen <= 0 || timing.outputRateNum <= 0 || timing.outputRateDen <= 0) return undefined;
  const numerator = (timing.trimEndPts - timing.trimStartPts) * timing.timeBaseNum * BigInt(timing.playbackRateDen) * BigInt(timing.outputRateNum);
  const denominator = timing.timeBaseDen * BigInt(timing.playbackRateNum) * BigInt(timing.outputRateDen);
  const frames = (numerator + denominator - 1n) / denominator;
  return frames <= BigInt(Number.MAX_SAFE_INTEGER) ? Number(frames) : undefined;
}

function derivedRenderHeight(sourceWidth: number, metric?: { widthMm?: number; heightMm?: number }) {
  if (!metric || !Number.isFinite(metric.widthMm) || !Number.isFinite(metric.heightMm) || !metric.widthMm || !metric.heightMm || sourceWidth <= 0) return undefined;
  const height = Math.round(sourceWidth * metric.heightMm / metric.widthMm);
  return Number.isSafeInteger(height) && height > 0 && height <= 4096 && sourceWidth <= 4096 ? height : undefined;
}

function motionFrames(candidate: MotionCandidate, rate: number) {
  const base = candidate.timebaseTicksPerSecond;
  if (!base || base.num <= 0n || base.den <= 0n || typeof candidate.durationTicks !== "bigint" || candidate.durationTicks <= 0n) return undefined;
  if (candidate.durationTicks * base.den > 10n * base.num) return undefined;
  const numerator = candidate.durationTicks * base.den * BigInt(rate);
  const denominator = base.num;
  const frames = (numerator + denominator - 1n) / denominator;
  return frames <= 300n ? Number(frames) : undefined;
}

function durationMatches(candidate: MotionCandidate, timing: VideoTiming) {
  const base = candidate.timebaseTicksPerSecond;
  if (!base || typeof candidate.durationTicks !== "bigint" || timing.trimEndPts <= timing.trimStartPts) return false;
  return candidate.durationTicks * base.den * BigInt(timing.timeBaseDen) * BigInt(timing.playbackRateNum) ===
    (timing.trimEndPts - timing.trimStartPts) * timing.timeBaseNum * BigInt(timing.playbackRateDen) * base.num;
}

function physicalBindingFromPreset(preset: Awaited<ReturnType<typeof resolvePhysicalShotPreset>>, media: SourceMedia, presentation: ScreenPresentationDraft, backend: PhysicalRenderBackend, motionRef: { id: string; revision: number; sha256: string }, candidate: MotionCandidate, mode: "still" | "motion", timing?: VideoTiming, motionProfile?: PhysicalMotionOutputProfile, frames?: number): PhysicalShotBinding {
  const setup = preset.preset.setup;
  const raster = sourceRaster(media);
  if (!setup || !setup.device || !setup.scene || !setup.motion || !setup.sceneSupportWorld || !setup.supportPivotM || !setup.deviceRootRest || !raster) {
    throw new Error("The selected saved setup is incomplete or the source raster is unavailable.");
  }
  const display = preset.device.candidate?.definition?.displays.find((item) => item.id === setup.displayId);
  const deviceSupport = preset.device.candidate?.supportFrames.find((item) => item.id === setup.deviceSupportId);
  const sceneSupport = preset.scene.candidate?.supportFrames.find((item) => item.id === setup.sceneSupportId);
  const camera = preset.scene.candidate?.cameraFrames.find((item) => item.id === setup.cameraId);
  if (!display || !deviceSupport || !sceneSupport || !camera) throw new Error("The selected saved setup no longer resolves to its pinned display, support, and camera frames.");
  const recording = media.kind === SourceMediaKind.VIDEO;
  const renderHeight = presentation.captureArea === CaptureArea.APP_ONLY ? derivedRenderHeight(raster.width, display.coordinateMetric) : raster.height;
  if (!renderHeight) throw new Error("The selected setup does not provide a usable display aspect for this app-only source.");
  if (recording && (!timing || !motionProfile || frames === undefined || frames < 1 || frames !== plannedFrames(timing) || !durationMatches(candidate, timing) || candidate.tracks.some((track) => track.target?.jointId))) throw new Error("Choose a source-clock-bounded trim matching a supported root/camera motion.");
  if (mode === "motion" && (!motionProfile || frames === undefined || frames < 1)) throw new Error("The saved setup does not list a compatible motion output profile.");
  const stillProfile = setup.stillProfiles.find((value) => value === PhysicalStillOutputProfile.PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1 || value === PhysicalStillOutputProfile.PHYSICAL_STILL_OUTPUT_PROFILE_3840X2160_V1);
  if (mode === "still" && !stillProfile) throw new Error("The saved setup has no supported still output profile.");
  if (mode === "motion" && !motionProfile) throw new Error("The saved setup has no compatible motion output profile.");
  return create(PhysicalShotBindingSchema, {
    device: create(ComponentRevisionRefSchema, setup.device),
    scene: create(ComponentRevisionRefSchema, setup.scene),
    motion: create(ComponentRevisionRefSchema, motionRef),
    displayId: setup.displayId,
    deviceSupportId: setup.deviceSupportId,
    sceneSupportId: setup.sceneSupportId,
    cameraId: setup.cameraId,
    sceneSupportWorld: setup.sceneSupportWorld,
    supportPivotM: setup.supportPivotM,
    deviceRootRest: setup.deviceRootRest,
    screenWidthPixels: raster.width,
    screenHeightPixels: renderHeight,
    rendererProfileId: PHYSICAL_PROFILE_ID,
    rendererProfileVersion: 1,
    outputProfile: mode === "motion" ? PhysicalStillOutputProfile.PHYSICAL_STILL_OUTPUT_PROFILE_UNSPECIFIED : stillProfile,
    motionSampleTick: mode === "still" ? create(RationalTimeSchema, { numerator: 0n, denominator: 1n }) : undefined,
    motionOutput: mode === "motion" ? create(PhysicalMotionOutputBindingSchema, { profileId: motionProfile === PhysicalMotionOutputProfile.PHYSICAL_MOTION_OUTPUT_PROFILE_1080P_V2 ? "device-studio-physical-motion-1080p-v2" : "device-studio-physical-motion-1080p-v1", profileVersion: motionProfile === PhysicalMotionOutputProfile.PHYSICAL_MOTION_OUTPUT_PROFILE_1080P_V2 ? 2 : 1, width: 1920, height: 1080, outputRateNumerator: timing?.outputRateNum ?? 24, outputRateDenominator: timing?.outputRateDen ?? 1, frameCount: frames ?? 0 }) : undefined,
    backend,
  });
}

function makeSnapshot(binding: PhysicalShotBinding, sourceMediaId: string, presentation: ScreenPresentationDraft, videoTiming?: VideoTiming) {
  return create(ShotSnapshotSchema, {
    motionMode: MotionMode.FIXED,
    physicalBinding: binding,
    presentation: screenPresentationFromDraft(sourceMediaId, presentation),
    videoTiming,
  });
}

function presentationForSource(source?: SourceMedia): ScreenPresentationDraft {
  const draft = emptyScreenPresentationDraft();
  if (source?.kind !== SourceMediaKind.VIDEO) {
    draft.captureArea = CaptureArea.ALREADY_COMPOSED;
    return draft;
  }
  draft.captureArea = CaptureArea.APP_ONLY;
  draft.kind = PresentationKind.PHONE_SYSTEM;
  draft.phoneAppearance = PresentationAppearance.DARK;
  draft.topFill = "#181a1d";
  draft.bottomFill = "#181a1d";
  draft.statusInk = "#f5f5f5";
  draft.homeInk = "#f5f5f5";
  return draft;
}

function sameJson(left: unknown, right: unknown) {
  return JSON.stringify(left, (_key, value: unknown) => typeof value === "bigint" ? value.toString() : value) ===
    JSON.stringify(right, (_key, value: unknown) => typeof value === "bigint" ? value.toString() : value);
}

function revisionKey(ref?: { id: string; revision: number; sha256: string }) {
  return ref ? JSON.stringify([ref.id, ref.revision, ref.sha256]) : "";
}

function BundleReviewSummary({ reference }: { reference?: ArtifactRef }) {
  const { t } = useTranslation();
  const metadata = useQuery({
    queryKey: ["physical-library-bundle", reference?.bundleId, reference?.bundleRevision],
    enabled: Boolean(reference?.bundleId && reference.bundleRevision > 0),
    queryFn: () => reference ? getBundleRevisionMetadata(reference.bundleId, reference.bundleRevision) : Promise.reject(new Error("No pinned artifact reference.")),
    retry: false,
  });
  if (!reference) return null;
  const revision = metadata.data;
  const artifact = revision?.manifest?.artifacts.find((item) => item.id === reference.artifactId);
  const identityMatches = Boolean(revision && artifact && revision.manifestSha256 === reference.manifestSha256 &&
    revision.reviewSha256 === reference.reviewSha256 && artifact.role === reference.role &&
    artifact.sha256 === reference.sha256 && artifact.sizeBytes === reference.sizeBytes);
  const renderReviewed = Boolean(identityMatches && revision?.review?.decision === "approved" &&
    revision.review.manifestSha256 === reference.manifestSha256 && revision.review.allowedUses.includes("render") &&
    artifact?.rights.includes("render"));
  return <p className="physical-shot__catalog-review" data-testid="physical-library-review">
    {metadata.isPending ? t(strings.physical.loadingRights) : metadata.error ? t(strings.physical.rightsUnavailable) : renderReviewed ? t(strings.physical.rightsReviewed) : t(strings.physical.rightsUnconfirmed)}
    {revision?.manifest && <span>{t(strings.physical.bundleReviewDetails, { bundleId: revision.manifest.bundleId, revision: revision.manifest.revision, artifactId: reference.artifactId, digest: reference.sha256.slice(0, 12), license: artifact?.licenseId || t(strings.shot.unspecified), rights: artifact?.rights.join(", ") || t(strings.shot.unspecified), uses: revision.review?.allowedUses.join(", ") || t(strings.shot.unspecified) })}</span>}
  </p>;
}

export function NewPhysicalShot({ projectId, onCreated, onCancel }: { projectId: string; onCreated: (shotId: string) => void; onCancel: () => void }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [name, setName] = useState("Physical setup");
  const [sourceId, setSourceId] = useState("");
  const [videoTiming, setVideoTiming] = useState<VideoTiming>();
  const [presentationDraft, setPresentationDraft] = useState<ScreenPresentationDraft>(() => presentationForSource());
  const [backend, setBackend] = useState(PhysicalRenderBackend.OPTIX);
  const [step, setStep] = useState(0);
  const [deviceKey, setDeviceKey] = useState("");
  const [sceneKey, setSceneKey] = useState("");
  const [motionKey, setMotionKey] = useState("");
  const [sourceMode, setSourceMode] = useState<"still" | "motion">("still");
  const [presetID, setPresetID] = useState("");
  const [starterID, setStarterID] = useState<string>(physicalStarterChoices[0].id);
  const mobile = useMediaQuery(MOBILE_QUERY);
  const presets = useQuery({ queryKey: ["physical-shot-presets"], queryFn: () => listPhysicalShotPresets() });
  const presetHeads = useMemo(() => presets.data?.presets ?? [], [presets.data]);
  const initializeStarter = useMutation({
    mutationFn: () => initializePhysicalStarter(starterID),
    onSuccess: async ({ preset }) => {
      await queryClient.invalidateQueries({ queryKey: ["physical-shot-presets"] });
      const refreshed = await presets.refetch();
      if (!refreshed.data?.presets.some((head) => head.presetId === preset.id)) throw new Error("The initialized candidate preset is not visible in the saved setup catalog.");
      setPresetID(preset.id);
      setDeviceKey(""); setSceneKey(""); setMotionKey(""); setStep(0);
    },
  });
  useEffect(() => {
    const first = presetHeads[0];
    if (!presetID && first) setPresetID(first.presetId);
  }, [presetID, presetHeads]);
  const selectedPresetHead = presetHeads.find((item) => item.presetId === presetID);
  const selectedPreset = useQuery({
    queryKey: ["physical-shot-preset", selectedPresetHead?.presetId, selectedPresetHead?.revision],
    enabled: Boolean(selectedPresetHead),
    queryFn: () => selectedPresetHead ? resolvePhysicalShotPreset(selectedPresetHead.presetId, selectedPresetHead.revision) : Promise.reject(new Error("No physical setup selected.")),
    retry: false,
  });
  const devices = useInfiniteQuery({ queryKey: ["physical-device-candidates"], initialPageParam: "", queryFn: ({ pageParam }) => listDeviceCandidates(pageParam), getNextPageParam: (page) => page.nextAfterDeviceId || undefined });
  const scenes = useInfiniteQuery({ queryKey: ["physical-scene-candidates"], initialPageParam: "", queryFn: ({ pageParam }) => listSceneCandidates(pageParam), getNextPageParam: (page) => page.nextAfterSceneId || undefined });
  const motions = useInfiniteQuery({ queryKey: ["physical-motion-candidates"], initialPageParam: "", queryFn: ({ pageParam }) => listMotionCandidates(pageParam), getNextPageParam: (page) => page.nextAfterMotionId || undefined });
  useEffect(() => {
    const resolved = selectedPreset.data;
    const setup = resolved?.preset.setup;
    if (!resolved || !setup?.device || !setup.scene || !setup.motion) return;
    setDeviceKey(revisionKey(setup.device));
    setSceneKey(revisionKey(setup.scene));
    setMotionKey(revisionKey(setup.motion));
    setName(`${resolved.preset.name} · physical`);
  }, [selectedPreset.data]);
  const media = useInfiniteQuery({
    queryKey: ["project-source-media", projectId], initialPageParam: "",
    queryFn: ({ pageParam }) => projectsClient.listSourceMedia({ projectId, pageSize: PAGE_SIZE, pageToken: pageParam }),
    getNextPageParam: (page) => page.nextPageToken || undefined,
  });
  const sources = useMemo(() => media.data?.pages.flatMap((page) => page.sourceMedia).filter((item) => Boolean(sourceRaster(item))) ?? [], [media.data]);
  const placementReady = Boolean(selectedPreset.data?.preset.candidateOnly && selectedPreset.data.preset.setup);
  const deviceRows = useMemo(() => (devices.data?.pages.flatMap((page) => page.candidates) ?? []).map((row) => ({ ref: { id: row.deviceId, revision: row.revision, sha256: row.sha256 }, state: row.state, name: row.candidate?.definition?.model || row.deviceId, candidate: row.candidate })), [devices.data]);
  const sceneRows = useMemo(() => (scenes.data?.pages.flatMap((page) => page.candidates) ?? []).map((row) => ({ ref: { id: row.sceneId, revision: row.revision, sha256: row.sha256 }, state: row.state, name: row.candidate?.sceneId || row.sceneId, candidate: row.candidate })), [scenes.data]);
  const motionRows = useMemo(() => (motions.data?.pages.flatMap((page) => page.candidates) ?? []).map((row) => ({ ref: { id: row.motionId, revision: row.revision, sha256: row.sha256 }, state: row.state, name: row.candidate?.motionId || row.motionId, candidate: row.candidate })), [motions.data]);
  const deviceRowsWithStarter = useMemo(() => selectedPreset.data?.device.candidate ? [...deviceRows.filter((row) => revisionKey(row.ref) !== revisionKey({ id: selectedPreset.data.device.deviceId, revision: selectedPreset.data.device.revision, sha256: selectedPreset.data.device.sha256 })), { ref: { id: selectedPreset.data.device.deviceId, revision: selectedPreset.data.device.revision, sha256: selectedPreset.data.device.sha256 }, state: selectedPreset.data.device.state, name: selectedPreset.data.device.candidate.definition?.model || selectedPreset.data.device.deviceId, candidate: selectedPreset.data.device.candidate }] : deviceRows, [deviceRows, selectedPreset.data]);
  const sceneRowsWithStarter = useMemo(() => selectedPreset.data?.scene.candidate ? [...sceneRows.filter((row) => revisionKey(row.ref) !== revisionKey({ id: selectedPreset.data.scene.sceneId, revision: selectedPreset.data.scene.revision, sha256: selectedPreset.data.scene.sha256 })), { ref: { id: selectedPreset.data.scene.sceneId, revision: selectedPreset.data.scene.revision, sha256: selectedPreset.data.scene.sha256 }, state: selectedPreset.data.scene.state, name: selectedPreset.data.scene.candidate.sceneId || selectedPreset.data.scene.sceneId, candidate: selectedPreset.data.scene.candidate }] : sceneRows, [sceneRows, selectedPreset.data]);
  const motionRowsWithStarter = useMemo(() => selectedPreset.data?.motion.candidate ? [...motionRows.filter((row) => revisionKey(row.ref) !== revisionKey({ id: selectedPreset.data.motion.motionId, revision: selectedPreset.data.motion.revision, sha256: selectedPreset.data.motion.sha256 })), { ref: { id: selectedPreset.data.motion.motionId, revision: selectedPreset.data.motion.revision, sha256: selectedPreset.data.motion.sha256 }, state: selectedPreset.data.motion.state, name: selectedPreset.data.motion.candidate.motionId || selectedPreset.data.motion.motionId, candidate: selectedPreset.data.motion.candidate }] : motionRows, [motionRows, selectedPreset.data]);
  const selectedDevice = deviceRowsWithStarter.find((row) => revisionKey(row.ref) === deviceKey);
  const selectedScene = sceneRowsWithStarter.find((row) => revisionKey(row.ref) === sceneKey);
  const selectedMotion = motionRowsWithStarter.find((row) => revisionKey(row.ref) === motionKey);
  const setup = selectedPreset.data?.preset.setup;
  const reviewedTuple = Boolean(setup?.device && setup.scene && deviceKey === revisionKey(setup.device) && sceneKey === revisionKey(setup.scene));
  const placementMismatch = Boolean(deviceKey && sceneKey && !reviewedTuple);
  const selected = sources.find((item) => item.id === sourceId);
  const selectedRaster = selected ? sourceRaster(selected) : undefined;
  const selectedIsVideo = selected?.kind === SourceMediaKind.VIDEO;
  const activeMode = selectedIsVideo ? "motion" : sourceMode;
  const selectedMotionCandidate = selectedMotion?.candidate;
  const motionTracks = selectedMotionCandidate?.tracks ?? [];
  const motionTargetsFit = Boolean(selectedMotionCandidate && motionTracks.length > 0 && motionTracks.every((track) => {
    const target = track.target;
    return Boolean(target && (target.deviceRoot || (target.jointId && selectedDevice?.candidate?.definition?.joints.some((joint) => joint.id === target.jointId)) || (target.sceneCameraId && selectedScene?.candidate?.cameraFrames.some((frame) => frame.id === target.sceneCameraId))));
  }));
  const videoMotionSupported = Boolean(selectedMotionCandidate && motionTracks.length > 0 && motionTracks.every((track) => track.target?.deviceRoot || Boolean(track.target?.sceneCameraId)));
  const requestedRate = selectedIsVideo ? videoTiming?.outputRateNum ?? 24 : 24;
  const plannedMotionFrames = selectedMotionCandidate ? motionFrames(selectedMotionCandidate, requestedRate) : undefined;
  const motionProfile = setup?.motionProfiles.find((profile) => {
    const max = profile === PhysicalMotionOutputProfile.PHYSICAL_MOTION_OUTPUT_PROFILE_1080P_V1 ? 48 : profile === PhysicalMotionOutputProfile.PHYSICAL_MOTION_OUTPUT_PROFILE_1080P_V2 ? 300 : 0;
    const frames = selectedIsVideo ? plannedFrames(videoTiming) : plannedMotionFrames;
    return max > 0 && frames !== undefined && frames > 0 && frames <= max;
  });
  const motionChoiceCompatible = Boolean(selectedMotionCandidate && motionTargetsFit &&
    (activeMode === "still" || (motionProfile && plannedMotionFrames && (!selectedIsVideo || (videoMotionSupported && videoTiming && plannedMotionFrames === plannedFrames(videoTiming) && durationMatches(selectedMotionCandidate, videoTiming))))));
  const outputFrames = plannedFrames(videoTiming);
  const placement = setup;
  const compatible = selected && selectedPreset.data?.device.candidate?.definition?.displays.find((item) => item.id === placement?.displayId)?.coordinateMetric;
  const derivedAppHeight = selectedRaster && presentationDraft.captureArea === CaptureArea.APP_ONLY ? derivedRenderHeight(selectedRaster.width, compatible) : selectedRaster?.height;
  const aspectValid = Boolean(selectedRaster && derivedAppHeight !== undefined);
  const presentationInvalid = screenPresentationDraftInvalid(presentationDraft);
  const selectedBytes = useQuery({ queryKey: ["physical-create-source", projectId, selected?.id, selected?.sha256], enabled: Boolean(selected && !selectedIsVideo), queryFn: async () => { if (!selected) throw new Error("No screen source selected"); return verifySourceMediaContent(await getSourceMediaContent(projectId, selected.id), selected); }, retry: false, gcTime: 0 });
  const [selectedUrl, setSelectedUrl] = useState("");
  useEffect(() => {
    if (!selectedBytes.data) { setSelectedUrl(""); return; }
    const url = URL.createObjectURL(selectedBytes.data);
    setSelectedUrl(url);
    return () => URL.revokeObjectURL(url);
  }, [selectedBytes.data]);
  const createShot = useMutation({
    mutationFn: async () => {
      if (!selectedPreset.data || !placementReady || !reviewedTuple || !selected || !selectedMotion || !selectedMotionCandidate || !motionChoiceCompatible || !aspectValid || (selectedIsVideo ? !videoTiming || !durationMatches(selectedMotionCandidate, videoTiming) : activeMode === "still" && !selectedBytes.data)) throw new Error("Choose a verified source, compatible exact motion revision, and saved physical setup.");
      const binding = physicalBindingFromPreset(selectedPreset.data, selected, presentationDraft, backend, selectedMotion.ref, selectedMotionCandidate, activeMode, selectedIsVideo ? videoTiming : undefined, motionProfile, selectedIsVideo ? outputFrames : plannedMotionFrames);
      const result = await shotsClient.createShot({ projectId, name: name.trim(), snapshot: makeSnapshot(binding, selected.id, presentationDraft, selectedIsVideo ? videoTiming : undefined) });
      if (!result.shot) throw new Error("The server did not return the created shot.");
      return result.shot.id;
    },
    onSuccess: async (shotId) => {
      await queryClient.invalidateQueries({ queryKey: ["project-shots", projectId] });
      onCreated(shotId);
    },
  });
  const steps = [t(strings.physical.scene), t(strings.physical.device), t(strings.physical.screen), t(strings.physical.motion), t(strings.physical.export)];
  return <section className="physical-shot" data-testid={selectors.physical.create} aria-labelledby="physical-shot-title">
    <header className="physical-shot__header"><div><p className="eyebrow">{t(strings.shot.workspaceEyebrow)}</p><h2 id="physical-shot-title">{t(strings.physical.newTitle)}</h2></div><Button variant="outline" className="min-h-11" onClick={onCancel}>{t(strings.physical.cancel)}</Button></header>
    <nav className="physical-shot__steps" aria-label={t(strings.physical.steps)}>{steps.map((label, index) => <button key={label} type="button" aria-current={step === index ? "step" : undefined} onClick={() => setStep(index)}>{index + 1}. {label}</button>)}</nav>
    <div className="physical-shot__layout">
      <Card className="physical-shot__stage"><CardHeader><CardTitle as="h3">{t(strings.physical.sourceStage)}</CardTitle></CardHeader><CardContent>
        {selectedIsVideo ? <video muted controls playsInline preload="metadata" src={sourceMediaContentURL(projectId, selected.id)} aria-label={t(strings.physical.sourceAlt)} /> : selectedUrl ? <img src={selectedUrl} alt={t(strings.physical.sourceAlt)} /> : <div className="physical-shot__empty-stage"><strong>{t(strings.physical.noRender)}</strong><span>{t(strings.physical.noPreview)}</span></div>}
        {selected && <p className="physical-shot__source-facts">{selectedIsVideo ? `${selected.originalFilename} · ${selectedRaster?.width} × ${selectedRaster?.height} · ${t(strings.physical.verifiedSource)}` : selectedBytes.isSuccess ? `${selected.originalFilename} · ${selectedRaster?.width} × ${selectedRaster?.height} · ${selected.sha256.slice(0, 12)}… · ${t(strings.physical.verifiedSource)}` : selectedBytes.isPending ? t(strings.physical.verifyingSource) : t(strings.physical.sourceNotVerified)}</p>}
      </CardContent></Card>
      <Card><CardHeader><CardTitle as="h3">{steps[step]}</CardTitle></CardHeader><CardContent className="physical-shot__fields">
        {presets.isPending && <p role="status">{t(strings.physical.loadingStarter)}</p>}
        {step === 0 && <div className="physical-shot__starter-choice"><label>{t(strings.physical.initializeStarter)}<select className="physical-shot__input" value={starterID} onChange={(event) => setStarterID(event.target.value)} aria-label={t(strings.physical.initializeStarter)}>{physicalStarterChoices.map((choice) => <option key={choice.id} value={choice.id}>{choice.label}</option>)}</select></label><Button type="button" variant="outline" disabled={initializeStarter.isPending} onClick={() => initializeStarter.mutate()}>{initializeStarter.isPending ? t(strings.physical.initializingStarter) : t(strings.physical.initializeStarter)}</Button>{initializeStarter.error && <p role="alert">{initializeStarter.error.message}</p>}</div>}
        {presets.error && <div role="alert"><p>{t(strings.physical.catalogUnavailable)}</p><Button type="button" variant="outline" onClick={() => void presets.refetch()}>{t(strings.media.retry)}</Button></div>}
        {presets.data && presetHeads.length === 0 && <p role="status">{t(strings.physical.noCompatiblePreset)}</p>}
        {selectedPreset.isPending && selectedPresetHead && <p role="status">{t(strings.physical.loadingStarter)}</p>}
        {selectedPreset.error && <p role="alert">{t(strings.physical.candidatePlacementUnavailable)}</p>}
        {selectedPreset.data && <>
          <p className="physical-shot__candidate">{t(strings.physical.candidateOnly)}</p>
          {step === 0 && <><label>{t(strings.physical.selectSavedSetup)}<select className="physical-shot__input" value={presetID} onChange={(event) => { setPresetID(event.target.value); setDeviceKey(""); setSceneKey(""); setMotionKey(""); }} aria-label={t(strings.physical.selectSavedSetup)}>{presetHeads.map((preset) => <option key={preset.presetId} value={preset.presetId}>{preset.name} · {t(strings.physical.candidateState, { state: preset.state, revision: preset.revision })}</option>)}</select></label>{selectedPresetHead && <details><summary>{t(strings.physical.technicalRefs)}</summary><p>{selectedPresetHead.presetId} · {selectedPresetHead.revision} · {selectedPresetHead.sha256}</p></details>}<label>{t(strings.physical.selectSceneCandidate)}<select className="physical-shot__input" value={sceneKey} onChange={(event) => setSceneKey(event.target.value)}><option value="">{t(strings.physical.catalogLoading)}</option>{sceneRowsWithStarter.map((row) => <option key={revisionKey(row.ref)} value={revisionKey(row.ref)}>{row.name} · {t(strings.physical.candidateState, { state: row.state, revision: row.ref.revision })}</option>)}</select></label>{scenes.error && <p role="alert">{t(strings.physical.catalogUnavailable)}</p>}{scenes.hasNextPage && <Button type="button" variant="outline" className="min-h-11" disabled={scenes.isFetchingNextPage} onClick={() => void scenes.fetchNextPage()}>{t(strings.physical.loadMoreCandidates)}</Button>}{selectedScene && <><p className="physical-shot__candidate">{t(strings.physical.candidateOnly)} · {selectedScene.name}</p>{(selectedScene.candidate?.artifacts ?? []).slice(0, 8).map((item) => <BundleReviewSummary key={`${item.ref?.bundleId}:${item.ref?.bundleRevision}:${item.ref?.artifactId}`} reference={item.ref} />)}<details><summary>{t(strings.physical.technicalRefs)}</summary><p>{t(strings.physical.sceneChoice, { name: selectedScene.name, revision: selectedScene.ref.revision })}</p><p>{t(strings.physical.sceneSupport, { id: setup?.sceneSupportId ?? "—" })}</p></details></>}</>}
          {step === 1 && <><label>{t(strings.physical.selectDeviceCandidate)}<select className="physical-shot__input" value={deviceKey} onChange={(event) => setDeviceKey(event.target.value)}><option value="">{t(strings.physical.catalogLoading)}</option>{deviceRowsWithStarter.map((row) => <option key={revisionKey(row.ref)} value={revisionKey(row.ref)}>{row.name} · {t(strings.physical.candidateState, { state: row.state, revision: row.ref.revision })}</option>)}</select></label>{devices.error && <p role="alert">{t(strings.physical.catalogUnavailable)}</p>}{devices.hasNextPage && <Button type="button" variant="outline" className="min-h-11" disabled={devices.isFetchingNextPage} onClick={() => void devices.fetchNextPage()}>{t(strings.physical.loadMoreCandidates)}</Button>}{selectedDevice && <><p className="physical-shot__candidate">{t(strings.physical.candidateOnly)} · {selectedDevice.name}</p><BundleReviewSummary reference={selectedDevice.candidate?.geometry} /><details><summary>{t(strings.physical.technicalRefs)}</summary><p>{t(strings.physical.deviceChoice, { name: selectedDevice.name, revision: selectedDevice.ref.revision })}</p><p>{t(strings.physical.displayChoice, { id: setup?.displayId ?? "—" })}</p></details></>}<LibraryIntakePanel /></>}
          {step === 2 && <><label>{t(strings.physical.sourceSelect)}<select className="physical-shot__input" value={sourceId} onChange={(event) => { const next = sources.find((item) => item.id === event.target.value); setSourceId(event.target.value); setVideoTiming(next?.kind === SourceMediaKind.VIDEO ? videoTimingFromMedia(next) : undefined); setPresentationDraft(presentationForSource(next)); }}><option value="">{t(strings.physical.sourcePlaceholder)}</option>{sources.map((item) => { const raster = sourceRaster(item); return <option key={item.id} value={item.id}>{item.originalFilename} · {item.kind === SourceMediaKind.VIDEO ? "video" : "image"} · {raster?.width} × {raster?.height}</option>; })}</select></label>{!selectedIsVideo && <label>{t(strings.physical.motion)}<select className="physical-shot__input" value={sourceMode} onChange={(event) => setSourceMode(event.target.value === "motion" ? "motion" : "still")}><option value="still">{t(strings.physical.renderStill)}</option><option value="motion">{t(strings.physical.renderMotion)}</option></select></label>}{selectedIsVideo && <><p>{t(strings.physical.appOnlyCapture)}</p><p>{t(strings.physical.derivedRenderRaster)}</p></>}{selectedIsVideo && videoTiming && <VideoTimingEditor projectId={projectId} source={selected} timing={videoTiming} disabled={false} onChange={setVideoTiming} />}<details open><summary>{t(strings.shot.screenPresentation)}</summary><ScreenPresentationEditor value={presentationDraft} disabled={false} onChange={(patch) => setPresentationDraft((current) => ({ ...current, ...patch }))} /></details></>}
          {selectedBytes.error && <p role="alert">{errorMessage(selectedBytes.error, t)}</p>}
          {selectedIsVideo && (!videoTiming || !motionChoiceCompatible) && <p role="alert">{t(strings.physical.recordingRangeLimit)}</p>}
          {media.hasNextPage && <Button type="button" variant="outline" className="min-h-11" disabled={media.isFetchingNextPage} onClick={() => void media.fetchNextPage()}>{media.isFetchingNextPage ? t(strings.shot.renderLoadingJobs) : t(strings.media.loadMore)}</Button>}
          {step === 3 && <><label>{t(strings.physical.selectMotionCandidate)}<select className="physical-shot__input" value={motionKey} onChange={(event) => setMotionKey(event.target.value)}><option value="">{t(strings.physical.catalogLoading)}</option>{motionRowsWithStarter.map((row) => <option key={revisionKey(row.ref)} value={revisionKey(row.ref)}>{row.name} · {t(strings.physical.candidateState, { state: row.state, revision: row.ref.revision })}</option>)}</select></label>{motions.error && <p role="alert">{t(strings.physical.catalogUnavailable)}</p>}{motions.hasNextPage && <Button type="button" variant="outline" className="min-h-11" disabled={motions.isFetchingNextPage} onClick={() => void motions.fetchNextPage()}>{t(strings.physical.loadMoreCandidates)}</Button>}{selectedMotion && <><p className="physical-shot__candidate">{t(strings.physical.candidateOnly)} · {selectedMotion.name}</p><details><summary>{t(strings.physical.technicalRefs)}</summary><p>{t(strings.physical.motionChoice, { name: selectedMotion.name, revision: selectedMotion.ref.revision })}</p><p>{t(strings.physical.poseDescription)}</p></details></>}{selectedMotionCandidate && !motionChoiceCompatible && <p role="alert">{t(strings.physical.motionIncompatible)}</p>}{selectedMotionCandidate && motionChoiceCompatible && <p role="status">{selectedIsVideo ? `${plannedMotionFrames} frames at ${requestedRate} fps from the exact candidate duration and source-clock trim.` : activeMode === "motion" ? `${plannedMotionFrames} frames at 24 fps from the exact candidate duration.` : t(strings.physical.stillTick)}</p>}</>}
          {step === 4 && <><label>{t(strings.shot.nameLabel)}<input className="physical-shot__input" value={name} maxLength={120} onChange={(event) => setName(event.target.value)} /></label><label>{t(strings.physical.backend)}<select className="physical-shot__input" value={backend === PhysicalRenderBackend.OPTIX ? "OPTIX" : backend === PhysicalRenderBackend.CUDA ? "CUDA" : "CPU"} onChange={(event) => setBackend(event.target.value === "CUDA" ? PhysicalRenderBackend.CUDA : event.target.value === "CPU" ? PhysicalRenderBackend.CPU : PhysicalRenderBackend.OPTIX)}><option value="OPTIX">{t(strings.physical.backendOptix)}</option><option value="CUDA">{t(strings.physical.backendCuda)}</option><option value="CPU">{t(strings.physical.backendCpu)}</option></select></label><p>{t(strings.physical.noFallback)}</p></>}
          {media.error && <p role="alert">{errorMessage(media.error, t)}</p>}
          {selected && !aspectValid && <p role="alert">{t(strings.physical.aspectIssue)}</p>}
          {placementMismatch && <p role="alert" data-testid="physical-placement-unavailable">{t(strings.physical.candidatePlacementUnavailable)}</p>}
          {step === 4 && selectedMotionCandidate && !motionChoiceCompatible && <p role="alert">{t(strings.physical.motionIncompatible)}</p>}
          {createShot.error && <p role="alert">{errorMessage(createShot.error, t)}</p>}
        </>}
          <div className={`physical-shot__actions${mobile ? " physical-shot__mobile-nav" : ""}`}><Button variant="outline" className="min-h-11" disabled={step === 0} onClick={() => setStep((value) => Math.max(0, value - 1))}>{t(strings.physical.back)}</Button>{step < steps.length - 1 ? <Button className="studio-primary min-h-11" onClick={() => setStep((value) => Math.min(steps.length - 1, value + 1))}>{t(strings.physical.continue)}</Button> : <Button className="studio-primary min-h-11" disabled={!name.trim() || !placementReady || !reviewedTuple || !selectedPreset.data || !selected || !selectedMotion || !motionChoiceCompatible || !aspectValid || presentationInvalid || (activeMode === "still" && !selectedBytes.data) || createShot.isPending} onClick={() => createShot.mutate()}>{createShot.isPending ? t(strings.physical.creating) : t(strings.physical.create)}</Button>}</div>
      </CardContent></Card>
    </div>
  </section>;
}

function copyBinding(binding: PhysicalShotBinding) {
  return create(PhysicalShotBindingSchema, {
    ...binding,
    device: binding.device ? { ...binding.device } : undefined,
    scene: binding.scene ? { ...binding.scene } : undefined,
    motion: binding.motion ? { ...binding.motion } : undefined,
    sceneSupportWorld: binding.sceneSupportWorld ? create(TRSSchema, { translationM: binding.sceneSupportWorld.translationM ? create(Vec3Schema, { x: binding.sceneSupportWorld.translationM.x, y: binding.sceneSupportWorld.translationM.y, z: binding.sceneSupportWorld.translationM.z }) : undefined, rotationXyzw: binding.sceneSupportWorld.rotationXyzw, scale: binding.sceneSupportWorld.scale ? create(Vec3Schema, { x: binding.sceneSupportWorld.scale.x, y: binding.sceneSupportWorld.scale.y, z: binding.sceneSupportWorld.scale.z }) : undefined }) : undefined,
    supportPivotM: binding.supportPivotM ? create(Vec3Schema, binding.supportPivotM) : undefined,
    deviceRootRest: binding.deviceRootRest ? create(TRSSchema, { translationM: binding.deviceRootRest.translationM ? create(Vec3Schema, { x: binding.deviceRootRest.translationM.x, y: binding.deviceRootRest.translationM.y, z: binding.deviceRootRest.translationM.z }) : undefined, rotationXyzw: binding.deviceRootRest.rotationXyzw, scale: binding.deviceRootRest.scale ? create(Vec3Schema, { x: binding.deviceRootRest.scale.x, y: binding.deviceRootRest.scale.y, z: binding.deviceRootRest.scale.z }) : undefined }) : undefined,
    motionOutput: binding.motionOutput ? create(PhysicalMotionOutputBindingSchema, binding.motionOutput) : undefined,
    motionSampleTick: binding.motionSampleTick ? create(RationalTimeSchema, { numerator: binding.motionSampleTick.numerator, denominator: binding.motionSampleTick.denominator }) : undefined,
  });
}

export function PhysicalShotEditor({ projectId, shotId }: { projectId: string; shotId: string }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const mobile = useMediaQuery(MOBILE_QUERY);
  const shot = useQuery({ queryKey: ["shot", shotId], queryFn: () => shotsClient.getShot({ id: shotId }) });
  const media = useInfiniteQuery({ queryKey: ["project-source-media", projectId], initialPageParam: "", queryFn: ({ pageParam }) => projectsClient.listSourceMedia({ projectId, pageSize: PAGE_SIZE, pageToken: pageParam }), getNextPageParam: (page) => page.nextPageToken || undefined });
  const sources = media.data?.pages.flatMap((page) => page.sourceMedia).filter((item) => Boolean(sourceRaster(item))) ?? [];
  const head = shot.data?.head;
  const savedBinding = head?.snapshot?.physicalBinding;
  const [binding, setBinding] = useState<PhysicalShotBinding>();
  const [sourceId, setSourceId] = useState("");
  const [videoTiming, setVideoTiming] = useState<VideoTiming>();
  const [presentationDraft, setPresentationDraft] = useState<ScreenPresentationDraft>(() => emptyScreenPresentationDraft());
  const [name, setName] = useState("");
  const [baseRevision, setBaseRevision] = useState<bigint>();
  const [showOutput, setShowOutput] = useState(true);
  const [output, setOutput] = useState<FinalRenderOutput>();
  const [compiledRevision, setCompiledRevision] = useState<bigint>();
  const [compileIssues, setCompileIssues] = useState<Array<{ fieldPath: string; summary: string; repair: string }>>([]);
  const [mobileStep, setMobileStep] = useState(0);
  const motionRevision = useQuery({
    queryKey: ["physical-shot-motion", binding?.motion?.id, binding?.motion?.revision],
    enabled: Boolean(binding?.motion?.id && binding.motion.revision > 0),
    queryFn: async () => {
      if (!binding?.motion) throw new Error("The shot has no pinned motion revision.");
      const result = await motionsClient.getMotionRevision({ motionId: binding.motion.id, revision: binding.motion.revision });
      const revision = result.revision;
      if (!revision?.candidate || revision.motionId !== binding.motion.id || revision.revision !== binding.motion.revision || revision.sha256 !== binding.motion.sha256) throw new Error("The exact pinned motion candidate failed identity or digest verification.");
      return revision.candidate;
    }, retry: false,
  });
  useEffect(() => {
    if (!head?.snapshot?.physicalBinding || !head.snapshot.presentation || baseRevision !== undefined) return;
    setBinding(copyBinding(head.snapshot.physicalBinding));
    setSourceId(head.snapshot.presentation.sourceMediaId);
    setPresentationDraft(screenPresentationDraftFromProto(head.snapshot.presentation));
    setVideoTiming(head.snapshot.videoTiming ? create(VideoTimingSchema, head.snapshot.videoTiming) : undefined);
    setName(head.name);
    setBaseRevision(head.revision);
  }, [head?.revision, head?.snapshot, head?.name, baseRevision]);
  const source = sources.find((item) => item.id === sourceId);
  const sourceIsVideo = source?.kind === SourceMediaKind.VIDEO;
  const sourceSize = source ? sourceRaster(source) : undefined;
  const compatibleSources = binding ? sources.filter((item) => item.kind === source?.kind) : [];
  const sourceBytes = useQuery({ queryKey: ["physical-source-bytes", projectId, source?.id, source?.sha256], enabled: Boolean(source && !sourceIsVideo), queryFn: async () => { if (!source) throw new Error("No screen source selected"); return verifySourceMediaContent(await getSourceMediaContent(projectId, source.id), source); }, retry: false, gcTime: 0 });
  const sourceUrl = useMemo(() => sourceBytes.data ? URL.createObjectURL(sourceBytes.data) : "", [sourceBytes.data]);
  useEffect(() => () => { if (sourceUrl) URL.revokeObjectURL(sourceUrl); }, [sourceUrl]);
  const compiled = useMutation({ mutationFn: async (revision: bigint) => shotsClient.compileShot({ shotId, revision }), onSuccess: async (result) => {
    const revision = result.compiled;
    if (revision?.renderEligible) { setCompiledRevision(revision.revision); setCompileIssues([]); }
    else { setCompiledRevision(undefined); setCompileIssues(revision?.issues ?? []); }
    await queryClient.invalidateQueries({ queryKey: ["shot", shotId] });
  } });
  const save = useMutation({ mutationFn: async () => {
    if (!binding || baseRevision === undefined || !head?.snapshot?.presentation) throw new Error("The physical shot is not loaded.");
    const presentation = screenPresentationFromDraft(sourceId, presentationDraft, head.snapshot.presentation);
    const snapshot = Object.assign(create(ShotSnapshotSchema), head.snapshot, { physicalBinding: binding, presentation, videoTiming });
    return shotsClient.saveShot({ shotId, expectedRevision: baseRevision, name: name.trim(), snapshot });
  }, onSuccess: async (result) => {
    if (result.revision) { setBaseRevision(result.revision.revision); setBinding(result.revision.snapshot?.physicalBinding ? copyBinding(result.revision.snapshot.physicalBinding) : binding); setCompiledRevision(undefined); setCompileIssues([]); }
    await queryClient.invalidateQueries({ queryKey: ["shot", shotId] });
    await queryClient.invalidateQueries({ queryKey: ["shot-revisions", shotId] });
    await queryClient.invalidateQueries({ queryKey: ["project-shots", projectId] });
  } });
  if (shot.isLoading) return <p role="status">{t(strings.shot.loading)}</p>;
  if (shot.error) return <div role="alert"><p>{errorMessage(shot.error, t)}</p><Button variant="outline" onClick={() => void shot.refetch()}>{t(strings.media.retry)}</Button></div>;
  if (!head) return <div role="alert"><p>{t(strings.physical.shotUnavailable)}</p><Button variant="outline" onClick={() => void shot.refetch()}>{t(strings.media.retry)}</Button></div>;
  if (!head.snapshot?.physicalBinding || !head.snapshot.presentation) return <div role="alert"><p>{t(strings.physical.invalidBinding)}</p><Button variant="outline" onClick={() => void shot.refetch()}>{t(strings.media.retry)}</Button></div>;
  if (!binding || baseRevision === undefined) return <p role="status">{t(strings.shot.loading)}</p>;
  const historical = head.revision !== baseRevision;
  const recordedFrames = plannedFrames(videoTiming);
  const timingReady = Boolean(source) && (sourceIsVideo
    ? Boolean(videoTiming && recordedFrames !== undefined && motionRevision.data && durationMatches(motionRevision.data, videoTiming) && recordedFrames === motionFrames(motionRevision.data, videoTiming.outputRateNum) && recordedFrames === binding.motionOutput?.frameCount && recordedFrames <= (binding.motionOutput?.profileVersion === 2 ? 300 : 48) && videoTiming.outputRateDen === 1 && [24, 30].includes(videoTiming.outputRateNum) && motionRevision.data.tracks.length > 0 && motionRevision.data.tracks.every((track) => track.target?.deviceRoot || Boolean(track.target?.sceneCameraId)))
    : !videoTiming);
  const changed = !sameJson(binding, savedBinding) || sourceId !== head.snapshot.presentation.sourceMediaId || name !== head.name || !sameJson(videoTiming, head.snapshot.videoTiming) || !sameJson(screenPresentationFromDraft(sourceId, presentationDraft, head.snapshot.presentation), head.snapshot.presentation);
  const presentationInvalid = screenPresentationDraftInvalid(presentationDraft);
  const renderRevision = head;
  const updateTRS = (key: "sceneSupportWorld" | "deviceRootRest", axis: "x" | "y" | "z", value: string) => {
    const parsed = Number(value);
    if (!Number.isFinite(parsed)) return;
    const current = binding[key];
    const previous = current?.translationM;
    const translation = create(Vec3Schema, { x: axis === "x" ? parsed : previous?.x ?? 0, y: axis === "y" ? parsed : previous?.y ?? 0, z: axis === "z" ? parsed : previous?.z ?? 0 });
    const next = create(TRSSchema, { ...(current ?? {}), translationM: translation });
    setBinding(create(PhysicalShotBindingSchema, { ...binding, [key]: next }));
  };
  const updateVideoTiming = (next?: VideoTiming) => {
    setVideoTiming(next);
    if (!next || !binding.motionOutput) return;
    const frames = plannedFrames(next);
    const maxFrames = binding.motionOutput.profileVersion === 2 ? 300 : 48;
    if (frames === undefined || frames < 1 || frames > maxFrames || next.outputRateDen !== 1 || !motionRevision.data || !durationMatches(motionRevision.data, next) || frames !== motionFrames(motionRevision.data, next.outputRateNum) || !motionRevision.data.tracks.length || !motionRevision.data.tracks.every((track) => track.target?.deviceRoot || Boolean(track.target?.sceneCameraId)) || ![24, 30].includes(next.outputRateNum)) return;
    setBinding(create(PhysicalShotBindingSchema, {
      ...binding,
      motionOutput: create(PhysicalMotionOutputBindingSchema, {
        ...binding.motionOutput,
        outputRateNumerator: next.outputRateNum,
        outputRateDenominator: next.outputRateDen,
        frameCount: frames,
      }),
    }));
  };
  const tick = binding.motionSampleTick ?? { numerator: 0n, denominator: 1n };
  const editTick = (field: "numerator" | "denominator", value: string) => {
    if (!/^-?\d+$/.test(value)) return;
    const next = create(RationalTimeSchema, { numerator: field === "numerator" ? BigInt(value) : tick.numerator, denominator: field === "denominator" ? BigInt(value) : tick.denominator });
    setBinding(create(PhysicalShotBindingSchema, { ...binding, motionSampleTick: next }));
  };
  const changeBackend = (value: string) => {
    const backend = value === "OPTIX" ? PhysicalRenderBackend.OPTIX : value === "CUDA" ? PhysicalRenderBackend.CUDA : PhysicalRenderBackend.CPU;
    setBinding(create(PhysicalShotBindingSchema, { ...binding, backend }));
    setCompileIssues([]);
  };
  const changeQuaternion = (axis: "x" | "y" | "z" | "w", value: string) => {
    const parsed = Number(value);
    if (!Number.isFinite(parsed)) return;
    const current = binding.deviceRootRest?.rotationXyzw;
    const rotationXyzw = create(QuatXYZWSchema, {
      x: axis === "x" ? parsed : current?.x ?? 0,
      y: axis === "y" ? parsed : current?.y ?? 0,
      z: axis === "z" ? parsed : current?.z ?? 0,
      w: axis === "w" ? parsed : current?.w ?? 1,
    });
    const currentRest = binding.deviceRootRest;
    const deviceRootRest = create(TRSSchema, { translationM: currentRest?.translationM, rotationXyzw, scale: currentRest?.scale });
    setBinding(create(PhysicalShotBindingSchema, { ...binding, deviceRootRest }));
    setCompileIssues([]);
  };
  const changeStillOutput = (value: string) => {
    if (binding.motionOutput) return;
    const outputProfile = value === "3840" ? PhysicalStillOutputProfile.PHYSICAL_STILL_OUTPUT_PROFILE_3840X2160_V1 : PhysicalStillOutputProfile.PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1;
    setBinding(create(PhysicalShotBindingSchema, { ...binding, outputProfile }));
  };
  const motion = binding.motionOutput;
  const outputKind = motion ? t(strings.physical.motionOutput, { frames: motion.frameCount, rate: motion.outputRateNumerator }) : t(strings.physical.stillOutput);
  const conflict = shotConflict(save.error);
  const reloadLatest = async (keepDraft: boolean) => {
    const draftBinding = keepDraft ? copyBinding(binding) : undefined;
    const draftSource = sourceId;
    const draftName = name;
    const draftTiming = videoTiming ? create(VideoTimingSchema, videoTiming) : undefined;
    const result = await shot.refetch();
    const latest = result.data?.head;
    if (!latest?.snapshot?.physicalBinding || !latest.snapshot.presentation) return;
    setBaseRevision(latest.revision);
    if (keepDraft && draftBinding) { setBinding(draftBinding); setSourceId(draftSource); setVideoTiming(draftTiming); setName(draftName); }
    else {
      setBinding(copyBinding(latest.snapshot.physicalBinding));
      setSourceId(latest.snapshot.presentation.sourceMediaId);
      setPresentationDraft(screenPresentationDraftFromProto(latest.snapshot.presentation));
      setVideoTiming(latest.snapshot.videoTiming ? create(VideoTimingSchema, latest.snapshot.videoTiming) : undefined);
      setName(latest.name);
    }
    setCompiledRevision(undefined); save.reset();
    await queryClient.invalidateQueries({ queryKey: ["project-shots", projectId] });
  };
  return <section className="physical-shot" data-testid={selectors.physical.editor}>
    <header className="physical-shot__header"><div><p className="eyebrow">{t(strings.physical.candidateOnly)}</p><h2>{name}</h2><p>{t(strings.shot.revision, { revision: head.revision.toString() })} · {outputKind}</p></div></header>
    {compileIssues.length > 0 && <div className="physical-shot__issues" role="alert">{compileIssues.map((issue, index) => <p key={`${issue.fieldPath}-${index}`}><strong>{issue.fieldPath}:</strong> {issue.summary}{issue.repair ? ` ${issue.repair}` : ""}</p>)}</div>}
    {mobile && <nav className="physical-shot__editor-steps" aria-label={t(strings.physical.steps)}>{[t(strings.physical.screen), t(strings.physical.export), t(strings.physical.advanced), t(strings.physical.compile)].map((label, index) => <button type="button" key={label} aria-current={mobileStep === index ? "step" : undefined} onClick={() => setMobileStep(index)}>{index + 1}. {label}</button>)}</nav>}
    <div className="physical-shot__layout">
      <Card className="physical-shot__stage"><CardHeader><CardTitle as="h3">{output && showOutput ? t(strings.physical.renderedOutput) : t(strings.physical.sourceStage)}</CardTitle><p>{output && showOutput ? `${output.metadata.widthPixels} × ${output.metadata.heightPixels}` : source ? `${source.originalFilename} · ${sourceSize?.width ?? "—"} × ${sourceSize?.height ?? "—"}` : t(strings.physical.noRender)}</p></CardHeader><CardContent>
        {output && showOutput ? output.isVideo ? <video controls playsInline preload="metadata" src={output.url} aria-label={t(strings.physical.outputAlt)}><track kind="captions" srcLang="en" label={t(strings.shot.videoNoCaptions)} src="data:text/vtt,WEBVTT%0A%0ANOTE%20Captions%20are%20not%20available%20for%20this%20preview." /></video> : <img src={output.url} alt={t(strings.physical.outputAlt)} /> : sourceIsVideo ? <video muted controls playsInline preload="metadata" src={sourceMediaContentURL(projectId, source.id)} aria-label={t(strings.physical.sourceAlt)} /> : sourceUrl ? <img src={sourceUrl} alt={t(strings.physical.sourceAlt)} /> : <div className="physical-shot__empty-stage"><strong>{t(strings.physical.noRender)}</strong><span>{t(strings.physical.noPreview)}</span></div>}
        <div className="physical-shot__stage-actions">
          {output && <Button type="button" variant="outline" className="min-h-11" aria-pressed={showOutput} onClick={() => setShowOutput(true)}>{t(strings.physical.renderedOutput)}</Button>}
          {source && <Button type="button" variant="outline" className="min-h-11" aria-pressed={!showOutput} onClick={() => setShowOutput(false)}>{t(strings.physical.sourceStage)}</Button>}
        </div>
        {source && <details className="physical-shot__source-details"><summary>{t(strings.physical.sourceStage)}</summary><p>{source.originalFilename} · {sourceSize?.width ?? "—"} × {sourceSize?.height ?? "—"}</p><p>{sourceBytes.isSuccess ? `${source.sha256} · ${source.sizeBytes.toString()} ${t(strings.physical.bytes)} · ${t(strings.physical.verifiedSource)}` : sourceIsVideo ? t(strings.physical.verifiedSource) : sourceBytes.isPending ? t(strings.physical.verifyingSource) : t(strings.physical.sourceNotVerified)}</p></details>}
      </CardContent></Card>
      <Card><CardHeader><CardTitle as="h3">{mobile ? t(([strings.physical.screen, strings.physical.export, strings.physical.advanced, strings.physical.compile] as const)[mobileStep] ?? strings.physical.compile) : t(strings.physical.savedBinding)}</CardTitle></CardHeader><CardContent className="physical-shot__fields">
        <p className="physical-shot__candidate">{t(strings.physical.candidateOnly)}</p>
        {(!mobile || mobileStep === 0) && <label>{t(strings.shot.nameLabel)}<input className="physical-shot__input" value={name} onChange={(event) => setName(event.target.value)} disabled={historical} /></label>}
        <details className="physical-shot__technical"><summary>{t(strings.physical.technicalRefs)}</summary><p>{t(strings.physical.deviceRef, { id: binding.device?.id ?? "—", revision: binding.device?.revision ?? 0, sha: binding.device?.sha256.slice(0, 12) ?? "—" })}</p>
        <p>{t(strings.physical.sceneRef, { id: binding.scene?.id ?? "—", revision: binding.scene?.revision ?? 0, sha: binding.scene?.sha256.slice(0, 12) ?? "—" })}</p>
        <p>{t(strings.physical.motionRef, { id: binding.motion?.id ?? "—", revision: binding.motion?.revision ?? 0, sha: binding.motion?.sha256.slice(0, 12) ?? "—" })}</p></details>
        {(!mobile || mobileStep === 0) && <label>{t(strings.physical.sourceSelect)}<select className="physical-shot__input" value={sourceId} onChange={(event) => {
          const next = sources.find((item) => item.id === event.target.value);
          setSourceId(event.target.value);
          if (sourceIsVideo && next?.kind === SourceMediaKind.VIDEO) updateVideoTiming(videoTimingFromMedia(next));
        }} disabled={historical}><option value="">{t(strings.physical.sourcePlaceholder)}</option>{compatibleSources.map((item) => { const raster = sourceRaster(item); return <option key={item.id} value={item.id}>{item.originalFilename} · {raster?.width} × {raster?.height}</option>; })}</select></label>}
        {(!mobile || mobileStep === 0) && <details open><summary>{t(strings.shot.screenPresentation)}</summary><ScreenPresentationEditor value={presentationDraft} disabled={historical} onChange={(patch) => setPresentationDraft((current) => ({ ...current, ...patch }))} /></details>}
        {sourceIsVideo && (!mobile || mobileStep === 0) && <><p>{t(strings.physical.appOnlyCapture)}</p><p>{t(strings.physical.derivedRenderRaster)}</p></>}
        {sourceIsVideo && videoTiming && (!mobile || mobileStep === 0) && <VideoTimingEditor projectId={projectId} source={source} timing={videoTiming} disabled={historical} onChange={updateVideoTiming} />}
        {sourceIsVideo && !timingReady && (!mobile || mobileStep === 0) && <p role="alert">{t(strings.physical.recordingRangeLimit)}</p>}
        {sourceBytes.error && (!mobile || mobileStep === 0) && <div role="alert"><p>{errorMessage(sourceBytes.error, t)}</p><Button variant="outline" onClick={() => void sourceBytes.refetch()}>{t(strings.media.retry)}</Button></div>}
        {media.hasNextPage && (!mobile || mobileStep === 0) && <Button type="button" variant="outline" className="min-h-11" disabled={media.isFetchingNextPage} onClick={() => void media.fetchNextPage()}>{media.isFetchingNextPage ? t(strings.shot.renderLoadingJobs) : t(strings.media.loadMore)}</Button>}
        {(!mobile || mobileStep === 1) && <div><label>{t(strings.physical.backend)}<select className="physical-shot__input" value={binding.backend === PhysicalRenderBackend.OPTIX ? "OPTIX" : binding.backend === PhysicalRenderBackend.CUDA ? "CUDA" : binding.backend === PhysicalRenderBackend.CPU ? "CPU" : ""} onChange={(event) => changeBackend(event.target.value)} disabled={historical}><option value="">{t(strings.physical.chooseBackend)}</option><option value="OPTIX">{t(strings.physical.backendOptix)}</option><option value="CUDA">{t(strings.physical.backendCuda)}</option><option value="CPU">{t(strings.physical.backendCpu)}</option></select></label>
        {!motion && <label>{t(strings.physical.outputProfile)}<select className="physical-shot__input" value={binding.outputProfile === PhysicalStillOutputProfile.PHYSICAL_STILL_OUTPUT_PROFILE_3840X2160_V1 ? "3840" : "2560"} onChange={(event) => changeStillOutput(event.target.value)} disabled={historical}><option value="2560">{t(strings.physical.output2560)}</option><option value="3840">{t(strings.physical.output3840)}</option></select></label>}</div>}
        <details open={mobile && mobileStep === 2} className={mobile && mobileStep !== 2 ? "physical-shot__mobile-hidden" : ""}><summary>{t(strings.physical.advanced)}</summary>
          <fieldset disabled={historical}><legend>{t(strings.physical.translationMeters)}</legend>{(["x", "y", "z"] as const).map((axis) => <label key={`scene-${axis}`}>{t(strings.physical.sceneSupport)} · {axis}<input className="physical-shot__input" type="number" step="0.001" value={binding.sceneSupportWorld?.translationM?.[axis] ?? 0} onChange={(event) => updateTRS("sceneSupportWorld", axis, event.target.value)} /></label>)}</fieldset>
          <fieldset disabled={historical}><legend>{t(strings.physical.deviceTranslation)}</legend>{(["x", "y", "z"] as const).map((axis) => <label key={`device-${axis}`}>{axis}<input className="physical-shot__input" type="number" step="0.001" value={binding.deviceRootRest?.translationM?.[axis] ?? 0} onChange={(event) => updateTRS("deviceRootRest", axis, event.target.value)} /></label>)}</fieldset>
          <fieldset disabled={historical}><legend>{t(strings.physical.deviceRotation)}</legend>{(["x", "y", "z", "w"] as const).map((axis) => <label key={`rotation-${axis}`}>{axis}<input className="physical-shot__input" type="number" step="0.001" value={binding.deviceRootRest?.rotationXyzw?.[axis] ?? (axis === "w" ? 1 : 0)} onChange={(event) => changeQuaternion(axis, event.target.value)} /></label>)}</fieldset>
          {!motion && <fieldset disabled={historical}><legend>{t(strings.physical.motionTick)}</legend><label>{t(strings.physical.tickNumerator)}<input className="physical-shot__input" inputMode="numeric" value={tick.numerator.toString()} onChange={(event) => editTick("numerator", event.target.value)} /></label><label>{t(strings.physical.tickDenominator)}<input className="physical-shot__input" inputMode="numeric" value={tick.denominator.toString()} onChange={(event) => editTick("denominator", event.target.value)} /></label></fieldset>}
          {motion && <p>{t(strings.physical.motionDetails, { width: motion.width, height: motion.height, frames: motion.frameCount, rate: motion.outputRateNumerator })}</p>}
        </details>
        {historical && <div role="alert" className="physical-shot__conflict"><p>{t(strings.physical.staleRevision)}</p><Button variant="outline" onClick={() => void reloadLatest(false)}>{t(strings.physical.reloadLatest)}</Button><Button variant="outline" disabled={!changed} onClick={() => void reloadLatest(true)}>{t(strings.physical.reapplyDraft)}</Button></div>}
        {save.error && <div role="alert"><p>{conflict ? t(strings.physical.staleRevision) : errorMessage(save.error, t)}</p>{conflict && <Button variant="outline" onClick={() => void reloadLatest(true)}>{t(strings.physical.reapplyDraft)}</Button>}</div>}
        {compiled.error && <p role="alert">{errorMessage(compiled.error, t)}</p>}
        {mobile && mobileStep < 3 && <div className="physical-shot__mobile-controls"><Button variant="outline" className="min-h-11" disabled={mobileStep === 0} onClick={() => setMobileStep((value) => Math.max(0, value - 1))}>{t(strings.physical.back)}</Button><Button className="studio-primary min-h-11" onClick={() => setMobileStep((value) => Math.min(3, value + 1))}>{t(strings.physical.continue)}</Button></div>}
        <div className={`physical-shot__actions${mobile && mobileStep !== 3 ? " physical-shot__mobile-hidden" : ""}`}>
          <Button variant="outline" className="min-h-11" disabled={!changed || !timingReady || presentationInvalid || historical || save.isPending} onClick={() => save.mutate()}>{save.isPending ? t(strings.physical.saving) : t(strings.physical.saveRevision)}</Button>
          <Button className="studio-primary min-h-11" disabled={changed || historical || compiled.isPending} onClick={() => compiled.mutate(baseRevision)}>{compiled.isPending ? t(strings.physical.compiling) : t(strings.physical.compile)}</Button>
        </div>
        {compiledRevision === baseRevision && (!mobile || mobileStep === 3) && <p role="status">{t(strings.physical.compileReady)}</p>}
        {compiledRevision === baseRevision && (!mobile || mobileStep === 3) && <RenderControls projectId={projectId} shotId={shotId} revision={renderRevision} historical={historical} hasUnsavedChanges={changed} showOutput={showOutput} onShowOutputChange={setShowOutput} onFinalOutputChange={setOutput} />}
      </CardContent></Card>
    </div>
  </section>;
}
