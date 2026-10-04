import { useEffect, useState } from "react";
import { confirmShoppingPurchases, getShoppingPreview, setShoppingChecked, setShoppingHaveThis, type ShoppingLine } from "../../api/planning";
import { ensureWorkspace } from "../../api/workspace";
import { consumeInventoryBatchPortion, correctInventoryBatchYield, listInventoryBatches, listInventoryEvents, prepareInventoryBatch, undoInventoryBatchPortion, type InventoryBatch, type InventoryEvent } from "../../api/inventory";
import { listRecipes } from "../../api/recipes";
import { exportGroceriesCSV } from "../../api/portability";
import { getProfile } from "../../api/profile";
import { Link } from "react-router-dom";

export function GroceriesPage() {
  const [lines, setLines] = useState<ShoppingLine[]>([]);
  const [workspaceId, setWorkspaceId] = useState("");
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");
  const [error, setError] = useState("");
  const [events, setEvents] = useState<InventoryEvent[]>([]);
  const [batches, setBatches] = useState<InventoryBatch[]>([]);
  const [planRevision, setPlanRevision] = useState<bigint>(-1n);
  const [exporting, setExporting] = useState(false);
  const [recipes, setRecipes] = useState<Awaited<ReturnType<typeof listRecipes>>>([]);
  const [profile, setProfile] = useState<Awaited<ReturnType<typeof getProfile>>>();
  const [recipeId, setRecipeId] = useState("");
  const [batchAction, setBatchAction] = useState<"idle" | "working">("idle");
  const [yieldEdits, setYieldEdits] = useState<Record<string, string>>({});
  const [portionEdits, setPortionEdits] = useState<Record<string, string>>({});
  const [purchaseEdits, setPurchaseEdits] = useState<Record<string, { amount: string; unit: string; price: string; omitted: boolean }>>({});
  const [purchaseReviewId, setPurchaseReviewId] = useState("");
  const [tripConfirmed, setTripConfirmed] = useState(false);

  useEffect(() => {
    let mounted = true;
    void ensureWorkspace().then(async (workspace) => {
      if (mounted) setWorkspaceId(workspace.id);
      const [shopping, inventory, recipeList, savedBatches, savedProfile] = await Promise.all([
        getShoppingPreview({ workspaceId: workspace.id, expectedRevision: -1n }),
        listInventoryEvents(workspace.id).catch(() => []), listRecipes(workspace.id).catch(() => []),
        listInventoryBatches(workspace.id).catch(() => []), getProfile(workspace.id).catch(() => undefined),
      ]);
      if (mounted) { setLines(shopping.lines); setTripConfirmed(shopping.lines.some((line) => Boolean(line.actualQuantity || line.actualPrice || line.purchaseOmitted))); setPlanRevision(shopping.revision); setEvents(inventory); setRecipes(recipeList); setBatches(savedBatches); setProfile(savedProfile); setState("ready"); }
    }).catch((err: unknown) => { if (mounted) { setError(err instanceof Error ? err.message : "Unable to load the Kitchen."); setState("error"); } });
    return () => { mounted = false; };
  }, []);

  async function toggle(line: ShoppingLine) {
    const checked = await setShoppingChecked({ workspaceId, lineKey: line.key, checked: !line.checked }).catch(() => line.checked);
    setLines((current) => current.map((item) => item.key === line.key ? { ...item, checked } : item));
  }
  async function toggleHaveThis(line: ShoppingLine) {
    const haveThis = await setShoppingHaveThis({ workspaceId, lineKey: line.key, haveThis: !line.haveThis }).catch(() => line.haveThis);
    setLines((current) => current.map((item) => item.key === line.key ? { ...item, haveThis } : item));
  }
  function purchaseEdit(line: ShoppingLine) { return purchaseEdits[line.key] ?? { amount: line.actualQuantity || "", unit: line.actualUnit || line.need.match(/\s([^\s]+)$/)?.[1] || "", price: line.actualPrice || "", omitted: line.purchaseOmitted || false }; }
  async function confirmTrip() {
    const reviewId = purchaseReviewId || `shopping-review-${crypto.randomUUID()}`;
    setPurchaseReviewId(reviewId); setBatchAction("working");
    try {
      await confirmShoppingPurchases({ workspaceId, reviewId, lines: lines.map((line) => { const actual = purchaseEdit(line); return { lineKey: line.key, itemId: line.key.replace(/^ingredient:/, ""), amount: actual.amount, unit: actual.unit, price: actual.price, omitted: actual.omitted }; }) });
      const [preview, nextEvents] = await Promise.all([getShoppingPreview({ workspaceId, expectedRevision: planRevision }), listInventoryEvents(workspaceId)]);
      setLines(preview.lines); setEvents(nextEvents); setPurchaseReviewId(""); setPurchaseEdits({}); setError("");
      setTripConfirmed(true);
    } catch (err: unknown) { setError(err instanceof Error ? err.message : "Unable to confirm actual purchases."); }
    finally { setBatchAction("idle"); }
  }
  async function prepareBatch() {
    const recipe = recipes.find((item) => item.id === recipeId);
    if (!recipe || !recipe.canonicalYield || !recipe.servingUnit) { setError("Choose a recipe with a canonical yield and serving unit before preparing a batch."); return; }
    setBatchAction("working");
    try {
      const prepared = await prepareInventoryBatch({ workspaceId, eventId: `prepare-${crypto.randomUUID()}`, batchId: `batch-${crypto.randomUUID()}`, recipeId: recipe.id, recipeRevision: recipe.revision, yieldAmount: recipe.canonicalYield, unit: recipe.servingUnit, requirements: recipe.ingredients.map((ingredient) => ({ itemId: ingredient.id || ingredient.name, amount: ingredient.amount, unit: ingredient.unit })) });
      setBatches((current) => [prepared, ...current.filter((item) => item.id !== prepared.id)]); setError("");
    } catch (err: unknown) { setError(err instanceof Error ? err.message : "Unable to prepare the batch."); }
    finally { setBatchAction("idle"); }
  }
  async function changeSavedBatch(item: InventoryBatch, action: "consume" | "undo") {
    const amount = portionEdits[item.id] ?? "1"; if (!amount) return; setBatchAction("working");
    try {
      const input = { workspaceId, eventId: `${action}-${crypto.randomUUID()}`, batchId: item.id, amount, unit: item.unit, recipeId: item.recipeId };
      const next = action === "consume" ? await consumeInventoryBatchPortion(input) : await undoInventoryBatchPortion(input);
      setBatches((current) => current.map((batch) => batch.id === next.id ? next : batch)); setEvents(await listInventoryEvents(workspaceId).catch(() => events)); setError("");
    } catch (err: unknown) { setError(err instanceof Error ? err.message : `Unable to ${action} the batch portion.`); }
    finally { setBatchAction("idle"); }
  }
  async function correctYield(item: InventoryBatch) {
    const yieldAmount = yieldEdits[item.id]?.trim(); if (!yieldAmount) return;
    setBatchAction("working");
    try {
      const corrected = await correctInventoryBatchYield({ workspaceId, eventId: `yield-correction-${crypto.randomUUID()}`, batchId: item.id, yieldAmount, unit: item.unit });
      setBatches((current) => current.map((batch) => batch.id === corrected.id ? corrected : batch));
      setError("");
    } catch (err: unknown) { setError(err instanceof Error ? err.message : "Unable to correct the measured yield."); }
    finally { setBatchAction("idle"); }
  }
  async function downloadCSV() {
    setExporting(true);
    try { const result = await exportGroceriesCSV({ workspaceId, expectedRevision: planRevision }); const blob = new Blob([result.content], { type: "text/csv;charset=utf-8" }); const url = URL.createObjectURL(blob); const anchor = document.createElement("a"); anchor.href = url; anchor.download = result.filename; anchor.click(); URL.revokeObjectURL(url); }
    catch (err: unknown) { setError(err instanceof Error ? err.message : "Unable to export groceries."); }
    finally { setExporting(false); }
  }

  return <section aria-labelledby="groceries-heading" className="flex flex-col gap-6">
    <header><p className="text-sm font-medium uppercase tracking-wide text-cyan-700">Kitchen</p><h1 id="groceries-heading" className="text-3xl font-semibold text-slate-900">Kitchen and shopping</h1><p className="mt-2 text-slate-600">Shopping is derived from the selected plan. Prepared stock appears here only after a measured batch yield is confirmed.</p></header>
    {state === "loading" && <p role="status" className="rounded border bg-white p-5 text-slate-600">Loading Kitchen…</p>}
    {(state === "error" || error) && <p role="alert" className="rounded border border-red-200 bg-red-50 p-5 text-red-800">{error || "Unable to load the Kitchen."}</p>}
    {state === "ready" && <article className="rounded-lg border bg-white p-5"><h2 className="font-semibold">Prepared Kitchen stock</h2>{batches.length ? <ul aria-label="Prepared Kitchen batches" className="mt-3 divide-y">{batches.map((item) => <li key={item.id} className="py-3"><p className="font-medium">{recipes.find((recipe) => recipe.id === item.recipeId)?.name || item.recipeId} · revision {item.recipeRevision.toString()}</p><p className="text-sm text-slate-600">Measured yield {item.yieldAmount} {item.unit} · {item.availableAmount} {item.unit} available</p><div className="mt-2 flex flex-wrap items-end gap-2"><label className="grid gap-1 text-sm">Correct measured yield<input aria-label={`Correct yield for ${item.id}`} className="min-h-11 rounded border px-3" inputMode="decimal" value={yieldEdits[item.id] ?? item.yieldAmount} onChange={(event) => setYieldEdits((current) => ({ ...current, [item.id]: event.target.value }))} /></label><button type="button" className="min-h-11 rounded border px-3" disabled={batchAction === "working" || (yieldEdits[item.id] ?? item.yieldAmount) === item.yieldAmount} onClick={() => void correctYield(item)}>{batchAction === "working" ? "Saving…" : "Save yield correction"}</button><label className="grid gap-1 text-sm">Portion amount<input aria-label={`Portion amount for ${item.id}`} className="min-h-11 rounded border px-3" inputMode="decimal" value={portionEdits[item.id] ?? "1"} onChange={(event) => setPortionEdits((current) => ({ ...current, [item.id]: event.target.value }))} /></label><button type="button" className="min-h-11 rounded border px-3" disabled={batchAction === "working"} onClick={() => void changeSavedBatch(item,"consume")}>Consume portion from {item.id}</button><button type="button" className="min-h-11 rounded border px-3" disabled={batchAction === "working"} onClick={() => void changeSavedBatch(item,"undo")}>Undo portion for {item.id}</button></div><p className="mt-1 text-xs text-slate-500">Corrections preserve portions already consumed. Portion actions change prepared stock; recorded nutrition intake stays separate.</p></li>)}</ul> : <p className="mt-2 text-slate-600">No prepared batches yet. Finishing a cooking session does not add stock by itself.</p>}</article>}
    {state === "ready" && <article className="rounded-lg border bg-white p-5"><h2 className="font-semibold">Cooking equipment</h2><p className="mt-1 text-sm text-slate-600">Only equipment you selected is shown as available. A recipe that needs another method remains unchanged.</p>{profile?.appliances.length ? <ul aria-label="Selected cooking equipment" className="mt-3 flex flex-wrap gap-2">{profile.appliances.map((appliance) => <li key={appliance} className="rounded-full bg-slate-100 px-3 py-1 text-sm">{appliance.split("_").join(" ")}</li>)}</ul> : <p className="mt-2 text-slate-600">No equipment capabilities are recorded. <Link className="underline" to="/setup">Choose equipment in setup</Link>.</p>}</article>}
    {state === "ready" && <button type="button" onClick={() => void downloadCSV()} disabled={exporting}>{exporting ? "Exporting…" : "Download grocery CSV"}</button>}
    {state === "ready" && <article className="rounded-lg border bg-white p-5"><h2 className="font-semibold">Shopping preview</h2>{lines.length ? <ul aria-label="Shopping lines" className="mt-3 divide-y">{lines.map((line) => { const actual = purchaseEdit(line); return <li key={line.key} className="grid gap-3 py-4 sm:grid-cols-[auto_auto_1fr]"><input type="checkbox" aria-label={`Check ${line.label}`} checked={line.checked} onChange={() => void toggle(line)} /><label className="flex items-center gap-2 text-sm"><input type="checkbox" aria-label={`Have this ${line.label}`} checked={line.haveThis} onChange={() => void toggleHaveThis(line)} />Have this</label><div className="min-w-0"><p className={line.checked ? "text-slate-500 line-through" : "font-medium text-slate-900"}>{line.label}</p><p className="mt-1 text-sm text-slate-600">Need: {line.need} · Stock: {line.stock} · Remaining need: {line.missing} · Packages: {line.packageCount} · Package price: {line.price}</p><p className="mt-1 text-sm text-slate-600">Portion cost: {line.portionCost ?? "unknown"} · Checkout total: {line.checkoutTotal ?? "unknown"} · Actual spend: {line.actualSpend ?? "unknown"}</p>{(line.actualQuantity || line.purchaseOmitted) && <p className="mt-1 text-sm text-slate-700">Last trip: {line.purchaseOmitted ? "not purchased" : `${line.actualQuantity} ${line.actualUnit}`} · Actual price: {line.actualPrice || "unknown"}</p>}<p className="mt-1 text-xs text-slate-500">Picked up is a checklist fact. Have this records no amount and checking does not change stock or consumption.</p><div className="mt-2 flex flex-wrap items-end gap-2"><label className="grid gap-1 text-sm">Actual quantity<input aria-label={`Actual quantity ${line.label}`} className="min-h-10 rounded border px-2" inputMode="decimal" value={actual.amount} disabled={actual.omitted || tripConfirmed} onChange={(event) => setPurchaseEdits((current) => ({ ...current, [line.key]: { ...purchaseEdit(line), amount: event.target.value } }))} /></label><label className="grid gap-1 text-sm">Unit<input aria-label={`Actual unit ${line.label}`} className="min-h-10 rounded border px-2" value={actual.unit} disabled={actual.omitted || tripConfirmed} onChange={(event) => setPurchaseEdits((current) => ({ ...current, [line.key]: { ...purchaseEdit(line), unit: event.target.value } }))} /></label><label className="grid gap-1 text-sm">Actual price<input aria-label={`Actual price ${line.label}`} className="min-h-10 rounded border px-2" value={actual.price} disabled={actual.omitted || tripConfirmed} onChange={(event) => setPurchaseEdits((current) => ({ ...current, [line.key]: { ...purchaseEdit(line), price: event.target.value } }))} /></label><label className="flex min-h-10 items-center gap-2 text-sm"><input type="checkbox" aria-label={`Omit ${line.label}`} checked={actual.omitted} disabled={tripConfirmed} onChange={(event) => setPurchaseEdits((current) => ({ ...current, [line.key]: { ...purchaseEdit(line), omitted: event.target.checked } }))} />Not purchased</label></div></div></li>; })}</ul> : <p className="mt-2 text-slate-600">No ingredient evidence yet. Add ingredient inputs to a planned recipe to derive shopping lines.</p>}{lines.length > 0 && <div className="mt-4 border-t pt-4"><p className="text-sm text-slate-600">Review actual quantities, prices and omissions. Confirming records purchase stock once; it does not record eating.</p>{tripConfirmed ? <div className="mt-3 flex items-center gap-3"><p role="status" className="text-sm text-slate-700">This trip is confirmed. Start a new trip before recording another purchase.</p><button type="button" className="min-h-11 rounded border px-4" onClick={() => { setTripConfirmed(false); setPurchaseReviewId(""); setPurchaseEdits({}); }}>Start a new trip</button></div> : <button type="button" className="mt-3 min-h-11 rounded bg-slate-900 px-4 text-white" disabled={batchAction === "working"} onClick={() => void confirmTrip()}>{batchAction === "working" ? "Saving…" : "Confirm purchases"}</button>}</div>}</article>}

    {state === "ready" && <article className="rounded-lg border bg-white p-5"><h2 className="font-semibold">Prepare a batch</h2><p className="mt-1 text-sm text-slate-600">Preparation records declared ingredient use once and creates stock only from a known recipe yield.</p><div className="mt-4 flex flex-wrap gap-3"><label>Recipe <select aria-label="Batch recipe" value={recipeId} onChange={(event) => setRecipeId(event.target.value)}><option value="">Choose a recipe</option>{recipes.map((recipe) => <option key={recipe.id} value={recipe.id}>{recipe.name}</option>)}</select></label><button type="button" onClick={() => void prepareBatch()} disabled={!recipeId || batchAction === "working"}>{batchAction === "working" ? "Working…" : "Prepare batch"}</button></div></article>}
  </section>;
}
