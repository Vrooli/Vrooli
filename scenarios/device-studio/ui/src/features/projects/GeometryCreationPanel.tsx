import { useCallback, useEffect, useMemo, useState } from "react";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";
import { MediaReadiness, SourceMediaKind } from "@vrooli/proto-types/device-studio/v1/projects/projects_pb";
import { cancelGeometryCreation, findGeometryCreationOperation, getGeometryCreation, startGeometryCreation, type GeometryCreationJob } from "../../api/creation";
import { projectsClient } from "../../api/projects";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

function newID(prefix: string) {
  const suffix = globalThis.crypto?.randomUUID?.() ?? `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`;
  return `${prefix}-${suffix}`;
}

type CreationRequest = { candidateId: string; idempotencyKey: string; brief: string; envelopeMM: [number, number, number]; referenceMediaIDs: string[] };
type CreationOperation = { request: CreationRequest; jobId?: string };
function operationStorageKey(projectID: string) { return `device-studio:geometry-creation:${projectID}`; }
function loadOperation(projectID: string): CreationOperation | null {
  try {
    const raw = localStorage.getItem(operationStorageKey(projectID));
    if (!raw) return null;
    const value = JSON.parse(raw) as CreationOperation;
    const request = value?.request;
    if (!request?.candidateId || !request.idempotencyKey || !request.brief || !Array.isArray(request.envelopeMM) || request.envelopeMM.length !== 3 || !Array.isArray(request.referenceMediaIDs)) return null;
    return value;
  } catch { return null; }
}

export function GeometryCreationPanel({ projectId }: { projectId: string }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [brief, setBrief] = useState("");
  const [dimensions, setDimensions] = useState<[string, string, string]>(["", "", ""]);
  const [selectedIDs, setSelectedIDs] = useState<string[]>([]);
  const [confirmed, setConfirmed] = useState(false);
  const [operation, setOperation] = useState<CreationOperation | null>(null);
  const [operationLoaded, setOperationLoaded] = useState(false);
  const [jobID, setJobID] = useState("");
  const [storageUnavailable, setStorageUnavailable] = useState(false);
  const persistOperation = useCallback((next: CreationOperation | null): boolean => {
    try {
      if (next) localStorage.setItem(operationStorageKey(projectId), JSON.stringify(next));
      else localStorage.removeItem(operationStorageKey(projectId));
      setOperation(next);
      setStorageUnavailable(false);
      return true;
    } catch {
      setStorageUnavailable(true);
      return false;
    }
  }, [projectId]);
  useEffect(() => {
    const saved = loadOperation(projectId);
    setOperation(saved);
    setJobID(saved?.jobId ?? "");
    if (saved) {
      setBrief(saved.request.brief);
      setDimensions(saved.request.envelopeMM.map(String) as [string, string, string]);
      setSelectedIDs(saved.request.referenceMediaIDs);
      setConfirmed(true);
    }
    setOperationLoaded(true);
  }, [projectId]);
  const media = useInfiniteQuery({
    queryKey: ["project-source-media", projectId],
    initialPageParam: "",
    queryFn: ({ pageParam }) => projectsClient.listSourceMedia({ projectId, pageSize: 25, pageToken: pageParam }),
    getNextPageParam: (page) => page.nextPageToken || undefined,
    enabled: Boolean(projectId),
  });
  const references = useMemo(() => (media.data?.pages.flatMap((page) => page.sourceMedia) ?? []).filter((item) =>
    item.kind === SourceMediaKind.IMAGE && item.readiness === MediaReadiness.READY &&
    (item.mediaType === "image/png" || item.mediaType === "image/jpeg") && Number(item.width) > 0 && Number(item.height) > 0,
  ), [media.data]);
  const job = useQuery({
    queryKey: ["geometry-creation", jobID],
    queryFn: () => getGeometryCreation(jobID),
    enabled: Boolean(jobID),
    refetchInterval: (query) => {
      const state = query.state.data?.state;
      return state === "succeeded" || state === "failed" || state === "cancelled" ? false : 1500;
    },
    refetchIntervalInBackground: false,
  });
  const operationReadback = useQuery({
    queryKey: ["geometry-creation-operation", projectId, operation?.request.idempotencyKey],
    queryFn: () => findGeometryCreationOperation(projectId, operation!.request.idempotencyKey),
    enabled: operationLoaded && Boolean(operation && !operation.jobId),
    retry: false,
  });
  useEffect(() => {
    const found = operationReadback.data;
    if (found && operation && !operation.jobId) {
      persistOperation({ ...operation, jobId: found.job_id });
      setJobID(found.job_id);
    }
  }, [operationReadback.data, operation, persistOperation]);
  const start = useMutation({
    mutationFn: (request: CreationRequest) => startGeometryCreation(projectId, request),
    onSuccess: async (created) => {
      setJobID(created.job_id);
      if (operation) persistOperation({ ...operation, jobId: created.job_id });
      await queryClient.invalidateQueries({ queryKey: ["geometry-creation", created.job_id] });
    },
  });
  const cancel = useMutation({
    mutationFn: (current: GeometryCreationJob) => cancelGeometryCreation(current),
    onSuccess: (updated) => queryClient.setQueryData(["geometry-creation", updated.job_id], updated),
  });
  useEffect(() => {
    if (job.data?.state === "succeeded") void queryClient.invalidateQueries({ queryKey: ["local-geometry-intake"] });
  }, [job.data?.state, queryClient]);
  const active = job.data && ["queued", "running", "checkpointed", "cancel_requested"].includes(job.data.state);
  const sizes = dimensions.map(Number);
  const selectedBytes = references.filter((item) => selectedIDs.includes(item.id)).reduce((total, item) => total + Number(item.sizeBytes), 0);
  const referencesWithinLimit = selectedBytes > 0 && selectedBytes <= 10 * 1024 * 1024;
  const ready = Boolean(operationLoaded && !operation && brief.trim() && brief.length <= 4000 && sizes.every((value) => Number.isFinite(value) && value > 0 && value <= 1000) && selectedIDs.length > 0 && selectedIDs.length <= 4 && referencesWithinLimit && confirmed && !start.isPending && !active);
  const toggleReference = (id: string) => setSelectedIDs((current) => current.includes(id) ? current.filter((value) => value !== id) : current.length < 4 ? [...current, id] : current);
  const begin = () => {
    const request = operation?.request ?? { candidateId: newID("creation"), idempotencyKey: newID("request"), brief, envelopeMM: dimensions.map(Number) as [number, number, number], referenceMediaIDs: selectedIDs };
    const next = operation ?? { request };
    if (!operation && !persistOperation(next)) return;
    start.mutate(request);
  };
  const startNew = () => {
    if (!persistOperation(null)) return;
    setJobID("");
    setConfirmed(false);
    setBrief("");
    setDimensions(["", "", ""]);
    setSelectedIDs([]);
  };
  const terminal = Boolean(job.data && ["succeeded", "failed", "cancelled"].includes(job.data.state));

  return <Card className="physical-shot__intake" data-testid="geometry-creation-panel">
    <CardHeader><CardTitle as="h3">{t(strings.physical.createTitle)}</CardTitle><p>{t(strings.physical.createHelp)}</p></CardHeader>
    <CardContent className="physical-shot__fields">
      <label>{t(strings.physical.createBrief)}<textarea className="physical-shot__input min-h-24" value={brief} maxLength={4000} disabled={Boolean(operation)} onChange={(event) => setBrief(event.target.value)} /></label>
      <div className="grid grid-cols-1 gap-space-sm sm:grid-cols-3">
        {dimensions.map((value, index) => <label key={index}>{t(index === 0 ? strings.physical.createWidth : index === 1 ? strings.physical.createHeight : strings.physical.createDepth)}<input className="physical-shot__input" type="number" min="0.1" max="1000" step="0.1" value={value} disabled={Boolean(operation)} onChange={(event) => setDimensions((current) => current.map((old, i) => i === index ? event.target.value : old) as [string, string, string])} /></label>)}
      </div>
      <fieldset className="flex flex-col gap-space-xs"><legend>{t(strings.physical.createReferences)}</legend>
        {media.isLoading && <p role="status">{t(strings.physical.catalogLoading)}</p>}
        {media.error && <p role="alert">{t(strings.physical.createReferenceUnavailable)}</p>}
        {!media.isLoading && !media.error && references.length === 0 && <p>{t(strings.physical.createNoImages)}</p>}
        {references.map((item) => <label key={item.id} className="physical-shot__checkbox-row"><input type="checkbox" checked={selectedIDs.includes(item.id)} disabled={Boolean(operation) || (!selectedIDs.includes(item.id) && selectedIDs.length >= 4)} onChange={() => toggleReference(item.id)} />{item.originalFilename} · {item.mediaType} · {item.sizeBytes.toString()} bytes</label>)}
        {selectedIDs.length > 0 && !referencesWithinLimit && <p role="alert">{t(strings.physical.createReferencesTooLarge)}</p>}
        {media.hasNextPage && <Button type="button" variant="outline" className="min-h-11" disabled={media.isFetchingNextPage} onClick={() => void media.fetchNextPage()}>{t(strings.physical.intakeLoadMore)}</Button>}
      </fieldset>
      <label className="physical-shot__checkbox-row"><input type="checkbox" checked={confirmed} disabled={Boolean(operation)} onChange={(event) => setConfirmed(event.target.checked)} /><span>{t(strings.physical.createDisclosure)}</span></label>
      {storageUnavailable && <p role="alert">{t(strings.physical.createStorageUnavailable)}</p>}
      {!operation && <Button type="button" className="studio-primary min-h-11" disabled={!ready} onClick={begin}>{start.isPending ? t(strings.physical.createSubmitting) : t(strings.physical.createSubmit)}</Button>}
      {operation && !jobID && <Button type="button" className="studio-primary min-h-11" disabled={!operationLoaded || start.isPending || operationReadback.isFetching || Boolean(operationReadback.error)} onClick={begin}>{start.isPending ? t(strings.physical.createSubmitting) : t(strings.physical.createRetry)}</Button>}
      {operationReadback.error && <p role="alert">{t(strings.physical.createUnavailable)}</p>}
      {(start.error || job.error) && <p role="alert">{start.error?.message ?? t(strings.physical.createUnavailable)}</p>}
      {job.data && <section aria-live="polite" data-testid="geometry-creation-status"><p>{t(strings.physical.createStatus, { state: job.data.state })}</p>
        {job.data.message && <p>{job.data.message}</p>}
        {job.data.state === "succeeded" && <p>{t(strings.physical.createResult)}</p>}
        <details><summary>{t(strings.physical.intakeTechnicalDetails)}</summary><dl><dt>Job ID</dt><dd><code>{job.data.job_id}</code></dd><dt>Candidate ID</dt><dd><code>{job.data.candidate_id}</code></dd><dt>Review status</dt><dd>{job.data.review_status}</dd></dl></details>
        {active && <Button type="button" variant="outline" className="min-h-11" disabled={cancel.isPending || !job.data} onClick={() => job.data && cancel.mutate(job.data)}>{t(strings.physical.createCancel)}</Button>}
        {cancel.error && <p role="alert">{t(strings.physical.createUnavailable)}</p>}
      </section>}
      {operation && terminal && <Button type="button" variant="outline" className="min-h-11" onClick={startNew}> {t(strings.physical.createNew)}</Button>}
    </CardContent>
  </Card>;
}
