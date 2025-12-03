import { useState } from "react";
import { Search } from "lucide-react";
import { useRequirements } from "../../hooks/useRequirements";
import { SyncStatusBanner } from "./SyncStatusBanner";
import { CoverageStats } from "./CoverageStats";
import { RequirementsTree } from "./RequirementsTree";
import { cn } from "../../lib/utils";
import { selectors } from "../../consts/selectors";

interface RequirementsPanelProps {
  scenarioName: string;
}

type FilterKey = "all" | "passed" | "failed" | "not_run";

const FILTERS: Array<{ key: FilterKey; label: string }> = [
  { key: "all", label: "All" },
  { key: "passed", label: "Passed" },
  { key: "failed", label: "Failed" },
  { key: "not_run", label: "Not Run" }
];

export function RequirementsPanel({ scenarioName }: RequirementsPanelProps) {
  const [filter, setFilter] = useState<FilterKey>("all");
  const [searchQuery, setSearchQuery] = useState("");

  const {
    isLoading,
    isError,
    coverage,
    syncStatus,
    modules,
    sync,
    isSyncing
  } = useRequirements(scenarioName);

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="h-12 animate-pulse rounded-xl bg-white/5" />
        <div className="grid grid-cols-4 gap-3">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="h-20 animate-pulse rounded-xl bg-white/5" />
          ))}
        </div>
        <div className="h-64 animate-pulse rounded-xl bg-white/5" />
      </div>
    );
  }

  if (isError) {
    return (
      <div className="rounded-xl border border-red-500/20 bg-red-500/5 p-6 text-center">
        <p className="text-red-300">Failed to load requirements data</p>
        <p className="mt-2 text-sm text-slate-400">
          Make sure the scenario has been set up correctly and tests have been run.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4" data-testid={selectors.requirements.panel}>
      {/* Sync Status Banner */}
      <SyncStatusBanner
        syncStatus={syncStatus}
        onSync={(options) => sync(options)}
        isSyncing={isSyncing}
      />

      {/* Coverage Stats */}
      <CoverageStats coverage={coverage} />

      {/* Filter and Search Bar */}
      <div className="flex flex-wrap items-center gap-3">
        <div className="flex rounded-lg border border-white/10 bg-black/30 p-1">
          {FILTERS.map((f) => (
            <button
              key={f.key}
              type="button"
              onClick={() => setFilter(f.key)}
              className={cn(
                "rounded-md px-3 py-1 text-sm transition",
                filter === f.key
                  ? "bg-white/10 text-white"
                  : "text-slate-400 hover:text-white"
              )}
              data-testid={selectors.requirements[`filter${f.key.charAt(0).toUpperCase() + f.key.slice(1).replace("_", "")}` as keyof typeof selectors.requirements]}
            >
              {f.label}
            </button>
          ))}
        </div>

        <div className="relative flex-1 min-w-[200px]">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-500" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search requirements..."
            className="w-full rounded-lg border border-white/10 bg-black/30 py-2 pl-9 pr-3 text-sm placeholder-slate-500 focus:border-white/20 focus:outline-none"
            data-testid={selectors.requirements.searchInput}
          />
        </div>
      </div>

      {/* Requirements Tree */}
      <RequirementsTree
        modules={modules}
        filter={filter}
        searchQuery={searchQuery}
      />
    </div>
  );
}
