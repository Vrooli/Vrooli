import { useCallback, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { applyPlan, generatePlan, getPlan, previewSwap, type PlanDraft } from "../../api/planning";
import { listRecipes } from "../../api/recipes";
import { listCatalogRevisions, type CatalogRevision } from "../../api/catalog";
import { ensureWorkspace } from "../../api/workspace";
import { exportWeeklyPDF } from "../../api/portability";
import { useDialogFocusTrap } from "../../lib/useDialogFocusTrap";
import { localDateString } from "../../lib/dates";
import { scheduledSupplementNutrients } from "../../lib/supplementNutrition";
import { selectors } from "../../consts/selectors";
import { listIntakes, type IntakeEvent } from "../../api/nutrition";
import { listSupplementSchedules, scheduleAppliesOnLocalDate, type SupplementSchedule } from "../../api/supplement";
import { ChevronLeft, ChevronRight, Leaf } from "lucide-react";

function datesFrom(start: Date): string[] { return Array.from({ length: 7 }, (_, index) => { const date = new Date(start); date.setDate(start.getDate() + index); return localDateString(date); }); }

export function WeekPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const requestedView = searchParams.get("view");
  const requestedAgenda = searchParams.get("range");
  const initialView = requestedView === "nutrition" || requestedView === "time-cost" ? requestedView : "meals";
  const initialAgenda = requestedAgenda === "all" ? "all" : "day";
  const [start, setStart] = useState(() => { const date = new Date(); date.setHours(0, 0, 0, 0); return date; });
  const [workspaceId, setWorkspaceId] = useState("");
  const [draft, setDraft] = useState<PlanDraft>();
  const [savedDraft, setSavedDraft] = useState<PlanDraft>();
  const [hasPlan, setHasPlan] = useState(false);
  const [recipes, setRecipes] = useState<{ id: string; name: string }[]>([]);
  const [intakes, setIntakes] = useState<IntakeEvent[]>([]);
  const [supplements, setSupplements] = useState<SupplementSchedule[]>([]);
  const [catalog, setCatalog] = useState<CatalogRevision[]>([]);
  const [view, setView] = useState<"meals" | "nutrition" | "time-cost">(initialView);
  const [agenda, setAgenda] = useState<"all" | "day">(initialAgenda);
  const [selectedDate, setSelectedDate] = useState(searchParams.get("date") ?? "");
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");
  const [error, setError] = useState("");
  const [swapDate, setSwapDate] = useState("");
  const [swapSlotName, setSwapSlotName] = useState("");
  const [replacementId, setReplacementId] = useState("");
  const [swapPreview, setSwapPreview] = useState<Awaited<ReturnType<typeof previewSwap>>>();
  const [exporting, setExporting] = useState(false);
  const [saving, setSaving] = useState(false);
  const [staleConflict, setStaleConflict] = useState(false);
  const [refreshingConflict, setRefreshingConflict] = useState(false);
  const [remoteDraft, setRemoteDraft] = useState<PlanDraft>();
  const [remoteHasPlan, setRemoteHasPlan] = useState(false);
  const [mergeCandidate, setMergeCandidate] = useState<PlanDraft>();
  const [mergeConflicts, setMergeConflicts] = useState<string[]>([]);
  const [staleSwapIntent, setStaleSwapIntent] = useState<{ date: string; slotName: string; replacementRecipeId: string; shoppingChanges: Awaited<ReturnType<typeof previewSwap>>["preview"]["shoppingChanges"] }>();
  const [compact, setCompact] = useState(() => typeof window !== "undefined" && window.innerWidth < 1100);
  const swapNeedsImpactReview = Boolean(swapPreview && ((swapPreview.affectedDates?.length ?? 0) > 1 || (swapPreview.preview.shoppingChanges?.length ?? 0) > 0 || (swapPreview.preview.relatedOccurrences?.length ?? 0) > 0 || (swapPreview.preview.preparedBatchImpacts?.length ?? 0) > 0));
  const closeSwap = useCallback(() => { setSwapDate(""); setSwapSlotName(""); setSwapPreview(undefined); }, []);
  useDialogFocusTrap(Boolean(swapDate), closeSwap, "[role=\"dialog\"]");
  const dates = useMemo(() => datesFrom(start), [start]);
  const activeDate = dates.includes(selectedDate) ? selectedDate : (dates[0] ?? "");
  const visibleDates = agenda === "all" ? dates : dates.filter((date) => date === activeDate);

  function updateView(next: "meals" | "nutrition" | "time-cost") {
    setView(next);
    setSearchParams((current) => { const updated = new URLSearchParams(current); updated.set("view", next); return updated; });
  }

  function updateAgenda(next: "all" | "day") {
    setAgenda(next);
    setSearchParams((current) => { const updated = new URLSearchParams(current); updated.set("range", next); return updated; });
  }

  function updateSelectedDate(next: string) {
    setSelectedDate(next);
    setSearchParams((current) => { const updated = new URLSearchParams(current); updated.set("date", next); return updated; });
  }

  useEffect(() => {
    const nextView = searchParams.get("view");
    const nextAgenda = searchParams.get("range");
    setView(nextView === "nutrition" || nextView === "time-cost" ? nextView : "meals");
    setAgenda(nextAgenda === "day" ? "day" : "all");
    const requestedDate = searchParams.get("date");
    if (requestedDate) setSelectedDate(requestedDate);
  }, [searchParams]);

  useEffect(() => {
    const resize = () => setCompact(window.innerWidth < 1100);
    window.addEventListener("resize", resize);
    return () => window.removeEventListener("resize", resize);
  }, []);

  useEffect(() => {
    let mounted = true;
    setState("loading");
    void ensureWorkspace().then(async (workspace) => {
      const [saved, available, intakeEvents, schedules, catalogRevisions] = await Promise.all([getPlan({ workspaceId: workspace.id, fromDate: dates[0] ?? "", toDate: dates[6] ?? "" }), listRecipes(workspace.id), listIntakes(workspace.id), listSupplementSchedules(workspace.id), listCatalogRevisions(workspace.id).catch(() => [])]);
      if (mounted) { setWorkspaceId(workspace.id); setDraft(saved.draft); setSavedDraft(saved.draft); setHasPlan(saved.hasPlan); setRecipes(available.map((recipe) => ({ id: recipe.id, name: recipe.name }))); setIntakes(intakeEvents); setSupplements(schedules); setCatalog(catalogRevisions); setSelectedDate((current) => dates.includes(current) ? current : (dates[0] ?? "")); setState("ready"); }
    }).catch((err: unknown) => { if (mounted) { setError(err instanceof Error ? err.message : "Unable to load this week."); setState("error"); } });
    return () => { mounted = false; };
  }, [dates]);

  async function makeWeekDraft(baseDraft = savedDraft) {
    if (!workspaceId) return;
    try {
      setError("");
      const mealSlots = dates.flatMap((date) => {
        const savedSlots = (baseDraft?.occurrences ?? []).filter((occurrence) => occurrence.date === date);
        if (savedSlots.length === 0) return [{ date, slotName: "dinner", mode: "flexible", quantity: "1" }];
        return savedSlots.map((occurrence) => ({ date, slotName: occurrence.slotName ?? "dinner", mode: occurrence.mode ?? "flexible", quantity: occurrence.quantity ?? "1", lockedRecipeId: occurrence.locked ? occurrence.recipeId : "" }));
      });
      setDraft(await generatePlan({ workspaceId, dates, mealSlots }));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to draft this week.");
    }
  }

  async function reviewLatestWeek() {
    if (!workspaceId) return;
    setRefreshingConflict(true);
    try {
      const latest = await getPlan({ workspaceId, fromDate: dates[0] ?? "", toDate: dates[6] ?? "" });
      setRemoteDraft(latest.draft);
      setRemoteHasPlan(latest.hasPlan);
      const result = mergeWeekDraft(savedDraft ?? draft ?? latest.draft, draft ?? latest.draft, latest.draft);
      setMergeCandidate(result.conflicts.length === 0 ? result.draft : undefined);
      setMergeConflicts(result.conflicts);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to load the latest saved week.");
    } finally {
      setRefreshingConflict(false);
    }
  }

  function keepMergedDraft() {
    if (!mergeCandidate || !remoteDraft) return;
    setSavedDraft(remoteDraft);
    setDraft(mergeCandidate);
    setHasPlan(remoteHasPlan);
    setError("");
    setStaleConflict(false);
    setRemoteDraft(undefined);
    setMergeCandidate(undefined);
  }

  function useLatestWeek() {
    if (!remoteDraft) return;
    setDraft(remoteDraft);
    setSavedDraft(remoteDraft);
    setHasPlan(remoteHasPlan);
    setError("");
    setStaleConflict(false);
    setRemoteDraft(undefined);
    setMergeCandidate(undefined);
    setMergeConflicts([]);
    setStaleSwapIntent(undefined);
  }

  async function makeSwapPreview() {
    if (!draft || !workspaceId || !swapDate || !swapSlotName || !replacementId) return;
    try { setError(""); setSwapPreview(await previewSwap({ workspaceId, expectedRevision: draft.currentRevision, date: swapDate, slotName: swapSlotName, replacementRecipeId: replacementId })); } catch (err) { setError(err instanceof Error ? err.message : "Unable to preview this swap."); }
  }

  async function applySwap() {
    if (!swapPreview || !workspaceId) return;
    try {
      const applied = await applyPlan({ workspaceId, expectedRevision: swapPreview.revision, draft: swapPreview.preview.draft });
      const saved = { ...swapPreview.preview.draft, currentRevision: applied.revision };
      setDraft(saved); setSavedDraft(saved); setHasPlan(true); setSwapDate(""); setSwapSlotName(""); setSwapPreview(undefined); setStaleSwapIntent(undefined); setStaleConflict(false); setRemoteDraft(undefined); setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to apply this swap.");
      if (isStalePlanConflict(err)) {
        setDraft(swapPreview.preview.draft);
        setStaleConflict(true);
        setStaleSwapIntent({ date: swapDate, slotName: swapSlotName, replacementRecipeId: replacementId, shoppingChanges: swapPreview.preview.shoppingChanges });
        setSwapDate(""); setSwapSlotName(""); setSwapPreview(undefined);
      }
    }
  }

  async function reviewStaleSwapAgain() {
    if (!staleSwapIntent || !draft || !workspaceId) return;
    try {
      const refreshed = await previewSwap({ workspaceId, expectedRevision: draft.currentRevision, date: staleSwapIntent.date, slotName: staleSwapIntent.slotName, replacementRecipeId: staleSwapIntent.replacementRecipeId });
      setSwapDate(staleSwapIntent.date); setSwapSlotName(staleSwapIntent.slotName); setReplacementId(staleSwapIntent.replacementRecipeId); setSwapPreview(refreshed); setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to refresh the swap impact.");
      if (isStalePlanConflict(err)) setStaleConflict(true);
    }
  }

  function updateOccurrence(date: string, slotName: string, update: (occurrence: PlanDraft["occurrences"][number]) => PlanDraft["occurrences"][number]) {
    setDraft((current) => current ? { ...current, occurrences: current.occurrences.map((occurrence) => occurrence.date === date && occurrence.slotName === slotName ? update(occurrence) : occurrence) } : current);
  }

  function toggleLock(date: string, slotName: string) {
    updateOccurrence(date, slotName, (occurrence) => ({ ...occurrence, locked: !occurrence.locked }));
  }

  function skipOccurrence(date: string, slotName: string) {
    updateOccurrence(date, slotName, (occurrence) => ({ ...occurrence, recipeId: "", recipeName: "", mode: "open", locked: false, reason: "Skipped by the user; this slot remains open." }));
  }

  async function saveWeek() {
    if (!draft || !workspaceId) return;
    setSaving(true);
    try {
      const applied = await applyPlan({ workspaceId, expectedRevision: draft.currentRevision, draft });
      const saved = { ...draft, currentRevision: applied.revision };
      setDraft(saved);
      setSavedDraft(saved);
      setHasPlan(true);
      setError("");
      setStaleConflict(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to save this week.");
      if (isStalePlanConflict(err)) setStaleConflict(true);
    } finally {
      setSaving(false);
    }
  }

  async function downloadPDF() {
    if (!draft || !workspaceId) return;
    setExporting(true);
    try { const result = await exportWeeklyPDF({ workspaceId, expectedRevision: draft.currentRevision }); const bytes = new Uint8Array(result.content); const url = URL.createObjectURL(new Blob([bytes.buffer], { type: "application/pdf" })); const anchor = document.createElement("a"); anchor.href = url; anchor.download = result.filename; anchor.click(); URL.revokeObjectURL(url); } catch (err) { setError(err instanceof Error ? err.message : "Unable to export this week."); } finally { setExporting(false); }
  }

  const occurrenceCount = draft?.occurrences.filter((item) => item.recipeId).length ?? 0;
  const openCount = draft?.occurrences.filter((item) => item.mode === "open" && !item.recipeId).length ?? 0;
  const lockedCount = draft?.occurrences.filter((item) => item.locked).length ?? 0;
  const slotNames = Array.from(new Set((draft?.occurrences ?? []).map((item) => item.slotName || "meal")));
  const rangeLabel = dates.length ? `${formatShortDate(dates[0] ?? "")} – ${formatShortDate(dates[6] ?? "")}` : "";
  const reviewingDraft = Boolean(draft && savedDraft && draft !== savedDraft);
  const draftChanges = draft && savedDraft ? describeWeekChanges(savedDraft, draft) : [];
  const draftNeedsSave = Boolean(draft && savedDraft && (draftChanges.length > 0 || (draft !== savedDraft && draft.unresolved.length > 0) || (!hasPlan && draft.occurrences.length > 0)));
  const openSwap = (date: string, slotName: string) => {
    setSwapDate(date);
    setSwapSlotName(slotName);
    setReplacementId("");
    setSwapPreview(undefined);
  };

  return (
    <section data-testid={selectors.pages.week} aria-labelledby="week-heading" className="week-page flex flex-col gap-6">
      <header className="week-page-heading">
        <div className="week-heading-title">
          <h1 id="week-heading">Your week</h1>
          <p>{rangeLabel} · {hasPlan ? "Saved plan" : "Draft week"}</p>
        </div>
        <div className="week-heading-actions">
          <button aria-label="Previous week" className="week-round-button" type="button" onClick={() => setStart((current) => new Date(current.getFullYear(), current.getMonth(), current.getDate() - 7))}><ChevronLeft aria-hidden="true" /></button>
          <span className="week-range-label" aria-live="polite">{rangeLabel}</span>
          <button aria-label="Next week" className="week-round-button" type="button" onClick={() => setStart((current) => new Date(current.getFullYear(), current.getMonth(), current.getDate() + 7))}><ChevronRight aria-hidden="true" /></button>
          {hasPlan && <button type="button" className="week-quiet-button" onClick={() => void makeWeekDraft()}>Replan week</button>}
          {hasPlan && <button type="button" className="week-quiet-button week-download-button" onClick={() => void downloadPDF()} disabled={exporting || !draft}>{exporting ? "Exporting…" : "Download weekly PDF"}</button>}
          {draftNeedsSave && <button type="button" className="week-primary-button" onClick={() => void saveWeek()} disabled={saving || staleConflict || Boolean(staleSwapIntent) || !draft}>{saving ? "Saving…" : "Save week"}</button>}
        </div>
      </header>
      {state === "loading" && <p role="status" className="rounded border bg-app-surface p-5 text-app-muted-foreground">Loading this week…</p>}
      {state === "error" && <p role="alert" className="rounded border border-app-danger bg-app-surface-muted p-5 text-app-danger">{error}</p>}
      {state === "ready" && !hasPlan && draft?.occurrences.length === 0 && draft.unresolved.length === 0 && (
        <article className="week-empty">
          <h2>No saved meals in this week</h2>
          <p>Draft suggestions when you are ready, then save the dates you want to keep.</p>
          <button type="button" className="week-primary-button" onClick={() => void makeWeekDraft()}>Plan my week</button>
        </article>
      )}
      {state === "ready" && draft && (
        <>
          <div className="week-summary" aria-label="Week summary">
            <span><strong>{occurrenceCount}</strong> meals assigned</span>
            <span><strong>{openCount}</strong> intentionally open slots</span>
            <span><strong>{lockedCount}</strong> locked</span>
            <a href="/groceries">Review groceries <ChevronRight aria-hidden="true" /></a>
          </div>
          {reviewingDraft && <WeekReview draft={draft} saved={savedDraft} />}
          {staleConflict && <section className="week-stale-review" aria-label="Stale plan review">
            <h2>Plan changed while you were reviewing</h2>
            <p>Your local draft is still here. Load the latest saved week to compare occurrence changes before deciding how to continue.</p>
            {staleSwapIntent && <div className="week-stale-swap-note"><strong>The proposed swap was not applied.</strong><p>Its previous shopping preview is shown for context only; refresh it before applying any replacement.</p>{staleSwapIntent.shoppingChanges.length > 0 && <ul>{staleSwapIntent.shoppingChanges.map((change) => { const line = change.after ?? change.before; return line ? <li key={change.key}>{change.after ? "Added" : "Removed"}: {line.label} · Need {line.need} · Stock {line.stock} · Missing {line.missing} · Packages {line.packageCount} · Price {line.price}</li> : null; })}</ul>}</div>}
            {!remoteDraft && <button type="button" className="week-primary-button" disabled={refreshingConflict} onClick={() => void reviewLatestWeek()}>{refreshingConflict ? "Loading latest week…" : "Review latest week"}</button>}
            {remoteDraft && <>
              <div className="week-stale-diffs">
                <div><h3>Saved changes since this draft began</h3><ChangeList changes={describeWeekChanges(savedDraft, remoteDraft)} empty="The saved occurrences are unchanged; only the revision moved." /></div>
                <div><h3>Your unsaved changes</h3><ChangeList changes={describeWeekChanges(savedDraft, draft)} empty="No local occurrence changes." /></div>
              </div>
              {mergeConflicts.length ? <div><p>Both drafts changed these same slots. Choose the current saved plan or make a fresh replan around it before saving again.</p><ul>{mergeConflicts.map((item) => <li key={item}>{item}</li>)}</ul></div> : <p>The current saved week and your draft can be combined without replacing the same changed slot. Review the updated summary, then explicitly save again.</p>}
              <div className="week-stale-actions">
                {!mergeConflicts.length && mergeCandidate && <button type="button" className="week-primary-button" onClick={keepMergedDraft}>Rebase my changes for review</button>}
                <button type="button" className="week-quiet-button" onClick={useLatestWeek}>Use latest saved week</button>
                <button type="button" className="week-quiet-button" onClick={() => { setSavedDraft(remoteDraft); setDraft(remoteDraft); setHasPlan(remoteHasPlan); setStaleConflict(false); setRemoteDraft(undefined); setMergeConflicts([]); setStaleSwapIntent(undefined); void makeWeekDraft(remoteDraft); }}>Replan from latest week</button>
              </div>
            </>}
          </section>}
          {staleSwapIntent && !staleConflict && <section className="week-stale-review" aria-label="Swap needs fresh review">
            <h2>Refresh the swap review</h2>
            <p>The draft is rebased to the current revision. Recheck the swap and its shopping effects before saving this change.</p>
            <div className="week-stale-actions"><button type="button" className="week-primary-button" onClick={() => void reviewStaleSwapAgain()}>Refresh swap and shopping review</button><button type="button" className="week-quiet-button" onClick={() => { if (savedDraft) setDraft(savedDraft); setStaleSwapIntent(undefined); }}>Discard local draft</button></div>
          </section>}
          <nav aria-label="Week views" className="week-view-toolbar">
            <div className="week-view-tabs" role="group" aria-label="Plan view">
              {(["meals", "nutrition", "time-cost"] as const).map((item) => (
                <button key={item} type="button" aria-pressed={view === item} onClick={() => updateView(item)}>
                  {item === "time-cost" ? "Time & cost" : item === "meals" ? "Meals" : "Nutrition"}
                </button>
              ))}
            </div>
            <div className="week-range-tabs" role="group" aria-label="Week range view">
              <button type="button" aria-pressed={agenda === "day"} onClick={() => updateAgenda("day")}>Day</button>
              <button type="button" aria-pressed={agenda === "all"} onClick={() => updateAgenda("all")}>All week</button>
            </div>
            <label className="sr-only" htmlFor="week-selected-date">Selected day</label>
            <select id="week-selected-date" className="week-selected-date" aria-label="Selected day" value={activeDate} onChange={(event) => updateSelectedDate(event.target.value)}>
              {dates.map((date) => <option key={date} value={date}>{formatLongDate(date)}</option>)}
            </select>
          </nav>
          {view === "meals" && (
            <>
              {!compact && <WeekBoard
                dates={dates}
                slotNames={slotNames}
                occurrences={draft.occurrences}
                unresolved={draft.unresolved}
                selectedDate={activeDate}
                onSelectDate={updateSelectedDate}
                onSwap={openSwap}
                onToggleLock={toggleLock}
                onSkip={skipOccurrence}
              />}
              {compact && <WeekAgenda
                dates={visibleDates}
                occurrences={draft.occurrences}
                unresolved={draft.unresolved}
                selectedDate={activeDate}
                onSwap={openSwap}
                onToggleLock={toggleLock}
                onSkip={skipOccurrence}
              />}
            </>
          )}
          {view === "nutrition" && (
            <ol aria-label="Daily nutrition scopes" className="week-evidence-list">
              {visibleDates.map((date) => {
                const meals = draft.occurrences.filter((item) => item.date === date && item.recipeName).map((item) => item.recipeName);
                const recorded = intakes.filter((event) => event.date === date);
                const scheduled = supplements.filter((schedule) => scheduleAppliesOnLocalDate(schedule, date));
                return <li key={date}>
                  <time dateTime={date}>{formatLongDate(date)}</time>
                  <div>
                    <p><strong>Planned:</strong> {meals.length ? meals.join(", ") : "no selected meals"}; nutrient totals unknown until verified recipe evidence exists.</p>
                    <p><strong>Recorded:</strong> {recorded.length ? recorded.map((event) => `${event.nutrientId} ${event.amount} ${event.unit}`).join(", ") : "no recorded intake"}.</p>
                    <p><strong>Expected supplements:</strong> {scheduled.length ? "confirmed schedule contributions below; no dose is recorded as taken" : "no confirmed supplement schedule applies"}.</p>
                    {scheduled.map((item) => <p key={`${item.id}:${item.revision}`}>{scheduledSupplementNutrients(item, catalog).map((contribution, index) => <span key={`${contribution.nutrientId}:${index}`}>{index > 0 ? "; " : ""}{contribution.productName} · {item.dose} {item.doseUnit} scheduled · {contribution.nutrientId ? `${contribution.nutrientId}: ` : "nutrient: "}{contribution.unresolvedReason || `${contribution.amount} ${contribution.unit} expected (${contribution.evidence || "catalog evidence"})`}</span>)}</p>)}
                  </div>
                </li>;
              })}
            </ol>
          )}
          {view === "time-cost" && (
            <ol aria-label="Daily time and cost coverage" className="week-evidence-list">
              {visibleDates.map((date) => <li key={date}>
                <time dateTime={date}>{formatLongDate(date)}</time>
                <div><p>Prep and cook time: unknown until a timed recipe estimate is verified.</p><p>Portion cost, checkout total, and actual spend: unknown without complete quantity and price coverage.</p></div>
              </li>)}
            </ol>
          )}
        </>
      )}
      {swapDate && (
        <div role="dialog" aria-labelledby="swap-heading" aria-modal="true" className="rounded-lg border border-app-accent bg-app-surface-muted p-5">
          <h2 id="swap-heading" className="font-semibold text-app-foreground">What sounds better?</h2>
          <p className="mt-1 text-sm text-app-foreground">Preview changes for {swapDate} · {swapSlotName} before applying them.</p>
          <label className="mt-4 block text-sm font-medium text-app-foreground" htmlFor="replacement-recipe">Replacement meal</label>
          <select id="replacement-recipe" className="mt-1 min-h-11 w-full rounded border bg-app-surface px-3" value={replacementId} onChange={(event) => setReplacementId(event.target.value)}>
            <option value="">Choose a meal</option>{recipes.map((recipe) => <option key={recipe.id} value={recipe.id}>{recipe.name}</option>)}
          </select>
          {swapPreview && (
            <>
              <ul className="mt-4 space-y-1 text-sm text-app-foreground">{swapPreview.preview.changes.map((change) => <li key={`${change.date}-${change.slotName}`}>{change.date} · {change.slotName}: {change.beforeName} → {change.afterName}</li>)}</ul>
              {swapNeedsImpactReview ? (
                <section aria-label="Shopping impact" className="mt-4 rounded border bg-app-surface p-3">
                  <h3 className="font-medium text-app-foreground">Shopping impact</h3>
                  {(swapPreview.preview.relatedOccurrences?.length ?? 0) > 0 && <div className="mt-2 text-sm text-app-foreground"><h4 className="font-medium">Later uses to review</h4><p>These later occurrences are unchanged. Confirm whether any depends on this meal before applying; no batch or leftover relationship is inferred.</p><ul className="mt-1 list-disc pl-5">{swapPreview.preview.relatedOccurrences?.map((item) => <li key={`${item.date}-${item.slotName}`}>{item.date} · {item.slotName}: {item.recipeName} · revision {item.recipeRevision}{item.locked ? " · locked" : ""}</li>)}</ul></div>}
                  {(swapPreview.preview.preparedBatchImpacts?.length ?? 0) > 0 && <div className="mt-2 text-sm text-app-foreground"><h4 className="font-medium">Prepared portions to review</h4><p>These portions remain in Kitchen inventory. Confirm whether this swap changes which meal should use them; the preview does not allocate or consume a batch.</p><ul className="mt-1 list-disc pl-5">{swapPreview.preview.preparedBatchImpacts?.map((item, index) => <li key={`${item.recipeId}-${item.recipeRevision}-${index}`}>{item.recipeName} · revision {item.recipeRevision} · batch {item.batchId}: {item.available} {item.unit} available</li>)}</ul></div>}
                  {swapPreview.preview.shoppingChanges.length ? <ul className="mt-2 space-y-2 text-sm text-app-foreground">{swapPreview.preview.shoppingChanges.map((change) => { const line = change.after ?? change.before; if (!line) return null; return <li key={change.key}><strong>{change.after ? "Added" : "Removed"}:</strong> {line.label} · Need: {line.need} · Stock: {line.stock} · Missing: {line.missing} · Packages: {line.packageCount} · Price: {line.price}</li>; })}</ul> : <p className="mt-1 text-sm text-app-muted-foreground">No shopping lines changed; dates affected: {swapPreview.affectedDates.join(", ")}.</p>}
                  <p className="mt-2 text-xs text-app-muted-foreground">This is a plan-derived review only; it does not reserve stock, mark items purchased, or change inventory.</p>
                </section>
              ) : <p className="mt-3 text-sm text-app-muted-foreground">This preview changes one meal slot and no shopping lines.</p>}
            </>
          )}
          <div className="mt-4 flex gap-2">
            <button type="button" className="min-h-11 rounded bg-app-primary px-4 font-medium text-app-primary-foreground" onClick={() => void (swapPreview ? applySwap() : makeSwapPreview())} disabled={!replacementId}>{swapPreview ? (swapNeedsImpactReview ? "Apply reviewed changes" : "Confirm swap") : "Preview swap"}</button>
            <button type="button" className="min-h-11 rounded border bg-app-surface px-4" onClick={() => { setSwapDate(""); setSwapSlotName(""); setSwapPreview(undefined); }}>Cancel</button>
          </div>
        </div>
      )}
      {error && state === "ready" && <p role="alert" className="rounded border border-app-danger bg-app-surface-muted p-4 text-app-danger">{error}</p>}
    </section>
  );
}

interface WeekOccurrenceActions {
  onSwap: (date: string, slotName: string) => void;
  onToggleLock: (date: string, slotName: string) => void;
  onSkip: (date: string, slotName: string) => void;
}

function MealOccurrence({ item, ...actions }: { item: PlanDraft["occurrences"][number] } & WeekOccurrenceActions) {
  const date = item.date;
  const slotName = item.slotName ?? "meal";
  const open = item.mode === "open" && !item.recipeId;
  return (
    <article className={`week-meal-card${item.locked ? " is-locked" : ""}`} data-week-occurrence="">
      <div className="week-meal-art" aria-hidden="true"><Leaf /><span>No photo</span></div>
      <div className="week-meal-copy">
        <span className="week-meal-slot">{slotName}</span>
        <strong>{item.recipeName || (open ? "Intentionally open" : "Recipe not selected")}</strong>
        <span className="week-meal-meta">{item.quantity ? `${item.quantity} portion${item.quantity === "1" ? "" : "s"}` : "Portion count unknown"}</span>
        <span className="week-meal-status">{item.locked ? "Locked" : open ? "Open slot" : item.recipeId ? "Planned" : "Not planned"}</span>
      </div>
      <div className="week-meal-actions">
        {item.recipeId && <button type="button" onClick={() => actions.onSwap(date, slotName)}>Swap meal</button>}
        {!open && item.recipeId && <button type="button" onClick={() => actions.onToggleLock(date, slotName)}>{item.locked ? "Unlock" : "Lock"}</button>}
        {item.recipeId && <button type="button" onClick={() => actions.onSkip(date, slotName)}>Skip</button>}
      </div>
    </article>
  );
}

function WeekBoard({ dates, slotNames, occurrences, unresolved, selectedDate, onSelectDate, ...actions }: {
  dates: string[];
  slotNames: string[];
  occurrences: PlanDraft["occurrences"];
  unresolved: PlanDraft["unresolved"];
  selectedDate: string;
  onSelectDate: (date: string) => void;
} & WeekOccurrenceActions) {
  const slots = slotNames.length ? slotNames : ["dinner"];
  return (
    <div className="week-board-wrap">
      <table className="week-board" aria-label="Meals planned across the week">
        <thead><tr><th scope="col">Meal</th>{dates.map((date) => <th key={date} scope="col"><button type="button" aria-pressed={selectedDate === date} onClick={() => onSelectDate(date)}><span>{formatWeekday(date)}</span><strong>{formatDay(date)}</strong></button></th>)}</tr></thead>
        <tbody>{slots.map((slot) => <tr key={slot}>
          <th scope="row">{slot}</th>
          {dates.map((date) => {
            const item = occurrences.find((occurrence) => occurrence.date === date && (occurrence.slotName ?? "meal") === slot);
            const unresolvedItem = unresolved.find((entry) => entry.date === date);
            return <td key={date}>{item ? <MealOccurrence item={item} {...actions} /> : <div className="week-no-occurrence">{unresolvedItem?.message ?? "No saved meal"}</div>}</td>;
          })}
        </tr>)}</tbody>
      </table>
    </div>
  );
}

function WeekAgenda({ dates, occurrences, unresolved, selectedDate, ...actions }: {
  dates: string[];
  occurrences: PlanDraft["occurrences"];
  unresolved: PlanDraft["unresolved"];
  selectedDate: string;
  onSwap: (date: string, slotName: string) => void;
  onToggleLock: (date: string, slotName: string) => void;
  onSkip: (date: string, slotName: string) => void;
}) {
  return (
    <ol className="week-phone-agenda" aria-label="Meals by day">
      {dates.map((date) => {
        const items = occurrences.filter((occurrence) => occurrence.date === date);
        const unresolvedItem = unresolved.find((entry) => entry.date === date);
        return <li key={date} className={selectedDate === date ? "is-selected" : ""}>
          <header><time dateTime={date}>{formatLongDate(date)}</time><span>{items.length ? `${items.length} planned` : unresolvedItem ? "Needs a choice" : "No saved meals"}</span></header>
          {items.length ? items.map((item) => <MealOccurrence key={`${date}-${item.slotName}`} item={item} {...actions} />) : <p>{unresolvedItem?.message ?? "No meal has been saved for this date."}</p>}
        </li>;
      })}
    </ol>
  );
}

function formatWeekday(date: string) {
  return new Intl.DateTimeFormat(undefined, { weekday: "short", timeZone: "UTC" }).format(new Date(`${date}T12:00:00Z`));
}
function formatDay(date: string) { return new Intl.DateTimeFormat(undefined, { day: "numeric", timeZone: "UTC" }).format(new Date(`${date}T12:00:00Z`)); }
function formatLongDate(date: string) { return new Intl.DateTimeFormat(undefined, { weekday: "long", month: "long", day: "numeric", timeZone: "UTC" }).format(new Date(`${date}T12:00:00Z`)); }
function formatShortDate(date: string) { return new Intl.DateTimeFormat(undefined, { month: "short", day: "numeric", timeZone: "UTC" }).format(new Date(`${date}T12:00:00Z`)); }

function isStalePlanConflict(err: unknown) {
  const value = err as { message?: unknown; code?: unknown };
  return /aborted|stale|changed elsewhere|revision/i.test(`${value?.message ?? err ?? ""} ${value?.code ?? ""}`);
}

function mergeWeekDraft(base: PlanDraft, mine: PlanDraft, latest: PlanDraft): { draft: PlanDraft; conflicts: string[] } {
  const conflicts: string[] = [];
  const equal = <T,>(a: T | undefined, b: T | undefined) => JSON.stringify(a) === JSON.stringify(b);
  function reconcile<T>(
    original: T[], local: T[], remote: T[], keyOf: (item: T) => string,
    labelOf: (key: string) => string,
  ) {
    const baseMap = new Map(original.map((item) => [keyOf(item), item]));
    const mineMap = new Map(local.map((item) => [keyOf(item), item]));
    const latestMap = new Map(remote.map((item) => [keyOf(item), item]));
    const keys = new Set([...baseMap.keys(), ...mineMap.keys(), ...latestMap.keys()]);
    const values: T[] = [];
    for (const key of keys) {
      const before = baseMap.get(key), localValue = mineMap.get(key), latestValue = latestMap.get(key);
      const localChanged = !equal(before, localValue), remoteChanged = !equal(before, latestValue);
      if (localChanged && remoteChanged && !equal(localValue, latestValue)) {
        conflicts.push(labelOf(key));
        continue;
      }
      const selected = localChanged ? localValue : latestValue;
      if (selected !== undefined) values.push(selected);
    }
    return values;
  }
  const occurrences = reconcile(base.occurrences, mine.occurrences, latest.occurrences,
    (item) => `${item.date}|${item.slotName ?? "meal"}`,
    (key) => key.replace("|", " · "));
  const unresolved = reconcile(base.unresolved, mine.unresolved, latest.unresolved,
    (item) => `${item.date}|${item.code}`,
    (key) => key.replace("|", " · "));
  return { draft: { ...latest, occurrences, unresolved }, conflicts };
}

function describeWeekChanges(before: PlanDraft | undefined, after: PlanDraft | undefined) {
  if (!before || !after) return [];
  const previous = new Map(before.occurrences.map((item) => [`${item.date}|${item.slotName ?? "meal"}`, item]));
  const current = new Map(after.occurrences.map((item) => [`${item.date}|${item.slotName ?? "meal"}`, item]));
  const output: string[] = [];
  for (const [key, item] of current) {
    const prior = previous.get(key);
    if (!prior) output.push(`${key.replace("|", " · ")}: added ${item.recipeName || "open slot"}`);
    else if (prior.recipeId !== item.recipeId || prior.locked !== item.locked || prior.mode !== item.mode || prior.quantity !== item.quantity) {
      output.push(`${key.replace("|", " · ")}: ${prior.recipeName || "open slot"} → ${item.recipeName || "open slot"}${prior.locked !== item.locked ? item.locked ? " (locked)" : " (unlocked)" : ""}`);
    }
  }
  for (const [key, item] of previous) if (!current.has(key)) output.push(`${key.replace("|", " · ")}: removed ${item.recipeName || "open slot"}`);
  return output;
}

function ChangeList({ changes, empty }: { changes: string[]; empty: string }) {
  return changes.length ? <ul>{changes.map((change) => <li key={change}>{change}</li>)}</ul> : <p>{empty}</p>;
}

function WeekReview({ draft, saved }: { draft: PlanDraft; saved?: PlanDraft }) {
  const previous = new Map((saved?.occurrences ?? []).map((item) => [`${item.date}|${item.slotName ?? "meal"}`, item]));
  const current = new Map(draft.occurrences.map((item) => [`${item.date}|${item.slotName ?? "meal"}`, item]));
  const changes: string[] = [];
  for (const [key, item] of current) {
    const before = previous.get(key);
    if (!before) changes.push(`${item.date} · ${item.slotName ?? "meal"}: added ${item.recipeName || "open slot"}`);
    else if (before.recipeId !== item.recipeId || before.locked !== item.locked || before.mode !== item.mode || before.quantity !== item.quantity) {
      changes.push(`${item.date} · ${item.slotName ?? "meal"}: ${before.recipeName || "open slot"} → ${item.recipeName || "open slot"}${before.locked !== item.locked ? (item.locked ? " (locked)" : " (unlocked)") : ""}`);
    }
  }
  for (const [key, item] of previous) if (!current.has(key)) changes.push(`${item.date} · ${item.slotName ?? "meal"}: removed ${item.recipeName || "open slot"}`);
  const unresolved = draft.unresolved ?? [];
  return <section aria-label="Plan changes" className="rounded-lg border border-app-accent bg-app-surface-muted p-4">
    <h2 className="font-semibold text-app-foreground">Review this week</h2>
    <p className="mt-1 text-sm text-app-foreground">{changes.length === 0 ? "No meal or lock changes from the saved week." : `${changes.length} meal or lock ${changes.length === 1 ? "change" : "changes"} to review.`} {unresolved.length > 0 && `${unresolved.length} date ${unresolved.length === 1 ? "needs" : "need"} a decision; no meal was assigned.`}</p>
    <details className="mt-2 text-sm text-app-foreground"><summary className="cursor-pointer font-medium">Show full-week changes and unresolved dates</summary>
      <ul className="mt-2 list-disc space-y-1 pl-5">{changes.map((change) => <li key={change}>{change}</li>)}{unresolved.map((item) => <li key={`${item.date}-${item.code}`}>{item.date}: {item.message}</li>)}{changes.length === 0 && unresolved.length === 0 && <li>No changes.</li>}</ul>
    </details>
  </section>;
}
