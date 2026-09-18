import { useEffect, useMemo, useState } from "react";
import { applyPlan, generatePlan, previewSwap, type PlanDraft } from "../../api/planning";
import { listRecipes } from "../../api/recipes";
import { ensureWorkspace } from "../../api/workspace";
import { exportWeeklyPDF } from "../../api/portability";

function datesFrom(start: Date): string[] { return Array.from({ length: 7 }, (_, index) => { const date = new Date(start); date.setDate(start.getDate() + index); return date.toISOString().slice(0, 10); }); }

export function WeekPage() {
  const [start, setStart] = useState(() => { const date = new Date(); date.setHours(0, 0, 0, 0); return date; });
  const [workspaceId, setWorkspaceId] = useState("");
  const [draft, setDraft] = useState<PlanDraft>();
  const [recipes, setRecipes] = useState<{ id: string; name: string }[]>([]);
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");
  const [error, setError] = useState("");
  const [swapDate, setSwapDate] = useState("");
  const [replacementId, setReplacementId] = useState("");
  const [swapPreview, setSwapPreview] = useState<Awaited<ReturnType<typeof previewSwap>>>();
  const [exporting, setExporting] = useState(false);
  const dates = useMemo(() => datesFrom(start), [start]);

  useEffect(() => {
    let mounted = true;
    setState("loading");
    void ensureWorkspace().then(async (workspace) => {
      const [result, available] = await Promise.all([generatePlan({ workspaceId: workspace.id, dates }), listRecipes(workspace.id)]);
      if (mounted) { setWorkspaceId(workspace.id); setDraft(result); setRecipes(available.map((recipe) => ({ id: recipe.id, name: recipe.name }))); setState("ready"); }
    }).catch((err: unknown) => { if (mounted) { setError(err instanceof Error ? err.message : "Unable to load this week."); setState("error"); } });
    return () => { mounted = false; };
  }, [dates]);

  async function makeSwapPreview() {
    if (!draft || !workspaceId || !swapDate || !replacementId) return;
    try { setError(""); setSwapPreview(await previewSwap({ workspaceId, expectedRevision: draft.currentRevision, date: swapDate, replacementRecipeId: replacementId })); } catch (err) { setError(err instanceof Error ? err.message : "Unable to preview this swap."); }
  }

  async function applySwap() {
    if (!swapPreview || !workspaceId) return;
    try { const applied = await applyPlan({ workspaceId, expectedRevision: swapPreview.revision, draft: swapPreview.preview.draft }); setDraft({ ...swapPreview.preview.draft, currentRevision: applied.revision }); setSwapDate(""); setSwapPreview(undefined); } catch (err) { setError(err instanceof Error ? err.message : "Unable to apply this swap."); }
  }

  async function downloadPDF() {
    if (!draft || !workspaceId) return;
    setExporting(true);
    try { const result = await exportWeeklyPDF({ workspaceId, expectedRevision: draft.currentRevision }); const bytes = new Uint8Array(result.content); const url = URL.createObjectURL(new Blob([bytes.buffer as ArrayBuffer], { type: "application/pdf" })); const anchor = document.createElement("a"); anchor.href = url; anchor.download = result.filename; anchor.click(); URL.revokeObjectURL(url); } catch (err) { setError(err instanceof Error ? err.message : "Unable to export this week."); } finally { setExporting(false); }
  }

  return <section aria-labelledby="week-heading" className="flex flex-col gap-6">
    <header className="flex flex-wrap items-end justify-between gap-3"><div><p className="text-sm font-medium uppercase tracking-wide text-cyan-700">Daily / Week</p><h1 id="week-heading" className="text-3xl font-semibold text-slate-900">Your dinner week</h1><p className="mt-2 text-slate-600">Seven consecutive local dates; locks, skips, and swaps remain explicit.</p></div><div className="flex gap-2"><button type="button" className="min-h-11 rounded border px-3" onClick={() => setStart((current) => new Date(current.getTime() - 7 * 86400000))}>Previous week</button><button type="button" className="min-h-11 rounded border px-3" onClick={() => setStart((current) => new Date(current.getTime() + 7 * 86400000))}>Next week</button><button type="button" className="min-h-11 rounded border px-3" onClick={() => void downloadPDF()} disabled={exporting || !draft}>{exporting ? "Exporting…" : "Download weekly PDF"}</button></div></header>
    {state === "loading" && <p role="status" className="rounded border bg-white p-5 text-slate-600">Loading this week…</p>}
    {state === "error" && <p role="alert" className="rounded border border-red-200 bg-red-50 p-5 text-red-800">{error}</p>}
    {state === "ready" && <ol className="divide-y rounded-lg border border-slate-200 bg-white">{dates.map((date) => { const item = draft?.occurrences.find((occurrence) => occurrence.date === date); return <li key={date} className="grid gap-2 p-4 sm:grid-cols-[10rem_1fr_auto_auto]"><time className="font-medium text-slate-700" dateTime={date}>{date}</time><span className="text-slate-900">{item?.recipeName ?? "Open dinner slot — unknown"}</span><span className="text-sm text-slate-600">{item?.locked ? "Locked" : item ? "Planned" : "Unresolved"}</span>{item && <button type="button" className="min-h-10 rounded border px-3 text-sm" onClick={() => { setSwapDate(date); setReplacementId(""); setSwapPreview(undefined); }}>Swap meal</button>}</li>; })}</ol>}
    {swapDate && <div role="dialog" aria-labelledby="swap-heading" aria-modal="true" className="rounded-lg border border-cyan-200 bg-cyan-50 p-5"><h2 id="swap-heading" className="font-semibold text-slate-900">What sounds better?</h2><p className="mt-1 text-sm text-slate-700">Preview changes for {swapDate} before applying them.</p><label className="mt-4 block text-sm font-medium text-slate-800" htmlFor="replacement-recipe">Replacement meal</label><select id="replacement-recipe" className="mt-1 min-h-11 w-full rounded border bg-white px-3" value={replacementId} onChange={(event) => setReplacementId(event.target.value)}><option value="">Choose a meal</option>{recipes.map((recipe) => <option key={recipe.id} value={recipe.id}>{recipe.name}</option>)}</select>{swapPreview && <ul className="mt-4 space-y-1 text-sm text-slate-700">{swapPreview.preview.changes.map((change) => <li key={change.date}>{change.date}: {change.beforeName} → {change.afterName}</li>)}</ul>}<div className="mt-4 flex gap-2"><button type="button" className="min-h-11 rounded bg-blue-700 px-4 font-medium text-white" onClick={() => void (swapPreview ? applySwap() : makeSwapPreview())} disabled={!replacementId}>{swapPreview ? "Apply swap" : "Preview swap"}</button><button type="button" className="min-h-11 rounded border bg-white px-4" onClick={() => { setSwapDate(""); setSwapPreview(undefined); }}>Cancel</button></div></div>}
    {error && state === "ready" && <p role="alert" className="rounded border border-red-200 bg-red-50 p-4 text-red-800">{error}</p>}
  </section>;
}
