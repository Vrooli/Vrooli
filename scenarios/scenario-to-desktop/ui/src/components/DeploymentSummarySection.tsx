import { Button } from "./ui/button";
import { findDeploymentOption, findServerTypeOption, type DeploymentMode, type ServerType } from "../domain/deployment";

interface DeploymentSummarySectionProps {
  deploymentMode: DeploymentMode;
  serverType: ServerType;
  onOpenDeploymentModal: () => void;
}

export function DeploymentSummarySection({
  deploymentMode,
  serverType,
  onOpenDeploymentModal
}: DeploymentSummarySectionProps) {
  const deployment = findDeploymentOption(deploymentMode);
  const server = findServerTypeOption(serverType);

  return (
    <div className="rounded-lg border border-slate-800 bg-slate-950/60 p-4">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="space-y-2">
          <div>
            <p className="text-xs uppercase tracking-wide text-slate-400">Deployment intent</p>
            <p className="text-sm font-semibold text-slate-100">{deployment.label}</p>
            <p className="text-xs text-slate-400">{deployment.description}</p>
          </div>
          <div>
            <p className="text-xs uppercase tracking-wide text-slate-400">Data source</p>
            <p className="text-sm font-semibold text-slate-100">{server.label}</p>
            <p className="text-xs text-slate-400">{server.description}</p>
          </div>
        </div>
        <Button type="button" variant="outline" size="sm" onClick={onOpenDeploymentModal}>
          Choose deployment
        </Button>
      </div>
    </div>
  );
}
