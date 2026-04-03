import type { KeyboardEvent } from "react";
import { Card } from "../ui/card";
import { selectors } from "../../consts/selectors";
import type { PromptCatalogEntry } from "../../types";

type PromptGroup = "capture" | "backlog" | "execution" | "archive" | "support";

const formatUsageLabel = (value: string) => value.replace(/_/g, " ");
const joinParts = (parts?: string[]) => (parts && parts.length > 0 ? parts.join(", ") : "-");

export interface PromptCatalogProps {
  catalogData: PromptCatalogEntry[];
  groupEntries: {
    group: PromptGroup;
    label: string;
    items: PromptCatalogEntry[];
  }[];
  onOpenInViewer: (skillId?: string) => void;
}

export function PromptCatalog({ catalogData, groupEntries, onOpenInViewer }: PromptCatalogProps) {
  const openInViewerOnKey = (event: KeyboardEvent<HTMLElement>, skillID?: string) => {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      onOpenInViewer(skillID);
    }
  };

  return (
    <div className="space-y-6">
      <Card className="space-y-3 p-4" data-testid={selectors.prompts.usageMatrix}>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h2 className="text-lg font-semibold text-slate-100">Prompt Inventory</h2>
          <p className="text-xs text-slate-400">Single source of truth for runtime prompts, generated prompts, and support skills.</p>
        </div>
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
          {groupEntries.map(({ group, label, items }) => (
            <div key={group} className="rounded-md border border-slate-700/60 bg-slate-900/30 p-3">
              <div className="mb-2 flex items-center justify-between">
                <h3 className="text-sm font-semibold text-slate-100">{label}</h3>
                <span className="rounded-full border border-slate-600/60 px-2 py-0.5 text-[10px] text-slate-300">
                  {items.length}
                </span>
              </div>
              <div className="space-y-2">
                {items.length > 0 ? (
                  items.map((entry) => {
                    if (entry.skill_id) {
                      return (
                        <button
                          key={entry.id}
                          type="button"
                          className="w-full rounded border border-slate-700/50 px-2 py-1.5 text-left transition hover:border-cyan-500/50 hover:bg-cyan-500/5"
                          onClick={() => onOpenInViewer(entry.skill_id)}
                          onKeyDown={(event) => openInViewerOnKey(event, entry.skill_id)}
                        >
                          <p className="text-[11px] text-slate-300">{entry.title}</p>
                          <div className="mt-1 flex flex-wrap items-center gap-2 text-[10px] text-slate-400">
                            <span>{formatUsageLabel(entry.usage_type)}</span>
                            <span>{joinParts(entry.modes ?? entry.operations)}</span>
                            <span className="font-mono text-cyan-300">{entry.skill_id}</span>
                          </div>
                        </button>
                      );
                    }
                    return (
                      <div key={entry.id} className="rounded border border-slate-700/50 px-2 py-1.5">
                        <p className="text-[11px] text-slate-300">{entry.title}</p>
                        <div className="mt-1 flex flex-wrap items-center gap-2 text-[10px] text-slate-400">
                          <span>{formatUsageLabel(entry.usage_type)}</span>
                          <span>{joinParts(entry.operations)}</span>
                          <span className="font-mono text-cyan-300">{entry.builder ?? "generated"}</span>
                        </div>
                      </div>
                    );
                  })
                ) : (
                  <p className="text-xs text-slate-500">No catalog entries</p>
                )}
              </div>
            </div>
          ))}
        </div>
      </Card>

      <Card className="space-y-3 p-4" data-testid={selectors.prompts.bindingMap}>
        <h2 className="text-lg font-semibold text-slate-100">Catalog Details</h2>
        <p className="text-sm text-slate-400">
          Each entry records how swarm-manager resolves the prompt, when it runs, and which artifacts it affects.
        </p>
        <div className="overflow-x-auto">
          <table className="w-full min-w-[1080px] text-left text-sm">
            <thead className="text-slate-400">
              <tr>
                <th className="px-2 py-2">Group</th>
                <th className="px-2 py-2">Usage</th>
                <th className="px-2 py-2">Trigger</th>
                <th className="px-2 py-2">Kinds / Modes</th>
                <th className="px-2 py-2">Runtime Prompt</th>
                <th className="px-2 py-2">Purpose</th>
                <th className="px-2 py-2">Outputs</th>
              </tr>
            </thead>
            <tbody>
              {catalogData.map((entry) =>
                entry.skill_id ? (
                  <tr
                    key={entry.id}
                    className="cursor-pointer border-t border-slate-700/60 text-slate-200 transition hover:bg-cyan-500/5"
                    role="button"
                    tabIndex={0}
                    onClick={() => onOpenInViewer(entry.skill_id)}
                    onKeyDown={(event) => openInViewerOnKey(event, entry.skill_id)}
                  >
                    <td className="px-2 py-2 uppercase text-xs text-slate-400">{entry.group}</td>
                    <td className="px-2 py-2 text-slate-300">{formatUsageLabel(entry.usage_type)}</td>
                    <td className="px-2 py-2">
                      <p className="text-slate-100">{entry.title}</p>
                      <p className="text-xs text-slate-400">{entry.trigger}</p>
                    </td>
                    <td className="px-2 py-2 text-slate-300">
                      {joinParts(entry.backlog_kinds)} / {joinParts(entry.modes ?? entry.operations)}
                    </td>
                    <td className="px-2 py-2">
                      <span className="font-mono text-cyan-300">{entry.skill_id}</span>
                    </td>
                    <td className="px-2 py-2 text-slate-300">{entry.purpose}</td>
                    <td className="px-2 py-2 text-xs text-slate-400">{joinParts(entry.output_paths)}</td>
                  </tr>
                ) : (
                  <tr key={entry.id} className="border-t border-slate-700/60 text-slate-200">
                    <td className="px-2 py-2 uppercase text-xs text-slate-400">{entry.group}</td>
                    <td className="px-2 py-2 text-slate-300">{formatUsageLabel(entry.usage_type)}</td>
                    <td className="px-2 py-2">
                      <p className="text-slate-100">{entry.title}</p>
                      <p className="text-xs text-slate-400">{entry.trigger}</p>
                    </td>
                    <td className="px-2 py-2 text-slate-300">
                      {joinParts(entry.backlog_kinds)} / {joinParts(entry.modes ?? entry.operations)}
                    </td>
                    <td className="px-2 py-2">
                      <span className="font-mono text-cyan-300">{entry.builder ?? "generated"}</span>
                    </td>
                    <td className="px-2 py-2 text-slate-300">{entry.purpose}</td>
                    <td className="px-2 py-2 text-xs text-slate-400">{joinParts(entry.output_paths)}</td>
                  </tr>
                )
              )}
            </tbody>
          </table>
        </div>
      </Card>
    </div>
  );
}
