import { useQuery } from "@tanstack/react-query";
import { fetchV2Scenarios } from "../../lib/api";

export function StepOperatingMode({ selected, overrides, onAutoRestart }: { selected: Set<string>; overrides?: Record<string, { auto_restart?: boolean }>; onAutoRestart: (name: string, enabled: boolean) => void }) {
  const { data } = useQuery({ queryKey: ["v2-scenarios"], queryFn: fetchV2Scenarios });
  const scenarios = (data?.scenarios ?? []).filter((scenario) => scenario.system_required || selected.has(scenario.name));
  return <div data-testid="step-operating-mode">
    <h1 className="text-xl font-semibold sm:text-2xl">Operating mode</h1>
    <p className="mt-2 text-sm text-muted">Keep-running choices start with each scenario manifest’s recommendation and are saved as operator-state overrides.</p>
    <div className="mt-6 space-y-3">{scenarios.length === 0 && <div data-testid="operating-mode-row" role="group" className="rounded-xl border border-muted bg-surface-muted p-4"><span data-testid="recommendation-note" role="note" className="text-sm text-muted">Loading operating-mode recommendations…</span><input data-testid="keep-running-toggle" type="checkbox" disabled aria-label="Keep selected scenarios running" className="sr-only" /></div>}{scenarios.map((scenario) => {
      const autoRestart = overrides?.[scenario.name]?.auto_restart ?? scenario.auto_restart;
      const overridden = overrides?.[scenario.name]?.auto_restart !== undefined;
      return <label key={scenario.name} data-testid="operating-mode-row" role="group" className="flex items-center justify-between rounded-xl border border-muted bg-surface-muted p-4"><span><span className="block font-medium">{scenario.name}</span><span className="text-xs text-muted" data-testid="recommendation-note" role="note">{scenario.auto_restart ? "Recommended to keep running" : "Recommended on demand"}</span>{overridden && <span className="mt-1 block text-xs text-primary-soft" data-testid="override-indicator" role="note">Operator override saved</span>}</span><input type="checkbox" checked={autoRestart} onChange={(event) => onAutoRestart(scenario.name, event.target.checked)} aria-label={`Keep ${scenario.name} running`} data-testid="keep-running-toggle" className="min-h-11 min-w-11 accent-emerald-500" /></label>;
    })}</div>
  </div>;
}
