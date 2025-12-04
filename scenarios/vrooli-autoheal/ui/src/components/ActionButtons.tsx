// Recovery action buttons for healable checks
// [REQ:HEAL-ACTION-001]
import { useState, useCallback, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Play, Square, RotateCcw, FileText, Loader2, AlertTriangle, CheckCircle2, XCircle, ChevronDown, ChevronUp, Info } from "lucide-react";
import { fetchCheckActions, executeAction, type RecoveryAction, type ActionResult } from "../lib/api";

interface ActionButtonsProps {
  checkId: string;
  category?: string;
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

export function ActionButtons({ checkId, category }: ActionButtonsProps) {
  const [expanded, setExpanded] = useState(false);
  const [confirmAction, setConfirmAction] = useState<RecoveryAction | null>(null);
  const [lastResult, setLastResult] = useState<ActionResult | null>(null);
  const queryClient = useQueryClient();

  // Only fetch actions for resource checks (healable checks)
  const isResourceCheck = category === "resource";

  const { data, isLoading } = useQuery({
    queryKey: ["check-actions", checkId],
    queryFn: () => fetchCheckActions(checkId),
    enabled: isResourceCheck && expanded,
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

  return (
    <div className="mt-2">
      {/* Toggle button */}
      <button
        onClick={() => setExpanded(!expanded)}
        className="flex items-center gap-1 text-xs text-slate-500 hover:text-slate-300 transition-colors"
      >
        {expanded ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
        <span>Actions</span>
      </button>

      {/* Expanded content */}
      {expanded && (
        <div className="mt-2 p-2 rounded-lg bg-black/20 border border-white/5">
          {isLoading ? (
            <div className="flex items-center gap-2 text-xs text-slate-500">
              <Loader2 size={12} className="animate-spin" />
              Loading actions...
            </div>
          ) : data?.actions && data.actions.length > 0 ? (
            <>
              {/* Available actions - prominent display */}
              {availableActions.length > 0 && (
                <div className="flex flex-wrap gap-2">
                  {availableActions.map((action) => {
                    const Icon = actionIcons[action.id] || Play;
                    return (
                      <button
                        key={action.id}
                        onClick={() => handleActionClick(action)}
                        disabled={executeMutation.isPending}
                        title={action.description}
                        className={`flex items-center gap-1.5 px-2 py-1 text-xs rounded border transition-colors ${
                          action.dangerous
                            ? "border-amber-500/30 bg-amber-500/10 text-amber-400 hover:bg-amber-500/20"
                            : "border-white/10 bg-white/5 text-slate-300 hover:bg-white/10"
                        }`}
                      >
                        {executeMutation.isPending && executeMutation.variables?.actionId === action.id ? (
                          <Loader2 size={12} className="animate-spin" />
                        ) : (
                          <Icon size={12} />
                        )}
                        {action.name}
                      </button>
                    );
                  })}
                </div>
              )}

              {/* Unavailable actions - subdued with tooltips explaining why */}
              {unavailableActions.length > 0 && (
                <div className={`flex flex-wrap gap-2 ${availableActions.length > 0 ? "mt-2 pt-2 border-t border-white/5" : ""}`}>
                  {unavailableActions.map((action) => {
                    const Icon = actionIcons[action.id] || Play;
                    const reason = getUnavailableReason(action.id, data.actions);
                    return (
                      <div
                        key={action.id}
                        title={reason}
                        className="flex items-center gap-1.5 px-2 py-1 text-xs rounded border border-white/5 bg-white/[0.02] text-slate-600 cursor-not-allowed group relative"
                      >
                        <Icon size={12} />
                        {action.name}
                        <Info size={10} className="ml-0.5 opacity-50" />
                      </div>
                    );
                  })}
                </div>
              )}

              {/* Confirmation dialog */}
              {confirmAction && (
                <div className="mt-3 p-3 rounded-lg bg-amber-500/10 border border-amber-500/20">
                  <div className="flex items-start gap-2">
                    <AlertTriangle size={16} className="text-amber-400 mt-0.5 flex-shrink-0" />
                    <div className="flex-1">
                      <p className="text-sm text-amber-300 font-medium">Confirm Action</p>
                      <p className="text-xs text-amber-200/80 mt-1">
                        Are you sure you want to {confirmAction.name.toLowerCase()} this resource?
                        {confirmAction.id === "stop" && " This will cause downtime."}
                        {confirmAction.id === "restart" && " This will cause brief downtime."}
                      </p>
                      <div className="flex gap-2 mt-3">
                        <button
                          onClick={handleConfirm}
                          disabled={executeMutation.isPending}
                          className="px-3 py-1 text-xs font-medium rounded bg-amber-500 text-black hover:bg-amber-400 disabled:opacity-50 transition-colors"
                        >
                          {executeMutation.isPending ? "Executing..." : "Confirm"}
                        </button>
                        <button
                          onClick={() => setConfirmAction(null)}
                          disabled={executeMutation.isPending}
                          className="px-3 py-1 text-xs rounded border border-white/10 text-slate-300 hover:bg-white/5 transition-colors"
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
                <div className={`mt-3 p-3 rounded-lg border ${
                  lastResult.success
                    ? "bg-emerald-500/10 border-emerald-500/20"
                    : "bg-red-500/10 border-red-500/20"
                }`}>
                  <div className="flex items-start gap-2">
                    {lastResult.success ? (
                      <CheckCircle2 size={16} className="text-emerald-400 mt-0.5 flex-shrink-0" />
                    ) : (
                      <XCircle size={16} className="text-red-400 mt-0.5 flex-shrink-0" />
                    )}
                    <div className="flex-1 min-w-0">
                      <p className={`text-sm font-medium ${lastResult.success ? "text-emerald-300" : "text-red-300"}`}>
                        {lastResult.message}
                      </p>
                      {lastResult.output && (
                        <pre className="mt-2 p-2 text-xs font-mono bg-black/30 rounded overflow-x-auto whitespace-pre-wrap max-h-32 overflow-y-auto text-slate-400">
                          {lastResult.output}
                        </pre>
                      )}
                      {lastResult.error && (
                        <p className="mt-1 text-xs text-red-400">{lastResult.error}</p>
                      )}
                      <button
                        onClick={() => setLastResult(null)}
                        className="mt-2 text-xs text-slate-500 hover:text-slate-300"
                      >
                        Dismiss
                      </button>
                    </div>
                  </div>
                </div>
              )}
            </>
          ) : (
            <p className="text-xs text-slate-500">No actions available</p>
          )}
        </div>
      )}
    </div>
  );
}
