import { useCallback, useEffect, useRef, useState } from "react";
import { Rocket, CheckCircle2, ExternalLink, PartyPopper, Server } from "lucide-react";
import { Button } from "../ui/button";
import { Alert } from "../ui/alert";
import { Card, CardContent } from "../ui/card";
import { SpawnAgentButton } from "./SpawnAgentButton";
import { InvestigationProgress } from "./InvestigationProgress";
import { InvestigationReport } from "./InvestigationReport";
import { useDeploymentInvestigation } from "../../hooks/useInvestigation";
import { useDeployment as useDeploymentRecord } from "../../hooks/useDeployments";
import { useApplyPlan, useAuthzMatrix, useCancelOperation, useCompiledPlan, useDurableOperation, useOperationStanding } from "../../hooks/useConsole";
import { newRequestKey } from "../../lib/consoleApi";
import { isTerminalState, targetKey } from "../../lib/consoleActions";
import type { useDeployment } from "../../hooks/useDeployment";
import { OperationPanel, PlanReview } from "../deployments/console";

interface StepDeployProps {
  deployment: ReturnType<typeof useDeployment>;
  onViewDeployments?: () => void;
}

/**
 * StepDeploy runs the reviewed path: create the record, review the compiled
 * plan (changes, data effects, downtime, recovery, shell preview), apply it
 * with a durable request key, then follow the operation standing. A reload
 * lands back on the operation view from the durable pointer.
 */
export function StepDeploy({ deployment, onViewDeployments }: StepDeployProps) {
  const { deploymentStatus, deploymentError, deploymentId, deploy, parsedManifest, reset, onDeploymentComplete, onOperationAdmitted } = deployment;

  const isReview = deploymentStatus === "review";
  const isDeploying = deploymentStatus === "deploying";
  const isSuccess = deploymentStatus === "success";
  const isFailed = deploymentStatus === "failed";
  const domain = parsedManifest.ok ? parsedManifest.value.edge?.domain : null;

  const [showInvestigationReport, setShowInvestigationReport] = useState(false);
  const investigation = useDeploymentInvestigation(deploymentId);
  const deploymentRecord = useDeploymentRecord(deploymentId);
  const matrix = useAuthzMatrix();
  const plan = useCompiledPlan(deploymentId, { enabled: isReview || isFailed });
  const apply = useApplyPlan();
  const cancel = useCancelOperation();
  const durable = useDurableOperation(deploymentId);
  const standing = useOperationStanding(durable.pointer?.operation_id ?? null);
  const requestKeyRef = useRef<string | null>(null);
  const terminalNotified = useRef<string | null>(null);
  const target = targetKey(deploymentRecord.data?.target);

  useEffect(() => {
    const current = standing.data;
    if (!current) return;
    if ((current.terminal || isTerminalState(current.state)) && terminalNotified.current !== current.operation_id) {
      terminalNotified.current = current.operation_id;
      onDeploymentComplete(current.state === "succeeded", current.state === "succeeded" ? undefined : current.error?.message ?? `Operation ${current.state}`);
    }
  }, [standing.data, onDeploymentComplete]);

  const onApply = useCallback(
    async (planDigest: string) => {
      if (!deploymentId) return;
      if (!requestKeyRef.current) requestKeyRef.current = newRequestKey();
      const result = await apply.mutateAsync({ deploymentId, planDigest, requestKey: requestKeyRef.current });
      if (result.operation_id) {
        terminalNotified.current = null;
        durable.attach({ deployment_id: deploymentId, operation_id: result.operation_id, plan_digest: result.plan_digest });
        onOperationAdmitted();
      } else {
        onDeploymentComplete(true);
      }
    },
    [apply, deploymentId, durable, onOperationAdmitted, onDeploymentComplete],
  );

  const isInvestigationOutdated = (inv?: { created_at: string } | null) => {
    const lastDeployedAt = deploymentRecord.data?.last_deployed_at;
    if (!inv || !lastDeployedAt) return false;
    return new Date(inv.created_at).getTime() < new Date(lastDeployedAt).getTime();
  };

  return (
    <div className="space-y-6">
      {deploymentStatus === "idle" && (
        <div className="flex items-center gap-3">
          <Button onClick={deploy} disabled={!parsedManifest.ok} data-testid="deploy-deploy-button">
            <Rocket className="h-4 w-4 mr-1.5" aria-hidden="true" />
            Review deployment plan
          </Button>
        </div>
      )}

      {deploymentError && (isFailed || isReview) && (
        <Alert variant="error" title="Deployment failed">
          {deploymentError}
        </Alert>
      )}

      {(isReview || (isFailed && !durable.pointer)) && deploymentId && (
        <PlanReview
          deploymentId={deploymentId}
          target={target}
          plan={plan.data}
          loading={plan.isLoading}
          error={plan.error}
          matrix={matrix.data}
          onApply={onApply}
          applyPending={apply.isPending}
          applyError={apply.error}
          onReplan={() => {
            requestKeyRef.current = null;
            apply.reset();
            void plan.refetch();
          }}
          confirmBeforeApply={false}
        />
      )}

      {(isDeploying || (isFailed && durable.pointer)) && deploymentId && (
        <OperationPanel
          standing={standing.data}
          loading={durable.rehydrating || (standing.isLoading && Boolean(durable.pointer))}
          error={standing.error}
          resumed={durable.resumed}
          target={target}
          matrix={matrix.data}
          onCancel={(operationId) => cancel.mutateAsync(operationId)}
          cancelPending={cancel.isPending}
          cancelError={cancel.error}
          handlers={{
            onResume: () => {
              durable.detach();
              void deploy();
            },
            onReplan: () => {
              durable.detach();
              void deploy();
            },
          }}
        />
      )}

      {isFailed && deploymentId && (
        <div className="flex items-center gap-3">
          <Button
            onClick={() => {
              durable.detach();
              requestKeyRef.current = null;
              void deploy();
            }}
            data-testid="deploy-retry-button"
          >
            <Rocket className="h-4 w-4 mr-1.5" aria-hidden="true" />
            Review the plan again
          </Button>
        </div>
      )}

      {deploymentId && (
        <div className="space-y-4">
          <SpawnAgentButton deploymentId={deploymentId} onTaskStarted={() => {}} />
          {investigation.activeInvestigation && (
            <InvestigationProgress
              investigation={investigation.activeInvestigation}
              isRunning={investigation.isRunning}
              onStop={investigation.stop}
              isStopping={investigation.isStopping}
              isOutdated={isInvestigationOutdated(investigation.activeInvestigation)}
              onViewReport={(invId) => {
                investigation.viewReport(invId);
                setShowInvestigationReport(true);
              }}
            />
          )}
          {showInvestigationReport && investigation.activeInvestigation && (
            <InvestigationReport
              investigation={investigation.activeInvestigation}
              onClose={() => setShowInvestigationReport(false)}
              isOutdated={isInvestigationOutdated(investigation.activeInvestigation)}
              onApplyFixes={async (invId, options) => {
                await investigation.applyFixes(invId, options);
                setShowInvestigationReport(false);
              }}
              isApplyingFixes={investigation.isApplyingFixes}
            />
          )}
        </div>
      )}

      {isSuccess && (
        <div className="space-y-6" data-testid="deploy-result">
          <Alert variant="success" title="Deployment Successful!">
            <div className="flex items-center gap-2">
              <PartyPopper className="h-4 w-4" aria-hidden="true" />
              Your scenario has been deployed and is now live.
            </div>
          </Alert>
          <Card>
            <CardContent className="py-6 text-center">
              <div className="mx-auto w-16 h-16 rounded-full bg-emerald-500/20 flex items-center justify-center mb-4">
                <CheckCircle2 className="h-8 w-8 text-emerald-300" aria-hidden="true" />
              </div>
              <h3 className="text-lg font-semibold text-white mb-2">Deployment Complete</h3>
              {deploymentId && <p className="text-xs text-slate-300 mb-4 font-mono">ID: {deploymentId}</p>}
              {domain && <p className="text-slate-200 mb-4">Your scenario is now live at:</p>}
              {domain && (
                <a
                  href={`https://${domain}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  data-testid="deploy-live-link"
                  className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-blue-500/20 text-blue-200 hover:bg-blue-500/30 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400"
                >
                  <ExternalLink className="h-4 w-4" aria-hidden="true" />
                  https://{domain}
                </a>
              )}
              <div className="mt-8 pt-6 border-t border-slate-700">
                <div className="flex flex-col sm:flex-row items-center justify-center gap-3">
                  {onViewDeployments && (
                    <Button onClick={onViewDeployments} variant="outline">
                      <Server className="h-4 w-4 mr-1.5" aria-hidden="true" />
                      View Deployments
                    </Button>
                  )}
                  <Button onClick={reset} data-testid="deploy-start-new-button">
                    Start New Deployment
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {deploymentStatus === "idle" && (
        <Card>
          <CardContent className="py-4">
            <h4 className="text-sm font-medium text-slate-200 mb-3">What happens next:</h4>
            <ol className="space-y-2 text-sm text-slate-300">
              <li className="flex items-start gap-2">
                <span className="text-slate-400">1.</span>
                The deployment record is created with your bundle and secrets
              </li>
              <li className="flex items-start gap-2">
                <span className="text-slate-400">2.</span>
                The executable plan is compiled and shown for review: changes, data effects, downtime and recovery
              </li>
              <li className="flex items-start gap-2">
                <span className="text-slate-400">3.</span>
                Applying the reviewed plan admits a durable operation you can leave and resume
              </li>
              <li className="flex items-start gap-2">
                <span className="text-slate-400">4.</span>
                Health checks verify the release before the operation reports success
              </li>
            </ol>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
