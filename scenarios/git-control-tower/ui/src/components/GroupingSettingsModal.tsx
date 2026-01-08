import { Plus, Trash2, X } from "lucide-react";
import { Button } from "./ui/button";
import { useIsMobile } from "../hooks";
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
    prefixes: [""],
    mode: "prefix"
  };
}

function getRulePrefixes(rule: GroupingRule) {
  if (Array.isArray(rule.prefixes) && rule.prefixes.length > 0) return rule.prefixes;
  if (typeof rule.prefix === "string" && rule.prefix.trim()) return [rule.prefix];
  return [""];
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
  const isMobile = useIsMobile();

  if (!isOpen) return null;

  // Mobile: full-screen modal
  if (isMobile) {
    return (
      <div
        className="fixed inset-0 z-50 flex flex-col bg-slate-950 animate-in slide-in-from-bottom duration-200"
        role="dialog"
        aria-modal="true"
        aria-label="Grouping settings"
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-4 pt-safe">
          <div>
            <h2 className="text-base font-semibold text-slate-100">Grouping Settings</h2>
            {repoDir && (
              <p className="text-xs text-slate-500 mt-1 truncate max-w-[200px]">
                {repoDir}
              </p>
            )}
          </div>
          <button
            type="button"
            className="h-11 w-11 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800/60 active:bg-slate-700 touch-target"
            onClick={onClose}
            aria-label="Close"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto px-4 py-6 space-y-6">
          {/* Enable toggle */}
          <div className="flex items-center justify-between gap-4 rounded-xl border border-slate-800/70 bg-slate-900/40 px-4 py-4">
            <div>
              <div className="text-sm font-semibold text-slate-200">Enable Grouping</div>
              <div className="text-xs text-slate-500 mt-1">
                Group changes by path prefixes (first match wins).
              </div>
            </div>
            <button
              type="button"
              onClick={onToggleGrouping}
              className={`h-10 px-5 rounded-full border text-sm touch-target ${
                groupingEnabled
                  ? "border-emerald-400/40 text-emerald-200 bg-emerald-900/20"
                  : "border-slate-700 text-slate-300 hover:bg-slate-800/50 active:bg-slate-700/50"
              }`}
            >
              {groupingEnabled ? "On" : "Off"}
            </button>
          </div>

          {/* Rules section */}
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-sm font-semibold text-slate-200">Grouping Rules</h3>
                <p className="text-xs text-slate-500 mt-1">
                  Add one or more prefixes per group.
                </p>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => onChangeRules([...rules, createRule()])}
                className="h-10 px-4 touch-target"
              >
                <Plus className="h-4 w-4 mr-1" />
                Add
              </Button>
            </div>

            {rules.length === 0 ? (
              <div className="rounded-xl border border-slate-800/70 bg-slate-900/40 px-4 py-6 text-sm text-slate-500 text-center">
                No grouping rules yet.
              </div>
            ) : (
              <div className="space-y-3">
                {rules.map((rule, index) => {
                  const prefixes = getRulePrefixes(rule);
                  return (
                    <div
                      key={rule.id}
                      className="rounded-xl border border-slate-800/60 bg-slate-950/50 p-4 space-y-4"
                    >
                      <div className="flex items-center justify-between">
                        <span className="text-xs text-slate-400">Group {index + 1}</span>
                        <button
                          type="button"
                          className="h-10 w-10 inline-flex items-center justify-center rounded-full border border-slate-800 text-slate-400 hover:bg-slate-800/60 active:bg-slate-700 touch-target"
                          onClick={() => onChangeRules(rules.filter((item) => item.id !== rule.id))}
                          aria-label="Remove group"
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </div>

                      <div className="space-y-2">
                        <div className="text-xs font-semibold text-slate-300">Label</div>
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
                          className="w-full rounded-lg border border-slate-800 bg-slate-950/60 px-4 py-3 text-sm text-slate-200 placeholder:text-slate-600 focus:outline-none focus:ring-2 focus:ring-blue-500/40 touch-target"
                        />
                      </div>

                      <div className="space-y-2">
                        <div className="text-xs font-semibold text-slate-300">Grouping mode</div>
                        <select
                          value={rule.mode ?? "prefix"}
                          onChange={(event) => {
                            const nextRules = rules.map((item) =>
                              item.id === rule.id
                                ? {
                                    ...item,
                                    mode: (event.target.value === "segment" ? "segment" : "prefix") as "prefix" | "segment"
                                  }
                                : item
                            );
                            onChangeRules(nextRules);
                          }}
                          className="w-full rounded-lg border border-slate-800 bg-slate-950/60 px-4 py-3 text-sm text-slate-200 focus:outline-none focus:ring-2 focus:ring-blue-500/40 touch-target"
                        >
                          <option value="prefix">Prefix</option>
                          <option value="segment">Prefix + segment</option>
                        </select>
                        <p className="text-xs text-slate-500">
                          Prefix keeps each prefix as one group. Prefix + segment groups by the next
                          path segment after the prefix (e.g., <span className="font-mono">scenarios/foo/</span>).
                        </p>
                      </div>

                      <div className="space-y-2">
                        <div className="flex items-center justify-between">
                          <div className="text-xs font-semibold text-slate-300">Prefixes</div>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => {
                              const nextRules = rules.map((item) =>
                                item.id === rule.id
                                  ? { ...item, prefixes: [...prefixes, ""], prefix: prefixes[0] ?? "" }
                                  : item
                              );
                              onChangeRules(nextRules);
                            }}
                            className="h-9 px-3"
                          >
                            <Plus className="h-4 w-4 mr-1" />
                            Add prefix
                          </Button>
                        </div>
                        <div className="space-y-2">
                          {prefixes.map((prefix, prefixIndex) => (
                            <div key={`${rule.id}-prefix-${prefixIndex}`} className="flex items-center gap-2">
                              <input
                                type="text"
                                value={prefix}
                                onChange={(event) => {
                                  const nextPrefixes = prefixes.map((item, itemIndex) =>
                                    itemIndex === prefixIndex ? event.target.value : item
                                  );
                                  const nextRules = rules.map((item) =>
                                    item.id === rule.id
                                      ? { ...item, prefixes: nextPrefixes, prefix: nextPrefixes[0] ?? "" }
                                      : item
                                  );
                                  onChangeRules(nextRules);
                                }}
                                placeholder="Path prefix (e.g., scenarios/)"
                                className="w-full rounded-lg border border-slate-800 bg-slate-950/60 px-4 py-3 text-sm text-slate-200 placeholder:text-slate-600 focus:outline-none focus:ring-2 focus:ring-blue-500/40 touch-target"
                              />
                              <button
                                type="button"
                                className="h-10 w-10 inline-flex items-center justify-center rounded-full border border-slate-800 text-slate-400 hover:bg-slate-800/60 active:bg-slate-700 touch-target"
                                onClick={() => {
                                  const nextPrefixes = prefixes.filter((_, itemIndex) => itemIndex !== prefixIndex);
                                  const normalizedPrefixes = nextPrefixes.length > 0 ? nextPrefixes : [""];
                                  const nextRules = rules.map((item) =>
                                    item.id === rule.id
                                      ? {
                                          ...item,
                                          prefixes: normalizedPrefixes,
                                          prefix: normalizedPrefixes[0] ?? ""
                                        }
                                      : item
                                  );
                                  onChangeRules(nextRules);
                                }}
                                aria-label="Remove prefix"
                              >
                                <Trash2 className="h-4 w-4" />
                              </button>
                            </div>
                          ))}
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="border-t border-slate-800 px-4 py-4 pb-safe">
          <Button
            variant="default"
            size="sm"
            onClick={onClose}
            className="w-full h-12 text-sm touch-target"
          >
            Done
          </Button>
        </div>
      </div>
    );
  }

  // Desktop: centered modal
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
                  Add one or more prefixes per group.
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
                {rules.map((rule, index) => {
                  const prefixes = getRulePrefixes(rule);
                  return (
                    <div
                      key={rule.id}
                      className="rounded-lg border border-slate-800/60 bg-slate-950/50 px-3 py-3 space-y-3"
                    >
                      <div className="flex items-center justify-between">
                        <div className="text-[11px] text-slate-400">Group {index + 1}</div>
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

                      <div className="grid grid-cols-2 gap-3">
                        <div className="col-span-2">
                          <div className="text-[11px] font-semibold text-slate-300 mb-1">Label</div>
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
                            className="w-full rounded-md border border-slate-800 bg-slate-950/60 px-2 py-1 text-xs text-slate-200 placeholder:text-slate-600 focus:outline-none focus:ring-2 focus:ring-blue-500/40"
                          />
                        </div>
                        <div className="col-span-2">
                          <div className="text-[11px] font-semibold text-slate-300 mb-1">Grouping mode</div>
                          <select
                            value={rule.mode ?? "prefix"}
                            onChange={(event) => {
                              const nextRules = rules.map((item) =>
                                item.id === rule.id
                                  ? {
                                      ...item,
                                      mode: (event.target.value === "segment" ? "segment" : "prefix") as "prefix" | "segment"
                                    }
                                  : item
                              );
                              onChangeRules(nextRules);
                            }}
                            className="w-full rounded-md border border-slate-800 bg-slate-950/60 px-2 py-1 text-xs text-slate-200 focus:outline-none focus:ring-2 focus:ring-blue-500/40"
                          >
                            <option value="prefix">Prefix</option>
                            <option value="segment">Prefix + segment</option>
                          </select>
                          <div className="text-[11px] text-slate-500 mt-1">
                            Prefix keeps each prefix as one group. Prefix + segment groups by the next
                            path segment after the prefix (e.g., <span className="font-mono">scenarios/foo/</span>).
                          </div>
                        </div>
                        <div className="col-span-2">
                          <div className="flex items-center justify-between mb-1">
                            <div className="text-[11px] font-semibold text-slate-300">Prefixes</div>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => {
                                const nextRules = rules.map((item) =>
                                  item.id === rule.id
                                    ? { ...item, prefixes: [...prefixes, ""], prefix: prefixes[0] ?? "" }
                                    : item
                                );
                                onChangeRules(nextRules);
                              }}
                              className="h-7 px-2"
                            >
                              <Plus className="h-3 w-3 mr-1" />
                              Add prefix
                            </Button>
                          </div>
                          <div className="space-y-2">
                            {prefixes.map((prefix, prefixIndex) => (
                              <div key={`${rule.id}-prefix-${prefixIndex}`} className="flex items-center gap-2">
                                <input
                                  type="text"
                                  value={prefix}
                                  onChange={(event) => {
                                    const nextPrefixes = prefixes.map((item, itemIndex) =>
                                      itemIndex === prefixIndex ? event.target.value : item
                                    );
                                    const nextRules = rules.map((item) =>
                                      item.id === rule.id
                                        ? { ...item, prefixes: nextPrefixes, prefix: nextPrefixes[0] ?? "" }
                                        : item
                                    );
                                    onChangeRules(nextRules);
                                  }}
                                  placeholder="Path prefix"
                                  className="w-full rounded-md border border-slate-800 bg-slate-950/60 px-2 py-1 text-xs text-slate-200 placeholder:text-slate-600 focus:outline-none focus:ring-2 focus:ring-blue-500/40"
                                />
                                <button
                                  type="button"
                                  className="h-7 w-7 inline-flex items-center justify-center rounded-full border border-slate-800 text-slate-400 hover:bg-slate-800/60"
                                  onClick={() => {
                                    const nextPrefixes = prefixes.filter((_, itemIndex) => itemIndex !== prefixIndex);
                                    const normalizedPrefixes = nextPrefixes.length > 0 ? nextPrefixes : [""];
                                    const nextRules = rules.map((item) =>
                                      item.id === rule.id
                                        ? {
                                            ...item,
                                            prefixes: normalizedPrefixes,
                                            prefix: normalizedPrefixes[0] ?? ""
                                          }
                                        : item
                                    );
                                    onChangeRules(nextRules);
                                  }}
                                  aria-label="Remove prefix"
                                >
                                  <Trash2 className="h-3 w-3" />
                                </button>
                              </div>
                            ))}
                          </div>
                        </div>
                      </div>
                    </div>
                  );
                })}
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
