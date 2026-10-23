import { create } from "@bufbuild/protobuf";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";
import {
  CoordinateConvention,
  MotionMode,
  PhotographicCameraMotionSchema,
  PhotographicSceneSchema,
  type ReadabilityProof,
  PlateQuadSchema,
  PresentationAppearance,
  PresentationRGBSchema,
  PresentationRectSchema,
  PresentationKind,
  ShotIssueCode,
  ShotSnapshotSchema,
  VideoEOFPolicy,
  VideoTimingSchema,
  type ShotRevision,
  type ShotSnapshot,
  type VideoTiming,
  type VideoSampleBinding,
} from "@vrooli/proto-types/device-studio/v1/shots/shots_pb";
import { MediaReadiness, SourceMediaKind } from "@vrooli/proto-types/device-studio/v1/projects/projects_pb";
import { strings } from "../../consts/strings";
import { selectors } from "../../consts/selectors";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { getSourceMediaContent, projectsClient, sourceMediaContentURL, verifySourceMediaContent, type SourceMedia } from "../../api/projects";
import { shotConflict, shotsClient } from "../../api/shots";
import { RenderControls, type FinalRenderOutput } from "./RenderControls";
import { VideoTimingEditor } from "./VideoTimingEditor";
import { ScreenPresentationEditor, emptyScreenPresentationDraft, screenPresentationDraftFromProto, screenPresentationDraftInvalid, screenPresentationFromDraft, type ScreenPresentationDraft } from "./ScreenPresentationEditor";
import { NewPhysicalShot, PhysicalShotEditor } from "./PhysicalShotWorkspace";

type PointDraft = { x: string; y: string };
type ShotDraft = ScreenPresentationDraft & {
  name: string;
  plateMediaId: string;
  activeMaskMediaId: string;
  hardwareMaskMediaId: string;
  displayWidthPixels: string;
  displayHeightPixels: string;
  topLeft: PointDraft;
  topRight: PointDraft;
  bottomRight: PointDraft;
  bottomLeft: PointDraft;
  depictionLabel: string;
  coordinateConvention: CoordinateConvention;
  motionMode: MotionMode;
  cameraProfile: string;
  cameraSceneEnabled: boolean;
  cameraSceneX: string;
  cameraSceneY: string;
  cameraSceneWidth: string;
  cameraSceneHeight: string;
  cameraDurationOutputFrames: string;
  cameraMatteColor: string;
  cameraTargetEnabled: boolean;
  cameraTargetX: string;
  cameraTargetY: string;
  cameraTargetWidth: string;
  cameraTargetHeight: string;
  sourceMediaId: string;
  videoTiming?: VideoTiming;
  readabilityProof?: ReadabilityProof;
};
type ShotDraftBuffer = { draft: ShotDraft; baseRevision: bigint };

function isPreviewableImage(media: SourceMedia | undefined): media is SourceMedia & { width: number; height: number } {
  return Boolean(media && media.kind === SourceMediaKind.IMAGE && media.readiness === MediaReadiness.READY &&
    media.width !== undefined && media.height !== undefined && Number.isInteger(media.width) && Number.isInteger(media.height) &&
    media.width > 0 && media.height > 0);
}

function isReadyInspectedVideo(media: SourceMedia | undefined): media is SourceMedia {
  return Boolean(media && media.kind === SourceMediaKind.VIDEO && media.readiness === MediaReadiness.READY && videoTimingFromMedia(media));
}

const emptyPoint = (): PointDraft => ({ x: "", y: "" });
const emptyDraft = (name = "") : ShotDraft => ({
  name, plateMediaId: "", activeMaskMediaId: "", hardwareMaskMediaId: "",
  displayWidthPixels: "", displayHeightPixels: "", topLeft: emptyPoint(), topRight: emptyPoint(), bottomRight: emptyPoint(), bottomLeft: emptyPoint(),
  depictionLabel: "", coordinateConvention: CoordinateConvention.PLATE_RASTER_EDGE_PIXELS_V1, motionMode: MotionMode.FIXED,
  cameraProfile: "screen-entry-stationary-scene-v1", cameraSceneEnabled: false, cameraSceneX: "0", cameraSceneY: "0", cameraSceneWidth: "100", cameraSceneHeight: "100",
  cameraDurationOutputFrames: "24", cameraMatteColor: "#000000",
  cameraTargetEnabled: false, cameraTargetX: "0", cameraTargetY: "0", cameraTargetWidth: "100", cameraTargetHeight: "100",
  ...emptyScreenPresentationDraft(), sourceMediaId: "",
});

function colorFromDraft(value: string) {
  const match = /^#?([\da-f]{2})([\da-f]{2})([\da-f]{2})$/i.exec(value.trim());
  return match ? create(PresentationRGBSchema, { red: Number.parseInt(match[1] ?? "", 16), green: Number.parseInt(match[2] ?? "", 16), blue: Number.parseInt(match[3] ?? "", 16) }) : undefined;
}

function colorToDraft(value?: { red: number; green: number; blue: number }) {
  return value ? `#${[value.red, value.green, value.blue].map((channel) => channel.toString(16).padStart(2, "0")).join("")}` : "";
}

function plannedOutputFrames(timing?: VideoTiming): bigint | undefined {
  if (!timing || timing.trimEndPts <= timing.trimStartPts || timing.timeBaseNum <= 0n || timing.timeBaseDen <= 0n ||
      timing.playbackRateNum <= 0 || timing.playbackRateDen <= 0 || timing.outputRateNum <= 0 || timing.outputRateDen <= 0) return undefined;
  const numerator = (timing.trimEndPts - timing.trimStartPts) * timing.timeBaseNum * BigInt(timing.playbackRateDen) * BigInt(timing.outputRateNum);
  const denominator = timing.timeBaseDen * BigInt(timing.playbackRateNum) * BigInt(timing.outputRateDen);
  return (numerator + denominator - 1n) / denominator;
}

function cameraScheduleLimitFrames(timing?: VideoTiming): bigint | undefined {
  if (!timing || timing.outputRateDen !== 1 || ![24, 30].includes(timing.outputRateNum)) return undefined;
  return BigInt(timing.outputRateNum * 12);
}

function cameraDurationSeconds(frames: bigint, timing?: VideoTiming): string {
  if (!timing || timing.outputRateNum <= 0 || timing.outputRateDen <= 0) return "—";
  const numerator = frames * BigInt(timing.outputRateDen) * 100n;
  const denominator = BigInt(timing.outputRateNum);
  const hundredths = (numerator + denominator / 2n) / denominator;
  return `${hundredths / 100n}.${(hundredths % 100n).toString().padStart(2, "0")}`;
}

function cameraRectInvalid(draft: ShotDraft, which: "Target" | "Scene"): boolean {
  if (draft.motionMode !== MotionMode.CROP_PUSH || !draft[`camera${which}Enabled`]) return false;
  const values = [draft[`camera${which}X`], draft[`camera${which}Y`], draft[`camera${which}Width`], draft[`camera${which}Height`]];
  const [x, y, width, height] = values.map(Number) as [number, number, number, number];
  const invalidPercent = values.some((value) => {
    const percent = Number(value);
    return value.trim() === "" || !Number.isFinite(percent) || percent < 0 || percent > 100
      || Math.abs(percent * 100 - Math.round(percent * 100)) > 1e-6;
  });
  return invalidPercent || width <= 0 || height <= 0 || x + width > 100 || y + height > 100;
}

function snapshotFromDraft(draft: ShotDraft) {
  const point = (value: PointDraft) => value.x !== "" && value.y !== "" ? { x: Number(value.x), y: Number(value.y) } : undefined;
  const quadPoints = {
    topLeft: point(draft.topLeft), topRight: point(draft.topRight),
    bottomRight: point(draft.bottomRight), bottomLeft: point(draft.bottomLeft),
  };
  const plateQuad = Object.values(quadPoints).some((value) => value !== undefined)
    ? create(PlateQuadSchema, quadPoints)
    : undefined;
  return create(ShotSnapshotSchema, {
    motionMode: draft.motionMode,
    cameraMotion: draft.motionMode === MotionMode.CROP_PUSH ? create(PhotographicCameraMotionSchema, {
      profileId: draft.cameraProfile, profileVersion: 1,
      sceneRect: draft.cameraProfile === "screen-entry-stationary-scene-v1" && draft.cameraSceneEnabled ? create(PresentationRectSchema, {
        x: Math.round(Number(draft.cameraSceneX)*100), y: Math.round(Number(draft.cameraSceneY)*100),
        width: Math.round(Number(draft.cameraSceneWidth)*100), height: Math.round(Number(draft.cameraSceneHeight)*100),
      }) : undefined,
      durationOutputFrames: Number(draft.cameraDurationOutputFrames),
      targetRect: draft.cameraTargetEnabled ? create(PresentationRectSchema, {
        x: Math.round(Number(draft.cameraTargetX) * 100), y: Math.round(Number(draft.cameraTargetY) * 100),
        width: Math.round(Number(draft.cameraTargetWidth) * 100), height: Math.round(Number(draft.cameraTargetHeight) * 100),
      }) : undefined,
      matteRgb: colorFromDraft(draft.cameraMatteColor) ?? create(PresentationRGBSchema, { red: 0, green: 0, blue: 0 }),
    }) : undefined,
    scene: create(PhotographicSceneSchema, {
      plateMediaId: draft.plateMediaId, activeMaskMediaId: draft.activeMaskMediaId,
      hardwareMaskMediaId: draft.hardwareMaskMediaId,
      displayWidthPixels: Number(draft.displayWidthPixels || 0), displayHeightPixels: Number(draft.displayHeightPixels || 0),
      plateQuad, coordinateConvention: draft.coordinateConvention, depictionLabel: draft.depictionLabel,
    }),
    presentation: screenPresentationFromDraft(draft.sourceMediaId, draft),
    videoTiming: draft.videoTiming,
    readabilityProof: draft.readabilityProof,
  });
}

function videoTimingFromMedia(source: SourceMedia): VideoTiming | undefined {
  const receipt = source.inspectionReceipt;
  if (source.kind !== SourceMediaKind.VIDEO || source.readiness !== MediaReadiness.READY || !receipt || receipt.classification !== "accepted" ||
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
    videoCompositionProfileId: "device-studio-fixed-plate-video-v1", videoCompositionProfileVersion: 1,
  });
}

function draftFromRevision(name: string, revision: ShotRevision): ShotDraft {
  const scene = revision.snapshot?.scene;
  const presentation = revision.snapshot?.presentation;
  const point = (value?: { x: number; y: number }): PointDraft => value ? { x: String(value.x), y: String(value.y) } : emptyPoint();
  return {
    ...emptyDraft(name), plateMediaId: scene?.plateMediaId ?? "", activeMaskMediaId: scene?.activeMaskMediaId ?? "",
    hardwareMaskMediaId: scene?.hardwareMaskMediaId ?? "", displayWidthPixels: scene?.displayWidthPixels ? String(scene.displayWidthPixels) : "",
    displayHeightPixels: scene?.displayHeightPixels ? String(scene.displayHeightPixels) : "", topLeft: point(scene?.plateQuad?.topLeft),
    topRight: point(scene?.plateQuad?.topRight), bottomRight: point(scene?.plateQuad?.bottomRight), bottomLeft: point(scene?.plateQuad?.bottomLeft),
    depictionLabel: scene?.depictionLabel ?? "", coordinateConvention: scene?.coordinateConvention ?? CoordinateConvention.UNSPECIFIED,
    motionMode: revision.snapshot?.motionMode ?? MotionMode.FIXED,
    cameraProfile: revision.snapshot?.cameraMotion?.profileId || "screen-entry-stationary-scene-v1",
    cameraSceneEnabled: Boolean(revision.snapshot?.cameraMotion?.sceneRect),
    cameraSceneX: String((revision.snapshot?.cameraMotion?.sceneRect?.x ?? 0)/100),
    cameraSceneY: String((revision.snapshot?.cameraMotion?.sceneRect?.y ?? 0)/100),
    cameraSceneWidth: String((revision.snapshot?.cameraMotion?.sceneRect?.width ?? 10000)/100),
    cameraSceneHeight: String((revision.snapshot?.cameraMotion?.sceneRect?.height ?? 10000)/100),
    cameraDurationOutputFrames: revision.snapshot?.cameraMotion ? String(revision.snapshot.cameraMotion.durationOutputFrames) : "24",
    cameraMatteColor: colorToDraft(revision.snapshot?.cameraMotion?.matteRgb) || "#000000",
    cameraTargetEnabled: Boolean(revision.snapshot?.cameraMotion?.targetRect),
    cameraTargetX: String((revision.snapshot?.cameraMotion?.targetRect?.x ?? 0) / 100),
    cameraTargetY: String((revision.snapshot?.cameraMotion?.targetRect?.y ?? 0) / 100),
    cameraTargetWidth: String((revision.snapshot?.cameraMotion?.targetRect?.width ?? 10000) / 100),
    cameraTargetHeight: String((revision.snapshot?.cameraMotion?.targetRect?.height ?? 10000) / 100),
    sourceMediaId: presentation?.sourceMediaId ?? "", ...screenPresentationDraftFromProto(presentation),
    videoTiming: revision.snapshot?.videoTiming,
    readabilityProof: revision.snapshot?.readabilityProof,
  };
}

function bytes(value: bigint) { return `${(Number(value) / 1024 / 1024).toFixed(2)} MiB`; }
function draftFingerprint(value: unknown) { return JSON.stringify(value, (_key, item: unknown) => typeof item === "bigint" ? item.toString() : item); }

export function ShotWorkspace({ projectId }: { projectId: string }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [selectedShotId, setSelectedShotId] = useState("");
  const [creatingPhysical, setCreatingPhysical] = useState(false);
  const [newName, setNewName] = useState("");
  const [drafts, setDrafts] = useState<Record<string, ShotDraftBuffer>>({});
  const cacheDraft = useCallback((buffer?: ShotDraftBuffer) => setDrafts((current) => { const next: Record<string, ShotDraftBuffer> = {}; for (const [id, value] of Object.entries(current)) if (id !== selectedShotId) next[id] = value; if (buffer) next[selectedShotId] = buffer; return next; }), [selectedShotId]);
  const shots = useInfiniteQuery({
    queryKey: ["project-shots", projectId], initialPageParam: "",
    queryFn: ({ pageParam }) => shotsClient.listShots({ projectId, pageSize: 25, pageToken: pageParam }),
    getNextPageParam: (page) => page.nextPageToken || undefined, enabled: Boolean(projectId),
  });
  const records = useMemo(() => shots.data?.pages.flatMap((page) => page.shots) ?? [], [shots.data]);
  useEffect(() => {
    const first = records[0];
    if (!selectedShotId && first) setSelectedShotId(first.id);
  }, [records, selectedShotId]);
  const createShot = useMutation({
    mutationFn: (name: string) => shotsClient.createShot({ projectId, name, snapshot: snapshotFromDraft(emptyDraft(name)) }),
    onSuccess: async (result) => {
      if (result.shot) setSelectedShotId(result.shot.id);
      await queryClient.invalidateQueries({ queryKey: ["project-shots", projectId] });
    },
  });
  return <section className="shot-workspace" data-testid="shot-workspace" aria-labelledby="shot-workspace-title">
    {shots.error && <p role="alert">{errorMessage(shots.error, t)}</p>}
    {createShot.error && <p role="alert">{errorMessage(createShot.error, t)}</p>}
    {creatingPhysical ? <NewPhysicalShot projectId={projectId} onCancel={() => setCreatingPhysical(false)} onCreated={(shotId) => { setCreatingPhysical(false); setSelectedShotId(shotId); }} /> : selectedShotId ? <ShotEditor key={selectedShotId} projectId={projectId} shotId={selectedShotId} mediaQueryKey={["project-source-media", projectId]} draftCache={drafts[selectedShotId]} onDraftChange={cacheDraft} /> :
      <p className="shot-workspace__empty">{records.length ? t(strings.shot.chooseSaved) : t(strings.shot.empty)}</p>}
    <details className="shot-management" open={shots.data !== undefined && records.length === 0}>
      <summary>{t(strings.shot.savedShots)} · {records.length}</summary>
      <div className="shot-management__content">
        <form className="shot-create" onSubmit={(event) => { event.preventDefault(); const name = newName.trim(); if (name) createShot.mutate(name); }}>
          <label className="sr-only" htmlFor="new-shot-name">{t(strings.shot.nameLabel)}</label>
          <input id="new-shot-name" data-testid="new-shot-name" value={newName} onChange={(event) => setNewName(event.target.value)} placeholder={t(strings.shot.newNamePlaceholder)} maxLength={120} />
          <Button type="submit" className="studio-primary min-h-11" disabled={!newName.trim() || createShot.isPending}>{t(strings.shot.create)}</Button>
          <Button type="button" variant="outline" className="min-h-11" onClick={() => setCreatingPhysical(true)}>{t(strings.physical.newButton)}</Button>
        </form>
        {records.length > 0 && <nav className="shot-list" aria-label={t(strings.shot.savedShots)}>{records.map((shot) =>
          <button key={shot.id} type="button" aria-pressed={selectedShotId === shot.id} onClick={() => setSelectedShotId(shot.id)}>{shot.name}<span>{t(strings.shot.revision, { revision: shot.headRevision.toString() })}</span></button>
        )}</nav>}
        {shots.hasNextPage && <Button variant="outline" className="min-h-11" disabled={shots.isFetchingNextPage} onClick={() => void shots.fetchNextPage()}>{t(strings.projects.loadMore)}</Button>}
      </div>
    </details>
  </section>;
}

function ShotEditor({ projectId, shotId, mediaQueryKey, draftCache, onDraftChange }: { projectId: string; shotId: string; mediaQueryKey: readonly unknown[]; draftCache?: ShotDraftBuffer; onDraftChange: (buffer?: ShotDraftBuffer) => void }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [draft, setDraft] = useState<ShotDraft | undefined>(draftCache?.draft);
  const [baseRevision, setBaseRevision] = useState<bigint | undefined>(draftCache?.baseRevision);
  const [selectedHistoricalRevision, setSelectedHistoricalRevision] = useState<bigint>();
  const [finalOutput, setFinalOutput] = useState<FinalRenderOutput>();
  const [showFinalPreview, setShowFinalPreview] = useState(false);
  const [sourcePlaybackFailed, setSourcePlaybackFailed] = useState(false);
  const stageRef = useRef<HTMLDivElement>(null);
  const onFinalOutputChange = useCallback((output?: FinalRenderOutput) => setFinalOutput(output), []);
  useEffect(() => {
    if (!showFinalPreview) return;
    const stage = stageRef.current;
    if (stage) {
      const scrollIntoView = Reflect.get(stage, "scrollIntoView");
      if (typeof scrollIntoView === "function") Reflect.apply(scrollIntoView, stage, [{ behavior: "smooth", block: "start" }]);
    }
    stageRef.current?.focus();
  }, [showFinalPreview]);
  const [issues, setIssues] = useState<Array<{ code: ShotIssueCode; fieldPath: string; summary: string; repair: string; blocksRender: boolean }>>([]);
  const shotQuery = useQuery({ queryKey: ["shot", shotId], queryFn: () => shotsClient.getShot({ id: shotId }) });
  const revisions = useInfiniteQuery({ queryKey: ["shot-revisions", shotId], initialPageParam: "", queryFn: ({ pageParam }) => shotsClient.listShotRevisions({ shotId, pageSize: 50, pageToken: pageParam }), getNextPageParam: (page) => page.nextPageToken || undefined });
  const record = shotQuery.data?.shot;
  const head = shotQuery.data?.head;
  useEffect(() => {
    if (!record || !head || selectedHistoricalRevision !== undefined) return;
    setBaseRevision((current) => current ?? head.revision);
    if (!draft) {
      const loaded = draftFromRevision(head.name, head);
      setDraft(loaded);
      onDraftChange({ draft: loaded, baseRevision: draftCache?.baseRevision ?? head.revision });
      setIssues(head.issues);
    }
  }, [record, head, draft, onDraftChange, draftCache?.baseRevision, selectedHistoricalRevision]);
  const selectedRevisionQuery = useQuery({
    queryKey: ["shot-revision", shotId, selectedHistoricalRevision?.toString()],
    queryFn: () => shotsClient.getShotRevision({ shotId, revision: selectedHistoricalRevision ?? 0n }),
    enabled: selectedHistoricalRevision !== undefined,
  });
  const selectedRevision = selectedHistoricalRevision === undefined ? head : selectedRevisionQuery.data?.revision;
  const historical = selectedHistoricalRevision !== undefined;
  const pinnedRevisionQuery = useQuery({
    queryKey: ["shot-revision", shotId, "saved-for-render", baseRevision?.toString()],
    queryFn: () => shotsClient.getShotRevision({ shotId, revision: baseRevision ?? 0n }),
    enabled: !historical && baseRevision !== undefined && head !== undefined && baseRevision !== head.revision,
  });
  const renderRevision = historical ? selectedRevision : baseRevision === head?.revision ? head : pinnedRevisionQuery.data?.revision;
  const viewDraft = historical && selectedRevision ? draftFromRevision(selectedRevision.name, selectedRevision) : draft;
  const sources = useInfiniteQuery({ queryKey: mediaQueryKey, initialPageParam: "", queryFn: ({ pageParam }) => projectsClient.listSourceMedia({ projectId, pageSize: 25, pageToken: pageParam }), getNextPageParam: (page) => page.nextPageToken || undefined });
  const media = sources.data?.pages.flatMap((page) => page.sourceMedia) ?? [];
  const listedPlate = media.find((item) => item.id === viewDraft?.plateMediaId);
  const plateMetadata = useQuery({ queryKey: ["project-source-media-item", projectId, viewDraft?.plateMediaId], queryFn: () => projectsClient.getSourceMedia({ projectId, id: viewDraft?.plateMediaId ?? "" }), enabled: Boolean(viewDraft?.plateMediaId && !listedPlate), gcTime: 0 });
  const plateCandidate = listedPlate ?? plateMetadata.data?.sourceMedia;
  const plate = isPreviewableImage(plateCandidate) ? plateCandidate : undefined;
  const listedSource = media.find((item) => item.id === viewDraft?.sourceMediaId);
  const sourceMetadata = useQuery({ queryKey: ["project-source-media-item", projectId, viewDraft?.sourceMediaId], queryFn: () => projectsClient.getSourceMedia({ projectId, id: viewDraft?.sourceMediaId ?? "" }), enabled: Boolean(viewDraft?.sourceMediaId && !listedSource), gcTime: 0 });
  const screenCandidate = listedSource ?? sourceMetadata.data?.sourceMedia;
  const screenSource = isPreviewableImage(screenCandidate) || isReadyInspectedVideo(screenCandidate) ? screenCandidate : undefined;
  const screenImage = isPreviewableImage(screenSource) ? screenSource : undefined;
  const screenVideo = isReadyInspectedVideo(screenSource) ? screenSource : undefined;
  useEffect(() => { setSourcePlaybackFailed(false); }, [screenVideo?.id, screenVideo?.sha256]);
  const plateBytes = useQuery({
    queryKey: ["shot-plate-content", projectId, plate?.id, plate?.sha256],
    queryFn: async () => { if (!plate) throw new Error("No scene plate selected"); return verifySourceMediaContent(await getSourceMediaContent(projectId, plate.id), plate); },
    enabled: Boolean(plate), gcTime: 0, retry: false,
  });
  const screenBytes = useQuery({
    queryKey: ["shot-screen-content", projectId, screenImage?.id, screenImage?.sha256],
    queryFn: async () => { if (!screenImage) throw new Error("No screen source selected"); return verifySourceMediaContent(await getSourceMediaContent(projectId, screenImage.id), screenImage); },
    enabled: Boolean(screenImage), gcTime: 0, retry: false,
  });
  const [plateUrl, setPlateUrl] = useState("");
  const [screenUrl, setScreenUrl] = useState("");
  useEffect(() => {
    if (!plateBytes.data) { setPlateUrl(""); return; }
    const url = URL.createObjectURL(plateBytes.data); setPlateUrl(url); return () => URL.revokeObjectURL(url);
  }, [plateBytes.data]);
  useEffect(() => {
    if (!screenBytes.data) { setScreenUrl(""); return; }
    const url = URL.createObjectURL(screenBytes.data); setScreenUrl(url); return () => URL.revokeObjectURL(url);
  }, [screenBytes.data]);
  const draftSnapshot = useMemo(() => draft ? snapshotFromDraft(draft) : undefined, [draft]);
  const hasUnsavedChanges = Boolean(draft && renderRevision && draftFingerprint(draft) !== draftFingerprint(draftFromRevision(renderRevision.name, renderRevision)));
  const validate = useMutation({
    mutationFn: () => { if (!draftSnapshot) throw new Error("Shot is not loaded"); return shotsClient.validateShot({ projectId, snapshot: draftSnapshot }); },
    onSuccess: (result) => setIssues(result.issues),
  });
  const save = useMutation({
    mutationFn: (submitted: { expectedRevision: bigint; name: string; snapshot: ShotSnapshot; draft: ShotDraft }) => shotsClient.saveShot({ shotId, expectedRevision: submitted.expectedRevision, name: submitted.name, snapshot: submitted.snapshot }),
    onSuccess: async (result, submitted) => {
      if (result.revision) {
        setBaseRevision(result.revision.revision);
        setSelectedHistoricalRevision(undefined);
        setIssues(result.revision.issues);
        const current = draftRef.current;
        const unchangedSinceSubmit = current !== undefined && draftFingerprint(current) === draftFingerprint(submitted.draft);
        const reconciled = unchangedSinceSubmit ? draftFromRevision(result.shot?.name ?? submitted.name, result.revision) : current ?? submitted.draft;
        setDraft(reconciled);
        onDraftChange({ draft: reconciled, baseRevision: result.revision.revision });
      }
      if (result.shot) queryClient.setQueryData(["shot", shotId], { shot: result.shot, head: result.revision });
      await queryClient.invalidateQueries({ queryKey: ["project-shots", projectId] });
      await queryClient.invalidateQueries({ queryKey: ["shot-revisions", shotId] });
    },
  });
  const acceptVideoSample = async (binding: VideoSampleBinding) => {
    const current = draftRef.current;
    const saved = renderRevision ? draftFromRevision(renderRevision.name, renderRevision) : undefined;
    if (!current || !current.videoTiming || !saved || baseRevision === undefined || historical || baseRevision !== renderRevision?.revision ||
        draftFingerprint({ ...current, name: "" }) !== draftFingerprint({ ...saved, name: "" }) || binding.baseShotRevision !== baseRevision ||
        binding.sourceSha256 !== current.videoTiming.sourceSha256 || binding.clockSha256 !== current.videoTiming.clockSha256 ||
        binding.trimStartPts !== current.videoTiming.trimStartPts || binding.trimEndPts !== current.videoTiming.trimEndPts) {
      throw new Error(t(strings.shot.videoSampleStale));
    }
    const sampled: ShotDraft = {
      ...current,
      videoTiming: create(VideoTimingSchema, { ...current.videoTiming, sampleBinding: binding, sampleBaseRecipeVersion: 1 }),
      topFill: binding.topFillSourceResolved ? colorToDraft({ red: binding.resolvedTopFillRed, green: binding.resolvedTopFillGreen, blue: binding.resolvedTopFillBlue }) : current.topFill,
      bottomFill: binding.bottomFillSourceResolved ? colorToDraft({ red: binding.resolvedBottomFillRed, green: binding.resolvedBottomFillGreen, blue: binding.resolvedBottomFillBlue }) : current.bottomFill,
      statusInk: binding.statusInkSourceResolved ? colorToDraft({ red: binding.resolvedStatusInkRed, green: binding.resolvedStatusInkGreen, blue: binding.resolvedStatusInkBlue }) : current.statusInk,
      homeInk: binding.homeInkSourceResolved ? colorToDraft({ red: binding.resolvedHomeInkRed, green: binding.resolvedHomeInkGreen, blue: binding.resolvedHomeInkBlue }) : current.homeInk,
    };
    draftRef.current = sampled;
    setDraft(sampled);
    onDraftChange({ draft: sampled, baseRevision });
    await save.mutateAsync({ expectedRevision: baseRevision, name: sampled.name, snapshot: snapshotFromDraft(sampled), draft: sampled });
  };

  const declareReadabilityProof = async (proof: ReadabilityProof) => {
    const current = draftRef.current;
    if (!current || !renderRevision || baseRevision === undefined || historical || hasUnsavedChanges || baseRevision !== renderRevision.revision) {
      throw new Error("Save or discard current shot edits before changing its readability proof.");
    }
    const next = { ...current, readabilityProof: proof };
    draftRef.current = next;
    setDraft(next);
    await save.mutateAsync({ expectedRevision: baseRevision, name: next.name, snapshot: snapshotFromDraft(next), draft: next });
  };
  const draftRef = useRef(draft);
  draftRef.current = draft;
  const conflict = shotConflict(save.error);
  const updateDraft = (changeDraft: (current: ShotDraft) => ShotDraft) => {
    if (!draft) return;
    let next = changeDraft(draft);
    const currentRecipe = { ...draft, name: "", videoTiming: draft.videoTiming ? { ...draft.videoTiming, sampleBinding: undefined, sampleBaseRecipeVersion: 0 } : undefined };
    const nextRecipe = { ...next, name: "", videoTiming: next.videoTiming ? { ...next.videoTiming, sampleBinding: undefined, sampleBaseRecipeVersion: 0 } : undefined };
    if (draft.videoTiming?.sampleBinding && draftFingerprint(currentRecipe) !== draftFingerprint(nextRecipe)) {
      const claim = draft.videoTiming.sampleBinding;
      next = {
        ...next,
        videoTiming: next.videoTiming ? create(VideoTimingSchema, { ...next.videoTiming, sampleBinding: undefined, sampleBaseRecipeVersion: 0 }) : undefined,
        topFill: claim.topFillSourceResolved && next.topFill === draft.topFill ? "" : next.topFill,
        bottomFill: claim.bottomFillSourceResolved && next.bottomFill === draft.bottomFill ? "" : next.bottomFill,
        statusInk: claim.statusInkSourceResolved && next.statusInk === draft.statusInk ? "" : next.statusInk,
        homeInk: claim.homeInkSourceResolved && next.homeInk === draft.homeInk ? "" : next.homeInk,
      };
    }
    draftRef.current = next;
    setDraft(next);
    if (baseRevision !== undefined) onDraftChange({ draft: next, baseRevision });
    setIssues([]);
    validate.reset();
  };
  const change = <K extends keyof ShotDraft>(key: K, value: ShotDraft[K]) => updateDraft((current) => ({
    ...current,
    [key]: value,
    ...(key === "kind" && value === PresentationKind.PHONE_SYSTEM && current.phoneAppearance === PresentationAppearance.UNSPECIFIED
      ? { phoneAppearance: PresentationAppearance.AUTO }
      : {}),
  }));
  const changePoint = (key: "topLeft" | "topRight" | "bottomRight" | "bottomLeft", axis: "x" | "y", value: string) => updateDraft((current) => ({ ...current, [key]: { ...current[key], [axis]: value } }));
  const reapplyDraftOnLatest = async () => {
    const result = await shotQuery.refetch();
    if (result.data?.head) {
      setBaseRevision(result.data.head.revision); setSelectedHistoricalRevision(undefined);
      if (draft) onDraftChange({ draft, baseRevision: result.data.head.revision });
      save.reset(); void queryClient.invalidateQueries({ queryKey: ["project-shots", projectId] }); void queryClient.invalidateQueries({ queryKey: ["shot-revisions", shotId] });
    }
  };
  const chooseRevision = (revision: bigint) => {
    setSelectedHistoricalRevision(revision === head?.revision ? undefined : revision);
    save.reset();
  };
  if (historical && selectedRevisionQuery.error) return <div role="alert"><p>{errorMessage(selectedRevisionQuery.error, t)}</p><Button variant="outline" className="min-h-11" onClick={() => void selectedRevisionQuery.refetch()}>{t(strings.media.retry)}</Button><Button variant="outline" className="min-h-11" onClick={() => setSelectedHistoricalRevision(undefined)}>{t(strings.shot.backToCurrent)}</Button></div>;
  if (historical && selectedRevisionQuery.isLoading) return <p role="status">{t(strings.shot.loading)}</p>;
  if (shotQuery.isLoading || !draft || !head) {
    if (shotQuery.error) return <div role="alert"><p>{errorMessage(shotQuery.error, t)}</p><Button variant="outline" className="min-h-11" onClick={() => void shotQuery.refetch()}>{t(strings.media.retry)}</Button></div>;
    return <p role="status">{t(strings.shot.loading)}</p>;
  }
  if (historical && !selectedRevision) return <p role="status">{t(strings.shot.loading)}</p>;
  if (head.snapshot?.physicalBinding) return <PhysicalShotEditor projectId={projectId} shotId={shotId} />;
  const displayDraft = viewDraft ?? draft;
  const integerValues = [draft.displayWidthPixels, draft.displayHeightPixels, draft.batteryPercent, draft.windowX, draft.windowY, draft.windowWidth, draft.windowHeight].filter(Boolean).map(Number);
  const pointDrafts = [draft.topLeft, draft.topRight, draft.bottomRight, draft.bottomLeft];
  const partialPoint = pointDrafts.some((point) => (point.x === "") !== (point.y === ""));
  const coordinateOutOfRange = pointDrafts.some((point) => [point.x, point.y].some((raw, axis) => {
    if (raw === "") return false;
    const value = Number(raw);
    return !Number.isFinite(value) || Math.abs(value) > Number.MAX_SAFE_INTEGER || value < 0 || Boolean(plate && value > (axis === 0 ? plate.width : plate.height));
  }));
  const rectValues = [draft.windowX, draft.windowY, draft.windowWidth, draft.windowHeight];
  const rectPartial = rectValues.some(Boolean) && !rectValues.every(Boolean);
  const rectNumeric = rectValues.map((value) => Number(value || 0));
  const [rectX = 0, rectY = 0, rectWidth = 0, rectHeight = 0] = rectNumeric;
  const rectOutOfBounds = rectValues.every(Boolean) && (rectX + rectWidth > 10000 || rectY + rectHeight > 10000 || rectWidth <= 0 || rectHeight <= 0);
  const colorInvalid = [draft.matteColor, draft.topFill, draft.bottomFill, draft.desktopColor, draft.windowColor, draft.chromeColor,
    ...(draft.motionMode === MotionMode.CROP_PUSH ? [draft.cameraMatteColor] : [])].some((value) => value !== "" && !/^#?[\da-f]{6}$/i.test(value));
  const cameraFrames = Number(draft.cameraDurationOutputFrames);
  const clipFrames = plannedOutputFrames(draft.videoTiming);
  const cameraLimitFrames = cameraScheduleLimitFrames(draft.videoTiming);
  const cameraDurationInvalid = draft.motionMode === MotionMode.CROP_PUSH && (!draft.videoTiming || !Number.isInteger(cameraFrames) || cameraFrames < 2 || cameraLimitFrames === undefined || BigInt(cameraFrames) > cameraLimitFrames || clipFrames === undefined || BigInt(cameraFrames) > clipFrames);
  const cameraMatteInvalid = draft.motionMode === MotionMode.CROP_PUSH && !/^#?[\da-f]{6}$/i.test(draft.cameraMatteColor);
  const displayCameraFrames = Number(displayDraft.cameraDurationOutputFrames);
  const displayClipFrames = plannedOutputFrames(displayDraft.videoTiming);
  const displayCameraLimitFrames = cameraScheduleLimitFrames(displayDraft.videoTiming);
  const displayCameraDurationInvalid = displayDraft.motionMode === MotionMode.CROP_PUSH && (!displayDraft.videoTiming || !Number.isInteger(displayCameraFrames) || displayCameraFrames < 2 || displayCameraLimitFrames === undefined || BigInt(displayCameraFrames) > displayCameraLimitFrames || displayClipFrames === undefined || BigInt(displayCameraFrames) > displayClipFrames);
  const displayCameraClipInvalid = displayDraft.motionMode === MotionMode.CROP_PUSH && (displayClipFrames !== undefined && (displayCameraLimitFrames === undefined || displayClipFrames > displayCameraLimitFrames));
  const displayCameraMatteInvalid = displayDraft.motionMode === MotionMode.CROP_PUSH && !/^#?[\da-f]{6}$/i.test(displayDraft.cameraMatteColor);
  const displayCameraDurationStatus = displayDraft.motionMode === MotionMode.CROP_PUSH && Number.isInteger(displayCameraFrames) && displayCameraFrames >= 0
    ? t(strings.shot.cameraDurationSummary, { frames: displayCameraFrames.toString(), seconds: cameraDurationSeconds(BigInt(displayCameraFrames), displayDraft.videoTiming), rate: displayDraft.videoTiming?.outputRateNum.toString() ?? "—" })
    : "";
  const presentationInvalid = screenPresentationDraftInvalid(draft);
  const numericInvalid = integerValues.some((value) => !Number.isFinite(value) || !Number.isInteger(value) || value < 0 || value > 4_294_967_295) || Number(draft.batteryPercent || 0) > 100 || rectPartial || rectOutOfBounds || rectNumeric.some((value) => value > 10000) || colorInvalid || partialPoint || coordinateOutOfRange || cameraDurationInvalid || cameraRectInvalid(draft, "Target") || (draft.cameraProfile === "screen-entry-stationary-scene-v1" && cameraRectInvalid(draft, "Scene"));
  const selectedMediaPending = Boolean((displayDraft.plateMediaId && !plate && plateMetadata.isLoading) || (displayDraft.sourceMediaId && !screenSource && sourceMetadata.isLoading));
  const availableMedia = [...media.filter((item) => isPreviewableImage(item)), ...(plate && !media.some((item) => item.id === plate.id) ? [plate] : [])];
  const screenMediaOptions = [...media.filter((item) => isPreviewableImage(item) || isReadyInspectedVideo(item)), ...(screenSource && !media.some((item) => item.id === screenSource.id) ? [screenSource] : [])];
  const select = (label: string, value: string, onChange: (value: string) => void, options: SourceMedia[], emptyLabel: string) => <label className="shot-field"><span>{label}</span><select value={value} onChange={(event) => onChange(event.target.value)}><option value="">{emptyLabel}</option>{options.map((item) => <option key={item.id} value={item.id}>{item.originalFilename}{item.width !== undefined && item.height !== undefined ? ` · ${item.width} × ${item.height}` : ` · ${item.mediaType}`}</option>)}</select></label>;
  const chooseScreenSource = (id: string) => {
    const source = screenMediaOptions.find((item) => item.id === id);
    updateDraft((current) => current.sourceMediaId === id ? current : ({ ...current, sourceMediaId: id, videoTiming: source && isReadyInspectedVideo(source) ? videoTimingFromMedia(source) : undefined }));
  };
  const chooseMotionMode = (motionMode: MotionMode) => updateDraft((current) => {
    const planned = plannedOutputFrames(current.videoTiming);
    const suggestedFrames = planned === undefined || planned > 24n ? 24 : Math.max(2, Number(planned));
    const currentFrames = Number(current.cameraDurationOutputFrames);
    const exceedsClip = planned !== undefined && Number.isInteger(currentFrames) && BigInt(currentFrames) > planned;
    return {
      ...current,
      motionMode,
      cameraDurationOutputFrames: motionMode === MotionMode.CROP_PUSH && exceedsClip ? String(suggestedFrames) : current.cameraDurationOutputFrames,
      cameraMatteColor: motionMode === MotionMode.CROP_PUSH && !current.cameraMatteColor ? "#000000" : current.cameraMatteColor,
    };
  });
  const numberField = (label: string, value: string, onChange: (value: string) => void, testId?: string) => <label className="shot-field"><span>{label}</span><input data-testid={testId} type="number" inputMode="decimal" value={value} onChange={(event) => onChange(event.target.value)} /></label>;
  const colorField = (label: string, value: string, onChange: (value: string) => void) => <label className="shot-field"><span>{label}</span><input type="text" inputMode="text" autoCapitalize="characters" maxLength={7} placeholder={t(strings.shot.hexPlaceholder)} value={value} onChange={(event) => onChange(event.target.value)} aria-invalid={value !== "" && !/^#?[\da-f]{6}$/i.test(value)} /></label>;
  const activeIssues = historical ? selectedRevision?.issues ?? [] : issues;
  const issuesFor = (path: string) => activeIssues.filter((issue) => !path || issue.fieldPath.toLowerCase().includes(path.toLowerCase()));
  const issueList = (path = "") => { const rows = path ? issuesFor(path) : activeIssues.filter((issue) => !issue.fieldPath.startsWith("scene.") && !issue.fieldPath.startsWith("presentation.")); return rows.length ? <ul className="shot-issues" aria-live="polite">{rows.map((issue, index) => <li key={`${issue.code}-${issue.fieldPath}-${index}`}><span>{issue.summary}</span>{issue.repair && <small>{issue.repair}</small>}<details><summary>{t(strings.shot.issueDetails)}</summary><code>{issue.fieldPath}</code><span>{ShotIssueCode[issue.code].replace(/_/g, " ")}</span></details></li>)}</ul> : null; };
  const pointFields = ([key, label]: ["topLeft" | "topRight" | "bottomRight" | "bottomLeft", string]) => <fieldset key={key} className="shot-point"><legend>{label}</legend>{numberField(t(strings.shot.plateX), displayDraft[key].x, (value) => changePoint(key, "x", value), `plate-x-${key}`)}{numberField(t(strings.shot.plateY), displayDraft[key].y, (value) => changePoint(key, "y", value), `plate-y-${key}`)}{issueList(`plate_quad.${key.replace(/[A-Z]/g, (letter) => `_${letter.toLowerCase()}`)}`)}</fieldset>;
  const saveDraft = () => {
    if (!draftSnapshot || baseRevision === undefined) return;
    save.mutate({ expectedRevision: baseRevision, name: draft.name, snapshot: draftSnapshot, draft });
  };

  return <div className="shot-editor" data-testid="shot-editor">
  <header className="shot-editor__title"><div><span className="studio-eyebrow">{t(strings.shot.workspaceEyebrow).toLocaleUpperCase()}</span><h2 id="shot-workspace-title"><label><span className="sr-only">{t(strings.shot.nameLabel)}</span><input className="shot-title-input" data-testid="shot-name-input" value={displayDraft.name} maxLength={120} onChange={(event) => change("name", event.target.value)} disabled={historical} /></label></h2><p>{t(strings.shot.revision, { revision: head.revision.toString() })}{historical ? ` · ${t(strings.shot.viewingRevision, { revision: selectedHistoricalRevision.toString() })}` : ""}</p></div>
      <details className="shot-history"><summary>{t(strings.shot.pinnedRevision)} · {selectedHistoricalRevision === undefined ? head.revision.toString() : selectedHistoricalRevision.toString()}</summary><label className="shot-field"><span>{t(strings.shot.pinnedRevision)}</span><select aria-label={t(strings.shot.pinnedRevision)} value={selectedHistoricalRevision === undefined ? head.revision.toString() : selectedHistoricalRevision.toString()} onChange={(event) => chooseRevision(BigInt(event.target.value))}>{(revisions.data?.pages.flatMap((page) => page.revisions) ?? [head]).map((revision) => <option key={revision.revision.toString()} value={revision.revision.toString()}>{t(strings.shot.revision, { revision: revision.revision.toString() })} · {revision.name}</option>)}</select></label></details>
    </header>
    {!historical && pinnedRevisionQuery.error && <p role="alert">{errorMessage(pinnedRevisionQuery.error, t)} <Button variant="outline" className="min-h-11" onClick={() => void pinnedRevisionQuery.refetch()}>{t(strings.media.retry)}</Button></p>}
    {historical && <p className="shot-readonly" role="status">{t(strings.shot.historicalReadonly)}</p>}
    {shotQuery.error && <p role="alert">{errorMessage(shotQuery.error, t)}</p>}
    <div className="shot-editor__body">
      <div ref={stageRef} tabIndex={-1} className="shot-stage-focus">
      <Card className="shot-plate-stage">
        <CardHeader><CardTitle as="h3">{showFinalPreview ? t(finalOutput?.isSample ? strings.shot.renderSampleLabel : finalOutput?.isVideo ? strings.shot.renderVideoFinal : strings.shot.renderFinalTitle, { revision: (finalOutput?.revision ?? renderRevision?.revision ?? 0n).toString() }) : t(strings.shot.scenePlate)}</CardTitle><p>{showFinalPreview && finalOutput ? `${finalOutput.metadata.widthPixels} × ${finalOutput.metadata.heightPixels} px` : plate ? `${plate.originalFilename} · ${plate.width} × ${plate.height}` : t(strings.shot.choosePlate)}</p>
          {showFinalPreview && <Button variant="outline" className="min-h-11" onClick={() => { setShowFinalPreview(false); setFinalOutput(undefined); }}>{t(strings.shot.renderHideOutput)}</Button>}
        </CardHeader>
        <CardContent>
          {showFinalPreview && !finalOutput ? <p role="status">{t(strings.shot.renderVerifyingOutput)}</p> : showFinalPreview && finalOutput ? <div className="shot-final-stage" data-testid={selectors.shot.videoFinalStage}>{finalOutput.isVideo ? <video src={finalOutput.url} aria-label={t(strings.shot.renderVideoOutputAlt, { revision: finalOutput.revision.toString() })} controls playsInline preload="metadata"><track kind="captions" srcLang="en" label={t(strings.shot.videoNoCaptions)} src="data:text/vtt,WEBVTT%0A%0ANOTE%20Captions%20are%20not%20available%20for%20this%20preview." /></video> : <img src={finalOutput.url} alt={t(finalOutput.isSample ? strings.shot.renderSampleOutputAlt : strings.shot.renderOutputAlt, { revision: finalOutput.revision.toString() })} />}</div> : <>
          {sources.error && <p role="alert">{errorMessage(sources.error, t)}</p>}
          {plateBytes.error && <p role="alert">{errorMessage(plateBytes.error, t)}</p>}
          {plate && plateUrl ? <div className="shot-plate-image" data-testid="shot-plate-preview"><div className="shot-plate-canvas" style={{ aspectRatio: `${plate.width} / ${plate.height}` }}><img src={plateUrl} alt={t(strings.shot.plateAlt, { name: plate.originalFilename })} />
            <svg viewBox={`0 0 ${plate.width} ${plate.height}`} preserveAspectRatio="none" aria-label={t(strings.shot.calibrationOverlay)}>
              {displayDraft.coordinateConvention === CoordinateConvention.PLATE_RASTER_EDGE_PIXELS_V1 && [displayDraft.topLeft,displayDraft.topRight,displayDraft.bottomRight,displayDraft.bottomLeft].every((point) => point.x !== "" && point.y !== "") && <polygon points={[displayDraft.topLeft,displayDraft.topRight,displayDraft.bottomRight,displayDraft.bottomLeft].map((point) => `${point.x},${point.y}`).join(" ")} />}
              {displayDraft.coordinateConvention === CoordinateConvention.PLATE_RASTER_EDGE_PIXELS_V1 && [displayDraft.topLeft,displayDraft.topRight,displayDraft.bottomRight,displayDraft.bottomLeft].map((point, index) => point.x !== "" && point.y !== "" ? <circle key={index} cx={point.x} cy={point.y} r={Math.max(plate.width, plate.height) * 0.008} /> : null)}
            </svg>
            </div><span>{t(strings.shot.calibrationOverlay)}</span>
          </div> : <div className="shot-plate-empty"><p>{t(strings.shot.choosePlate)}</p><p>{t(strings.shot.plateStageHelp)}</p></div>}
          {plate && <p className="shot-source-caption">{t(strings.shot.originalImage)} · {plate.mediaType} · {bytes(plate.sizeBytes)}</p>}
          </>}
        </CardContent>
      </Card>
      </div>
      <aside className="shot-inspector" aria-label={t(strings.shot.inspector)}>
        <Card><CardHeader><CardTitle as="h3">{t(strings.shot.scene)}</CardTitle></CardHeader><CardContent className="shot-fields"><fieldset disabled={historical}>
          {plateMetadata.error && <p role="alert">{errorMessage(plateMetadata.error, t)}</p>}
          {sourceMetadata.error && <p role="alert">{errorMessage(sourceMetadata.error, t)}</p>}
          {select(t(strings.shot.scenePlate), displayDraft.plateMediaId, (value) => change("plateMediaId", value), availableMedia, t(strings.shot.chooseSource))}
          {select(t(strings.shot.screenSource), displayDraft.sourceMediaId, chooseScreenSource, screenMediaOptions, t(strings.shot.chooseSource))}
          {screenVideo && displayDraft.videoTiming && <VideoTimingEditor projectId={projectId} source={screenVideo} timing={displayDraft.videoTiming} disabled={historical} onChange={(timing) => updateDraft((current) => ({ ...current, videoTiming: timing }))} />}
          <div className="shot-fields">
            <label className="shot-field"><span>{t(strings.shot.motionMode)}</span><select value={displayDraft.motionMode} disabled={historical} onChange={(event) => chooseMotionMode(Number(event.target.value))}>
              {displayDraft.motionMode !== MotionMode.FIXED && displayDraft.motionMode !== MotionMode.CROP_PUSH && <option value={displayDraft.motionMode} disabled>{t(strings.shot.unsupportedMotion)}</option>}
              <option value={MotionMode.FIXED}>{t(strings.shot.fixedMotion)}</option>
              <option value={MotionMode.CROP_PUSH} disabled={!displayDraft.videoTiming && displayDraft.motionMode !== MotionMode.CROP_PUSH}>{t(strings.shot.screenEntry)}</option>
            </select></label>
            {displayDraft.motionMode === MotionMode.FIXED && <p className="shot-help">{t(strings.shot.fixedModeHelp)}</p>}
            {displayDraft.motionMode === MotionMode.CROP_PUSH && <>
              <p className="shot-help">{t(strings.shot.screenEntryContainHelp)}</p>
              <label className="shot-field"><span>{t(strings.shot.cameraProfile)}</span><select value={displayDraft.cameraProfile} disabled={historical} onChange={(event) => change("cameraProfile", event.target.value)}><option value="screen-entry-stationary-scene-v1">{t(strings.shot.cameraStationary)}</option><option value="screen-entry-projective-contain-v1">{t(strings.shot.cameraProjective)}</option></select></label>
              <p className="shot-help">{t(strings.shot.cameraProfileHelp)}</p>
              {displayDraft.cameraProfile === "screen-entry-stationary-scene-v1" && <>
                <label className="shot-field"><span><input type="checkbox" checked={displayDraft.cameraSceneEnabled} disabled={historical} onChange={(event) => change("cameraSceneEnabled", event.target.checked)} /> {t(strings.shot.cameraSceneEnabled)}</span></label>
                {displayDraft.cameraSceneEnabled && <div className="shot-fields shot-fields--pair">{(["cameraSceneX", "cameraSceneY", "cameraSceneWidth", "cameraSceneHeight"] as const).map((key) => <label className="shot-field" key={key}><span>{t(strings.shot[key])}</span><input type="number" min={key === "cameraSceneWidth" || key === "cameraSceneHeight" ? 0.01 : 0} max={100} step={0.01} value={displayDraft[key]} disabled={historical} onChange={(event) => change(key, event.target.value)} /></label>)}</div>}
                {cameraRectInvalid(displayDraft, "Scene") && <p role="alert">{t(strings.shot.cameraTargetInvalid)}</p>}
              </>}
              {!displayDraft.videoTiming && <p role="alert">{t(strings.shot.cameraTimingRequired)}</p>}
              {displayCameraClipInvalid && <p role="alert">{t(strings.shot.cameraClipTooLong)}</p>}
              <label className="shot-field"><span>{t(strings.shot.cameraDurationFrames)}</span><input type="number" min={2} max={displayClipFrames === undefined || displayCameraLimitFrames === undefined ? 360 : Number(displayClipFrames > displayCameraLimitFrames ? displayCameraLimitFrames : displayClipFrames)} step={1} value={displayDraft.cameraDurationOutputFrames} disabled={historical} onChange={(event) => change("cameraDurationOutputFrames", event.target.value)} /></label>
              {displayCameraDurationStatus && <p role="status">{displayCameraDurationStatus}</p>}
              {displayCameraDurationInvalid && displayDraft.videoTiming && <p role="alert">{t(strings.shot.cameraDurationInvalid)}</p>}
              {colorField(t(strings.shot.cameraMatte), displayDraft.cameraMatteColor, (value) => change("cameraMatteColor", value))}
              {displayCameraMatteInvalid && <p role="alert">{t(strings.shot.cameraMatteInvalid)}</p>}
              <label className="shot-field"><span><input type="checkbox" checked={displayDraft.cameraTargetEnabled} disabled={historical} onChange={(event) => change("cameraTargetEnabled", event.target.checked)} /> {t(strings.shot.cameraTargetEnabled)}</span></label>
              <p className="shot-help">{t(strings.shot.cameraTargetHelp)}</p>
              {displayDraft.cameraTargetEnabled && <div className="shot-fields shot-fields--pair">
                {(["cameraTargetX", "cameraTargetY", "cameraTargetWidth", "cameraTargetHeight"] as const).map((key) => <label className="shot-field" key={key}><span>{t(strings.shot[key])}</span><input type="number" min={key === "cameraTargetWidth" || key === "cameraTargetHeight" ? 0.01 : 0} max={100} step={0.01} value={displayDraft[key]} disabled={historical} onChange={(event) => change(key, event.target.value)} /></label>)}
              </div>}
              {cameraRectInvalid(displayDraft, "Target") && <p role="alert">{t(strings.shot.cameraTargetInvalid)}</p>}
            </>}
          </div>
          <details><summary>{t(strings.shot.masksAndGeometry)}</summary><div className="shot-fields">
            {select(t(strings.shot.activeMask), displayDraft.activeMaskMediaId, (value) => change("activeMaskMediaId", value), availableMedia, t(strings.shot.noMask))}
            {select(t(strings.shot.hardwareMask), displayDraft.hardwareMaskMediaId, (value) => change("hardwareMaskMediaId", value), availableMedia, t(strings.shot.noMask))}
            <label className="shot-field"><span>{t(strings.shot.coordinateConvention)}</span><select value={displayDraft.coordinateConvention} onChange={(event) => change("coordinateConvention", Number(event.target.value))}><option value={CoordinateConvention.UNSPECIFIED}>{t(strings.shot.unspecified)}</option><option value={CoordinateConvention.PLATE_RASTER_EDGE_PIXELS_V1}>{t(strings.shot.plateRasterEdge)}</option></select></label>
            {displayDraft.coordinateConvention === CoordinateConvention.UNSPECIFIED && <p role="status" className="shot-help">{t(strings.shot.coordinateWarning)}</p>}
            <label className="shot-field"><span>{t(strings.shot.depictionLabel)}</span><input value={displayDraft.depictionLabel} onChange={(event) => change("depictionLabel", event.target.value)} /></label>
          <div className="shot-fields shot-fields--pair">{numberField(t(strings.shot.displayWidth), displayDraft.displayWidthPixels, (value) => change("displayWidthPixels", value))}{numberField(t(strings.shot.displayHeight), displayDraft.displayHeightPixels, (value) => change("displayHeightPixels", value))}</div>
            <p className="shot-help">{t(strings.shot.platePixelCoordinates)}</p>
            <div className="shot-quad-grid">{([ ["topLeft", t(strings.shot.topLeft)], ["topRight", t(strings.shot.topRight)], ["bottomRight", t(strings.shot.bottomRight)], ["bottomLeft", t(strings.shot.bottomLeft)] ] as ["topLeft" | "topRight" | "bottomRight" | "bottomLeft", string][]).map(pointFields)}</div>
            {issueList("scene")}
          </div></details>
          <details><summary>{t(strings.shot.screenPresentation)}</summary>
            <ScreenPresentationEditor value={displayDraft} disabled={historical} onChange={(patch) => updateDraft((current) => ({ ...current, ...patch }))} />
            {issueList("presentation")}
          </details>
          {issueList()}
          {numericInvalid && !historical && <p role="alert">{t(strings.shot.numericIssue)}</p>}
          {validate.error && <p role="alert">{errorMessage(validate.error, t)}</p>}
          {save.error && <div role="alert" className="shot-conflict"><p>{conflict ? t(strings.shot.conflict) : errorMessage(save.error, t)}</p>{conflict && <Button variant="outline" className="min-h-11" onClick={() => void reapplyDraftOnLatest()}>{t(strings.shot.reloadKeepDraft)}</Button>}</div>}
          {save.data?.revision && <p role="status">{t(strings.shot.savedRevision, { revision: save.data.revision.revision.toString() })}</p>}
          <div className="shot-actions"><Button variant="outline" className="min-h-11" disabled={validate.isPending || historical || numericInvalid || cameraMatteInvalid || presentationInvalid || selectedMediaPending} onClick={() => validate.mutate()}>{validate.isPending ? t(strings.shot.checking) : t(strings.shot.checkDraft)}</Button><Button className="studio-primary min-h-11" disabled={save.isPending || historical || numericInvalid || cameraMatteInvalid || presentationInvalid || selectedMediaPending || baseRevision === undefined || !draft.name.trim()} onClick={saveDraft}>{save.isPending ? t(strings.shot.saving) : t(strings.shot.save)}</Button></div>
        </fieldset></CardContent></Card>
        <Card><CardHeader><CardTitle as="h3">{t(strings.shot.screenSource)}</CardTitle></CardHeader><CardContent>
          {screenSource ? <p className="shot-source-choice">{screenSource.originalFilename}<span>{screenSource.width !== undefined && screenSource.height !== undefined ? `${screenSource.width} × ${screenSource.height}` : screenSource.mediaType}</span></p> : <p className="shot-help">{t(strings.shot.noScreenSource)}</p>}
          {screenBytes.error && <p role="alert">{errorMessage(screenBytes.error, t)}</p>}
          {screenUrl && screenImage && <img className="shot-source-thumb" data-testid="shot-screen-preview" src={screenUrl} alt={t(strings.shot.screenSourceAlt, { name: screenImage.originalFilename })} />}
          {screenVideo && <><video className="shot-source-thumb" data-testid={selectors.shot.videoSourcePreview} src={sourceMediaContentURL(projectId, screenVideo.id)} controls preload="metadata" playsInline muted aria-label={t(strings.media.videoPreviewAlt, { name: screenVideo.originalFilename })} onError={() => setSourcePlaybackFailed(true)} /><p className="shot-help">{t(strings.media.videoPreviewHelp)}</p>{sourcePlaybackFailed && <p role="alert">{t(strings.media.videoPreviewError)}</p>}</>}
          {sources.hasNextPage && <Button variant="outline" className="min-h-11" disabled={sources.isFetchingNextPage} onClick={() => void sources.fetchNextPage()}>{t(strings.projects.loadMore)}</Button>}
        </CardContent></Card>
        <RenderControls projectId={projectId} shotId={shotId} revision={renderRevision} historical={historical} hasUnsavedChanges={hasUnsavedChanges} showOutput={showFinalPreview} onShowOutputChange={setShowFinalPreview} onFinalOutputChange={onFinalOutputChange} onAcceptVideoSample={acceptVideoSample} onDeclareReadabilityProof={declareReadabilityProof} />
      </aside>
    </div>
  </div>;
}
