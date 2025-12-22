import { Plus, Trash2, X } from "lucide-react";
import { Button } from "./ui/button";
import type { GroupingRule } from "./FileList";

interface GroupingSettingsModalProps {
  isOpen: boolean;
  repoDir?: string;
  groupingEnabled: boolean;
  onToggleGrouping: () => void;
  rules: GroupingRule[];
  onChangeRules: (rules: GroupingRule[]) => void;
  onClose: () => void;
}

function createRule(): GroupingRule {
  return {
    id: `group-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    label: "",
    prefix: "",
    mode: "prefix"
  };
}

export function GroupingSettingsModal({
  isOpen,
  repoDir,
  groupingEnabled,
  onToggleGrouping,
  rules,
  onChangeRules,
  onClose
}: GroupingSettingsModalProps) {
  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 px-4"
      role="dialog"
      aria-modal="true"
      aria-label="Grouping settings"
    >
      <div className="w-full max-w-2xl rounded-xl border border-slate-800 bg-slate-950 shadow-xl">
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-3">
          <div>
            <h2 className="text-sm font-semibold text-slate-100">Grouping settings</h2>
            {repoDir && (
              <p className="text-[11px] text-slate-500 mt-1">Repo: {repoDir}</p>
            )}
          </div>
          <button
            type="button"
            className="h-8 w-8 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800/60"
            onClick={onClose}
            aria-label="Close settings"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="px-4 py-4 space-y-4">
          <div className="flex items-center justify-between gap-2 rounded-lg border border-slate-800/70 bg-slate-900/40 px-3 py-2">
            <div>
              <div className="text-xs font-semibold text-slate-200">Enable grouping</div>
              <div className="text-[11px] text-slate-500">
                Group changes by path prefixes (first match wins).
              </div>
            </div>
            <button
              type="button"
              onClick={onToggleGrouping}
              className={`h-7 px-3 rounded-full border text-xs ${
                groupingEnabled
                  ? "border-emerald-400/40 text-emerald-200 bg-emerald-900/20"
                  : "border-slate-700 text-slate-300 hover:bg-slate-800/50"
              }`}
            >
              {groupingEnabled ? "On" : "Off"}
            </button>
          </div>

          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-xs font-semibold text-slate-200">Grouping rules</h3>
                <p className="text-[11px] text-slate-500">
                  Prefix examples: <span className="font-mono">scenarios/</span>,{" "}
                  <span className="font-mono">resources/</span>
                </p>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => onChangeRules([...rules, createRule()])}
                className="h-7 px-2"
              >
                <Plus className="h-3 w-3 mr-1" />
                Add group
              </Button>
            </div>

            {rules.length === 0 ? (
              <div className="rounded-lg border border-slate-800/70 bg-slate-900/40 px-3 py-4 text-xs text-slate-500">
                No grouping rules yet. Add one to start grouping changes.
              </div>
            ) : (
              <div className="space-y-2">
                {rules.map((rule, index) => (
                  <div
                    key={rule.id}
                    className="flex flex-wrap items-center gap-2 rounded-lg border border-slate-800/60 bg-slate-950/50 px-3 py-2"
                  >
                    <div className="text-[11px] text-slate-500 w-6">#{index + 1}</div>
                    <input
                      type="text"
                      value={rule.label}
                      onChange={(event) => {
                        const nextRules = rules.map((item) =>
                          item.id === rule.id ? { ...item, label: event.target.value } : item
                        );
                        onChangeRules(nextRules);
                      }}
                      placeholder="Label"
                      className="flex-1 min-w-[140px] rounded-md border border-slate-800 bg-slate-950/60 px-2 py-1 text-xs text-slate-200 placeholder:text-slate-600 focus:outline-none focus:ring-2 focus:ring-blue-500/40"
                    />
                    <input
                      type="text"
                      value={rule.prefix}
                      onChange={(event) => {
                        const nextRules = rules.map((item) =>
                          item.id === rule.id ? { ...item, prefix: event.target.value } : item
                        );
                        onChangeRules(nextRules);
                      }}
                      placeholder="Path prefix"
                      className="flex-1 min-w-[160px] rounded-md border border-slate-800 bg-slate-950/60 px-2 py-1 text-xs text-slate-200 placeholder:text-slate-600 focus:outline-none focus:ring-2 focus:ring-blue-500/40"
                    />
                    <select
                      value={rule.mode ?? "prefix"}
                      onChange={(event) => {
                        const nextRules = rules.map((item) =>
                          item.id === rule.id
                            ? {
                                ...item,
                                mode: event.target.value === "segment" ? "segment" : "prefix"
                              }
                            : item
                        );
                        onChangeRules(nextRules);
                      }}
                      className="rounded-md border border-slate-800 bg-slate-950/60 px-2 py-1 text-xs text-slate-200 focus:outline-none focus:ring-2 focus:ring-blue-500/40"
                    >
                      <option value="prefix">Prefix</option>
                      <option value="segment">Prefix + segment</option>
                    </select>
                    <button
                      type="button"
                      className="h-7 w-7 inline-flex items-center justify-center rounded-full border border-slate-800 text-slate-400 hover:bg-slate-800/60"
                      onClick={() => onChangeRules(rules.filter((item) => item.id !== rule.id))}
                      aria-label="Remove group"
                      title="Remove group"
                    >
                      <Trash2 className="h-3 w-3" />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        <div className="flex items-center justify-end gap-2 border-t border-slate-800 px-4 py-3">
          <Button variant="outline" size="sm" onClick={onClose} className="h-8 px-3">
            Done
          </Button>
        </div>
      </div>
    </div>
  );
}
