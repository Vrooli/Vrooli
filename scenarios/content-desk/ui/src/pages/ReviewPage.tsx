import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import type { Draft } from "@vrooli/proto-types/content-desk/v1/artifacts/artifacts_pb";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1.3.1";
import { artifactsClient } from "../api/artifacts";
import { reviewClient } from "../api/review";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { DashboardPage } from "./DashboardPage";

type ReviewFilter = "needsReview" | "approved" | "published" | "all";
const REVIEW_FILTERS: readonly ReviewFilter[] = ["needsReview", "approved", "published", "all"];
function matchesReviewFilter(draft: Draft, filter: ReviewFilter): boolean {
  const status = draft.status.toLowerCase();
  if (filter === "all") return true;
  if (filter === "needsReview") return ["requested", "draft", "drafting", "drafted", "checking", "blocked", "reviewed"].includes(status);
  return status === filter;
}

type EditorialPackageContext = { channel?: string; format?: string; project_id?: string; campaign_id?: string; post_type_id?: string; lane?: string; sku?: string; scenario_name?: string; revision_authoring?: { actor_kind?: string; capacity?: string; created_at?: string }; attachments?: Array<{ id: string; asset_id: string; role: string; aspect_ratio: string; alt_text: string; position?: number }> };
function parseEditorialPackageContext(manifestJson: string): EditorialPackageContext | null {
  try {
    const parsed = JSON.parse(manifestJson) as { context?: EditorialPackageContext };
    return parsed.context ?? null;
  } catch {
    return null;
  }
}

/** Reuses the existing exact-draft gate workbench while the full 0.3 Review
 * queue and selected-state compositions are implemented. */
export function ReviewPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const [drafts, setDrafts] = useState<Draft[]>([]);
  const [packageState, setPackageState] = useState<"idle" | "saving" | "saved" | "error">("idle");
  const [savedPackage, setSavedPackage] = useState<Awaited<ReturnType<typeof reviewClient.createEditorialPackage>>["editorialPackage"]>();
  const [packageError, setPackageError] = useState("");
  const [requiredModes, setRequiredModes] = useState<string[]>([]);
  const [verdicts, setVerdicts] = useState<Record<string, { result: "" | "pass" | "fail"; evidence: string; finding: string }>>({});
  const [reviewBusy, setReviewBusy] = useState(false);
  const [reviewError, setReviewError] = useState("");
  const [reviewReceipt, setReviewReceipt] = useState("");
  const [reviewBasis, setReviewBasis] = useState<{ hasReview: boolean; runID: string; revisionID: string } | null>(null);
  const packageOperationKeys = useRef(new Map<string, string>());
  const savedFilter = searchParams.get("queue") as ReviewFilter | null;
  const filter = savedFilter && REVIEW_FILTERS.includes(savedFilter) ? savedFilter : "needsReview";
  const [state, setState] = useState<"loading" | "ready" | "error">(() => import.meta.env.MODE === "test" ? "ready" : "loading");
  const [error, setError] = useState("");
  const selectedID = searchParams.get("draft");
  const returnTo = searchParams.get("return");
  const safeReturnTo = returnTo === "/content" || returnTo?.startsWith("/content?") ? returnTo : null;
  const refresh = useCallback(async () => {
    setState("loading");
    try {
      const response = await artifactsClient.listDrafts({});
      setDrafts(response.drafts);
      setError("");
      setState("ready");
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : t(strings.pages.review.loadError));
      setState("error");
    }
  }, [t]);
  useEffect(() => { void refresh(); }, [refresh]);
  const visibleDrafts = useMemo(() => drafts.filter((draft) => matchesReviewFilter(draft, filter)), [drafts, filter]);
  const selectedDraft = drafts.find((draft) => draft.id === selectedID);
  useEffect(() => {
    let active = true;
    setRequiredModes([]);
    setVerdicts({});
    setReviewReceipt("");
    setReviewError("");
    setReviewBasis(null);
    if (!selectedDraft) return () => { active = false; };
    void Promise.all([
      reviewClient.listRequiredReviewModes({ postTypeId: selectedDraft.postTypeId }),
      artifactsClient.getDraftCurrentRevision({ id: selectedDraft.id }),
    ]).then(([modes, current]) => {
      if (!active) return;
      setRequiredModes(modes.modes);
      setReviewBasis({ hasReview: Boolean(current.currentRevision?.hasApplicableReview), runID: current.currentRevision?.reviewRunId ?? "", revisionID: current.currentRevision?.revisionId ?? "" });
    }).catch((caught: unknown) => {
      if (active) setReviewError(caught instanceof Error ? caught.message : t(strings.pages.review.loadError));
    });
    return () => { active = false; };
  }, [selectedDraft, t]);
  const savedPackageContext = savedPackage?.manifestJson ? parseEditorialPackageContext(savedPackage.manifestJson) : null;

  async function createWordingPackage() {
    if (!selectedDraft) return;
    setPackageState("saving");
    setPackageError("");
    try {
      const current = await artifactsClient.getDraftCurrentRevision({ id: selectedDraft.id });
      const revision = current.currentRevision;
      const expectedBody = revision?.body ?? "";
      if (expectedBody !== selectedDraft.body) throw new Error(t(strings.pages.review.staleWording));
      const expectedRevisionId = revision?.revisionId ?? "";
      const scope = selectedDraft.id + "\u0000" + expectedRevisionId + "\u0000" + expectedBody;
      let idempotencyKey = packageOperationKeys.current.get(scope);
      if (!idempotencyKey) {
        idempotencyKey = crypto.randomUUID();
        packageOperationKeys.current.set(scope, idempotencyKey);
      }
      const response = await reviewClient.createEditorialPackage({ draftId: selectedDraft.id, expectedRevisionId, expectedBody, idempotencyKey });
      setSavedPackage(response.editorialPackage);
      setPackageState("saved");
    } catch (caught) {
      setPackageError(caught instanceof Error ? caught.message : t(strings.pages.review.packageError));
      setPackageState("error");
    }
  }
  async function prepareReview() {
    if (!selectedDraft) return;
    setReviewBusy(true); setReviewError(""); setReviewReceipt("");
    try {
      const steps: Record<string, string[]> = { requested: ["begin", "complete", "check"], drafting: ["complete", "check"], drafted: ["check"], blocked: ["check"] };
      const events = steps[selectedDraft.status.toLowerCase()];
      if (!events) throw new Error(t(strings.pages.review.checkingOnly));
      for (const event of events) await artifactsClient.transitionDraft({ id: selectedDraft.id, event });
      await refresh();
    } catch (caught) {
      setReviewError(caught instanceof Error ? caught.message : t(strings.pages.review.loadError));
    } finally { setReviewBusy(false); }
  }
  async function recordManualReview() {
    if (!selectedDraft) return;
    const complete = requiredModes.length > 0 && requiredModes.every((mode) => {
      const verdict = verdicts[mode];
      return verdict?.result && verdict.evidence.trim() && (verdict.result === "pass" || verdict.finding.trim());
    });
    if (!complete) { setReviewError(t(strings.pages.review.reviewIncomplete)); return; }
    setReviewBusy(true); setReviewError(""); setReviewReceipt("");
    try {
      const current = await artifactsClient.getDraftCurrentRevision({ id: selectedDraft.id });
      const revision = current.currentRevision;
      if (!revision || revision.body !== selectedDraft.body) throw new Error(t(strings.pages.review.staleWording));
      const result = requiredModes.map((mode) => {
        const verdict = verdicts[mode];
        if (!verdict) throw new Error(t(strings.pages.review.reviewIncomplete));
        return { mode, passed: verdict.result === "pass", evidence: verdict.evidence.trim(), finding: verdict.finding.trim() };
      });
      const response = await reviewClient.recordReviewRun({ draftId: selectedDraft.id, expectedRevisionId: revision.revisionId || "", expectedBody: revision.body, verdicts: result });
      const passed = response.reviewRun?.outcome === "passed";
      await artifactsClient.transitionDraft({ id: selectedDraft.id, event: passed ? "review_pass" : "block" });
      setReviewReceipt(t(strings.pages.review.reviewedReceipt, { outcome: t(passed ? strings.pages.review.reviewPassed : strings.pages.review.reviewBlocked), revision: revision.revisionId }));
      await refresh();
    } catch (caught) {
      setReviewError(caught instanceof Error ? caught.message : t(strings.pages.review.loadError));
    } finally { setReviewBusy(false); }
  }
  return <section className="mx-auto w-full max-w-7xl space-y-4 px-4 py-6 sm:px-6 lg:px-8" data-experience-surface="review" data-testid="page-review">
    <header>
      <h1 className="font-serif text-4xl font-medium tracking-tight">{t(strings.pages.review.title)}</h1>
      <p className="mt-2 max-w-2xl text-sm text-app-muted-foreground">{t(strings.pages.review.description)}</p>
      {safeReturnTo ? <Button className="mt-3" variant="secondary" onClick={() => navigate(safeReturnTo)}>{t(strings.pages.review.returnToContent)}</Button> : null}
    </header>
    <Card>
      <CardHeader><CardTitle>{t(strings.pages.review.queueTitle)}</CardTitle></CardHeader>
      <CardContent className="space-y-4">
        <div role="group" aria-label={t(strings.pages.review.filterLabel)} className="flex flex-wrap gap-2">
          {REVIEW_FILTERS.map((choice) => <Button key={choice} size="sm" variant={filter === choice ? "primary" : "secondary"} aria-pressed={filter === choice} onClick={() => setSearchParams((current) => { const next = new URLSearchParams(current); next.set("queue", choice); return next; })}>{t(strings.pages.review.filters[choice])}</Button>)}
        </div>
        {state === "loading" ? <p aria-live="polite">{t(strings.pages.review.loading)}</p> : null}
        {state === "error" ? <section role="alert"><p>{error}</p><Button className="mt-2" variant="secondary" onClick={() => void refresh()}>{t(strings.pages.review.retry)}</Button></section> : null}
        {state === "ready" && visibleDrafts.length === 0 ? <p className="text-sm text-app-muted-foreground">{drafts.length === 0 ? t(strings.pages.review.empty) : t(strings.pages.review.noMatch)}</p> : null}
        {state === "ready" && visibleDrafts.length > 0 ? <ul className="grid gap-2 sm:grid-cols-2">
          {visibleDrafts.map((draft) => <li key={draft.id}><article className={`rounded-panel border p-3 ${draft.id === selectedID ? "border-app-primary" : "border-app-border"}`}>
            <div className="flex items-start justify-between gap-3"><div className="min-w-0"><h3 className="break-all font-medium">{draft.id}</h3><p className="text-sm text-app-muted-foreground">{draft.status} · {draft.postTypeId || t(strings.pages.review.unspecifiedType)} · {t(strings.pages.review.exactSnapshot)}</p></div>
              <Button size="sm" variant={draft.id === selectedID ? "primary" : "secondary"} aria-current={draft.id === selectedID ? "page" : undefined} onClick={() => { const next = new URLSearchParams(searchParams); next.set("draft", draft.id); setSearchParams(next); }}>{draft.id === selectedID ? t(strings.pages.review.selected) : t(strings.pages.review.open)}</Button></div>
            <p className="mt-2 line-clamp-3 whitespace-pre-wrap break-words text-sm">{draft.body || t(strings.pages.review.emptyBody)}</p>
          </article></li>)}
        </ul> : null}
        {selectedDraft ? <section className="rounded-control border border-app-border p-4" aria-label={t(strings.pages.review.packageTitle)}>
          <h3 className="font-medium">{t(strings.pages.review.packageTitle)}</h3>
          <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.pages.review.packageDescription)}</p>
          <Button className="mt-3" variant="secondary" disabled={packageState === "saving"} onClick={() => void createWordingPackage()}>{packageState === "saving" ? t(strings.pages.review.packageSaving) : t(strings.pages.review.packageCreate)}</Button>
          {packageState === "error" ? <p role="alert" className="mt-2 text-sm text-app-danger">{packageError}</p> : null}
          {packageState === "saved" && savedPackage ? <div className="mt-3 space-y-2 rounded-control border border-app-border bg-app-background p-3">
            <p className="break-all text-xs text-app-muted-foreground">{t(strings.pages.review.packageReceipt, { id: savedPackage.id, revision: savedPackage.sourceRevisionId || t(strings.pages.review.legacyRevision), hash: savedPackage.packageSha256 })}</p>
            <pre className="whitespace-pre-wrap break-words text-sm">{savedPackage.body}</pre>
            {savedPackageContext ? <details className="rounded-control border border-app-border p-3"><summary className="cursor-pointer text-sm font-medium">{t(strings.pages.review.packageContextTitle)}</summary>
              <p className="mt-2 text-sm">{t(strings.pages.review.packageChannelFormat, { channel: savedPackageContext.channel || t(strings.pages.plan.unspecified), format: savedPackageContext.format || t(strings.pages.plan.unspecified) })}</p>
              <p className="mt-1 text-sm">{t(strings.pages.review.packageProjectCampaign, { project: savedPackageContext.project_id || t(strings.pages.plan.unspecified), campaign: savedPackageContext.campaign_id || t(strings.pages.plan.unspecified) })}</p>
              <p className="mt-1 text-sm">{t(strings.pages.review.packageNativeContext, { postType: savedPackageContext.post_type_id || t(strings.pages.plan.unspecified), lane: savedPackageContext.lane || t(strings.pages.plan.unspecified), sku: savedPackageContext.sku || t(strings.pages.plan.unspecified), scenario: savedPackageContext.scenario_name || t(strings.pages.plan.unspecified) })}</p>
              {savedPackageContext.revision_authoring ? <p className="mt-1 text-sm">{t(strings.pages.review.packageRevisionProvenance, { actorKind: savedPackageContext.revision_authoring.actor_kind || t(strings.pages.plan.unspecified), capacity: savedPackageContext.revision_authoring.capacity || t(strings.pages.plan.unspecified), createdAt: savedPackageContext.revision_authoring.created_at || t(strings.pages.plan.unspecified) })}</p> : null}
              <h4 className="mt-2 text-sm font-medium">{t(strings.pages.review.packageMediaReferences)}</h4>
              {savedPackageContext.attachments?.length ? <ol className="mt-1 space-y-1 text-xs">{savedPackageContext.attachments.map((attachment, index) => <li key={attachment.id}>{attachment.position ?? index} · {attachment.asset_id} · {attachment.role} · {attachment.aspect_ratio} · {attachment.alt_text}</li>)}</ol> : <p className="mt-1 text-xs text-app-muted-foreground">{t(strings.pages.review.packageNoMedia)}</p>}
            </details> : null}
            <p className="text-xs text-app-muted-foreground">{t(strings.pages.review.packageLimit)}</p>
          </div> : null}
        </section> : null}
        {selectedDraft ? <section className="rounded-control border border-app-border p-4" aria-label={t(strings.pages.review.manualReviewTitle)}>
          <h3 className="font-medium">{t(strings.pages.review.manualReviewTitle)}</h3>
          <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.pages.review.manualReviewDescription)}</p>
          {selectedDraft.status === "reviewed" ? <Button className="mt-3" onClick={() => navigate(`/plan?draft=${encodeURIComponent(selectedDraft.id)}&return=${encodeURIComponent(`/review?draft=${selectedDraft.id}`)}`)}>{t(strings.pages.review.planExactDraft)}</Button> : null}
          {reviewBasis?.hasReview ? <p data-testid="review-basis-receipt" data-run-id={reviewBasis.runID} data-revision-id={reviewBasis.revisionID} role="status" className="mt-3 break-all text-sm">{t(strings.pages.review.reviewRunReceipt, { run: reviewBasis.runID, revision: reviewBasis.revisionID || t(strings.pages.review.noRevision) })}</p> : null}
          {["requested", "drafting", "drafted", "blocked"].includes(selectedDraft.status.toLowerCase()) ? <Button className="mt-3" variant="secondary" disabled={reviewBusy} onClick={() => void prepareReview()}>{t(strings.pages.review.prepareReview)}</Button> : null}
          {selectedDraft.status.toLowerCase() === "checking" ? <div className="mt-3 space-y-3">
            {requiredModes.map((mode) => {
              const value = verdicts[mode] ?? { result: "" as const, evidence: "", finding: "" };
              return <fieldset key={mode} className="grid gap-2 rounded-control border border-app-border p-3 sm:grid-cols-2">
                <legend className="px-1 text-sm font-medium">{mode}</legend>
                <label className="text-sm">{t(strings.pages.review.reviewResult)}<select aria-label={`${mode} ${t(strings.pages.review.reviewResult)}`} className="mt-1 block w-full rounded-control border border-app-border bg-app-background p-2" value={value.result} onChange={(event) => setVerdicts((current) => ({ ...current, [mode]: { ...value, result: event.target.value as "" | "pass" | "fail" } }))}><option value="">{t(strings.pages.review.chooseResult)}</option><option value="pass">{t(strings.pages.review.pass)}</option><option value="fail">{t(strings.pages.review.fail)}</option></select></label>
                <label className="text-sm">{t(strings.pages.review.reviewEvidence)}<textarea aria-label={`${mode} ${t(strings.pages.review.reviewEvidence)}`} className="mt-1 block min-h-20 w-full rounded-control border border-app-border bg-app-background p-2" value={value.evidence} onChange={(event) => setVerdicts((current) => ({ ...current, [mode]: { ...value, evidence: event.target.value } }))} /></label>
                {value.result === "fail" ? <label className="text-sm sm:col-span-2">{t(strings.pages.review.reviewFinding)}<textarea aria-label={`${mode} ${t(strings.pages.review.reviewFinding)}`} className="mt-1 block min-h-16 w-full rounded-control border border-app-border bg-app-background p-2" value={value.finding} onChange={(event) => setVerdicts((current) => ({ ...current, [mode]: { ...value, finding: event.target.value } }))} /></label> : null}
              </fieldset>;
            })}
            <Button disabled={reviewBusy || requiredModes.length === 0} onClick={() => void recordManualReview()}>{reviewBusy ? t(strings.pages.review.recordingReview) : t(strings.pages.review.recordReview)}</Button>
          </div> : null}
          {reviewError ? <p role="alert" className="mt-2 text-sm text-app-danger">{reviewError}</p> : null}
          {reviewReceipt ? <p aria-live="polite" className="mt-2 text-sm">{reviewReceipt}</p> : null}
        </section> : null}
        <p className="text-xs text-app-muted-foreground">{t(strings.pages.review.queueLimit)}</p>
      </CardContent>
    </Card>
    <DashboardPage key={`${selectedID ?? "no-selected-draft"}:${selectedDraft?.status ?? "unloaded"}`} initialSelectedDraftId={selectedID ?? undefined} preserveEmptySelection />
  </section>;
}
