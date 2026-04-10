import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Shield, AlertTriangle, RotateCcw } from "lucide-react";
import { fetchRecoveryState, triggerRecovery, resetCircuit, type RecoveryState } from "../lib/api";
import { timeAgo } from "../lib/utils";
import { RecoveryTimeline } from "../components/RecoveryTimeline";
import { Button } from "../components/ui/button";
import { Tooltip } from "../components/ui/tooltip";
import { StatusBadge } from "../components/ui/status-badge";
import { ConfirmDialog } from "../components/ui/confirm-dialog";
import { RefreshButton } from "../components/ui/refresh-button";
import { QueryState } from "../components/ui/query-state";

function RecoveryStatePanel() {
  const queryClient = useQueryClient();
  const { data, isLoading, error, refetch, isFetching } = useQuery({
    queryKey: ["recovery-state"],
    queryFn: fetchRecoveryState,
    refetchInterval: 15000,
  });

  const triggerMutation = useMutation({
    mutationFn: (force: boolean) => triggerRecovery(force),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["recovery-state"] });
      queryClient.invalidateQueries({ queryKey: ["recovery-events"] });
    },
  });

  const resetMutation = useMutation({
    mutationFn: resetCircuit,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["recovery-state"] });
    },
  });

  return (
    <div className="rounded-xl border border-white/10 bg-white/5 p-4 sm:p-6" data-testid="recovery-state-panel">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Shield className="h-5 w-5 text-slate-300" aria-hidden="true" />
          <h2 className="text-lg font-semibold">Recovery State</h2>
        </div>
        <RefreshButton onClick={() => refetch()} disabled={isFetching} aria-label="Refresh recovery state" />
      </div>

      <QueryState
        isLoading={isLoading}
        error={error}
        refetch={refetch}
        loadingLabel="Loading recovery state..."
        errorLabel="Failed to load recovery state."
        skeleton={
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 lg:grid-cols-4">
            {[1, 2, 3, 4].map((i) => <div key={i} className="h-16 rounded-lg bg-white/5" />)}
          </div>
        }
      >
        {data && <RecoveryStateDetails state={data} onTrigger={(force) => triggerMutation.mutate(force)} onReset={() => resetMutation.mutate()} isPending={triggerMutation.isPending || resetMutation.isPending} />}
      </QueryState>
    </div>
  );
}

function RecoveryStateDetails({ state, onTrigger, onReset, isPending }: { state: RecoveryState; onTrigger: (force: boolean) => void; onReset: () => void; isPending: boolean }) {
  const [showTriggerConfirm, setShowTriggerConfirm] = useState(false);
  const [showResetConfirm, setShowResetConfirm] = useState(false);

  return (
    <div className="mt-4 space-y-4">
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-sm lg:grid-cols-4">
        <div className="rounded-lg bg-black/20 p-3">
          <p className="text-slate-300">Status</p>
          <p className="font-medium text-slate-100" data-testid="recovery-status-value">{state.status}</p>
        </div>
        <div className="rounded-lg bg-black/20 p-3">
          <Tooltip content="When open, recovery attempts are paused to prevent rapid-fire restarts. Closes automatically after a cooldown period.">
            <p className="text-slate-300 cursor-help border-b border-dotted border-slate-600">Circuit Breaker</p>
          </Tooltip>
          <StatusBadge
            variant={state.circuit_open ? "error" : "success"}
            label={state.circuit_open ? "OPEN" : "CLOSED"}
            className="mt-1"
            data-testid="recovery-circuit-value"
          />
        </div>
        <div className="rounded-lg bg-black/20 p-3">
          <Tooltip content="How many consecutive recovery attempts have failed in a row">
            <p className="text-slate-300 cursor-help border-b border-dotted border-slate-600">Consecutive Failures</p>
          </Tooltip>
          <p className="font-medium text-slate-100">{state.consecutive_failures}</p>
        </div>
        <div className="rounded-lg bg-black/20 p-3">
          <Tooltip content="Number of times the system has backed off before retrying recovery">
            <p className="text-slate-300 cursor-help border-b border-dotted border-slate-600">Backoff Retries</p>
          </Tooltip>
          <p className="font-medium text-slate-100">{state.backoff_retries}</p>
        </div>
      </div>

      {state.last_recovery_at && (
        <Tooltip content={new Date(state.last_recovery_at).toLocaleString()}>
          <p className="text-sm text-slate-300 cursor-help">
            Last recovery: {timeAgo(state.last_recovery_at)}
          </p>
        </Tooltip>
      )}

      <div className="flex flex-wrap gap-2">
        <Button variant="outline" size="sm" onClick={() => setShowTriggerConfirm(true)} disabled={isPending} data-testid="recovery-trigger-button">
          <RotateCcw className="h-4 w-4 mr-2" aria-hidden="true" />
          Trigger Recovery
        </Button>
        {state.circuit_open && (
          <Button variant="outline" size="sm" onClick={() => setShowResetConfirm(true)} disabled={isPending} className="text-yellow-400 border-yellow-400/30" data-testid="recovery-reset-circuit">
            <AlertTriangle className="h-4 w-4 mr-2" aria-hidden="true" />
            Reset Circuit
          </Button>
        )}
      </div>

      <ConfirmDialog
        open={showTriggerConfirm}
        title="Trigger Recovery"
        description="This will restart the cloudflared tunnel service. Active connections may be briefly interrupted. Are you sure?"
        confirmLabel="Trigger"
        cancelLabel="Cancel"
        variant="danger"
        isPending={isPending}
        onConfirm={() => {
          onTrigger(false);
          setShowTriggerConfirm(false);
        }}
        onCancel={() => setShowTriggerConfirm(false)}
      />
      <ConfirmDialog
        open={showResetConfirm}
        title="Reset Circuit Breaker"
        description="This will close the circuit breaker and allow recovery attempts to resume immediately. Only use this if you believe the underlying issue has been resolved."
        confirmLabel="Reset"
        cancelLabel="Cancel"
        variant="danger"
        isPending={isPending}
        onConfirm={() => {
          onReset();
          setShowResetConfirm(false);
        }}
        onCancel={() => setShowResetConfirm(false)}
      />
    </div>
  );
}

export default function RecoveryPage() {
  return (
    <div className="space-y-4 sm:space-y-6">
      <div>
        <h1 className="text-lg sm:text-xl font-semibold">Auto-Recovery</h1>
        <p className="text-sm text-slate-300">Monitor and control tunnel self-healing. Trigger manual recovery or reset the circuit breaker.</p>
      </div>
      <RecoveryStatePanel />
      <RecoveryTimeline />
    </div>
  );
}
