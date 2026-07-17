/**
 * WorkflowBindingsPanel — operator controls for operation→mode bindings.
 *
 * Renders every operation resolvable for a target (GetResolvedBindings):
 * winning mode + EXACT revision, source label, and the full contribution
 * ladder (system < initiative < item) so an item override visibly shadows an
 * initiative override. Per-operation actions: "Override" (PutBindingOverride,
 * offered modes restricted to the server's ListCompatibleModes verdicts) and
 * "Reset to inherited" (DeleteBindingOverride, confirm-gated) when an
 * override document exists at THIS owner's layer.
 *
 * The server is authoritative for precedence and compatibility — this panel
 * only displays typed results. The single client-side display rule is the
 * stale-revision indicator: a strict equality check between an override's
 * pinned mode revision and the mode's current revision from
 * ListCompatibleModes ("pinned to older revision"), no further judgement.
 *
 * Binding resolution is snapshot-at-invoke: changes here apply only to
 * operations started after the change (surfaced in the panel copy).
 */

import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { History, Loader2, RotateCcw, Settings2 } from "lucide-react";
import { Button } from "../ui/button";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { selectors } from "../../consts/selectors";
import { agentOperationsService } from "../../services";
import {
  agentOpsKeys,
  useBindingOverridesQuery,
  useCompatibleModesQuery,
  useResolvedBindingsQuery,
} from "../../hooks/useAgentOpsQueries";
import { formatRelativeTime } from "../../lib";
import type {
  AgentOpsTarget,
  WorkflowBindingContribution,
  WorkflowBindingLayer,
  WorkflowBindingOverrideDocument,
  WorkflowCompatibleMode,
  WorkflowResolvedBinding,
} from "../../types/agent-operations";
import { WORKFLOW_BINDING_LAYER_LABELS } from "../../types/agent-operations";
import { BindingOverrideDialog } from "./binding-override-dialog";

/** Display order of the binding ladder (lower renders first; higher wins). */
const LAYER_RANK: Record<WorkflowBindingLayer, number> = {
  unspecified: 0,
  "system-default": 1,
  "initiative-override": 2,
  "backlog-item-override": 3,
  "authorized-invocation": 4,
};

/** Find the override document stored at this owner's layer for an operation row. */
function ownOverrideFor(
  overrides: WorkflowBindingOverrideDocument[],
  row: WorkflowResolvedBinding,
): WorkflowBindingOverrideDocument | undefined {
  return overrides.find(
    (doc) =>
      doc.binding.operation === row.operation &&
      (doc.binding.operationVersion === "" ||
        row.operationVersion === "" ||
        doc.binding.operationVersion === row.operationVersion),
  );
}

/**
 * Equality-only stale check: the override pins a mode revision that no longer
 * matches the mode's current revision per the server's compatible-modes data.
 */
function staleRevisionInfo(
  override: WorkflowBindingOverrideDocument | undefined,
  compatibleModes: WorkflowCompatibleMode[],
): { pinned: string; current: string } | null {
  if (!override || override.binding.disabled) return null;
  const current = compatibleModes.find((mode) => mode.mode === override.binding.mode);
  if (!current) return null;
  if (!override.binding.modeRevision || !current.modeRevision) return null;
  if (override.binding.modeRevision === current.modeRevision) return null;
  return { pinned: override.binding.modeRevision, current: current.modeRevision };
}

export function WorkflowBindingsPanel({ target }: { target: AgentOpsTarget }) {
  const queryClient = useQueryClient();
  const resolvedQuery = useResolvedBindingsQuery(target);
  const overridesQuery = useBindingOverridesQuery(target);
  const compatibleQuery = useCompatibleModesQuery(target);

  const [overrideRow, setOverrideRow] = useState<WorkflowResolvedBinding | null>(null);
  const [resetRow, setResetRow] = useState<{
    row: WorkflowResolvedBinding;
    override: WorkflowBindingOverrideDocument;
  } | null>(null);

  const invalidateBindingQueries = () => {
    // An override at this owner changes resolution for every target beneath
    // it (initiative overrides feed item resolution), so invalidate the
    // resolved-bindings and overrides prefixes for ALL targets.
    void queryClient.invalidateQueries({ queryKey: agentOpsKeys.allResolvedBindings });
    void queryClient.invalidateQueries({ queryKey: agentOpsKeys.allBindingOverrides });
  };

  const putMutation = useMutation({
    mutationFn: (args: { operation: string; operationVersion: string; mode: WorkflowCompatibleMode }) =>
      agentOperationsService.putBindingOverride({
        owner: target,
        operation: args.operation,
        operationVersion: args.operationVersion,
        mode: args.mode.mode,
        modeRevision: args.mode.modeRevision,
      }),
    onSuccess: () => {
      invalidateBindingQueries();
      setOverrideRow(null);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (args: { operation: string; operationVersion: string }) =>
      agentOperationsService.deleteBindingOverride(target, args.operation, args.operationVersion),
    onSuccess: () => {
      invalidateBindingQueries();
      setResetRow(null);
    },
  });

  const rows = resolvedQuery.data ?? [];
  const overrides = overridesQuery.data ?? [];
  const compatibleModes = compatibleQuery.data ?? [];

  return (
    <div className="space-y-3" data-testid={selectors.workflowBindings.panel}>
      <p className="text-xs text-slate-500">
        Which mode implements each operation for this scope. Bindings are snapshotted when an
        operation starts — changes apply to operations started after this change.
      </p>

      {resolvedQuery.isLoading && (
        <p className="flex items-center gap-2 text-sm text-slate-400" role="status">
          <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
          Loading resolved bindings…
        </p>
      )}
      {Boolean(resolvedQuery.error) && (
        <p className="rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
          {resolvedQuery.error instanceof Error
            ? resolvedQuery.error.message
            : "Failed to load resolved bindings."}
        </p>
      )}
      {!resolvedQuery.isLoading && !resolvedQuery.error && rows.length === 0 && (
        <p className="text-sm italic text-slate-500">No operations are resolvable for this scope.</p>
      )}

      {rows.length > 0 && (
        <ul className="space-y-2">
          {rows.map((row) => {
            const override = ownOverrideFor(overrides, row);
            const stale = staleRevisionInfo(override, compatibleModes);
            return (
              <BindingRow
                key={`${row.operation}@${row.operationVersion}`}
                row={row}
                override={override}
                stale={stale}
                onOverride={() => setOverrideRow(row)}
                onReset={override ? () => setResetRow({ row, override }) : undefined}
              />
            );
          })}
        </ul>
      )}

      <BindingOverrideDialog
        isOpen={overrideRow !== null}
        onClose={() => {
          putMutation.reset();
          setOverrideRow(null);
        }}
        operation={overrideRow?.operation ?? ""}
        operationVersion={overrideRow?.operationVersion ?? ""}
        currentMode={overrideRow?.binding?.mode}
        compatibleModes={compatibleModes}
        modesLoading={compatibleQuery.isLoading}
        modesError={compatibleQuery.error ?? undefined}
        isMutating={putMutation.isPending}
        mutationError={putMutation.error ?? undefined}
        onConfirm={(mode) => {
          if (!overrideRow) return;
          putMutation.mutate({
            operation: overrideRow.operation,
            operationVersion: overrideRow.operationVersion,
            mode,
          });
        }}
      />

      <ConfirmDialog
        isOpen={resetRow !== null}
        onClose={() => {
          deleteMutation.reset();
          setResetRow(null);
        }}
        onConfirm={() => {
          if (!resetRow) return;
          deleteMutation.mutate({
            operation: resetRow.override.binding.operation,
            operationVersion: resetRow.override.binding.operationVersion,
          });
        }}
        title="Reset to inherited binding"
        description={`Remove the override for "${resetRow?.override.binding.operation ?? ""}" set at this scope. The operation falls back to the inherited binding. Applies to operations started after this change.`}
        confirmLabel="Reset to inherited"
        isLoading={deleteMutation.isPending}
        errorMessage={
          deleteMutation.error
            ? deleteMutation.error instanceof Error
              ? deleteMutation.error.message
              : "Failed to reset the override."
            : undefined
        }
        testIds={{
          dialog: selectors.workflowBindings.resetConfirmDialog,
          confirmButton: selectors.workflowBindings.resetConfirmButton,
        }}
      />
    </div>
  );
}

function BindingRow({
  row,
  override,
  stale,
  onOverride,
  onReset,
}: {
  row: WorkflowResolvedBinding;
  override?: WorkflowBindingOverrideDocument;
  stale: { pinned: string; current: string } | null;
  onOverride: () => void;
  onReset?: () => void;
}) {
  const winning = row.binding;
  const sortedContributions = [...row.contributions].sort(
    (a, b) => LAYER_RANK[a.binding.layer] - LAYER_RANK[b.binding.layer],
  );

  return (
    <li
      className="space-y-2 rounded-lg border border-slate-800 bg-slate-900/40 p-3"
      data-testid={selectors.workflowBindings.row}
    >
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0 space-y-1">
          <p className="font-mono text-sm text-slate-100">
            {row.operation}
            {row.operationVersion && (
              <span className="text-slate-500">@{row.operationVersion}</span>
            )}
          </p>
          {row.resolved && winning ? (
            <p className="text-xs text-slate-400">
              {winning.disabled ? (
                <span className="text-amber-300">Disabled (fail-closed veto)</span>
              ) : (
                <>
                  <span className="font-medium text-slate-200">{winning.mode}</span>
                  <span className="font-mono text-slate-500"> rev {winning.modeRevision || "—"}</span>
                </>
              )}
              <span
                className="ml-2 rounded-full border border-slate-700 bg-slate-950/60 px-2 py-0.5 text-[11px] text-slate-300"
                data-testid={selectors.workflowBindings.rowSource}
              >
                {WORKFLOW_BINDING_LAYER_LABELS[winning.layer]}
              </span>
            </p>
          ) : (
            <p
              className="text-xs text-amber-300"
              data-testid={selectors.workflowBindings.rowError}
            >
              Not resolved — {row.error || "unknown"}
              {row.errorMessage ? `: ${row.errorMessage}` : ""}
            </p>
          )}
        </div>
        <div className="flex shrink-0 items-center gap-1.5">
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={onOverride}
            data-testid={selectors.workflowBindings.overrideButton}
          >
            <Settings2 className="mr-1 h-3.5 w-3.5" aria-hidden="true" />
            Override
          </Button>
          {onReset && (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={onReset}
              data-testid={selectors.workflowBindings.resetButton}
            >
              <RotateCcw className="mr-1 h-3.5 w-3.5" aria-hidden="true" />
              Reset to inherited
            </Button>
          )}
        </div>
      </div>

      {sortedContributions.length > 1 && (
        <div className="flex flex-wrap items-center gap-1.5">
          {sortedContributions.map((contribution, index) => (
            <LayerChip key={`${contribution.binding.layer}-${index}`} contribution={contribution} />
          ))}
        </div>
      )}

      {override && (
        <p
          className="flex flex-wrap items-center gap-1.5 text-[11px] text-slate-500"
          data-testid={selectors.workflowBindings.overrideProvenance}
        >
          <History className="h-3 w-3" aria-hidden="true" />
          Override set here ({override.binding.ownerKind} {override.binding.ownerId})
          {override.revision && <span className="font-mono">rev {override.revision}</span>}
          {override.updatedAt && (
            <span title={override.updatedAt}>updated {formatRelativeTime(override.updatedAt)}</span>
          )}
          {stale && (
            <span
              className="rounded-full border border-amber-500/40 bg-amber-500/10 px-2 py-0.5 text-amber-300"
              data-testid={selectors.workflowBindings.staleRevision}
              title={`Pinned rev ${stale.pinned}; the mode's current rev is ${stale.current}.`}
            >
              Pinned to older revision ({stale.pinned} → {stale.current})
            </span>
          )}
        </p>
      )}
    </li>
  );
}

/**
 * One rung of the binding ladder. The winning layer is highlighted; layers it
 * shadows stay visible but dimmed, so "item override shadows initiative
 * override" is readable at a glance.
 */
function LayerChip({ contribution }: { contribution: WorkflowBindingContribution }) {
  const { binding, winning } = contribution;
  return (
    <span
      data-testid={selectors.workflowBindings.layerChip}
      data-winning={winning || undefined}
      className={`inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px] ${
        winning
          ? "border-cyan-500/50 bg-cyan-500/10 text-cyan-200"
          : "border-slate-800 bg-slate-950/50 text-slate-500"
      }`}
      title={
        winning
          ? `${WORKFLOW_BINDING_LAYER_LABELS[binding.layer]} — winning`
          : `${WORKFLOW_BINDING_LAYER_LABELS[binding.layer]} — shadowed by a higher layer`
      }
    >
      {WORKFLOW_BINDING_LAYER_LABELS[binding.layer]}
      <span className="font-mono">
        {binding.disabled ? "disabled" : binding.mode}
      </span>
      {!winning && <span className="italic">shadowed</span>}
    </span>
  );
}
