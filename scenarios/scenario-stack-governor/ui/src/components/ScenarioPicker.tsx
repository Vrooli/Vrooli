import { useState } from "react";

export function ScenarioPicker({
  scenarios,
  selectedScenarios,
  allSelected,
  onToggleAll,
  onToggleScenario
}: {
  scenarios: string[];
  selectedScenarios: string[];
  allSelected: boolean;
  onToggleAll: () => void;
  onToggleScenario: (name: string) => void;
}) {
  const [filter, setFilter] = useState("");
  const selectedSet = new Set(selectedScenarios);
  const filtered = filter
    ? scenarios.filter((s) => s.toLowerCase().includes(filter.toLowerCase()))
    : scenarios;

  return (
    <div className="rounded-2xl border border-white/10 bg-white/5 p-5 backdrop-blur">
      <h3 className="text-sm font-medium text-slate-200">Scenarios</h3>
      <label className="mt-3 flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          checked={allSelected}
          onChange={onToggleAll}
          className="h-4 w-4 accent-slate-100"
        />
        <span className="text-slate-200">All scenarios</span>
      </label>

      {!allSelected && (
        <div className="mt-3">
          <input
            type="text"
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            placeholder="Filter scenarios..."
            className="w-full rounded-lg border border-white/10 bg-black/20 px-3 py-1.5 text-sm text-slate-200 placeholder-slate-500 outline-none focus:border-white/20"
          />
          <div className="mt-2 max-h-48 space-y-1 overflow-y-auto">
            {filtered.map((name) => (
              <label key={name} className="flex items-center gap-2 rounded px-1 py-0.5 text-sm hover:bg-white/5">
                <input
                  type="checkbox"
                  checked={selectedSet.has(name)}
                  onChange={() => onToggleScenario(name)}
                  className="h-3.5 w-3.5 accent-slate-100"
                />
                <span className="text-slate-300 truncate">{name}</span>
              </label>
            ))}
            {filtered.length === 0 && (
              <p className="text-xs text-slate-500">No scenarios match filter.</p>
            )}
          </div>
          <p className="mt-1 text-xs text-slate-500">
            {selectedScenarios.length} of {scenarios.length} selected
          </p>
        </div>
      )}
    </div>
  );
}
