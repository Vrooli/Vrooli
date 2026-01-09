import { useState, useMemo } from "react";
import { Search, ChevronUp, ChevronDown } from "lucide-react";
import { Button } from "../ui/button";
import { StatusPill } from "../cards/StatusPill";
import { selectors } from "../../consts/selectors";
import { formatRelative } from "../../lib/formatters";
import type { ScenarioDirectoryEntry } from "../../hooks/useScenarios";

type SortField = "name" | "status" | "pending" | "lastRun" | "lastFailure";
type SortDirection = "asc" | "desc";

interface ScenarioTableProps {
  scenarios: ScenarioDirectoryEntry[];
  onScenarioClick: (scenarioName: string) => void;
  onQueueClick: (scenario: ScenarioDirectoryEntry) => void;
  onRunClick: (scenario: ScenarioDirectoryEntry) => void;
  isLoading?: boolean;
}

export function ScenarioTable({
  scenarios,
  onScenarioClick,
  onQueueClick,
  onRunClick,
  isLoading
}: ScenarioTableProps) {
  const [search, setSearch] = useState("");
  const [sortField, setSortField] = useState<SortField>("name");
  const [sortDirection, setSortDirection] = useState<SortDirection>("asc");

  const filteredAndSorted = useMemo(() => {
    let result = scenarios;

    // Filter by search
    const trimmed = search.trim().toLowerCase();
    if (trimmed) {
      result = result.filter((s) => s.scenarioName.toLowerCase().includes(trimmed));
    }

    // Sort
    result = [...result].sort((a, b) => {
      let comparison = 0;
      switch (sortField) {
        case "name":
          comparison = a.scenarioName.localeCompare(b.scenarioName);
          break;
        case "status":
          comparison = (a.scenarioStatus ?? "").localeCompare(b.scenarioStatus ?? "");
          break;
        case "pending":
          comparison = a.pendingRequests - b.pendingRequests;
          break;
        case "lastRun":
          comparison = (a.lastExecutionAt ?? "").localeCompare(b.lastExecutionAt ?? "");
          break;
        case "lastFailure":
          comparison = (a.lastFailureAt ?? "").localeCompare(b.lastFailureAt ?? "");
          break;
      }
      return sortDirection === "desc" ? -comparison : comparison;
    });

    return result;
  }, [scenarios, search, sortField, sortDirection]);

  const toggleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDirection((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortField(field);
      setSortDirection("asc");
    }
  };

  const SortIcon = ({ field }: { field: SortField }) => {
    if (sortField !== field) return null;
    return sortDirection === "asc" ? (
      <ChevronUp className="h-4 w-4" />
    ) : (
      <ChevronDown className="h-4 w-4" />
    );
  };

  return (
    <div data-testid={selectors.runs.scenarioTable}>
      <div className="mb-4 flex items-center gap-4">
        <div className="relative flex-1">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-500" />
          <input
            className="w-full rounded-full border border-white/10 bg-black/40 pl-9 pr-4 py-2 text-sm text-white focus:outline-none focus:ring-2 focus:ring-cyan-400"
            placeholder="Search scenarios..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        {search && (
          <Button variant="outline" size="sm" onClick={() => setSearch("")}>
            Clear
          </Button>
        )}
      </div>

      <div className="overflow-x-auto rounded-2xl border border-white/5">
        <table className="w-full min-w-[700px] text-left text-sm text-slate-200">
          <thead className="bg-white/5 text-xs uppercase tracking-wide text-slate-400">
            <tr>
              <th
                className="px-4 py-3 cursor-pointer hover:text-white"
                onClick={() => toggleSort("name")}
              >
                <span className="flex items-center gap-1">
                  Name <SortIcon field="name" />
                </span>
              </th>
              <th
                className="px-4 py-3 cursor-pointer hover:text-white"
                onClick={() => toggleSort("status")}
              >
                <span className="flex items-center gap-1">
                  Status <SortIcon field="status" />
                </span>
              </th>
              <th
                className="px-4 py-3 cursor-pointer hover:text-white"
                onClick={() => toggleSort("pending")}
              >
                <span className="flex items-center gap-1">
                  Pending <SortIcon field="pending" />
                </span>
              </th>
              <th
                className="px-4 py-3 cursor-pointer hover:text-white"
                onClick={() => toggleSort("lastRun")}
              >
                <span className="flex items-center gap-1">
                  Last Run <SortIcon field="lastRun" />
                </span>
              </th>
              <th
                className="px-4 py-3 cursor-pointer hover:text-white"
                onClick={() => toggleSort("lastFailure")}
              >
                <span className="flex items-center gap-1">
                  Last Failure <SortIcon field="lastFailure" />
                </span>
              </th>
              <th className="px-4 py-3 text-right">Actions</th>
            </tr>
          </thead>
          <tbody>
            {isLoading && filteredAndSorted.length === 0 && (
              <tr>
                <td className="px-4 py-6 text-center text-slate-400" colSpan={6}>
                  Loading scenarios...
                </td>
              </tr>
            )}
            {!isLoading && filteredAndSorted.length === 0 && (
              <tr>
                <td className="px-4 py-6 text-center text-slate-400" colSpan={6}>
                  {search
                    ? `No scenarios match "${search}"`
                    : "No scenarios have been tracked yet. Run tests to populate this table."}
                </td>
              </tr>
            )}
            {filteredAndSorted.map((scenario) => (
              <tr
                key={scenario.scenarioName}
                className="border-t border-white/5 cursor-pointer hover:bg-white/5"
                onClick={() => onScenarioClick(scenario.scenarioName)}
              >
                <td className="px-4 py-3">
                  <div className="font-semibold text-white">{scenario.scenarioName}</div>
                  {scenario.scenarioDescription && (
                    <p className="text-xs text-slate-400 line-clamp-1">
                      {scenario.scenarioDescription}
                    </p>
                  )}
                </td>
                <td className="px-4 py-3">
                  <StatusPill status={scenario.scenarioStatus ?? "unknown"} />
                </td>
                <td className="px-4 py-3">
                  {scenario.pendingRequests > 0 ? (
                    <span className="rounded-full bg-amber-400/20 px-3 py-1 text-xs text-amber-100">
                      {scenario.pendingRequests} pending
                    </span>
                  ) : (
                    <span className="text-xs text-slate-400">None</span>
                  )}
                </td>
                <td className="px-4 py-3 text-sm text-slate-300">
                  {scenario.lastExecutionAt
                    ? `${scenario.lastExecutionSuccess ? "Passed" : "Failed"} ${formatRelative(scenario.lastExecutionAt)}`
                    : "No runs"}
                </td>
                <td className="px-4 py-3 text-sm text-slate-300">
                  {scenario.lastFailureAt ? formatRelative(scenario.lastFailureAt) : "—"}
                </td>
                <td className="px-4 py-3">
                  <div
                    className="flex justify-end gap-2"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => onQueueClick(scenario)}
                    >
                      Queue
                    </Button>
                    <Button
                      size="sm"
                      onClick={() => onRunClick(scenario)}
                      disabled={!scenario.lastExecutionAt}
                    >
                      Run
                    </Button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
