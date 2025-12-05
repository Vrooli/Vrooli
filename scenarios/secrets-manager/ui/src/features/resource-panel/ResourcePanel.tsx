import { ChevronDown } from "lucide-react";
import { Button } from "../../components/ui/button";
import { SecretDetail } from "./SecretDetail";
import { VulnerabilityList } from "./VulnerabilityList";
import type { ResourceDetail, UpdateResourceSecretPayload, UpdateSecretStrategyPayload } from "../../lib/api";

interface ResourceStatus {
  resource_name: string;
  secrets_total: number;
  secrets_found: number;
  secrets_missing: number;
  health_status: string;
}

interface ResourcePanelProps {
  activeResource: string;
  resourceDetail?: ResourceDetail;
  isLoading: boolean;
  isFetching: boolean;
  selectedSecretKey: string | null;
  strategyTier: string;
  strategyHandling: string;
  strategyPrompt: string;
  strategyDescription: string;
  tierReadiness: Array<{ tier: string; label: string }>;
  allResources: ResourceStatus[];
  onClose: () => void;
  onSwitchResource: (resourceName: string) => void;
  onSelectSecret: (secretKey: string) => void;
  onUpdateSecret: (secretKey: string, payload: UpdateResourceSecretPayload) => void;
  onApplyStrategy: () => void;
  onUpdateVulnerabilityStatus: (id: string, status: string) => void;
  onSetStrategyTier: (value: string) => void;
  onSetStrategyHandling: (value: string) => void;
  onSetStrategyPrompt: (value: string) => void;
  onSetStrategyDescription: (value: string) => void;
}

export const ResourcePanel = ({
  activeResource,
  resourceDetail,
  isLoading,
  isFetching,
  selectedSecretKey,
  strategyTier,
  strategyHandling,
  strategyPrompt,
  strategyDescription,
  tierReadiness,
  allResources,
  onClose,
  onSwitchResource,
  onSelectSecret,
  onUpdateSecret,
  onApplyStrategy,
  onUpdateVulnerabilityStatus,
  onSetStrategyTier,
  onSetStrategyHandling,
  onSetStrategyPrompt,
  onSetStrategyDescription
}: ResourcePanelProps) => {
  const selectedSecret = resourceDetail?.secrets.find((secret) => secret.secret_key === selectedSecretKey) ??
    resourceDetail?.secrets[0];

  // Sort resources: those with missing secrets first, then alphabetically
  const sortedResources = [...allResources].sort((a, b) => {
    if (a.secrets_missing > 0 && b.secrets_missing === 0) return -1;
    if (a.secrets_missing === 0 && b.secrets_missing > 0) return 1;
    return a.resource_name.localeCompare(b.resource_name);
  });

  const currentResourceIdx = sortedResources.findIndex(r => r.resource_name === activeResource);
  const resourcesWithIssues = sortedResources.filter(r => r.secrets_missing > 0);

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-black/80 px-4 py-10">
      <div className="w-full max-w-5xl rounded-3xl border border-white/10 bg-slate-950/95 p-6 shadow-2xl shadow-emerald-500/20 max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-xs uppercase tracking-[0.3em] text-emerald-300">Resource Workbench</p>
            <div className="mt-1 flex items-center gap-3">
              <div className="relative">
                <select
                  value={activeResource}
                  onChange={(e) => onSwitchResource(e.target.value)}
                  className="appearance-none rounded-xl border border-white/20 bg-slate-800 py-2 pl-4 pr-10 text-2xl font-semibold text-white hover:border-white/40 focus:border-emerald-500/60 focus:outline-none [&_option]:bg-slate-800 [&_option]:text-white [&_option]:text-base [&_option]:font-normal"
                >
                  {sortedResources.map((resource) => (
                    <option key={resource.resource_name} value={resource.resource_name}>
                      {resource.resource_name} {resource.secrets_missing > 0 ? `(${resource.secrets_missing} missing)` : ""}
                    </option>
                  ))}
                </select>
                <ChevronDown className="pointer-events-none absolute right-3 top-1/2 h-5 w-5 -translate-y-1/2 text-white/60" />
              </div>
              {resourcesWithIssues.length > 1 && (
                <span className="rounded-full border border-amber-400/30 bg-amber-400/10 px-3 py-1 text-xs text-amber-200">
                  {currentResourceIdx + 1} of {resourcesWithIssues.length} needing attention
                </span>
              )}
            </div>
            <p className="mt-1 text-sm text-white/60">
              {resourceDetail?.valid_secrets ?? 0}/{resourceDetail?.total_secrets ?? 0} secrets valid · Missing {resourceDetail?.missing_secrets ?? 0}
            </p>
          </div>
          <Button variant="ghost" onClick={onClose} className="text-white/70">
            Close
          </Button>
        </div>
        <div className="mt-6 grid gap-6 lg:grid-cols-[1.2fr,1fr]">
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <h4 className="text-sm uppercase tracking-[0.3em] text-white/60">Secrets</h4>
              {isFetching ? (
                <span className="text-xs text-white/40">Syncing…</span>
              ) : null}
            </div>
            <div className="space-y-2 rounded-2xl border border-white/10 bg-black/30 p-3 max-h-[55vh] overflow-y-auto">
              {isLoading || !resourceDetail ? (
                <p className="text-sm text-white/60">Loading secrets…</p>
              ) : resourceDetail.secrets.length ? (
                resourceDetail.secrets.map((secret) => (
                  <button
                    key={secret.id}
                    onClick={() => onSelectSecret(secret.secret_key)}
                    className={`w-full rounded-xl border px-3 py-2 text-left ${
                      selectedSecretKey === secret.secret_key
                        ? "border-emerald-500/60 bg-emerald-500/10"
                        : "border-white/10 bg-white/5"
                    }`}
                  >
                    <p className="font-mono text-xs text-white">{secret.secret_key}</p>
                    <p className="text-[11px] text-white/60">{secret.description || secret.secret_type}</p>
                    <div className="mt-2 flex items-center justify-between text-[11px] text-white/60">
                      <span>
                        {secret.classification} · {secret.required ? "Required" : "Optional"}
                      </span>
                      <span>{secret.validation_state}</span>
                    </div>
                  </button>
                ))
              ) : (
                <div className="space-y-2 rounded-xl border border-dashed border-white/20 bg-white/5 p-3 text-sm text-white/70">
                  <p className="font-semibold text-white">No secrets were returned for this resource.</p>
                  <p className="text-white/70">
                    If this resource has secrets, make sure its <code>config/secrets.yaml</code> is present under{" "}
                    <code>resources/{activeResource}/</code> and sync with Vault via{" "}
                    <code>resource-vault secrets check {activeResource}</code> or <code>secrets-manager</code> refresh.
                  </p>
                  <p className="text-white/70">
                    Need to add declarations? Run <code>resource-vault secrets create-template {activeResource}</code>,
                    edit the file, then <code>resource-vault secrets init {activeResource}</code> to populate Vault.
                  </p>
                </div>
              )}
            </div>
          </div>
          <div className="space-y-4">
            <SecretDetail
              selectedSecret={selectedSecret}
              tierReadiness={tierReadiness}
              strategyTier={strategyTier}
              strategyHandling={strategyHandling}
              strategyPrompt={strategyPrompt}
              strategyDescription={strategyDescription}
              onUpdateSecret={onUpdateSecret}
              onApplyStrategy={onApplyStrategy}
              onSetStrategyTier={onSetStrategyTier}
              onSetStrategyHandling={onSetStrategyHandling}
              onSetStrategyPrompt={onSetStrategyPrompt}
              onSetStrategyDescription={onSetStrategyDescription}
            />
            <VulnerabilityList
              vulnerabilities={resourceDetail?.open_vulnerabilities ?? []}
              onUpdateStatus={onUpdateVulnerabilityStatus}
            />
          </div>
        </div>
      </div>
    </div>
  );
};
