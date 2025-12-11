import { Laptop, AlertTriangle } from "lucide-react";
import type { DiscoveredCertificate, LinuxSigningConfig } from "../../lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";
import { Label } from "../ui/label";
import { Input } from "../ui/input";
import { Checkbox } from "../ui/checkbox";

interface LinuxSigningFormProps {
  config?: LinuxSigningConfig;
  onChange: (config: LinuxSigningConfig | undefined) => void;
  discovered?: DiscoveredCertificate[];
  onApplyDiscovered?: (cert: DiscoveredCertificate) => void;
}

export function LinuxSigningForm({ config, onChange, discovered, onApplyDiscovered }: LinuxSigningFormProps) {
  const isConfigured = !!config;

  const handleChange = (updates: Partial<LinuxSigningConfig>) => {
    onChange({
      ...config,
      ...updates
    });
  };

  const handleEnable = (enabled: boolean) => {
    if (enabled) {
      onChange({
        gpg_key_id: ""
      });
    } else {
      onChange(undefined);
    }
  };

  return (
    <Card className="border-slate-700/50 bg-slate-950/40">
      <CardHeader className="pb-3">
        <CardTitle className="text-base flex items-center justify-between">
          <span className="flex items-center gap-2">
            <Laptop className="h-4 w-4 text-orange-400" />
            Linux
          </span>
          <div className="flex items-center gap-2">
            <Checkbox
              id="linux-enabled"
              checked={isConfigured}
              onChange={(e) => handleEnable(e.target.checked)}
            />
            <Label htmlFor="linux-enabled" className="text-xs text-slate-400">
              Configure
            </Label>
          </div>
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {discovered && discovered.length > 0 && onApplyDiscovered && (
          <div className="rounded border border-slate-800 bg-slate-950/50 p-2 text-xs text-slate-200 space-y-2">
            <div className="flex items-center gap-2">
              <span className="font-medium">Discovered keys:</span>
              <select
                className="flex-1 rounded border border-slate-700 bg-slate-900 px-2 py-1"
                onChange={(e) => {
                  const selected = discovered.find((c) => c.id === e.target.value);
                  if (selected) {
                    onApplyDiscovered(selected);
                  }
                }}
                defaultValue=""
              >
                <option value="">Select to apply</option>
                {discovered.map((cert) => (
                  <option key={cert.id} value={cert.id}>
                    {cert.name || cert.id}
                  </option>
                ))}
              </select>
            </div>
            {discovered.some((c) => c.days_to_expiry <= 30 && !c.is_expired) && (
              <div className="flex items-center gap-1 text-amber-300">
                <AlertTriangle className="h-3 w-3" />
                <span>Some keys expire within 30 days.</span>
              </div>
            )}
          </div>
        )}
        {isConfigured ? (
          <>
            {/* GPG Key ID */}
            <div>
              <Label htmlFor="linux-gpg-key" className="text-xs">GPG Key ID</Label>
              <Input
                id="linux-gpg-key"
                value={config.gpg_key_id || ""}
                onChange={(e) => handleChange({ gpg_key_id: e.target.value })}
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
                value={config.gpg_passphrase_env || ""}
                onChange={(e) => handleChange({ gpg_passphrase_env: e.target.value })}
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
                value={config.gpg_homedir || ""}
                onChange={(e) => handleChange({ gpg_homedir: e.target.value })}
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
                <strong className="text-slate-300">Note:</strong> Linux signing uses GPG to sign
                .deb and .rpm packages. AppImage signing is also supported. Make sure your GPG key
                is available in the keyring on the build machine.
              </p>
            </div>
          </>
        ) : (
          <p className="text-xs text-slate-500 text-center py-4">
            Enable Linux signing to configure GPG key settings.
          </p>
        )}
      </CardContent>
    </Card>
  );
}
