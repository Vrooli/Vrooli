import { useEffect, useState } from "react";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";
import { MediaReadiness, SourceMediaKind } from "@vrooli/proto-types/device-studio/v1/projects/projects_pb";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { getSourceMediaContent, projectsClient, sourceMediaContentURL, uploadSourceMedia, verifySourceMediaContent, type SourceMedia } from "../../api/projects";

const activeInspectionStates = new Set(["queued", "running", "checkpointed", "cancel_requested"]);
const terminalInspectionStates = new Set(["succeeded", "failed", "cancelled", "canceled"]);

function inspectionIsActive(media: SourceMedia): boolean {
  return media.kind === SourceMediaKind.VIDEO && media.readiness === MediaReadiness.PENDING &&
    Boolean(media.inspectionJob && activeInspectionStates.has(media.inspectionJob.state.toLowerCase()));
}

function inspectionNeedsProjectionRefresh(media: SourceMedia): boolean {
  if (inspectionIsActive(media)) return true;
  return media.kind === SourceMediaKind.VIDEO && media.readiness === MediaReadiness.PENDING &&
    Boolean(media.inspectionJob && terminalInspectionStates.has(media.inspectionJob.state.toLowerCase()));
}

function readinessKey(readiness: MediaReadiness) {
  switch (readiness) {
    case MediaReadiness.READY: return strings.media.readinessReady;
    case MediaReadiness.PENDING: return strings.media.readinessPending;
    case MediaReadiness.REJECTED: return strings.media.readinessRejected;
    case MediaReadiness.RECONCILIATION_REQUIRED: return strings.media.readinessUncertain;
    default: return strings.media.readinessUnknown;
  }
}

function canRetryInspection(media: SourceMedia): boolean {
  return media.readiness === MediaReadiness.REJECTED && Boolean(media.inspectionJob &&
    ["failed", "cancelled", "canceled"].includes(media.inspectionJob.state.toLowerCase()) &&
    ["inspection_execution_failed", "inspection_execution_cancelled"].includes(media.inspectionIssueCode));
}

function canCancelInspection(media: SourceMedia): boolean {
  return inspectionIsActive(media) && Boolean(media.inspectionJob &&
    ["queued", "running", "checkpointed"].includes(media.inspectionJob.state.toLowerCase()));
}

function formatBytes(value: bigint): string {
  return `${(Number(value) / 1024 / 1024).toFixed(2)} MiB`;
}

export function SourceMediaPanel({ projectId }: { projectId: string }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [file, setFile] = useState<File>();
  const [selectedId, setSelectedId] = useState("");
  const list = useInfiniteQuery({
    queryKey: ["project-source-media", projectId],
    initialPageParam: "",
    queryFn: ({ pageParam }) => projectsClient.listSourceMedia({ projectId, pageSize: 25, pageToken: pageParam }),
    getNextPageParam: (page) => page.nextPageToken || undefined,
    enabled: Boolean(projectId),
    refetchInterval: (query) => query.state.data?.pages.some((page) => page.sourceMedia.some(inspectionNeedsProjectionRefresh)) ? 2000 : false,
    refetchIntervalInBackground: false,
  });
  const media = list.data?.pages.flatMap((page) => page.sourceMedia) ?? [];
  const upload = useMutation({
    mutationFn: () => {
      if (!file) throw new Error(t(strings.media.chooseFile));
      return uploadSourceMedia(projectId, file);
    },
    onSuccess: async (saved) => {
      setSelectedId(saved.id);
      queryClient.setQueryData(["project-source-media-item", projectId, saved.id], { sourceMedia: saved });
      setFile(undefined);
      const input = document.getElementById(selectors.media.fileInput) as HTMLInputElement | null;
      if (input) input.value = "";
      await queryClient.invalidateQueries({ queryKey: ["project-source-media", projectId] });
    },
  });
  const selectedQuery = useQuery({
    queryKey: ["project-source-media-item", projectId, selectedId],
    queryFn: () => projectsClient.getSourceMedia({ projectId, id: selectedId }),
    enabled: Boolean(selectedId),
    refetchInterval: (query) => query.state.data?.sourceMedia && inspectionNeedsProjectionRefresh(query.state.data.sourceMedia) ? 2000 : false,
    refetchIntervalInBackground: false,
  });
  const selected = selectedQuery.data?.sourceMedia ?? media.find((item) => item.id === selectedId);
  const [previewError, setPreviewError] = useState(false);
  useEffect(() => setPreviewError(false), [selectedId]);
  const inspectionAction = useMutation({
    mutationFn: async (variables: { action: "ensure" | "retry" | "cancel"; projectId: string; mediaId: string; expectedJobVersion?: bigint }) => {
      if (variables.action === "ensure") return projectsClient.ensureSourceMediaInspection({ projectId: variables.projectId, mediaId: variables.mediaId });
      if (variables.expectedJobVersion === undefined) throw new Error("Source inspection has no current job version");
      if (variables.action === "retry") return projectsClient.retrySourceMediaInspection({ projectId: variables.projectId, mediaId: variables.mediaId, expectedJobVersion: variables.expectedJobVersion });
      return projectsClient.cancelSourceMediaInspection({ projectId: variables.projectId, mediaId: variables.mediaId, expectedJobVersion: variables.expectedJobVersion });
    },
    onSuccess: async (result, variables) => {
      if (result.sourceMedia) queryClient.setQueryData(["project-source-media-item", variables.projectId, result.sourceMedia.id], { sourceMedia: result.sourceMedia });
      await queryClient.invalidateQueries({ queryKey: ["project-source-media", variables.projectId] });
    },
    onError: async (_error, variables) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["project-source-media-item", variables.projectId, variables.mediaId] }),
        queryClient.invalidateQueries({ queryKey: ["project-source-media", variables.projectId] }),
      ]);
    },
  });
  const inspectionActionError = inspectionAction.variables?.projectId === projectId && inspectionAction.variables.mediaId === selected?.id ? inspectionAction.error : null;
  const pendingInspectionAction = inspectionAction.variables && inspectionAction.isPending && inspectionAction.variables.projectId === projectId && inspectionAction.variables.mediaId === selected?.id ? inspectionAction.variables.action : undefined;
  const runInspectionAction = (action: "ensure" | "retry" | "cancel") => {
    if (!selected) return;
    inspectionAction.mutate({
      action,
      projectId,
      mediaId: selected.id,
      expectedJobVersion: action === "ensure" ? undefined : selected.inspectionJob?.version,
    });
  };
  const content = useQuery({
    queryKey: ["project-source-media-content", projectId, selected?.id, selected?.sha256],
    queryFn: async () => {
      if (!selected) throw new Error(t(strings.media.selectToPreview));
      const blob = await getSourceMediaContent(projectId, selected.id);
      return verifySourceMediaContent(blob, selected);
    },
    enabled: selected?.kind === SourceMediaKind.IMAGE,
    retry: false,
    gcTime: 0,
  });
  const [previewUrl, setPreviewUrl] = useState("");
  useEffect(() => {
    if (!content.data) { setPreviewUrl(""); return; }
    const url = URL.createObjectURL(content.data);
    setPreviewUrl(url);
    return () => URL.revokeObjectURL(url);
  }, [content.data]);

  return <section className="source-media-layout grid grid-cols-1 gap-space-md lg:grid-cols-[minmax(0,1.65fr)_minmax(17rem,1fr)]" data-testid={selectors.media.panel}>
    <Card className="source-media-stage min-w-0" data-testid={selectors.media.preview}>
      <CardHeader><CardTitle as="h3">{t(strings.media.sourcePreview)}</CardTitle><p className="text-sm text-app-muted-foreground">{selected?.kind === SourceMediaKind.VIDEO ? t(strings.media.videoSourcePreviewDescription) : t(strings.media.sourcePreviewDescription)}</p></CardHeader>
      <CardContent className="flex min-h-[22rem] flex-col gap-space-md">
        {selected?.kind === SourceMediaKind.IMAGE && content.isLoading && <p role="status">{t(strings.media.loadingContent)}</p>}
        {selected?.kind === SourceMediaKind.IMAGE && content.error && <div role="alert" className="text-sm"><p>{errorMessage(content.error, t)}</p><Button variant="outline" className="mt-2 min-h-11" onClick={() => { void content.refetch(); }}>{t(strings.media.retry)}</Button></div>}
        {selectedQuery.isLoading && selectedId && <p role="status">{t(strings.media.loading)}</p>}
        {selectedQuery.error && <div role="alert" className="text-sm"><p>{errorMessage(selectedQuery.error, t)}</p><Button variant="outline" className="mt-2 min-h-11" onClick={() => { void selectedQuery.refetch(); }}>{t(strings.media.retry)}</Button></div>}
        {!selected && !selectedQuery.isLoading && !selectedQuery.error && <div className="grid min-h-[19rem] flex-1 place-items-center rounded-panel border border-dashed border-app-border bg-app-muted px-space-md text-center text-sm text-app-muted-foreground">{t(strings.media.emptyStage)}</div>}
        {selected?.kind === SourceMediaKind.VIDEO && <div className="flex flex-col gap-space-sm">
          <video data-testid={selectors.media.videoPreview} aria-label={t(strings.media.videoPreviewAlt, { name: selected.originalFilename })} className="max-h-[min(70vh,48rem)] min-h-[19rem] w-full flex-1 rounded-control bg-black object-contain" src={sourceMediaContentURL(projectId, selected.id)} controls preload="metadata" playsInline muted onError={() => setPreviewError(true)} onLoadedMetadata={() => setPreviewError(false)} />
          {previewError ? <p role="alert">{t(strings.media.videoPreviewError)}</p> : <p className="text-sm text-app-muted-foreground">{t(strings.media.videoPreviewHelp)}</p>}
        </div>}
        {selected?.kind === SourceMediaKind.IMAGE && content.data && previewUrl && <img data-testid={selectors.media.image} src={previewUrl} alt={t(strings.media.imageAlt, { name: selected.originalFilename })} className="max-h-[min(70vh,48rem)] min-h-[19rem] w-full flex-1 rounded-control bg-app-muted object-contain" />}
        {selected && <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 break-all text-xs text-app-muted-foreground">
          {selected.width !== undefined && selected.height !== undefined && <><dt>{t(strings.media.dimensions)}</dt><dd>{selected.width} × {selected.height}</dd></>}
          <dt>{t(strings.media.fileType)}</dt><dd>{selected.mediaType}</dd>
          <dt>{t(strings.media.fileSize)}</dt><dd>{formatBytes(selected.sizeBytes)}</dd>
        </dl>}
        {selected?.kind === SourceMediaKind.VIDEO && <div className="rounded-panel border border-app-border bg-app-muted p-space-sm text-sm">
          <p className="font-medium">{t(strings.media.inspectionStatus)}: {t(readinessKey(selected.readiness))}</p>
          {selected.inspectionJob && <p className="mt-1 text-app-muted-foreground">{t(strings.media.jobStatus, { state: selected.inspectionJob.state.replace(/_/g, " ") })}</p>}
          {selected.inspectionIssueMessage && <p className="mt-1 text-app-muted-foreground">{selected.inspectionIssueMessage}</p>}
          {inspectionActionError && <p role="alert" className="mt-1">{errorMessage(inspectionActionError, t)}</p>}
          <div className="mt-2 flex flex-wrap gap-space-sm">
            {selected.readiness === MediaReadiness.PENDING && !inspectionIsActive(selected) && <Button variant="outline" className="min-h-11" disabled={inspectionAction.isPending} onClick={() => runInspectionAction("ensure")}>{t(strings.media.checkInspection)}</Button>}
            {canRetryInspection(selected) && <Button variant="outline" className="min-h-11" disabled={inspectionAction.isPending} onClick={() => runInspectionAction("retry")}>{pendingInspectionAction === "retry" ? t(strings.media.retryingInspection) : t(strings.media.retryInspection)}</Button>}
            {canCancelInspection(selected) && <Button variant="outline" className="min-h-11" disabled={inspectionAction.isPending} onClick={() => runInspectionAction("cancel")}>{pendingInspectionAction === "cancel" ? t(strings.media.cancellingInspection) : t(strings.media.cancelInspection)}</Button>}
            {selected.readiness === MediaReadiness.PENDING && <Button variant="outline" className="min-h-11" onClick={() => { void selectedQuery.refetch(); void list.refetch(); }}>{t(strings.media.refreshStatus)}</Button>}
          </div>
        </div>}
        {selected && <details className="text-xs text-app-muted-foreground"><summary className="min-h-11 cursor-pointer py-3 font-medium">{t(strings.media.sourceDetails)}</summary><p className="break-all font-mono"><span>{t(strings.media.sha256)}: </span>{selected.sha256}</p></details>}
      </CardContent>
    </Card>
    <Card className="source-media-controls min-w-0">
      <CardHeader><CardTitle as="h3">{t(strings.media.title)}</CardTitle><p className="text-sm text-app-muted-foreground">{t(strings.media.description)}</p></CardHeader>
      <CardContent className="flex flex-col gap-space-md">
      <form className="flex flex-col gap-space-sm" onSubmit={(event) => { event.preventDefault(); upload.mutate(); }}>
        <label htmlFor={selectors.media.fileInput} className="text-sm font-medium">{t(strings.media.fileLabel)}</label>
        <input id={selectors.media.fileInput} data-testid={selectors.media.fileInput} type="file" accept="image/png,image/jpeg,video/mp4,video/quicktime,video/webm,video/x-matroska" disabled={upload.isPending} onChange={(event) => { setFile(event.currentTarget.files?.[0]); upload.reset(); }} aria-describedby={selectors.media.fileHelp} className="min-h-11 rounded-control border border-app-border bg-app-surface px-3 py-2 text-base text-app-foreground file:mr-3 file:min-h-8 file:rounded-control file:border-0 file:bg-app-muted file:px-3 file:text-sm file:font-medium" />
        <p id={selectors.media.fileHelp} className="text-sm text-app-muted-foreground">{t(strings.media.fileHelp)}</p>
        <div><Button data-testid={selectors.media.uploadButton} type="submit" className="min-h-11" disabled={!file || upload.isPending}>{upload.isPending ? t(strings.media.uploading) : t(strings.media.upload)}</Button></div>
        {upload.isPending && <p data-testid={selectors.media.pending} role="status">{t(strings.media.uploading)}</p>}
        {upload.error && <div data-testid={selectors.media.uploadError} role="alert" className="rounded-panel border border-app-danger/40 bg-app-danger/10 p-space-sm text-sm"><p>{errorMessage(upload.error, t)}</p><Button type="submit" variant="outline" className="mt-2 min-h-11" disabled={upload.isPending}>{t(strings.media.retry)}</Button></div>}
        {upload.data && <p role="status" className="text-sm text-app-muted-foreground">{t(strings.media.importSuccess, { name: upload.data.originalFilename })}</p>}
      </form>

      <section aria-labelledby={selectors.media.listHeading} className="flex flex-col gap-space-sm">
        <h4 id={selectors.media.listHeading} className="font-medium">{t(strings.media.saved)}</h4>
        {list.isLoading && <p data-testid={selectors.media.loading} role="status">{t(strings.media.loading)}</p>}
        {list.error && <div role="alert" className="text-sm"><p>{errorMessage(list.error, t)}</p><Button variant="outline" className="mt-2 min-h-11" onClick={() => { void list.refetch(); }}>{t(strings.media.retry)}</Button></div>}
        {!list.isLoading && !list.error && media.length === 0 && <p data-testid={selectors.media.empty} className="text-sm text-app-muted-foreground">{t(strings.media.empty)}</p>}
        {media.length > 0 && <ul data-testid={selectors.media.list} className="flex flex-col divide-y divide-app-border rounded-panel border border-app-border">
          {media.map((item) => <li key={item.id}>
            <button type="button" data-testid={selectors.media.selectButton({ id: item.id })} aria-pressed={selectedId === item.id} onClick={() => { setSelectedId(item.id); }} className="flex min-h-14 w-full items-center justify-between gap-3 px-3 py-2 text-left hover:bg-app-muted focus-visible:outline focus-visible:outline-2 focus-visible:outline-app-primary">
              <span className="min-w-0"><span className="block truncate text-sm font-medium">{item.originalFilename}</span><span className="block text-xs text-app-muted-foreground">{item.mediaType}{item.width !== undefined && item.height !== undefined ? ` · ${item.width} × ${item.height}` : ""} · {formatBytes(item.sizeBytes)}{item.kind === SourceMediaKind.VIDEO ? ` · ${t(readinessKey(item.readiness))}` : ""}</span></span>
              <span className="shrink-0 text-xs font-medium text-app-primary">{selectedId === item.id ? t(strings.media.selected) : t(strings.media.select)}</span>
            </button>
          </li>)}
        </ul>}
        {list.hasNextPage && <Button data-testid={selectors.media.loadMore} variant="outline" className="min-h-11 self-start" disabled={list.isFetchingNextPage} onClick={() => { void list.fetchNextPage(); }}>{list.isFetchingNextPage ? t(strings.media.loading) : t(strings.media.loadMore)}</Button>}
        {list.isFetchNextPageError && <p role="alert">{errorMessage(list.error, t)} <Button variant="outline" className="min-h-11" onClick={() => { void list.fetchNextPage(); }}>{t(strings.media.retry)}</Button></p>}
      </section>

      </CardContent>
    </Card>
  </section>;
}
