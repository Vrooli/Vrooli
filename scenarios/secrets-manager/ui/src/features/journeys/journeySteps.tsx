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
  readinessByTier?: Record<
    string,
    {
      summary?: {
        strategized_secrets: number;
        total_secrets: number;
        requires_action: number;
        blocking_secrets?: string[];
        scope_readiness?: Record<string, string>;
      };
      generatedAt?: string;
      loading?: boolean;
      error?: string;
    }
  >;
  readinessSummary?: {
    strategized_secrets: number;
    total_secrets: number;
    requires_action: number;
    blocking_secrets?: string[];
    scope_readiness?: Record<string, string>;
  };
  readinessGeneratedAt?: string;
  readinessIsLoading?: boolean;
  readinessIsError?: boolean;
  readinessError?: Error;
  manifestIsLoading: boolean;
  manifestIsError: boolean;
  manifestError?: Error;
  topResourceNeedingAttention?: string;
  onOpenResource: (resourceName?: string, secretKey?: string) => void;
  onRefetchVulnerabilities: () => void;
  onRefreshReadiness: () => void;
  onManifestRequest: () => void;
  onSetDeploymentScenario: (value: string) => void;
  onSetDeploymentTier: (value: string) => void;
  onSetResourceInput: (value: string) => void;
  onSetProvisionResourceInput: (value: string) => void;
  onSetProvisionSecretKey: (value: string) => void;
  onSetProvisionSecretValue: (value: string) => void;
  onProvisionSubmit: () => void;
  scenarioSelectionContent?: React.ReactNode;
  scenarioSelection?: unknown;
  vulnerabilitySummary?: {
    critical: number;
    high: number;
    medium: number;
    low: number;
  };
  onNavigateTab?: (tab: "dashboard" | "resources" | "compliance" | "deployment") => void;
}

export const buildJourneySteps = (journeyId: JourneyId | null, options: JourneyStepOptions) => {
  if (!journeyId) return [];

  const {
    heroStats,
    manifestData,
    manifestIsLoading,
    manifestIsError,
    manifestError,
    topResourceNeedingAttention,
    scenarioSelectionContent,
    onOpenResource,
    onRefetchVulnerabilities,
    onManifestRequest,
    onNavigateTab
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
      title: "Configure secrets by resource & tier",
      description: "See what’s missing and how strategies differ by environment.",
      content: (
        <div className="space-y-3">
          <p className="text-sm text-white/70">
            Tiers expect different handling: infrastructure secrets stay server-side, desktop/mobile bundles strip them,
            and SaaS delegates to cloud providers. Start with the Resources tab to see tier coverage and per-resource gaps.
          </p>
          <Button size="sm" onClick={() => onNavigateTab?.("resources")}>Open Resources</Button>
        </div>
      )
    });
    steps.push({
      title: "Address vulnerabilities with context",
      description: "Understand how findings are scored and where to start.",
      content: (
        <div className="space-y-3">
          <p className="text-sm text-white/70">
            Risk score blends severity and coverage. Critical/high findings block production; medium/low can be planned.
            Use the Compliance tab filters to triage quickly.
          </p>
          <Button size="sm" onClick={() => onNavigateTab?.("compliance")}>Go to Compliance</Button>
        </div>
      )
    });
    steps.push({
      title: "Prep deployment campaigns",
      description: "Learn how manifests are generated and how to resume work.",
      content: (
        <div className="space-y-3">
          <p className="text-sm text-white/70">
            Campaigns tie a scenario and target tier together. We analyze required secrets, check strategy coverage,
            and export a manifest for deployment-manager or scenario-to-*.
          </p>
          <Button size="sm" onClick={() => onNavigateTab?.("deployment")}>Open Deployment</Button>
        </div>
      )
    });
  }

  if (journeyId === "configure-secrets") {
    steps.push({
      title: "Audit coverage and jump to workbench",
      description: "See missing secrets and tier strategies, then open the right resource.",
      content: (
        <div className="space-y-3">
          <ul className="space-y-1 text-sm text-white/80">
            <li>Configured resources: {heroStats?.vault_configured ?? 0}</li>
            <li>Total resources tracked: {heroStats?.vault_total ?? 0}</li>
            <li>Missing secrets: {heroStats?.missing_secrets ?? 0}</li>
          </ul>
          <div className="flex flex-wrap gap-2">
            <Button size="sm" onClick={() => onNavigateTab?.("resources")}>
              Open Resources tab
            </Button>
            <Button size="sm" variant="outline" onClick={() => onOpenResource(topResourceNeedingAttention)}>
              {topResourceNeedingAttention ? `Open ${topResourceNeedingAttention}` : "Top blocker"}
            </Button>
          </div>
          <p className="text-[11px] text-white/60">
            Use By Tier for strategies and Per Resource for search + sort across everything.
          </p>
        </div>
      )
    });
  }

  if (journeyId === "fix-vulnerabilities") {
    steps.push({
      title: "Understand and triage findings",
      description: "Start with critical/high issues, then medium/low by component.",
      content: (
        <div className="space-y-3">
          <p className="text-sm text-white/70">
            Risk score blends severity and count. Critical/high items block launch; medium/low can be scheduled.
          </p>
          <Button size="sm" onClick={() => onNavigateTab?.("compliance")}>
            Open Compliance tab
          </Button>
          <Button variant="outline" size="sm" onClick={onRefetchVulnerabilities}>
            Re-run scan
          </Button>
        </div>
      )
    });
  }

  if (journeyId === "prep-deployment") {
    steps.push({
      title: "Pick a campaign",
      description: "Choose the scenario and target tier you want to ship.",
      content: (
        <div className="space-y-3">
          {scenarioSelectionContent || <p className="text-sm text-white/70">Scenario list not available.</p>}
          <p className="text-xs text-white/60">
            Campaigns are searchable and sortable in the Deployment tab. Select one to load readiness and manifest tools.
          </p>
          <Button size="sm" onClick={() => onNavigateTab?.("deployment")}>
            Go to Deployment
          </Button>
        </div>
      )
    });
    steps.push({
      title: "Check readiness and blockers",
      description: "Review tier coverage and open the top resource to clear strategies.",
      content: (
        <div className="space-y-3">
          <p className="text-sm text-white/70">
            Use the campaign stepper: jump to readiness to see strategies per tier, then open the blocker directly.
          </p>
          <Button size="sm" variant="outline" onClick={() => onOpenResource(topResourceNeedingAttention)}>
            {topResourceNeedingAttention ? `Open ${topResourceNeedingAttention}` : "Open resource workbench"}
          </Button>
        </div>
      )
    });
    steps.push({
      title: "Generate and export manifest",
      description: "Create the deployment manifest for handoff.",
      content: (
        <div className="space-y-3">
          <p className="text-sm text-white/70">
            After clearing blockers, generate the manifest from the Deployment tab and export it for deployment-manager or
            scenario-to-*.
          </p>
          <Button size="sm" onClick={onManifestRequest} disabled={manifestIsLoading}>
            {manifestIsLoading ? "Generating…" : manifestData ? "Regenerate manifest" : "Generate manifest"}
          </Button>
          {manifestIsError ? (
            <p className="text-xs text-red-300">{manifestError?.message}</p>
          ) : null}
        </div>
      )
    });
  }

  return steps;
};
