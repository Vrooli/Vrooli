/**
 * RecordsList — filterable list of records.
 */

import { useMemo } from "react";
import type { RecordItem, RecordKind } from "../../types";
import { ALL_RECORD_KINDS } from "../../types";
import { RecordCard } from "./RecordCard";

interface RecordsListProps {
  records: RecordItem[];
  kindFilter?: RecordKind | "";
  scenarioFilter?: string;
  onKindFilterChange?: (kind: RecordKind | "") => void;
  onScenarioFilterChange?: (scenario: string) => void;
  includeStubs?: boolean;
  onIncludeStubsChange?: (value: boolean) => void;
}

export function RecordsList(props: RecordsListProps) {
  const {
    records,
    kindFilter = "",
    scenarioFilter = "",
    onKindFilterChange,
    onScenarioFilterChange,
    includeStubs = false,
    onIncludeStubsChange,
  } = props;

  const scenarios = useMemo(() => {
    const set = new Set<string>();
    for (const r of records) {
      if (r.scenario) set.add(r.scenario);
    }
    return [...set].sort();
  }, [records]);

  const filtered = useMemo(() => {
    return records.filter((r) => {
      if (kindFilter && r.kind !== kindFilter) return false;
      if (scenarioFilter && r.scenario !== scenarioFilter) return false;
      if (!includeStubs && r.stub) return false;
      return true;
    });
  }, [records, kindFilter, scenarioFilter, includeStubs]);

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-3 rounded border border-slate-800 bg-slate-900/40 p-3">
        <label className="text-xs text-slate-400">
          Kind{" "}
          <select
            className="ml-1 rounded border border-slate-700 bg-slate-900 px-2 py-1 text-sm text-slate-200"
            value={kindFilter}
            onChange={(e) => onKindFilterChange?.((e.target.value as RecordKind) || "")}
            data-testid="records-filter-kind"
          >
            <option value="">all</option>
            {ALL_RECORD_KINDS.map((k) => (
              <option key={k} value={k}>
                {k}
              </option>
            ))}
          </select>
        </label>
        <label className="text-xs text-slate-400">
          Scenario{" "}
          <select
            className="ml-1 rounded border border-slate-700 bg-slate-900 px-2 py-1 text-sm text-slate-200"
            value={scenarioFilter}
            onChange={(e) => onScenarioFilterChange?.(e.target.value)}
            data-testid="records-filter-scenario"
          >
            <option value="">all</option>
            {scenarios.map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </select>
        </label>
        <label className="flex items-center gap-2 text-xs text-slate-400">
          <input
            type="checkbox"
            checked={includeStubs}
            onChange={(e) => onIncludeStubsChange?.(e.target.checked)}
            data-testid="records-filter-include-stubs"
          />
          Include stubs
        </label>
        <span className="ml-auto text-xs text-slate-500">{filtered.length} record(s)</span>
      </div>

      {filtered.length === 0 ? (
        <div className="rounded border border-dashed border-slate-700 bg-slate-900/30 p-6 text-center text-sm text-slate-400">
          No records match the current filters.
        </div>
      ) : (
        <ul className="flex flex-col gap-2" data-testid="records-list">
          {filtered.map((r) => (
            <li key={r.id}>
              <RecordCard record={r} />
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
