import { useEffect, useMemo, useState } from "react";
import { applyProfile, getProfile, saveProfileDraft } from "../../api/profile";
import { ensureWorkspace } from "../../api/workspace";

const steps = ["Diet", "Allergies", "Kitchen", "Priorities"] as const;
const appliances = ["oven", "stove", "microwave", "air_fryer", "blender", "rice_cooker", "slow_cooker", "none"] as const;
const presets = ["everything", "vegan", "vegetarian", "pescatarian", "plant-forward", "my-own-way"] as const;

export function OnboardingPage() {
  const [step, setStep] = useState(0);
  const [workspaceId, setWorkspaceId] = useState("");
  const [preset, setPreset] = useState("everything");
  const [allergies, setAllergies] = useState("");
  const [selectedAppliances, setSelectedAppliances] = useState<string[]>([]);
  const [cost, setCost] = useState(0.34);
  const [effort, setEffort] = useState(0.33);
  const [variety, setVariety] = useState(0.33);
  const [status, setStatus] = useState<"loading" | "ready" | "saving" | "saved" | "error">("loading");
  const [error, setError] = useState("");
  const [mealCounts, setMealCounts] = useState<{ matching: bigint; needsReview: bigint; excluded: bigint }>();

  const draft = useMemo(() => JSON.stringify({ step, preset, allergies, appliances: selectedAppliances, cost, effort, variety }), [step, preset, allergies, selectedAppliances, cost, effort, variety]);
  useEffect(() => {
    let mounted = true;
    void ensureWorkspace().then(async (workspace) => {
      const existing = await getProfile(workspace.id).catch(() => undefined);
      if (!mounted) return;
      setWorkspaceId(workspace.id);
      if (existing?.draftJson) {
        try {
          const saved = JSON.parse(existing.draftJson) as { step?: number; preset?: string; allergies?: string; appliances?: string[]; cost?: number; effort?: number; variety?: number };
          if (typeof saved.step === "number") setStep(Math.min(3, Math.max(0, saved.step)));
          if (saved.preset) setPreset(saved.preset);
          if (typeof saved.allergies === "string") setAllergies(saved.allergies);
          if (saved.appliances) setSelectedAppliances(saved.appliances);
          if (typeof saved.cost === "number") setCost(saved.cost);
          if (typeof saved.effort === "number") setEffort(saved.effort);
          if (typeof saved.variety === "number") setVariety(saved.variety);
        } catch { setError("Your saved setup draft could not be read; starting with a blank draft."); }
      }
      setStatus("ready");
    }).catch((err: unknown) => { if (mounted) { setError(err instanceof Error ? err.message : "Unable to load setup."); setStatus("error"); } });
    return () => { mounted = false; };
  }, []);

  async function next() {
    if (!workspaceId) return;
    setStatus("saving"); setError("");
    try {
      if (step < steps.length - 1) { await saveProfileDraft(workspaceId, draft); setStep((current) => current + 1); setStatus("ready"); }
      else { const result = await applyProfile({ workspaceId, preset, excludedGroups: [], allergies: allergies.split(",").map((value) => value.trim()).filter(Boolean), appliances: selectedAppliances.filter((value) => value !== "none"), costWeight: cost, effortWeight: effort, varietyWeight: variety }); setMealCounts({ matching: result.matchingMeals, needsReview: result.needsReviewMeals, excluded: result.excludedMeals }); setStatus("saved"); }
    } catch (err: unknown) { setError(err instanceof Error ? err.message : "Setup was not saved. Your answers are still here."); setStatus("error"); }
  }

  function toggleAppliance(value: string) { setSelectedAppliances((current) => value === "none" ? (current.includes("none") ? [] : ["none"]) : current.filter((item) => item !== "none").includes(value) ? current.filter((item) => item !== value) : [...current.filter((item) => item !== "none"), value]); }

  return <section aria-labelledby="setup-heading" className="flex max-w-3xl flex-col gap-6"><header><p className="text-sm font-medium uppercase tracking-wide text-cyan-700">Daily / Setup</p><h1 id="setup-heading" className="text-3xl font-semibold text-slate-900">Make your plan fit your life</h1><p className="mt-2 text-slate-600">Your draft is saved separately from active rules until you apply it.</p></header><ol aria-label="Setup steps" className="grid grid-cols-4 gap-2">{steps.map((label, index) => <li key={label} aria-current={index === step ? "step" : undefined} className={`rounded border p-2 text-center text-sm ${index === step ? "border-blue-600 bg-blue-50 font-semibold" : "border-slate-200"}`}>{index + 1}. {label}</li>)}</ol><div className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">{step === 0 && <fieldset><legend className="text-lg font-semibold">What kind of eating pattern fits?</legend><div className="mt-4 grid gap-2 sm:grid-cols-2">{presets.map((value) => <label key={value} className="flex min-h-11 items-center gap-2 rounded border p-3"><input type="radio" name="preset" value={value} checked={preset === value} onChange={() => setPreset(value)} />{value}</label>)}</div></fieldset>}{step === 1 && <div><label className="text-lg font-semibold" htmlFor="allergies">Allergies or exclusions to review</label><input id="allergies" value={allergies} onChange={(event) => setAllergies(event.target.value)} className="mt-4 min-h-11 w-full rounded border border-slate-300 px-3" placeholder="e.g. peanut, soy" /><p className="mt-2 text-sm text-slate-600">Separate items with commas. Missing evidence stays unknown.</p></div>}{step === 2 && <fieldset><legend className="text-lg font-semibold">Which kitchen capabilities do you have?</legend><div className="mt-4 grid gap-2 sm:grid-cols-2">{appliances.map((value) => <label key={value} className="flex min-h-11 items-center gap-2 rounded border p-3"><input type="checkbox" checked={selectedAppliances.includes(value)} onChange={() => toggleAppliance(value)} />{value === "none" ? "None — assemble-ready meals are valid" : value.replace("_", " ")}</label>)}</div></fieldset>}{step === 3 && <fieldset><legend className="text-lg font-semibold">What matters most this week?</legend><div className="mt-4 grid gap-4">{[["cost", cost, setCost], ["effort", effort, setEffort], ["variety", variety, setVariety]].map(([label, value, setter]) => <label key={label as string} className="grid gap-1 text-sm capitalize">{label as string}: {Number(value).toFixed(2)}<input type="range" min="0" max="1" step="0.01" value={value as number} onChange={(event) => (setter as (v: number) => void)(Number(event.target.value))} /></label>)}</div><p className="mt-3 text-sm text-slate-600">These are editable weights; a custom balance stays custom.</p></fieldset>}{error && <p role="alert" className="mt-5 rounded border border-red-200 bg-red-50 p-3 text-sm text-red-800">{error}</p>}{status === "saved" && <div role="status" className="mt-5 rounded border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-800"><p>Active profile applied.</p>{mealCounts && <dl className="mt-1"><div>Fits: {mealCounts.matching.toString()}</div><div>Needs review: {mealCounts.needsReview.toString()}</div><div>Excluded: {mealCounts.excluded.toString()}</div></dl>}{mealCounts?.matching === 0n && <p className="mt-1">No cataloged meals currently fit these rules; restrictions were not relaxed.</p>}</div>}<div className="mt-6 flex justify-between"><button type="button" className="min-h-11 rounded border px-4" disabled={step === 0 || status === "saving"} onClick={() => setStep((current) => current - 1)}>Back</button><button type="button" className="min-h-11 rounded bg-blue-700 px-4 font-medium text-white disabled:opacity-60" disabled={status === "loading" || status === "saving"} onClick={() => void next()}>{step === steps.length - 1 ? (status === "saving" ? "Applying…" : "Apply profile") : (status === "saving" ? "Saving…" : "Save and next")}</button></div></div></section>;
}
