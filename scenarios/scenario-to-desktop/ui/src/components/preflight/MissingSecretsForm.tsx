/**
 * Form for entering missing secrets during preflight validation.
 */

import {
  SecretClass,
  type PreflightSecret,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/preflight_results_pb";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { Label } from "../ui/label";

interface MissingSecretsFormProps {
  missingSecrets: PreflightSecret[];
  secretInputs: Record<string, string>;
  preflightPending: boolean;
  onSecretChange: (id: string, value: string) => void;
  onApplySecrets: (secrets: Record<string, string>) => void;
}

export function MissingSecretsForm({
  missingSecrets,
  secretInputs,
  preflightPending,
  onSecretChange,
  onApplySecrets,
}: MissingSecretsFormProps) {
  if (missingSecrets.length === 0) {
    return null;
  }

  return (
    <div className="rounded-md border border-amber-800/60 bg-amber-950/20 p-3 text-xs text-amber-100 space-y-3">
      <p className="font-semibold text-amber-100">Missing required secrets</p>
      <p className="text-amber-100/80">
        Enter temporary values to validate readiness. Values are used only for
        this preflight run.
      </p>
      <div className="space-y-2">
        {missingSecrets.map((secret) => (
          <div key={secret.id} className="space-y-1">
            <Label htmlFor={`preflight-${secret.id}`}>
              {secret.prompt.label || secret.id}
            </Label>
            <Input
              id={`preflight-${secret.id}`}
              type="password"
              value={secretInputs[secret.id] || ""}
              onChange={(e) => {
                onSecretChange(secret.id, e.target.value);
              }}
              placeholder={secret.prompt.hint || "Enter value"}
            />
            {secret.secretClass !== SecretClass.UNSPECIFIED && (
              <p className="text-[11px] text-amber-200/80">
                {SecretClass[secret.secretClass]
                  .replace("SECRET_CLASS_", "")
                  .replace(/_/g, " ")
                  .toLowerCase()}
              </p>
            )}
          </div>
        ))}
      </div>
      <Button
        type="button"
        size="sm"
        variant="outline"
        onClick={() => {
          onApplySecrets(secretInputs);
        }}
        disabled={preflightPending}
      >
        Apply secrets and re-run
      </Button>
    </div>
  );
}
