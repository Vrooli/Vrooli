import { Layers, AlertCircle, Rocket, PlayCircle, FileOutput, Download, Search } from "lucide-react";
import { Button } from "../components/ui/button";
import { LoadingStatCard } from "../components/ui/LoadingStates";
import { HelpDialog } from "../components/ui/HelpDialog";
import type { DeploymentManifestResponse } from "../lib/api";

interface TierReadinessData {
  tier: string;
  label: string;
  ready_percent: number;
  strategized: number;
  total: number;
  blocking_secret_sample: string[];
}

interface ResourceInsight {
  resource_name: string;
  total_secrets: number;
  valid_secrets: number;
  missing_secrets: number;
}

interface TierReadinessProps {
  tierReadiness: TierReadinessData[];
  isLoading: boolean;
  onOpenResource: (resourceName: string, secretKey?: string) => void;
  resourceInsights: ResourceInsight[];
  resourceStatuses: Array<{
    resource_name: string;
    secrets_total: number;
    secrets_found: number;
    secrets_missing: number;
    secrets_optional: number;
    health_status: string;
    last_checked: string;
  }>;
}

const getFirstResourceNeedingAttention = (insights: ResourceInsight[]) => {
  const resourceWithMissing = insights.find((resource) => resource.missing_secrets > 0);
  return resourceWithMissing?.resource_name ?? insights[0]?.resource_name;
};

interface DeploymentFlowState {
  scenario: string;
  tier: string;
  resourcesInput: string;
  manifestData?: DeploymentManifestResponse;
  manifestIsLoading: boolean;
  manifestIsError: boolean;
  manifestError?: Error;
  onSetScenario: (value: string) => void;
  onSetTier: (value: string) => void;
  onSetResourcesInput: (value: string) => void;
  onGenerateManifest: () => void;
}

interface DeploymentReadinessPanelProps {
  tierReadiness: TierReadinessData[];
  resourceInsights: ResourceInsight[];
  manifestState: DeploymentFlowState;
  onOpenResource: (resourceName: string, secretKey?: string) => void;
  onStartJourney?: () => void;
}

const suggestedScenarios = [
  "secrets-manager",
  "deployment-manager",
  "ecosystem-manager",
  "scenario-auditor",
  "scenario-authenticator",
  "scenario-dependency-analyzer",
  "system-monitor",
  "app-issue-tracker",
  "brand-manager"
];

export const DeploymentReadinessPanel = ({
  tierReadiness,
  resourceInsights,
  manifestState,
  onOpenResource,
  onStartJourney
}: DeploymentReadinessPanelProps) => {
  const blockedTiers = tierReadiness.filter((tier) => tier.ready_percent < 100 || tier.strategized < tier.total);
  const topBlockedTier = blockedTiers.sort((a, b) => a.ready_percent - b.ready_percent)[0];
  const firstResourceNeedingAttention = getFirstResourceNeedingAttention(resourceInsights);
  const manifestSummary = manifestState.manifestData?.summary;
  const filteredSuggestions = suggestedScenarios.filter((scenario) =>
    scenario.toLowerCase().includes(manifestState.scenario.toLowerCase())
  );

  const handleExportManifest = () => {
    if (!manifestState.manifestData) return;
    const blob = new Blob([JSON.stringify(manifestState.manifestData, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `${manifestState.manifestData.scenario}-manifest-${manifestState.manifestData.tier}.json`;
    link.click();
    URL.revokeObjectURL(url);
  };

  return (
    <section className="rounded-3xl border border-white/10 bg-white/5 p-6">
      <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.3em] text-emerald-300">Deployment Readiness</p>
          <h2 className="mt-1 text-2xl font-semibold text-white">Scenario deployment quickstart</h2>
          <p className="mt-1 max-w-2xl text-sm text-white/70">
            Resource readiness answers “are my resources strategized per tier?” Scenario readiness answers “can this
            scenario ship to a target tier right now?”. Use this panel to generate manifests and clear blockers per scenario.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          {onStartJourney ? (
            <Button variant="secondary" size="sm" className="gap-2" onClick={onStartJourney}>
              <PlayCircle className="h-4 w-4" />
              Guided prep flow
            </Button>
          ) : null}
          {firstResourceNeedingAttention ? (
            <Button variant="outline" size="sm" className="gap-2" onClick={() => onOpenResource(firstResourceNeedingAttention)}>
              <Rocket className="h-4 w-4" />
              Open top blocker
            </Button>
          ) : null}
        </div>
      </div>

      <div className="mt-4 grid gap-4 md:grid-cols-3">
        <div className="rounded-2xl border border-emerald-400/30 bg-emerald-500/5 p-4">
          <p className="text-[11px] uppercase tracking-[0.2em] text-emerald-200/80">Tiers tracked</p>
          <p className="mt-1 text-3xl font-semibold text-white">{tierReadiness.length || 0}</p>
          <p className="text-xs text-white/60">Vrooli Tier 1-5 coverage</p>
        </div>
        <div className="rounded-2xl border border-cyan-400/30 bg-cyan-500/5 p-4">
          <p className="text-[11px] uppercase tracking-[0.2em] text-cyan-200/80">Blocks to clear</p>
          <p className="mt-1 text-3xl font-semibold text-white">
            {blockedTiers.length > 0 ? `${blockedTiers.length} tier${blockedTiers.length === 1 ? "" : "s"}` : "None"}
          </p>
          <p className="text-xs text-white/60">
            {topBlockedTier
              ? `${topBlockedTier.label} at ${topBlockedTier.ready_percent}% (${topBlockedTier.strategized}/${topBlockedTier.total})`
              : "No blocking tiers detected"}
          </p>
        </div>
        <div className="rounded-2xl border border-amber-400/30 bg-amber-500/5 p-4">
          <p className="text-[11px] uppercase tracking-[0.2em] text-amber-200/80">Next action</p>
          <p className="mt-1 text-lg font-semibold text-white">
            {firstResourceNeedingAttention ? `Configure ${firstResourceNeedingAttention}` : "Generate manifest"}
          </p>
          <p className="text-xs text-white/60">
            {firstResourceNeedingAttention
              ? "Open the resource panel to assign strategies for blocked tiers."
              : "Use the manifest generator to verify bundle coverage."}
          </p>
        </div>
      </div>

      <div className="mt-5 grid gap-4 md:grid-cols-2">
        <div className="rounded-2xl border border-white/10 bg-black/30 p-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-[11px] uppercase tracking-[0.2em] text-white/60">Manifest generator</p>
              <p className="text-sm text-white/70">Select scope and emit deployment bundle</p>
            </div>
            <FileOutput className="h-4 w-4 text-white/60" />
          </div>
          <div className="mt-3 space-y-3 text-sm text-white/80">
            <label className="flex flex-col gap-1 text-[11px] uppercase tracking-[0.2em] text-white/60">
              Scenario
              <div className="relative">
                <input
                  type="text"
                  value={manifestState.scenario}
                  onChange={(event) => manifestState.onSetScenario(event.target.value)}
                  className="w-full rounded-2xl border border-white/10 bg-white/5 px-3 py-2 pr-10 text-sm text-white"
                  placeholder="picker-wheel, scenario-name"
                />
                <Search className="pointer-events-none absolute right-3 top-2.5 h-4 w-4 text-white/40" />
              </div>
              <div className="mt-2 flex flex-wrap gap-2">
                {(filteredSuggestions.length > 0 ? filteredSuggestions : suggestedScenarios.slice(0, 4)).map((scenario) => (
                  <button
                    key={scenario}
                    type="button"
                    className="rounded-full border border-white/15 bg-white/5 px-3 py-1 text-[11px] uppercase tracking-[0.2em] text-white/70 hover:border-white/40"
                    onClick={() => manifestState.onSetScenario(scenario)}
                  >
                    {scenario}
                  </button>
                ))}
              </div>
            </label>
            <label className="flex flex-col gap-1 text-[11px] uppercase tracking-[0.2em] text-white/60">
              Tier
              <select
                value={manifestState.tier}
                onChange={(event) => manifestState.onSetTier(event.target.value)}
                className="rounded-2xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
              >
                {tierReadiness.map((tier) => (
                  <option key={tier.tier} value={tier.tier}>
                    {tier.label}
                  </option>
                ))}
              </select>
            </label>
            <label className="flex flex-col gap-1 text-[11px] uppercase tracking-[0.2em] text-white/60">
              Resources (optional)
              <textarea
                value={manifestState.resourcesInput}
                onChange={(event) => manifestState.onSetResourcesInput(event.target.value)}
                placeholder="postgres, vault"
                rows={2}
                className="rounded-2xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
              />
            </label>
            <Button
              variant="secondary"
              size="sm"
              className="w-full"
              onClick={manifestState.onGenerateManifest}
              disabled={manifestState.manifestIsLoading}
            >
              {manifestState.manifestIsLoading ? "Generating..." : "Generate manifest"}
            </Button>
            {manifestState.manifestIsError ? (
              <div className="rounded-xl border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-100">
                {manifestState.manifestError?.message || "Manifest generation failed"}
              </div>
            ) : null}
          </div>
        </div>
        <div className="rounded-2xl border border-white/10 bg-black/30 p-4">
          <p className="text-[11px] uppercase tracking-[0.2em] text-white/60">Manifest status</p>
          {manifestState.manifestIsLoading ? (
              <div className="mt-3 flex items-center gap-3 rounded-xl border border-cyan-400/30 bg-cyan-400/5 px-3 py-2">
                <div className="h-5 w-5 animate-spin rounded-full border-2 border-cyan-400 border-t-transparent" />
                <p className="text-sm text-cyan-50">Analyzing dependencies and strategies...</p>
              </div>
            ) : manifestSummary ? (
            <div className="mt-3 space-y-3">
              <div className="grid grid-cols-3 gap-2 text-xs text-white/70">
                <div className="rounded-xl border border-emerald-400/30 bg-emerald-500/5 px-3 py-2">
                  <p className="text-[10px] uppercase tracking-[0.2em] text-emerald-200/80">Covered</p>
                  <p className="text-lg font-semibold text-white">
                    {manifestSummary.strategized_secrets}/{manifestSummary.total_secrets}
                  </p>
                </div>
                <div className="rounded-xl border border-amber-400/30 bg-amber-500/5 px-3 py-2">
                  <p className="text-[10px] uppercase tracking-[0.2em] text-amber-200/80">Requires action</p>
                  <p className="text-lg font-semibold text-white">{manifestSummary.requires_action}</p>
                </div>
                <div className="rounded-xl border border-white/20 bg-white/5 px-3 py-2">
                  <p className="text-[10px] uppercase tracking-[0.2em] text-white/60">Scope</p>
                  <p className="text-lg font-semibold text-white">{manifestState.manifestData?.resources.length ?? 0}</p>
                </div>
              </div>
              <div className="flex flex-wrap items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  className="gap-2"
                  onClick={handleExportManifest}
                >
                  <Download className="h-4 w-4" />
                  Export manifest (json)
                </Button>
                <p className="text-[11px] text-white/50">Hand off to deployment-manager or scenario-to-*</p>
              </div>
              {manifestSummary.blocking_secrets?.length ? (
                <div className="rounded-xl border border-amber-400/30 bg-amber-500/5 px-3 py-2 text-xs text-amber-100">
                  <p className="font-semibold">Blockers</p>
                  <p className="text-[11px] text-amber-100/80">
                    {manifestSummary.blocking_secrets.slice(0, 3).join(", ")}
                    {manifestSummary.blocking_secrets.length > 3 ? " ..." : ""}
                  </p>
                </div>
              ) : (
                <div className="rounded-xl border border-emerald-400/30 bg-emerald-500/5 px-3 py-2 text-xs text-emerald-100">
                  ✓ No blocking secrets detected
                </div>
              )}
              <div className="rounded-xl border border-white/10 bg-white/5 px-3 py-2 text-[11px] text-white/60">
                Generated {manifestState.manifestData?.generated_at ? new Date(manifestState.manifestData.generated_at).toLocaleString() : "just now"}
              </div>
            </div>
          ) : (
            <div className="mt-3 rounded-xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white/60">
              No manifest generated yet. Use the form to produce a bundle preview for deployment-manager or scenario-to-*.
            </div>
          )}
        </div>
      </div>
    </section>
  );
};

export const TierReadiness = ({
  tierReadiness,
  isLoading,
  onOpenResource,
  resourceInsights,
  resourceStatuses
}: TierReadinessProps) => {
  const firstResourceNeedingAttention = getFirstResourceNeedingAttention(resourceInsights);
  const totalRequired = tierReadiness[0]?.total ?? 0;
  const missingCount = resourceStatuses.reduce((acc, status) => acc + status.secrets_missing, 0);

  return (
    <section>
      <div className="mb-4 flex items-center gap-3">
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <h2 className="text-xl font-semibold text-white">Deployment Readiness by Tier</h2>
            <HelpDialog title="Deployment Tiers">
              <p>
                Vrooli supports 5 deployment tiers, each with different secret handling requirements:
              </p>
              <ul className="mt-2 space-y-2">
                <li>
                  <strong className="text-white">Tier 1 - Local/Developer:</strong> Full access to infrastructure secrets, runs on your development machine
                </li>
                <li>
                  <strong className="text-white">Tier 2 - Desktop:</strong> Bundled app for end users, infrastructure secrets stripped, service secrets generated
                </li>
                <li>
                  <strong className="text-white">Tier 3 - Mobile:</strong> iOS/Android apps, similar to desktop but with mobile-specific constraints
                </li>
                <li>
                  <strong className="text-white">Tier 4 - SaaS/Cloud:</strong> Remote server deployment, infrastructure secrets delegated to cloud providers
                </li>
                <li>
                  <strong className="text-white">Tier 5 - Enterprise/Appliance:</strong> Self-hosted hardware, customer manages infrastructure secrets
                </li>
              </ul>
              <p className="mt-3">
                The readiness percentage shows how many required secrets have deployment strategies defined for each tier.
              </p>
            </HelpDialog>
          </div>
          <p className="mt-1 text-sm text-white/60">
            Resource readiness = % of required secrets with strategies for each tier across all resources ({totalRequired} total required).
            Missing secrets across all resources: {missingCount}.
          </p>
        </div>
      </div>
      <div className="grid gap-4 md:grid-cols-3">
        {isLoading ? (
          <>
            <LoadingStatCard />
            <LoadingStatCard />
            <LoadingStatCard />
          </>
        ) : (
          tierReadiness.map((tier) => {
            const blockedCount = tier.total - tier.strategized;
            const isBlocked = blockedCount > 0;
            const hasBlockingSample = tier.blocking_secret_sample.length > 0;

            return (
              <div key={tier.tier} className="rounded-3xl border border-white/10 bg-white/5 p-4">
                <div className="flex items-center justify-between text-xs uppercase tracking-[0.2em] text-white/60">
                  <span>{tier.label}</span>
                  <Layers className="h-4 w-4 text-white/60" />
                </div>
                <p className="mt-2 text-3xl font-semibold text-white">{tier.ready_percent}%</p>
                <p className="text-sm text-white/60">
                  {tier.strategized}/{tier.total} required secrets covered
                </p>
                {isBlocked ? (
                  <div className="mt-3 space-y-2">
                    <div className="flex items-center gap-2 rounded-xl border border-amber-400/30 bg-amber-400/5 px-3 py-2">
                      <AlertCircle className="h-4 w-4 text-amber-300" />
                      <p className="text-xs text-amber-100">
                        {blockedCount} secret{blockedCount !== 1 ? "s" : ""} blocking deployment
                      </p>
                    </div>
                    {hasBlockingSample && firstResourceNeedingAttention && (
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => onOpenResource(firstResourceNeedingAttention)}
                        className="w-full text-xs"
                      >
                        Configure Strategies →
                      </Button>
                    )}
                  </div>
                ) : (
                  <div className="mt-3 rounded-xl border border-emerald-400/30 bg-emerald-400/5 px-3 py-2 text-xs text-emerald-100">
                    ✓ Ready for deployment
                  </div>
                )}
              </div>
            );
          })
        )}
      </div>
    </section>
  );
};
