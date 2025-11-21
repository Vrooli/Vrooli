import { Button } from "../../components/ui/button";

export type JourneyId = "orientation" | "configure-secrets" | "fix-vulnerabilities" | "prep-deployment";

interface JourneyStepOptions {
  heroStats?: {
    vault_configured: number;
    vault_total: number;
    missing_secrets: number;
    risk_score: number;
  };
  orientationData?: {
    hero_stats: {
      risk_score: number;
    };
  };
  tierReadiness: Array<{
    tier: string;
    label: string;
    ready_percent: number;
    strategized: number;
    total: number;
  }>;
  deploymentScenario: string;
  deploymentTier: string;
  resourceInput: string;
  provisionResourceInput: string;
  provisionSecretKey: string;
  provisionSecretValue: string;
  provisionIsLoading: boolean;
  provisionIsSuccess: boolean;
  provisionError?: string;
  manifestData?: {
    summary: {
      strategized_secrets: number;
      total_secrets: number;
      requires_action: number;
    };
  };
  manifestIsLoading: boolean;
  manifestIsError: boolean;
  manifestError?: Error;
  topResourceNeedingAttention?: string;
  onOpenResource: (resourceName?: string) => void;
  onRefetchVulnerabilities: () => void;
  onManifestRequest: () => void;
  onSetDeploymentScenario: (value: string) => void;
  onSetDeploymentTier: (value: string) => void;
  onSetResourceInput: (value: string) => void;
  onSetProvisionResourceInput: (value: string) => void;
  onSetProvisionSecretKey: (value: string) => void;
  onSetProvisionSecretValue: (value: string) => void;
  onProvisionSubmit: () => void;
}

export const buildJourneySteps = (journeyId: JourneyId | null, options: JourneyStepOptions) => {
  if (!journeyId) return [];

  const {
    heroStats,
    orientationData,
    tierReadiness,
    deploymentScenario,
    deploymentTier,
    resourceInput,
    provisionResourceInput,
    provisionSecretKey,
    provisionSecretValue,
    provisionIsLoading,
    provisionIsSuccess,
    provisionError,
    manifestData,
    manifestIsLoading,
    manifestIsError,
    manifestError,
    topResourceNeedingAttention,
    onOpenResource,
    onRefetchVulnerabilities,
    onManifestRequest,
    onSetDeploymentScenario,
    onSetDeploymentTier,
    onSetResourceInput,
    onSetProvisionResourceInput,
    onSetProvisionSecretKey,
    onSetProvisionSecretValue,
    onProvisionSubmit
  } = options;

  const steps = [] as Array<{ title: string; description: string; content?: React.ReactNode }>;

  if (journeyId === "orientation") {
    steps.push({
      title: "Welcome to Secrets Manager",
      description: "Your security operations hub for Vrooli's secrets infrastructure.",
      content: (
        <div className="space-y-4">
          <p className="text-sm text-white/70">
            This dashboard helps you discover, validate, and provision secrets across all scenarios and resources.
            Let's start with a snapshot of your current security posture.
          </p>
          <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
            <h4 className="text-xs uppercase tracking-[0.2em] text-white/60 mb-3">Readiness Snapshot</h4>
            <div className="grid gap-3 md:grid-cols-3 text-sm">
              <div>
                <p className="text-white/60">Overall Score</p>
                <p className="text-2xl font-semibold text-white">{heroStats?.vault_configured ?? 0}%</p>
              </div>
              <div>
                <p className="text-white/60">Risk Score</p>
                <p className="text-2xl font-semibold text-white">{heroStats?.risk_score ?? 0}</p>
              </div>
              <div>
                <p className="text-white/60">Missing Secrets</p>
                <p className="text-2xl font-semibold text-white">{heroStats?.missing_secrets ?? 0}</p>
              </div>
            </div>
            <div className="mt-3 pt-3 border-t border-white/10 text-xs text-white/50">
              {heroStats?.vault_configured ?? 0} of {heroStats?.vault_total ?? 0} resources configured
            </div>
          </div>
        </div>
      )
    });
    steps.push({
      title: "Understand Your Risks",
      description: "Review the most critical vulnerabilities and missing secrets that need attention.",
      content: (
        <div className="space-y-3">
          <p className="text-sm text-white/70">
            Your security score is calculated from vault coverage and vulnerability severity.
            The higher your risk score ({orientationData?.hero_stats.risk_score ?? 0}), the more urgent action is needed.
          </p>
          {(heroStats?.missing_secrets ?? 0) > 0 ? (
            <div className="rounded-2xl border border-amber-500/30 bg-amber-500/10 px-4 py-3">
              <p className="text-sm font-semibold text-amber-200">
                ⚠️ {heroStats?.missing_secrets} missing secrets detected
              </p>
              <p className="text-xs text-amber-200/70 mt-1">
                These gaps may prevent resources from starting correctly.
              </p>
            </div>
          ) : (
            <div className="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3">
              <p className="text-sm font-semibold text-emerald-200">
                ✓ All required secrets are configured
              </p>
            </div>
          )}
        </div>
      )
    });
    steps.push({
      title: "Choose Your Path",
      description: "Three guided journeys help you secure your infrastructure.",
      content: (
        <div className="space-y-3">
          <div className="rounded-2xl border border-white/10 bg-white/5 p-3 hover:border-white/20 transition">
            <p className="font-semibold text-white text-sm">Configure Secrets</p>
            <p className="text-xs text-white/60 mt-1">
              Audit vault coverage, prioritize fixes, and provision missing secrets.
            </p>
          </div>
          <div className="rounded-2xl border border-white/10 bg-white/5 p-3 hover:border-white/20 transition">
            <p className="font-semibold text-white text-sm">Fix Vulnerabilities</p>
            <p className="text-xs text-white/60 mt-1">
              Review security findings, plan remediation, and verify fixes.
            </p>
          </div>
          <div className="rounded-2xl border border-white/10 bg-white/5 p-3 hover:border-white/20 transition">
            <p className="font-semibold text-white text-sm">Prep Deployment</p>
            <p className="text-xs text-white/60 mt-1">
              Assess tier readiness, generate manifests, and export bundles.
            </p>
          </div>
          <p className="text-xs text-white/50 mt-4">
            Click any journey card on the right to begin →
          </p>
        </div>
      )
    });
    steps.push({
      title: "Quick Wins",
      description: "Start with the highest-impact actions.",
      content: (
        <div className="space-y-3">
          {topResourceNeedingAttention ? (
            <>
              <p className="text-sm text-white/70">
                <strong className="text-white">{topResourceNeedingAttention}</strong> needs immediate attention.
              </p>
              <Button size="sm" onClick={() => onOpenResource(topResourceNeedingAttention)}>
                Open {topResourceNeedingAttention} Panel
              </Button>
            </>
          ) : (
            <p className="text-sm text-white/70">
              Great! No urgent issues detected. Explore the other journeys to maintain your security posture.
            </p>
          )}
          <div className="pt-3 border-t border-white/10">
            <p className="text-xs text-white/50">
              Pro tip: Use the Resource Workbench section below to see all resource health at a glance.
            </p>
          </div>
        </div>
      )
    });
  }

  if (journeyId === "configure-secrets") {
    steps.push({
      title: "Audit coverage",
      description: "Review vault readiness and identify resources with missing requirements.",
      content: (
        <ul className="mt-2 space-y-1 text-sm text-white/80">
          <li>Configured resources: {heroStats?.vault_configured ?? 0}</li>
          <li>Total resources tracked: {heroStats?.vault_total ?? 0}</li>
          <li>Missing secrets: {heroStats?.missing_secrets ?? 0}</li>
        </ul>
      )
    });
    steps.push({
      title: "Prioritize fixes",
      description: "Work the highest-risk resources first using the insights table below.",
      content: (
        <div className="space-y-3">
          <p className="text-sm text-white/70">
            Start with degraded resources in the Workbench section. Each resource card exposes inline actions to
            manage its secrets and deployment strategies.
          </p>
          <Button size="sm" onClick={() => onOpenResource(topResourceNeedingAttention)} disabled={!topResourceNeedingAttention}>
            {topResourceNeedingAttention ? `Open ${topResourceNeedingAttention}` : "No targets"}
          </Button>
        </div>
      )
    });
    steps.push({
      title: "Provision",
      description: "Add secrets directly to Vault or use the CLI for bulk operations.",
      content: (
        <div className="space-y-3">
          <div className="space-y-2">
            <label className="flex flex-col gap-1 text-xs uppercase tracking-[0.2em] text-white/60">
              Resource
              <input
                type="text"
                value={provisionResourceInput}
                onChange={(e) => onSetProvisionResourceInput(e.target.value)}
                placeholder="postgres, vault, etc."
                className="rounded-2xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
              />
            </label>
            <label className="flex flex-col gap-1 text-xs uppercase tracking-[0.2em] text-white/60">
              Secret Key
              <input
                type="text"
                value={provisionSecretKey}
                onChange={(e) => onSetProvisionSecretKey(e.target.value)}
                placeholder="SECRET_NAME"
                className="rounded-2xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
              />
            </label>
            <label className="flex flex-col gap-1 text-xs uppercase tracking-[0.2em] text-white/60">
              Secret Value
              <input
                type="password"
                value={provisionSecretValue}
                onChange={(e) => onSetProvisionSecretValue(e.target.value)}
                placeholder="•••••"
                className="rounded-2xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
              />
            </label>
          </div>
          <Button
            variant="secondary"
            size="sm"
            className="w-full"
            onClick={onProvisionSubmit}
            disabled={provisionIsLoading || !provisionResourceInput || !provisionSecretKey || !provisionSecretValue}
          >
            {provisionIsLoading ? "Provisioning..." : "Provision Secret"}
          </Button>
          {provisionIsSuccess && (
            <div className="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 px-3 py-2 text-sm text-emerald-200">
              ✓ Secret provisioned successfully
            </div>
          )}
          {provisionError && (
            <div className="rounded-2xl border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-200">
              {provisionError}
            </div>
          )}
          <div className="pt-3 border-t border-white/10">
            <p className="text-xs text-white/50">
              Alternative: Use CLI for bulk provisioning
            </p>
            <code className="block mt-1 rounded-xl border border-white/20 bg-black/40 px-3 py-2 font-mono text-xs text-emerald-200">
              secrets-manager plan --scenario picker-wheel | jq
            </code>
          </div>
        </div>
      )
    });
  }

  if (journeyId === "fix-vulnerabilities") {
    steps.push({
      title: "Review findings",
      description: "Filter findings by severity and assign owners before remediation.",
      content: (
        <p className="text-sm text-white/70">Risk score: {orientationData?.hero_stats.risk_score ?? 0}</p>
      )
    });
    steps.push({
      title: "Plan remediation",
      description: "Trigger claude-code fixer runs for repetitive issues or create app-issue-tracker tickets.",
      content: (
        <p className="text-sm text-white/70">
          Use the vulnerability list below to copy file references and recommendations. When fixes are ready, run the
          automated tests to confirm stability.
        </p>
      )
    });
    steps.push({
      title: "Verify",
      description: "Re-run scans and confirm compliance deltas.",
      content: (
        <Button variant="outline" size="sm" onClick={onRefetchVulnerabilities}>
          Re-run scan
        </Button>
      )
    });
  }

  if (journeyId === "prep-deployment") {
    steps.push({
      title: "Assess tier readiness",
      description: "Confirm which tiers have full secret strategies.",
      content: (
        <ul className="mt-2 space-y-1 text-sm text-white/80">
          {tierReadiness.slice(0, 3).map((tier) => (
            <li key={tier.tier}>
              {tier.label}: {tier.ready_percent}% ({tier.strategized}/{tier.total})
            </li>
          ))}
        </ul>
      )
    });
    steps.push({
      title: "Generate manifest",
      description: "Select scenario, tier, and optional resource filters to emit a bundle manifest.",
      content: (
        <div className="space-y-3 text-sm text-white/80">
          <label className="flex flex-col gap-1 text-xs uppercase tracking-[0.2em] text-white/60">
            Scenario
            <input
              type="text"
              value={deploymentScenario}
              onChange={(event) => onSetDeploymentScenario(event.target.value)}
              className="rounded-2xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
            />
          </label>
          <label className="flex flex-col gap-1 text-xs uppercase tracking-[0.2em] text-white/60">
            Tier
            <select
              value={deploymentTier}
              onChange={(event) => onSetDeploymentTier(event.target.value)}
              className="rounded-2xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
            >
              {tierReadiness.map((tier) => (
                <option key={tier.tier} value={tier.tier}>
                  {tier.label}
                </option>
              ))}
            </select>
          </label>
          <label className="flex flex-col gap-1 text-xs uppercase tracking-[0.2em] text-white/60">
            Resources (optional)
            <textarea
              value={resourceInput}
              onChange={(event) => onSetResourceInput(event.target.value)}
              placeholder="postgres, vault"
              rows={2}
              className="rounded-2xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
            />
          </label>
          <Button variant="secondary" size="sm" className="w-full" onClick={onManifestRequest} disabled={manifestIsLoading}>
            {manifestIsLoading ? "Generating…" : "Generate manifest"}
          </Button>
          {manifestIsError ? (
            <p className="text-xs text-red-300">{manifestError?.message}</p>
          ) : null}
        </div>
      )
    });
    steps.push({
      title: "Review manifest",
      description: "Inspect the resulting manifest and hand it off to deployment-manager or scenario-to-*.",
      content: manifestIsLoading ? (
        <div className="space-y-3">
          <div className="flex items-center gap-3 rounded-2xl border border-cyan-400/30 bg-cyan-400/5 px-4 py-3">
            <div className="h-5 w-5 animate-spin rounded-full border-2 border-cyan-400 border-t-transparent" />
            <div className="flex-1">
              <p className="text-sm font-semibold text-cyan-100">Generating Deployment Manifest</p>
              <p className="text-xs text-cyan-200/70">Analyzing dependencies and secret strategies...</p>
            </div>
          </div>
        </div>
      ) : manifestIsError ? (
        <div className="space-y-3">
          <div className="rounded-2xl border border-red-500/30 bg-red-500/10 px-4 py-3">
            <p className="text-sm font-semibold text-red-200">❌ Failed to Generate Manifest</p>
            <p className="text-xs text-red-200/70 mt-1">{manifestError?.message || "Unknown error occurred"}</p>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={onManifestRequest}
            className="w-full"
          >
            Retry Generation
          </Button>
        </div>
      ) : manifestData ? (
        <div className="space-y-3 text-xs text-white/70">
          <div className="grid gap-2 rounded-2xl border border-emerald-500/30 bg-emerald-500/5 p-3">
            <div className="flex items-center justify-between">
              <span className="text-white/60">Secrets Covered:</span>
              <span className="font-semibold text-emerald-100">
                {manifestData.summary.strategized_secrets}/{manifestData.summary.total_secrets}
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-white/60">Requires Action:</span>
              <span className={`font-semibold ${manifestData.summary.requires_action > 0 ? 'text-amber-100' : 'text-emerald-100'}`}>
                {manifestData.summary.requires_action}
              </span>
            </div>
          </div>
          {manifestData.analyzer_generated_at && (() => {
            const analyzerAge = Date.now() - new Date(manifestData.analyzer_generated_at).getTime();
            const hoursOld = Math.floor(analyzerAge / (1000 * 60 * 60));
            if (hoursOld > 24) {
              return (
                <div className="rounded-2xl border border-amber-500/30 bg-amber-500/10 px-3 py-2">
                  <p className="text-sm font-semibold text-amber-200">
                    ⚠️ Dependency analysis is {hoursOld}h old
                  </p>
                  <p className="text-xs text-amber-200/70 mt-1">
                    Run scenario-dependency-analyzer to refresh fitness scores
                  </p>
                </div>
              );
            }
            return null;
          })()}
          <details className="rounded-xl border border-white/10 bg-black/40">
            <summary className="cursor-pointer px-3 py-2 text-xs font-semibold text-white hover:bg-white/5">
              View Full Manifest
            </summary>
            <div className="border-t border-white/10 p-3 font-mono">
              <pre className="overflow-x-auto text-[10px] text-emerald-100">
{JSON.stringify(manifestData.summary, null, 2)}
              </pre>
            </div>
          </details>
        </div>
      ) : (
        <div className="rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-center text-sm text-white/60">
          No manifest generated yet. Complete the previous step to generate.
        </div>
      )
    });
  }

  return steps;
};
