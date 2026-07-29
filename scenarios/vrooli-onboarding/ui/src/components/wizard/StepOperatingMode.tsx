import { useQuery } from "@tanstack/react-query";
import { fetchV2Scenarios } from "../../lib/api";

export function StepOperatingMode({ selected, overrides, onAutoRestart }: { selected: Set<string>; overrides?: Record<string, { auto_restart?: boolean }>; onAutoRestart: (name: string, enabled: boolean) => void }) {
  const { data } = useQuery({ queryKey: ["v2-scenarios"], queryFn: fetchV2Scenarios });
  const scenarios = (data?.scenarios ?? []).filter((scenario) => scenario.system_required || selected.has(scenario.name));
  return <div data-testid="step-operating-mode">
    <h1 className="text-xl font-semibold sm:text-2xl">Operating mode</h1>
    <p className="mt-2 text-sm text-slate-300">Keep-running choices start with each scenario manifest’s recommendation and are saved as operator-state overrides.</p>
    <div className="mt-6 space-y-3">{scenarios.map((scenario) => {
      const autoRestart = overrides?.[scenario.name]?.auto_restart ?? scenario.auto_restart;
      return <label key={scenario.name} className="flex items-center justify-between rounded-xl border border-white/10 bg-white/5 p-4"><span><span className="block font-medium">{scenario.name}</span><span className="text-xs text-slate-300">{scenario.auto_restart ? "Recommended to keep running" : "Recommended on demand"}</span></span><input type="checkbox" checked={autoRestart} onChange={(event) => onAutoRestart(scenario.name, event.target.checked)} aria-label={`Keep ${scenario.name} running`} className="h-4 w-4 accent-emerald-500" /></label>;
    })}</div>
  </div>;
}
