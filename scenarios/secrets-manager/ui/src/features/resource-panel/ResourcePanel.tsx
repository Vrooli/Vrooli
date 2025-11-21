import { Button } from "../../components/ui/button";
import { SecretDetail } from "./SecretDetail";
import { VulnerabilityList } from "./VulnerabilityList";
import type { ResourceDetail, UpdateResourceSecretPayload, UpdateSecretStrategyPayload } from "../../lib/api";

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
  onClose: () => void;
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
  onClose,
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

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center bg-black/80 px-4 py-10">
      <div className="w-full max-w-5xl rounded-3xl border border-white/10 bg-slate-950/95 p-6 shadow-2xl shadow-emerald-500/20">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-xs uppercase tracking-[0.3em] text-emerald-300">Resource Workbench</p>
            <h3 className="text-2xl font-semibold text-white">{activeResource}</h3>
            <p className="text-sm text-white/60">
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
              {isLoading ? (
                <p className="text-sm text-white/60">Loading secrets…</p>
              ) : resourceDetail?.secrets.length ? (
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
                <p className="text-sm text-white/60">No secrets tracked for this resource yet.</p>
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
