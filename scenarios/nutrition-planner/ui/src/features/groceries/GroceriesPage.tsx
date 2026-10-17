import { useEffect, useState } from "react";
import { confirmShoppingPurchases, getShoppingPreview, setShoppingChecked, setShoppingHaveThis, type ShoppingLine } from "../../api/planning";
import { ensureWorkspace } from "../../api/workspace";
import { consumeInventoryBatchPortion, correctInventoryBatchYield, listInventoryBatches, listInventoryEvents, prepareInventoryBatch, undoInventoryBatchPortion, type InventoryBatch, type InventoryEvent } from "../../api/inventory";
import { listRecipes } from "../../api/recipes";
import { exportGroceriesCSV } from "../../api/portability";
import { getProfile } from "../../api/profile";
import { Link } from "react-router-dom";
import { Code, ConnectError } from "@connectrpc/connect";

type PurchaseLineRequest = Readonly<{ lineKey: string; itemId: string; amount: string; unit: string; price: string; omitted: boolean }>;
type PendingPurchase = Readonly<{ workspaceId: string; reviewId: string; lines: readonly PurchaseLineRequest[] }>;

function pendingPurchaseKey(workspaceId: string) { return `nutrition-planner.groceries.pending-purchase:${workspaceId}`; }

function immutablePurchase(workspaceId: string, reviewId: string, lines: readonly PurchaseLineRequest[]): PendingPurchase {
  return Object.freeze({ workspaceId, reviewId, lines: Object.freeze(lines.map((line) => Object.freeze({ ...line }))) });
}

function readPendingPurchase(workspaceId: string): PendingPurchase | undefined {
  try {
    const raw = sessionStorage.getItem(pendingPurchaseKey(workspaceId));
    if (!raw) return undefined;
    const value = JSON.parse(raw) as Partial<PendingPurchase>;
    if (value.workspaceId !== workspaceId || typeof value.reviewId !== "string" || !value.reviewId || !Array.isArray(value.lines) || value.lines.length === 0) return undefined;
    const validLines = value.lines.every((line) => line && typeof line.lineKey === "string" && typeof line.itemId === "string" && typeof line.amount === "string" && typeof line.unit === "string" && typeof line.price === "string" && typeof line.omitted === "boolean");
    return validLines ? immutablePurchase(workspaceId, value.reviewId, value.lines as PurchaseLineRequest[]) : undefined;
  } catch { return undefined; }
}

function isCorrectablePurchaseValidation(err: unknown): boolean {
  if (!(err instanceof ConnectError) || err.code !== Code.InvalidArgument) return false;
  return /^purchase amount for ".+":/.test(err.rawMessage)
    || /^purchase row ".+" requires a positive actual quantity$/.test(err.rawMessage)
    || err.rawMessage === "purchase review rows require unique keys, item ids, and units";
}

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
  const [pendingPurchase, setPendingPurchase] = useState<PendingPurchase>();
  const [pendingPurchaseStored, setPendingPurchaseStored] = useState(false);
  const [purchasePreviewOpen, setPurchasePreviewOpen] = useState(false);
  const [tripConfirmed, setTripConfirmed] = useState(false);
  const [shoppingMode, setShoppingMode] = useState<"review" | "shop">(() => {
    try { return localStorage.getItem("nutrition-planner.groceries.mode") === "shop" ? "shop" : "review"; }
    catch { return "review"; }
  });
  const [searchQuery, setSearchQuery] = useState("");
  const [pendingPickupFocus, setPendingPickupFocus] = useState<{ key: string; action: "check" | "undo" }>();

  function changeShoppingMode(mode: "review" | "shop") {
    setShoppingMode(mode);
    try { localStorage.setItem("nutrition-planner.groceries.mode", mode); } catch { /* Keep the current view usable without storage. */ }
  }

  useEffect(() => {
    let mounted = true;
    void ensureWorkspace().then(async (workspace) => {
      if (mounted) setWorkspaceId(workspace.id);
      const savedPendingPurchase = readPendingPurchase(workspace.id);
      if (mounted && savedPendingPurchase) { setPendingPurchase(savedPendingPurchase); setPendingPurchaseStored(true); setPurchasePreviewOpen(true); setShoppingMode("review"); try { localStorage.setItem("nutrition-planner.groceries.mode", "review"); } catch { /* Recovery remains available without mode storage. */ } }
      const shoppingLoad = getShoppingPreview({ workspaceId: workspace.id, expectedRevision: -1n }).then((data) => ({ data })).catch((error: unknown) => ({ error }));
      const [shopping, inventory, recipeList, savedBatches, savedProfile] = await Promise.all([
        shoppingLoad,
        listInventoryEvents(workspace.id).catch(() => []), listRecipes(workspace.id).catch(() => []),
        listInventoryBatches(workspace.id).catch(() => []), getProfile(workspace.id).catch(() => undefined),
      ]);
      if ("error" in shopping && !savedPendingPurchase) throw shopping.error;
      if (mounted) {
        const recoveredShopping = "data" in shopping ? shopping.data : undefined;
        setLines(recoveredShopping?.lines ?? []);
        setTripConfirmed(!savedPendingPurchase && Boolean(recoveredShopping?.lines.some((line) => Boolean(line.actualQuantity || line.actualPrice || line.purchaseOmitted))));
        setPlanRevision(recoveredShopping?.revision ?? -1n);
        if (!recoveredShopping && "error" in shopping) setError(shopping.error instanceof Error ? shopping.error.message : "Unable to refresh the pending purchase preview.");
        setEvents(inventory); setRecipes(recipeList); setBatches(savedBatches); setProfile(savedProfile); setState("ready");
      }
    }).catch((err: unknown) => { if (mounted) { setError(err instanceof Error ? err.message : "Unable to load groceries."); setState("error"); } });
    return () => { mounted = false; };
  }, []);

  useEffect(() => {
    if (!pendingPickupFocus) return;
    const target = [...document.querySelectorAll<HTMLElement>("[data-groceries-focus]")].find((element) => element.dataset.lineKey === pendingPickupFocus.key && element.dataset.groceriesFocus === pendingPickupFocus.action);
    if (target) {
      target.focus();
      setPendingPickupFocus(undefined);
    }
  }, [lines, pendingPickupFocus]);

  async function toggle(line: ShoppingLine) {
    const checked = await setShoppingChecked({ workspaceId, lineKey: line.key, checked: !line.checked }).catch(() => line.checked);
    if (shoppingMode === "shop" && checked !== line.checked) setPendingPickupFocus({ key: line.key, action: checked ? "undo" : "check" });
    setLines((current) => current.map((item) => item.key === line.key ? { ...item, checked } : item));
  }
  async function toggleHaveThis(line: ShoppingLine) {
    const haveThis = await setShoppingHaveThis({ workspaceId, lineKey: line.key, haveThis: !line.haveThis }).catch(() => line.haveThis);
    setLines((current) => current.map((item) => item.key === line.key ? { ...item, haveThis } : item));
  }
  function purchaseEdit(line: ShoppingLine) {
    const pending = pendingPurchase?.lines.find((item) => item.lineKey === line.key);
    if (pending) return { amount: pending.amount, unit: pending.unit, price: pending.price, omitted: pending.omitted };
    return purchaseEdits[line.key] ?? { amount: line.actualQuantity || "", unit: line.actualUnit || line.need.match(/\s([^\s]+)$/)?.[1] || "", price: line.actualPrice || "", omitted: line.purchaseOmitted || false };
  }
  function editPurchase(line: ShoppingLine, patch: Partial<{ amount: string; unit: string; price: string; omitted: boolean }>) {
    if (pendingPurchase) return;
    setPurchaseEdits((current) => ({ ...current, [line.key]: { ...purchaseEdit(line), ...patch } }));
  }
  async function confirmTrip() {
    const request = pendingPurchase ?? immutablePurchase(workspaceId, `shopping-review-${crypto.randomUUID()}`, lines.map((line) => {
      const actual = purchaseEdit(line);
      return { lineKey: line.key, itemId: line.key.replace(/^ingredient:/, ""), amount: actual.amount, unit: actual.unit, price: actual.price, omitted: actual.omitted };
    }));
    let pendingPersisted = pendingPurchase ? pendingPurchaseStored : false;
    if (!pendingPurchase) {
      setPendingPurchase(request);
      try { sessionStorage.setItem(pendingPurchaseKey(workspaceId), JSON.stringify(request)); pendingPersisted = true; setPendingPurchaseStored(true); }
      catch { setPendingPurchaseStored(false); }
    }
    setPurchasePreviewOpen(true); setBatchAction("working"); setError(pendingPersisted ? "" : "This browser could not retain the pending review. Keep this tab open while confirmation is unresolved.");
    let commitConfirmed = false;
    try {
      await confirmShoppingPurchases({ workspaceId: request.workspaceId, reviewId: request.reviewId, lines: request.lines.map((line) => ({ ...line })) });
      commitConfirmed = true;
      const [preview, nextEvents] = await Promise.all([getShoppingPreview({ workspaceId, expectedRevision: planRevision }), listInventoryEvents(workspaceId)]);
      setLines(preview.lines); setEvents(nextEvents); setPurchaseEdits({}); setPendingPurchase(undefined); setError("");
      try { sessionStorage.removeItem(pendingPurchaseKey(workspaceId)); } catch { /* The in-memory request is cleared after confirmed refresh. */ }
      setPendingPurchaseStored(false);
      setTripConfirmed(true);
      setPurchasePreviewOpen(false);
    } catch (err: unknown) {
      if (!commitConfirmed && isCorrectablePurchaseValidation(err)) {
        setPendingPurchase(undefined);
        try { sessionStorage.removeItem(pendingPurchaseKey(workspaceId)); } catch { /* Keep validation correction available in memory. */ }
        setPendingPurchaseStored(false);
      }
      setError(err instanceof Error ? err.message : "Unable to confirm actual purchases.");
    }
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

  const visibleLines = lines.filter((line) => line.label.toLocaleLowerCase().includes(searchQuery.trim().toLocaleLowerCase()));
  const contributingMeals = [...new Set(lines.flatMap((line) => line.sourceRecipeIds))]
    .map((id) => recipes.find((recipe) => recipe.id === id)?.name)
    .filter((name): name is string => Boolean(name));
  const purchasePreviewRows = pendingPurchase
    ? pendingPurchase.lines.map((purchase) => ({ key: purchase.lineKey, label: lines.find((line) => line.key === purchase.lineKey)?.label || purchase.itemId || purchase.lineKey, amount: purchase.amount, unit: purchase.unit, price: purchase.price, omitted: purchase.omitted }))
    : lines.map((line) => ({ key: line.key, label: line.label, ...purchaseEdit(line) }));
  const hasPurchaseRows = pendingPurchase ? pendingPurchase.lines.length > 0 : lines.length > 0;
  const purchaseFieldsLocked = tripConfirmed || Boolean(pendingPurchase) || batchAction === "working";

  return <section aria-labelledby="groceries-heading" className="groceries-page flex flex-col gap-6">
    <header className="groceries-heading"><div><p>For your selected meals</p><h1 id="groceries-heading">Your groceries</h1><span>Plan-derived needs · checked items are not purchases</span></div><Link to="/kitchen">Your kitchen <span aria-hidden="true">›</span></Link></header>
    {state === "loading" && <p role="status" className="rounded border bg-white p-5 text-slate-600">Loading Kitchen…</p>}
    {(state === "error" || error) && <p role="alert" className="rounded border border-red-200 bg-red-50 p-5 text-red-800">{error || "Unable to load the Kitchen."}</p>}
    {state === "ready" && <article className="groceries-ledger-card"><h2 className="font-semibold">Prepared Kitchen stock</h2>{batches.length ? <ul aria-label="Prepared Kitchen batches" className="mt-3 divide-y">{batches.map((item) => <li key={item.id} className="py-3"><p className="font-medium">{recipes.find((recipe) => recipe.id === item.recipeId)?.name || item.recipeId} · revision {item.recipeRevision.toString()}</p><p className="text-sm text-slate-600">Measured yield {item.yieldAmount} {item.unit} · {item.availableAmount} {item.unit} available</p><div className="mt-2 flex flex-wrap items-end gap-2"><label className="grid gap-1 text-sm">Correct measured yield<input aria-label={`Correct yield for ${item.id}`} className="min-h-11 rounded border px-3" inputMode="decimal" value={yieldEdits[item.id] ?? item.yieldAmount} onChange={(event) => setYieldEdits((current) => ({ ...current, [item.id]: event.target.value }))} /></label><button type="button" className="min-h-11 rounded border px-3" disabled={batchAction === "working" || (yieldEdits[item.id] ?? item.yieldAmount) === item.yieldAmount} onClick={() => void correctYield(item)}>{batchAction === "working" ? "Saving…" : "Save yield correction"}</button><label className="grid gap-1 text-sm">Portion amount<input aria-label={`Portion amount for ${item.id}`} className="min-h-11 rounded border px-3" inputMode="decimal" value={portionEdits[item.id] ?? "1"} onChange={(event) => setPortionEdits((current) => ({ ...current, [item.id]: event.target.value }))} /></label><button type="button" className="min-h-11 rounded border px-3" disabled={batchAction === "working"} onClick={() => void changeSavedBatch(item,"consume")}>Consume portion from {item.id}</button><button type="button" className="min-h-11 rounded border px-3" disabled={batchAction === "working"} onClick={() => void changeSavedBatch(item,"undo")}>Undo portion for {item.id}</button></div><p className="mt-1 text-xs text-slate-500">Corrections preserve portions already consumed. Portion actions change prepared stock; recorded nutrition intake stays separate.</p></li>)}</ul> : <p className="mt-2 text-slate-600">No prepared batches yet. Finishing a cooking session does not add stock by itself.</p>}</article>}
    {state === "ready" && <article className="groceries-ledger-card"><h2 className="font-semibold">Cooking equipment</h2><p className="mt-1 text-sm text-slate-600">Only equipment you selected is shown as available. A recipe that needs another method remains unchanged.</p>{profile?.appliances.length ? <ul aria-label="Selected cooking equipment" className="mt-3 flex flex-wrap gap-2">{profile.appliances.map((appliance) => <li key={appliance} className="rounded-full bg-slate-100 px-3 py-1 text-sm">{appliance.split("_").join(" ")}</li>)}</ul> : <p className="mt-2 text-slate-600">No equipment capabilities are recorded. <Link className="underline" to="/setup">Choose equipment in setup</Link>.</p>}</article>}
    {state === "ready" && <div className="groceries-workspace">
      <div className="groceries-modebar" role="group" aria-label="Shopping mode"><button type="button" aria-pressed={shoppingMode === "review"} onClick={() => changeShoppingMode("review")}>Review</button><button type="button" aria-pressed={shoppingMode === "shop"} onClick={() => changeShoppingMode("shop")}>Shop</button><span>{lines.filter((line) => !line.checked).length} to pick up · {lines.filter((line) => line.checked).length} picked up</span></div>
      <div className="groceries-columns"><article className="groceries-list-card"><header className="groceries-card-heading"><div><p>{shoppingMode === "review" ? "Before you shop" : "At the store"}</p><h2>{shoppingMode === "review" ? "Review what you need" : "Shopping list"}</h2></div><button type="button" onClick={() => void downloadCSV()} disabled={exporting}>{exporting ? "Exporting…" : "Export list"}</button></header>
      <label className="groceries-search">Search items<input type="search" value={searchQuery} onChange={(event) => setSearchQuery(event.target.value)} placeholder="Search your list" /></label>
      {lines.length ? <ul aria-label="Shopping lines" className="groceries-line-list">{visibleLines.filter((line) => shoppingMode === "review" || !line.checked).map((line) => { const actual = purchaseEdit(line); return <li key={line.key} className={line.checked ? "is-picked-up" : ""}><label className="groceries-check"><input data-groceries-focus="check" data-line-key={line.key} type="checkbox" aria-label={`Check ${line.label}`} checked={line.checked} onChange={() => void toggle(line)} /><span>{line.checked ? "Picked up" : "Mark picked up"}</span></label><div className="groceries-line-copy"><h3>{line.label}</h3><p>Need {line.need} · Stock {line.stock}</p><p className="groceries-line-quantity">{line.missing === "unknown" ? "Check pantry" : `Need ${line.missing}`}</p>{shoppingMode === "review" && <label className="groceries-have"><input type="checkbox" aria-label={`Have this ${line.label}`} checked={line.haveThis} onChange={() => void toggleHaveThis(line)} /> Have this · amount not recorded</label>}
      {shoppingMode === "review" && <details className="groceries-details"><summary>Details and purchase review</summary><p>Remaining need: {line.missing} · Packages: {line.packageCount} · Package price: {line.price}</p><p>Portion cost: {line.portionCost ?? "unknown"} · Checkout total: {line.checkoutTotal ?? "unknown"} · Actual spend: {line.actualSpend ?? "unknown"}</p>{(line.actualQuantity || line.purchaseOmitted) && <p>Last trip: {line.purchaseOmitted ? "not purchased" : `${line.actualQuantity} ${line.actualUnit}`} · Actual price: {line.actualPrice || "unknown"}</p>}<div className="groceries-purchase-fields"><label>Actual quantity<input aria-label={`Actual quantity ${line.label}`} inputMode="decimal" value={actual.amount} disabled={actual.omitted || purchaseFieldsLocked} onChange={(event) => editPurchase(line, { amount: event.target.value })} /></label><label>Unit<input aria-label={`Actual unit ${line.label}`} value={actual.unit} disabled={actual.omitted || purchaseFieldsLocked} onChange={(event) => editPurchase(line, { unit: event.target.value })} /></label><label>Actual price<input aria-label={`Actual price ${line.label}`} value={actual.price} disabled={purchaseFieldsLocked} onChange={(event) => editPurchase(line, { price: event.target.value })} /></label><label><input type="checkbox" aria-label={`Omit ${line.label}`} checked={actual.omitted} disabled={purchaseFieldsLocked} onChange={(event) => editPurchase(line, { omitted: event.target.checked })} /> Not purchased</label></div></details>}</div><span className="groceries-line-action">{line.checked ? "Picked up" : shoppingMode === "shop" ? line.missing : "Details"}</span></li>; })}</ul> : <p className="groceries-empty">No ingredient evidence yet. Add ingredient inputs to a planned recipe to derive shopping lines.</p>}
      {shoppingMode === "shop" && lines.some((line) => line.checked) && <section className="groceries-picked"><h3>Picked up</h3><ul>{lines.filter((line) => line.checked && line.label.toLocaleLowerCase().includes(searchQuery.trim().toLocaleLowerCase())).map((line) => <li key={line.key}><span>{line.label}</span><button data-groceries-focus="undo" data-line-key={line.key} type="button" aria-label={`Undo pickup of ${line.label}`} onClick={() => void toggle(line)}>Undo</button></li>)}</ul></section>}
      <p className="groceries-note">Picked up is a checklist fact. It does not record payment, inventory, or eating.</p></article>
      <aside className="groceries-summary"><p className="groceries-eyebrow">Plan evidence</p><h2>From your planned meals</h2><p>Quantities follow the current plan and recorded stock evidence. Unknown amounts stay unknown.</p><section className="groceries-contributors"><h3>Contributing meals</h3>{contributingMeals.length ? <ul>{contributingMeals.map((meal) => <li key={meal}>{meal}</li>)}</ul> : <p>Meal names are unavailable from the current shopping evidence.</p>}</section><dl><div><dt>To pick up</dt><dd>{lines.filter((line) => !line.checked).length}</dd></div><div><dt>Picked up</dt><dd>{lines.filter((line) => line.checked).length}</dd></div><div><dt>Checkout estimate</dt><dd>Not recorded</dd></div></dl>
      {hasPurchaseRows && <div className="groceries-confirm"><p>Review actual quantities, prices, and omissions. Confirming records purchase stock once; it does not record eating.</p>{tripConfirmed ? <div><p role="status">This trip is confirmed. Start a new trip before recording another purchase.</p><button type="button" onClick={() => { setTripConfirmed(false); setPurchaseEdits({}); setPurchasePreviewOpen(false); }}>Start a new trip</button></div> : shoppingMode === "review" ? <><button type="button" disabled={purchaseFieldsLocked} onClick={() => setPurchasePreviewOpen(true)}>{purchasePreviewOpen ? "Update purchase preview" : "Preview purchase"}</button>{purchasePreviewOpen && <section className="groceries-purchase-preview" aria-label="Purchase preview"><h3>Confirm actual quantities</h3><p>{pendingPurchase ? `Confirmation is unresolved. Your preview is locked; retry sends the same review ID and exact purchase facts. ${pendingPurchaseStored ? "This review is retained for this browser tab." : "Keep this tab open; the API does not provide a receipt lookup if the tab session ends."}` : "This preview reflects your entered purchase facts. It records stock only after the final confirmation."}</p><ul>{purchasePreviewRows.map((row) => <li key={row.key}><strong>{row.label}</strong><span>{row.omitted ? "Not purchased" : row.amount ? `${row.amount} ${row.unit}` : "Quantity not recorded"}</span><small>Actual price: {row.price || "Not recorded"}</small></li>)}</ul><div><button type="button" disabled={Boolean(pendingPurchase) || batchAction === "working"} onClick={() => setPurchasePreviewOpen(false)}>Back to review</button><button type="button" disabled={batchAction === "working"} onClick={() => void confirmTrip()}>{batchAction === "working" ? "Saving…" : pendingPurchase ? "Retry exact confirmation" : "Confirm and record purchase"}</button></div></section>}</> : <button type="button" onClick={() => { changeShoppingMode("review"); setPurchasePreviewOpen(true); }}>Review purchases</button>}</div>}</aside></div></div>}

    {state === "ready" && <article className="groceries-ledger-card"><h2 className="font-semibold">Prepare a batch</h2><p className="mt-1 text-sm text-slate-600">Preparation records declared ingredient use once and creates stock only from a known recipe yield.</p><div className="mt-4 flex flex-wrap gap-3"><label>Recipe <select aria-label="Batch recipe" value={recipeId} onChange={(event) => setRecipeId(event.target.value)}><option value="">Choose a recipe</option>{recipes.map((recipe) => <option key={recipe.id} value={recipe.id}>{recipe.name}</option>)}</select></label><button type="button" onClick={() => void prepareBatch()} disabled={!recipeId || batchAction === "working"}>{batchAction === "working" ? "Working…" : "Prepare batch"}</button></div></article>}
  </section>;
}
