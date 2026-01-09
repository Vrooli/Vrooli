import { Button } from "../../components/ui/button";
import { ShieldCheck, AlertTriangle, KeyRound } from "lucide-react";

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
  onStartTutorial?: (journey: JourneyId, startStep?: number) => void;
}

export const buildJourneySteps = (journeyId: JourneyId | null, options: JourneyStepOptions) => {
  if (!journeyId) return [];

  const {
    heroStats,
    vulnerabilitySummary,
    onRefetchVulnerabilities,
    onNavigateTab,
    onStartTutorial,
    tierReadiness,
    onOpenResource
  } = options;

  const blockedTiers =
    tierReadiness?.filter((tier) => tier.ready_percent < 100 || tier.strategized < tier.total) ?? [];

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
                <div className="flex items-center gap-2 text-white/80">
                  <ShieldCheck className="h-4 w-4 text-emerald-300" />
                  <p className="text-white/60">Overall Score</p>
                </div>
                <p className="text-2xl font-semibold text-white">{heroStats?.vault_configured ?? 0}%</p>
                <p className="text-[11px] text-white/50">
                  % of resources with all required secrets configured. &lt;80% → start Configure Secrets now.
                </p>
              </div>
              <div>
                <div className="flex items-center gap-2 text-white/80">
                  <AlertTriangle className="h-4 w-4 text-amber-200" />
                  <p className="text-white/60">Risk Score</p>
                </div>
                <p className="text-2xl font-semibold text-white">{heroStats?.risk_score ?? 0}</p>
                <p className="text-[11px] text-white/50">
                  Higher = more severe vulnerabilities. ≥60 → start Address Vulnerabilities.
                </p>
              </div>
              <div>
                <div className="flex items-center gap-2 text-white/80">
                  <KeyRound className="h-4 w-4 text-cyan-200" />
                  <p className="text-white/60">Missing Secrets</p>
                </div>
                <p className="text-2xl font-semibold text-white">{heroStats?.missing_secrets ?? 0}</p>
                <p className="text-[11px] text-white/50">
                  Required secrets not present in Vault. &gt;0 blocks services from starting.
                </p>
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
      title: "Understand your coverage",
      description: "See missing secrets and tier strategies, then jump into Resources with guidance if needed.",
      content: (
        <div className="space-y-3">
          <div className="grid gap-3 md:grid-cols-3 text-sm">
            <div className="rounded-2xl border border-white/10 bg-white/5 p-3">
              <div className="flex items-center gap-2 text-white/80">
                <ShieldCheck className="h-4 w-4 text-emerald-300" />
                <span className="text-white/60">Configured</span>
              </div>
              <p className="text-2xl font-semibold text-white">{heroStats?.vault_configured ?? 0}</p>
              <p className="text-[11px] text-white/50">Resources fully configured. Aim for = total tracked.</p>
            </div>
            <div className="rounded-2xl border border-white/10 bg-white/5 p-3">
              <div className="flex items-center gap-2 text-white/80">
                <KeyRound className="h-4 w-4 text-cyan-200" />
                <span className="text-white/60">Missing secrets</span>
              </div>
              <p className="text-2xl font-semibold text-amber-100">{heroStats?.missing_secrets ?? 0}</p>
              <p className="text-[11px] text-white/50">Any number here blocks services. Fix before shipping.</p>
            </div>
            <div className="rounded-2xl border border-white/10 bg-white/5 p-3">
              <div className="flex items-center gap-2 text-white/80">
                <AlertTriangle className="h-4 w-4 text-amber-200" />
                <span className="text-white/60">Coverage ratio</span>
              </div>
              <p className="text-2xl font-semibold text-white">
                {heroStats && heroStats.vault_total > 0
                  ? Math.round((heroStats.vault_configured / heroStats.vault_total) * 100)
                  : 0}
                %
              </p>
              <p className="text-[11px] text-white/50">&lt;80% means many services are unready.</p>
            </div>
          </div>
          <p className="text-xs text-white/60">
            Meaning: missing secrets = immediate blockers; configured/total shows coverage (shoot for 100%); low coverage or any
            missing secrets → start Configure Secrets now.
          </p>
          <div className="flex flex-wrap gap-2">
            <Button size="sm" onClick={() => onStartTutorial?.("configure-secrets")}>
              Start guided tutorial
            </Button>
            <Button size="sm" variant="outline" onClick={() => onNavigateTab?.("resources")}>
              View Resources tab
            </Button>
          </div>
        </div>
      )
    });
    steps.push({
      title: "Clear blockers by tier",
      description: "Use the By Tier view to see which environments are blocked.",
      content: (
        <div className="space-y-3 text-sm text-white/70">
          <p>
            In the Resources tab, stay on <strong className="text-white">By Tier</strong>. Tiers with missing strategies show
            “action needed”. Open the highlighted tier to review blocking secrets.
          </p>
          <ul className="list-disc space-y-2 pl-5 text-xs text-white/60">
            <li>Blocked tiers mean deployment to that environment will fail.</li>
            <li>Prioritize Tier 2+ if you plan to ship desktop/mobile builds soon.</li>
          </ul>
          <Button size="sm" onClick={() => onNavigateTab?.("resources")}>
            Jump to By Tier
          </Button>
        </div>
      )
    });
    steps.push({
      title: "Assign strategies per resource",
      description: "Use the per-resource table to open a resource and set strategies.",
      content: (
        <div className="space-y-3 text-sm text-white/70">
          <p>Switch to <strong className="text-white">Per Resource</strong> to search and sort every resource.</p>
          <p className="text-xs text-white/60">
            Click a row’s <strong className="text-white">Open</strong> button to launch the resource panel, set classifications,
            and define tier strategies for each secret.
          </p>
          <div className="flex flex-wrap gap-2">
            <Button size="sm" onClick={() => onNavigateTab?.("resources")}>
              View resource table
            </Button>
            <Button size="sm" variant="outline" onClick={() => onOpenResource?.()}>
              Open top blocker
            </Button>
          </div>
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
          <div className="grid gap-3 md:grid-cols-2 text-sm">
            <div className="rounded-2xl border border-white/10 bg-white/5 p-3">
              <div className="flex items-center gap-2 text-white/80">
                <AlertTriangle className="h-4 w-4 text-amber-200" />
                <span className="text-white/60">Critical / High</span>
              </div>
              <p className="text-2xl font-semibold text-amber-100">
                {(vulnerabilitySummary?.critical ?? 0) + (vulnerabilitySummary?.high ?? 0)}
              </p>
              <p className="text-[11px] text-white/50">Launch blockers. If &gt;0, start remediation now.</p>
            </div>
            <div className="rounded-2xl border border-white/10 bg-white/5 p-3">
              <div className="flex items-center gap-2 text-white/80">
                <ShieldCheck className="h-4 w-4 text-emerald-300" />
                <span className="text-white/60">Medium / Low</span>
              </div>
              <p className="text-2xl font-semibold text-white">
                {(vulnerabilitySummary?.medium ?? 0) + (vulnerabilitySummary?.low ?? 0)}
              </p>
              <p className="text-[11px] text-white/50">Schedule-able risks. Tackle after blockers.</p>
            </div>
          </div>
          <p className="text-xs text-white/60">
            Critical/high = launch blockers; medium/low = schedule-able but still risky. If Critical/High &gt; 0, start now.
          </p>
          <div className="flex flex-wrap gap-2">
            <Button size="sm" onClick={() => onStartTutorial?.("fix-vulnerabilities")}>
              Start guided tutorial
            </Button>
            <Button size="sm" variant="outline" onClick={() => onNavigateTab?.("compliance")}>
              Open Compliance tab
            </Button>
            <Button variant="outline" size="sm" onClick={onRefetchVulnerabilities}>
              Re-run scan
            </Button>
          </div>
        </div>
      )
    });
    steps.push({
      title: "Use compliance overview to prioritize",
      description: "Check the compliance summary cards first.",
      content: (
        <div className="space-y-2 text-sm text-white/70">
          <p>Open the <strong className="text-white">Compliance</strong> tab and look at the overall score and vault health.</p>
          <p className="text-xs text-white/60">If vault health is low, fix missing secrets before tackling code findings.</p>
          <Button size="sm" onClick={() => onNavigateTab?.("compliance")}>
            Jump to compliance overview
          </Button>
        </div>
      )
    });
    steps.push({
      title: "Filter findings for action",
      description: "Filter by severity and component, then open the resource.",
      content: (
        <div className="space-y-3 text-sm text-white/70">
          <p>In Security Findings, set Severity to Critical/High first. Narrow by component type to isolate noisy services.</p>
          <p className="text-xs text-white/60">Use the row action to open the resource panel and mark findings as resolved after fixes.</p>
          <div className="flex flex-wrap gap-2">
            <Button size="sm" onClick={() => onNavigateTab?.("compliance")}>
              Go to findings table
            </Button>
            <Button size="sm" variant="outline" onClick={onRefetchVulnerabilities}>
              Refresh scan results
            </Button>
          </div>
        </div>
      )
    });
  }

  if (journeyId === "prep-deployment") {
    steps.push({
      title: "Assess deployment urgency",
      description: "See if blocked tiers or missing secrets justify running the prep flow now.",
      content: (
        <div className="space-y-3">
          <div className="grid gap-3 md:grid-cols-3 text-sm">
            <div className="rounded-2xl border border-white/10 bg-white/5 p-3">
              <div className="flex items-center gap-2 text-white/80">
                <ShieldCheck className="h-4 w-4 text-emerald-300" />
                <span className="text-white/60">Tiers tracked</span>
              </div>
              <p className="text-2xl font-semibold text-white">{tierReadiness?.length ?? 0}</p>
              <p className="text-[11px] text-white/50">Tier 1–5 coverage under watch.</p>
            </div>
            <div className="rounded-2xl border border-white/10 bg-white/5 p-3">
              <div className="flex items-center gap-2 text-white/80">
                <AlertTriangle className="h-4 w-4 text-amber-200" />
                <span className="text-white/60">Blocked tiers</span>
              </div>
              <p className="text-2xl font-semibold text-amber-100">{blockedTiers.length}</p>
              <p className="text-[11px] text-white/50">Any blocked tier can’t ship. Clear strategies first.</p>
            </div>
            <div className="rounded-2xl border border-white/10 bg-white/5 p-3">
              <div className="flex items-center gap-2 text-white/80">
                <KeyRound className="h-4 w-4 text-cyan-200" />
                <span className="text-white/60">Missing secrets</span>
              </div>
              <p className="text-2xl font-semibold text-white">{heroStats?.missing_secrets ?? 0}</p>
              <p className="text-[11px] text-white/50">Still missing values block deployment across tiers.</p>
            </div>
          </div>
          {blockedTiers.length > 0 ? (
            <p className="text-xs text-white/60">
              Blocked tiers = environments that can’t ship because strategies are missing. Blocked:{" "}
              {blockedTiers.map((tier) => tier.label).slice(0, 3).join(", ")}
              {blockedTiers.length > 3 ? "..." : ""}
            </p>
          ) : (
            <p className="text-xs text-white/60">
              All tiers show full strategy coverage. Next step: generate/export a manifest for your target tier.
            </p>
          )}
          <p className="text-sm text-white/70">
            When to start: any blocked tier or missing secret → run Prep Deployment now. If none are blocked, you can jump straight
            to manifest export.
          </p>
          <div className="flex flex-wrap gap-2">
            <Button size="sm" onClick={() => onStartTutorial?.("prep-deployment")}>
              Start guided tutorial
            </Button>
            <Button size="sm" variant="outline" onClick={() => onNavigateTab?.("deployment")}>
              Open Deployment tab
            </Button>
          </div>
        </div>
      )
    });
    steps.push({
      title: "Choose a scenario campaign",
      description: "Use the campaigns table to select the scenario/tier combo to work on.",
      content: (
        <div className="space-y-2 text-sm text-white/70">
          <p>Open the Deployment tab and pick a campaign row. The summary shows coverage and required actions.</p>
          <p className="text-xs text-white/60">Selecting a row keeps your place in the stepper and readiness views.</p>
          <Button size="sm" onClick={() => onNavigateTab?.("deployment")}>
            Go to campaigns
          </Button>
        </div>
      )
    });
    steps.push({
      title: "Walk the campaign stepper",
      description: "Use the stepper to jump between primer, readiness, blockers, and export.",
      content: (
        <div className="space-y-2 text-sm text-white/70">
          <p>Stay in Deployment and use the stepper buttons to move between readiness and manifest actions.</p>
          <p className="text-xs text-white/60">You can click any step directly—no need to follow a strict sequence.</p>
        </div>
      )
    });
    steps.push({
      title: "Run readiness and clear blockers",
      description: "Open the readiness panel to see tier coverage and blocking secrets.",
      content: (
        <div className="space-y-3 text-sm text-white/70">
          <p>Check the readiness cards for the selected tier. Use the “Open resource workbench” action on blockers.</p>
          <p className="text-xs text-white/60">If blockers remain, switch to the Resources tab to define missing strategies.</p>
          <Button size="sm" variant="outline" onClick={() => onOpenResource?.()}>
            Open top blocker
          </Button>
        </div>
      )
    });
    steps.push({
      title: "Generate and export manifest",
      description: "Create the manifest for the selected campaign and hand it to deployment-manager.",
      content: (
        <div className="space-y-2 text-sm text-white/70">
          <p>Use the manifest form to generate, then export the JSON. Re-run after any strategy updates.</p>
          <p className="text-xs text-white/60">Keep this step in view to confirm coverage numbers before handing off.</p>
        </div>
      )
    });
  }

  return steps;
};
