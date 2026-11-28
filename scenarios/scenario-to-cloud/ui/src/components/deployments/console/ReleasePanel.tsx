import { compareRelease, shortDigest } from "../../../lib/consoleActions";
import type { HealthObservation } from "../../../lib/api";
import type { AuthzMatrix, CompiledPlanResponse } from "../../../types/console";
import { CopyButton, KeyValue, Panel, SkeletonRows, StatusPill } from "./ConsolePrimitives";
import { DeniedState } from "./DeniedState";
import { ApiError } from "../../../lib/apiErrors";

export interface ReleasePanelProps {
  deploymentId: string;
  plan: CompiledPlanResponse | null | undefined;
  planError: unknown;
  planLoading: boolean;
  observation: HealthObservation | null | undefined;
  matrix?: AuthzMatrix | null;
  target?: string;
}

/**
 * ReleasePanel shows the desired release (from the compiled plan) and the
 * observed release (from the health observation) side by side and marks a
 * mismatch. Desired is never derived from the deployment record.
 */
export function ReleasePanel({ deploymentId, plan, planError, planLoading, observation, matrix, target }: ReleasePanelProps) {
  const desired = plan?.plan.release_digest;
  const observed = observation?.observed_release_digest;
  const comparison = compareRelease(desired, observed);
  const configComparison = compareRelease(plan?.plan.configuration_digest, observation?.observed_configuration_digest);
  const denial = planError instanceof ApiError ? planError : null;
  const state = planLoading ? "loading" : denial && ["unauthenticated", "forbidden_scope", "forbidden_target", "forbidden_origin"].includes(denial.code) ? "denied" : comparison === "mismatch" ? "degraded" : "ready";

  return (
    <Panel
      testId="console-release"
      title="Release"
      state={state}
      description="Desired comes from the compiled plan; observed comes from the health observation."
      actions={
        comparison === "mismatch" ? (
          <StatusPill tone="warning" testId="console-release-mismatch">
            desired ≠ observed
          </StatusPill>
        ) : comparison === "match" ? (
          <StatusPill tone="success" testId="console-release-match">
            in sync
          </StatusPill>
        ) : (
          <StatusPill tone="neutral" testId="console-release-unknown">
            comparison unavailable
          </StatusPill>
        )
      }
    >
      {planLoading && <SkeletonRows rows={3} />}
      {state === "denied" && <DeniedState error={planError} matrix={matrix} method="POST" path={`/api/v1/deployments/${deploymentId}/plan`} target={target} autoFocus={false} testId="console-release-denied" />}
      {!planLoading && Boolean(planError) && state !== "denied" && (
        <p role="alert" className="text-sm text-amber-200" data-testid="console-release-plan-error">
          Desired release unavailable: {planError instanceof ApiError ? `${planError.code}: ${planError.message}` : String(planError)}
        </p>
      )}
      {!planLoading && (
        <div className="mt-2 grid grid-cols-1 gap-3 md:grid-cols-2">
          <div className="rounded-md border border-white/10 p-3">
            <p className="text-xs uppercase tracking-wide text-slate-400 mb-1">Desired</p>
            <dl className="space-y-1">
            <KeyValue label="Release" mono testId="console-release-desired">
              {desired ? (
                <>
                  <span title={desired}>{shortDigest(desired)}</span> <CopyButton value={desired} label="desired release digest" />
                </>
              ) : (
                <span className="font-sans text-slate-300">not compiled</span>
              )}
            </KeyValue>
            <KeyValue label="Configuration" mono>
              {plan?.plan.configuration_digest ? shortDigest(plan.plan.configuration_digest) : <span className="font-sans text-slate-300">unknown</span>}
            </KeyValue>
            <KeyValue label="Revision">{typeof plan?.plan.desired_revision === "number" ? plan.plan.desired_revision : "unknown"}</KeyValue>
            <KeyValue label="Plan outcome" testId="console-release-outcome">
              {plan?.plan.outcome ?? "unknown"}
            </KeyValue>
            </dl>
          </div>
          <div className="rounded-md border border-white/10 p-3">
            <p className="text-xs uppercase tracking-wide text-slate-400 mb-1">Observed</p>
            <dl className="space-y-1">
            <KeyValue label="Release" mono testId="console-release-observed">
              {observed ? (
                <>
                  <span title={observed}>{shortDigest(observed)}</span> <CopyButton value={observed} label="observed release digest" />
                </>
              ) : (
                <span className="font-sans text-slate-300">not observed</span>
              )}
            </KeyValue>
            <KeyValue label="Configuration" mono>
              {observation?.observed_configuration_digest ? shortDigest(observation.observed_configuration_digest) : <span className="font-sans text-slate-300">unknown</span>}
              {configComparison === "mismatch" && (
                <span className="ml-2 font-sans text-amber-200" data-testid="console-release-config-mismatch">
                  differs from desired
                </span>
              )}
            </KeyValue>
            <KeyValue label="Target" mono>
              {observation?.target_id ?? <span className="font-sans text-slate-300">unknown</span>}
            </KeyValue>
            </dl>
          </div>
        </div>
      )}
    </Panel>
  );
}
