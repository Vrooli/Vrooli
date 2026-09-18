import { useEffect, useState } from "react";
import type { Recipe } from "@vrooli/proto-types/nutrition-planner/v1/recipe/recipe_pb";

type Tab = "map" | "read" | "cook";

export function RecipeViewer({ recipe, onClose }: { recipe: Recipe; onClose: () => void }) {
  const [tab, setTab] = useState<Tab>("map");
  const [step, setStep] = useState(0);
  const [deadline, setDeadline] = useState<number | undefined>();
  const method = recipe.methods[0];
  const steps = method?.steps ?? [];
  const activeStep = steps[step];
  const [remaining, setRemaining] = useState(0);
  useEffect(() => {
    const stored = window.localStorage.getItem(`daily.recipe-timer.${recipe.id}`);
    const parsed = stored ? Number(stored) : 0;
    if (Number.isFinite(parsed) && parsed > Date.now()) { setDeadline(parsed); setRemaining(parsed - Date.now()); }
  }, [recipe.id]);
  useEffect(() => { if (deadline === undefined) return undefined; window.localStorage.setItem(`daily.recipe-timer.${recipe.id}`, String(deadline)); const timer = window.setInterval(() => { const next = Math.max(0, deadline - Date.now()); setRemaining(next); if (next === 0) window.localStorage.removeItem(`daily.recipe-timer.${recipe.id}`); }, 250); return () => window.clearInterval(timer); }, [deadline, recipe.id]);
  return <section aria-labelledby="recipe-viewer-heading" className="rounded-lg border border-blue-200 bg-blue-50 p-5"><div className="flex items-start justify-between gap-3"><div><p className="text-sm font-medium uppercase tracking-wide text-blue-700">Recipe revision {recipe.revision.toString()}</p><h2 id="recipe-viewer-heading" className="text-xl font-semibold text-slate-900">{recipe.name}</h2></div><button type="button" className="min-h-11 rounded border border-slate-300 bg-white px-3" onClick={onClose}>Close</button></div><div role="tablist" aria-label="Recipe views" className="mt-4 flex gap-2">{(["map", "read", "cook"] as const).map((value) => <button key={value} type="button" role="tab" aria-selected={tab === value} className={`min-h-11 rounded px-3 ${tab === value ? "bg-blue-700 text-white" : "bg-white text-slate-700"}`} onClick={() => setTab(value)}>{value === "map" ? "Recipe map" : value === "read" ? "Read" : "Cook"}</button>)}</div>{tab === "map" && <div className="mt-4"><p className="text-sm text-slate-600">Ingredients and actions are linked by revision; no method graph is available until entered.</p>{steps.length > 0 && <ol className="mt-3 space-y-2">{steps.map((item) => <li key={item.id} className="rounded bg-white p-3"><strong>{item.id}</strong> {item.instruction}</li>)}</ol>}</div>}{tab === "read" && <div className="mt-4 whitespace-pre-wrap rounded bg-white p-4 text-slate-700">{recipe.originalText || recipe.notes || "No preparation text has been entered yet."}</div>}{tab === "cook" && <div className="mt-4 rounded bg-white p-4"><p className="text-sm text-slate-600">Step {steps.length === 0 ? 0 : step + 1} of {steps.length}</p><p className="mt-2 text-slate-900">{activeStep?.instruction || "Add method steps to cook this recipe."}</p><div className="mt-4 flex flex-wrap gap-2"><button type="button" className="min-h-11 rounded border px-3" disabled={step === 0} onClick={() => setStep((current) => Math.max(0, current - 1))}>Previous</button><button type="button" className="min-h-11 rounded bg-blue-700 px-3 text-white" disabled={steps.length === 0 || step >= steps.length - 1} onClick={() => setStep((current) => Math.min(steps.length - 1, current + 1))}>Next</button><button type="button" className="min-h-11 rounded border px-3" onClick={() => { const next = Date.now() + 60000; setDeadline(next); setRemaining(60000); }}>Start 1-minute timer</button></div>{deadline !== undefined && <p role="status" className="mt-3 text-sm text-slate-600">Timer: {Math.ceil(remaining / 1000)} seconds remaining.</p>}</div>}</section>;
}
