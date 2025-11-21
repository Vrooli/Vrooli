import { Target } from "lucide-react";
import { Button } from "../components/ui/button";
import { LoadingResourceCard } from "../components/ui/LoadingStates";
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
  isLoading: boolean;
  onOpenResource: (resourceName: string) => void;
}

export const ResourceWorkbench = ({ resourceInsights, isLoading, onOpenResource }: ResourceWorkbenchProps) => (
  <section className="rounded-3xl border border-white/10 bg-white/5 p-6">
    <div className="flex items-center justify-between">
      <div>
        <p className="text-xs uppercase tracking-[0.3em] text-white/60">Resource Workbench</p>
        <h2 className="mt-1 text-2xl font-semibold text-white">Per-Resource Control</h2>
      </div>
      <Target className="h-5 w-5 text-white/60" />
    </div>
    <div className="mt-4 grid gap-4 md:grid-cols-2">
      {isLoading ? (
        <>
          <LoadingResourceCard />
          <LoadingResourceCard />
          <LoadingResourceCard />
          <LoadingResourceCard />
        </>
      ) : resourceInsights.length === 0 ? (
        <div className="col-span-2 rounded-2xl border border-white/10 bg-white/5 p-6 text-center text-sm text-white/60">
          No resource insights available. API data is still loading or no resources are configured.
        </div>
      ) : (
        resourceInsights.map((resource) => (
          <div key={resource.resource_name} className="rounded-2xl border border-white/10 bg-black/30 p-4">
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
            <div className="mt-3 space-y-2">
              {resource.secrets.map((secret) => (
                <div key={secret.secret_key} className="rounded-xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white/80">
                  <div className="flex items-center justify-between">
                    <span className="font-mono text-xs text-white">{secret.secret_key}</span>
                    <span
                      className={`rounded-full border px-2 py-0.5 text-[10px] uppercase tracking-[0.2em] ${
                        classificationTone[secret.classification] ?? "border-white/20"
                      }`}
                    >
                      {secret.classification}
                    </span>
                  </div>
                  <p className="text-xs text-white/50">{secret.secret_type}</p>
                  {Object.keys(secret.tier_strategies || {}).length ? (
                    <p className="text-[10px] text-white/50">
                      Strategies: {Object.entries(secret.tier_strategies || {})
                        .map(([tier, value]) => `${tier}:${value}`)
                        .join(" · ")}
                    </p>
                  ) : (
                    <p className="text-[10px] text-amber-300">No tier strategies defined</p>
                  )}
                </div>
              ))}
            </div>
          </div>
        ))
      )}
    </div>
  </section>
);
