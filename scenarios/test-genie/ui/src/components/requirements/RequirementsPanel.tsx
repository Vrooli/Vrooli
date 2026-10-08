import { useState } from "react";
import { Search } from "lucide-react";
import { useRequirements } from "../../hooks/useRequirements";
import { SyncStatusBanner } from "./SyncStatusBanner";
import { CoverageStats } from "./CoverageStats";
import { RequirementsHelpSection } from "./RequirementsHelpSection";
import { RequirementsTree } from "./RequirementsTree";
import { cn } from "../../lib/utils";
import { selectors } from "../../consts/selectors";

interface RequirementsPanelProps { scenarioName: string }
type FilterKey = "all" | "passed" | "failed" | "not_run";
const filters: Array<{ key: FilterKey; label: string }> = [{ key: "all", label: "All" }, { key: "passed", label: "Passed" }, { key: "failed", label: "Failed" }, { key: "not_run", label: "Not Run" }];

export function RequirementsPanel({ scenarioName }: RequirementsPanelProps) {
  const [filter, setFilter] = useState<FilterKey>("all");
  const [searchQuery, setSearchQuery] = useState("");
  const { isLoading, isError, coverage, syncStatus, modules, sync, isSyncing, lastSyncSuccess } = useRequirements(scenarioName);
  if (isLoading) return <div className="h-64 animate-pulse rounded-xl bg-white/5" />;
  if (isError) return <div className="rounded-xl border border-red-500/20 bg-red-500/5 p-6 text-center text-red-300">Failed to load requirement evidence.</div>;
  return <div className="space-y-4" data-testid={selectors.requirements.panel}>
    <SyncStatusBanner syncStatus={syncStatus} onSync={() => sync({})} isSyncing={isSyncing} lastSyncSuccess={lastSyncSuccess} />
    <CoverageStats coverage={coverage} />
    <RequirementsHelpSection />
    <p className="rounded-lg border border-cyan-400/20 bg-cyan-400/5 p-3 text-sm text-slate-300">Requirement status and validations are evidence. Select failed requirements together with run findings from the scenario overview’s unified remediation builder.</p>
    <div className="flex flex-wrap items-center gap-3"><div className="flex rounded-lg border border-white/10 bg-black/30 p-1">{filters.map((item) => <button key={item.key} type="button" onClick={() => setFilter(item.key)} className={cn("rounded-md px-3 py-1 text-sm transition", filter === item.key ? "bg-white/10 text-white" : "text-slate-400 hover:text-white")}>{item.label}</button>)}</div><div className="relative min-w-[200px] flex-1"><Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-500"/><input value={searchQuery} onChange={(event) => setSearchQuery(event.target.value)} placeholder="Search requirements..." className="w-full rounded-lg border border-white/10 bg-black/30 py-2 pl-9 pr-3 text-sm placeholder-slate-500 focus:border-white/20 focus:outline-none" data-testid={selectors.requirements.searchInput}/></div></div>
    <RequirementsTree modules={modules} filter={filter} searchQuery={searchQuery} />
  </div>;
}
