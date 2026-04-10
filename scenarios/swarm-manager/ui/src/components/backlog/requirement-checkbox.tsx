import { useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";
import type { ArchiveRequirementGroup } from "../../types";

export function RequirementCheckboxGroup({
  groups,
  selectedIds,
  onToggle,
  depth = 0,
}: {
  groups: ArchiveRequirementGroup[];
  selectedIds: Set<string>;
  onToggle: (id: string) => void;
  depth?: number;
}) {
  return (
    <div className={depth > 0 ? "ml-3 border-l border-slate-700/50 pl-2" : ""}>
      {groups.map((group) => (
        <RequirementCheckboxNode key={group.id} group={group} selectedIds={selectedIds} onToggle={onToggle} depth={depth} />
      ))}
    </div>
  );
}

function RequirementCheckboxNode({
  group,
  selectedIds,
  onToggle,
  depth,
}: {
  group: ArchiveRequirementGroup;
  selectedIds: Set<string>;
  onToggle: (id: string) => void;
  depth: number;
}) {
  const [expanded, setExpanded] = useState(depth === 0);
  const hasContent = group.requirements.length > 0 || group.children.length > 0;

  if (!hasContent) return null;

  return (
    <div>
      <button
        type="button"
        onClick={() => setExpanded(!expanded)}
        className="flex w-full items-center gap-1.5 rounded-md px-1 py-1 text-left text-xs font-medium text-slate-300 hover:bg-slate-800/50"
      >
        {expanded ? <ChevronDown className="h-3 w-3 text-slate-500" /> : <ChevronRight className="h-3 w-3 text-slate-500" />}
        <span>{group.name}</span>
        <span className="text-slate-500">({group.requirements.length})</span>
      </button>
      {expanded && (
        <div className="space-y-0.5">
          {group.requirements.map((req) => (
            <label
              key={req.id}
              className="flex items-start gap-2 rounded-md px-2 py-1 text-xs hover:bg-slate-800/50 cursor-pointer"
            >
              <input
                type="checkbox"
                checked={selectedIds.has(req.id)}
                onChange={() => onToggle(req.id)}
                className="mt-0.5 h-3.5 w-3.5 accent-cyan-500"
              />
              <div className="min-w-0 flex-1">
                <span className="font-mono text-slate-500">{req.id}</span>
                <span className="ml-1 text-slate-300">{req.title}</span>
              </div>
            </label>
          ))}
          {group.children.length > 0 && (
            <RequirementCheckboxGroup groups={group.children} selectedIds={selectedIds} onToggle={onToggle} depth={depth + 1} />
          )}
        </div>
      )}
    </div>
  );
}
