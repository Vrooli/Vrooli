import { useEffect, useRef, useState } from "react";
import { ApiError, ErrorCodes } from "../../../lib/apiErrors";
import { applyActionFor, shortDigest } from "../../../lib/consoleActions";
import type { AuthzMatrix, CompiledPlanResponse } from "../../../types/console";
import { Collapsible } from "../../ui/collapsible";
import { ActionButton, ActivityIndicator, KeyValue, LiveRegion, Panel, SkeletonRows, StatusPill } from "./ConsolePrimitives";
import { DeniedState } from "./DeniedState";
import { DestructiveActionDialog } from "./DestructiveActionDialog";
import { NextActionControl } from "./NextActionControl";

export interface PlanReviewProps {
  deploymentId: string;
  target: string;
  plan: CompiledPlanResponse | null | undefined;
  loading: boolean;
  error: unknown;
  matrix?: AuthzMatrix | null;
  onApply: (planDigest: string) => Promise<unknown>;
  applyPending: boolean;
  applyError: unknown;
  onReplan: () => void;
  /** Skip the typed confirmation (wizard first deploy of a fresh record). */
  confirmBeforeApply?: boolean;
}

const AUTHORITY_CODES = ["unauthenticated", "forbidden_scope", "forbidden_target", "forbidden_origin"];

/**
 * PlanReview renders the compiled plan preview (changes, data effects,
 * downtime, recovery strategy, collapsed shell preview) and the apply action
 * whose availability comes only from plan.outcome. A needs_input plan renders
 * the onboarding handoff instead of any inline secret prompt.
 */
export function PlanReview({ deploymentId, target, plan, loading, error, matrix, onApply, applyPending, applyError, onReplan, confirmBeforeApply = true }: PlanReviewProps) {
  const [confirming, setConfirming] = useState(false);
  const handoffRef = useRef<HTMLDivElement>(null);
  const denied = error instanceof ApiError && AUTHORITY_CODES.includes(error.code);
  const applyDenied = applyError instanceof ApiError && AUTHORITY_CODES.includes(applyError.code);
  const applyRefusal = applyError instanceof ApiError && !applyDenied ? applyError : null;
  const outcome = plan?.plan.outcome;
  const handoff = plan?.plan.handoff ?? plan?.preview.handoff;
  const action = applyActionFor(outcome, "scenario-to-cloud:destructive");
  const state = loading ? "loading" : denied ? "denied" : !plan ? "empty" : outcome === "needs_input" ? "needs-input" : outcome === "no_op" ? "no-op" : "ready";
  const dataEffects = plan?.preview.data_effects ?? [];

  useEffect(() => {
    if (state === "needs-input") handoffRef.current?.querySelector<HTMLElement>("a,button")?.focus();
  }, [state]);

  const stale = applyRefusal && (applyRefusal.code === ErrorCodes.planStale || applyRefusal.code === ErrorCodes.planDigestMismatch);
  const announcement = plan ? `Plan reviewed: outcome ${outcome}` : "";

  async function confirmApply() {
    if (!plan) return;
    await onApply(plan.plan_digest);
    setConfirming(false);
  }

  return (
    <Panel
      testId="console-review"
      title={plan?.plan.presentation.title || "Plan review"}
      state={state}
      description={plan?.plan.presentation.summary}
      actions={
        plan ? (
          <StatusPill tone={outcome === "apply" ? "info" : outcome === "no_op" ? "success" : "warning"} testId="console-review-outcome">
            {outcome}
          </StatusPill>
        ) : null
      }
    >
      <LiveRegion message={announcement} testId="console-review-live" />
      {loading && <SkeletonRows rows={5} />}
      {denied && <DeniedState error={error} matrix={matrix} method="POST" path={`/api/v1/deployments/${deploymentId}/plan`} target={target} testId="console-review-denied" />}
      {!loading && !denied && Boolean(error) && (
        <div role="alert" className="space-y-2 text-sm text-amber-200" data-testid="console-review-error">
          <p>Plan unavailable: {error instanceof ApiError ? `${error.code}: ${error.message}` : String(error)}</p>
          {error instanceof ApiError && <NextActionControl next={error.nextAction} testId="console-review-error-next-action" />}
          <ActionButton available tone="secondary" testId="console-review-retry" onClick={onReplan}>
            Compile again
          </ActionButton>
        </div>
      )}
      {plan && (
        <div className="space-y-4">
          <dl className="grid grid-cols-1 gap-1 md:grid-cols-2">
            <KeyValue label="Target" mono>
              {plan.preview.target}
            </KeyValue>
            <KeyValue label="Release" mono>
              {shortDigest(plan.plan.release_digest) || "unknown"}
            </KeyValue>
            <KeyValue label="Plan digest" mono testId="console-review-digest">
              {plan.plan_digest}
            </KeyValue>
            <KeyValue label="Downtime" testId="console-review-downtime">
              {plan.preview.downtime.expected_seconds > 0 ? `${plan.preview.downtime.expected_seconds}s (${plan.preview.downtime.reason || "unspecified"})` : "none expected"}
            </KeyValue>
            <KeyValue label="Recovery" testId="console-review-recovery">
              {plan.preview.recovery_strategy}
            </KeyValue>
            {plan.closure_status && (
              <KeyValue label="Closure">
                {plan.closure_status}
              </KeyValue>
            )}
          </dl>
          {plan.plan.presentation.downtime_note && <p className="text-sm text-amber-100">{plan.plan.presentation.downtime_note}</p>}
          {plan.plan.presentation.recovery_note && <p className="text-sm text-slate-200">{plan.plan.presentation.recovery_note}</p>}

          {handoff && (
            <div ref={handoffRef} className="rounded-md border border-blue-400/40 bg-blue-500/10 p-3 space-y-2" data-testid="console-review-handoff">
              <p className="text-sm text-blue-100">
                Missing inputs: <span className="font-mono">{handoff.missing.join(", ") || "unnamed"}</span>. Secrets are supplied in onboarding, never here.
              </p>
              <NextActionControl next={{ owner: handoff.owner, kind: handoff.kind, reference: handoff.reference, label: "Finish setup in onboarding" }} testId="console-review-handoff-link" />
            </div>
          )}

          <div>
            <h3 className="text-xs uppercase tracking-wide text-slate-400 mb-1">Changes ({plan.preview.changes.length})</h3>
            {plan.preview.changes.length === 0 ? (
              <p className="text-sm text-slate-200" data-testid="console-review-no-changes">
                No changes.
              </p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-xs" data-testid="console-review-changes">
                  <thead>
                    <tr className="text-left text-slate-400">
                      <th scope="col" className="py-1 pr-3">Operation</th>
                      <th scope="col" className="py-1 pr-3">Effect</th>
                      <th scope="col" className="py-1 pr-3">Summary</th>
                      <th scope="col" className="py-1 pr-3">Verification</th>
                      <th scope="col" className="py-1 pr-3">Recovery</th>
                      <th scope="col" className="py-1 pr-3">Retry</th>
                      <th scope="col" className="py-1">Cancel point</th>
                    </tr>
                  </thead>
                  <tbody>
                    {plan.preview.changes.map((change) => (
                      <tr key={change.action_id} className="border-t border-white/10 align-top text-slate-100">
                        <td className="py-1 pr-3 font-mono">{change.operation}</td>
                        <td className="py-1 pr-3">{change.effect}</td>
                        <td className="py-1 pr-3">{change.summary}</td>
                        <td className="py-1 pr-3">{change.verification}</td>
                        <td className="py-1 pr-3">{change.recovery}</td>
                        <td className="py-1 pr-3">{change.retry}</td>
                        <td className="py-1">{change.cancel_point ? "yes" : "no"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          <div>
            <h3 className="text-xs uppercase tracking-wide text-slate-400 mb-1">Data effects ({dataEffects.length})</h3>
            {dataEffects.length === 0 ? (
              <p className="text-sm text-slate-200" data-testid="console-review-no-data-effects">
                No declared data is touched.
              </p>
            ) : (
              <ul className="space-y-0.5 text-sm" data-testid="console-review-data-effects">
                {dataEffects.map((effect) => (
                  <li key={`${effect.action_id}-${effect.subject}`} className="flex flex-wrap gap-2">
                    <span className="font-mono text-slate-100">{effect.subject}</span>
                    <span className="text-slate-300">{effect.effect}</span>
                  </li>
                ))}
              </ul>
            )}
          </div>

          {plan.preview.shell_preview.length > 0 && (
            <Collapsible title="Show equivalent commands">
              <pre className="mt-2 overflow-x-auto rounded bg-slate-950 p-3 text-xs text-slate-200" data-testid="console-review-shell">
                {plan.preview.shell_preview.map((line) => `# ${line.action_id}\n${line.command}`).join("\n")}
              </pre>
            </Collapsible>
          )}

          {applyDenied && <DeniedState error={applyError} matrix={matrix} method="POST" path={`/api/v1/deployments/${deploymentId}/plan/apply`} target={target} testId="console-review-apply-denied" />}
          {applyRefusal && (
            <div role="alert" className="rounded-md border border-amber-500/40 bg-amber-500/10 p-3 text-sm text-amber-100 space-y-2" data-testid="console-review-apply-refusal">
              <p>
                {applyRefusal.code}: {applyRefusal.message}
              </p>
              {applyRefusal.code === ErrorCodes.needsInput && applyRefusal.nextAction && (
                <NextActionControl next={applyRefusal.nextAction} testId="console-review-apply-handoff" />
              )}
              {stale && (
                <ActionButton available tone="secondary" testId="console-review-replan" onClick={onReplan}>
                  Review the plan again
                </ActionButton>
              )}
              {!stale && applyRefusal.code !== ErrorCodes.needsInput && <NextActionControl next={applyRefusal.nextAction} handlers={{ onReplan }} testId="console-review-apply-next-action" />}
            </div>
          )}

          <div className="flex flex-wrap items-center gap-3">
            <ActionButton
              available={action.available && !applyPending}
              reason={action.reason?.message}
              testId="console-review-apply"
              onClick={() => (confirmBeforeApply ? setConfirming(true) : void confirmApply())}
            >
              {applyPending ? <ActivityIndicator label="Applying" /> : action.label}
            </ActionButton>
            <ActionButton available={!applyPending} tone="secondary" testId="console-review-recompile" onClick={onReplan}>
              Compile again
            </ActionButton>
          </div>
        </div>
      )}
      {confirming && plan && (
        <DestructiveActionDialog
          title="Apply this plan?"
          description={`${plan.preview.changes.length} change(s) will be applied on the target as durable operation steps.`}
          target={plan.preview.target || target}
          affectedData={dataEffects.map((effect) => `${effect.subject} (${effect.effect})`)}
          details={[
            { label: "Downtime", value: plan.preview.downtime.expected_seconds > 0 ? `${plan.preview.downtime.expected_seconds}s` : "none expected" },
            { label: "Recovery", value: plan.preview.recovery_strategy },
            { label: "Plan digest", value: <span className="font-mono text-xs break-all">{plan.plan_digest}</span> },
          ]}
          confirmText={deploymentId.slice(0, 8)}
          confirmLabel="Apply plan"
          isPending={applyPending}
          error={applyError instanceof Error ? applyError.message : null}
          onConfirm={confirmApply}
          onCancel={() => setConfirming(false)}
        />
      )}
    </Panel>
  );
}
