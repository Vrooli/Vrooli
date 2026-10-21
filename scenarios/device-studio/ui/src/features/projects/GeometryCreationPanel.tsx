import { useCallback, useEffect, useMemo, useState } from "react";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";
import { MediaReadiness, SourceMediaKind } from "@vrooli/proto-types/device-studio/v1/projects/projects_pb";
import { cancelGeometryCreation, findGeometryCreationOperation, getGeometryCorrection, getGeometryCreation, startGeometryCreation, type GeometryCreationJob } from "../../api/creation";
import { projectsClient } from "../../api/projects";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

function newID(prefix: string) {
  const suffix = globalThis.crypto?.randomUUID?.() ?? `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`;
  return `${prefix}-${suffix}`;
}

type FundingRoute = "local" | "byok" | "vrooli_credits";
type CreationRequest = { candidateId: string; idempotencyKey: string; brief: string; envelopeMM: [number, number, number]; referenceMediaIDs: string[]; fundingRoute: FundingRoute; correctionSourceJobId?: string; correctionRecipeSha256?: string; correctionFeedback?: string };
type CreationOperation = { request: CreationRequest; jobId?: string; phase?: "draft" | "pending" };
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
  const [fundingRoute, setFundingRoute] = useState<FundingRoute>("local");
  const [operation, setOperation] = useState<CreationOperation | null>(null);
  const [operationLoaded, setOperationLoaded] = useState(false);
  const [jobID, setJobID] = useState("");
  const [storageUnavailable, setStorageUnavailable] = useState(false);
  const [correctionFeedback, setCorrectionFeedback] = useState("");
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
      setFundingRoute(saved.request.fundingRoute ?? "local");
      setCorrectionFeedback(saved.request.correctionFeedback ?? "");
      setConfirmed(saved.phase === "draft" ? false : true);
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
    enabled: operationLoaded && Boolean(operation && !operation.jobId && operation.phase !== "draft"),
    retry: false,
  });
  useEffect(() => {
    const found = operationReadback.data;
    if (found && operation && !operation.jobId && operation.phase !== "draft") {
      persistOperation({ ...operation, jobId: found.job_id });
      setJobID(found.job_id);
    }
  }, [operationReadback.data, operation, persistOperation]);
  const start = useMutation({
    mutationFn: (request: CreationRequest) => startGeometryCreation(projectId, request),
    onSuccess: async (created, request) => {
      setJobID(created.job_id);
      const saved = loadOperation(projectId);
      if (saved?.request.idempotencyKey === request.idempotencyKey) persistOperation({ ...saved, phase: "pending", jobId: created.job_id });
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
  const correctionSourceJobID = operation?.phase === "draft" ? operation.request.correctionSourceJobId : job.data?.state === "failed" ? jobID : undefined;
  const correction = useQuery({
    queryKey: ["geometry-correction", correctionSourceJobID],
    queryFn: () => getGeometryCorrection(correctionSourceJobID!),
    enabled: Boolean(correctionSourceJobID),
    retry: false,
  });
  const editable = !operation || operation.phase === "draft";
  const sizes = dimensions.map(Number);
  const selectedBytes = references.filter((item) => selectedIDs.includes(item.id)).reduce((total, item) => total + Number(item.sizeBytes), 0);
  const referencesWithinLimit = selectedBytes > 0 && selectedBytes <= 10 * 1024 * 1024;
  const ready = Boolean(operationLoaded && editable && brief.trim() && brief.length <= 4000 && sizes.every((value) => Number.isFinite(value) && value > 0 && value <= 1000) && selectedIDs.length > 0 && selectedIDs.length <= 4 && referencesWithinLimit && confirmed && !start.isPending && !active);
  const toggleReference = (id: string) => setSelectedIDs((current) => current.includes(id) ? current.filter((value) => value !== id) : current.length < 4 ? [...current, id] : current);
  const begin = () => {
    const request = operation?.phase === "draft"
      ? { ...operation.request, brief, envelopeMM: dimensions.map(Number) as [number, number, number], referenceMediaIDs: selectedIDs, fundingRoute, correctionFeedback: correctionFeedback.trim() }
      : operation?.request ?? { candidateId: newID("creation"), idempotencyKey: newID("request"), brief, envelopeMM: dimensions.map(Number) as [number, number, number], referenceMediaIDs: selectedIDs, fundingRoute };
    const next: CreationOperation = operation ? { ...operation, request, phase: "pending" } : { request, phase: "pending" };
    if (!persistOperation(next)) return;
    start.mutate(request);
  };
  const prepareCorrection = () => {
    const details = correction.data;
    const previous = operation?.request;
    if (!details || details.disposition !== "available" || !details.issue || !details.recipe_sha256 || !previous || !jobID) return;
    const axisIndex = { x: 0, y: 1, z: 2 }[details.issue.axis];
    const half = Number(dimensions[axisIndex]) / 2;
    const next: CreationOperation = {
      phase: "draft",
      request: {
        candidateId: newID("creation"),
        idempotencyKey: newID("request"),
        brief: previous.brief,
        envelopeMM: previous.envelopeMM,
        referenceMediaIDs: [...previous.referenceMediaIDs],
        fundingRoute: previous.fundingRoute,
        correctionSourceJobId: jobID,
        correctionRecipeSha256: details.recipe_sha256,
        correctionFeedback: `Correct part ${details.issue.part_id} on axis ${details.issue.axis}; keep its full rotated half-extents within [-${half}, +${half}] mm. Preserve valid geometry from the prior recipe.`
      }
    };
    if (!persistOperation(next)) return;
    setJobID("");
    setBrief(previous.brief);
    setDimensions(previous.envelopeMM.map(String) as [string, string, string]);
    setSelectedIDs([...previous.referenceMediaIDs]);
    setFundingRoute(previous.fundingRoute);
    setCorrectionFeedback(next.request.correctionFeedback ?? "");
    setConfirmed(false);
  };
  const startNew = () => {
    if (!persistOperation(null)) return;
    setJobID("");
    setConfirmed(false);
    setBrief("");
    setDimensions(["", "", ""]);
    setSelectedIDs([]);
    setCorrectionFeedback("");
  };
  const terminal = Boolean(job.data && ["succeeded", "failed", "cancelled"].includes(job.data.state));
  const issueEnvelope = correction.data?.issue ? correction.data.envelope_mm[{ x: 0, y: 1, z: 2 }[correction.data.issue.axis]] : undefined;
  const issueLower = issueEnvelope === undefined ? undefined : -issueEnvelope / 2;
  const issueUpper = issueEnvelope === undefined ? undefined : issueEnvelope / 2;

  return <Card className="physical-shot__intake" data-testid="geometry-creation-panel">
    <CardHeader><CardTitle as="h3">{t(strings.physical.createTitle)}</CardTitle><p>{t(strings.physical.createHelp)}</p></CardHeader>
    <CardContent className="physical-shot__fields">
      <label>{t(strings.physical.createBrief)}<textarea className="physical-shot__input min-h-24" value={brief} maxLength={4000} disabled={!editable} onChange={(event) => setBrief(event.target.value)} /></label>
      <div className="grid grid-cols-1 gap-space-sm sm:grid-cols-3">
        {dimensions.map((value, index) => <label key={index}>{t(index === 0 ? strings.physical.createWidth : index === 1 ? strings.physical.createHeight : strings.physical.createDepth)}<input className="physical-shot__input" type="number" min="0.1" max="1000" step="0.1" value={value} disabled={!editable} onChange={(event) => setDimensions((current) => current.map((old, i) => i === index ? event.target.value : old) as [string, string, string])} /></label>)}
      </div>
      <fieldset className="flex flex-col gap-space-xs"><legend>{t(strings.physical.createReferences)}</legend>
        {media.isLoading && <p role="status">{t(strings.physical.catalogLoading)}</p>}
        {media.error && <p role="alert">{t(strings.physical.createReferenceUnavailable)}</p>}
        {!media.isLoading && !media.error && references.length === 0 && <p>{t(strings.physical.createNoImages)}</p>}
        {references.map((item) => <label key={item.id} className="physical-shot__checkbox-row"><input type="checkbox" checked={selectedIDs.includes(item.id)} disabled={!editable || (!selectedIDs.includes(item.id) && selectedIDs.length >= 4)} onChange={() => toggleReference(item.id)} />{item.originalFilename} · {item.mediaType} · {item.sizeBytes.toString()} bytes</label>)}
        {selectedIDs.length > 0 && !referencesWithinLimit && <p role="alert">{t(strings.physical.createReferencesTooLarge)}</p>}
        {media.hasNextPage && <Button type="button" variant="outline" className="min-h-11" disabled={media.isFetchingNextPage} onClick={() => void media.fetchNextPage()}>{t(strings.physical.intakeLoadMore)}</Button>}
      </fieldset>
      <label>{t(strings.physical.createFundingRoute)}<select className="physical-shot__input" value={fundingRoute} disabled={!editable} onChange={(event) => { setFundingRoute(event.target.value as FundingRoute); setConfirmed(false); }}>
        <option value="local">{t(strings.physical.createFundingLocal)}</option>
        <option value="byok">{t(strings.physical.createFundingBYOK)}</option>
        <option value="vrooli_credits">{t(strings.physical.createFundingCredits)}</option>
      </select></label>
      <p>{fundingRoute === "local" ? t(strings.physical.createDisclosure) : t(strings.physical.createRemoteDisclosure, { provider: fundingRoute === "byok" ? "OpenRouter BYOK" : "Vrooli credits via LPBS" })}</p>
      <label className="physical-shot__checkbox-row"><input type="checkbox" checked={confirmed} disabled={!editable} onChange={(event) => setConfirmed(event.target.checked)} /><span>{fundingRoute === "local" ? t(strings.physical.createLocalConfirm) : t(strings.physical.createRemoteConfirm)}</span></label>
      {storageUnavailable && <p role="alert">{t(strings.physical.createStorageUnavailable)}</p>}
      {operation?.phase === "draft" && correction.data?.disposition === "available" && <section data-testid="geometry-correction-draft" aria-live="polite">
        <h4>{t(strings.physical.createCorrectionTitle)}</h4>
        {correction.data.issue && <p role="alert">{t(strings.physical.createCorrectionIssue, { part: correction.data.issue.part_id, axis: correction.data.issue.axis, lower: issueLower, upper: issueUpper })}</p>}
        <details><summary>{t(strings.physical.createCorrectionPrior)}</summary><pre className="max-h-64 overflow-auto whitespace-pre-wrap text-xs">{correction.data.prior_recipe_json}</pre><p>{t(strings.physical.createCorrectionDigest, { sha: correction.data.recipe_sha256 })}</p></details>
        <label>{t(strings.physical.createCorrectionFeedback)}<textarea className="physical-shot__input min-h-24" value={correctionFeedback} maxLength={1200} onChange={(event) => setCorrectionFeedback(event.target.value)} /></label>
      </section>}
      {operation?.phase === "draft" && correction.isError && <p role="alert">{t(strings.physical.createCorrectionUnavailable)}</p>}
      {(!operation || operation.phase === "draft") && <Button type="button" className="studio-primary min-h-11" disabled={!ready || Boolean(operation?.phase === "draft" && correction.data?.disposition !== "available")} onClick={begin}>{start.isPending ? t(strings.physical.createSubmitting) : t(operation?.phase === "draft" ? strings.physical.createCorrectionSubmit : strings.physical.createSubmit)}</Button>}
      {operation && operation.phase !== "draft" && !jobID && <Button type="button" className="studio-primary min-h-11" disabled={!operationLoaded || start.isPending || operationReadback.isFetching || Boolean(operationReadback.error)} onClick={begin}>{start.isPending ? t(strings.physical.createSubmitting) : t(strings.physical.createRetry)}</Button>}
      {operationReadback.error && <p role="alert">{t(strings.physical.createUnavailable)}</p>}
      {(start.error || job.error) && <p role="alert">{start.error?.message ?? t(strings.physical.createUnavailable)}</p>}
      {job.data && <section aria-live="polite" data-testid="geometry-creation-status"><p>{t(strings.physical.createStatus, { state: job.data.state })}</p>
        {job.data.message && <p>{job.data.message}</p>}
        {job.data.state === "succeeded" && <p>{t(strings.physical.createResult)}</p>}
        <details><summary>{t(strings.physical.intakeTechnicalDetails)}</summary><dl><dt>Job ID</dt><dd><code>{job.data.job_id}</code></dd><dt>Candidate ID</dt><dd><code>{job.data.candidate_id}</code></dd><dt>Review status</dt><dd>{job.data.review_status}</dd></dl></details>
        {active && <Button type="button" variant="outline" className="min-h-11" disabled={cancel.isPending || !job.data} onClick={() => job.data && cancel.mutate(job.data)}>{t(strings.physical.createCancel)}</Button>}
        {job.data.state === "failed" && operation?.phase !== "draft" && correction.data?.disposition === "available" && <Button type="button" variant="outline" className="min-h-11" onClick={prepareCorrection}>{t(strings.physical.createCorrectionStart)}</Button>}
        {job.data.state === "failed" && correction.isLoading && <p role="status">{t(strings.physical.createCorrectionLoading)}</p>}
        {job.data.state === "failed" && correction.data?.disposition === "unavailable" && <p role="status">{t(strings.physical.createCorrectionUnavailable)}</p>}
        {cancel.error && <p role="alert">{t(strings.physical.createUnavailable)}</p>}
      </section>}
      {operation && operation.phase !== "draft" && terminal && <Button type="button" variant="outline" className="min-h-11" onClick={startNew}> {t(strings.physical.createNew)}</Button>}
    </CardContent>
  </Card>;
}
