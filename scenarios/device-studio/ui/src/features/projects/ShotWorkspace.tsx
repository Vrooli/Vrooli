import { create } from "@bufbuild/protobuf";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";
import {
  CaptureArea,
  CoordinateConvention,
  MotionMode,
  PhotographicSceneSchema,
  PlateQuadSchema,
  PhoneSystemChromeSchema,
  BrowserDesktopChromeSchema,
  PresentationKind,
  ScreenFitMode,
  ScreenPresentationSchema,
  ShotIssueCode,
  ShotSnapshotSchema,
  type ShotRevision,
} from "@vrooli/proto-types/device-studio/v1/shots/shots_pb";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { getSourceMediaContent, projectsClient, verifySourceMediaContent, type SourceMedia } from "../../api/projects";
import { shotConflict, shotsClient } from "../../api/shots";

type PointDraft = { x: string; y: string };
type ShotDraft = {
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
  sourceMediaId: string;
  captureArea: CaptureArea;
  kind: PresentationKind;
  fitMode: ScreenFitMode;
  carrierText: string;
  clockText: string;
  batteryPercent: string;
  showCellularIndicator: boolean;
  showWifiIndicator: boolean;
  showHomeIndicator: boolean;
  windowTitle: string;
  addressText: string;
  showWindowControls: boolean;
  showToolbar: boolean;
};
type ShotDraftBuffer = { draft: ShotDraft; baseRevision: bigint };

const emptyPoint = (): PointDraft => ({ x: "", y: "" });
const emptyDraft = (name = "") : ShotDraft => ({
  name, plateMediaId: "", activeMaskMediaId: "", hardwareMaskMediaId: "",
  displayWidthPixels: "", displayHeightPixels: "", topLeft: emptyPoint(), topRight: emptyPoint(), bottomRight: emptyPoint(), bottomLeft: emptyPoint(),
  depictionLabel: "", coordinateConvention: CoordinateConvention.PLATE_RASTER_EDGE_PIXELS_V1, motionMode: MotionMode.FIXED,
  sourceMediaId: "", captureArea: CaptureArea.APP_ONLY, kind: PresentationKind.PLAIN, fitMode: ScreenFitMode.CONTAIN,
  carrierText: "", clockText: "9:41", batteryPercent: "100",
  showCellularIndicator: true, showWifiIndicator: true, showHomeIndicator: true, windowTitle: "", addressText: "",
  showWindowControls: true, showToolbar: true,
});

function snapshotFromDraft(draft: ShotDraft) {
  const point = (value: PointDraft) => value.x !== "" && value.y !== "" ? { x: Number(value.x), y: Number(value.y) } : undefined;
  const quadPoints = {
    topLeft: point(draft.topLeft), topRight: point(draft.topRight),
    bottomRight: point(draft.bottomRight), bottomLeft: point(draft.bottomLeft),
  };
  const plateQuad = Object.values(quadPoints).some((value) => value !== undefined)
    ? create(PlateQuadSchema, quadPoints)
    : undefined;
  const phoneSystemChrome = create(PhoneSystemChromeSchema, {
    carrierText: draft.carrierText, clockText: draft.clockText,
    batteryPercent: Number(draft.batteryPercent || 0), showCellularIndicator: draft.showCellularIndicator,
    showWifiIndicator: draft.showWifiIndicator, showHomeIndicator: draft.showHomeIndicator,
  });
  const browserDesktopChrome = create(BrowserDesktopChromeSchema, {
    windowTitle: draft.windowTitle, addressText: draft.addressText,
    showWindowControls: draft.showWindowControls, showToolbar: draft.showToolbar,
  });
  return create(ShotSnapshotSchema, {
    motionMode: draft.motionMode,
    scene: create(PhotographicSceneSchema, {
      plateMediaId: draft.plateMediaId, activeMaskMediaId: draft.activeMaskMediaId,
      hardwareMaskMediaId: draft.hardwareMaskMediaId,
      displayWidthPixels: Number(draft.displayWidthPixels || 0), displayHeightPixels: Number(draft.displayHeightPixels || 0),
      plateQuad, coordinateConvention: draft.coordinateConvention, depictionLabel: draft.depictionLabel,
    }),
    presentation: create(ScreenPresentationSchema, {
      sourceMediaId: draft.sourceMediaId, captureArea: draft.captureArea, kind: draft.kind, fitMode: draft.fitMode,
      viewportWidthPixels: 0, viewportHeightPixels: 0,
      phoneSystemChrome: draft.kind === PresentationKind.PHONE_SYSTEM ? phoneSystemChrome : undefined,
      browserDesktopChrome: draft.kind === PresentationKind.BROWSER_DESKTOP ? browserDesktopChrome : undefined,
    }),
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
    sourceMediaId: presentation?.sourceMediaId ?? "", captureArea: presentation?.captureArea ?? CaptureArea.APP_ONLY, kind: presentation?.kind ?? PresentationKind.PLAIN,
    fitMode: presentation?.fitMode ?? ScreenFitMode.CONTAIN,
    carrierText: presentation?.phoneSystemChrome?.carrierText ?? "", clockText: presentation?.phoneSystemChrome?.clockText ?? "9:41",
    batteryPercent: presentation?.phoneSystemChrome ? String(presentation.phoneSystemChrome.batteryPercent) : "100",
    showCellularIndicator: presentation?.phoneSystemChrome?.showCellularIndicator ?? true,
    showWifiIndicator: presentation?.phoneSystemChrome?.showWifiIndicator ?? true,
    showHomeIndicator: presentation?.phoneSystemChrome?.showHomeIndicator ?? true,
    windowTitle: presentation?.browserDesktopChrome?.windowTitle ?? "", addressText: presentation?.browserDesktopChrome?.addressText ?? "",
    showWindowControls: presentation?.browserDesktopChrome?.showWindowControls ?? true,
    showToolbar: presentation?.browserDesktopChrome?.showToolbar ?? true,
  };
}

function bytes(value: bigint) { return `${(Number(value) / 1024 / 1024).toFixed(2)} MiB`; }

export function ShotWorkspace({ projectId }: { projectId: string }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [selectedShotId, setSelectedShotId] = useState("");
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
    {selectedShotId ? <ShotEditor key={selectedShotId} projectId={projectId} shotId={selectedShotId} mediaQueryKey={["project-source-media", projectId]} draftCache={drafts[selectedShotId]} onDraftChange={cacheDraft} /> :
      <p className="shot-workspace__empty">{records.length ? t(strings.shot.chooseSaved) : t(strings.shot.empty)}</p>}
    <details className="shot-management" open={shots.data !== undefined && records.length === 0}>
      <summary>{t(strings.shot.savedShots)} · {records.length}</summary>
      <div className="shot-management__content">
        <form className="shot-create" onSubmit={(event) => { event.preventDefault(); const name = newName.trim(); if (name) createShot.mutate(name); }}>
          <label className="sr-only" htmlFor="new-shot-name">{t(strings.shot.nameLabel)}</label>
          <input id="new-shot-name" data-testid="new-shot-name" value={newName} onChange={(event) => setNewName(event.target.value)} placeholder={t(strings.shot.newNamePlaceholder)} maxLength={120} />
          <Button type="submit" className="studio-primary min-h-11" disabled={!newName.trim() || createShot.isPending}>{t(strings.shot.create)}</Button>
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
  const viewDraft = historical && selectedRevision ? draftFromRevision(selectedRevision.name, selectedRevision) : draft;
  const sources = useInfiniteQuery({ queryKey: mediaQueryKey, initialPageParam: "", queryFn: ({ pageParam }) => projectsClient.listSourceMedia({ projectId, pageSize: 25, pageToken: pageParam }), getNextPageParam: (page) => page.nextPageToken || undefined });
  const media = sources.data?.pages.flatMap((page) => page.sourceMedia) ?? [];
  const listedPlate = media.find((item) => item.id === viewDraft?.plateMediaId);
  const plateMetadata = useQuery({ queryKey: ["project-source-media-item", projectId, viewDraft?.plateMediaId], queryFn: () => projectsClient.getSourceMedia({ projectId, id: viewDraft?.plateMediaId ?? "" }), enabled: Boolean(viewDraft?.plateMediaId && !listedPlate), gcTime: 0 });
  const plate = listedPlate ?? plateMetadata.data?.sourceMedia;
  const listedSource = media.find((item) => item.id === viewDraft?.sourceMediaId);
  const sourceMetadata = useQuery({ queryKey: ["project-source-media-item", projectId, viewDraft?.sourceMediaId], queryFn: () => projectsClient.getSourceMedia({ projectId, id: viewDraft?.sourceMediaId ?? "" }), enabled: Boolean(viewDraft?.sourceMediaId && !listedSource), gcTime: 0 });
  const screenSource = listedSource ?? sourceMetadata.data?.sourceMedia;
  const plateBytes = useQuery({
    queryKey: ["shot-plate-content", projectId, plate?.id, plate?.sha256],
    queryFn: async () => { if (!plate) throw new Error("No scene plate selected"); return verifySourceMediaContent(await getSourceMediaContent(projectId, plate.id), plate); },
    enabled: Boolean(plate), gcTime: 0, retry: false,
  });
  const screenBytes = useQuery({
    queryKey: ["shot-screen-content", projectId, screenSource?.id, screenSource?.sha256],
    queryFn: async () => { if (!screenSource) throw new Error("No screen source selected"); return verifySourceMediaContent(await getSourceMediaContent(projectId, screenSource.id), screenSource); },
    enabled: Boolean(screenSource), gcTime: 0, retry: false,
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
  const validate = useMutation({
    mutationFn: () => { if (!draftSnapshot) throw new Error("Shot is not loaded"); return shotsClient.validateShot({ projectId, snapshot: draftSnapshot }); },
    onSuccess: (result) => setIssues(result.issues),
  });
  const save = useMutation({
    mutationFn: () => { if (!draftSnapshot || baseRevision === undefined || !draft) throw new Error("Shot revision is not loaded"); return shotsClient.saveShot({ shotId, expectedRevision: baseRevision, name: draft.name, snapshot: draftSnapshot }); },
    onSuccess: async (result) => {
      if (result.revision && draft) { setBaseRevision(result.revision.revision); setSelectedHistoricalRevision(undefined); setIssues(result.revision.issues); onDraftChange({ draft, baseRevision: result.revision.revision }); }
      if (result.shot) queryClient.setQueryData(["shot", shotId], { shot: result.shot, head: result.revision });
      await queryClient.invalidateQueries({ queryKey: ["project-shots", projectId] });
      await queryClient.invalidateQueries({ queryKey: ["shot-revisions", shotId] });
    },
  });
  const conflict = shotConflict(save.error);
  const updateDraft = (changeDraft: (current: ShotDraft) => ShotDraft) => {
    if (!draft) return;
    const next = changeDraft(draft);
    setDraft(next);
    if (baseRevision !== undefined) onDraftChange({ draft: next, baseRevision });
    setIssues([]);
    validate.reset();
  };
  const change = <K extends keyof ShotDraft>(key: K, value: ShotDraft[K]) => updateDraft((current) => ({ ...current, [key]: value }));
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
  const displayDraft = viewDraft ?? draft;
  const integerValues = [draft.displayWidthPixels, draft.displayHeightPixels, draft.batteryPercent].filter(Boolean).map(Number);
  const pointDrafts = [draft.topLeft, draft.topRight, draft.bottomRight, draft.bottomLeft];
  const partialPoint = pointDrafts.some((point) => (point.x === "") !== (point.y === ""));
  const coordinateOutOfRange = pointDrafts.some((point) => [point.x, point.y].some((raw, axis) => {
    if (raw === "") return false;
    const value = Number(raw);
    return !Number.isFinite(value) || Math.abs(value) > Number.MAX_SAFE_INTEGER || value < 0 || Boolean(plate && value > (axis === 0 ? plate.width : plate.height));
  }));
  const numericInvalid = integerValues.some((value) => !Number.isFinite(value) || !Number.isInteger(value) || value < 0 || value > 4_294_967_295) || Number(draft.batteryPercent || 0) > 100 || partialPoint || coordinateOutOfRange;
  const selectedMediaPending = Boolean((displayDraft.plateMediaId && !plate && plateMetadata.isLoading) || (displayDraft.sourceMediaId && !screenSource && sourceMetadata.isLoading));
  const availableMedia = [...media, ...(plate && !media.some((item) => item.id === plate.id) ? [plate] : []), ...(screenSource && !media.some((item) => item.id === screenSource.id) && screenSource.id !== plate?.id ? [screenSource] : [])];
  const select = (label: string, value: string, onChange: (value: string) => void, options: SourceMedia[], emptyLabel: string) => <label className="shot-field"><span>{label}</span><select value={value} onChange={(event) => onChange(event.target.value)}><option value="">{emptyLabel}</option>{options.map((item) => <option key={item.id} value={item.id}>{item.originalFilename} · {item.width} × {item.height}</option>)}</select></label>;
  const numberField = (label: string, value: string, onChange: (value: string) => void, testId?: string) => <label className="shot-field"><span>{label}</span><input data-testid={testId} type="number" inputMode="decimal" value={value} onChange={(event) => onChange(event.target.value)} /></label>;
  const activeIssues = historical ? selectedRevision?.issues ?? [] : issues;
  const issuesFor = (path: string) => activeIssues.filter((issue) => !path || issue.fieldPath.toLowerCase().includes(path.toLowerCase()));
  const issueList = (path = "") => { const rows = path ? issuesFor(path) : activeIssues.filter((issue) => !issue.fieldPath.startsWith("scene.") && !issue.fieldPath.startsWith("presentation.")); return rows.length ? <ul className="shot-issues" aria-live="polite">{rows.map((issue, index) => <li key={`${issue.code}-${issue.fieldPath}-${index}`}><span>{issue.summary}</span>{issue.repair && <small>{issue.repair}</small>}<details><summary>{t(strings.shot.issueDetails)}</summary><code>{issue.fieldPath}</code><span>{ShotIssueCode[issue.code].replace(/_/g, " ")}</span></details></li>)}</ul> : null; };
  const pointFields = ([key, label]: ["topLeft" | "topRight" | "bottomRight" | "bottomLeft", string]) => <fieldset key={key} className="shot-point"><legend>{label}</legend>{numberField(t(strings.shot.plateX), displayDraft[key].x, (value) => changePoint(key, "x", value), `plate-x-${key}`)}{numberField(t(strings.shot.plateY), displayDraft[key].y, (value) => changePoint(key, "y", value), `plate-y-${key}`)}{issueList(`plate_quad.${key.replace(/[A-Z]/g, (letter) => `_${letter.toLowerCase()}`)}`)}</fieldset>;

  return <div className="shot-editor" data-testid="shot-editor">
    <header className="shot-editor__title"><div><span className="studio-eyebrow">{t(strings.shot.workspaceEyebrow).toLocaleUpperCase()}</span><h2 id="shot-workspace-title"><label><span className="sr-only">{t(strings.shot.nameLabel)}</span><input className="shot-title-input" data-testid="shot-name-input" value={displayDraft.name} maxLength={120} onChange={(event) => change("name", event.target.value)} disabled={historical} /></label></h2><p>{t(strings.shot.revision, { revision: head.revision.toString() })}{historical ? ` · ${t(strings.shot.viewingRevision, { revision: selectedHistoricalRevision.toString() })}` : ""}</p></div>
      <details className="shot-history"><summary>{t(strings.shot.pinnedRevision)} · {selectedHistoricalRevision === undefined ? head.revision.toString() : selectedHistoricalRevision.toString()}</summary><label className="shot-field"><span>{t(strings.shot.pinnedRevision)}</span><select aria-label={t(strings.shot.pinnedRevision)} value={selectedHistoricalRevision === undefined ? head.revision.toString() : selectedHistoricalRevision.toString()} onChange={(event) => chooseRevision(BigInt(event.target.value))}>{(revisions.data?.pages.flatMap((page) => page.revisions) ?? [head]).map((revision) => <option key={revision.revision.toString()} value={revision.revision.toString()}>{t(strings.shot.revision, { revision: revision.revision.toString() })} · {revision.name}</option>)}</select></label></details>
    </header>
    {historical && <p className="shot-readonly" role="status">{t(strings.shot.historicalReadonly)}</p>}
    {shotQuery.error && <p role="alert">{errorMessage(shotQuery.error, t)}</p>}
    <div className="shot-editor__body">
      <Card className="shot-plate-stage">
        <CardHeader><CardTitle as="h3">{t(strings.shot.scenePlate)}</CardTitle><p>{plate ? `${plate.originalFilename} · ${plate.width} × ${plate.height}` : t(strings.shot.choosePlate)}</p></CardHeader>
        <CardContent>
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
        </CardContent>
      </Card>
      <aside className="shot-inspector" aria-label={t(strings.shot.inspector)}>
        <Card><CardHeader><CardTitle as="h3">{t(strings.shot.scene)}</CardTitle></CardHeader><CardContent className="shot-fields"><fieldset disabled={historical}>
          {plateMetadata.error && <p role="alert">{errorMessage(plateMetadata.error, t)}</p>}
          {sourceMetadata.error && <p role="alert">{errorMessage(sourceMetadata.error, t)}</p>}
          {select(t(strings.shot.scenePlate), displayDraft.plateMediaId, (value) => change("plateMediaId", value), availableMedia, t(strings.shot.chooseSource))}
          {select(t(strings.shot.screenSource), displayDraft.sourceMediaId, (value) => change("sourceMediaId", value), availableMedia, t(strings.shot.chooseSource))}
          <p className="shot-help">{t(strings.shot.motionMode)}: {MotionMode[displayDraft.motionMode].replace(/_/g, " ") || t(strings.shot.fixedMotion)}. {displayDraft.motionMode === MotionMode.FIXED ? t(strings.shot.fixedModeHelp) : t(strings.shot.unsupportedMotion)}</p>
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
          <details><summary>{t(strings.shot.screenPresentation)}</summary><div className="shot-fields">
            <label className="shot-field"><span>{t(strings.shot.captureArea)}</span><select value={displayDraft.captureArea} onChange={(event) => change("captureArea", Number(event.target.value))}><option value={CaptureArea.UNSPECIFIED}>{t(strings.shot.unspecified)}</option><option value={CaptureArea.APP_ONLY}>{t(strings.shot.appOnly)}</option><option value={CaptureArea.ALREADY_COMPOSED}>{t(strings.shot.alreadyComposed)}</option></select></label>
            <label className="shot-field"><span>{t(strings.shot.presentationKind)}</span><select value={displayDraft.kind} onChange={(event) => change("kind", Number(event.target.value))}><option value={PresentationKind.UNSPECIFIED}>{t(strings.shot.unspecified)}</option><option value={PresentationKind.PLAIN}>{t(strings.shot.plainCapture)}</option><option value={PresentationKind.PHONE_SYSTEM}>{t(strings.shot.phoneSystem)}</option><option value={PresentationKind.BROWSER_DESKTOP}>{t(strings.shot.browserDesktop)}</option></select></label>
            <label className="shot-field"><span>{t(strings.shot.fitMode)}</span><select value={displayDraft.fitMode} onChange={(event) => change("fitMode", Number(event.target.value))}><option value={ScreenFitMode.UNSPECIFIED}>{t(strings.shot.unspecified)}</option><option value={ScreenFitMode.CONTAIN}>{t(strings.shot.contain)}</option><option value={ScreenFitMode.COVER}>{t(strings.shot.cover)}</option></select></label>
            {displayDraft.kind === PresentationKind.PHONE_SYSTEM && <details><summary>{t(strings.shot.phoneChrome)}</summary><div className="shot-fields">
              <label className="shot-field"><span>{t(strings.shot.carrier)}</span><input value={displayDraft.carrierText} onChange={(event) => change("carrierText", event.target.value)} /></label>
              <label className="shot-field"><span>{t(strings.shot.clock)}</span><input value={displayDraft.clockText} onChange={(event) => change("clockText", event.target.value)} /></label>
              {numberField(t(strings.shot.battery), displayDraft.batteryPercent, (value) => change("batteryPercent", value))}
              {([ ["showCellularIndicator", t(strings.shot.cellular)], ["showWifiIndicator", t(strings.shot.wifi)], ["showHomeIndicator", t(strings.shot.homeIndicator)] ] as ["showCellularIndicator" | "showWifiIndicator" | "showHomeIndicator", string][]).map(([key,label]) => <label className="shot-check" key={key}><input type="checkbox" checked={displayDraft[key]} onChange={(event) => change(key, event.target.checked)} />{label}</label>)}
            </div></details>}
            {displayDraft.kind === PresentationKind.BROWSER_DESKTOP && <details><summary>{t(strings.shot.browserChrome)}</summary><div className="shot-fields">
              <label className="shot-field"><span>{t(strings.shot.windowTitle)}</span><input value={displayDraft.windowTitle} onChange={(event) => change("windowTitle", event.target.value)} /></label>
              <label className="shot-field"><span>{t(strings.shot.addressText)}</span><input value={displayDraft.addressText} onChange={(event) => change("addressText", event.target.value)} /></label>
              {([ ["showWindowControls", t(strings.shot.windowControls)], ["showToolbar", t(strings.shot.toolbar)] ] as ["showWindowControls" | "showToolbar", string][]).map(([key,label]) => <label className="shot-check" key={key}><input type="checkbox" checked={displayDraft[key]} onChange={(event) => change(key, event.target.checked)} />{label}</label>)}
            </div></details>}
            {issueList("presentation")}
          </div></details>
          {issueList()}
          {numericInvalid && <p role="alert">{t(strings.shot.numericIssue)}</p>}
          {validate.error && <p role="alert">{errorMessage(validate.error, t)}</p>}
          {save.error && <div role="alert" className="shot-conflict"><p>{conflict ? t(strings.shot.conflict) : errorMessage(save.error, t)}</p>{conflict && <Button variant="outline" className="min-h-11" onClick={() => void reapplyDraftOnLatest()}>{t(strings.shot.reloadKeepDraft)}</Button>}</div>}
          {save.data?.revision && <p role="status">{t(strings.shot.savedRevision, { revision: save.data.revision.revision.toString() })}</p>}
          <div className="shot-actions"><Button variant="outline" className="min-h-11" disabled={validate.isPending || historical || numericInvalid || selectedMediaPending} onClick={() => validate.mutate()}>{validate.isPending ? t(strings.shot.checking) : t(strings.shot.checkDraft)}</Button><Button className="studio-primary min-h-11" disabled={save.isPending || historical || numericInvalid || selectedMediaPending || baseRevision === undefined || !draft.name.trim()} onClick={() => save.mutate()}>{save.isPending ? t(strings.shot.saving) : t(strings.shot.save)}</Button></div>
        </fieldset></CardContent></Card>
        <Card><CardHeader><CardTitle as="h3">{t(strings.shot.screenSource)}</CardTitle></CardHeader><CardContent>
          {screenSource ? <p className="shot-source-choice">{screenSource.originalFilename}<span>{screenSource.width} × {screenSource.height}</span></p> : <p className="shot-help">{t(strings.shot.noScreenSource)}</p>}
          {screenBytes.error && <p role="alert">{errorMessage(screenBytes.error, t)}</p>}
          {screenUrl && screenSource && <img className="shot-source-thumb" data-testid="shot-screen-preview" src={screenUrl} alt={t(strings.shot.screenSourceAlt, { name: screenSource.originalFilename })} />}
          {sources.hasNextPage && <Button variant="outline" className="min-h-11" disabled={sources.isFetchingNextPage} onClick={() => void sources.fetchNextPage()}>{t(strings.projects.loadMore)}</Button>}
        </CardContent></Card>
      </aside>
    </div>
  </div>;
}
