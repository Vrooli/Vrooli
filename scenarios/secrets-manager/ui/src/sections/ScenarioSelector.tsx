import { Search, Sparkles } from "lucide-react";
import { Skeleton } from "../components/ui/LoadingStates";
import { Button } from "../components/ui/button";
import type { ScenarioSummary } from "../lib/api";

interface ScenarioSelectorProps {
  scenarios: ScenarioSummary[];
  filtered: ScenarioSummary[];
  search: string;
  isLoading: boolean;
  selectedScenario: string;
  onSearchChange: (value: string) => void;
  onSelect: (scenario: string) => void;
}

export const ScenarioSelector = ({
  scenarios,
  filtered,
  search,
  isLoading,
  selectedScenario,
  onSearchChange,
  onSelect
}: ScenarioSelectorProps) => {
  return (
    <section className="rounded-3xl border border-white/10 bg-white/5 p-4">
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="text-[11px] uppercase tracking-[0.2em] text-emerald-200/80">Scenario</p>
          <p className="text-lg font-semibold text-white">Choose a scenario to prepare</p>
          <p className="text-sm text-white/60">Fast list from lifecycle, deeper readiness loads after selection.</p>
        </div>
        <Sparkles className="h-5 w-5 text-emerald-300" />
      </div>

      <div className="mt-3 flex flex-col gap-3 md:flex-row md:items-center">
        <div className="relative flex-1">
          <input
            type="text"
            value={search}
            onChange={(event) => onSearchChange(event.target.value)}
            placeholder="Search scenarios (desktop, cloud, pick a name...)"
            className="w-full rounded-2xl border border-white/10 bg-black/30 px-3 py-2 pr-10 text-sm text-white placeholder:text-white/40"
          />
          <Search className="pointer-events-none absolute right-3 top-2.5 h-4 w-4 text-white/40" />
        </div>
        <div className="text-xs text-white/50">
          {isLoading ? "Loading scenarios..." : `${scenarios.length} available`}
        </div>
      </div>

      <div className="mt-3 grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
        {isLoading ? (
          Array.from({ length: 6 }).map((_, idx) => <Skeleton key={idx} className="h-16 w-full" />)
        ) : filtered.length === 0 ? (
          <div className="col-span-full rounded-2xl border border-white/10 bg-black/30 p-3 text-sm text-white/60">
            No scenarios match “{search}”.
          </div>
        ) : (
          filtered.slice(0, 12).map((scenario) => {
            const isSelected = scenario.name === selectedScenario;
            return (
              <button
                key={scenario.name}
                onClick={() => onSelect(scenario.name)}
                className={`w-full rounded-2xl border px-3 py-3 text-left transition ${
                  isSelected
                    ? "border-emerald-400 bg-emerald-500/10 shadow-[0_10px_40px_-15px_rgba(16,185,129,0.4)]"
                    : "border-white/10 bg-black/30 hover:border-white/30"
                }`}
              >
                <div className="flex items-center justify-between text-[11px] uppercase tracking-[0.2em] text-white/60">
                  <span>{scenario.version || "v1"}</span>
                  {isSelected ? (
                    <span className="rounded-full border border-emerald-300/40 bg-emerald-500/10 px-2 py-[2px] text-[10px] font-semibold text-emerald-100">
                      Selected
                    </span>
                  ) : null}
                </div>
                <p className="mt-1 text-sm font-semibold text-white truncate">{scenario.name}</p>
                <p className="text-xs text-white/60 line-clamp-2">
                  {scenario.description || "No description provided."}
                </p>
              </button>
            );
          })
        )}
      </div>

      <div className="mt-3 flex flex-wrap gap-2 text-[11px] text-white/50">
        {selectedScenario ? (
          <Button variant="outline" size="sm" className="text-[11px] uppercase tracking-[0.2em]">
            Working on: {selectedScenario}
          </Button>
        ) : null}
      </div>
    </section>
  );
};
