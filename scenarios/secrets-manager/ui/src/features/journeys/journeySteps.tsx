import { Button } from "../../components/ui/button";

export type JourneyId = "configure-secrets" | "fix-vulnerabilities" | "prep-deployment";

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
    onSetResourceInput
  } = options;

  const steps = [] as Array<{ title: string; description: string; content?: React.ReactNode }>;

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
      description: "Use the CLI/API provisioning hooks or jump into secrets-manager CLI to apply changes.",
      content: (
        <div className="space-y-2 text-sm text-white/70">
          <p>Recommended command:</p>
          <code className="block rounded-xl border border-white/20 bg-black/40 px-3 py-2 font-mono text-xs text-emerald-200">
            secrets-manager plan --scenario picker-wheel --tier tier-2-desktop | jq
          </code>
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
      content: manifestData ? (
        <div className="space-y-2 text-xs text-white/70">
          <p>
            Secrets covered: {manifestData.summary.strategized_secrets}/{manifestData.summary.total_secrets}
          </p>
          <p>Blocking: {manifestData.summary.requires_action}</p>
          <div className="rounded-xl border border-white/10 bg-black/40 p-3 font-mono">
            <pre className="overflow-x-auto text-[10px] text-emerald-100">
{JSON.stringify(manifestData.summary, null, 2)}
            </pre>
          </div>
        </div>
      ) : (
        <p className="text-sm text-white/60">No manifest generated yet.</p>
      )
    });
  }

  return steps;
};
