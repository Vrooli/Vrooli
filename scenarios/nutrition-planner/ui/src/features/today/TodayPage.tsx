import { useCallback, useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { Leaf } from "lucide-react";

import { applyPlan, generatePlan, getPlan, previewSwap, recordFeedback, undoFeedback, type PlanDraft } from "../../api/planning";
import { listRecipes } from "../../api/recipes";
import { listCatalogRevisions, type CatalogRevision } from "../../api/catalog";
import { getRecipe } from "../../api/recipes";
import { ensureWorkspace } from "../../api/workspace";
import { listSupplementSchedules, scheduleAppliesOnLocalDate, type SupplementSchedule } from "../../api/supplement";
import { useDialogFocusTrap } from "../../lib/useDialogFocusTrap";
import { localDateString } from "../../lib/dates";
import { selectors } from "../../consts/selectors";
import { scheduledSupplementNutrients } from "../../lib/supplementNutrition";
import { useTheme } from "../../theme/ThemeProvider";
import { MealMediaFrame } from "../media/mealMedia";

export function TodayPage() {
  const { resolved } = useTheme();
  const [draft, setDraft] = useState<PlanDraft>();
  const [savedDraft, setSavedDraft] = useState<PlanDraft>();
  const [hasSavedPlan, setHasSavedPlan] = useState(false);
  const [staleToday, setStaleToday] = useState(false);
  const [latestToday, setLatestToday] = useState<{ draft: PlanDraft; hasPlan: boolean }>();
  const [staleSwapReplacement, setStaleSwapReplacement] = useState("");
  const [refreshingSwap, setRefreshingSwap] = useState(false);
  const [weekDraft, setWeekDraft] = useState<PlanDraft>();
  const [weekReadState, setWeekReadState] = useState<"loading" | "ready" | "error">("loading");
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");
  const [error, setError] = useState("");
  const [workspaceId, setWorkspaceId] = useState("");
  const [action, setAction] = useState<"idle" | "saving" | "saved">("idle");
  const [hasDraft, setHasDraft] = useState(false);
  const [selectedOccurrenceKey, setSelectedOccurrenceKey] = useState("");
  const [feedbackState, setFeedbackState] = useState<"unknown" | "recording" | "recorded">("unknown");
  const [recipes, setRecipes] = useState<{ id: string; name: string }[]>([]);
  const [supplements, setSupplements] = useState<SupplementSchedule[]>([]);
  const [catalog, setCatalog] = useState<CatalogRevision[]>([]);
  const [supplementState, setSupplementState] = useState<"loading" | "ready" | "error">("loading");
  const [swapOpen, setSwapOpen] = useState(false);
  const [replacementId, setReplacementId] = useState("");
  const [swapPreview, setSwapPreview] = useState<Awaited<ReturnType<typeof previewSwap>>>();
  const navigate = useNavigate();
  const occurrence = draft?.occurrences.find((item) => `${item.date}:${item.slotName ?? "meal"}` === selectedOccurrenceKey) ?? draft?.occurrences[0];
  const todayChanges = describeTodayChanges(savedDraft, draft);
  const swapNeedsImpactReview = Boolean(swapPreview && ((swapPreview.affectedDates?.length ?? 0) > 1 || (swapPreview.preview.shoppingChanges?.length ?? 0) > 0 || (swapPreview.preview.relatedOccurrences?.length ?? 0) > 0 || (swapPreview.preview.preparedBatchImpacts?.length ?? 0) > 0));
  const closeSwap = useCallback(() => { setSwapOpen(false); setSwapPreview(undefined); }, []);

  useDialogFocusTrap(swapOpen, closeSwap, "[role=\"dialog\"]");

  useEffect(() => {
    let mounted = true;
    const today = localDateString();
    void ensureWorkspace().then(async (workspace) => {
      if (mounted) setWorkspaceId(workspace.id);
      void Promise.all([listSupplementSchedules(workspace.id), listCatalogRevisions(workspace.id).catch(() => [])]).then(([items, revisions]) => {
        if (mounted) { setSupplements(items); setCatalog(revisions); setSupplementState("ready"); }
      }).catch(() => { if (mounted) setSupplementState("error"); });
      const [result, week] = await Promise.all([
        getPlan({ workspaceId: workspace.id, fromDate: today, toDate: today }),
        getPlan({ workspaceId: workspace.id, fromDate: today, toDate: addLocalDays(today, 6) }).catch(() => undefined),
      ]);
      if (mounted) {
        setDraft(result.draft);
        setSavedDraft(result.draft);
        setHasSavedPlan(result.hasPlan);
        setSelectedOccurrenceKey(result.draft.occurrences[0] ? `${result.draft.occurrences[0].date}:${result.draft.occurrences[0].slotName ?? "meal"}` : "");
        setHasDraft(result.hasPlan);
        if (week) { setWeekDraft(week.draft); setWeekReadState("ready"); } else setWeekReadState("error");
        setState("ready");
      }
    }).catch((err: unknown) => {
      if (mounted) { setError(err instanceof Error ? err.message : "Unable to load today’s plan."); setState("error"); }
    });
    return () => { mounted = false; };
  }, []);

  async function makeTodayDraft(baseDraft = savedDraft) {
    if (!workspaceId) return;
    const today = localDateString();
    try {
      setError("");
      const existing = (baseDraft?.occurrences ?? []).filter((item) => item.date === today);
      const mealSlots = existing.length ? existing.map((item) => ({ date: item.date, slotName: item.slotName ?? "meal", mode: item.mode ?? "flexible", quantity: item.quantity ?? "1", lockedRecipeId: item.locked ? item.recipeId : "" })) : [
        { date: today, slotName: "breakfast" }, { date: today, slotName: "lunch" }, { date: today, slotName: "dinner" }, { date: today, slotName: "snack", mode: "open" },
      ];
      const result = await generatePlan({ workspaceId, dates: [today], mealSlots });
      setDraft(result);
      setSelectedOccurrenceKey(result.occurrences[0] ? `${result.occurrences[0].date}:${result.occurrences[0].slotName ?? "meal"}` : "");
      setHasDraft(true);
      setAction("idle");
      setStaleToday(false);
      setLatestToday(undefined);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to draft today’s plan.");
    }
  }

  async function savePlan() {
    if (!draft || !workspaceId) return;
    setAction("saving");
    try { const saved = await applyPlan({ workspaceId, expectedRevision: draft.currentRevision, draft }); const next = { ...draft, currentRevision: saved.revision }; setDraft(next); setSavedDraft(next); setHasSavedPlan(true); setHasDraft(true); setAction("saved"); setStaleToday(false); setLatestToday(undefined); }
    catch (err) { setError(err instanceof Error ? err.message : "Unable to save today’s plan."); if (isStalePlanConflict(err)) setStaleToday(true); setAction("idle"); }
  }

  async function markEaten() {
    if (!draft || !workspaceId || !occurrence?.recipeId) return;
    setFeedbackState("recording");
    try { await recordFeedback({ workspaceId, expectedRevision: draft.currentRevision, date: occurrence.date, recipeId: occurrence.recipeId }); setFeedbackState("recorded"); }
    catch (err) { setError(err instanceof Error ? err.message : "Unable to record meal feedback."); setFeedbackState("unknown"); }
  }

  async function undoEaten() {
    if (!workspaceId || !occurrence) return;
    try { await undoFeedback({ workspaceId, date: occurrence.date }); setFeedbackState("unknown"); }
    catch (err) { setError(err instanceof Error ? err.message : "Unable to undo meal feedback."); }
  }

  async function openSwap() {
    if (!workspaceId) return;
    try {
      setError("");
      const available = await listRecipes(workspaceId);
      setRecipes(available.map((recipe) => ({ id: recipe.id, name: recipe.name })));
      setReplacementId(""); setSwapPreview(undefined); setSwapOpen(true);
    } catch (err) { setError(err instanceof Error ? err.message : "Unable to load replacement meals."); }
  }

  async function makeSwapPreview() {
    if (!draft || !workspaceId || !occurrence || !replacementId) return;
    try { setError(""); setSwapPreview(await previewSwap({ workspaceId, expectedRevision: draft.currentRevision, date: occurrence.date, slotName: occurrence.slotName ?? "meal", replacementRecipeId: replacementId })); }
    catch (err) { setError(err instanceof Error ? err.message : "Unable to preview this swap."); }
  }

  async function applySwap() {
    if (!swapPreview || !workspaceId) return;
    try {
      const applied = await applyPlan({ workspaceId, expectedRevision: swapPreview.revision, draft: swapPreview.preview.draft });
      const next = { ...swapPreview.preview.draft, currentRevision: applied.revision };
      setDraft(next); setSavedDraft(next); setHasSavedPlan(true); setSwapOpen(false); setSwapPreview(undefined); setStaleToday(false); setLatestToday(undefined);
    } catch (err) { setError(err instanceof Error ? err.message : "Unable to apply this swap."); if (isStalePlanConflict(err)) { setDraft(swapPreview.preview.draft); setStaleSwapReplacement(replacementId); setStaleToday(true); setSwapOpen(false); setSwapPreview(undefined); } }
  }

  async function reviewLatestToday() {
    if (!workspaceId) return;
    try { const today = localDateString(); setLatestToday(await getPlan({ workspaceId, fromDate: today, toDate: today })); }
    catch (err) { setError(err instanceof Error ? err.message : "Unable to load the latest saved day."); }
  }

  function useLatestToday() {
    if (!latestToday) return;
    setDraft(latestToday.draft); setSavedDraft(latestToday.draft); setHasDraft(latestToday.hasPlan); setHasSavedPlan(latestToday.hasPlan); setStaleToday(false); setLatestToday(undefined); setError("");
  }

  async function refreshStaleSwap() {
    if (!draft || !workspaceId || !occurrence || !staleSwapReplacement) return;
    setRefreshingSwap(true);
    try {
      const preview = await previewSwap({ workspaceId, expectedRevision: draft.currentRevision, date: occurrence.date, slotName: occurrence.slotName ?? "meal", replacementRecipeId: staleSwapReplacement });
      setSwapPreview(preview); setReplacementId(staleSwapReplacement); setSwapOpen(true); setStaleSwapReplacement(""); setError("");
    } catch (err) { setError(err instanceof Error ? err.message : "Unable to refresh the swap review."); }
    finally { setRefreshingSwap(false); }
  }

  async function openRecipe(target = occurrence) {
    if (!workspaceId || !target?.recipeId) return;
    try { setError(""); const recipe = await getRecipe(workspaceId, target.recipeId); navigate(`/recipes/${encodeURIComponent(recipe.id)}/revisions/${recipe.revision.toString()}`); }
    catch (err) { setError(err instanceof Error ? err.message : "Unable to load this recipe."); }
  }

  function startCookingForOccurrence(target = occurrence) {
    if (!target?.recipeId) return;
    if (!target.recipeRevision || target.recipeRevision < 1) {
      setError("This planned meal has no pinned recipe revision, so cooking cannot be started safely.");
      return;
    }
    setError("");
    navigate(`/recipes/${encodeURIComponent(target.recipeId)}/revisions/${target.recipeRevision}`);
  }

  return (
    <section data-testid={selectors.pages.today} aria-labelledby="today-heading" className="today-page flex flex-col gap-6">
      <header className="today-page-heading">
        <p>Today · {formatTodayDate(localDateString())}</p>
        <h1 id="today-heading">Your day</h1>
      </header>
      {state === "loading" && <p role="status" className="rounded border border-app-border bg-app-surface p-5 text-app-muted-foreground">Loading today’s plan…</p>}
      {state === "error" && <p role="alert" className="rounded border border-app-danger bg-app-surface-muted p-5 text-app-danger">{error}</p>}
      {state === "ready" && error && <p role="alert" className="rounded border border-app-danger bg-app-surface-muted p-4 text-app-danger">{error}</p>}
      {state === "ready" && staleToday && <section className="today-stale-review" aria-label="Stale day review">
        <h2>Today changed while you were reviewing</h2>
        <p>Your local draft remains here. Review the latest saved day before choosing how to continue.</p>
        {!latestToday && <button type="button" className="today-primary-action" onClick={() => void reviewLatestToday()}>Review latest day</button>}
        {latestToday && <>
          <div className="today-stale-diffs"><div><h3>Saved changes</h3><TodayChangeList changes={describeTodayChanges(savedDraft, latestToday.draft)} empty="The saved day’s occurrences are unchanged; only its revision moved." /></div><div><h3>Your draft</h3><TodayChangeList changes={describeTodayChanges(savedDraft, draft)} empty="No local occurrence changes." /></div></div>
          <div className="today-hero-actions"><button type="button" className="today-primary-action" onClick={useLatestToday}>Use latest saved day</button><button type="button" className="today-secondary-action" onClick={() => { setSavedDraft(latestToday.draft); setHasSavedPlan(latestToday.hasPlan); void makeTodayDraft(latestToday.draft); }}>Replan from latest day</button></div>
        </>}
      </section>}
      {state === "ready" && draft && hasSavedPlan && !staleToday && todayChanges.length > 0 && <section className="today-stale-review" aria-label="Review today’s changes"><h2>Review today’s changes</h2><TodayChangeList changes={todayChanges} empty="No changes." /></section>}
      {state === "ready" && staleSwapReplacement && !staleToday && <section className="today-stale-review" aria-label="Swap needs fresh review"><h2>Refresh the swap review</h2><p>The day is based on the latest saved revision. Recheck the replacement and its effects before applying it.</p><button type="button" className="today-primary-action" disabled={refreshingSwap} onClick={() => void refreshStaleSwap()}>{refreshingSwap ? "Refreshing…" : "Refresh swap review"}</button><button type="button" className="today-secondary-action" onClick={() => { setStaleSwapReplacement(""); if (savedDraft) setDraft(savedDraft); }}>Discard local draft</button></section>}
      {state === "ready" && draft && (
        <article className="today-hero" aria-label="Next planned meal">
          <div className="today-hero-copy">
            <p className="today-hero-kicker">{occurrence ? `${formatWeekday(occurrence.date)} · ${occurrence.slotName ?? "meal"}` : "Ready when you are"}</p>
            <h2>{occurrence?.recipeName || (occurrence ? "Open slot" : hasDraft ? "Choose a meal for today" : "Nothing planned for today")}</h2>
            <p>{occurrence?.recipeName ? occurrence.reason : occurrence ? "This slot remains user-controlled; no recipe was inferred." : "Your saved meals will appear here. Open space stays yours to fill."}</p>
            {occurrence && <p className="today-hero-context">{occurrence.date} · {occurrence.slotName ?? "meal"} plan actions</p>}
            <div className="today-hero-actions">
              {occurrence?.recipeId && <button type="button" className="today-secondary-action" onClick={() => void openRecipe()}>See the recipe map</button>}
              {occurrence?.recipeId && <button type="button" className="today-secondary-action" onClick={() => void openSwap()}>Swap meal</button>}
              {occurrence?.recipeId && <button type="button" className="today-secondary-action" onClick={() => void (feedbackState === "recorded" ? undoEaten() : markEaten())} disabled={feedbackState === "recording"}>{feedbackState === "recording" ? "Recording…" : feedbackState === "recorded" ? "Undo eaten" : "I ate this"}</button>}
              {hasDraft && <button type="button" className="today-secondary-action" onClick={() => void savePlan()} disabled={action === "saving" || staleToday || Boolean(staleSwapReplacement)}>{action === "saving" ? "Saving plan…" : "Save/Apply plan"}</button>}
              {occurrence?.recipeId && <button type="button" className="today-primary-action" onClick={() => startCookingForOccurrence()} disabled={!occurrence.recipeRevision || occurrence.recipeRevision < 1}>Start cooking</button>}
            </div>
            {occurrence?.recipeId && (!occurrence.recipeRevision || occurrence.recipeRevision < 1) && <p className="text-sm text-app-warning">This planned meal has no pinned recipe revision, so cooking cannot be started safely.</p>}
            {action === "saved" && <p role="status" className="today-success">This plan is saved for the workspace.</p>}
            {feedbackState === "recorded" && <p role="status" className="today-success">Recorded as eaten. This explicit feedback is separate from the scheduled plan.</p>}
          </div>
          <MealMediaFrame className="today-hero-scene" recipeId={occurrence?.recipeId} recipeRevision={occurrence?.recipeRevision} appearance={resolved === "dark" ? "evening" : "light"} label={occurrence?.recipeName ?? "Planned meal"} fallbackText="Meal photo unavailable" />
        </article>
      )}
      {state === "ready" && (
        <section className="today-week-section" aria-labelledby="today-week-heading">
          <div className="today-section-heading">
            <h2 id="today-week-heading">Your week</h2>
            <Link to="/week">Open week <span aria-hidden="true">›</span></Link>
          </div>
          {weekReadState === "loading" && <p role="status" className="text-sm text-app-muted-foreground">Loading saved meals…</p>}
          {weekReadState === "error" && <p role="status" className="text-sm text-app-warning">The rest of this week is unavailable right now.</p>}
          {weekReadState === "ready" && <ol className="today-week-strip" aria-label="Your week">
            {Array.from({ length: 7 }, (_, index) => {
              const date = addLocalDays(localDateString(), index);
              const meals = weekDraft?.occurrences.filter((item) => item.date === date && item.recipeId) ?? [];
              const preview = meals.find((item) => item.slotName === "dinner") ?? meals[0];
              return <li key={date}>
                <Link to={`/week?range=day&date=${date}`} aria-label={`${formatWeekday(date)}, ${preview?.recipeName ?? "no saved meal"}`}>
                  <span className="today-week-day">{formatWeekday(date)}</span>
                  <span className="today-week-photo" aria-hidden="true">{preview ? <Leaf /> : <span>＋</span>}</span>
                  <strong>{preview?.recipeName ?? "No saved meal"}</strong>
                </Link>
              </li>;
            })}
          </ol>}
        </section>
      )}
      {state === "ready" && <section className="today-shopping-footer" aria-label="Plan shopping next step"><div><p>Ready for tonight</p><span>See what your selected plan needs and what is already on hand.</span></div><Link to="/groceries">Review groceries <span aria-hidden="true">›</span></Link></section>}
      {state === "ready" && <section aria-labelledby="today-supplements-heading" className="today-supplements">
        <div className="today-section-heading"><h2 id="today-supplements-heading">Fixed supplements</h2><span>Expected today</span></div>
        <p>Mapped catalog amounts appear in Expected only on confirmed dates. A schedule does not record that a dose was taken; unmapped or incompatible amounts stay unknown.</p>
        {supplementState === "loading" && <p role="status">Checking confirmed schedules…</p>}
        {supplementState === "error" && <p role="status">Schedule status is unavailable; today’s supplement expectations are unknown.</p>}
        {supplementState === "ready" && (() => { const due = supplements.filter((schedule) => scheduleAppliesOnLocalDate(schedule, localDateString())); return due.length === 0 ? <p>No confirmed supplement schedule applies today.</p> : <ul aria-label="Supplements scheduled today">{due.map((schedule) => { const contributions = scheduledSupplementNutrients(schedule, catalog); return <li key={`${schedule.id}:${schedule.revision}`}><strong>{contributions[0]?.productName ?? schedule.productRevisionId}</strong> · {schedule.dose} {schedule.doseUnit} scheduled<ul>{contributions.map((item, index) => <li key={`${schedule.id}:${schedule.revision}:${item.nutrientId}:${index}`}>{item.nutrientId ? `${item.nutrientId}: ` : "Nutrient contribution: "}{item.unresolvedReason || `${item.amount} ${item.unit} expected (${item.evidence || "catalog evidence"})`}</li>)}</ul></li>; })}</ul>; })()}
      </section>}
      {state === "ready" && !hasDraft && <article className="today-empty-state"><h2>Nothing planned for today</h2><p>Start a draft when you are ready; planning never writes until you save.</p><button type="button" className="today-primary-action" onClick={() => void makeTodayDraft()}>Draft today’s meals</button></article>}
      {state === "ready" && draft && (
        <section className="today-overview" aria-labelledby="today-overview-heading">
          <div className="today-section-heading"><div><h2 id="today-overview-heading">Your whole day</h2><p>Meals, snacks and recurring items · {formatTodayDate(localDateString())}</p></div><span>Planned / Recorded / Expected</span></div>
          {draft.occurrences.filter((item) => item !== occurrence).length === 0 && <p className="today-overview-empty">No other saved meal slots today.</p>}
          <div className="today-slot-list">{draft.occurrences.filter((item) => item !== occurrence).map((item, index) => {
            const itemKey = `${item.date}:${item.slotName ?? "meal"}`;
            return <article key={`${item.date}-${item.slotName ?? index}`} className="today-slot-row">
              <span className="today-slot-name">{item.slotName ?? "meal"}</span>
              <div><h3>{item.recipeName || "Intentionally open"}</h3><p>{item.recipeName ? item.reason : "This slot remains user-controlled; no recipe was inferred."}</p></div>
              <button type="button" aria-pressed={selectedOccurrenceKey === itemKey} onClick={() => { setSelectedOccurrenceKey(itemKey); setFeedbackState("unknown"); }}>{selectedOccurrenceKey === itemKey ? "Selected" : `Select ${item.slotName ?? "meal"} actions`}</button>
            </article>;
          })}</div>
          {draft.unresolved.map((item) => <div key={`${item.date}-${item.code}`} className="today-unresolved"><strong>{item.date}: no eligible meal</strong><p>{item.message}</p></div>)}
          <dl className="today-evidence-metrics">{["Energy", "Protein", "Portion cost", "Time"].map((metric) => <div key={metric}><dt>{metric}</dt><dd>Unknown until entered or verified.</dd></div>)}</dl>
        </section>
      )}
      {swapOpen && <div role="dialog" aria-labelledby="today-swap-heading" aria-modal="true" className="rounded-lg border border-app-accent bg-app-surface-muted p-5"><h2 id="today-swap-heading" className="font-semibold text-app-foreground">Preview a replacement</h2><label className="mt-4 block text-sm font-medium text-app-foreground" htmlFor="today-replacement-recipe">Replacement meal</label><select id="today-replacement-recipe" className="mt-1 min-h-11 w-full rounded border bg-app-surface px-3" value={replacementId} onChange={(event) => setReplacementId(event.target.value)}><option value="">Choose a meal</option>{recipes.map((recipe) => <option key={recipe.id} value={recipe.id}>{recipe.name}</option>)}</select>{swapPreview && <><ul className="mt-4 space-y-1 text-sm text-app-foreground">{swapPreview.preview.changes.map((change) => <li key={`${change.date}-${change.slotName}`}>{change.date} · {change.slotName}: {change.beforeName} → {change.afterName}</li>)}</ul>{swapNeedsImpactReview ? <section aria-label="Shopping impact" className="mt-4 rounded border bg-app-surface p-3"><h3 className="font-medium text-app-foreground">Shopping impact</h3>{(swapPreview.preview.relatedOccurrences?.length ?? 0) > 0 && <div className="mt-2 text-sm text-app-foreground"><h4 className="font-medium">Later uses to review</h4><p>These later occurrences are unchanged. Confirm whether any depends on this meal before applying; no batch or leftover relationship is inferred.</p><ul className="mt-1 list-disc pl-5">{swapPreview.preview.relatedOccurrences?.map((item) => <li key={`${item.date}-${item.slotName}`}>{item.date} · {item.slotName}: {item.recipeName} · revision {item.recipeRevision}{item.locked ? " · locked" : ""}</li>)}</ul></div>}{(swapPreview.preview.preparedBatchImpacts?.length ?? 0) > 0 && <div className="mt-2 text-sm text-app-foreground"><h4 className="font-medium">Prepared portions to review</h4><p>These portions remain in Kitchen inventory. Confirm whether this swap changes which meal should use them; the preview does not allocate or consume a batch.</p><ul className="mt-1 list-disc pl-5">{swapPreview.preview.preparedBatchImpacts?.map((item, index) => <li key={`${item.recipeId}-${item.recipeRevision}-${index}`}>{item.recipeName} · revision {item.recipeRevision} · batch {item.batchId}: {item.available} {item.unit} available</li>)}</ul></div>}{swapPreview.preview.shoppingChanges.length ? <ul className="mt-2 space-y-2 text-sm text-app-foreground">{swapPreview.preview.shoppingChanges.map((change) => { const line = change.after ?? change.before; if (!line) return null; return <li key={change.key}><strong>{change.after ? "Added" : "Removed"}:</strong> {line.label} · Need: {line.need} · Stock: {line.stock} · Missing: {line.missing} · Packages: {line.packageCount} · Price: {line.price}</li>; })}</ul> : <p className="mt-1 text-sm text-app-muted-foreground">No ingredient lines changed; dates affected: {swapPreview.affectedDates.join(", ")}.</p>}<p className="mt-2 text-xs text-app-muted-foreground">This is a plan-derived review only; it does not reserve stock, mark items purchased, or change inventory.</p></section> : <p className="mt-3 text-sm text-app-muted-foreground">This preview changes one meal slot and no shopping lines.</p>}</> }<div className="mt-4 flex gap-2"><button type="button" className="min-h-11 rounded bg-app-primary px-4 font-medium text-app-primary-foreground" onClick={() => void (swapPreview ? applySwap() : makeSwapPreview())} disabled={!replacementId}>{swapPreview ? (swapNeedsImpactReview ? "Apply reviewed changes" : "Confirm swap") : "Preview swap"}</button><button type="button" className="min-h-11 rounded border bg-app-surface px-4" onClick={() => setSwapOpen(false)}>Cancel</button></div></div>}
    </section>
  );
}


function addLocalDays(date: string, amount: number) {
  const value = new Date(`${date}T12:00:00`);
  value.setDate(value.getDate() + amount);
  return localDateString(value);
}
function formatTodayDate(date: string) {
  return new Intl.DateTimeFormat(undefined, { month: "long", day: "numeric", year: "numeric" }).format(new Date(`${date}T12:00:00`));
}
function formatWeekday(date: string) {
  return new Intl.DateTimeFormat(undefined, { weekday: "short", timeZone: "UTC" }).format(new Date(`${date}T12:00:00Z`));
}

function isStalePlanConflict(err: unknown) {
  const value = err as { message?: unknown; code?: unknown };
  return /aborted|stale|changed elsewhere|revision/i.test(`${value?.message ?? err ?? ""} ${value?.code ?? ""}`);
}

function describeTodayChanges(before: PlanDraft | undefined, after: PlanDraft | undefined) {
  if (!before || !after) return [];
  const previous = new Map(before.occurrences.map((item) => [`${item.date}|${item.slotName ?? "meal"}`, item]));
  const current = new Map(after.occurrences.map((item) => [`${item.date}|${item.slotName ?? "meal"}`, item]));
  const changes: string[] = [];
  for (const [key, item] of current) {
    const prior = previous.get(key);
    if (!prior) changes.push(`${key.replace("|", " · ")}: added ${item.recipeName || "open slot"}`);
    else if (prior.recipeId !== item.recipeId || prior.locked !== item.locked || prior.mode !== item.mode || prior.quantity !== item.quantity) changes.push(`${key.replace("|", " · ")}: ${prior.recipeName || "open slot"} → ${item.recipeName || "open slot"}${prior.locked !== item.locked ? item.locked ? " (locked)" : " (unlocked)" : ""}`);
  }
  for (const [key, item] of previous) if (!current.has(key)) changes.push(`${key.replace("|", " · ")}: removed ${item.recipeName || "open slot"}`);
  return changes;
}

function TodayChangeList({ changes, empty }: { changes: string[]; empty: string }) {
  return changes.length ? <ul>{changes.map((change) => <li key={change}>{change}</li>)}</ul> : <p>{empty}</p>;
}
