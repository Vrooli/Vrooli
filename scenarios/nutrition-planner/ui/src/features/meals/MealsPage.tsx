import { useEffect, useState, type FormEvent } from "react";
import type { Recipe } from "@vrooli/proto-types/nutrition-planner/v1/recipe/recipe_pb";
import { createRecipe, listRecipes } from "../../api/recipes";
import { ensureWorkspace } from "../../api/workspace";
import { RecipeViewer } from "../recipe/RecipeViewer";
import { exportRecipePDF } from "../../api/portability";

export function MealsPage() {
  const [recipes, setRecipes] = useState<Recipe[]>([]);
  const [name, setName] = useState("");
  const [notes, setNotes] = useState("");
  const [sourceUrl, setSourceUrl] = useState("");
  const [state, setState] = useState<"loading" | "ready" | "saving" | "saved" | "error">("loading");
  const [error, setError] = useState("");
  const [workspaceId, setWorkspaceId] = useState("");
  const [selectedRecipe, setSelectedRecipe] = useState<Recipe | undefined>();
  const [exportingId, setExportingId] = useState("");

  useEffect(() => {
    let mounted = true;
    void ensureWorkspace().then((workspace) => listRecipes(workspace.id).then((items) => { if (mounted) { setWorkspaceId(workspace.id); setRecipes(items); setState("ready"); } })).catch((err: unknown) => { if (mounted) { setError(err instanceof Error ? err.message : "Unable to load saved meals."); setState("error"); } });
    return () => { mounted = false; };
  }, []);

  async function save(event: FormEvent) {
    event.preventDefault();
    if (!name.trim()) { setError("Give this meal a name before saving."); setState("error"); return; }
    setState("saving"); setError("");
    if (!workspaceId) { setError("Your workspace is still loading. Try again in a moment."); setState("error"); return; }
    try { const recipe = await createRecipe({ workspaceId, name: name.trim(), notes, sourceUrl, originalText: notes, sourceType: sourceUrl ? "url" : "manual" }); setRecipes((current) => [recipe, ...current]); setName(""); setNotes(""); setSourceUrl(""); setState("saved"); }
    catch (err: unknown) { setError(err instanceof Error ? err.message : "The meal was not saved. Your draft is still here."); setState("error"); }
  }

  async function downloadPDF(recipe: Recipe) {
    setExportingId(recipe.id); setError("");
    try { const result = await exportRecipePDF({ workspaceId, recipeId: recipe.id }); const bytes = new Uint8Array(result.content); const url = URL.createObjectURL(new Blob([bytes.buffer], { type: "application/pdf" })); const anchor = document.createElement("a"); anchor.href = url; anchor.download = result.filename; anchor.click(); URL.revokeObjectURL(url); } catch (err: unknown) { setError(err instanceof Error ? err.message : "Unable to export this recipe."); } finally { setExportingId(""); }
  }

  return <section aria-labelledby="meals-heading" className="flex flex-col gap-6">
    <header><p className="text-sm font-medium uppercase tracking-wide text-cyan-700">Daily / Meals</p><h1 id="meals-heading" className="text-3xl font-semibold text-slate-900">Your meal collection</h1><p className="mt-2 max-w-2xl text-slate-600">Capture the name first. Nutrition, cost, yield, and preparation stay honestly unknown until you add evidence.</p></header>
    <div className="grid gap-6 lg:grid-cols-[minmax(0,0.8fr)_minmax(0,1.2fr)]">
      <form onSubmit={save} className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm" aria-label="Quick capture meal">
        <h2 className="text-lg font-semibold text-slate-900">Quick capture</h2>
        <label className="mt-4 block text-sm font-medium text-slate-700" htmlFor="meal-name">Meal name</label><input id="meal-name" value={name} onChange={(e) => setName(e.target.value)} className="mt-1 min-h-11 w-full rounded border border-slate-300 px-3" placeholder="e.g. Lentil salad" />
        <label className="mt-4 block text-sm font-medium text-slate-700" htmlFor="meal-notes">Notes or pasted text</label><textarea id="meal-notes" value={notes} onChange={(e) => setNotes(e.target.value)} className="mt-1 min-h-28 w-full rounded border border-slate-300 p-3" placeholder="Keep the original rough notes; refine later." />
        <label className="mt-4 block text-sm font-medium text-slate-700" htmlFor="meal-url">Source URL (optional)</label><input id="meal-url" value={sourceUrl} onChange={(e) => setSourceUrl(e.target.value)} className="mt-1 min-h-11 w-full rounded border border-slate-300 px-3" placeholder="https://…" />
        {error && <p role="alert" className="mt-4 rounded border border-red-200 bg-red-50 p-3 text-sm text-red-800">{error}</p>}
        {state === "saved" && <p role="status" className="mt-4 rounded border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-800">Saved to your workspace.</p>}
        <button type="submit" disabled={state === "saving"} className="mt-5 min-h-11 rounded bg-blue-700 px-4 font-medium text-white disabled:opacity-60">{state === "saving" ? "Saving…" : "Save draft"}</button>
      </form>
      <div>{selectedRecipe && <RecipeViewer recipe={selectedRecipe} onClose={() => setSelectedRecipe(undefined)} />}<div className="mb-3 flex items-baseline justify-between"><h2 className="text-lg font-semibold text-slate-900">Saved meals</h2><span className="text-sm text-slate-600">{state === "loading" ? "Loading…" : `${recipes.length} ${recipes.length === 1 ? "meal" : "meals"}`}</span></div>
        {state === "loading" && <p role="status" className="rounded border border-slate-200 bg-white p-5 text-slate-600">Loading your collection…</p>}
        {state !== "loading" && recipes.length === 0 && <p className="rounded border border-dashed border-slate-300 bg-white p-8 text-slate-600">No saved meals yet. Start with a name; incomplete evidence is allowed.</p>}
        <ul className="space-y-3">{recipes.map((recipe) => <li key={recipe.id} className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm"><div className="flex items-start justify-between gap-3"><div><h3 className="font-semibold text-slate-900">{recipe.name}</h3><p className="mt-1 text-sm text-slate-600">Status: {recipe.status || "Draft"} · Revision {recipe.revision.toString()}</p></div><span className="rounded-full bg-amber-50 px-2 py-1 text-xs font-medium text-amber-800">incomplete</span></div>{recipe.notes && <p className="mt-3 whitespace-pre-wrap text-sm text-slate-700">{recipe.notes}</p>}<p className="mt-3 text-xs text-slate-500">Nutrition · cost · yield: Unknown until entered or verified.</p><div className="mt-3 flex flex-wrap gap-2"><button type="button" className="min-h-11 rounded border border-slate-300 px-3 text-sm" onClick={() => setSelectedRecipe(recipe)}>Open recipe</button><button type="button" className="min-h-11 rounded border border-slate-300 px-3 text-sm" onClick={() => void downloadPDF(recipe)} disabled={exportingId === recipe.id}>{exportingId === recipe.id ? "Exporting…" : "Download recipe PDF"}</button></div></li>)}</ul>
      </div>
    </div>
  </section>;
}
