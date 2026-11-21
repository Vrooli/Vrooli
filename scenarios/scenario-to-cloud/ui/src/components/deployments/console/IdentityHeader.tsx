import { ArrowLeft } from "lucide-react";
import { targetKey } from "../../../lib/consoleActions";
import type { DeploymentIdentity } from "../../../types/console";
import { CopyButton, StatusPill } from "./ConsolePrimitives";

export interface IdentityHeaderProps {
  deployment: DeploymentIdentity;
  domain?: string | null;
  onBack: () => void;
  /** Badges rendered next to the name (status, health summary). */
  badges?: React.ReactNode;
  actions?: React.ReactNode;
}

/**
 * IdentityHeader answers "which environment and machine am I controlling?"
 * It stays visible above every tab and carries copy affordances for the
 * identifiers an operator pastes into the CLI.
 */
export function IdentityHeader({ deployment, domain, onBack, badges, actions }: IdentityHeaderProps) {
  const target = targetKey(deployment.target);
  const transport = deployment.target?.transport ?? "unbound";
  const environment = deployment.environment ?? "production";
  const generation = deployment.target?.enrollment_generation;
  return (
    <header data-testid="console-identity" className="space-y-3 min-w-0">
      <button
        type="button"
        onClick={onBack}
        data-testid="console-back"
        className="inline-flex items-center gap-2 rounded text-sm text-slate-300 hover:text-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden="true" />
        Back to deployments
      </button>
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <h1 className="text-2xl font-bold text-white break-words">{deployment.name}</h1>
          <div className="mt-2 flex flex-wrap items-center gap-2">{badges}</div>
        </div>
        {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
      </div>
      <dl className="grid grid-cols-1 gap-x-6 gap-y-1 rounded-lg border border-white/10 bg-slate-900/60 p-3 text-sm sm:grid-cols-2 lg:grid-cols-4">
        <div className="min-w-0">
          <dt className="text-xs uppercase tracking-wide text-slate-400">Environment</dt>
          <dd className="text-slate-100" data-testid="console-identity-environment">
            <StatusPill tone="info">{environment}</StatusPill>
          </dd>
        </div>
        <div className="min-w-0">
          <dt className="text-xs uppercase tracking-wide text-slate-400">Target</dt>
          <dd className="flex flex-wrap items-center gap-1 font-mono text-xs text-slate-100 break-all" data-testid="console-identity-target">
            <span>{target}</span>
            {typeof generation === "number" && generation > 0 && <span className="text-slate-400">gen {generation}</span>}
            <CopyButton value={target} label="target key" />
          </dd>
        </div>
        <div className="min-w-0">
          <dt className="text-xs uppercase tracking-wide text-slate-400">Transport</dt>
          <dd className="text-slate-100" data-testid="console-identity-transport">
            {transport}
            {deployment.target?.locator?.host && <span className="ml-2 font-mono text-xs text-slate-300">{deployment.target.locator.host}</span>}
          </dd>
        </div>
        <div className="min-w-0">
          <dt className="text-xs uppercase tracking-wide text-slate-400">Deployment id</dt>
          <dd className="flex flex-wrap items-center gap-1 font-mono text-xs text-slate-100 break-all" data-testid="console-identity-id">
            <span>{deployment.id}</span>
            <CopyButton value={deployment.id} label="deployment id" />
          </dd>
        </div>
        {domain && (
          <div className="min-w-0 sm:col-span-2">
            <dt className="text-xs uppercase tracking-wide text-slate-400">Domain</dt>
            <dd className="text-slate-100" data-testid="console-identity-domain">
              <a href={`https://${domain}`} target="_blank" rel="noopener noreferrer" className="text-blue-200 underline-offset-2 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400 rounded">
                {domain}
              </a>
            </dd>
          </div>
        )}
        {typeof deployment.fence === "number" && (
          <div className="min-w-0">
            <dt className="text-xs uppercase tracking-wide text-slate-400">Fence</dt>
            <dd className="font-mono text-xs text-slate-100" data-testid="console-identity-fence">
              {deployment.fence}
            </dd>
          </div>
        )}
      </dl>
    </header>
  );
}
