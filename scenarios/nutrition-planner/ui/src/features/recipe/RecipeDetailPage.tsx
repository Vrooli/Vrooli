import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import type { Recipe } from "@vrooli/proto-types/nutrition-planner/v1/recipe/recipe_pb";
import { ensureWorkspace } from "../../api/workspace";
import { getRecipeRevision } from "../../api/recipes";
import { listCookingSessions, startCookingSession, type CookingSession } from "../../api/cooking";
import { scaleAmount } from "./scaling";

type View = "map" | "read";

export function RecipeDetailPage() {
  const { id = "", revision = "" } = useParams();
  const returnPath = new URLSearchParams(window.location.search).get("return") || "/meals";
  const navigate = useNavigate();
  const [recipe, setRecipe] = useState<Recipe>();
  const [selectedMethodID, setSelectedMethodID] = useState("");
  const [view, setView] = useState<View>("map");
  const [yieldAmount, setYieldAmount] = useState("");
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");
  const [error, setError] = useState("");
  const [starting, setStarting] = useState(false);
  const [sessions, setSessions] = useState<CookingSession[]>([]);

  useEffect(() => {
    let mounted = true;
    if (!/^\d+$/.test(revision) || BigInt(revision) < 1n || !id) {
      setError("This recipe revision link is invalid.");
      setState("error");
      return () => { mounted = false; };
    }
    void ensureWorkspace().then(async (workspace) => {
      const [loaded, cookingSessions] = await Promise.all([getRecipeRevision(workspace.id, id, BigInt(revision)), listCookingSessions(workspace.id)]);
      if (mounted) { setRecipe(loaded); setSessions(cookingSessions); setSelectedMethodID(loaded.methods[0]?.id ?? ""); setYieldAmount(loaded.canonicalYield); setState("ready"); }
    }).catch((err: unknown) => {
      if (mounted) { setError(err instanceof Error ? err.message : "Unable to load this recipe revision."); setState("error"); }
    });
    return () => { mounted = false; };
  }, [id, revision]);

  if (state === "loading") return <section aria-labelledby="recipe-detail-heading"><p role="status">Loading recipe revision…</p></section>;
  if (state === "error" || !recipe) return <section aria-labelledby="recipe-detail-heading"><Link to={returnPath}>Back to Explore</Link><h1 id="recipe-detail-heading" className="mt-4 text-2xl font-semibold">Recipe unavailable</h1><p role="alert" className="mt-2 text-red-800">{error}</p></section>;

  const canonicalYield = recipe.canonicalYield ?? "";
  const hasYield = canonicalYield.trim() !== "";
  const method = recipe.methods.find((candidate) => candidate.id === selectedMethodID) ?? recipe.methods[0];
  const resumable = sessions.find((session) => session.status === "active" && session.recipeId === recipe.id && session.recipeRevision === recipe.revision && session.methodId === method?.id);
  const ingredientNames = new Map(recipe.ingredients.map((ingredient) => [ingredient.id, ingredient.name || ingredient.id]));
  const outputOwners = new Map((method?.steps ?? []).flatMap((step) => step.outputs.map((output) => [output, step.id] as const)));
  const resolveInput = (id: string) => ingredientNames.get(id) ?? (outputOwners.has(id) ? `${id} from ${outputOwners.get(id)}` : `Unresolved component: ${id}`);
  const resolveStep = (id: string) => method?.steps.find((step) => step.id === id)?.instruction || id;
  async function startCooking() {
    const pinnedRecipe = recipe;
    if (!method || !pinnedRecipe) { setError("Choose a saved method before starting cooking."); return; }
    setStarting(true);
    try {
      const workspace = await ensureWorkspace();
      const session = await startCookingSession({ workspaceId: workspace.id, sessionId: crypto.randomUUID(), recipeId: pinnedRecipe.id, recipeRevision: pinnedRecipe.revision, methodId: method.id, scale: yieldAmount.trim() || canonicalYield || "1" });
      navigate(`/cook/${session.id}`);
    } catch (reason: unknown) { setError(reason instanceof Error ? reason.message : "Unable to start cooking."); }
    finally { setStarting(false); }
  }
  return <section aria-labelledby="recipe-detail-heading" className="flex flex-col gap-5">
    <Link to={returnPath} className="min-h-11 self-start rounded border bg-white px-3 py-2">Back to {returnPath.startsWith("/meals/explore") ? "Explore" : "meals"}</Link>
    <header><p className="text-sm font-medium uppercase tracking-wide text-cyan-700">Recipe detail · revision {recipe.revision.toString()}</p><h1 id="recipe-detail-heading" className="mt-1 text-3xl font-semibold text-slate-900">{recipe.name}</h1><p className="mt-2 text-slate-600">This page is pinned to revision {recipe.revision.toString()}. Later edits do not change this recipe map.</p></header>
    {hasYield && <div className="flex flex-wrap items-end gap-3 rounded-lg border bg-white p-4"><label className="grid gap-1 text-sm font-medium" htmlFor="recipe-serving-scale">Displayed servings<input id="recipe-serving-scale" className="min-h-11 rounded border px-3" inputMode="decimal" value={yieldAmount} onChange={(event) => setYieldAmount(event.target.value)} /></label><p className="text-sm text-slate-600">Canonical yield: {canonicalYield} {recipe.servingUnit || "servings"}. Scaling changes display amounts only; method times and this revision stay unchanged.</p></div>}
    {recipe.methods.length > 1 && <label className="grid max-w-md gap-1 text-sm font-medium" htmlFor="recipe-method">Preparation method<select id="recipe-method" className="min-h-11 rounded border bg-white px-3" value={method?.id ?? ""} onChange={(event) => setSelectedMethodID(event.target.value)}>{recipe.methods.map((candidate) => <option key={candidate.id} value={candidate.id}>{candidate.name || candidate.id}</option>)}</select></label>}
    {error && <p role="alert" className="rounded border border-red-200 bg-red-50 p-3 text-red-800">{error}</p>}
    {resumable && <Link className="min-h-11 self-start rounded border bg-emerald-50 px-4 py-2 text-emerald-900" to={`/cook/${resumable.id}`}>Resume active cooking session</Link>}
    <button type="button" className="min-h-11 self-start rounded bg-blue-700 px-4 text-white" disabled={!method || starting} onClick={() => void startCooking()}>{starting ? "Starting cooking…" : "Start cooking this method"}</button>
    <div role="tablist" aria-label="Recipe detail views" className="flex gap-2"><button type="button" role="tab" aria-selected={view === "map"} className={`min-h-11 rounded px-4 ${view === "map" ? "bg-blue-700 text-white" : "border bg-white"}`} onClick={() => setView("map")}>Recipe map</button><button type="button" role="tab" aria-selected={view === "read"} className={`min-h-11 rounded px-4 ${view === "read" ? "bg-blue-700 text-white" : "border bg-white"}`} onClick={() => setView("read")}>Read original</button></div>
    {view === "map" && <div className="grid gap-4 lg:grid-cols-2"><article className="rounded-lg border bg-white p-5"><h2 className="text-lg font-semibold">Ingredients</h2>{recipe.ingredients.length ? <ul aria-label="Recipe ingredients" className="mt-3 space-y-2">{recipe.ingredients.map((item) => { const amount = hasYield ? scaleAmount(item.amount, canonicalYield, yieldAmount) : item.amount; return <li key={item.id} className="rounded bg-slate-50 p-3"><strong>{item.name || item.id}</strong>: {amount || "Unknown"} {item.unit}{item.preparation && <span className="text-slate-600"> · {item.preparation}</span>}{item.discrete && amount?.includes(".") && <p className="mt-1 text-sm text-amber-800">Whole units are required; adjust servings.</p>}</li>; })}</ul> : <p className="mt-3 text-slate-600">Ingredient amounts have not been entered.</p>}</article><article className="rounded-lg border bg-white p-5"><h2 className="text-lg font-semibold">Preparation method map</h2><p className="mt-1 text-sm text-slate-600">Connections below come from this saved revision’s ingredient inputs, outputs, and declared step dependencies.</p>{method?.steps.length ? <ol aria-label="Recipe method map" className="mt-3 space-y-3">{method.steps.map((step) => <li key={step.id} className="rounded border bg-slate-50 p-3"><h3 className="font-semibold">{resolveStep(step.id)}</h3><p className="mt-1">{step.instruction || "Instruction not entered."}</p><dl className="mt-3 grid gap-2 text-sm sm:grid-cols-3"><div><dt className="font-medium text-slate-600">Uses</dt><dd>{step.inputs.map(resolveInput).join(", ") || "No inputs linked."}</dd></div><div><dt className="font-medium text-slate-600">After</dt><dd>{step.dependsOn.map(resolveStep).join(", ") || "No step dependency recorded."}</dd></div><div><dt className="font-medium text-slate-600">Makes</dt><dd>{step.outputs.join(", ") || "No output recorded."}</dd></div></dl></li>)}</ol> : <p className="mt-3 text-slate-600">No preparation method has been entered for this recipe revision.</p>}</article></div>}
    {view === "read" && <article className="whitespace-pre-wrap rounded-lg border bg-white p-5 text-slate-700">{recipe.originalText || recipe.notes || "No original recipe text has been entered."}</article>}
  </section>;
}
