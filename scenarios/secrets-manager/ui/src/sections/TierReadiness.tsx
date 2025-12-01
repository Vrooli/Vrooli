import { Layers, AlertCircle } from "lucide-react";
import { Button } from "../components/ui/button";
import { LoadingStatCard } from "../components/ui/LoadingStates";
import { HelpDialog } from "../components/ui/HelpDialog";

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
}

export const TierReadiness = ({ tierReadiness, isLoading, onOpenResource, resourceInsights }: TierReadinessProps) => {
  // Get the first resource that needs attention (has missing secrets or needs strategies)
  const getFirstResourceNeedingAttention = () => {
    // Prioritize resources with missing secrets
    const resourceWithMissing = resourceInsights.find(r => r.missing_secrets > 0);
    if (resourceWithMissing) return resourceWithMissing.resource_name;

    // Fall back to any resource (they all likely need tier strategies defined)
    return resourceInsights[0]?.resource_name;
  };

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
          Secret coverage for each deployment target
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
                    {blockedCount} secret{blockedCount !== 1 ? 's' : ''} blocking deployment
                  </p>
                </div>
                {tier.blocking_secret_sample.length > 0 && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      const resource = getFirstResourceNeedingAttention();
                      if (resource) {
                        onOpenResource(resource);
                      }
                    }}
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
