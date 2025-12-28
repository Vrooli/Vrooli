import { useState } from "react";
import {
  Server,
  RefreshCw,
  ExternalLink,
  Trash2,
  Eye,
  Search,
  AlertCircle,
  CheckCircle2,
  Clock,
  Loader2,
  XCircle,
  Pause,
  Globe,
} from "lucide-react";
import { useDeployments, useDeleteDeployment, useInspectDeployment, getStatusInfo } from "../../hooks/useDeployments";
import { cn } from "../../lib/utils";
import type { DeploymentSummary } from "../../lib/api";
import { DeploymentDetails } from "./DeploymentDetails";

interface DeploymentsPageProps {
  onBack: () => void;
}

export function DeploymentsPage({ onBack }: DeploymentsPageProps) {
  const { data: deployments, isLoading, error, refetch } = useDeployments();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [showDeleteDialog, setShowDeleteDialog] = useState<string | null>(null);
  const [deleteStopOnVPS, setDeleteStopOnVPS] = useState(false);

  const deleteMutation = useDeleteDeployment();
  const inspectMutation = useInspectDeployment();

  const handleDelete = async () => {
    if (!showDeleteDialog) return;
    await deleteMutation.mutateAsync({ id: showDeleteDialog, stopOnVPS: deleteStopOnVPS });
    setShowDeleteDialog(null);
    setDeleteStopOnVPS(false);
  };

  const handleInspect = async (id: string) => {
    await inspectMutation.mutateAsync(id);
  };

  // If a deployment is selected, show the detail view
  if (selectedId) {
    return (
      <DeploymentDetails
        deploymentId={selectedId}
        onBack={() => setSelectedId(null)}
      />
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Deployments</h1>
          <p className="text-sm text-slate-400 mt-1">
            Manage your deployed scenarios
          </p>
        </div>
        <button
          onClick={() => refetch()}
          disabled={isLoading}
          className={cn(
            "flex items-center gap-2 px-4 py-2 rounded-lg border border-white/10",
            "hover:bg-white/5 transition-colors text-sm font-medium",
            isLoading && "opacity-50 cursor-not-allowed"
          )}
        >
          <RefreshCw className={cn("h-4 w-4", isLoading && "animate-spin")} />
          Refresh
        </button>
      </div>

      {/* Error state */}
      {error && (
        <div className="p-4 rounded-lg border border-red-500/30 bg-red-500/10 text-red-400">
          <div className="flex items-center gap-2">
            <AlertCircle className="h-5 w-5" />
            <span>Failed to load deployments: {error.message}</span>
          </div>
        </div>
      )}

      {/* Loading state */}
      {isLoading && !deployments && (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-blue-400" />
        </div>
      )}

      {/* Empty state */}
      {!isLoading && deployments && deployments.length === 0 && (
        <div className="text-center py-12 border border-white/10 rounded-lg bg-slate-900/50">
          <Server className="h-12 w-12 text-slate-600 mx-auto mb-4" />
          <h3 className="text-lg font-medium text-white mb-2">No deployments yet</h3>
          <p className="text-slate-400 max-w-md mx-auto">
            Start a new deployment from the Dashboard to see your deployed scenarios here.
          </p>
          <button
            onClick={onBack}
            className="mt-4 px-4 py-2 rounded-lg bg-blue-500 hover:bg-blue-600 text-white font-medium transition-colors"
          >
            Go to Dashboard
          </button>
        </div>
      )}

      {/* Deployments list */}
      {deployments && deployments.length > 0 && (
        <div className="space-y-3">
          {deployments.map((deployment) => (
            <DeploymentCard
              key={deployment.id}
              deployment={deployment}
              onSelect={() => setSelectedId(deployment.id)}
              onInspect={() => handleInspect(deployment.id)}
              onDelete={() => setShowDeleteDialog(deployment.id)}
              isInspecting={inspectMutation.isPending && inspectMutation.variables === deployment.id}
            />
          ))}
        </div>
      )}

      {/* Delete confirmation dialog */}
      {showDeleteDialog && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-slate-900 border border-white/10 rounded-lg p-6 max-w-md w-full mx-4">
            <h3 className="text-lg font-semibold text-white mb-2">Delete Deployment</h3>
            <p className="text-slate-400 mb-4">
              Are you sure you want to delete this deployment record?
            </p>
            <label className="flex items-center gap-2 mb-4 text-sm">
              <input
                type="checkbox"
                checked={deleteStopOnVPS}
                onChange={(e) => setDeleteStopOnVPS(e.target.checked)}
                className="rounded border-white/20 bg-slate-800 text-blue-500 focus:ring-blue-500"
              />
              <span className="text-slate-300">Also stop the scenario on the VPS</span>
            </label>
            <div className="flex justify-end gap-3">
              <button
                onClick={() => {
                  setShowDeleteDialog(null);
                  setDeleteStopOnVPS(false);
                }}
                className="px-4 py-2 rounded-lg border border-white/10 hover:bg-white/5 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleDelete}
                disabled={deleteMutation.isPending}
                className="px-4 py-2 rounded-lg bg-red-500 hover:bg-red-600 text-white font-medium transition-colors disabled:opacity-50"
              >
                {deleteMutation.isPending ? "Deleting..." : "Delete"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// Status badge component
function StatusBadge({ status }: { status: DeploymentSummary["status"] }) {
  const info = getStatusInfo(status);

  const colorClasses = {
    slate: "bg-slate-500/20 text-slate-400",
    blue: "bg-blue-500/20 text-blue-400",
    emerald: "bg-emerald-500/20 text-emerald-400",
    red: "bg-red-500/20 text-red-400",
    amber: "bg-amber-500/20 text-amber-400",
  };

  const icons = {
    clock: Clock,
    loader: Loader2,
    check: CheckCircle2,
    "check-circle": CheckCircle2,
    "x-circle": XCircle,
    pause: Pause,
    help: AlertCircle,
  };

  const IconComponent = icons[info.icon as keyof typeof icons] || AlertCircle;
  const isAnimated = info.icon === "loader";

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium",
        colorClasses[info.color as keyof typeof colorClasses]
      )}
    >
      <IconComponent className={cn("h-3.5 w-3.5", isAnimated && "animate-spin")} />
      {info.label}
    </span>
  );
}

// Deployment card component
interface DeploymentCardProps {
  deployment: DeploymentSummary;
  onSelect: () => void;
  onInspect: () => void;
  onDelete: () => void;
  isInspecting: boolean;
}

function DeploymentCard({
  deployment,
  onSelect,
  onInspect,
  onDelete,
  isInspecting,
}: DeploymentCardProps) {
  const formattedDate = deployment.last_deployed_at
    ? new Date(deployment.last_deployed_at).toLocaleString()
    : new Date(deployment.created_at).toLocaleString();

  return (
    <div className="border border-white/10 rounded-lg bg-slate-900/50 p-4 hover:border-white/20 transition-colors">
      <div className="flex items-start justify-between gap-4">
        {/* Left: Info */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-3 mb-2">
            <h3 className="text-lg font-medium text-white truncate">
              {deployment.name}
            </h3>
            <StatusBadge status={deployment.status} />
          </div>
          <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-slate-400">
            <span className="flex items-center gap-1.5">
              <Server className="h-4 w-4" />
              {deployment.scenario_id}
            </span>
            {deployment.domain && (
              <span className="flex items-center gap-1.5">
                <Globe className="h-4 w-4" />
                {deployment.domain}
              </span>
            )}
            <span className="flex items-center gap-1.5">
              <Clock className="h-4 w-4" />
              {formattedDate}
            </span>
          </div>
          {deployment.error_message && (
            <p className="mt-2 text-sm text-red-400 truncate">
              Error: {deployment.error_message}
            </p>
          )}
        </div>

        {/* Right: Actions */}
        <div className="flex items-center gap-2">
          <button
            onClick={(e) => {
              e.stopPropagation();
              onInspect();
            }}
            disabled={isInspecting}
            className={cn(
              "p-2 rounded-lg border border-white/10 hover:bg-white/5 transition-colors",
              isInspecting && "opacity-50 cursor-not-allowed"
            )}
            title="Inspect deployment"
          >
            {isInspecting ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Search className="h-4 w-4" />
            )}
          </button>
          {deployment.domain && deployment.status === "deployed" && (
            <a
              href={`https://${deployment.domain}`}
              target="_blank"
              rel="noopener noreferrer"
              onClick={(e) => e.stopPropagation()}
              className="p-2 rounded-lg border border-white/10 hover:bg-white/5 transition-colors"
              title="Open in browser"
            >
              <ExternalLink className="h-4 w-4" />
            </a>
          )}
          <button
            onClick={(e) => {
              e.stopPropagation();
              onSelect();
            }}
            className="p-2 rounded-lg border border-white/10 hover:bg-white/5 transition-colors"
            title="View details"
          >
            <Eye className="h-4 w-4" />
          </button>
          <button
            onClick={(e) => {
              e.stopPropagation();
              onDelete();
            }}
            className="p-2 rounded-lg border border-red-500/30 hover:bg-red-500/10 text-red-400 transition-colors"
            title="Delete deployment"
          >
            <Trash2 className="h-4 w-4" />
          </button>
        </div>
      </div>
    </div>
  );
}
