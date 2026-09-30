import { useCallback, useEffect, useState } from "react";

import { applyPlan, generatePlan, previewSwap, recordFeedback, undoFeedback, type PlanDraft } from "../../api/planning";
import { listRecipes } from "../../api/recipes";
import { getRecipe } from "../../api/recipes";
import { ensureWorkspace } from "../../api/workspace";
import { RecipeViewer } from "../recipe/RecipeViewer";
import { useDialogFocusTrap } from "../../lib/useDialogFocusTrap";
import { selectors } from "../../consts/selectors";

export function TodayPage() {
  const [draft, setDraft] = useState<PlanDraft>();
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");
  const [error, setError] = useState("");
  const [workspaceId, setWorkspaceId] = useState("");
  const [action, setAction] = useState<"idle" | "saving" | "saved">("idle");
  const [feedbackState, setFeedbackState] = useState<"unknown" | "recording" | "recorded">("unknown");
  const [recipes, setRecipes] = useState<{ id: string; name: string }[]>([]);
  const [swapOpen, setSwapOpen] = useState(false);
  const [replacementId, setReplacementId] = useState("");
  const [swapPreview, setSwapPreview] = useState<Awaited<ReturnType<typeof previewSwap>>>();
  const [selectedRecipe, setSelectedRecipe] = useState<Awaited<ReturnType<typeof getRecipe>>>();
  const occurrence = draft?.occurrences[0];
  const closeSwap = useCallback(() => { setSwapOpen(false); setSwapPreview(undefined); }, []);

  useDialogFocusTrap(swapOpen, closeSwap, "[role=\"dialog\"]");

  useEffect(() => {
    let mounted = true;
    const today = new Date().toISOString().slice(0, 10);
    void ensureWorkspace().then((workspace) => {
      if (mounted) setWorkspaceId(workspace.id);
      return generatePlan({ workspaceId: workspace.id, dates: [today], mealSlots: [
        { date: today, slotName: "breakfast" },
        { date: today, slotName: "lunch" },
        { date: today, slotName: "dinner" },
        { date: today, slotName: "snack", mode: "open" },
      ] });
    }).then((result) => {
      if (mounted) { setDraft(result); setState("ready"); }
    }).catch((err: unknown) => {
      if (mounted) { setError(err instanceof Error ? err.message : "Unable to load today’s plan."); setState("error"); }
    });
    return () => { mounted = false; };
  }, []);

  async function savePlan() {
    if (!draft || !workspaceId) return;
    setAction("saving");
    try { await applyPlan({ workspaceId, expectedRevision: draft.currentRevision, draft }); setAction("saved"); }
    catch (err) { setError(err instanceof Error ? err.message : "Unable to save today’s plan."); setAction("idle"); }
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
    try { setError(""); setSwapPreview(await previewSwap({ workspaceId, expectedRevision: draft.currentRevision, date: occurrence.date, replacementRecipeId: replacementId })); }
    catch (err) { setError(err instanceof Error ? err.message : "Unable to preview this swap."); }
  }

  async function applySwap() {
    if (!swapPreview || !workspaceId) return;
    try {
      const applied = await applyPlan({ workspaceId, expectedRevision: swapPreview.revision, draft: swapPreview.preview.draft });
      setDraft({ ...swapPreview.preview.draft, currentRevision: applied.revision }); setSwapOpen(false); setSwapPreview(undefined);
    } catch (err) { setError(err instanceof Error ? err.message : "Unable to apply this swap."); }
  }

  async function openRecipe() {
    if (!workspaceId || !occurrence?.recipeId) return;
    try { setError(""); setSelectedRecipe(await getRecipe(workspaceId, occurrence.recipeId)); }
    catch (err) { setError(err instanceof Error ? err.message : "Unable to load this recipe."); }
  }

  return (
    <section data-testid={selectors.pages.today} aria-labelledby="today-heading" className="flex flex-col gap-6">
      <header><p className="text-sm font-medium uppercase tracking-wide text-cyan-700">Daily / Today</p><h1 id="today-heading" className="text-3xl font-semibold text-slate-900">Your meals today</h1><p className="mt-2 text-slate-600">Breakfast, lunch, dinner, and open snack space stay visible. Unknown facts stay unknown until evidence exists.</p></header>
      {state === "loading" && <p role="status" className="rounded border border-slate-200 bg-white p-5 text-slate-600">Loading today’s plan…</p>}
      {state === "error" && <p role="alert" className="rounded border border-red-200 bg-red-50 p-5 text-red-800">{error}</p>}
      {state === "ready" && draft && <div className="grid gap-4 md:grid-cols-2">{draft.occurrences.map((item, index) => <article key={`${item.date}-${item.slotName ?? index}`} className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm"><p className="text-sm font-medium uppercase tracking-wide text-cyan-700">{item.slotName ?? "meal"}</p>{item.recipeName ? <><h2 className="mt-2 text-xl font-semibold text-slate-900">{item.recipeName}</h2><p className="mt-2 text-slate-600">{item.reason}</p></> : <><h2 className="mt-2 text-xl font-semibold text-slate-900">Open slot</h2><p className="mt-2 text-slate-600">This slot remains user-controlled; no recipe was inferred.</p></>}<div className="mt-5 flex flex-wrap gap-2">{item.recipeName && <button type="button" className="min-h-11 rounded border px-4" onClick={() => void openRecipe()}>See the recipe map</button>}</div></article>)}{draft.unresolved.map((item) => <div key={`${item.date}-${item.code}`} className="rounded border border-amber-200 bg-amber-50 p-5"><h2 className="font-semibold text-amber-950">{item.date}: no eligible meal</h2><p className="mt-2 text-amber-900">{item.message}</p></div>)}{draft.occurrences.length === 0 && draft.unresolved.length > 0 && <div className="rounded border border-dashed border-slate-300 bg-white p-6 md:col-span-2"><h2 className="font-semibold text-slate-900">No dinner planned yet</h2><p className="mt-2 text-slate-600">No eligible meal was available. Required rules were not relaxed to fill the slot.</p></div>}{draft.occurrences.length > 0 && <article className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm md:col-span-2"><p className="text-sm text-slate-600">{occurrence?.date ?? new Date().toISOString().slice(0, 10)} · plan actions</p><dl className="mt-4 grid gap-3 sm:grid-cols-4">{["Energy", "Protein", "Portion cost", "Time"].map((metric) => <div key={metric} className="rounded bg-slate-50 p-3"><dt className="text-xs uppercase tracking-wide text-slate-500">{metric}</dt><dd className="mt-1 text-sm text-slate-700">Unknown until entered or verified.</dd></div>)}</dl><div className="mt-4 flex flex-wrap gap-3"><button type="button" className="min-h-11 rounded bg-blue-700 px-4 font-medium text-white" onClick={() => void savePlan()} disabled={action === "saving"}>{action === "saving" ? "Saving…" : action === "saved" ? "Saved" : "Let’s make it"}</button>{occurrence?.recipeId && <><button type="button" className="min-h-11 rounded border px-4" onClick={() => void (feedbackState === "recorded" ? undoEaten() : markEaten())} disabled={feedbackState === "recording"}>{feedbackState === "recording" ? "Recording…" : feedbackState === "recorded" ? "Undo eaten" : "I ate this"}</button><button type="button" className="min-h-11 rounded border px-4" onClick={() => void openSwap()}>Swap meal</button></>}</div>{action === "saved" && <p role="status" className="mt-3 text-sm text-emerald-700">This plan is saved for the workspace.</p>}{feedbackState === "recorded" && <p role="status" className="mt-3 text-sm text-emerald-700">Recorded as eaten. This explicit feedback is separate from the scheduled plan.</p>}{error && action === "idle" && <p role="alert" className="mt-3 text-sm text-red-700">{error}</p>}</article>}</div>}
      {selectedRecipe && <RecipeViewer recipe={selectedRecipe} onClose={() => setSelectedRecipe(undefined)} />}
      {swapOpen && <div role="dialog" aria-labelledby="today-swap-heading" aria-modal="true" className="rounded-lg border border-cyan-200 bg-cyan-50 p-5"><h2 id="today-swap-heading" className="font-semibold text-slate-900">Preview a replacement</h2><label className="mt-4 block text-sm font-medium text-slate-800" htmlFor="today-replacement-recipe">Replacement meal</label><select id="today-replacement-recipe" className="mt-1 min-h-11 w-full rounded border bg-white px-3" value={replacementId} onChange={(event) => setReplacementId(event.target.value)}><option value="">Choose a meal</option>{recipes.map((recipe) => <option key={recipe.id} value={recipe.id}>{recipe.name}</option>)}</select>{swapPreview && <ul className="mt-4 space-y-1 text-sm text-slate-700">{swapPreview.preview.changes.map((change) => <li key={change.date}>{change.date}: {change.beforeName} → {change.afterName}</li>)}</ul>}<div className="mt-4 flex gap-2"><button type="button" className="min-h-11 rounded bg-blue-700 px-4 font-medium text-white" onClick={() => void (swapPreview ? applySwap() : makeSwapPreview())} disabled={!replacementId}>{swapPreview ? "Apply swap" : "Preview swap"}</button><button type="button" className="min-h-11 rounded border bg-white px-4" onClick={() => setSwapOpen(false)}>Cancel</button></div></div>}
    </section>
  );
}
