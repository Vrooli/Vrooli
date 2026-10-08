import { useEffect, useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { create } from "@bufbuild/protobuf";
import { TimerSchema } from "@vrooli/proto-types/nutrition-planner/v1/cooking/cooking_pb";
import type { Recipe } from "@vrooli/proto-types/nutrition-planner/v1/recipe/recipe_pb";
import { ensureWorkspace } from "../../api/workspace";
import { getRecipeRevision } from "../../api/recipes";
import { getCookingSession, saveCookingSession, type CookingSession, type Timer } from "../../api/cooking";
import { listInventoryBatches, prepareInventoryBatch, type InventoryBatch } from "../../api/inventory";

function elapsed(timer: Timer, now: number): number {
  const started = Date.parse(timer.startedAt);
  const end = timer.pausedAt ? Date.parse(timer.pausedAt) : now;
  return Number(timer.elapsedSeconds) + Math.max(0, Math.floor((end - started) / 1000));
}

export function CookSessionPage() {
  const { sessionId = "" } = useParams();
  const [workspaceId, setWorkspaceId] = useState("");
  const [session, setSession] = useState<CookingSession>();
  const [kitchenBatch, setKitchenBatch] = useState<InventoryBatch>();
  const [recipe, setRecipe] = useState<Recipe>();
  const [yieldAmount, setYieldAmount] = useState("");
  const [timerSeconds, setTimerSeconds] = useState("10");
  const [now, setNow] = useState(Date.now());
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    let active = true;
    const tick = window.setInterval(() => setNow(Date.now()), 1000);
    void ensureWorkspace().then(async (workspace) => {
      const saved = await getCookingSession(workspace.id, sessionId);
      const [pinned, batches] = await Promise.all([getRecipeRevision(workspace.id, saved.recipeId, saved.recipeRevision), listInventoryBatches(workspace.id)]);
      if (active) { setWorkspaceId(workspace.id); setSession(saved); setRecipe(pinned); setKitchenBatch(batches.find((batch) => batch.id === `cook-session:${saved.id}`)); setState("ready"); }
    }).catch((reason: unknown) => { if (active) { setError(reason instanceof Error ? reason.message : "Unable to resume this cooking session."); setState("error"); } });
    return () => { active = false; window.clearInterval(tick); };
  }, [sessionId]);

  const method = useMemo(() => recipe?.methods.find((candidate) => candidate.id === session?.methodId), [recipe, session]);
  const step = method?.steps[session?.currentStepIndex ?? 0];
  async function persist(change: (current: CookingSession) => Partial<CookingSession>, finish = false) {
    if (!session || !workspaceId) return;
    setSaving(true);
    try {
      const patch = change(session);
      const next = await saveCookingSession({
        workspaceId, sessionId: session.id, eventId: crypto.randomUUID(), expectedVersion: session.version,
        currentStepIndex: Number(patch.currentStepIndex ?? session.currentStepIndex), completedSteps: patch.completedSteps ?? session.completedSteps,
        timers: patch.timers ?? session.timers, finish, actualYield: finish ? yieldAmount.trim() : "", yieldUnit: finish && yieldAmount.trim() ? recipe?.servingUnit || "serving" : "",
      });
      setSession(next); setError("");
    } catch (reason: unknown) { setError(reason instanceof Error ? reason.message : "Unable to save cooking progress."); }
    finally { setSaving(false); }
  }

  async function addMeasuredYieldToKitchen() {
    if (!session || !recipe || !workspaceId || !session.actualYield || !session.yieldUnit) return;
    setSaving(true);
    try {
      const prepared = await prepareInventoryBatch({
        workspaceId, eventId: `cook-yield:${session.id}`, batchId: `cook-session:${session.id}`,
        recipeId: recipe.id, recipeRevision: session.recipeRevision, yieldAmount: session.actualYield, unit: session.yieldUnit,
        requirements: recipe.ingredients.map((ingredient) => ({ itemId: ingredient.id || ingredient.name, amount: ingredient.amount, unit: ingredient.unit })),
      });
      setKitchenBatch(prepared); setError("");
    } catch (reason: unknown) { setError(reason instanceof Error ? reason.message : "Unable to update Kitchen stock."); }
    finally { setSaving(false); }
  }

  if (state === "loading") return <section><p role="status">Resuming cooking session…</p></section>;
  if (state === "error" || !session || !recipe || !method) return <section><Link to="/">Back to meals</Link><h1 className="mt-4 text-2xl font-semibold">Cooking session unavailable</h1><p role="alert" className="mt-2 text-red-800">{error || "The pinned recipe method is no longer available."}</p></section>;
  const finished = session.status === "finished";
  const stepComplete = step ? session.completedSteps.includes(step.id) : false;
  const runningTimers = session.timers.map((timer) => ({ timer, elapsed: elapsed(timer, now), remaining: Math.max(0, Number(timer.durationSeconds) - elapsed(timer, now)) }));

  return <section aria-labelledby="cook-heading" className="flex flex-col gap-5">
    <Link to={`/recipes/${recipe.id}/revisions/${recipe.revision}`} className="min-h-11 self-start rounded border bg-white px-3 py-2">Recipe map</Link>
    <header><p className="text-sm font-medium uppercase tracking-wide text-cyan-700">Cooking · revision {session.recipeRevision.toString()} · {method.name || method.id}</p><h1 id="cook-heading" className="mt-1 text-3xl font-semibold">{recipe.name}</h1><p className="mt-2 text-slate-600">Pinned scale: {session.scale}. Progress is saved to this workspace.</p></header>
    {error && <p role="alert" className="rounded border border-red-200 bg-red-50 p-3 text-red-800">{error}</p>}
    {step ? <article className="rounded-xl border bg-white p-5 sm:p-7"><p className="text-sm text-slate-600">Step {session.currentStepIndex + 1} of {method.steps.length}</p><h2 className="mt-2 text-2xl font-semibold">{step.instruction || "Instruction not entered."}</h2><p className="mt-3 text-sm text-slate-600">Navigation does not mark a step complete. Use the explicit button when the step is done.</p><div className="mt-5 flex flex-wrap gap-3"><button type="button" className="min-h-11 rounded border px-4" disabled={session.currentStepIndex <= 0 || saving} onClick={() => void persist(() => ({ currentStepIndex: Math.max(0, session.currentStepIndex - 1) }))}>Previous step</button><button type="button" className="min-h-11 rounded border px-4" disabled={stepComplete || saving || finished} onClick={() => void persist(() => ({ completedSteps: [...session.completedSteps, step.id] }))}>Mark step done</button><button type="button" className="min-h-11 rounded border px-4" disabled={session.currentStepIndex >= method.steps.length - 1 || saving} onClick={() => void persist(() => ({ currentStepIndex: Math.min(method.steps.length - 1, session.currentStepIndex + 1) }))}>Next step</button>{stepComplete && <span role="status" className="self-center text-emerald-800">Step completed</span>}</div></article> : <article className="rounded-xl border bg-white p-5"><h2 className="text-xl font-semibold">No steps recorded</h2><p className="mt-2 text-slate-600">This saved method has no instructions. You can still finish the session.</p></article>}
    <article className="rounded-xl border bg-white p-5"><h2 className="text-lg font-semibold">Timers</h2><form className="mt-3 flex flex-wrap items-end gap-3" onSubmit={(event) => { event.preventDefault(); const minutes=Number(timerSeconds); if (!Number.isFinite(minutes)||minutes<=0||!step) return; const timer=create(TimerSchema,{id:crypto.randomUUID(),stepId:step.id,durationSeconds:BigInt(Math.floor(minutes*60)),startedAt:new Date().toISOString(),pausedAt:"",elapsedSeconds:0n}); void persist((current)=>({timers:[...current.timers,timer]})); }}><label className="grid gap-1 text-sm">Minutes<input aria-label="Timer minutes" className="min-h-11 rounded border px-3" inputMode="decimal" value={timerSeconds} onChange={(event)=>setTimerSeconds(event.target.value)}/></label><button className="min-h-11 rounded border px-4" disabled={!step||saving||finished}>Start timer for this step</button></form>{runningTimers.length===0&&<p className="mt-3 text-slate-600">No timers running.</p>}<ul className="mt-3 space-y-2">{runningTimers.map(({timer,elapsed:seconds,remaining})=><li key={timer.id} className="flex flex-wrap items-center justify-between gap-3 rounded bg-slate-50 p-3"><span>{timer.stepId}: {Math.floor(remaining/60)}:{String(remaining%60).padStart(2,"0")} {remaining===0&&"· elapsed"}</span><button type="button" className="min-h-11 rounded border bg-white px-3" disabled={saving||finished} onClick={()=>{const stamp=new Date().toISOString();const nextTimer=timer.pausedAt?{...timer,startedAt:stamp,pausedAt:""}:{...timer,elapsedSeconds:BigInt(seconds),startedAt:stamp,pausedAt:stamp};void persist((current)=>({timers:current.timers.map((item)=>item.id===timer.id?nextTimer:item)}));}}>{timer.pausedAt?"Resume":"Pause"}</button></li>)}</ul><p className="mt-2 text-sm text-slate-500">Timer timestamps and elapsed time are stored, so a reload restores each timer.</p></article>
    <article className="rounded-xl border bg-white p-5"><h2 className="text-lg font-semibold">Finish cooking</h2>{finished?<><p role="status" className="mt-2 text-emerald-800">Cooking finished{session.actualYield?`; actual yield ${session.actualYield} ${session.yieldUnit} recorded`:"; no stock change recorded"}.</p>{session.actualYield && (kitchenBatch ? <p className="mt-3 rounded bg-emerald-50 p-3 text-emerald-900">Kitchen stock updated: {kitchenBatch.availableAmount} {kitchenBatch.unit} available from the measured {kitchenBatch.yieldAmount} {kitchenBatch.unit} yield.</p> : <><p className="mt-2 text-slate-600">Stock changes only after you confirm this measured yield. Confirming records the pinned recipe’s entered ingredient quantities once.</p><button type="button" className="mt-4 min-h-11 rounded bg-emerald-700 px-4 text-white" disabled={saving} onClick={()=>void addMeasuredYieldToKitchen()}>{saving?"Updating Kitchen…":"Add measured yield to Kitchen stock"}</button></>)}</>:<><p className="mt-2 text-slate-600">Finishing does not change Kitchen stock. Enter an actual yield only if you measured it; stock can be updated separately.</p><label className="mt-3 grid max-w-sm gap-1 text-sm">Actual yield (optional)<input aria-label="Actual yield" className="min-h-11 rounded border px-3" inputMode="decimal" value={yieldAmount} onChange={(event)=>setYieldAmount(event.target.value)}/></label><button type="button" className="mt-4 min-h-11 rounded bg-blue-700 px-4 text-white" disabled={saving} onClick={()=>void persist(()=>({}),true)}>{saving?"Saving…":"Finish cooking"}</button></>}</article>
  </section>;
}
