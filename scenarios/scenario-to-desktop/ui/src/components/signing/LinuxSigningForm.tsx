import { Laptop } from "lucide-react";
import type {
  DiscoveredCertificate,
  LinuxSigningConfig,
} from "../../domain/signing";
import { Label } from "../ui/label";
import { Input } from "../ui/input";
import { Button } from "../ui/button";
import { SigningFormWrapper } from "./SigningFormWrapper";
import { DiscoveredCertSelector } from "./DiscoveredCertSelector";
import { selectors } from "../../consts/selectors";

interface LinuxSigningFormProps {
  config?: LinuxSigningConfig;
  onChange: (config: LinuxSigningConfig | undefined) => void;
  discovered?: DiscoveredCertificate[];
  onApplyDiscovered?: (cert: DiscoveredCertificate) => void;
  onGenerate?: () => void;
  generating?: boolean;
  generationMessage?: string;
}

export function LinuxSigningForm({
  config,
  onChange,
  discovered,
  onApplyDiscovered,
  onGenerate,
  generating,
  generationMessage,
}: LinuxSigningFormProps) {
  const handleChange = (updates: Partial<LinuxSigningConfig>) => {
    onChange({
      ...config,
      ...updates,
    });
  };

  const handleEnable = (enabled: boolean) => {
    if (enabled) {
      onChange({ gpg_key_id: "" });
    } else {
      onChange(undefined);
    }
  };

  const headerActions = onGenerate ? (
    <Button
      size="sm"
      variant="outline"
      onClick={onGenerate}
      disabled={generating}
    >
      {generating ? "Generating\u2026" : "Generate GPG key"}
    </Button>
  ) : null;

  return (
    <SigningFormWrapper
      platform="Linux"
      platformId="linux"
      icon={Laptop}
      iconClassName="h-4 w-4 text-orange-400"
      isConfigured={!!config}
      onToggle={handleEnable}
      headerActions={headerActions}
      disabledMessage="Enable Linux signing to configure GPG key settings."
      testId={selectors.signing.linuxForm}
    >
      {discovered && discovered.length > 0 && onApplyDiscovered && (
        <DiscoveredCertSelector
          label="Discovered keys"
          discovered={discovered}
          onSelect={onApplyDiscovered}
          expiryWarningText="Some keys expire within 30 days."
        />
      )}

      {/* GPG Key ID */}
      <div>
        <Label htmlFor="linux-gpg-key" className="text-xs">
          GPG Key ID
        </Label>
        <Input
          id="linux-gpg-key"
          value={config?.gpg_key_id || ""}
          onChange={(e) => {
            handleChange({ gpg_key_id: e.target.value });
          }}
          placeholder="ABC123DEF456789012345678"
          className="mt-1 text-sm font-mono"
        />
        <p className="text-xs text-slate-500 mt-1">
          Your GPG key fingerprint or key ID for signing packages
        </p>
      </div>

      {/* GPG Passphrase Environment Variable */}
      <div>
        <Label htmlFor="linux-gpg-passphrase-env" className="text-xs">
          Passphrase Environment Variable (Optional)
        </Label>
        <Input
          id="linux-gpg-passphrase-env"
          value={config?.gpg_passphrase_env || ""}
          onChange={(e) => {
            handleChange({ gpg_passphrase_env: e.target.value });
          }}
          placeholder="GPG_PASSPHRASE"
          className="mt-1 text-sm"
        />
        <p className="text-xs text-slate-500 mt-1">
          Name of environment variable containing the GPG passphrase
        </p>
      </div>

      {/* GPG Home Directory */}
      <div>
        <Label htmlFor="linux-gpg-homedir" className="text-xs">
          GPG Home Directory (Optional)
        </Label>
        <Input
          id="linux-gpg-homedir"
          value={config?.gpg_homedir || ""}
          onChange={(e) => {
            handleChange({ gpg_homedir: e.target.value });
          }}
          placeholder="~/.gnupg"
          className="mt-1 text-sm"
        />
        <p className="text-xs text-slate-500 mt-1">
          Custom GPG home directory. Leave empty to use default.
        </p>
      </div>

      {/* Info Box */}
      <div className="p-3 rounded-lg bg-slate-800/50 border border-slate-700">
        <p className="text-xs text-slate-400">
          <strong className="text-slate-300">Note:</strong> Linux signing uses
          GPG to sign .deb and .rpm packages. AppImage signing is also
          supported. Make sure your GPG key is available in the keyring on the
          build machine.
        </p>
      </div>
      {generationMessage && (
        <div className="p-3 rounded-lg bg-slate-800/40 border border-slate-700 text-xs text-slate-200">
          {generationMessage}
        </div>
      )}
    </SigningFormWrapper>
  );
}
