import { useState, useMemo } from "react";
import {
  Box,
  ChevronDown,
  ChevronRight,
  Clock,
  HardDrive,
  Loader2,
  Play,
  Square,
  CheckCircle,
  XCircle,
  AlertCircle,
  FolderOpen,
} from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent } from "./ui/card";
import { ScrollArea } from "./ui/scroll-area";
import { Badge } from "./ui/badge";
import type { Sandbox, Status } from "../lib/api";
import { formatBytes, formatRelativeTime } from "../lib/api";
import { SELECTORS } from "../consts/selectors";

interface SandboxListProps {
  sandboxes: Sandbox[];
  selectedId?: string;
  onSelect: (sandbox: Sandbox) => void;
  isLoading: boolean;
}

interface SandboxGroupProps {
  title: string;
  status: Status;
  sandboxes: Sandbox[];
  selectedId?: string;
  onSelect: (sandbox: Sandbox) => void;
  icon: React.ReactNode;
  defaultExpanded?: boolean;
}

const STATUS_CONFIG: Record<Status, { icon: React.ReactNode; label: string }> = {
  creating: { icon: <Loader2 className="h-3.5 w-3.5 text-blue-400 animate-spin" />, label: "Creating" },
  active: { icon: <Play className="h-3.5 w-3.5 text-emerald-400" />, label: "Active" },
  stopped: { icon: <Square className="h-3.5 w-3.5 text-amber-400" />, label: "Stopped" },
  approved: { icon: <CheckCircle className="h-3.5 w-3.5 text-green-400" />, label: "Approved" },
  rejected: { icon: <XCircle className="h-3.5 w-3.5 text-red-400" />, label: "Rejected" },
  deleted: { icon: <Box className="h-3.5 w-3.5 text-slate-500" />, label: "Deleted" },
  error: { icon: <AlertCircle className="h-3.5 w-3.5 text-red-400" />, label: "Error" },
};

function SandboxGroup({
  title,
  status,
  sandboxes,
  selectedId,
  onSelect,
  icon,
  defaultExpanded = true,
}: SandboxGroupProps) {
  const [expanded, setExpanded] = useState(defaultExpanded);

  if (sandboxes.length === 0) return null;

  return (
    <div className="mb-3" data-testid={SELECTORS.sandboxGroup(status)}>
      <button
        className="flex items-center gap-2 w-full text-left px-2 py-1.5 hover:bg-slate-800/50 rounded transition-colors"
        onClick={() => setExpanded(!expanded)}
      >
        {expanded ? (
          <ChevronDown className="h-3 w-3 text-slate-500" />
        ) : (
          <ChevronRight className="h-3 w-3 text-slate-500" />
        )}
        {icon}
        <span className="text-xs font-medium text-slate-400 uppercase tracking-wider">
          {title}
        </span>
        <span className="text-xs text-slate-600 ml-auto">{sandboxes.length}</span>
      </button>

      {expanded && (
        <ul className="mt-1 space-y-0.5">
          {sandboxes.map((sandbox) => {
            const isSelected = selectedId === sandbox.id;

            return (
              <li
                key={sandbox.id}
                className={`group flex flex-col px-2 py-2 rounded cursor-pointer transition-colors ${
                  isSelected
                    ? "bg-slate-700/50 border-l-2 border-l-emerald-500"
                    : "hover:bg-slate-800/50 border-l-2 border-l-transparent"
                }`}
                data-testid={SELECTORS.sandboxItem}
                data-sandbox-id={sandbox.id}
                onClick={() => onSelect(sandbox)}
              >
                {/* Sandbox scope path */}
                <div className="flex items-center gap-2 min-w-0">
                  <FolderOpen className="h-3.5 w-3.5 text-slate-500 flex-shrink-0" />
                  <span className="font-mono text-xs text-slate-200 truncate">
                    {sandbox.scopePath || "/"}
                  </span>
                </div>

                {/* Metadata row */}
                <div className="flex items-center gap-3 mt-1.5 text-xs text-slate-500">
                  {/* Owner */}
                  {sandbox.owner && (
                    <span className="truncate max-w-[100px]" title={sandbox.owner}>
                      {sandbox.owner}
                    </span>
                  )}

                  {/* Size */}
                  <span className="flex items-center gap-1">
                    <HardDrive className="h-3 w-3" />
                    {formatBytes(sandbox.sizeBytes)}
                  </span>

                  {/* Created time */}
                  <span className="flex items-center gap-1">
                    <Clock className="h-3 w-3" />
                    {formatRelativeTime(sandbox.createdAt)}
                  </span>
                </div>

                {/* Error message if present */}
                {sandbox.errorMessage && (
                  <div className="mt-1.5 text-xs text-red-400 truncate" title={sandbox.errorMessage}>
                    {sandbox.errorMessage}
                  </div>
                )}
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}

export function SandboxList({
  sandboxes,
  selectedId,
  onSelect,
  isLoading,
}: SandboxListProps) {
  // Group sandboxes by status
  const grouped = useMemo(() => {
    const groups: Record<Status, Sandbox[]> = {
      creating: [],
      active: [],
      stopped: [],
      approved: [],
      rejected: [],
      deleted: [],
      error: [],
    };

    for (const sb of sandboxes) {
      if (groups[sb.status]) {
        groups[sb.status].push(sb);
      }
    }

    // Sort each group by createdAt descending
    for (const status of Object.keys(groups) as Status[]) {
      groups[status].sort((a, b) =>
        new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
      );
    }

    return groups;
  }, [sandboxes]);

  const hasAnySandboxes = sandboxes.length > 0;

  return (
    <Card className="h-full flex flex-col" data-testid={SELECTORS.sandboxList}>
      <CardHeader className="flex-row items-center justify-between space-y-0 py-3">
        <CardTitle className="flex items-center gap-2">
          <Box className="h-4 w-4 text-slate-500" />
          Sandboxes
        </CardTitle>
        {isLoading && (
          <Loader2 className="h-4 w-4 animate-spin text-slate-500" />
        )}
      </CardHeader>

      <CardContent className="flex-1 p-0 overflow-hidden">
        <ScrollArea className="h-full px-2 py-2">
          {/* Active and Creating - most important */}
          <SandboxGroup
            title="Active"
            status="active"
            sandboxes={[...grouped.creating, ...grouped.active]}
            selectedId={selectedId}
            onSelect={onSelect}
            icon={STATUS_CONFIG.active.icon}
            defaultExpanded={true}
          />

          {/* Stopped - needs review */}
          <SandboxGroup
            title="Stopped"
            status="stopped"
            sandboxes={grouped.stopped}
            selectedId={selectedId}
            onSelect={onSelect}
            icon={STATUS_CONFIG.stopped.icon}
            defaultExpanded={true}
          />

          {/* Error - needs attention */}
          <SandboxGroup
            title="Error"
            status="error"
            sandboxes={grouped.error}
            selectedId={selectedId}
            onSelect={onSelect}
            icon={STATUS_CONFIG.error.icon}
            defaultExpanded={true}
          />

          {/* Approved - historical */}
          <SandboxGroup
            title="Approved"
            status="approved"
            sandboxes={grouped.approved}
            selectedId={selectedId}
            onSelect={onSelect}
            icon={STATUS_CONFIG.approved.icon}
            defaultExpanded={false}
          />

          {/* Rejected - historical */}
          <SandboxGroup
            title="Rejected"
            status="rejected"
            sandboxes={grouped.rejected}
            selectedId={selectedId}
            onSelect={onSelect}
            icon={STATUS_CONFIG.rejected.icon}
            defaultExpanded={false}
          />

          {/* Empty State */}
          {!isLoading && !hasAnySandboxes && (
            <div className="flex flex-col items-center justify-center py-12 text-center" data-testid={SELECTORS.emptyState}>
              <Box className="h-10 w-10 text-slate-700 mb-4" />
              <p className="text-sm text-slate-400">No sandboxes yet</p>
              <p className="text-xs text-slate-500 mt-1">
                Create a sandbox to get started
              </p>
            </div>
          )}
        </ScrollArea>
      </CardContent>
    </Card>
  );
}
