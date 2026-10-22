import { useCallback, useEffect, useMemo, useState } from "react";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";
import { MediaReadiness, SourceMediaKind } from "@vrooli/proto-types/device-studio/v1/projects/projects_pb";
import { UsageProvenance } from "@vrooli/proto-types/ai-gateway/v1/inference/inference_pb";
import { cancelGeometryCreation, cancelGeometryCreationSession, createGeometryCreationSession, findGeometryCreationOperation, getGeometryCorrection, getGeometryCreation, getGeometryCreationSession, refineGeometryCreationSession, startGeometryCreation, type GeometryCreationJob } from "../../api/creation";
import { getLocalGeometryCreationLineage } from "../../api/library-intake";
import { projectsClient } from "../../api/projects";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

function newID(prefix: string) {
  const suffix = globalThis.crypto?.randomUUID?.() ?? `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`;
  return `${prefix}-${suffix}`;
}

type FundingRoute = "local" | "byok" | "vrooli_credits";
type CreationRequest = { candidateId: string; idempotencyKey: string; brief: string; envelopeMM: [number, number, number]; referenceMediaIDs: string[]; fundingRoute: FundingRoute; maxCostMicros?: number; correctionSourceJobId?: string; correctionMode?: "semantic_repair" | "visual_refinement"; correctionRecipeSha256?: string; correctionFeedback?: string };
type CreationOperation = { request: CreationRequest; jobId?: string; sessionId?: string; phase?: "draft" | "pending" };
function operationStorageKey(projectID: string) { return `device-studio:geometry-creation:${projectID}`; }
function parseMaxSpendMicros(value: string): number | undefined {
  const match = /^(0|[1-9]\d*)(?:\.(\d{1,6}))?$/.exec(value.trim());
  if (!match) return undefined;
  const whole = BigInt(match[1] ?? "0");
  const fraction = BigInt((match[2] ?? "").padEnd(6, "0") || "0");
  const micros = whole * 1_000_000n + fraction;
  if (micros <= 0n || micros > BigInt(Number.MAX_SAFE_INTEGER)) return undefined;
  return Number(micros);
}
function maxSpendText(micros?: number) { return micros && Number.isSafeInteger(micros) ? (micros / 1_000_000).toString() + "." + (micros % 1_000_000).toString().padStart(6, "0") : ""; }
function displayMicros(micros?: bigint) {
  if (micros === undefined || micros < 0n) return undefined;
  const whole = micros / 1_000_000n;
  const fraction = (micros % 1_000_000n).toString().padStart(6, "0").replace(/0+$/, "");
  return fraction ? `${whole}.${fraction}` : whole.toString();
}
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
  const [maxSpendUSD, setMaxSpendUSD] = useState("");
  const [operation, setOperation] = useState<CreationOperation | null>(null);
  const [operationLoaded, setOperationLoaded] = useState(false);
  const [jobID, setJobID] = useState("");
  const [storageUnavailable, setStorageUnavailable] = useState(false);
  const [correctionFeedback, setCorrectionFeedback] = useState("");
  const [pendingScopeExpansion, setPendingScopeExpansion] = useState<{ sessionId: string; request: CreationRequest } | null>(null);
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
      setMaxSpendUSD(maxSpendText(saved.request.maxCostMicros));
      setCorrectionFeedback(saved.request.correctionFeedback ?? "");
      setConfirmed(saved.request.fundingRoute === "local" ? false : saved.phase !== "draft");
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
  const lineage = useQuery({
    queryKey: ["geometry-creation-lineage", projectId, job.data?.candidate_id],
    queryFn: () => getLocalGeometryCreationLineage(job.data!.candidate_id, 1),
    enabled: job.data?.state === "succeeded" && operation?.request.fundingRoute !== "local" && Boolean(job.data.candidate_id),
    retry: false,
  });
  const operationReadback = useQuery({
    queryKey: ["geometry-creation-operation", projectId, operation?.request.idempotencyKey],
    queryFn: () => findGeometryCreationOperation(projectId, operation!.request.idempotencyKey),
    enabled: operationLoaded && Boolean(operation && !operation.jobId && operation.phase !== "draft"),
    retry: false,
  });
  const session = useQuery({
    queryKey: ["geometry-creation-session", operation?.sessionId],
    queryFn: () => getGeometryCreationSession(operation!.sessionId!),
    enabled: Boolean(operation?.sessionId),
    refetchInterval: (query) => {
      const latest = query.state.data?.children.at(-1)?.state;
      return latest === "queued" || latest === "running" || latest === "checkpointed" || latest === "cancel_requested" ? 1500 : false;
    },
    refetchIntervalInBackground: false,
    retry: false,
  });
  const sessionContinuationAuthorized = Boolean(operation?.phase === "draft" && operation.sessionId && operation.request.fundingRoute === "local" && session.data?.state === "active" && session.data.children_admitted < session.data.max_children && session.data.provider_calls_reserved < session.data.max_provider_calls && session.data.policy.project_id === projectId && session.data.policy.brief === operation.request.brief && session.data.policy.envelope_mm.every((value, index) => value === operation.request.envelopeMM[index]) && session.data.policy.reference_media_ids.length === operation.request.referenceMediaIDs.length && session.data.policy.reference_media_ids.every((value, index) => value === operation.request.referenceMediaIDs[index]) && session.data.children.at(-1)?.job_id === operation.request.correctionSourceJobId);
  useEffect(() => {
    const found = operationReadback.data;
    if (found && operation && !operation.jobId && operation.phase !== "draft") {
      persistOperation({ ...operation, jobId: found.job_id });
      setJobID(found.job_id);
    }
  }, [operationReadback.data, operation, persistOperation]);
  const start = useMutation({
    mutationFn: async (input: { request: CreationRequest; sessionId?: string; confirmScopeExpansion?: boolean; confirmLocalAI: boolean }) => {
      const request = input.request;
      if (request.fundingRoute === "local") {
        if (request.correctionSourceJobId && !input.sessionId) return { job: await startGeometryCreation(projectId, request), session: undefined };
        const created = request.correctionSourceJobId
          ? await refineGeometryCreationSession(input.sessionId!, { candidateId: request.candidateId, idempotencyKey: request.idempotencyKey, brief: request.brief, envelopeMM: request.envelopeMM, referenceMediaIDs: request.referenceMediaIDs, correctionSourceJobId: request.correctionSourceJobId, correctionMode: request.correctionMode!, correctionRecipeSha256: request.correctionRecipeSha256!, correctionFeedback: request.correctionFeedback ?? "", confirmScopeExpansion: input.confirmScopeExpansion ?? false, confirmLocalAI: input.confirmLocalAI })
          : await createGeometryCreationSession(projectId, { ...request, confirmLocalAI: input.confirmLocalAI });
        if (created.needs_confirmation) return { job: undefined, session: created, needsConfirmation: true };
        const child = created.children.at(-1);
        if (!child) throw new Error("The creation session did not return a child job.");
        return { job: { job_id: child.job_id, project_id: projectId, candidate_id: child.candidate_id, state: child.state, version: child.version, review_status: "unreviewed" as const }, session: created };
      }
      return { job: await startGeometryCreation(projectId, request), session: undefined };
    },
    onSuccess: async ({ job: created, session: createdSession, needsConfirmation }, input) => {
      if (needsConfirmation && createdSession) {
        const saved = loadOperation(projectId);
        if (saved?.request.idempotencyKey === input.request.idempotencyKey) persistOperation({ ...saved, phase: "draft", jobId: undefined, sessionId: input.sessionId });
        setJobID("");
        setPendingScopeExpansion({ sessionId: input.sessionId!, request: input.request });
        queryClient.setQueryData(["geometry-creation-session", createdSession.session_id], createdSession);
        return;
      }
      if (!created) return;
      setPendingScopeExpansion(null);
      setJobID(created.job_id);
      const saved = loadOperation(projectId);
      if (saved?.request.idempotencyKey === input.request.idempotencyKey) persistOperation({ ...saved, phase: "pending", jobId: created.job_id, sessionId: createdSession?.session_id ?? input.sessionId });
      await queryClient.invalidateQueries({ queryKey: ["geometry-creation", created.job_id] });
      if (createdSession) queryClient.setQueryData(["geometry-creation-session", createdSession.session_id], createdSession);
    },
  });
  const cancel = useMutation({
    mutationFn: async (current: GeometryCreationJob) => operation?.sessionId && session.data
      ? { session: await cancelGeometryCreationSession(operation.sessionId, session.data.version) }
      : { job: await cancelGeometryCreation(current) },
    onSuccess: ({ job: updated, session: updatedSession }) => {
      if (updated) queryClient.setQueryData(["geometry-creation", updated.job_id], updated);
      if (updatedSession) queryClient.setQueryData(["geometry-creation-session", updatedSession.session_id], updatedSession);
    },
  });
  useEffect(() => {
    if (job.data?.state === "succeeded") void queryClient.invalidateQueries({ queryKey: ["local-geometry-intake"] });
  }, [job.data?.state, queryClient]);
  const active = job.data && ["queued", "running", "checkpointed", "cancel_requested"].includes(job.data.state);
  const correctionSourceJobID = operation?.phase === "draft" ? operation.request.correctionSourceJobId : job.data?.state === "failed" || job.data?.state === "succeeded" ? jobID : undefined;
  const correction = useQuery({
    queryKey: ["geometry-correction", correctionSourceJobID],
    queryFn: () => getGeometryCorrection(correctionSourceJobID!),
    enabled: Boolean(correctionSourceJobID),
    retry: false,
  });
  const correctionMode = correction.data?.mode ?? "semantic_repair";
  const editable = !operation || operation.phase === "draft";
  const visualRefinementDraft = operation?.phase === "draft" && operation.request.correctionMode === "visual_refinement";
  const sizes = dimensions.map(Number);
  const selectedBytes = references.filter((item) => selectedIDs.includes(item.id)).reduce((total, item) => total + Number(item.sizeBytes), 0);
  const referencesWithinLimit = selectedBytes > 0 && selectedBytes <= 10 * 1024 * 1024;
  const maxSpendMicros = fundingRoute === "local" ? undefined : parseMaxSpendMicros(maxSpendUSD);
  const spendCeilingReady = fundingRoute === "local" || maxSpendMicros !== undefined;
  const ready = Boolean(operationLoaded && editable && brief.trim() && brief.length <= 4000 && sizes.every((value) => Number.isFinite(value) && value > 0 && value <= 1000) && selectedIDs.length > 0 && selectedIDs.length <= 4 && referencesWithinLimit && spendCeilingReady && (confirmed || sessionContinuationAuthorized) && !start.isPending && !active && !pendingScopeExpansion);
  const toggleReference = (id: string) => setSelectedIDs((current) => current.includes(id) ? current.filter((value) => value !== id) : current.length < 4 ? [...current, id] : current);
  const begin = () => {
    const request = operation?.phase === "draft"
      ? { ...operation.request, brief: visualRefinementDraft ? operation.request.brief : brief, envelopeMM: visualRefinementDraft ? operation.request.envelopeMM : dimensions.map(Number) as [number, number, number], referenceMediaIDs: visualRefinementDraft ? [...operation.request.referenceMediaIDs] : selectedIDs, fundingRoute, correctionFeedback: correctionFeedback.trim() }
      : operation?.request ?? { candidateId: newID("creation"), idempotencyKey: newID("request"), brief, envelopeMM: dimensions.map(Number) as [number, number, number], referenceMediaIDs: selectedIDs, fundingRoute, maxCostMicros: maxSpendMicros };
    if (request.fundingRoute === "local") delete request.maxCostMicros;
    else request.maxCostMicros = maxSpendMicros;
    const next: CreationOperation = operation ? { ...operation, request, phase: "pending" } : { request, phase: "pending" };
    if (!persistOperation(next)) return;
    // A pending durable operation is a same-key recovery of the user's prior
    // explicit submission, not a fresh scope or a second dispatch identity.
    const confirmedLocalChoice = confirmed || (operation?.phase === "pending" && request.fundingRoute === "local");
    start.mutate({ request, sessionId: operation?.sessionId, confirmLocalAI: confirmedLocalChoice });
  };
  const prepareCorrection = () => {
    const details = correction.data;
    const previous = operation?.request;
    const mode = details?.mode ?? "semantic_repair";
    if (!details || details.disposition !== "available" || !details.recipe_sha256 || !previous || !jobID || mode === "semantic_repair" && !details.issue || mode === "visual_refinement" && details.issue) return;
    const axisIndex = details.issue?.axis ? { x: 0, y: 1, z: 2 }[details.issue.axis] : 0;
    const half = Number(details.envelope_mm[axisIndex]) / 2;
    const issue = details.issue;
    const correctionGuidance = issue?.code === "bevel_exceeds_part_dimensions"
      ? `For ${issue.part_id}, the current bevel is ${issue.bevel_mm} mm and its smallest part dimension is ${issue.minimum_dimension_mm} mm. Set bevel_mm to a positive value strictly less than ${issue.max_bevel_exclusive_mm} mm (half the smallest dimension); equality is invalid. Preserve the locked envelope and valid geometry from the prior recipe.`
      : issue ? `Correct part ${issue.part_id} on axis ${issue.axis}; keep its full rotated half-extents within [-${half}, +${half}] mm. Preserve valid geometry from the prior recipe.` : "";
    const next: CreationOperation = {
      phase: "draft",
      request: {
        candidateId: newID("creation"),
        idempotencyKey: newID("request"),
        brief: previous.brief,
        envelopeMM: details.envelope_mm,
        referenceMediaIDs: [...(details.reference_media_ids ?? previous.referenceMediaIDs)],
        fundingRoute: previous.fundingRoute,
        maxCostMicros: previous.maxCostMicros,
        correctionSourceJobId: jobID,
        correctionMode: mode,
        correctionRecipeSha256: details.recipe_sha256,
        correctionFeedback: correctionGuidance
      },
      sessionId: operation?.sessionId,
    };
    if (!persistOperation(next)) return;
    setJobID("");
    setBrief(previous.brief);
    setDimensions(details.envelope_mm.map(String) as [string, string, string]);
    setSelectedIDs([...(details.reference_media_ids ?? previous.referenceMediaIDs)]);
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
    setMaxSpendUSD("");
    setCorrectionFeedback("");
    setPendingScopeExpansion(null);
  };
  const terminal = Boolean(job.data && ["succeeded", "failed", "cancelled"].includes(job.data.state));
  const issueAxisIndex = correction.data?.issue?.axis ? { x: 0, y: 1, z: 2 }[correction.data.issue.axis] : undefined;
  const issueEnvelope = issueAxisIndex === undefined ? undefined : correction.data?.envelope_mm[issueAxisIndex];
  const issueLower = issueEnvelope === undefined ? undefined : -issueEnvelope / 2;
  const issueUpper = issueEnvelope === undefined ? undefined : issueEnvelope / 2;

  return <Card className="physical-shot__intake" data-testid="geometry-creation-panel">
    <CardHeader><CardTitle as="h3">{t(strings.physical.createTitle)}</CardTitle><p>{t(strings.physical.createHelp)}</p></CardHeader>
    <CardContent className="physical-shot__fields">
      <label>{t(strings.physical.createBrief)}<textarea className="physical-shot__input min-h-24" value={brief} maxLength={4000} disabled={!editable || visualRefinementDraft} onChange={(event) => setBrief(event.target.value)} /></label>
      <div className="grid grid-cols-1 gap-space-sm sm:grid-cols-3">
        {dimensions.map((value, index) => <label key={index}>{t(index === 0 ? strings.physical.createWidth : index === 1 ? strings.physical.createHeight : strings.physical.createDepth)}<input className="physical-shot__input" type="number" min="0.1" max="1000" step="0.1" value={value} disabled={!editable || visualRefinementDraft} onChange={(event) => setDimensions((current) => current.map((old, i) => i === index ? event.target.value : old) as [string, string, string])} /></label>)}
      </div>
      <fieldset className="flex flex-col gap-space-xs"><legend>{t(strings.physical.createReferences)}</legend>
        {media.isLoading && <p role="status">{t(strings.physical.catalogLoading)}</p>}
        {media.error && <p role="alert">{t(strings.physical.createReferenceUnavailable)}</p>}
        {!media.isLoading && !media.error && references.length === 0 && <p>{t(strings.physical.createNoImages)}</p>}
        {references.map((item) => <label key={item.id} className="physical-shot__checkbox-row"><input type="checkbox" checked={selectedIDs.includes(item.id)} disabled={!editable || visualRefinementDraft || (!selectedIDs.includes(item.id) && selectedIDs.length >= 4)} onChange={() => toggleReference(item.id)} />{item.originalFilename} · {item.mediaType} · {item.sizeBytes.toString()} bytes</label>)}
        {selectedIDs.length > 0 && !referencesWithinLimit && <p role="alert">{t(strings.physical.createReferencesTooLarge)}</p>}
        {media.hasNextPage && <Button type="button" variant="outline" className="min-h-11" disabled={media.isFetchingNextPage} onClick={() => void media.fetchNextPage()}>{t(strings.physical.intakeLoadMore)}</Button>}
      </fieldset>
      <label>{t(strings.physical.createFundingRoute)}<select className="physical-shot__input" value={fundingRoute} disabled={!editable || visualRefinementDraft} onChange={(event) => { setFundingRoute(event.target.value as FundingRoute); setConfirmed(false); }}>
        <option value="local">{t(strings.physical.createFundingLocal)}</option>
        <option value="byok">{t(strings.physical.createFundingBYOK)}</option>
        <option value="vrooli_credits">{t(strings.physical.createFundingCredits)}</option>
      </select></label>
      <p>{fundingRoute === "local" ? t(strings.physical.createDisclosure) : t(strings.physical.createRemoteDisclosure, { provider: fundingRoute === "byok" ? "OpenRouter BYOK" : "Vrooli credits via LPBS" })}</p>
      {fundingRoute !== "local" && <>
        <label>{t(strings.physical.createSpendCeiling)}<input className="physical-shot__input" type="text" inputMode="decimal" autoComplete="off" value={maxSpendUSD} disabled={!editable || visualRefinementDraft} onChange={(event) => setMaxSpendUSD(event.target.value)} aria-describedby="geometry-spend-help" aria-invalid={maxSpendUSD.length > 0 && maxSpendMicros === undefined} /></label>
        <p id="geometry-spend-help">{t(strings.physical.createSpendHelp)}</p>
        {maxSpendMicros !== undefined && !operation?.jobId && <p role="status">{t(strings.physical.createSpendCapKnown, { amount: maxSpendText(maxSpendMicros) })}</p>}
        {maxSpendUSD.length > 0 && maxSpendMicros === undefined && <p role="alert">{t(strings.physical.createSpendInvalid)}</p>}
      </>}
      {sessionContinuationAuthorized ? <p role="status">{t(strings.physical.createSessionContinuation)}</p> : <label className="physical-shot__checkbox-row"><input type="checkbox" checked={confirmed} disabled={!editable} onChange={(event) => setConfirmed(event.target.checked)} /><span>{fundingRoute === "local" ? t(strings.physical.createLocalConfirm) : t(strings.physical.createRemoteConfirm)}</span></label>}
      {storageUnavailable && <p role="alert">{t(strings.physical.createStorageUnavailable)}</p>}
      {operation?.phase === "draft" && correction.data?.disposition === "available" && <section data-testid="geometry-correction-draft" aria-live="polite">
        <h4>{t(visualRefinementDraft ? strings.physical.createVisualRefinementTitle : strings.physical.createCorrectionTitle)}</h4>
        {correction.data.issue && (correction.data.issue.code === "bevel_exceeds_part_dimensions"
          ? <p role="alert">{t(strings.physical.createCorrectionBevelIssue, { part: correction.data.issue.part_id, bevel: correction.data.issue.bevel_mm, dimension: correction.data.issue.minimum_dimension_mm, upperExclusive: correction.data.issue.max_bevel_exclusive_mm })}</p>
          : <p role="alert">{t(strings.physical.createCorrectionIssue, { part: correction.data.issue.part_id, axis: correction.data.issue.axis, lower: issueLower, upper: issueUpper })}</p>)}
        <details><summary>{t(strings.physical.createCorrectionPrior)}</summary><pre className="max-h-64 overflow-auto whitespace-pre-wrap text-xs">{correction.data.prior_recipe_json}</pre><p>{t(strings.physical.createCorrectionDigest, { sha: correction.data.recipe_sha256 })}</p></details>
        <label>{t(strings.physical.createCorrectionFeedback)}<textarea className="physical-shot__input min-h-24" value={correctionFeedback} maxLength={1200} onChange={(event) => setCorrectionFeedback(event.target.value)} /></label>
      </section>}
      {operation?.phase === "draft" && correction.isError && <p role="alert">{t(strings.physical.createCorrectionUnavailable)}</p>}
      {pendingScopeExpansion && <section role="alert" data-testid="geometry-scope-expansion">
        <p>{t(strings.physical.createScopeExpansion, { fields: session.data?.changed_scope?.join(", ") ?? "" })}</p>
        <Button type="button" variant="outline" className="min-h-11" disabled={start.isPending || !confirmed} onClick={() => start.mutate({ request: pendingScopeExpansion.request, sessionId: pendingScopeExpansion.sessionId, confirmScopeExpansion: true, confirmLocalAI: confirmed })}>{t(strings.physical.createScopeExpansionConfirm)}</Button>
      </section>}
      {(!operation || operation.phase === "draft") && <Button type="button" className="studio-primary min-h-11" disabled={!ready || Boolean(operation?.phase === "draft" && correction.data?.disposition !== "available")} onClick={begin}>{start.isPending ? t(strings.physical.createSubmitting) : t(operation?.phase === "draft" ? strings.physical.createCorrectionSubmit : strings.physical.createSubmit)}</Button>}
      {operation && operation.phase !== "draft" && !jobID && <Button type="button" className="studio-primary min-h-11" disabled={!operationLoaded || start.isPending || operationReadback.isFetching || Boolean(operationReadback.error)} onClick={begin}>{start.isPending ? t(strings.physical.createSubmitting) : t(strings.physical.createRetry)}</Button>}
      {operationReadback.error && <p role="alert">{t(strings.physical.createUnavailable)}</p>}
      {(start.error || job.error) && <p role="alert">{start.error?.message ?? t(strings.physical.createUnavailable)}</p>}
      {job.data && <section aria-live="polite" data-testid="geometry-creation-status"><p>{t(strings.physical.createStatus, { state: job.data.state })}</p>
        {job.data.message && <p>{job.data.message}</p>}
        {job.data.state === "succeeded" && <p>{t(strings.physical.createResult)}</p>}
        {operation?.request.fundingRoute !== "local" && (lineage.data?.accountingStatus === "settled" && lineage.data.creditsCharged !== undefined && lineage.data.creditsCharged >= 0n
          ? <p role="status">{t(strings.physical.createSpendSettled, { cost: displayMicros(lineage.data.costProvenance === UsageProvenance.PROVIDER_REPORTED ? lineage.data.costMicros : undefined) ?? "unknown", credits: lineage.data.creditsCharged.toString() })}</p>
          : active || lineage.isLoading || lineage.data?.accountingStatus === "pending_reconciliation" ? <p role="status">{t(strings.physical.createSpendPending, { amount: maxSpendText(operation?.request.maxCostMicros) })}</p>
            : <p role="status">{t(strings.physical.createSpendUnknown)}</p>)}
        {lineage.error && <p role="alert">{t(strings.physical.createSpendUnknown)}</p>}
        <details><summary>{t(strings.physical.intakeTechnicalDetails)}</summary><dl><dt>Job ID</dt><dd><code>{job.data.job_id}</code></dd><dt>Candidate ID</dt><dd><code>{job.data.candidate_id}</code></dd><dt>Review status</dt><dd>{job.data.review_status}</dd></dl></details>
        {active && <Button type="button" variant="outline" className="min-h-11" disabled={cancel.isPending || !job.data || Boolean(operation?.sessionId && !session.data)} onClick={() => job.data && cancel.mutate(job.data)}>{t(strings.physical.createCancel)}</Button>}
        {job.data.state === "failed" && operation?.phase !== "draft" && correctionMode === "semantic_repair" && correction.data?.disposition === "available" && <Button type="button" variant="outline" className="min-h-11" onClick={prepareCorrection}>{t(strings.physical.createCorrectionStart)}</Button>}
        {job.data.state === "succeeded" && operation?.phase !== "draft" && correction.data?.mode === "visual_refinement" && correction.data?.disposition === "available" && <Button type="button" variant="outline" className="min-h-11" onClick={prepareCorrection}>{t(strings.physical.createVisualRefinementStart)}</Button>}
        {(job.data.state === "failed" || job.data.state === "succeeded") && correction.isLoading && <p role="status">{t(strings.physical.createCorrectionLoading)}</p>}
        {(job.data.state === "failed" || job.data.state === "succeeded") && correction.data?.disposition === "unavailable" && <p role="status">{t(strings.physical.createCorrectionUnavailable)}</p>}
        {cancel.error && <p role="alert">{t(strings.physical.createUnavailable)}</p>}
      </section>}
      {operation && operation.phase !== "draft" && terminal && <Button type="button" variant="outline" className="min-h-11" onClick={startNew}> {t(strings.physical.createNew)}</Button>}
    </CardContent>
  </Card>;
}
