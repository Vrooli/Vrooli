import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
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
function agendaFromSearch(value: string | null, compact: boolean): "all" | "day" {
  if (value === "day") return "day";
  if (value === "all") return "all";
  return compact ? "day" : "all";
}
function exploreSlotPath(date: string, slot: string, revision: bigint, expectedRecipeId: string) {
  const params = new URLSearchParams({ date, slot, basePlanRevision: revision.toString(), expectedRecipeId });
  if (expectedRecipeId) params.set("replace", "occurrence");
  return `/meals/explore?${params.toString()}`;
}

export function WeekPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const requestedView = searchParams.get("view");
  const requestedAgenda = searchParams.get("range");
  const initialView = requestedView === "nutrition" || requestedView === "time-cost" ? requestedView : "meals";
  const initialAgenda = agendaFromSearch(requestedAgenda, typeof window !== "undefined" && window.innerWidth < 1100);
  const initialNutritionScope = searchParams.get("scope") === "recorded" || searchParams.get("scope") === "expected" ? searchParams.get("scope") as "recorded" | "expected" : "planned";
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
  const [nutritionScope, setNutritionScope] = useState<"planned" | "recorded" | "expected">(initialNutritionScope);
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
  const [appliedChanges, setAppliedChanges] = useState<string[]>([]);
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

  function updateNutritionScope(next: "planned" | "recorded" | "expected") {
    setNutritionScope(next);
    setSearchParams((current) => { const updated = new URLSearchParams(current); updated.set("scope", next); return updated; });
  }

  useEffect(() => {
    const nextView = searchParams.get("view");
    const nextAgenda = searchParams.get("range");
    setView(nextView === "nutrition" || nextView === "time-cost" ? nextView : "meals");
    setAgenda(agendaFromSearch(nextAgenda, compact));
    const nextScope = searchParams.get("scope");
    setNutritionScope(nextScope === "recorded" || nextScope === "expected" ? nextScope : "planned");
    const requestedDate = searchParams.get("date");
    if (requestedDate) setSelectedDate(requestedDate);
  }, [searchParams, compact]);

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
      setAppliedChanges([]);
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
    setAppliedChanges([]);
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
    setAppliedChanges([]);
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
      setAppliedChanges(describeWeekChanges(savedDraft, saved)); setDraft(saved); setSavedDraft(saved); setHasPlan(true); setSwapDate(""); setSwapSlotName(""); setSwapPreview(undefined); setStaleSwapIntent(undefined); setStaleConflict(false); setRemoteDraft(undefined); setError("");
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
    setAppliedChanges([]);
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
    const savedChanges = describeWeekChanges(savedDraft, draft);
    setSaving(true);
    try {
      const applied = await applyPlan({ workspaceId, expectedRevision: draft.currentRevision, draft });
      const saved = { ...draft, currentRevision: applied.revision };
      setDraft(saved);
      setSavedDraft(saved);
      setHasPlan(true);
      setAppliedChanges(savedChanges);
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
  const appliedDates = Array.from(new Set(appliedChanges.map((change) => change.match(/^\d{4}-\d{2}-\d{2}/)?.[0]).filter((date): date is string => Boolean(date)))).sort();
  const appliedDateLabels = appliedDates.map(formatWeekdayLong);
  const appliedHeading = appliedDates.length > 1 ? `Your ${appliedDates.length}-day change is saved` : "Your plan change is saved";
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
          {appliedChanges.length > 0 && <section className="week-applied-review" aria-label="Saved plan changes">
            <h2>{appliedHeading}</h2>
            <div className="week-applied-banner" role="status"><strong>{appliedDateLabels.length > 1 ? `${appliedDateLabels.join(" and ")} updated together` : "Plan revision saved"}</strong><p>The plan changes are saved. They have not purchased groceries or consumed inventory.</p></div>
            <div className="week-applied-summary">
              <h3>The plan after your choice</h3>
              <p>Saved changes:</p>
              <ul>{appliedChanges.map((change) => <li key={change}>{change}</li>)}</ul>
            </div>
            <aside><h3>Review what follows</h3><p>Plan changes do not purchase groceries or consume inventory. Review the current grocery quantities separately.</p><a href="/groceries">Review groceries <ChevronRight aria-hidden="true" /></a></aside>
          </section>}
          {staleConflict && <section className="week-stale-review" aria-label="Stale plan review">
            <h2>Your draft is still here</h2>
            <div className="week-stale-banner"><strong>Plan changed while you were reviewing</strong><p>Your local draft is retained. Review the latest saved changes before deciding how to continue; nothing has been applied.</p></div>
            {staleSwapIntent && <div className="week-stale-swap-note"><strong>The proposed swap was not applied.</strong><p>Its previous shopping preview is shown for context only; refresh it before applying any replacement.</p>{staleSwapIntent.shoppingChanges.length > 0 && <ul>{staleSwapIntent.shoppingChanges.map((change) => { const line = change.after ?? change.before; return line ? <li key={change.key}>{change.after ? "Added" : "Removed"}: {line.label} · Need {line.need} · Stock {line.stock} · Missing {line.missing} · Packages {line.packageCount} · Price {line.price}</li> : null; })}</ul>}</div>}
            {!remoteDraft && <button type="button" className="week-primary-button" disabled={refreshingConflict} onClick={() => void reviewLatestWeek()}>{refreshingConflict ? "Loading latest week…" : "Review latest week"}</button>}
            {remoteDraft && <>
              <div className="week-stale-diffs">
                <div><h3>Keep the change you were considering</h3><p>Your local edits are still a draft.</p><ChangeList changes={describeWeekChanges(savedDraft, draft)} empty="No local occurrence changes." /></div>
                <div><h3>Review only what changed</h3><p>These changes arrived in the latest saved week.</p><ChangeList changes={describeWeekChanges(savedDraft, remoteDraft)} empty="The saved occurrences are unchanged; only the revision moved." /></div>
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
          <nav aria-label="Choose a day" className={`week-date-strip${view !== "meals" ? " is-visible" : ""}`}>
            {dates.map((date) => <button key={date} type="button" aria-pressed={activeDate === date} onClick={() => updateSelectedDate(date)}>
              <span>{formatWeekday(date)}</span><strong>{formatDay(date)}</strong>
            </button>)}
          </nav>
          {view === "nutrition" && <div className="week-scope-tabs" role="group" aria-label="Nutrition scope">
            {(["planned", "recorded", "expected"] as const).map((scope) => <button key={scope} type="button" aria-pressed={nutritionScope === scope} onClick={() => updateNutritionScope(scope)}>{scope[0]?.toUpperCase()}{scope.slice(1)}</button>)}
          </div>}
          {view === "meals" && (
            <>
              {!compact && <WeekBoard
                dates={dates}
                slotNames={slotNames}
                occurrences={draft.occurrences}
                unresolved={draft.unresolved}
                selectedDate={activeDate}
                planRevision={draft.currentRevision}
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
                planRevision={draft.currentRevision}
                supplements={supplements}
                catalog={catalog}
                onSwap={openSwap}
                onToggleLock={toggleLock}
                onSkip={skipOccurrence}
              />}
            </>
          )}
          {view === "nutrition" && (
            <div className="week-context-layout">
              <ol aria-label="Daily nutrition scopes" className="week-evidence-list">
                {visibleDates.map((date) => {
                  const meals = draft.occurrences.filter((item) => item.date === date && item.recipeName).map((item) => item.recipeName);
                  const recorded = intakes.filter((event) => event.date === date);
                  const scheduled = supplements.filter((schedule) => scheduleAppliesOnLocalDate(schedule, date));
                  return <li key={date}>
                    <time dateTime={date}>{formatLongDate(date)}</time>
                    <div>
                      {nutritionScope === "planned" && <>
                        <p><strong>Planned meals:</strong> {meals.length ? meals.join(", ") : "no selected meals"}.</p>
                        <p>Nutrient totals are unknown until verified recipe evidence exists; planned meals are not recorded as eaten.</p>
                      </>}
                      {nutritionScope === "recorded" && <p><strong>Recorded intake:</strong> {recorded.length ? recorded.map((event) => `${event.nutrientId} ${event.amount} ${event.unit}`).join(", ") : "no recorded intake"}.</p>}
                      {nutritionScope === "expected" && <>
                        <p><strong>Planned meals:</strong> {meals.length ? meals.join(", ") : "no selected meals"}; recipe nutrient contributions remain unknown.</p>
                        <p><strong>Expected supplements:</strong> {scheduled.length ? "confirmed schedule contributions below; no dose is recorded as taken" : "no confirmed supplement schedule applies"}.</p>
                        {scheduled.map((item) => <p key={`${item.id}:${item.revision}`}>{scheduledSupplementNutrients(item, catalog).map((contribution, index) => <span key={`${contribution.nutrientId}:${index}`}>{index > 0 ? "; " : ""}{contribution.productName} · {item.dose} {item.doseUnit} scheduled · {contribution.nutrientId ? `${contribution.nutrientId}: ` : "nutrient: "}{contribution.unresolvedReason || `${contribution.amount} ${contribution.unit} expected (${contribution.evidence || "catalog evidence"})`}</span>)}</p>)}
                      </>}
                    </div>
                  </li>;
                })}
              </ol>
              <aside className="week-context-summary">
                <h2>Planned is not recorded</h2>
                <p>{intakes.some((event) => event.date === activeDate) ? "Recorded intake appears separately below; a planned meal is not counted as eaten." : "No intake has been recorded for this day."}</p>
                <h3>Nutrition and bounds</h3>
                <p>Recipe nutrition remains unknown until verified recipe evidence is available. No total or personal target is inferred from a meal name.</p>
                <p>Confirmed supplement schedules appear as expected contributions below; they are not recorded as taken.</p>
                <a href="/meals">Review meals <ChevronRight aria-hidden="true" /></a>
              </aside>
            </div>
          )}
          {view === "time-cost" && (
            <div className="week-context-layout">
              <ol aria-label="Daily time and cost coverage" className="week-evidence-list">
                {visibleDates.map((date) => {
                  const meals = draft.occurrences.filter((item) => item.date === date);
                  return <li key={date}>
                    <time dateTime={date}>{formatLongDate(date)}</time>
                    <div className="week-time-day">
                      {meals.map((item) => <article className="week-time-item" key={`${item.date}-${item.slotName}`}>
                        <strong>{item.slotName || "Meal"} · {item.recipeName || (item.mode === "open" ? "Intentionally open" : "Recipe not selected")}</strong>
                        <p>{item.quantity ? `${item.quantity} planned portion${item.quantity === "1" ? "" : "s"}` : "Portion count unknown"} · Prep and cook time: unknown until a timed recipe estimate is verified.</p>
                      </article>)}
                      {!meals.length && <p>No saved meal time is available for this date.</p>}
                      <p>Portion cost, checkout total, and actual spend: unknown without complete quantity and price coverage.</p>
                    </div>
                  </li>;
                })}
              </ol>
              <aside className="week-context-summary">
                <h2>Time & cost coverage</h2>
                <p>Active time and elapsed duration remain unknown until timed recipe estimates are verified.</p>
                <h3>Checkout & portion cost</h3>
                <p>Prices are not complete for this plan, so no checkout estimate, portion cost or actual spend is shown.</p>
                <a href="/groceries">Review grocery quantities <ChevronRight aria-hidden="true" /></a>
                <small>No calendar schedule or duration is inferred from servings.</small>
              </aside>
            </div>
          )}
        </>
      )}
      {swapDate && (
        <div role="dialog" aria-labelledby="swap-heading" aria-modal="true" className="rounded-lg border border-app-accent bg-app-surface-muted p-5">
          <h2 id="swap-heading" className="font-semibold text-app-foreground">{swapPreview ? "Review your changes" : "What sounds better?"}</h2>
          <p className="mt-1 text-sm text-app-foreground">{swapPreview ? "Nothing has been applied yet. Review the current and proposed meals and their effects before saving." : `Preview changes for ${swapDate} · ${swapSlotName} before applying them.`}</p>
          <label className="mt-4 block text-sm font-medium text-app-foreground" htmlFor="replacement-recipe">Replacement meal</label>
          <select id="replacement-recipe" className="mt-1 min-h-11 w-full rounded border bg-app-surface px-3" value={replacementId} onChange={(event) => setReplacementId(event.target.value)}>
            <option value="">Choose a meal</option>{recipes.map((recipe) => <option key={recipe.id} value={recipe.id}>{recipe.name}</option>)}
          </select>
          {swapPreview && (
            <>
              <div className="week-swap-review-cards" role="region" aria-label="Current and proposed meals">
                {swapPreview.preview.changes.map((change) => <div className="week-swap-review-pair" key={`${change.date}-${change.slotName}`}>
                  <article className="week-swap-meal-card"><span>Current · {formatShortDate(change.date)} · {change.slotName}</span><div className="week-swap-meal-art"><Leaf aria-hidden="true" /><small>No photo</small></div><h3>{change.beforeName || "No meal assigned"}</h3><p>Saved plan</p></article>
                  <article className="week-swap-meal-card is-proposed"><span>Proposed · {formatShortDate(change.date)} · {change.slotName}</span><div className="week-swap-meal-art"><Leaf aria-hidden="true" /><small>No photo</small></div><h3>{change.afterName || "No meal assigned"}</h3><p>Not applied</p></article>
                </div>)}
              </div>
              <ul className="week-swap-change-list">{swapPreview.preview.changes.map((change) => <li key={`${change.date}-${change.slotName}`}>{change.date} · {change.slotName}: {change.beforeName} → {change.afterName}</li>)}</ul>
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
            <button type="button" className="min-h-11 rounded border bg-app-surface px-4" onClick={() => { setSwapDate(""); setSwapSlotName(""); setSwapPreview(undefined); }}>{swapPreview ? "Keep current plan" : "Cancel"}</button>
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

function MealOccurrence({ item, planRevision, ...actions }: { item: PlanDraft["occurrences"][number]; planRevision: bigint } & WeekOccurrenceActions) {
  const date = item.date;
  const slotName = item.slotName ?? "meal";
  const open = item.mode === "open" && !item.recipeId;
  return (
    <article className={`week-meal-card${item.locked ? " is-locked" : ""}`} data-week-occurrence="">
      <div className="week-meal-art"><Leaf aria-hidden="true" /><span>No photo</span></div>
      <div className="week-meal-copy">
        <span className="week-meal-slot">{slotName}</span>
        <strong>{item.recipeName || (open ? "Intentionally open" : "Recipe not selected")}</strong>
        <span className="week-meal-meta">{item.quantity ? `${item.quantity} portion${item.quantity === "1" ? "" : "s"}` : "Portion count unknown"} · Time not set</span>
        {(item.locked || open || !item.recipeId) && <span className={`week-meal-status${item.locked ? " is-locked" : ""}`}>{item.locked ? "Locked" : open ? "Open slot" : "Choose a meal"}</span>}
      </div>
      <div className="week-meal-actions">
        <Link to={exploreSlotPath(date, slotName, planRevision, item.recipeId)}>Explore options</Link>
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
  planRevision: bigint;
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
            return <td key={date}>{item ? <MealOccurrence item={item} {...actions} /> : <div className="week-no-occurrence"><span>{unresolvedItem?.message ?? "No meal chosen"}</span><Link to={exploreSlotPath(date, slot, actions.planRevision, "")}>Explore meals</Link></div>}</td>;
          })}
        </tr>)}</tbody>
      </table>
    </div>
  );
}

function WeekAgenda({ dates, occurrences, unresolved, selectedDate, supplements, catalog, planRevision, onSwap, onToggleLock, onSkip }: {
  dates: string[];
  occurrences: PlanDraft["occurrences"];
  unresolved: PlanDraft["unresolved"];
  selectedDate: string;
  supplements: SupplementSchedule[];
  catalog: CatalogRevision[];
  planRevision: bigint;
  onSwap: (date: string, slotName: string) => void;
  onToggleLock: (date: string, slotName: string) => void;
  onSkip: (date: string, slotName: string) => void;
}) {
  return (
    <ol className="week-phone-agenda" aria-label="Meals by day">
      {dates.map((date) => {
        const items = occurrences.filter((occurrence) => occurrence.date === date);
        const unresolvedItem = unresolved.find((entry) => entry.date === date);
        const assignedCount = items.filter((item) => Boolean(item.recipeId)).length;
        const openCount = items.filter((item) => item.mode === "open" && !item.recipeId).length;
        const toChooseCount = items.length - assignedCount - openCount;
        const scheduled = supplements.filter((item) => scheduleAppliesOnLocalDate(item, date));
        return <li key={date} className={selectedDate === date ? "is-selected" : ""}>
          <header><time dateTime={date}>{formatLongDate(date)}</time><div className="week-day-heading-meta"><span className="week-day-count">{items.length ? `${assignedCount} assigned${openCount ? ` · ${openCount} open` : ""}${toChooseCount ? ` · ${toChooseCount} to choose` : ""}` : unresolvedItem ? "Needs a choice" : "No saved meals"}</span><span className="week-day-full-day">Full day</span></div></header>
          {items.length ? items.map((item) => <MealOccurrence key={`${date}-${item.slotName}`} item={item} planRevision={planRevision} onSwap={onSwap} onToggleLock={onToggleLock} onSkip={onSkip} />) : <p>{unresolvedItem?.message ?? "No meal has been saved for this date."} <Link to={exploreSlotPath(date, "dinner", planRevision, "")}>Explore dinner options</Link></p>}
          <section className="week-agenda-supplements" aria-label={`Supplement schedule for ${formatLongDate(date)}`}>
            <header><strong>Supplements</strong><span>Optional</span></header>
            {scheduled.length ? scheduled.map((item) => <p key={`${item.id}:${item.revision}`}>{scheduledSupplementNutrients(item, catalog).map((contribution, index) => <span key={`${contribution.nutrientId}:${index}`}>{index > 0 ? "; " : ""}{contribution.productName} · {item.dose} {item.doseUnit} expected · {contribution.unresolvedReason || `${contribution.nutrientId ? `${contribution.nutrientId}: ` : ""}${contribution.amount} ${contribution.unit} expected (${contribution.evidence})`}</span>)}</p>) : <p>No confirmed supplement schedule applies.</p>}
            <small>Expected schedule only; no dose is recorded as taken.</small>
          </section>
        </li>;
      })}
    </ol>
  );
}

function formatWeekday(date: string) {
  return new Intl.DateTimeFormat(undefined, { weekday: "short", timeZone: "UTC" }).format(new Date(`${date}T12:00:00Z`));
}
function formatWeekdayLong(date: string) {
  return new Intl.DateTimeFormat(undefined, { weekday: "long", timeZone: "UTC" }).format(new Date(`${date}T12:00:00Z`));
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
