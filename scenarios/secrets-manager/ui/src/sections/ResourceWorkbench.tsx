import { Target } from "lucide-react";
import { useMemo, useState } from "react";
import { Button } from "../components/ui/button";
import { LoadingResourceCard } from "../components/ui/LoadingStates";
import { HelpDialog } from "../components/ui/HelpDialog";
import { classificationTone } from "../lib/constants";

interface ResourceSecret {
  secret_key: string;
  secret_type: string;
  classification: string;
  tier_strategies?: Record<string, string>;
}

interface ResourceInsight {
  resource_name: string;
  total_secrets: number;
  valid_secrets: number;
  missing_secrets: number;
  secrets: ResourceSecret[];
}

interface ResourceWorkbenchProps {
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
  isLoading: boolean;
  onOpenResource: (resourceName: string, secretKey?: string) => void;
}

const SecretBadge = ({ classification }: { classification: string }) => (
  <span
    className={`rounded-full border px-2 py-0.5 text-[10px] uppercase tracking-[0.2em] ${
      classificationTone[classification] ?? "border-white/20"
    }`}
  >
    {classification}
  </span>
);

const ResourceSecrets = ({ secrets }: { secrets: ResourceSecret[] }) => (
  <div className="mt-3 space-y-2">
    {secrets.map((secret) => {
      const tierStrategies = secret.tier_strategies ?? {};
      const hasTierStrategies = Object.keys(tierStrategies).length > 0;

      return (
        <div key={secret.secret_key} className="rounded-xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white/80">
          <div className="flex items-center justify-between">
            <span className="font-mono text-xs text-white">{secret.secret_key}</span>
            <SecretBadge classification={secret.classification} />
          </div>
          <p className="text-xs text-white/50">{secret.secret_type}</p>
          {hasTierStrategies ? (
            <p className="text-[10px] text-white/50">
              Strategies: {Object.entries(tierStrategies).map(([tier, value]) => `${tier}:${value}`).join(" · ")}
            </p>
          ) : (
            <p className="text-[10px] text-amber-300">No tier strategies defined</p>
          )}
        </div>
      );
    })}
  </div>
);

const ResourceCard = ({ resource, onOpenResource }: { resource: ResourceInsight; onOpenResource: ResourceWorkbenchProps["onOpenResource"] }) => (
  <div className="rounded-2xl border border-white/10 bg-black/30 p-4">
    <div className="flex items-center justify-between">
      <div>
        <p className="text-lg font-semibold text-white">{resource.resource_name}</p>
        <p className="text-xs text-white/60">
          {resource.valid_secrets}/{resource.total_secrets} valid · Missing {resource.missing_secrets}
        </p>
      </div>
      <Button
        variant="outline"
        size="sm"
        className="text-[10px] uppercase tracking-[0.2em]"
        onClick={() => onOpenResource(resource.resource_name)}
      >
        Manage
      </Button>
    </div>
    <ResourceSecrets secrets={resource.secrets} />
  </div>
);

export const ResourceWorkbench = ({ resourceInsights, resourceStatuses, isLoading, onOpenResource }: ResourceWorkbenchProps) => {
  const [showAllResources, setShowAllResources] = useState(false);
  const sortedStatuses = useMemo(
    () =>
      [...resourceStatuses].sort((a, b) => {
        if (a.secrets_missing === b.secrets_missing) return a.resource_name.localeCompare(b.resource_name);
        return b.secrets_missing - a.secrets_missing;
      }),
    [resourceStatuses]
  );

  const renderContent = () => {
    if (isLoading) {
      return (
        <>
          <LoadingResourceCard />
          <LoadingResourceCard />
          <LoadingResourceCard />
          <LoadingResourceCard />
        </>
      );
    }

    if (resourceInsights.length === 0) {
      return (
        <div className="col-span-2 rounded-2xl border border-white/10 bg-white/5 p-6 text-center text-sm text-white/60">
          No resource insights available. API data is still loading or no resources are configured.
        </div>
      );
    }

    return resourceInsights.map((resource) => (
      <ResourceCard key={resource.resource_name} resource={resource} onOpenResource={onOpenResource} />
    ));
  };

  return (
    <section className="rounded-3xl border border-white/10 bg-white/5 p-6">
      <div className="flex items-center justify-between">
        <div className="flex-1">
        <div className="flex items-center gap-2">
          <h2 className="text-2xl font-semibold text-white">Per-Resource Secret Management</h2>
          <HelpDialog title="Resource Workbench">
            <p>
              Each resource (postgres, redis, vault, etc.) requires specific secrets to function. This workbench shows the configuration status per resource.
            </p>
            <div className="mt-3 space-y-2">
                <p><strong className="text-white">Secret Classifications:</strong></p>
                <ul className="ml-4 space-y-1">
                  <li><strong className="text-sky-200">Infrastructure:</strong> Critical secrets like database passwords - should never be bundled in desktop/mobile apps</li>
                  <li><strong className="text-purple-200">Service:</strong> App-level secrets like JWT keys - can be generated during deployment</li>
                  <li><strong className="text-amber-200">User:</strong> User-provided secrets like API keys - prompt user during setup</li>
                </ul>
              </div>
              <p className="mt-3">
                Click <strong className="text-white">Manage</strong> on any resource to configure secret classifications and deployment strategies.
              </p>
            </HelpDialog>
          </div>
          <p className="mt-1 text-sm text-white/60">
            Configure secrets, classifications, and deployment strategies. We surface the top resources needing attention;
            toggle below to see all resources, including healthy ones.
          </p>
        </div>
        <Target className="h-5 w-5 text-white/60" />
      </div>
      <div className="mt-4 grid gap-4 md:grid-cols-2">
        {renderContent()}
      </div>
      <div className="mt-6 rounded-2xl border border-white/10 bg-black/30 p-4">
        <div className="flex items-center justify-between gap-3">
          <div>
            <p className="text-xs uppercase tracking-[0.25em] text-white/60">All resources</p>
            <p className="text-sm text-white/70">
              {sortedStatuses.filter((status) => status.secrets_missing === 0).length}/{sortedStatuses.length || 0} healthy ·
              view full list to confirm coverage.
            </p>
          </div>
          <Button variant="outline" size="sm" onClick={() => setShowAllResources((value) => !value)}>
            {showAllResources ? "Hide" : "Show"} all resources
          </Button>
        </div>
        {showAllResources ? (
          <div className="mt-3 grid gap-2 md:grid-cols-2">
            {sortedStatuses.map((status) => (
              <button
                key={status.resource_name}
                className="flex items-center justify-between rounded-xl border border-white/10 bg-white/5 px-3 py-2 text-left transition hover:border-white/30"
                onClick={() => onOpenResource(status.resource_name)}
              >
                <div>
                  <p className="text-sm font-semibold text-white">{status.resource_name}</p>
                  <p className="text-[11px] text-white/60">
                    {status.secrets_found}/{status.secrets_total} configured · Missing {status.secrets_missing}
                  </p>
                </div>
                <span
                  className={`rounded-full border px-2 py-0.5 text-[10px] uppercase tracking-[0.2em] ${
                    status.secrets_missing > 0 ? "border-amber-400/50 text-amber-100" : "border-emerald-400/50 text-emerald-100"
                  }`}
                >
                  {status.secrets_missing > 0 ? "action needed" : "healthy"}
                </span>
              </button>
            ))}
          </div>
        ) : (
          <p className="mt-3 text-[11px] text-white/50">
            Showing top {resourceInsights.length} resources needing attention. Use "Show all resources" to view everything, including healthy items.
          </p>
        )}
      </div>
    </section>
  );
};
