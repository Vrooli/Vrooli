import { Plus, Trash2 } from "lucide-react";
import { Button } from "./ui/button";
import type { GroupingRule } from "./FileList";


/** Extract the prefixes array from a rule, falling back to the singular prefix field. */
function getRulePrefixes(rule: GroupingRule): string[] {
  if (rule.prefixes && rule.prefixes.length > 0) return rule.prefixes;
  if (rule.prefix) return [rule.prefix];
  return [""];
}

interface SettingsTabGroupingProps {
  groupingEnabled: boolean;
  onToggleGrouping: () => void;
  rules: GroupingRule[];
  onChangeRules: (rules: GroupingRule[]) => void;
  isMobile: boolean;
}

function createRule(): GroupingRule {
  return {
    id: `group-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    label: "",
    prefixes: [""],
    mode: "prefix"
  };
}

export function SettingsTabGrouping({
  groupingEnabled,
  onToggleGrouping,
  rules,
  onChangeRules,
  isMobile
}: SettingsTabGroupingProps) {
  if (isMobile) {
    return (
      <div className="space-y-6">
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
                        {prefixes.map((prefix: string, prefixIndex: number) => (
                          <div key={`${rule.id}-prefix-${prefixIndex}`} className="flex items-center gap-2">
                            <input
                              type="text"
                              value={prefix}
                              onChange={(event) => {
                                const nextPrefixes = prefixes.map((item: string, itemIndex: number) =>
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
                                const nextPrefixes = prefixes.filter((_: string, itemIndex: number) => itemIndex !== prefixIndex);
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
    );
  }

  // Desktop
  return (
    <div className="space-y-4">
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
                        {prefixes.map((prefix: string, prefixIndex: number) => (
                          <div key={`${rule.id}-prefix-${prefixIndex}`} className="flex items-center gap-2">
                            <input
                              type="text"
                              value={prefix}
                              onChange={(event) => {
                                const nextPrefixes = prefixes.map((item: string, itemIndex: number) =>
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
                                const nextPrefixes = prefixes.filter((_: string, itemIndex: number) => itemIndex !== prefixIndex);
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
  );
}
