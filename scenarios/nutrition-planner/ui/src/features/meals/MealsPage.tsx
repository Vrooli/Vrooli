import { useEffect, useMemo, useState, type FormEvent } from "react";
import type { Recipe } from "@vrooli/proto-types/nutrition-planner/v1/recipe/recipe_pb";
import { Link } from "react-router-dom";
import { createRecipe, listRecipes, updateRecipe } from "../../api/recipes";
import { ensureWorkspace } from "../../api/workspace";
import { exportRecipePDF } from "../../api/portability";

function recipeStatus(recipe: Recipe) {
  return recipe.status ? recipe.status.replace(/_/g, " ") : "Draft";
}

export function MealsPage() {
  const [recipes, setRecipes] = useState<Recipe[]>([]);
  const [name, setName] = useState("");
  const [notes, setNotes] = useState("");
  const [sourceUrl, setSourceUrl] = useState("");
  const [state, setState] = useState<"loading" | "ready" | "saving" | "saved" | "error">("loading");
  const [error, setError] = useState("");
  const [workspaceId, setWorkspaceId] = useState("");
  const [editingRecipe, setEditingRecipe] = useState<Recipe | undefined>();
  const [exportingId, setExportingId] = useState("");
  const [query, setQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");

  const filteredRecipes = useMemo(() => {
    const needle = query.trim().toLocaleLowerCase();
    return recipes.filter((recipe) => {
      const matchesQuery = !needle || [recipe.name, recipe.notes, recipe.originalText, recipe.sourceUrl,
        ...recipe.ingredients.map((ingredient) => ingredient.name), ...recipe.groups]
        .some((value) => value.toLocaleLowerCase().includes(needle));
      const matchesStatus = statusFilter === "all" || (statusFilter === "drafts"
        ? !recipe.status || recipe.status.toLocaleLowerCase() === "draft"
        : recipe.status.toLocaleLowerCase() === statusFilter);
      return matchesQuery && matchesStatus;
    });
  }, [recipes, query, statusFilter]);

  useEffect(() => {
    let mounted = true;
    void ensureWorkspace().then((workspace) => listRecipes(workspace.id).then((items) => {
      if (mounted) { setWorkspaceId(workspace.id); setRecipes(items); setState("ready"); }
    })).catch((err: unknown) => {
      if (mounted) { setError(err instanceof Error ? err.message : "Unable to load saved meals."); setState("error"); }
    });
    return () => { mounted = false; };
  }, []);

  async function save(event: FormEvent) {
    event.preventDefault();
    if (!name.trim()) { setError("Give this meal a name before saving."); setState("error"); return; }
    setState("saving"); setError("");
    if (!workspaceId) { setError("Your workspace is still loading. Try again in a moment."); setState("error"); return; }
    try {
      const recipe = editingRecipe
        ? await updateRecipe({ workspaceId, id: editingRecipe.id, expectedRevision: editingRecipe.revision, name: name.trim(), notes, sourceUrl, originalText: editingRecipe.originalText, sourceType: editingRecipe.sourceType, methods: editingRecipe.methods, groups: editingRecipe.groups, requiredAppliances: editingRecipe.requiredAppliances, allergenEvidence: editingRecipe.allergenEvidence, canonicalYield: editingRecipe.canonicalYield, servingUnit: editingRecipe.servingUnit, ingredients: editingRecipe.ingredients })
        : await createRecipe({ workspaceId, name: name.trim(), notes, sourceUrl, originalText: notes, sourceType: sourceUrl ? "url" : "manual" });
      setRecipes((current) => editingRecipe ? current.map((item) => item.id === recipe.id ? recipe : item) : [recipe, ...current]);
      setEditingRecipe(undefined); setName(""); setNotes(""); setSourceUrl(""); setState("saved");
    } catch (err: unknown) { setError(err instanceof Error ? err.message : "The meal was not saved. Your draft is still here."); setState("error"); }
  }

  async function downloadPDF(recipe: Recipe) {
    setExportingId(recipe.id); setError("");
    try {
      const result = await exportRecipePDF({ workspaceId, recipeId: recipe.id });
      const bytes = new Uint8Array(result.content);
      const url = URL.createObjectURL(new Blob([bytes.buffer], { type: "application/pdf" }));
      const anchor = document.createElement("a"); anchor.href = url; anchor.download = result.filename; anchor.click(); URL.revokeObjectURL(url);
    } catch (err: unknown) { setError(err instanceof Error ? err.message : "Unable to export this recipe."); }
    finally { setExportingId(""); }
  }

  return <section aria-labelledby="meals-heading" className="meals-page">
    <header className="meals-heading">
      <div><p>Your collection, ready for the week</p><h1 id="meals-heading">Your meals</h1><span>Keep the recipes and rough notes you want close.</span></div>
      <div className="flex flex-wrap gap-2"><Link to="/meals/explore" className="min-h-11 rounded border border-emerald-800 px-4 py-2 text-emerald-900">Explore meals</Link><a href="#meal-name">Add a meal <span aria-hidden="true">＋</span></a></div>
    </header>

    <section className="meals-collection" aria-labelledby="saved-meals-heading">
      <div className="meals-collection-heading">
        <div><p className="meals-eyebrow">YOUR LIBRARY</p><h2 id="saved-meals-heading">Saved meals</h2></div>
        <span className="meals-count" aria-live="polite">{state === "loading" ? "Loading…" : `${filteredRecipes.length} shown · ${recipes.length} saved`}</span>
      </div>
      <div className="meals-tools">
        <label className="meals-search" htmlFor="meal-search"><span aria-hidden="true">⌕</span><span className="sr-only">Search saved meals</span><input id="meal-search" type="search" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search meals, ingredients, or notes" /></label>
        <div className="meals-filters" role="group" aria-label="Filter saved meals">
          {[ ["all", "All meals"], ["drafts", "Drafts"], ["archived", "Archived"] ].map(([value, label]) => <button key={value} type="button" aria-pressed={statusFilter === value} onClick={() => setStatusFilter(value || "all")}>{label}</button>)}
        </div>
      </div>
      {state === "loading" && <p role="status" className="meals-empty">Loading your collection…</p>}
      {error && <p role="alert" className="meals-empty">{error}</p>}
      {state !== "loading" && state !== "error" && recipes.length === 0 && <div className="meals-empty"><span aria-hidden="true" className="meals-empty-mark">＋</span><h3>Your collection starts here</h3><p>Save a meal by name or rough notes. You can fill in the details whenever you’re ready.</p><a href="#meal-name">Capture your first meal</a></div>}
      {state !== "loading" && recipes.length > 0 && filteredRecipes.length === 0 && <p role="status" className="meals-empty">No saved meals match {query ? `“${query}”` : "this filter"}. Try another search or show all meals.</p>}
      <ul className="meals-grid">{filteredRecipes.map((recipe) => <li key={recipe.id} className="meal-card">
        <div className="meal-card-art" aria-hidden="true"><span>{recipe.name.trim().charAt(0).toLocaleUpperCase()}</span><i></i></div>
        <div className="meal-card-body">
          <div className="meal-card-meta"><span>{recipeStatus(recipe)}</span><span>Revision {recipe.revision.toString()}</span></div>
          <h3>{recipe.name}</h3>
          <p className="meal-card-summary">{recipe.notes || recipe.originalText || "No notes added yet."}</p>
          <div className="meal-card-facts" aria-label="Recipe details">
            <span>{recipe.ingredients.length ? `${recipe.ingredients.length} ingredients` : "Ingredients not added"}</span>
            {recipe.canonicalYield && <span>Yield · {recipe.canonicalYield}{recipe.servingUnit ? ` ${recipe.servingUnit}` : ""}</span>}
            {recipe.groups.slice(0, 2).map((group) => <span key={group}>{group}</span>)}
          </div>
          {recipe.sourceUrl && <p className="meal-card-source">Source saved <span aria-hidden="true">↗</span></p>}
          <div className="meal-card-actions">
            <Link to={`/recipes/${encodeURIComponent(recipe.id)}/revisions/${recipe.revision.toString()}`}>Open recipe <span aria-hidden="true">→</span></Link>
            <button type="button" aria-label={`Edit ${recipe.name}`} onClick={() => { setEditingRecipe(recipe); setName(recipe.name); setNotes(recipe.notes || ""); setSourceUrl(recipe.sourceUrl || ""); setState("ready"); setError(""); document.getElementById("meal-name")?.focus(); }}>Edit</button>
            <button type="button" aria-label={`Download PDF for ${recipe.name}`} onClick={() => void downloadPDF(recipe)} disabled={exportingId === recipe.id}>{exportingId === recipe.id ? "Exporting…" : "PDF"}</button>
          </div>
        </div>
      </li>)}</ul>
    </section>

    <details className="meals-capture" open>
      <summary>{editingRecipe ? "Edit captured meal" : "Quick capture a meal"}<span>{editingRecipe ? "Update your saved notes" : "Start with a name or rough notes"}</span></summary>
      <form onSubmit={save} aria-label="Quick capture meal">
        <label htmlFor="meal-name">Meal name</label><input id="meal-name" value={name} onChange={(event) => setName(event.target.value)} placeholder="Name this meal" />
        <label htmlFor="meal-notes">Notes or pasted text</label><textarea id="meal-notes" value={notes} onChange={(event) => setNotes(event.target.value)} placeholder="Keep the original rough notes; refine later." />
        <label htmlFor="meal-url">Source URL <span>(optional)</span></label><input id="meal-url" value={sourceUrl} onChange={(event) => setSourceUrl(event.target.value)} placeholder="https://…" />
        {state === "saved" && <p role="status">Saved to your workspace.</p>}
        <div className="meals-capture-actions"><button type="submit" disabled={state === "saving"}>{state === "saving" ? "Saving…" : editingRecipe ? "Save changes" : "Save draft"}</button>{editingRecipe && <button type="button" onClick={() => { setEditingRecipe(undefined); setName(""); setNotes(""); setSourceUrl(""); setError(""); }}>Cancel edit</button>}</div>
      </form>
    </details>
  </section>;
}
