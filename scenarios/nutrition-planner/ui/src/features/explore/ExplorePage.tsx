import { useEffect, useMemo, useState } from "react";
import { Link, useLocation, useSearchParams } from "react-router-dom";
import { applyPlan, exploreRecipes, getPlan, previewSwap, type ExploreRecipe, type PlanDraft } from "../../api/planning";
import { ensureWorkspace } from "../../api/workspace";
import { useTheme } from "../../theme/ThemeProvider";
import { MealMediaFrame } from "../media/mealMedia";

type LoadState = "loading" | "ready" | "error" | "offline";
type ExploreSwapReview = { recipe: ExploreRecipe; revision: bigint; preview: Awaited<ReturnType<typeof previewSwap>>["preview"] };

function refreshedSlotHref(date: string, slot: string, revision: bigint, recipeId: string) {
  const params = new URLSearchParams({ date, slot, basePlanRevision: revision.toString(), expectedRecipeId: recipeId });
  if (recipeId) params.set("replace", "occurrence");
  return `/meals/explore?${params.toString()}`;
}

export function ExplorePage() {
  const { resolved } = useTheme();
  const [searchParams] = useSearchParams();
  const location = useLocation();
  const [state, setState] = useState<LoadState>("loading");
  const [error, setError] = useState("");
  const [candidates, setCandidates] = useState<ExploreRecipe[]>([]);
  const [blockingReasons, setBlockingReasons] = useState<ExploreRecipe["fitReasons"]>([]);
  const [savedRecipeCount, setSavedRecipeCount] = useState(0);
  const [workspaceId, setWorkspaceId] = useState("");
  const [profileConfigured, setProfileConfigured] = useState(false);
  const [profileRevision, setProfileRevision] = useState(0n);
  const [planRevision, setPlanRevision] = useState(0n);
  const [query, setQuery] = useState("");
  const [actionMessage, setActionMessage] = useState("");
  const [actionError, setActionError] = useState("");
  const [savingRecipe, setSavingRecipe] = useState("");
  const [staleContext, setStaleContext] = useState<{ currentRecipeId: string; currentRecipeName: string; href: string }>();
  const [swapReview, setSwapReview] = useState<ExploreSwapReview>();
  const date = searchParams.get("date") ?? "";
  const slot = searchParams.get("slot") ?? "";
  const replacing = searchParams.has("occurrenceId") || Boolean(searchParams.get("replace"));
  const hasExpectedTarget = searchParams.has("expectedRecipeId");
  const expectedRecipeId = searchParams.get("expectedRecipeId") ?? "";
  const returnTo = `/meals/explore${location.search}`;
  const dayLabel = date ? new Date(`${date}T12:00:00`).toLocaleDateString(undefined, { weekday: "long" }) : "";
  const slotLabel = date && slot ? `${dayLabel} ${slot}` : slot;

  useEffect(() => {
    let mounted = true;
    setState("loading");
    void ensureWorkspace().then(async (workspace) => {
      const result = await exploreRecipes(workspace.id);
      const current = date && slot && hasExpectedTarget ? await getPlan({ workspaceId: workspace.id, fromDate: date, toDate: date }) : undefined;
      return { workspace, result, current };
    }).then(({ workspace, result, current }) => {
      if (mounted) {
        setWorkspaceId(workspace.id);
        setCandidates(result.candidates);
        setBlockingReasons(result.blockingReasons ?? []);
        setSavedRecipeCount(result.savedRecipeCount ?? result.candidates.length);
        setProfileConfigured(result.profileConfigured);
        setProfileRevision(result.profileRevision);
        const revision = current?.draft.currentRevision ?? result.planRevision;
        setPlanRevision(revision);
        const latestSlot = current?.draft.occurrences.find((item) => item.date === date && (item.slotName || "dinner") === slot);
        const latestRecipeId = latestSlot?.recipeId ?? "";
        if (current && latestRecipeId !== expectedRecipeId) {
          setStaleContext({ currentRecipeId: latestRecipeId, currentRecipeName: latestSlot?.recipeName ?? "no meal", href: refreshedSlotHref(date, slot, revision, latestRecipeId) });
        } else {
          setStaleContext(undefined);
        }
        setState("ready");
      }
    }).catch((reason: unknown) => {
      if (mounted) {
        const offline = typeof navigator !== "undefined" && navigator.onLine === false;
        setError(offline ? "Explore is offline. Reconnect to check your saved recipes and planning context." : reason instanceof Error ? reason.message : "Explore could not load saved meals.");
        setState(offline ? "offline" : "error");
      }
    });
    return () => { mounted = false; };
  }, [date, expectedRecipeId, hasExpectedTarget, slot]);

  const filtered = useMemo(() => {
    const needle = query.trim().toLocaleLowerCase();
    if (!needle) return candidates;
    return candidates.filter((recipe) => [recipe.name, recipe.summary, ...recipe.fitReasons.map((reason) => reason.message)].some((value) => value.toLocaleLowerCase().includes(needle)));
  }, [candidates, query]);

  const contextLabel = date && slot ? `Planning ${new Date(`${date}T12:00:00`).toLocaleDateString(undefined, { weekday: "long" })} ${slot}` : "Discover meals from your saved collection";

  async function addToOpenSlot(recipe: ExploreRecipe) {
    if (!date || !slot || !workspaceId) return;
    setSavingRecipe(recipe.recipeId); setActionError(""); setActionMessage("");
    try {
      // ApplyPlan validates the full saved draft, including locked occurrences
      // on other dates. Read the complete supported date domain so this focused
      // slot update retains those occurrences and their snapshots.
      const current = await getPlan({ workspaceId, fromDate: "0001-01-01", toDate: "9999-12-31" });
      const target = current.draft.occurrences.find((item) => item.date === date && (item.slotName || "dinner") === slot);
      if (target?.recipeId) {
        if (target.recipeId === recipe.recipeId && target.recipeRevision === Number(recipe.recipeRevision)) {
          setActionMessage(`${recipe.name} is already saved in ${slot} on ${date}.`);
          return;
        }
        setActionError("This slot now has a planned meal. Refresh the plan context and preview a replacement before changing it.");
        setPlanRevision(current.draft.currentRevision);
        setStaleContext({ currentRecipeId: target.recipeId, currentRecipeName: target.recipeName || "a saved meal", href: refreshedSlotHref(date, slot, current.draft.currentRevision, target.recipeId) });
        return;
      }
      const nextDraft: PlanDraft = {
        ...current.draft,
        occurrences: [
          ...current.draft.occurrences.filter((item) => !(item.date === date && (item.slotName || "dinner") === slot && !item.recipeId)),
          { date, slotName: slot, mode: "fixed", quantity: "1", recipeId: recipe.recipeId, recipeRevision: Number(recipe.recipeRevision), recipeName: recipe.name, reason: recipe.fitReasons.map((reason) => reason.message).join(" "), locked: false },
        ],
        inputReferences: [...new Set([...current.draft.inputReferences, `recipe:${recipe.recipeId}:${recipe.recipeRevision.toString()}`, `profile:${profileRevision.toString()}`])],
      };
      await applyPlan({ workspaceId, expectedRevision: current.draft.currentRevision, draft: nextDraft });
      setActionMessage(`${recipe.name} was added to ${slot} on ${date}.`);
      setPlanRevision(current.draft.currentRevision + 1n);
    } catch (reason: unknown) {
      setActionError(reason instanceof Error ? `${reason.message} Refresh Explore before trying again.` : "The plan changed before this meal could be added. Refresh Explore and review the slot again.");
    } finally { setSavingRecipe(""); }
  }

  async function prepareReplacement(recipe: ExploreRecipe) {
    if (!date || !slot || !workspaceId) return;
    setSavingRecipe(recipe.recipeId); setActionError(""); setActionMessage("");
    try {
      const result = await previewSwap({ workspaceId, expectedRevision: planRevision, date, slotName: slot, replacementRecipeId: recipe.recipeId });
      setSwapReview({ recipe, revision: result.revision, preview: result.preview });
    } catch (reason: unknown) {
      setActionError(reason instanceof Error ? `${reason.message} Refresh Explore and review the current plan.` : "The plan changed before a replacement could be previewed.");
    } finally { setSavingRecipe(""); }
  }

  async function confirmReplacement() {
    if (!swapReview || !workspaceId) return;
    setSavingRecipe(swapReview.recipe.recipeId); setActionError("");
    try {
      await applyPlan({ workspaceId, expectedRevision: swapReview.revision, draft: swapReview.preview.draft });
      setActionMessage(`${swapReview.recipe.name} replaced the meal in ${slot} on ${date}.`);
      setPlanRevision(swapReview.revision + 1n);
      setSwapReview(undefined);
    } catch (reason: unknown) {
      setActionError(reason instanceof Error ? `${reason.message} Nothing was replaced. Refresh the plan and preview again.` : "The plan changed before confirmation. Nothing was replaced.");
      setSwapReview(undefined);
    } finally { setSavingRecipe(""); }
  }

  return <section aria-labelledby="explore-heading" className="explore-page">
    <header className="explore-heading">
      <div><p>Your collection and ideas for your week</p><h1 id="explore-heading">Meals worth making</h1><span>Explore checks saved meals against your current food rules and kitchen. Nutrition, price, and stock stay unknown unless their evidence is available.</span></div>
      <Link to="/meals">Your meals</Link>
    </header>
    <nav className="explore-tabs" aria-label="Meals sections"><Link to="/meals">Your meals</Link><span aria-current="page">Explore</span></nav>
    <section aria-label="Planning context" className="explore-context">
      <p><span aria-hidden="true">ⓘ</span><strong>{contextLabel}</strong></p>
      {date && slot && <span>{replacing ? "Choose a meal to review as a replacement. Your plan will not change until you preview and confirm." : "Choose a saved meal for this open slot. The plan is checked again before anything is added."}</span>}
      {(date || slot) && <Link to="/meals/explore">Browse without this slot context</Link>}
    </section>
    {state === "loading" && <div role="status" className="explore-loading"><span>Checking saved meals against your setup…</span><div aria-hidden="true"><i /><i /><i /></div></div>}
    {(state === "error" || state === "offline") && <div className={`explore-state ${state === "offline" ? "is-warning" : "is-error"}`}><p role="alert">{error}</p><button type="button" onClick={() => window.location.reload()}>Try again</button></div>}
    {state === "ready" && <>
      {actionError && <p role="alert" className="explore-state is-error">{actionError} {staleContext && <Link to={staleContext.href}>Review the latest slot</Link>}</p>}
      {staleContext && <p role="alert" className="explore-state is-warning">This slot changed since you opened Explore. It now contains {staleContext.currentRecipeName}; your previous choice is not applied. <Link to={staleContext.href}>Review the latest slot</Link></p>}
      {actionMessage && <p role="status" className="explore-state is-success">{actionMessage} <Link to={`/week?date=${encodeURIComponent(date)}`}>Review your week</Link></p>}
      {swapReview && <section role="dialog" aria-modal="true" aria-labelledby="replacement-review-heading" className="explore-review"><h2 id="replacement-review-heading">Review this replacement</h2><p>{swapReview.recipe.name} will replace the planned meal in {slot} on {date}. The plan has not changed.</p><ul>{swapReview.preview.changes.map((change) => <li key={`${change.date}:${change.slotName}`}>{change.date} · {change.slotName}: {change.beforeName} → {change.afterName}</li>)}</ul>
        {swapReview.preview.shoppingChanges.length > 0 && <section aria-label="Shopping impact"><h3>Shopping changes</h3><ul>{swapReview.preview.shoppingChanges.map((change) => <li key={change.key}>{change.after?.label ?? change.before?.label ?? change.key}: need {change.before?.need ?? "unknown"} → {change.after?.need ?? "unknown"}; stock and cost remain {change.after?.stock ?? change.before?.stock ?? "unknown"} / {change.after?.price ?? change.before?.price ?? "unknown"}.</li>)}</ul></section>}
        {swapReview.preview.relatedOccurrences.length > 0 && <section aria-label="Related planned meals"><h3>Related occurrences to review</h3><ul>{swapReview.preview.relatedOccurrences.map((item) => <li key={`${item.date}:${item.slotName}`}>{item.date} · {item.slotName}: {item.recipeName}, revision {item.recipeRevision}{item.locked ? " · locked" : ""}. This occurrence is not changed by the preview.</li>)}</ul></section>}
        {swapReview.preview.preparedBatchImpacts.length > 0 && <section aria-label="Prepared meal impact"><h3>Prepared portions</h3><ul>{swapReview.preview.preparedBatchImpacts.map((item) => <li key={item.batchId}>{item.recipeName}: {item.available} {item.unit} prepared; review before replacing.</li>)}</ul></section>}
        <div><button type="button" disabled={Boolean(savingRecipe)} onClick={() => void confirmReplacement()}>{savingRecipe ? "Applying…" : "Confirm replacement"}</button><button type="button" disabled={Boolean(savingRecipe)} onClick={() => setSwapReview(undefined)}>Keep current meal</button></div></section>}
      {!profileConfigured && <p className="explore-state is-warning">Setup is not applied yet. These saved meals are checked only against the evidence on each recipe; they do not reflect food rules you have not entered.</p>}
      <div className="explore-tools"><label htmlFor="explore-search">Search eligible saved meals<input id="explore-search" type="search" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search: meal or ingredient" /></label><span>Results reflect saved recipe evidence</span></div>
      {candidates.length === 0 ? <div className="explore-state"><h2>{savedRecipeCount === 0 ? "No saved meals to explore yet" : "No saved meals fit yet"}</h2><p>{savedRecipeCount === 0 ? "Explore only uses recipes saved in this workspace. Add a meal to your collection before planning." : "These saved meals did not pass every required rule. Review the blocking evidence before planning."}</p>{blockingReasons.length > 0 && <ul aria-label="Why saved meals were excluded">{blockingReasons.map((reason) => <li key={`${reason.code}:${reason.rule}:${reason.reference}`}>{reason.message}{reason.rule ? ` · ${reason.rule}` : ""}</li>)}</ul>}<Link to="/meals">Review saved meals</Link></div>
        : filtered.length === 0 ? <div role="status" className="explore-state"><p>No eligible saved meals match “{query}”.</p><button type="button" onClick={() => setQuery("")}>Clear search</button></div>
          : <ul className="explore-grid">{filtered.map((recipe) => <li key={`${recipe.recipeId}:${recipe.recipeRevision}`} className="explore-card">
            <MealMediaFrame className="explore-card-art" recipeId={recipe.recipeId} recipeRevision={Number(recipe.recipeRevision)} appearance={resolved === "dark" ? "evening" : "light"} label={recipe.name} fallbackText="Recipe photo unavailable" detail={`Saved recipe · revision ${recipe.recipeRevision.toString()}`} />
            <div className="explore-card-copy"><p className="explore-card-eyebrow">Saved to Your meals · private revision {recipe.recipeRevision.toString()}</p><h2>{recipe.name}</h2><p className="explore-card-summary">{recipe.summary || "No recipe notes saved."}</p><ul aria-label={`Why ${recipe.name} fits`}>{recipe.fitReasons.map((reason) => <li key={`${reason.code}:${reason.rule}:${reason.reference}`}>{reason.message}</li>)}</ul></div>
            {date && slot && !replacing ? <button type="button" className="explore-primary-action" disabled={Boolean(savingRecipe) || Boolean(swapReview) || Boolean(staleContext)} onClick={() => void addToOpenSlot(recipe)}>{savingRecipe === recipe.recipeId ? "Adding…" : `＋ Add to ${slotLabel}`}</button>
              : date && slot ? <button type="button" className="explore-primary-action" disabled={Boolean(savingRecipe) || Boolean(swapReview) || Boolean(staleContext)} onClick={() => void prepareReplacement(recipe)}>{savingRecipe === recipe.recipeId ? "Preparing preview…" : `Preview replacing ${slotLabel}`}</button>
                : <Link className="explore-primary-action" to={`/recipes/${encodeURIComponent(recipe.recipeId)}/revisions/${recipe.recipeRevision.toString()}?return=${encodeURIComponent(returnTo)}`}>Open recipe</Link>}
            {date && slot && <div className="explore-card-links"><Link to={`/recipes/${encodeURIComponent(recipe.recipeId)}/revisions/${recipe.recipeRevision.toString()}?return=${encodeURIComponent(returnTo)}`}>Open recipe</Link></div>}
          </li>)}</ul>}
    </>}
  </section>;
}
