import type { RuleWithState, RulesConfig } from "../lib/api";

function severityLabel(severity: string) {
  switch (severity) {
    case "error":
      return { label: "Error", className: "bg-red-500/15 text-red-200 border-red-500/30" };
    case "warn":
      return { label: "Warn", className: "bg-amber-500/15 text-amber-200 border-amber-500/30" };
    default:
      return { label: "Info", className: "bg-slate-500/15 text-slate-200 border-slate-500/30" };
  }
}

export function RuleSelector({
  rules,
  config,
  selectedRuleIds,
  saving,
  onToggleEnabled,
  onToggleSelected,
  onSelectAll,
  onSelectNone
}: {
  rules: RuleWithState[];
  config: RulesConfig;
  selectedRuleIds: Set<string>;
  saving: boolean;
  onToggleEnabled: (id: string, enabled: boolean) => void;
  onToggleSelected: (id: string) => void;
  onSelectAll: () => void;
  onSelectNone: () => void;
}) {
  return (
    <div>
      <div className="flex items-center justify-between gap-4">
        <h3 className="text-sm font-medium text-slate-200">Rules</h3>
        <div className="flex gap-2">
          <button onClick={onSelectAll} className="text-xs text-slate-400 hover:text-slate-200">
            Select all
          </button>
          <span className="text-xs text-slate-600">|</span>
          <button onClick={onSelectNone} className="text-xs text-slate-400 hover:text-slate-200">
            Select none
          </button>
        </div>
      </div>
      <div className="mt-3 grid gap-3">
        {rules.map((rule) => {
          const enabled = config.enabled_rules[rule.id] ?? false;
          const selected = selectedRuleIds.has(rule.id);
          const badge = severityLabel(rule.severity);

          return (
            <div key={rule.id} className="rounded-2xl border border-white/10 bg-white/5 p-5 backdrop-blur">
              <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <h4 className="text-base font-semibold text-slate-50">{rule.title}</h4>
                    <span className={`inline-flex items-center rounded-full border px-2 py-0.5 text-xs ${badge.className}`}>
                      {badge.label}
                    </span>
                    <span className="text-xs text-slate-400">{rule.category}</span>
                    {rule.fixable && (
                      <span className="inline-flex items-center rounded-full border border-emerald-500/30 bg-emerald-500/15 px-2 py-0.5 text-xs text-emerald-200">
                        Fixable
                      </span>
                    )}
                  </div>
                  <p className="mt-2 text-sm text-slate-300">{rule.summary}</p>
                  <details className="mt-2">
                    <summary className="cursor-pointer text-sm text-slate-200">Why this matters</summary>
                    <p className="mt-1 text-sm text-slate-300">{rule.why_important}</p>
                  </details>
                </div>
                <div className="flex items-center gap-4 md:flex-col md:items-end">
                  <label className="flex items-center gap-2 text-sm" title="Include in next run">
                    <input
                      type="checkbox"
                      checked={selected}
                      onChange={() => onToggleSelected(rule.id)}
                      className="h-4 w-4 accent-blue-400"
                    />
                    <span className="text-slate-300">Run</span>
                  </label>
                  <label className="flex items-center gap-2 text-sm" title="Persist enabled state">
                    <input
                      type="checkbox"
                      checked={enabled}
                      disabled={saving}
                      onChange={(e) => onToggleEnabled(rule.id, e.target.checked)}
                      className="h-4 w-4 accent-slate-100"
                    />
                    <span className="text-slate-200">{enabled ? "Enabled" : "Disabled"}</span>
                  </label>
                  <p className="text-xs text-slate-500">{saving ? "Saving..." : rule.id}</p>
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
