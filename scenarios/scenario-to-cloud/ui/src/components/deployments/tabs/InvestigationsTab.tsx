import { useMemo } from "react";
import {
  RefreshCw,
  Loader2,
  AlertCircle,
  CheckCircle2,
  XCircle,
  StopCircle,
  ChevronRight,
  Bot,
  Wrench,
  Search,
} from "lucide-react";
import { useInvestigations } from "../../../hooks/useInvestigation";
import { Tooltip } from "../../ui/tooltip";
import { cn } from "../../../lib/utils";
import type { InvestigationSummary, InvestigationStatus } from "../../../types/investigation";

interface InvestigationsTabProps {
  deploymentId: string;
  deploymentRunId?: string;
  lastDeployedAt?: string;
  onViewReport: (investigation: InvestigationSummary) => void;
}

interface InvestigationNode {
  investigation: InvestigationSummary;
  fixApplications: InvestigationSummary[];
}

function StatusIcon({ status }: { status: InvestigationStatus }) {
  switch (status) {
    case "completed":
      return <CheckCircle2 className="h-4 w-4 text-emerald-400" />;
    case "failed":
      return <XCircle className="h-4 w-4 text-red-400" />;
    case "cancelled":
      return <StopCircle className="h-4 w-4 text-slate-400" />;
    case "running":
    case "pending":
    default:
      return <Loader2 className="h-4 w-4 text-blue-400 animate-spin" />;
  }
}

function formatDate(dateString: string): string {
  const date = new Date(dateString);
  return date.toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function InvestigationItem({
  investigation,
  isFixApplication,
  isOutdated,
  onClick,
}: {
  investigation: InvestigationSummary;
  isFixApplication: boolean;
  isOutdated: boolean;
  onClick: () => void;
}) {
  const Icon = isFixApplication ? Wrench : Bot;
  const label = isFixApplication ? "Fix Application" : "Investigation";

  return (
    <button
      onClick={onClick}
      className={cn(
        "w-full flex items-center gap-3 px-3 py-2 rounded-lg text-left transition-colors",
        "hover:bg-white/5 border border-transparent hover:border-white/10",
        isFixApplication && "ml-6"
      )}
    >
      <Icon className={cn(
        "h-4 w-4 flex-shrink-0",
        isFixApplication ? "text-amber-400" : "text-blue-400"
      )} />
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-slate-200">{label}</span>
          <span className="text-xs text-slate-500">{formatDate(investigation.created_at)}</span>
          {isOutdated && (
            <Tooltip content="From a previous deployment run, not the current one." side="top">
              <span className="text-[10px] uppercase tracking-wide text-amber-300 border border-amber-500/40 rounded-full px-2 py-0.5">
                Outdated
              </span>
            </Tooltip>
          )}
        </div>
        {investigation.error_message && (
          <p className="text-xs text-red-400 truncate mt-0.5">
            {investigation.error_message}
          </p>
        )}
      </div>
      <StatusIcon status={investigation.status} />
      <ChevronRight className="h-4 w-4 text-slate-500" />
    </button>
  );
}

function isOutdatedInvestigation(
  investigation: InvestigationSummary,
  deploymentRunId?: string,
  lastDeployedAt?: string
): boolean {
  if (deploymentRunId && investigation.deployment_run_id) {
    return investigation.deployment_run_id !== deploymentRunId;
  }
  if (lastDeployedAt) {
    return new Date(investigation.created_at).getTime() < new Date(lastDeployedAt).getTime();
  }
  return false;
}

export function InvestigationsTab({ deploymentId, deploymentRunId, lastDeployedAt, onViewReport }: InvestigationsTabProps) {
  const {
    data: investigations,
    isLoading,
    error,
    refetch,
    isFetching,
  } = useInvestigations(deploymentId, 50);

  // Group fix applications under their parent investigations
  const investigationTree = useMemo(() => {
    if (!investigations) return [];

    const nodes: InvestigationNode[] = [];
    const fixApplicationMap = new Map<string, InvestigationSummary[]>();

    // First pass: identify fix applications and group them by source
    for (const inv of investigations) {
      if (inv.source_investigation_id) {
        const fixes = fixApplicationMap.get(inv.source_investigation_id) ?? [];
        fixes.push(inv);
        fixApplicationMap.set(inv.source_investigation_id, fixes);
      }
    }

    // Second pass: build tree nodes for parent investigations
    for (const inv of investigations) {
      if (!inv.source_investigation_id) {
        nodes.push({
          investigation: inv,
          fixApplications: fixApplicationMap.get(inv.id) ?? [],
        });
      }
    }

    // Handle orphan fix applications (parent deleted - unlikely but possible)
    for (const [sourceId, fixes] of fixApplicationMap) {
      const hasParent = investigations.some((inv) => inv.id === sourceId);
      if (!hasParent) {
        for (const fix of fixes) {
          nodes.push({
            investigation: fix,
            fixApplications: [],
          });
        }
      }
    }

    return nodes;
  }, [investigations]);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-blue-400" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-4 rounded-lg border border-red-500/30 bg-red-500/10 text-red-400">
        <div className="flex items-center gap-2">
          <AlertCircle className="h-5 w-5" />
          <span>Failed to load investigations: {error.message}</span>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Header with refresh */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-slate-400">
          <Search className="h-4 w-4" />
          <span className="text-sm font-medium">
            {investigationTree.length} investigation{investigationTree.length !== 1 ? "s" : ""}
          </span>
        </div>
        <button
          onClick={() => refetch()}
          disabled={isFetching}
          className={cn(
            "flex items-center gap-2 px-3 py-1.5 rounded-lg border border-white/10",
            "hover:bg-white/5 transition-colors text-sm font-medium",
            isFetching && "opacity-50 cursor-not-allowed"
          )}
        >
          {isFetching ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <RefreshCw className="h-4 w-4" />
          )}
          Refresh
        </button>
      </div>

      {/* Investigation list */}
      {investigationTree.length === 0 ? (
        <div className="text-center py-12 border border-white/10 rounded-lg bg-slate-900/50">
          <Bot className="h-12 w-12 text-slate-600 mx-auto mb-3" />
          <p className="text-slate-400 mb-1">No investigations yet</p>
          <p className="text-sm text-slate-500">
            Click "Investigate" on a failed deployment to analyze the issue
          </p>
        </div>
      ) : (
        <div className="space-y-1 border border-white/10 rounded-lg bg-slate-900/50 p-2">
          {investigationTree.map((node) => (
            <div key={node.investigation.id}>
              <InvestigationItem
                investigation={node.investigation}
                isFixApplication={false}
                isOutdated={isOutdatedInvestigation(node.investigation, deploymentRunId, lastDeployedAt)}
                onClick={() => onViewReport(node.investigation)}
              />
              {node.fixApplications.map((fix) => (
                <InvestigationItem
                  key={fix.id}
                  investigation={fix}
                  isFixApplication={true}
                  isOutdated={isOutdatedInvestigation(fix, deploymentRunId, lastDeployedAt)}
                  onClick={() => onViewReport(fix)}
                />
              ))}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
