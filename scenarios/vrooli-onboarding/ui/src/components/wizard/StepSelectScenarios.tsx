import { useQuery } from "@tanstack/react-query";
import { Check, Loader2, LockKeyhole } from "lucide-react";
import { fetchV2Scenarios } from "../../lib/api";
import { cn } from "../../lib/utils";

interface Props { selected: Set<string>; onToggle: (name: string) => void; }

// StepSelectScenarios is the V2 entry point. Resources are visible only as
// manifest-derived consequences of a scenario choice, never as the primary
// configuration authority.
export function StepSelectScenarios({ selected, onToggle }: Props) {
  const { data, isLoading, error } = useQuery({ queryKey: ["v2-scenarios"], queryFn: fetchV2Scenarios });
  const scenarios = data?.scenarios ?? [];
  return <div data-testid="step-select-scenarios">
    <h1 className="text-xl font-semibold sm:text-2xl">Choose capabilities</h1>
    <p className="mt-2 text-sm text-slate-300 sm:text-base">Choose the scenarios this installation should run. Required system capabilities stay enabled.</p>
    {isLoading && <div className="flex items-center gap-2 py-16 text-slate-300" role="status"><Loader2 className="h-5 w-5 animate-spin" aria-hidden="true" /> Loading scenarios…</div>}
    {error && <p className="py-10 text-red-400" role="alert">Unable to load scenario manifests. Check the control plane and try again.</p>}
    {!isLoading && !error && <div className="mt-6 grid gap-3 sm:grid-cols-2">{scenarios.map((scenario) => {
      const checked = scenario.system_required || selected.has(scenario.name);
      return <button key={scenario.name} type="button" disabled={scenario.system_required} onClick={() => onToggle(scenario.name)} data-testid={`scenario-card-${scenario.name}`} aria-pressed={checked} className={cn("rounded-xl border p-4 text-left transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/50", checked ? "border-emerald-500/50 bg-emerald-500/10" : "border-white/10 bg-white/5 hover:bg-white/10", scenario.system_required && "cursor-not-allowed opacity-90")}>
        <div className="flex items-start gap-3"><span className={cn("mt-0.5 flex h-5 w-5 items-center justify-center rounded border", checked ? "border-emerald-500 bg-emerald-500" : "border-white/30")}>{checked && <Check className="h-3 w-3 text-white" aria-hidden="true" />}</span><span className="min-w-0 flex-1"><span className="flex items-center gap-2 font-medium">{scenario.name}{scenario.system_required && <LockKeyhole className="h-3.5 w-3.5 text-amber-300" aria-label="Required system scenario" />}</span>{scenario.description && <span className="mt-1 block text-xs text-slate-300">{scenario.description}</span>}<span className="mt-2 block text-xs text-slate-300">Uses: {scenario.resources.length ? scenario.resources.join(", ") : "no local resources"}</span></span></div>
      </button>;
    })}</div>}
  </div>;
}
