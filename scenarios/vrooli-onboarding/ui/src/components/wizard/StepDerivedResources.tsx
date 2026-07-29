import { useQuery } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";
import { fetchV2Scenarios } from "../../lib/api";

export function StepDerivedResources({ selected }: { selected: Set<string> }) {
  const { data, isLoading, error } = useQuery({ queryKey: ["v2-scenarios"], queryFn: fetchV2Scenarios });
  const resources = Array.from(new Set((data?.scenarios ?? []).filter((scenario) => scenario.system_required || selected.has(scenario.name)).flatMap((scenario) => scenario.resources))).sort();
  return <div data-testid="step-derived-resources">
    <h1 className="text-xl font-semibold sm:text-2xl">Derived resources</h1>
    <p className="mt-2 text-sm text-slate-300">Resources are derived from selected scenario manifests. Required resources cannot be independently disabled here.</p>
    {isLoading && <p className="mt-6 flex items-center gap-2 text-slate-300" role="status"><Loader2 className="h-4 w-4 animate-spin" />Loading derived resources…</p>}
    {error && <p className="mt-6 text-red-400" role="alert">Unable to derive resources from scenario manifests.</p>}
    {!isLoading && !error && <ul className="mt-6 grid gap-2 sm:grid-cols-2">{resources.map((resource) => <li key={resource} className="rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-3 py-2 text-sm text-emerald-200">{resource}</li>)}</ul>}
    {!isLoading && !error && resources.length === 0 && <p className="mt-6 text-slate-300">The selected scenarios do not require local resources.</p>}
  </div>;
}
