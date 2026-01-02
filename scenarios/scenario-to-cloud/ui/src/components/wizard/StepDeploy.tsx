import { useState } from "react";
import { Rocket, CheckCircle2, ExternalLink, PartyPopper, Server } from "lucide-react";
import { Button } from "../ui/button";
import { Alert } from "../ui/alert";
import { Card, CardContent } from "../ui/card";
import { DeploymentProgress } from "./DeploymentProgress";
import { InvestigateButton } from "./InvestigateButton";
import { InvestigationProgress } from "./InvestigationProgress";
import { InvestigationReport } from "./InvestigationReport";
import { useDeploymentInvestigation } from "../../hooks/useInvestigation";
import type { useDeployment } from "../../hooks/useDeployment";

interface StepDeployProps {
  deployment: ReturnType<typeof useDeployment>;
  onViewDeployments?: () => void;
}

export function StepDeploy({ deployment, onViewDeployments }: StepDeployProps) {
  const {
    deploymentStatus,
    deploymentError,
    deploymentId,
    deploy,
    parsedManifest,
    reset,
    onDeploymentComplete,
  } = deployment;

  const isDeploying = deploymentStatus === "deploying";
  const isSuccess = deploymentStatus === "success";
  const isFailed = deploymentStatus === "failed";

  // Get domain for success message
  const domain = parsedManifest.ok ? parsedManifest.value.edge?.domain : null;

  // Investigation state
  const [showInvestigationReport, setShowInvestigationReport] = useState(false);
  const investigation = useDeploymentInvestigation(deploymentId);

  // Handler for when progress completes (called by DeploymentProgress via SSE)
  const handleProgressComplete = (success: boolean, error?: string) => {
    onDeploymentComplete(success, error);
  };

  // Handler for when investigation starts
  const handleInvestigationStarted = (investigationId: string) => {
    // Could navigate to investigation view or show inline
  };

  return (
    <div className="space-y-6">
      {/* Deploy Button - only show when idle */}
      {deploymentStatus === "idle" && (
        <div className="flex items-center gap-3">
          <Button
            onClick={deploy}
            disabled={!parsedManifest.ok}
          >
            <Rocket className="h-4 w-4 mr-1.5" />
            Deploy to VPS
          </Button>
        </div>
      )}

      {/* Progress Tracking - show when deploying OR failed (to see which steps passed/failed) */}
      {(isDeploying || isFailed) && deploymentId && (
        <DeploymentProgress
          deploymentId={deploymentId}
          onComplete={handleProgressComplete}
        />
      )}

      {/* Retry and Investigate buttons when failed */}
      {isFailed && deploymentId && (
        <div className="space-y-4">
          <div className="flex items-center gap-3">
            <Button onClick={deploy}>
              <Rocket className="h-4 w-4 mr-1.5" />
              Retry Deployment
            </Button>
            <InvestigateButton
              deploymentId={deploymentId}
              onInvestigationStarted={handleInvestigationStarted}
            />
          </div>

          {/* Investigation progress - show for active OR recently completed investigations */}
          {investigation.activeInvestigation && (
            <InvestigationProgress
              deploymentId={deploymentId}
              onViewReport={() => setShowInvestigationReport(true)}
            />
          )}

          {/* Investigation report modal */}
          {showInvestigationReport && investigation.activeInvestigation && (
            <InvestigationReport
              investigation={investigation.activeInvestigation}
              onClose={() => setShowInvestigationReport(false)}
              onApplyFixes={async (invId, options) => {
                await investigation.applyFixes(invId, options);
                setShowInvestigationReport(false);
              }}
              isApplyingFixes={investigation.isApplyingFixes}
            />
          )}
        </div>
      )}

      {/* Success */}
      {isSuccess && (
        <div className="space-y-6">
          <Alert variant="success" title="Deployment Successful!">
            <div className="flex items-center gap-2">
              <PartyPopper className="h-4 w-4" />
              Your scenario has been deployed and is now live.
            </div>
          </Alert>

          <Card>
            <CardContent className="py-6 text-center">
              <div className="mx-auto w-16 h-16 rounded-full bg-emerald-500/20 flex items-center justify-center mb-4">
                <CheckCircle2 className="h-8 w-8 text-emerald-400" />
              </div>

              <h3 className="text-lg font-semibold text-white mb-2">
                Deployment Complete
              </h3>

              {deploymentId && (
                <p className="text-xs text-slate-500 mb-4 font-mono">
                  ID: {deploymentId}
                </p>
              )}

              {domain && (
                <p className="text-slate-300 mb-4">
                  Your scenario is now live at:
                </p>
              )}

              {domain && (
                <a
                  href={`https://${domain}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-blue-500/20 text-blue-300 hover:bg-blue-500/30 transition-colors"
                >
                  <ExternalLink className="h-4 w-4" />
                  https://{domain}
                </a>
              )}

              <div className="mt-8 pt-6 border-t border-slate-700">
                <div className="flex flex-col sm:flex-row items-center justify-center gap-3">
                  {onViewDeployments && (
                    <Button onClick={onViewDeployments} variant="outline">
                      <Server className="h-4 w-4 mr-1.5" />
                      View Deployments
                    </Button>
                  )}
                  <Button onClick={reset}>
                    Start New Deployment
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Pre-deploy info */}
      {deploymentStatus === "idle" && (
        <Card>
          <CardContent className="py-4">
            <h4 className="text-sm font-medium text-slate-300 mb-3">What happens during deployment:</h4>
            <ol className="space-y-2 text-sm text-slate-400">
              <li className="flex items-start gap-2">
                <span className="text-slate-500">1.</span>
                Bundle is uploaded to the target server via SCP
              </li>
              <li className="flex items-start gap-2">
                <span className="text-slate-500">2.</span>
                Vrooli setup runs to configure the environment
              </li>
              <li className="flex items-start gap-2">
                <span className="text-slate-500">3.</span>
                Required resources are started (Postgres, Redis, etc.)
              </li>
              <li className="flex items-start gap-2">
                <span className="text-slate-500">4.</span>
                Scenario services are started with fixed ports
              </li>
              <li className="flex items-start gap-2">
                <span className="text-slate-500">5.</span>
                Caddy configures HTTPS with Let's Encrypt
              </li>
              <li className="flex items-start gap-2">
                <span className="text-slate-500">6.</span>
                Health checks verify the deployment is working
              </li>
            </ol>
          </CardContent>
        </Card>
      )}

    </div>
  );
}
