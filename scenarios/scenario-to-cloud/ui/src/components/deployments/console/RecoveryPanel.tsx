import { useState } from "react";
import { ApiError } from "../../../lib/apiErrors";
import { formatAge, shortDigest } from "../../../lib/consoleActions";
import type { AuthzMatrix, CompiledPlanResponse, PlanHandoff, RecoveryPoint, RollbackVerdict } from "../../../types/console";
import { ActionButton, KeyValue, Panel, SkeletonRows, StatusPill } from "./ConsolePrimitives";
import { DeniedState } from "./DeniedState";
import { DestructiveActionDialog } from "./DestructiveActionDialog";
import { NextActionControl } from "./NextActionControl";

export interface RecoveryPanelProps {
  deploymentId: string;
  target: string;
  recoveryPoints: RecoveryPoint[] | null | undefined;
  loading: boolean;
  error: unknown;
  plan?: CompiledPlanResponse | null;
  matrix?: AuthzMatrix | null;
  headingRef?: React.Ref<HTMLHeadingElement>;
  onCheckRollback?: (input: { recoveryPointId: string; currentSchema: string; targetSchema: string }) => Promise<RollbackVerdict>;
  onRestore?: (input: { recoveryPointId: string; into: Record<string, string> }) => Promise<unknown>;
  restorePending?: boolean;
}

const AUTHORITY_CODES = ["unauthenticated", "forbidden_scope", "forbidden_target", "forbidden_origin"];

/**
 * RecoveryPanel answers "what is the safe recovery or next action?": the
 * resume handoff when the plan needs input, rollback eligibility from the
 * API verdict, and the recovery points with a restore that always goes
 * through the destructive preview naming the affected bindings and target.
 */
export function RecoveryPanel({
  deploymentId,
  target,
  recoveryPoints,
  loading,
  error,
  plan,
  matrix,
  headingRef,
  onCheckRollback,
  onRestore,
  restorePending = false,
}: RecoveryPanelProps) {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [verdicts, setVerdicts] = useState<Record<string, RollbackVerdict | { refusal: ApiError }>>({});
  const [checking, setChecking] = useState<string | null>(null);
  const [restoreTarget, setRestoreTarget] = useState<RecoveryPoint | null>(null);
  const [restoreError, setRestoreError] = useState<unknown>(null);

  const handoff: PlanHandoff | undefined = plan?.plan.handoff ?? plan?.preview.handoff;
  const denied = error instanceof ApiError && AUTHORITY_CODES.includes(error.code);
  const points = recoveryPoints ?? [];
  const state = loading ? "loading" : denied ? "denied" : handoff ? "needs-input" : points.length === 0 ? "empty" : "ready";
  const newest = points[0];
  const selected = points.find((p) => p.id === selectedId) ?? null;

  async function checkRollback(point: RecoveryPoint) {
    if (!onCheckRollback || !newest) return;
    setChecking(point.id);
    try {
      const verdict = await onCheckRollback({ recoveryPointId: point.id, currentSchema: newest.schema_version, targetSchema: point.schema_version });
      setVerdicts((prev) => ({ ...prev, [point.id]: verdict }));
    } catch (err) {
      if (err instanceof ApiError) {
        setVerdicts((prev) => ({ ...prev, [point.id]: { refusal: err } }));
      }
    } finally {
      setChecking(null);
    }
  }

  return (
    <Panel testId="console-recovery" title="Recovery" state={state} headingRef={headingRef} description="Rollback eligibility and restores are decided by the API; every restore is previewed first.">
      {loading && <SkeletonRows rows={3} />}
      {denied && <DeniedState error={error} matrix={matrix} method="GET" path={`/api/v1/deployments/${deploymentId}/recovery-points`} target={target} autoFocus={false} testId="console-recovery-denied" />}
      {!loading && !denied && Boolean(error) && (
        <p role="alert" className="text-sm text-amber-200" data-testid="console-recovery-error">
          Recovery points unavailable: {error instanceof ApiError ? `${error.code}: ${error.message}` : String(error)}
        </p>
      )}
      {handoff && (
        <div className="mb-3 rounded-md border border-blue-400/40 bg-blue-500/10 p-3 space-y-2" data-testid="console-recovery-handoff">
          <p className="text-sm text-blue-100">
            The plan needs input before it can continue. Missing: <span className="font-mono">{handoff.missing.join(", ") || "unnamed inputs"}</span>.
          </p>
          <NextActionControl next={{ owner: handoff.owner, kind: handoff.kind, reference: handoff.reference, label: "Finish setup in onboarding" }} testId="console-recovery-handoff-link" />
        </div>
      )}
      {!loading && !denied && points.length === 0 && !error && (
        <p className="text-sm text-slate-200" data-testid="console-recovery-empty">
          No recovery points have been captured for this deployment.
        </p>
      )}
      {points.length > 0 && (
        <ul className="space-y-2" aria-label="Recovery points" data-testid="console-recovery-points">
          {points.map((point) => {
            const verdict = verdicts[point.id];
            const isSelected = selectedId === point.id;
            return (
              <li key={point.id} data-testid="console-recovery-point" data-selected={isSelected} className="rounded-md border border-white/10 p-3">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <button
                    type="button"
                    onClick={() => setSelectedId(isSelected ? null : point.id)}
                    aria-pressed={isSelected}
                    className="text-left font-mono text-xs text-slate-100 hover:text-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400 rounded"
                  >
                    {point.id}
                  </button>
                  <div className="flex flex-wrap items-center gap-1">
                    <StatusPill tone={point.encrypted ? "success" : "warning"}>{point.encrypted ? "encrypted" : "not encrypted"}</StatusPill>
                    {point.protected && <StatusPill tone="info">protected</StatusPill>}
                    <StatusPill tone="neutral">{formatAge(point.captured_at) || point.captured_at}</StatusPill>
                  </div>
                </div>
                <dl className="mt-2 space-y-0.5">
                  <KeyValue label="Release" mono>
                    {shortDigest(point.release_digest) || "unknown"}
                  </KeyValue>
                  <KeyValue label="Schema" mono>
                    {point.schema_version || "unknown"}
                  </KeyValue>
                  <KeyValue label="Bindings" mono>
                    {point.binding_ids.join(", ") || "none"}
                  </KeyValue>
                </dl>
                {isSelected && (
                  <div className="mt-3 space-y-2" data-testid="console-recovery-selected">
                    <div className="flex flex-wrap gap-2">
                      {onCheckRollback && (
                        <ActionButton available={checking !== point.id} tone="secondary" testId="console-recovery-check-rollback" onClick={() => checkRollback(point)}>
                          {checking === point.id ? "Checking eligibility" : "Check rollback eligibility"}
                        </ActionButton>
                      )}
                      {onRestore && (
                        <ActionButton
                          available={!restorePending && (!verdict || ("compatible" in verdict && verdict.compatible))}
                          reason={verdict && "compatible" in verdict && !verdict.compatible ? `Not eligible: ${verdict.reason ?? verdict.reason_code ?? "incompatible"}` : verdict && "refusal" in verdict ? `${verdict.refusal.code}: ${verdict.refusal.message}` : undefined}
                          tone="danger"
                          testId="console-recovery-restore"
                          onClick={() => {
                            setRestoreError(null);
                            setRestoreTarget(point);
                          }}
                        >
                          Restore this recovery point
                        </ActionButton>
                      )}
                    </div>
                    <p className="text-xs text-slate-300">Eligibility compares this point's schema with the schema of the most recent recovery point.</p>
                    {verdict && "compatible" in verdict && (
                      <div data-testid="console-recovery-verdict" data-compatible={verdict.compatible} className={verdict.compatible ? "text-sm text-emerald-200" : "text-sm text-amber-200"}>
                        {verdict.compatible ? `Eligible (${verdict.schema_strategy})` : `Not eligible: ${verdict.reason ?? verdict.reason_code ?? "incompatible"}`}
                        {verdict.plan && (
                          <ul className="mt-1 list-disc pl-5 text-xs text-slate-200">
                            {verdict.plan.steps.map((step) => (
                              <li key={step}>{step}</li>
                            ))}
                          </ul>
                        )}
                      </div>
                    )}
                    {verdict && "refusal" in verdict && (
                      <DeniedState error={verdict.refusal} matrix={matrix} method="POST" path={`/api/v1/deployments/${deploymentId}/recovery-points/rollback-admission`} target={target} autoFocus={false} testId="console-recovery-verdict-denied" />
                    )}
                    {verdict && "refusal" in verdict && !AUTHORITY_CODES.includes(verdict.refusal.code) && (
                      <div role="alert" className="text-sm text-amber-200" data-testid="console-recovery-verdict-refusal">
                        {verdict.refusal.code}: {verdict.refusal.message}
                        <div className="mt-1">
                          <NextActionControl next={verdict.refusal.nextAction} testId="console-recovery-verdict-next-action" />
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </li>
            );
          })}
        </ul>
      )}
      {restoreError instanceof ApiError && !restoreTarget && (
        <div className="mt-3">
          <DeniedState error={restoreError} matrix={matrix} method="POST" path={`/api/v1/deployments/${deploymentId}/recovery-points/${selected?.id ?? "rp"}/restore`} target={target} testId="console-recovery-restore-denied" />
          {!AUTHORITY_CODES.includes(restoreError.code) && (
            <p role="alert" className="text-sm text-amber-200" data-testid="console-recovery-restore-refusal">
              {restoreError.code}: {restoreError.message}
            </p>
          )}
        </div>
      )}
      {restoreTarget && onRestore && (
        <DestructiveActionDialog
          title="Restore a recovery point?"
          description="Restoring replaces the data in the named bindings with the captured contents. The API refuses when a binding is not clean."
          target={target}
          affectedData={restoreTarget.bindings.length > 0 ? restoreTarget.bindings.map((b) => `${b.id ?? "binding"} (${b.kind ?? "unknown"}) → ${b.locator ?? "declared locator"}`) : restoreTarget.binding_ids}
          details={[
            { label: "Recovery point", value: <span className="font-mono text-xs">{restoreTarget.id}</span> },
            { label: "Captured", value: formatAge(restoreTarget.captured_at) || restoreTarget.captured_at },
            { label: "Release", value: <span className="font-mono text-xs">{shortDigest(restoreTarget.release_digest) || "unknown"}</span> },
          ]}
          confirmText={deploymentId.slice(0, 8)}
          confirmLabel="Restore"
          isPending={restorePending}
          error={restoreError instanceof Error ? restoreError.message : null}
          onConfirm={async () => {
            const into: Record<string, string> = {};
            for (const binding of restoreTarget.bindings) {
              if (binding.id) into[binding.id] = typeof binding.locator === "string" ? binding.locator : "";
            }
            for (const id of restoreTarget.binding_ids) {
              if (!(id in into)) into[id] = "";
            }
            try {
              await onRestore({ recoveryPointId: restoreTarget.id, into });
              setRestoreTarget(null);
            } catch (err) {
              setRestoreError(err);
              if (err instanceof ApiError && AUTHORITY_CODES.includes(err.code)) setRestoreTarget(null);
            }
          }}
          onCancel={() => setRestoreTarget(null)}
        />
      )}
    </Panel>
  );
}
