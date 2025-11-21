import { Button } from "../../components/ui/button";
import type { ResourceSecretDetail, UpdateResourceSecretPayload } from "../../lib/api";

interface SecretDetailProps {
  selectedSecret?: ResourceSecretDetail;
  tierReadiness: Array<{ tier: string; label: string }>;
  strategyTier: string;
  strategyHandling: string;
  strategyPrompt: string;
  strategyDescription: string;
  onUpdateSecret: (secretKey: string, payload: UpdateResourceSecretPayload) => void;
  onApplyStrategy: () => void;
  onSetStrategyTier: (value: string) => void;
  onSetStrategyHandling: (value: string) => void;
  onSetStrategyPrompt: (value: string) => void;
  onSetStrategyDescription: (value: string) => void;
}

export const SecretDetail = ({
  selectedSecret,
  tierReadiness,
  strategyTier,
  strategyHandling,
  strategyPrompt,
  strategyDescription,
  onUpdateSecret,
  onApplyStrategy,
  onSetStrategyTier,
  onSetStrategyHandling,
  onSetStrategyPrompt,
  onSetStrategyDescription
}: SecretDetailProps) => (
  <div className="rounded-2xl border border-white/10 bg-black/30 p-4">
    <h4 className="text-sm uppercase tracking-[0.3em] text-white/60">Secret Detail</h4>
    {selectedSecret ? (
      <div className="mt-3 space-y-4">
        <div>
          <p className="font-mono text-sm text-white">{selectedSecret.secret_key}</p>
          <p className="text-xs text-white/60">{selectedSecret.description || selectedSecret.secret_type}</p>
        </div>
        <label className="text-xs uppercase tracking-[0.2em] text-white/60">
          Classification
          <select
            value={selectedSecret.classification}
            onChange={(event) => onUpdateSecret(selectedSecret.secret_key, { classification: event.target.value })}
            className="mt-1 w-full rounded-xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
          >
            <option value="infrastructure">Infrastructure</option>
            <option value="service">Service</option>
            <option value="user">User</option>
          </select>
        </label>
        <div className="flex items-center justify-between rounded-xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white">
          <span>Required secret</span>
          <Button
            variant="outline"
            size="sm"
            onClick={() => onUpdateSecret(selectedSecret.secret_key, { required: !selectedSecret.required })}
          >
            {selectedSecret.required ? "Mark optional" : "Mark required"}
          </Button>
        </div>
        <div>
          <p className="text-xs uppercase tracking-[0.2em] text-white/60">Tier strategies</p>
          {Object.entries(selectedSecret.tier_strategies || {}).length ? (
            <ul className="mt-2 space-y-1 text-xs text-white/70">
              {Object.entries(selectedSecret.tier_strategies || {}).map(([tier, strategy]) => (
                <li key={`${tier}-${strategy}`}>{tier}: {strategy}</li>
              ))}
            </ul>
          ) : (
            <p className="mt-1 text-xs text-amber-200">No tier strategies recorded</p>
          )}
        </div>
        <div className="space-y-2">
          <p className="text-xs uppercase tracking-[0.2em] text-white/60">Add / update strategy</p>
          <div className="grid gap-2">
            <label className="text-xs text-white/60">
              Deployment Tier
              <select
                value={strategyTier}
                onChange={(event) => onSetStrategyTier(event.target.value)}
                className="mt-1 w-full rounded-xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
              >
                {tierReadiness.map((tier) => (
                  <option key={tier.tier} value={tier.tier}>
                    {tier.label}
                  </option>
                ))}
              </select>
            </label>
            <label className="text-xs text-white/60">
              Handling Strategy
              <select
                value={strategyHandling}
                onChange={(event) => onSetStrategyHandling(event.target.value)}
                className="mt-1 w-full rounded-xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
              >
                <option value="prompt">Prompt (ask user for value)</option>
                <option value="generate">Generate (auto-create at build time)</option>
                <option value="strip">Strip (exclude from bundle)</option>
                <option value="delegate">Delegate (use cloud provider secret)</option>
              </select>
            </label>
            {strategyHandling === "prompt" && (
              <>
                <label className="text-xs text-white/60">
                  Prompt Label <span className="text-amber-300">*</span>
                  <input
                    value={strategyPrompt}
                    onChange={(event) => onSetStrategyPrompt(event.target.value)}
                    className="mt-1 w-full rounded-xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
                    placeholder="e.g., Database Password"
                    required
                  />
                </label>
                <label className="text-xs text-white/60">
                  Prompt Description <span className="text-amber-300">*</span>
                  <textarea
                    value={strategyDescription}
                    onChange={(event) => onSetStrategyDescription(event.target.value)}
                    className="mt-1 w-full rounded-xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
                    placeholder="Explain what this secret is used for and how to obtain it"
                    rows={2}
                    required
                  />
                </label>
              </>
            )}
            {strategyHandling === "generate" && (
              <div className="rounded-xl border border-cyan-400/30 bg-cyan-400/5 px-3 py-2 text-xs text-cyan-100">
                ℹ️ This secret will be auto-generated at deployment time using a secure random generator.
              </div>
            )}
            {strategyHandling === "strip" && (
              <div className="rounded-xl border border-amber-400/30 bg-amber-400/5 px-3 py-2 text-xs text-amber-100">
                ⚠️ This secret will be excluded from the deployment bundle. Ensure the feature doesn't require it.
              </div>
            )}
            {strategyHandling === "delegate" && (
              <div className="rounded-xl border border-purple-400/30 bg-purple-400/5 px-3 py-2 text-xs text-purple-100">
                ☁️ This secret will be managed by the cloud provider (e.g., AWS Secrets Manager, Vault).
              </div>
            )}
            <Button
              size="sm"
              onClick={onApplyStrategy}
              disabled={strategyHandling === "prompt" && (!strategyPrompt.trim() || !strategyDescription.trim())}
            >
              Apply strategy
            </Button>
          </div>
        </div>
      </div>
    ) : (
      <p className="mt-3 text-sm text-white/60">Select a secret to view details.</p>
    )}
  </div>
);
