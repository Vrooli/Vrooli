// Recovery action buttons for healable checks
// [REQ:HEAL-ACTION-001]
import { useState, useCallback, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Play, Square, RotateCcw, FileText, Loader2, AlertTriangle, CheckCircle2, XCircle, Info } from "lucide-react";
import { fetchCheckActions, executeAction, type RecoveryAction, type ActionResult } from "../../lib/api";
import { CodePreview } from "./CodePreview";

interface ActionButtonsProps {
  checkId: string;
  category?: string;
  compact?: boolean; // Smaller buttons for inline use
}

// Map action IDs to icons
const actionIcons: Record<string, typeof Play> = {
  start: Play,
  stop: Square,
  restart: RotateCcw,
  logs: FileText,
};

// Get human-readable reason why an action is unavailable
function getUnavailableReason(actionId: string, allActions: RecoveryAction[]): string {
  // Check if the opposite action is available to infer state
  const startAction = allActions.find((a) => a.id === "start");
  const stopAction = allActions.find((a) => a.id === "stop");

  switch (actionId) {
    case "start":
      if (stopAction?.available) {
        return "Resource is already running";
      }
      return "Cannot start resource";
    case "stop":
      if (startAction?.available) {
        return "Resource is not running";
      }
      return "Cannot stop resource";
    case "restart":
      if (startAction?.available) {
        return "Resource is not running";
      }
      return "Cannot restart resource";
    default:
      return "Action not available";
  }
}

export function ActionButtons({ checkId, category, compact = false }: ActionButtonsProps) {
  const [confirmAction, setConfirmAction] = useState<RecoveryAction | null>(null);
  const [lastResult, setLastResult] = useState<ActionResult | null>(null);
  const queryClient = useQueryClient();

  // Only fetch actions for resource checks (healable checks)
  const isResourceCheck = category === "resource";

  const { data, isLoading } = useQuery({
    queryKey: ["check-actions", checkId],
    queryFn: () => fetchCheckActions(checkId),
    enabled: isResourceCheck,
    staleTime: 30000,
  });

  const executeMutation = useMutation({
    mutationFn: ({ actionId }: { actionId: string }) => executeAction(checkId, actionId),
    onSuccess: (result) => {
      setLastResult(result);
      setConfirmAction(null);
      // Invalidate queries to refresh data
      queryClient.invalidateQueries({ queryKey: ["status"] });
      queryClient.invalidateQueries({ queryKey: ["check-actions", checkId] });
      queryClient.invalidateQueries({ queryKey: ["action-history"] });
    },
    onError: () => {
      setConfirmAction(null);
    },
  });

  const handleActionClick = useCallback((action: RecoveryAction) => {
    if (action.dangerous) {
      setConfirmAction(action);
    } else {
      executeMutation.mutate({ actionId: action.id });
    }
  }, [executeMutation]);

  const handleConfirm = useCallback(() => {
    if (confirmAction) {
      executeMutation.mutate({ actionId: confirmAction.id });
    }
  }, [confirmAction, executeMutation]);

  // Separate actions into available and unavailable groups
  const { availableActions, unavailableActions } = useMemo(() => {
    if (!data?.actions) return { availableActions: [], unavailableActions: [] };
    return {
      availableActions: data.actions.filter((a) => a.available),
      unavailableActions: data.actions.filter((a) => !a.available),
    };
  }, [data?.actions]);

  // Don't show for non-resource checks
  if (!isResourceCheck) {
    return null;
  }

  // Loading state
  if (isLoading) {
    return (
      <div className="mt-2 flex items-center gap-2 text-xs text-text-muted">
        <Loader2 size={12} className="animate-spin" />
        Loading actions...
      </div>
    );
  }

  // No actions available
  if (!data?.actions || data.actions.length === 0) {
    return null;
  }

  const buttonSize = compact ? "px-1.5 py-0.5" : "px-2 py-1";
  const iconSize = compact ? 10 : 12;
  const textSize = compact ? "text-[10px]" : "text-xs";

  return (
    <div className="mt-2 min-w-0 space-y-2">
      {/* Actions row */}
      <div className="flex min-w-0 flex-wrap gap-1.5">
        {/* Available actions - prominent display */}
        {availableActions.map((action) => {
          const Icon = actionIcons[action.id] || Play;
          return (
            <button
              key={action.id}
              onClick={(e) => {
                e.stopPropagation();
                handleActionClick(action);
              }}
              disabled={executeMutation.isPending}
              title={action.description}
              className={`flex min-w-0 items-center gap-1 ${buttonSize} ${textSize} rounded border transition-colors ${
                action.dangerous
                  ? "border-accent-warning/30 bg-accent-warning/10 text-accent-warning hover:bg-accent-warning/20"
                  : "border-border-default/70 bg-surface-overlay/40 text-text-primary hover:bg-surface-overlay/70"
              }`}
            >
              {executeMutation.isPending && executeMutation.variables?.actionId === action.id ? (
                <Loader2 size={iconSize} className="animate-spin" />
              ) : (
                <Icon size={iconSize} />
              )}
              <span className="break-words">{action.name}</span>
            </button>
          );
        })}

        {/* Unavailable actions - subdued with tooltips explaining why */}
        {unavailableActions.map((action) => {
          const Icon = actionIcons[action.id] || Play;
          const reason = getUnavailableReason(action.id, data.actions);
          return (
            <div
              key={action.id}
              title={reason}
              className={`flex min-w-0 cursor-not-allowed items-center gap-1 rounded border border-border-default/50 bg-surface-overlay/20 ${buttonSize} ${textSize} text-text-muted/70`}
            >
              <Icon size={iconSize} />
              <span className="break-words">{action.name}</span>
              <Info size={compact ? 8 : 10} className="ml-0.5 opacity-50" />
            </div>
          );
        })}
      </div>

      {/* Confirmation dialog */}
      {confirmAction && (
        <div className="rounded-lg border border-accent-warning/20 bg-accent-warning/10 p-3">
          <div className="flex items-start gap-2">
            <AlertTriangle size={16} className="mt-0.5 shrink-0 text-accent-warning" />
            <div className="flex-1">
              <p className="text-sm font-medium text-accent-warning">Confirm Action</p>
              <p className="mt-1 text-xs text-text-muted">
                Are you sure you want to {confirmAction.name.toLowerCase()} this resource?
                {confirmAction.id === "stop" && " This will cause downtime."}
                {confirmAction.id === "restart" && " This will cause brief downtime."}
              </p>
              <div className="mt-3 flex flex-wrap gap-2">
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    handleConfirm();
                  }}
                  disabled={executeMutation.isPending}
                  className="rounded bg-accent-warning px-3 py-1 text-xs font-medium text-text-inverse transition-colors hover:bg-accent-warning/80 disabled:opacity-50"
                >
                  {executeMutation.isPending ? "Executing..." : "Confirm"}
                </button>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    setConfirmAction(null);
                  }}
                  disabled={executeMutation.isPending}
                  className="rounded border border-border-default/70 px-3 py-1 text-xs text-text-primary transition-colors hover:bg-surface-overlay/40"
                >
                  Cancel
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Last result */}
      {lastResult && (
        <div className={`rounded-lg border p-3 ${
          lastResult.success
            ? "border-accent-success/20 bg-accent-success/10"
            : "border-accent-danger/20 bg-accent-danger/10"
        }`}>
          <div className="flex items-start gap-2">
            {lastResult.success ? (
              <CheckCircle2 size={16} className="text-emerald-400 mt-0.5 flex-shrink-0" />
            ) : (
              <XCircle size={16} className="text-red-400 mt-0.5 flex-shrink-0" />
            )}
            <div className="flex-1 min-w-0">
              <p className={`text-sm font-medium ${lastResult.success ? "text-accent-success" : "text-accent-danger"}`}>
                {lastResult.message}
              </p>
              {lastResult.output && (
                <CodePreview code={lastResult.output} language="text" maxHeight="8rem" className="mt-2" />
              )}
              {lastResult.error && (
                <p className="mt-1 text-xs text-accent-danger">{lastResult.error}</p>
              )}
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  setLastResult(null);
                }}
                className="mt-2 text-xs text-text-muted transition-colors hover:text-text-primary"
              >
                Dismiss
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
