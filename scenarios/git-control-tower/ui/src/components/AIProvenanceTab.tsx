import { useState } from "react";
import { ChevronDown, ChevronRight, Plus, Minus, FileEdit, AlertCircle } from "lucide-react";
import { useProvenance } from "../lib/hooks";
import type { ProvenanceRunGroup, ProvenanceFile } from "../lib/api";

interface AIProvenanceTabProps {
  repoId?: string | null;
}

function changeTypeIcon(changeType: string) {
  switch (changeType) {
    case "added":
      return <Plus className="h-3 w-3 text-green-400" />;
    case "deleted":
      return <Minus className="h-3 w-3 text-red-400" />;
    default:
      return <FileEdit className="h-3 w-3 text-yellow-400" />;
  }
}

function formatTimestamp(ts: string): string {
  try {
    const d = new Date(ts);
    return d.toLocaleString(undefined, {
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return ts;
  }
}

function truncateId(id: string, len = 8): string {
  return id.length > len ? id.slice(0, len) : id;
}

function RunGroupCard({ group }: { group: ProvenanceRunGroup }) {
  const [expanded, setExpanded] = useState(true);
  const fileCount = group.files.length;

  return (
    <div className="border border-slate-700 rounded-lg overflow-hidden">
      <button
        type="button"
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center gap-3 px-4 py-3 bg-slate-800/50 hover:bg-slate-800 transition-colors text-left"
      >
        {expanded ? (
          <ChevronDown className="h-4 w-4 text-slate-400 shrink-0" />
        ) : (
          <ChevronRight className="h-4 w-4 text-slate-400 shrink-0" />
        )}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-slate-200">
              {group.sandboxOwner || "Unknown"}
            </span>
            {group.runId && (
              <span className="text-xs text-slate-500 font-mono">
                run:{truncateId(group.runId)}
              </span>
            )}
          </div>
          <div className="text-xs text-slate-500 mt-0.5">
            {fileCount} file{fileCount !== 1 ? "s" : ""}
            {" \u00b7 "}
            {formatTimestamp(group.latestAppliedAt)}
          </div>
        </div>
      </button>

      {expanded && (
        <div className="divide-y divide-slate-800">
          {group.files.map((file: ProvenanceFile) => (
            <div
              key={file.filePath}
              className="flex items-center gap-2 px-4 py-2 text-xs"
            >
              {changeTypeIcon(file.changeType)}
              <span className="text-slate-300 truncate flex-1 font-mono">
                {file.relativePath}
              </span>
              <span className="text-slate-600 shrink-0">
                {formatTimestamp(file.appliedAt)}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export function AIProvenanceTab({ repoId }: AIProvenanceTabProps) {
  const provenanceQuery = useProvenance(repoId);

  if (provenanceQuery.isLoading) {
    return (
      <div className="space-y-3">
        <div className="h-20 animate-pulse bg-slate-800 rounded-lg" />
        <div className="h-20 animate-pulse bg-slate-800 rounded-lg" />
      </div>
    );
  }

  if (provenanceQuery.error) {
    return (
      <div className="flex items-center gap-2 text-red-400 text-sm">
        <AlertCircle className="h-4 w-4" />
        Failed to load provenance data
      </div>
    );
  }

  const data = provenanceQuery.data;

  if (!data?.available) {
    return (
      <div className="text-sm text-slate-500">
        {data?.warning || "Workspace Sandbox is not available"}
      </div>
    );
  }

  const groups = data.runGroups ?? [];

  if (groups.length === 0) {
    return (
      <div className="text-sm text-slate-500">
        No AI-authored uncommitted changes
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <div className="text-xs text-slate-500">
        {groups.length} run{groups.length !== 1 ? "s" : ""} with uncommitted changes
      </div>
      {groups.map((group) => (
        <RunGroupCard key={group.runId || group.sandboxId} group={group} />
      ))}
    </div>
  );
}
