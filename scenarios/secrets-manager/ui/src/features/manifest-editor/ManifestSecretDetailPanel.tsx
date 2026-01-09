import { Key, AlertTriangle, CircleDot, ExternalLink } from "lucide-react";
import { Button } from "../../components/ui/button";
import { HelpDialog } from "../../components/ui/HelpDialog";
import type { DeploymentManifestSecret } from "../../lib/api";
import type { OverrideFields, PendingOverrideEdit } from "./types";
import { isSecretBlocking, getStrategyColorClass } from "./utils";

interface ManifestSecretDetailPanelProps {
  secret: DeploymentManifestSecret | null;
  isOverridden: boolean;
  isExcluded: boolean;
  pendingEdit?: PendingOverrideEdit;
  isSaving: boolean;
  isDeleting: boolean;
  onUpdatePendingChange: (changes: Partial<OverrideFields>) => void;
  onSave: () => void;
  onReset: () => void;
  onToggleExclude: () => void;
  onOpenInResourcePanel?: () => void;
}

const STRATEGY_OPTIONS = [
  { value: "prompt", label: "Prompt (ask user for value)" },
  { value: "generate", label: "Generate (auto-create at build time)" },
  { value: "strip", label: "Strip (exclude from bundle)" },
  { value: "delegate", label: "Delegate (use cloud provider secret)" }
];

export function ManifestSecretDetailPanel({
  secret,
  isOverridden,
  isExcluded,
  pendingEdit,
  isSaving,
  isDeleting,
  onUpdatePendingChange,
  onSave,
  onReset,
  onToggleExclude,
  onOpenInResourcePanel
}: ManifestSecretDetailPanelProps) {
  if (!secret) {
    return (
      <div className="flex h-full items-center justify-center text-white/50">
        <div className="text-center">
          <Key className="mx-auto h-10 w-10 opacity-30" />
          <p className="mt-2 text-sm">Select a secret to view details</p>
        </div>
      </div>
    );
  }

  const isBlocking = isSecretBlocking(secret);
  const currentStrategy = pendingEdit?.changes.handling_strategy ?? secret.handling_strategy ?? "";
  const currentFallback = pendingEdit?.changes.fallback_strategy ?? secret.fallback_strategy ?? "";
  const currentRequiresInput = pendingEdit?.changes.requires_user_input ?? secret.requires_user_input ?? false;
  const currentPromptLabel = pendingEdit?.changes.prompt_label ?? secret.prompt?.label ?? "";
  const currentPromptDescription = pendingEdit?.changes.prompt_description ?? secret.prompt?.description ?? "";
  const currentGeneratorTemplate = pendingEdit?.changes.generator_template ?? secret.generator_template ?? null;
  const generatorTemplateStr = currentGeneratorTemplate ? JSON.stringify(currentGeneratorTemplate, null, 2) : "";
  const hasPendingChanges = pendingEdit?.isDirty ?? false;

  const handleGeneratorTemplateChange = (value: string) => {
    if (!value.trim()) {
      onUpdatePendingChange({ generator_template: undefined });
      return;
    }
    try {
      const parsed = JSON.parse(value);
      onUpdatePendingChange({ generator_template: parsed });
    } catch {
      // Keep the string in state even if invalid JSON - user is still typing
      // We'll validate on save
    }
  };

  return (
    <div className="space-y-4 p-4">
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-2">
            <h3 className="font-mono text-sm font-medium text-white">{secret.secret_key}</h3>
            {isOverridden && (
              <span className="flex items-center gap-1 rounded-full border border-purple-400/30 bg-purple-500/10 px-2 py-0.5 text-[10px] text-purple-200">
                <CircleDot className="h-3 w-3" />
                overridden
              </span>
            )}
            {isExcluded && (
              <span className="rounded-full border border-white/20 bg-white/5 px-2 py-0.5 text-[10px] text-white/50 line-through">
                excluded
              </span>
            )}
          </div>
          <p className="mt-1 text-xs text-white/60">{secret.resource_name}</p>
        </div>
        <div className="flex gap-1">
          {onOpenInResourcePanel && (
            <Button variant="ghost" size="sm" onClick={onOpenInResourcePanel} className="text-xs gap-1">
              <ExternalLink className="h-3 w-3" />
              Open
            </Button>
          )}
          <Button variant="ghost" size="sm" onClick={onToggleExclude} className="text-xs">
            {isExcluded ? "Include" : "Exclude"}
          </Button>
        </div>
      </div>

      <div className="space-y-3 rounded-xl border border-white/10 bg-white/5 p-3">
        <div className="flex items-center justify-between text-xs">
          <span className="text-white/60">Type</span>
          <span className="text-white">{secret.secret_type}</span>
        </div>
        <div className="flex items-center justify-between text-xs">
          <span className="text-white/60">Classification</span>
          <span className={`rounded-full border px-2 py-0.5 ${
            secret.classification === "infrastructure"
              ? "border-sky-400/30 bg-sky-500/10 text-sky-200"
              : secret.classification === "service"
              ? "border-purple-400/30 bg-purple-500/10 text-purple-200"
              : "border-amber-400/30 bg-amber-500/10 text-amber-200"
          }`}>
            {secret.classification}
          </span>
        </div>
        <div className="flex items-center justify-between text-xs">
          <span className="text-white/60">Required</span>
          <span className={secret.required ? "text-amber-200" : "text-white/50"}>
            {secret.required ? "Yes" : "No"}
          </span>
        </div>
        {secret.description && (
          <div className="text-xs">
            <span className="text-white/60">Description</span>
            <p className="mt-1 text-white/80">{secret.description}</p>
          </div>
        )}
      </div>

      <div className="space-y-3">
        <div className="flex items-center gap-2">
          <h4 className="text-xs uppercase tracking-[0.2em] text-white/60">Strategy Configuration</h4>
          <HelpDialog title="Handling Strategies">
            <p>Each strategy determines how the secret is handled during deployment:</p>
            <ul className="mt-2 space-y-2">
              <li><strong className="text-emerald-200">Prompt:</strong> Ask user for value during installation</li>
              <li><strong className="text-cyan-200">Generate:</strong> Auto-create secure random value at build time</li>
              <li><strong className="text-amber-200">Strip:</strong> Exclude from the deployment bundle entirely</li>
              <li><strong className="text-purple-200">Delegate:</strong> Use cloud provider secret management</li>
            </ul>
          </HelpDialog>
        </div>

        {isBlocking && (
          <div className="flex items-center gap-2 rounded-xl border border-amber-400/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-100">
            <AlertTriangle className="h-4 w-4" />
            <span>No strategy defined - blocking deployment</span>
          </div>
        )}

        <div>
          <label className="text-xs text-white/60">Handling Strategy</label>
          <select
            value={currentStrategy}
            onChange={(e) => onUpdatePendingChange({ handling_strategy: e.target.value })}
            className="mt-1 w-full rounded-xl border border-white/10 bg-slate-800 px-3 py-2 text-sm text-white [&_option]:bg-slate-800 [&_option]:text-white"
          >
            <option value="">Select strategy...</option>
            {STRATEGY_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label className="text-xs text-white/60">Fallback Strategy (optional)</label>
          <select
            value={currentFallback}
            onChange={(e) => onUpdatePendingChange({ fallback_strategy: e.target.value || undefined })}
            className="mt-1 w-full rounded-xl border border-white/10 bg-slate-800 px-3 py-2 text-sm text-white [&_option]:bg-slate-800 [&_option]:text-white"
          >
            <option value="">None</option>
            {STRATEGY_OPTIONS.filter((opt) => opt.value !== currentStrategy).map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
          <p className="mt-1 text-[10px] text-white/40">
            Used if the primary strategy fails or is unavailable
          </p>
        </div>

        {currentStrategy === "prompt" && (
          <>
            <div>
              <label className="text-xs text-white/60">
                Prompt Label <span className="text-amber-300">*</span>
              </label>
              <input
                type="text"
                value={currentPromptLabel}
                onChange={(e) => onUpdatePendingChange({ prompt_label: e.target.value })}
                placeholder="e.g., Database Password"
                className="mt-1 w-full rounded-xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
              />
            </div>
            <div>
              <label className="text-xs text-white/60">
                Prompt Description <span className="text-amber-300">*</span>
              </label>
              <textarea
                value={currentPromptDescription}
                onChange={(e) => onUpdatePendingChange({ prompt_description: e.target.value })}
                placeholder="Explain what this secret is used for..."
                rows={2}
                className="mt-1 w-full rounded-xl border border-white/10 bg-white/5 px-3 py-2 text-sm text-white"
              />
            </div>
            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                id="requires-input"
                checked={currentRequiresInput}
                onChange={(e) => onUpdatePendingChange({ requires_user_input: e.target.checked })}
                className="h-4 w-4 rounded border-white/20 bg-slate-800 text-emerald-500"
              />
              <label htmlFor="requires-input" className="text-xs text-white/80">
                Requires user input (can't be skipped)
              </label>
            </div>
          </>
        )}

        {currentStrategy === "generate" && (
          <div className="space-y-3">
            <div className="rounded-xl border border-cyan-400/30 bg-cyan-400/5 px-3 py-2 text-xs text-cyan-100">
              This secret will be auto-generated at deployment time using a secure random generator.
            </div>
            <div>
              <label className="text-xs text-white/60">Generator Template (JSON, optional)</label>
              <textarea
                value={generatorTemplateStr}
                onChange={(e) => handleGeneratorTemplateChange(e.target.value)}
                placeholder={'{\n  "type": "alphanumeric",\n  "length": 32\n}'}
                rows={4}
                className="mt-1 w-full rounded-xl border border-white/10 bg-white/5 px-3 py-2 font-mono text-xs text-white"
              />
              <p className="mt-1 text-[10px] text-white/40">
                Configure generator behavior (type, length, charset, etc.)
              </p>
            </div>
          </div>
        )}

        {currentStrategy === "strip" && (
          <div className="rounded-xl border border-amber-400/30 bg-amber-400/5 px-3 py-2 text-xs text-amber-100">
            This secret will be excluded from the deployment bundle. Ensure the feature doesn't require it.
          </div>
        )}

        {currentStrategy === "delegate" && (
          <div className="rounded-xl border border-purple-400/30 bg-purple-400/5 px-3 py-2 text-xs text-purple-100">
            This secret will be managed by the cloud provider (e.g., AWS Secrets Manager, Vault).
          </div>
        )}

        <div className="flex gap-2 pt-2">
          <Button
            size="sm"
            onClick={onSave}
            disabled={!hasPendingChanges || isSaving || (currentStrategy === "prompt" && (!currentPromptLabel || !currentPromptDescription))}
            className="flex-1"
          >
            {isSaving ? "Saving..." : "Save Override"}
          </Button>
          {isOverridden && (
            <Button
              variant="outline"
              size="sm"
              onClick={onReset}
              disabled={isDeleting}
              className="text-red-300 hover:text-red-200"
            >
              {isDeleting ? "Reverting..." : "Reset to Default"}
            </Button>
          )}
        </div>

        {hasPendingChanges && (
          <p className="text-center text-[10px] text-amber-200/60">
            You have unsaved changes
          </p>
        )}
      </div>

      {secret.tier_strategies && Object.keys(secret.tier_strategies).length > 0 && (
        <div className="space-y-2">
          <h4 className="text-xs uppercase tracking-[0.2em] text-white/60">Other Tier Strategies</h4>
          <div className="space-y-1 rounded-xl border border-white/10 bg-white/5 p-3">
            {Object.entries(secret.tier_strategies).map(([tier, strategy]) => (
              <div key={tier} className="flex items-center justify-between text-xs">
                <span className="text-white/60">{tier}</span>
                <span className={`rounded-full border px-2 py-0.5 ${getStrategyColorClass(strategy)}`}>
                  {strategy}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
