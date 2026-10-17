import { useEffect, useState, type KeyboardEvent } from "react";
import { Link } from "react-router-dom";
import type { Recipe } from "@vrooli/proto-types/nutrition-planner/v1/recipe/recipe_pb";
import type { Profile } from "@vrooli/proto-types/nutrition-planner/v1/profile/profile_pb";

import { listInventoryBatches, listInventoryEvents, type InventoryBatch, type InventoryEvent } from "../../api/inventory";
import { listRecipes } from "../../api/recipes";
import { applyProfile, getProfile } from "../../api/profile";
import { ensureWorkspace } from "../../api/workspace";

type KitchenState = "loading" | "ready" | "error";
type KitchenTab = "on-hand" | "equipment" | "preferences";
type EquipmentCategory = "Appliances" | "Cookware" | "Tools";

const applianceChoices = [
  { id: "stove", label: "Cooktop" },
  { id: "oven", label: "Oven" },
  { id: "microwave", label: "Microwave" },
  { id: "air_fryer", label: "Air fryer" },
  { id: "rice_cooker", label: "Rice cooker" },
  { id: "slow_cooker", label: "Slow cooker" },
  { id: "blender", label: "Blender" },
] as const;
const tabs: { id: KitchenTab; label: string }[] = [
  { id: "on-hand", label: "On hand" },
  { id: "equipment", label: "Equipment" },
  { id: "preferences", label: "Preferences" },
];
const equipmentCategories: EquipmentCategory[] = ["Appliances", "Cookware", "Tools"];

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
  const [tab, setTab] = useState<KitchenTab>("on-hand");
  const [equipmentCategory, setEquipmentCategory] = useState<EquipmentCategory>("Appliances");
  const [savingAppliance, setSavingAppliance] = useState("");

  useEffect(() => {
    let active = true;
    void ensureWorkspace().then(async ({ id }) => {
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

  const recentEvents = events.slice(-6).reverse();
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
            <p className="kitchen-eyebrow">Stock evidence</p><h2>Recent activity</h2>
            <p>Each entry is a recorded event. Opening this page does not refresh stock or infer a total.</p>
            {recentEvents.length ? <ul aria-label="Recent inventory activity">{recentEvents.map((event) => <li key={event.id}><div><strong>{event.itemId || event.recipeId || "Prepared food"}</strong><span>{event.kind.replace(/_/g, " ")} · {new Date(event.createdAt).toLocaleDateString()}</span></div><span>{event.amount ? `${event.amount} ${event.unit}` : "Amount not recorded"}</span></li>)}</ul> : <p className="kitchen-empty-copy">No inventory events are recorded. Raw ingredient amounts stay unknown until evidence is added.</p>}
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
            </> : <div className="kitchen-category-empty"><h3>{equipmentCategory} are not in the saved profile</h3><p>The current profile records appliance capabilities only. No cookware or tool selection is inferred.</p><Link to="/setup">Review setup</Link></div>}</div>
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
