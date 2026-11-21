import { useEffect, useState, type FormEvent, type KeyboardEvent } from "react";
import { Link } from "react-router-dom";
import type { Recipe } from "@vrooli/proto-types/nutrition-planner/v1/recipe/recipe_pb";
import type { Profile } from "@vrooli/proto-types/nutrition-planner/v1/profile/profile_pb";

import { listInventoryBatches, listInventoryEvents, recordInventoryEvent, type InventoryAmountMode, type InventoryBatch, type InventoryEvent } from "../../api/inventory";
import { listRecipes } from "../../api/recipes";
import { applyProfile, getProfile, saveKitchenDevices } from "../../api/profile";
import { ensureWorkspace } from "../../api/workspace";

type KitchenState = "loading" | "ready" | "error";
type KitchenTab = "on-hand" | "equipment" | "preferences";
type EquipmentCategory = "Appliances" | "Cookware" | "Tools";
type StorageFilter = "all" | "fridge" | "freezer" | "pantry";
const storageLabels: Record<Exclude<StorageFilter, "all"> | "", string> = { fridge: "Fridge", freezer: "Freezer", pantry: "Pantry", "": "Storage not recorded" };

const applianceChoices = [
  { id: "stove", label: "Cooktop" },
  { id: "oven", label: "Oven" },
  { id: "microwave", label: "Microwave" },
  { id: "air_fryer", label: "Air fryer" },
  { id: "rice_cooker", label: "Rice cooker" },
  { id: "slow_cooker", label: "Slow cooker" },
  { id: "blender", label: "Blender" },
] as const;
const cookwareChoices = [
  { id: "frying_pan", label: "Frying pan" }, { id: "saucepan", label: "Saucepan" },
  { id: "stockpot", label: "Stockpot" }, { id: "baking_sheet", label: "Baking sheet" },
] as const;
const toolChoices = [
  { id: "chef_knife", label: "Chef’s knife" }, { id: "cutting_board", label: "Cutting board" },
  { id: "mixing_bowl", label: "Mixing bowl" }, { id: "spatula", label: "Spatula" },
] as const;
const tabs: { id: KitchenTab; label: string }[] = [
  { id: "on-hand", label: "On hand" },
  { id: "equipment", label: "Equipment" },
  { id: "preferences", label: "Preferences" },
];
const equipmentCategories: EquipmentCategory[] = ["Appliances", "Cookware", "Tools"];
const deviceCapabilities: Record<string, Array<{ id: string; label: string }>> = {
  range: [{ id: "cooktop", label: "Cooktop" }, { id: "oven", label: "Oven" }],
  multicooker: [{ id: "pressure_cook", label: "Pressure cook" }, { id: "slow_cook", label: "Slow cook" }, { id: "saute", label: "Sauté" }, { id: "rice_cook", label: "Rice cook" }],
  microwave: [{ id: "microwave_heat", label: "Microwave heat" }],
  other: [{ id: "custom", label: "Custom function" }],
};

function moveTabFocus<T extends string>(event: KeyboardEvent<HTMLButtonElement>, items: readonly T[], selected: T, select: (item: T) => void, prefix: string) {
  const current = items.indexOf(selected);
  let next = current;
  if (event.key === "ArrowRight" || event.key === "ArrowDown") next = (current + 1) % items.length;
  else if (event.key === "ArrowLeft" || event.key === "ArrowUp") next = (current + items.length - 1) % items.length;
  else if (event.key === "Home") next = 0;
  else if (event.key === "End") next = items.length - 1;
  else return;
  event.preventDefault();
  select(items[next]!);
  document.getElementById(`${prefix}${items[next]}`)?.focus();
}

export function KitchenPage() {
  const [state, setState] = useState<KitchenState>("loading");
  const [error, setError] = useState("");
  const [batches, setBatches] = useState<InventoryBatch[]>([]);
  const [events, setEvents] = useState<InventoryEvent[]>([]);
  const [recipes, setRecipes] = useState<Recipe[]>([]);
  const [profile, setProfile] = useState<Profile>();
  const [appliances, setAppliances] = useState<string[]>([]);
  const [cookware, setCookware] = useState<string[]>([]);
  const [tools, setTools] = useState<string[]>([]);
  const [tab, setTab] = useState<KitchenTab>("on-hand");
  const [equipmentCategory, setEquipmentCategory] = useState<EquipmentCategory>("Appliances");
  const [savingAppliance, setSavingAppliance] = useState("");
  const [deviceID, setDeviceID] = useState("");
  const [deviceType, setDeviceType] = useState("range");
  const [deviceCount, setDeviceCount] = useState("1");
  const [deviceCapacity, setDeviceCapacity] = useState("");
  const [deviceFunctions, setDeviceFunctions] = useState<string[]>([]);
  const [savingDevices, setSavingDevices] = useState(false);
  const [stockItem, setStockItem] = useState("");
  const [stockMode, setStockMode] = useState<InventoryAmountMode>("exact");
  const [stockAmount, setStockAmount] = useState("");
  const [stockExpression, setStockExpression] = useState("");
  const [stockUnit, setStockUnit] = useState("");
  const [savingStock, setSavingStock] = useState(false);
  const [workspaceId, setWorkspaceId] = useState("");
  const [storageFilter, setStorageFilter] = useState<StorageFilter>("all");
  const [inventorySearch, setInventorySearch] = useState("");
  const [stockStorage, setStockStorage] = useState<Exclude<StorageFilter, "all">>("pantry");
  const [bestBeforeDate, setBestBeforeDate] = useState("");
  const [useSoon, setUseSoon] = useState(false);
  const [adjustingEventId, setAdjustingEventId] = useState("");
  const [movingStorage, setMovingStorage] = useState<Record<string, Exclude<StorageFilter, "all">>>({});

  useEffect(() => {
    let active = true;
    void ensureWorkspace().then(async ({ id }) => {
      if (active) setWorkspaceId(id);
      const [inventory, activity, mealList, savedProfile] = await Promise.all([
        listInventoryBatches(id),
        listInventoryEvents(id),
        listRecipes(id),
        getProfile(id),
      ]);
      if (!active) return;
      setBatches(inventory);
      setEvents(activity);
      setRecipes(mealList);
      setProfile(savedProfile);
      setAppliances(savedProfile?.appliances ?? []);
      setCookware(savedProfile?.cookware ?? []);
      setTools(savedProfile?.tools ?? []);
      setState("ready");
    }).catch((reason: unknown) => {
      if (!active) return;
      setError(reason instanceof Error ? reason.message : "Unable to load Kitchen inventory.");
      setState("error");
    });
    return () => { active = false; };
  }, []);

  async function toggleAppliance(appliance: string) {
    if (!profile || savingAppliance) return;
    const previous = appliances;
    const next = previous.includes(appliance)
      ? previous.filter((value) => value !== appliance)
      : [...previous, appliance];
    setAppliances(next);
    setSavingAppliance(appliance);
    setError("");
    try {
      const result = await applyProfile({
        workspaceId: profile.workspaceId,
        preset: profile.preset,
        excludedGroups: profile.excludedGroups,
        allergies: profile.allergies,
        appliances: next,
        cookware: profile.cookware,
        tools: profile.tools,
        costWeight: profile.costWeight,
        effortWeight: profile.effortWeight,
        varietyWeight: profile.varietyWeight,
      });
      setProfile(result.profile);
      setAppliances(result.profile.appliances);
    } catch (reason: unknown) {
      setAppliances(previous);
      setError(reason instanceof Error ? reason.message : "Equipment selection could not be saved.");
    } finally {
      setSavingAppliance("");
    }
  }

  async function toggleEquipment(category: "Cookware" | "Tools", id: string) {
    if (!profile || savingAppliance) return;
    const previous = category === "Cookware" ? cookware : tools;
    const next = previous.includes(id) ? previous.filter((value) => value !== id) : [...previous, id];
    const update = category === "Cookware" ? setCookware : setTools;
    update(next);
    setSavingAppliance(id);
    setError("");
    try {
      const result = await applyProfile({
        workspaceId: profile.workspaceId, preset: profile.preset,
        excludedGroups: profile.excludedGroups, allergies: profile.allergies,
        appliances: profile.appliances, cookware: category === "Cookware" ? next : profile.cookware,
        tools: category === "Tools" ? next : profile.tools,
        costWeight: profile.costWeight, effortWeight: profile.effortWeight, varietyWeight: profile.varietyWeight,
      });
      setProfile(result.profile);
      setAppliances(result.profile.appliances);
      setCookware(result.profile.cookware);
      setTools(result.profile.tools);
    } catch (reason: unknown) {
      update(previous);
      setError(reason instanceof Error ? reason.message : `${category} selection could not be saved.`);
    } finally {
      setSavingAppliance("");
    }
  }

  function clearDeviceDraft() {
    setDeviceID(""); setDeviceType("range"); setDeviceCount("1"); setDeviceCapacity(""); setDeviceFunctions([]);
  }

  async function saveDevice(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!profile || savingDevices) return;
    setSavingDevices(true); setError("");
    const id = deviceID || `device-${crypto.randomUUID()}`;
    const devices = [...(profile.kitchenDevices ?? []).filter((device) => device.id !== deviceID), { id, type: deviceType, count: Number(deviceCount), capacity: deviceCapacity.trim(), capabilities: deviceFunctions }];
    try {
      const result = await saveKitchenDevices(profile.workspaceId, devices);
      setProfile(result);
      clearDeviceDraft();
    } catch (reason: unknown) {
      setError(reason instanceof Error ? reason.message : "Physical device could not be saved.");
    } finally { setSavingDevices(false); }
  }

  async function removeDevice(id: string) {
    if (!profile || savingDevices) return;
    setSavingDevices(true); setError("");
    try {
      const result = await saveKitchenDevices(profile.workspaceId, (profile.kitchenDevices ?? []).filter((device) => device.id !== id));
      setProfile(result);
      if (deviceID === id) clearDeviceDraft();
    } catch (reason: unknown) {
      setError(reason instanceof Error ? reason.message : "Physical device could not be removed.");
    } finally { setSavingDevices(false); }
  }

  function editDevice(device: Profile["kitchenDevices"][number]) {
    setDeviceID(device.id); setDeviceType(device.type); setDeviceCount(String(device.count));
    setDeviceCapacity(device.capacity); setDeviceFunctions([...device.capabilities]);
  }

  function equipmentTiles(choices: readonly { id: string; label: string }[], selectedIds: string[], toggle: (id: string) => void) {
    return <div className="kitchen-equipment-tiles">{choices.map((choice) => {
      const selected = selectedIds.includes(choice.id);
      return <button key={choice.id} type="button" className={selected ? "is-selected" : ""} aria-pressed={selected} aria-label={`${selected ? "Remove" : "Select"} ${choice.label}`} disabled={!profile || Boolean(savingAppliance)} onClick={() => toggle(choice.id)}><span className="kitchen-equipment-icon" aria-hidden="true">{selected ? "✓" : "+"}</span><strong>{choice.label}</strong><small>{savingAppliance === choice.id ? "Saving…" : selected ? "Selected" : "Not selected"}</small></button>;
    })}</div>;
  }

  async function addStockAssertion(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!workspaceId || savingStock) return;
    setSavingStock(true);
    setError("");
    try {
      const saved = await recordInventoryEvent({
        workspaceId, id: `assertion-${crypto.randomUUID()}`, lotId: adjustingEventId || `lot-${crypto.randomUUID()}`, kind: "assertion", itemId: stockItem.trim(),
        amount: stockMode === "exact" || stockMode === "estimated" ? stockAmount.trim() : "unknown",
        amountMode: stockMode, expression: stockMode === "qualitative" ? stockExpression.trim() : "",
        storage: stockStorage, bestBeforeDate, useSoon,
        unit: stockMode === "exact" || stockMode === "estimated" ? stockUnit.trim() : "",
      });
      setEvents((current) => [...current, saved]);
      setAdjustingEventId("");
      setStockItem(""); setStockAmount(""); setStockExpression(""); setStockUnit(""); setBestBeforeDate(""); setUseSoon(false);
    } catch (reason: unknown) {
      setError(reason instanceof Error ? reason.message : "Stock evidence could not be saved.");
    } finally { setSavingStock(false); }
  }

  async function recordEvidenceRevision(event: InventoryEvent, changes: Partial<Pick<InventoryEvent, "amount" | "amountMode" | "expression" | "storage" | "bestBeforeDate" | "useSoon" | "unit">>) {
    if (savingStock || !workspaceId) return;
    const mode = changes.amountMode ?? event.amountMode;
    const numeric = mode === "exact" || mode === "estimated";
    setSavingStock(true);
    setError("");
    try {
      const saved = await recordInventoryEvent({ workspaceId, id: `assertion-${crypto.randomUUID()}`, lotId: event.lotId || event.id, kind: "assertion", itemId: event.itemId,
        amount: numeric ? changes.amount ?? event.amount : "unknown", amountMode: mode, expression: mode === "qualitative" ? changes.expression ?? event.expression : "",
        storage: changes.storage ?? event.storage, bestBeforeDate: changes.bestBeforeDate ?? event.bestBeforeDate, useSoon: changes.useSoon ?? event.useSoon,
        unit: numeric ? changes.unit ?? event.unit : "" });
      setEvents((current) => [...current, saved]);
    } catch (reason: unknown) {
      setError(reason instanceof Error ? reason.message : "Stock evidence could not be saved.");
    } finally { setSavingStock(false); }
  }

  function prepareAmountAdjustment(event: InventoryEvent) {
    setStockItem(event.itemId);
    setStockMode(event.amountMode);
    setStockAmount(event.amount === "unknown" ? "" : event.amount);
    setStockExpression(event.expression);
    setStockUnit(event.unit);
    setStockStorage(event.storage === "fridge" || event.storage === "freezer" ? event.storage : "pantry");
    setBestBeforeDate(event.bestBeforeDate);
    setUseSoon(event.useSoon);
    setAdjustingEventId(event.id);
    document.querySelector(".kitchen-stock-form")?.scrollIntoView({ behavior: "smooth", block: "center" });
  }

  function eventAmount(event: InventoryEvent) {
    if (event.amountMode === "qualitative") return event.expression || "Qualitative amount not recorded";
    if (event.amountMode === "out") return "Out";
    if (event.amountMode === "unknown") return "Amount not set";
    const quantity = event.amount && event.amount !== "unknown" ? `${event.amount}${event.unit ? ` ${event.unit}` : ""}` : "Amount not set";
    return event.amountMode === "estimated" ? `About ${quantity}` : quantity;
  }

  const localNow = new Date();
  const today = `${localNow.getFullYear()}-${String(localNow.getMonth() + 1).padStart(2, "0")}-${String(localNow.getDate()).padStart(2, "0")}`;
  const latestByLot = new Map<string, InventoryEvent>();
  for (const event of events) {
    if (!event.itemId) continue;
    const key = event.lotId || event.id;
    const current = latestByLot.get(key);
    if (!current || event.createdAt >= current.createdAt) latestByLot.set(key, event);
  }
  const currentLots = [...latestByLot.values()];
  const useSoonItems = currentLots.filter((event) => {
    if (!event.itemId) return false;
    if (event.amountMode === "out") return false;
    if (event.useSoon) return true;
    if (!event.bestBeforeDate) return false;
    const days = Math.floor((Date.parse(event.bestBeforeDate) - Date.parse(today)) / 86400000);
    return days >= 0 && days <= 3;
  });
  function dateDerivedUseSoon(event: InventoryEvent) {
    if (!event.bestBeforeDate) return false;
    const days = Math.floor((Date.parse(event.bestBeforeDate) - Date.parse(today)) / 86400000);
    return days >= 0 && days <= 3;
  }
  function eventStatus(event: InventoryEvent) {
    if (event.useSoon || dateDerivedUseSoon(event)) return event.useSoon ? "Use soon · set by you" : "Use soon · based on recorded date";
    if (event.amountMode === "out") return "Out";
    if (event.amountMode === "unknown") return "Unknown";
    return "Available";
  }
  const latestCheck = events.filter((event) => event.kind === "assertion").reduce((latest, event) => event.createdAt > latest ? event.createdAt : latest, "");
  const searchTerm = inventorySearch.trim().toLocaleLowerCase();
  const inventoryEvents = currentLots.filter((event) => (storageFilter === "all" || event.storage === storageFilter) && (!searchTerm || event.itemId.toLocaleLowerCase().includes(searchTerm)));
  function inventoryEventRow(event: InventoryEvent) {
    return <li key={event.id}><div><strong>{event.itemId}</strong><span>{event.kind.replace(/_/g, " ")} · {new Date(event.createdAt).toLocaleDateString()}</span><span>{eventStatus(event)}</span></div><span>{eventAmount(event)}{event.bestBeforeDate ? ` · Best before ${event.bestBeforeDate}` : ""}</span>
      <details><summary>Actions</summary><button type="button" onClick={() => prepareAmountAdjustment(event)}>Adjust amount</button>
        <label>Move to<select aria-label={`Move ${event.itemId} to storage`} value={movingStorage[event.id] ?? (event.storage === "fridge" || event.storage === "freezer" ? event.storage : "pantry")} onChange={(change) => setMovingStorage((current) => ({ ...current, [event.id]: change.target.value as Exclude<StorageFilter, "all"> }))}><option value="fridge">Fridge</option><option value="freezer">Freezer</option><option value="pantry">Pantry</option></select></label>
        <button type="button" disabled={savingStock} onClick={() => void recordEvidenceRevision(event, { storage: movingStorage[event.id] ?? "pantry" })}>Save storage change</button>
        <button type="button" disabled={savingStock || event.amountMode === "out"} onClick={() => void recordEvidenceRevision(event, { amountMode: "out", amount: "unknown", unit: "", useSoon: false })}>Mark out</button>
        <button type="button" disabled={savingStock || event.useSoon} onClick={() => void recordEvidenceRevision(event, { useSoon: true })}>Flag use soon</button>
        <details><summary>View evidence</summary><p>Recorded {new Date(event.createdAt).toLocaleString()}; storage {storageLabels[event.storage as Exclude<StorageFilter, "all"> | ""] ?? "not recorded"}; amount mode {event.amountMode}{event.expression ? `: ${event.expression}` : ""}.</p></details>
      </details>
    </li>;
  }

  const selectedApplianceLabels = appliances.map((id) => applianceChoices.find((choice) => choice.id === id)?.label ?? id.replace(/_/g, " "));
  return <section className="kitchen-page" aria-labelledby="kitchen-heading">
    <header className="kitchen-heading">
      <div><p>Your kitchen</p><h1 id="kitchen-heading">{tab === "on-hand" ? "Your kitchen" : tab === "equipment" ? "Your kitchen, your methods" : "Your preferences"}</h1><span>{tab === "on-hand" ? "On hand · recorded stock evidence" : tab === "equipment" ? selectedApplianceLabels.length ? `Selected: ${selectedApplianceLabels.join(", ")}` : "No appliance capabilities selected" : "Saved food and planning profile"}</span></div>
      <Link to="/groceries">Open groceries</Link>
    </header>
    {state === "loading" && <p role="status" className="kitchen-state">Loading your saved kitchen…</p>}
    {state === "error" && <p role="alert" className="kitchen-state">{error}</p>}
    {state === "ready" && <>
      <nav className="kitchen-tabs" aria-label="Kitchen sections" role="tablist">
        {tabs.map((item) => <button key={item.id} id={`kitchen-tab-${item.id}`} type="button" role="tab" tabIndex={tab === item.id ? 0 : -1} aria-selected={tab === item.id} aria-controls="kitchen-panel" onKeyDown={(event) => moveTabFocus(event, tabs.map((entry) => entry.id), tab, setTab, "kitchen-tab-")} onClick={() => setTab(item.id)}>{item.label}</button>)}
      </nav>
      {error && <p role="alert" className="kitchen-inline-error">{error}</p>}
      <div id="kitchen-panel" role="tabpanel" aria-labelledby={`kitchen-tab-${tab}`} className="kitchen-panel">
        {tab === "on-hand" && <div className="kitchen-layout">
          <article className="kitchen-card kitchen-stock">
            <div className="kitchen-card-heading"><div><p>Prepared food</p><h2>On hand</h2></div><span>{batches.length} {batches.length === 1 ? "batch" : "batches"}</span></div>
            {batches.length ? <ul aria-label="Prepared portions">{batches.map((batch) => {
              const recipe = recipes.find((item) => item.id === batch.recipeId);
              return <li key={batch.id}><div><h3>{recipe?.name ?? "Saved recipe"}</h3><p>Recipe revision {batch.recipeRevision.toString()} · measured yield {batch.yieldAmount} {batch.unit}</p></div><strong>{batch.availableAmount} {batch.unit}<span>recorded portions</span></strong></li>;
            })}</ul> : <div className="kitchen-empty"><h3>No prepared portions yet</h3><p>Finished cooking sessions appear here only after you confirm a measured yield.</p><Link to="/meals">Browse your meals</Link></div>}
            <p className="kitchen-note">Prepared portions, raw ingredients, shopping pickups and eaten meals remain separate records.</p>
          </article>
          <aside className="kitchen-ledger">
            <p className="kitchen-eyebrow">Stock evidence</p><h2>Ingredients</h2>
            <p>Last checked: {latestCheck ? new Date(latestCheck).toLocaleString() : "No check recorded"}. Opening this page does not change it.</p>
            <label htmlFor="kitchen-inventory-search">Search ingredients<input id="kitchen-inventory-search" type="search" value={inventorySearch} onChange={(event) => setInventorySearch(event.target.value)} placeholder="Search your stock" /></label>
            <div aria-label="Storage filters" role="group">{(["all", "fridge", "freezer", "pantry"] as const).map((filter) => <button key={filter} type="button" aria-pressed={storageFilter === filter} onClick={() => setStorageFilter(filter)}>{filter === "all" ? "All" : storageLabels[filter]}</button>)}</div>
            <form className="kitchen-stock-form" aria-label="Add ingredient" onSubmit={(event) => void addStockAssertion(event)}>
              <label>Ingredient<input required value={stockItem} onChange={(event) => setStockItem(event.target.value)} placeholder="e.g. rice" /></label>
              <label>Storage<select value={stockStorage} onChange={(event) => setStockStorage(event.target.value as Exclude<StorageFilter, "all">)}><option value="fridge">Fridge</option><option value="freezer">Freezer</option><option value="pantry">Pantry</option></select></label>
              <label>Best-before date<input type="date" value={bestBeforeDate} onChange={(event) => setBestBeforeDate(event.target.value)} /></label>
              <label><input type="checkbox" checked={useSoon} onChange={(event) => setUseSoon(event.target.checked)} />Use soon (set by you)</label>
              <label>Amount mode<select value={stockMode} onChange={(event) => setStockMode(event.target.value as InventoryAmountMode)}><option value="exact">Exact</option><option value="estimated">Estimated</option><option value="qualitative">Qualitative</option><option value="out">Out</option><option value="unknown">Unknown</option></select></label>
              {(stockMode === "exact" || stockMode === "estimated") && <><label>Amount<input required inputMode="decimal" value={stockAmount} onChange={(event) => setStockAmount(event.target.value)} placeholder="0" /></label><label>Unit<input required value={stockUnit} onChange={(event) => setStockUnit(event.target.value)} placeholder="g, bags…" /></label></>}
              {stockMode === "qualitative" && <label>Original expression<input required value={stockExpression} onChange={(event) => setStockExpression(event.target.value)} placeholder="Some left, about half a bag…" /></label>}
              <button type="submit" disabled={savingStock || !workspaceId}>{savingStock ? "Saving…" : adjustingEventId ? "Save amount update" : "Add ingredient"}</button>
            </form>
            {useSoonItems.length > 0 && <section aria-label="Use soon"><h3>Use soon</h3><p>Date-derived items appear within 3 days of their recorded best-before date. This is a planning reminder, not food-safety advice.</p><ul>{[...new Set(useSoonItems.map((event) => event.itemId))].map((item) => <li key={item}>{item} · {useSoonItems.some((event) => event.itemId === item && event.useSoon) ? "set by you" : "based on recorded date"}</li>)}</ul><Link to={`/meals/explore?ingredients=${encodeURIComponent([...new Set(useSoonItems.map((event) => event.itemId))].join(","))}&rules=${encodeURIComponent(profile?.activeRules.join(",") ?? "")}`}>Find meals</Link></section>}
            <p>Storage is where you recorded each item. Entries are evidence, not a computed stock total.</p>
            {inventoryEvents.length ? <div aria-label="Inventory grouped by storage">{(["fridge", "freezer", "pantry", ""] as const).filter((storage) => inventoryEvents.some((event) => event.storage === storage)).map((storage) => <section key={storage || "unrecorded"}><h3>{storageLabels[storage]}</h3><ul>{inventoryEvents.filter((event) => event.storage === storage).map(inventoryEventRow)}</ul></section>)}</div> : <p className="kitchen-empty-copy">No ingredient evidence matches this search and storage filter. Add an ingredient or change the filter.</p>}
            <Link to="/groceries">Manage shopping and receipts</Link>
          </aside>
        </div>}

        {tab === "equipment" && <div className="kitchen-equipment-layout">
          <div className="kitchen-scene-fallback" role="img" aria-label="Kitchen scene preview unavailable; equipment tiles show saved capabilities">
            <span aria-hidden="true">✳</span><p>Equipment defines your methods</p><small>No approved kitchen scene is available in this build.</small>
          </div>
          <article className="kitchen-card kitchen-equipment-panel">
            <div className="kitchen-card-heading"><div><p>Capabilities, not decoration</p><h2>Equipment</h2></div><Link to="/setup">Setup</Link></div>
            <nav className="kitchen-category-tabs" aria-label="Equipment categories" role="tablist">
              {equipmentCategories.map((category) => <button key={category} id={`equipment-tab-${category}`} type="button" role="tab" tabIndex={equipmentCategory === category ? 0 : -1} aria-selected={equipmentCategory === category} aria-controls="equipment-category-panel" onKeyDown={(event) => moveTabFocus(event, equipmentCategories, equipmentCategory, setEquipmentCategory, "equipment-tab-")} onClick={() => setEquipmentCategory(category)}>{category}</button>)}
            </nav>
            <div id="equipment-category-panel" role="tabpanel" aria-labelledby={`equipment-tab-${equipmentCategory}`}>{equipmentCategory === "Appliances" ? <>
              <p className="kitchen-section-note">Only saved choices are treated as available. Selecting a tile saves this capability to your existing profile.</p>
              <div className="kitchen-equipment-tiles" aria-label="Appliance capabilities">{applianceChoices.map((choice) => {
                const selected = appliances.includes(choice.id);
                return <button key={choice.id} type="button" className={selected ? "is-selected" : ""} aria-pressed={selected} aria-label={`${selected ? "Remove" : "Select"} ${choice.label}`} disabled={!profile || Boolean(savingAppliance)} onClick={() => void toggleAppliance(choice.id)}><span className="kitchen-equipment-icon" aria-hidden="true">{selected ? "✓" : "+"}</span><strong>{choice.label}</strong><small>{savingAppliance === choice.id ? "Saving…" : selected ? "Selected" : "Not selected"}</small></button>;
              })}</div>
              {!profile && <p className="kitchen-section-note">No saved profile exists yet. Choose equipment in setup before changing capabilities.</p>}
            </> : <><p className="kitchen-section-note">Choose the items you have. Selections save to this workspace profile.</p>{equipmentCategory === "Cookware" ? equipmentTiles(cookwareChoices, cookware, (id) => void toggleEquipment("Cookware", id)) : equipmentTiles(toolChoices, tools, (id) => void toggleEquipment("Tools", id))}{!profile && <p className="kitchen-section-note">No saved profile exists yet. Choose equipment in setup before changing capabilities.</p>}</>}</div>
            <section className="kitchen-device-details" aria-labelledby="kitchen-devices-heading">
              <div className="kitchen-card-heading"><div><p>Physical equipment</p><h3 id="kitchen-devices-heading">Devices</h3></div></div>
              <p className="kitchen-section-note">Record each appliance once, then select only the functions enabled on that device. Device records do not change capability category selections.</p>
              {profile?.kitchenDevices?.length ? <ul className="kitchen-device-list">{profile.kitchenDevices.map((device) => <li key={device.id}>
                <div><strong>{device.type === "multicooker" ? "Multicooker" : device.type === "range" ? "Range" : device.type}</strong><span>Identity: {device.id}</span><span>Count: {device.count} · Capacity: {device.capacity}</span><span>Enabled: {device.capabilities.length ? device.capabilities.map((id) => deviceCapabilities[device.type]?.find((option) => option.id === id)?.label ?? id).join(", ") : "No functions recorded"}</span></div>
                <div><button type="button" disabled={savingDevices} onClick={() => editDevice(device)}>Edit</button><button type="button" disabled={savingDevices} onClick={() => void removeDevice(device.id)}>Remove</button></div>
              </li>)}</ul> : <p className="kitchen-empty-copy">No physical devices recorded. Your capability selections remain separate.</p>}
              {profile ? <form className="kitchen-device-form" aria-label={deviceID ? "Edit physical device" : "Add physical device"} onSubmit={(event) => void saveDevice(event)}>
                <label>Device type<select value={deviceType} onChange={(event) => { setDeviceType(event.target.value); setDeviceFunctions([]); }}><option value="range">Range</option><option value="multicooker">Multicooker</option><option value="microwave">Microwave</option><option value="other">Other</option></select></label>
                <label>Count<input type="number" min="1" required value={deviceCount} onChange={(event) => setDeviceCount(event.target.value)} /></label>
                <label>Capacity<input required value={deviceCapacity} onChange={(event) => setDeviceCapacity(event.target.value)} placeholder="e.g. 6 qt or 5 burners" /></label>
                <fieldset><legend>Enabled functions</legend>{deviceCapabilities[deviceType]?.map((capability) => <label key={capability.id}><input type="checkbox" checked={deviceFunctions.includes(capability.id)} onChange={(event) => setDeviceFunctions((current) => event.target.checked ? [...current, capability.id] : current.filter((value) => value !== capability.id))} />{capability.label}</label>)}</fieldset>
                <button type="submit" disabled={savingDevices}>{savingDevices ? "Saving…" : deviceID ? "Save device" : "Add device"}</button>
                {deviceID && <button type="button" onClick={clearDeviceDraft}>Cancel edit</button>}
              </form> : <p className="kitchen-section-note">Complete setup before recording physical devices.</p>}
            </section>
            <p className="kitchen-note">Physical devices and preparation capabilities are distinct. A selected cooktop does not imply an oven.</p>
          </article>
        </div>}

        {tab === "preferences" && <div className="kitchen-preferences-layout">
          <article className="kitchen-card kitchen-preferences">
            <div className="kitchen-card-heading"><div><p>Your saved setup</p><h2>Preferences</h2></div><Link to="/setup">Review setup</Link></div>
            {profile ? <div className="kitchen-preference-groups">
              <section><h3>Food pattern</h3><p>{profile.preset || "Not configured"}</p></section>
              <section><h3>Allergies</h3><p>{profile.allergies.length ? profile.allergies.join(", ") : "No allergies recorded"}</p></section>
              <section><h3>Excluded groups</h3><p>{profile.excludedGroups.length ? profile.excludedGroups.join(", ") : "No excluded groups recorded"}</p></section>
              <section><h3>Planning priorities</h3><p>Cost {profile.costWeight.toFixed(2)} · Effort {profile.effortWeight.toFixed(2)} · Variety {profile.varietyWeight.toFixed(2)}</p></section>
              <section><h3>Active rules</h3><p>{profile.activeRules.length ? profile.activeRules.join(", ") : "No active rules recorded"}</p></section>
            </div> : <div className="kitchen-empty"><h3>No saved profile yet</h3><p>Preferences remain unconfigured until you complete setup.</p><Link to="/setup">Start setup</Link></div>}
            <p className="kitchen-note">These are saved profile values. Nutrition targets, household servings, schedules and shopping preferences are not represented in this profile response.</p>
          </article>
        </div>}
      </div>
    </>}
  </section>;
}
