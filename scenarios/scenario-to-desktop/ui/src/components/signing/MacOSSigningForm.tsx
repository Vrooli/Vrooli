import { Apple, AlertTriangle } from "lucide-react";
import type { DiscoveredCertificate, MacOSSigningConfig } from "../../lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";
import { Label } from "../ui/label";
import { Input } from "../ui/input";
import { Checkbox } from "../ui/checkbox";

interface MacOSSigningFormProps {
  config?: MacOSSigningConfig;
  onChange: (config: MacOSSigningConfig | undefined) => void;
  discovered?: DiscoveredCertificate[];
  onApplyDiscovered?: (cert: DiscoveredCertificate) => void;
}

export function MacOSSigningForm({ config, onChange, discovered, onApplyDiscovered }: MacOSSigningFormProps) {
  const isConfigured = !!config;

  const handleChange = (updates: Partial<MacOSSigningConfig>) => {
    onChange({
      identity: "",
      team_id: "",
      hardened_runtime: true,
      notarize: false,
      ...config,
      ...updates
    });
  };

  const handleEnable = (enabled: boolean) => {
    if (enabled) {
      onChange({
        identity: "",
        team_id: "",
        hardened_runtime: true,
        notarize: false
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
            <Apple className="h-4 w-4 text-slate-300" />
            macOS
          </span>
          <div className="flex items-center gap-2">
            <Checkbox
              id="macos-enabled"
              checked={isConfigured}
              onChange={(e) => handleEnable(e.target.checked)}
            />
            <Label htmlFor="macos-enabled" className="text-xs text-slate-400">
              Configure
            </Label>
          </div>
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {discovered && discovered.length > 0 && onApplyDiscovered && (
          <div className="rounded border border-slate-800 bg-slate-950/50 p-2 text-xs text-slate-200 space-y-2">
            <div className="flex items-center gap-2">
              <span className="font-medium">Discovered identities:</span>
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
                    {cert.name || cert.subject || cert.id}
                  </option>
                ))}
              </select>
            </div>
            {discovered.some((c) => c.days_to_expiry <= 30 && !c.is_expired) && (
              <div className="flex items-center gap-1 text-amber-300">
                <AlertTriangle className="h-3 w-3" />
                <span>Some identities expire within 30 days.</span>
              </div>
            )}
          </div>
        )}
        {isConfigured ? (
          <>
            {/* Signing Identity */}
            <div>
              <Label htmlFor="macos-identity" className="text-xs">Signing Identity</Label>
              <Input
                id="macos-identity"
                value={config.identity}
                onChange={(e) => handleChange({ identity: e.target.value })}
                placeholder="Developer ID Application: Company Name (TEAMID)"
                className="mt-1 text-sm"
              />
              <p className="text-xs text-slate-500 mt-1">
                Full identity string from your Developer ID certificate
              </p>
            </div>

            {/* Team ID */}
            <div>
              <Label htmlFor="macos-team-id" className="text-xs">Team ID</Label>
              <Input
                id="macos-team-id"
                value={config.team_id}
                onChange={(e) => handleChange({ team_id: e.target.value })}
                placeholder="TEAMID1234"
                className="mt-1 text-sm font-mono"
              />
              <p className="text-xs text-slate-500 mt-1">
                10-character Apple Developer Team ID
              </p>
            </div>

            {/* Hardened Runtime */}
            <div className="flex items-center gap-2">
              <Checkbox
                id="macos-hardened"
                checked={config.hardened_runtime}
                onChange={(e) => handleChange({ hardened_runtime: e.target.checked })}
              />
              <Label htmlFor="macos-hardened" className="text-xs">
                Enable Hardened Runtime (required for notarization)
              </Label>
            </div>

            {/* Entitlements */}
            <div>
              <Label htmlFor="macos-entitlements" className="text-xs">Entitlements File (Optional)</Label>
              <Input
                id="macos-entitlements"
                value={config.entitlements_file || ""}
                onChange={(e) => handleChange({ entitlements_file: e.target.value })}
                placeholder="/path/to/entitlements.plist"
                className="mt-1 text-sm"
              />
            </div>

            {/* Notarization Section */}
            <div className="pt-3 border-t border-slate-800">
              <div className="flex items-center gap-2 mb-3">
                <Checkbox
                  id="macos-notarize"
                  checked={config.notarize}
                  onChange={(e) => handleChange({ notarize: e.target.checked })}
                />
                <Label htmlFor="macos-notarize" className="text-sm font-medium">
                  Enable Notarization
                </Label>
              </div>

              {config.notarize && (
                <div className="space-y-4 pl-6">
                  <p className="text-xs text-slate-400 mb-3">
                    Choose an authentication method for notarization:
                  </p>

                  {/* API Key Authentication (Recommended) */}
                  <div className="p-3 rounded-lg border border-slate-700 bg-slate-900/50">
                    <p className="text-xs font-medium text-slate-300 mb-2">
                      App Store Connect API Key (Recommended)
                    </p>
                    <div className="space-y-3">
                      <div>
                        <Label htmlFor="macos-api-key-id" className="text-xs">API Key ID</Label>
                        <Input
                          id="macos-api-key-id"
                          value={config.apple_api_key_id || ""}
                          onChange={(e) => handleChange({ apple_api_key_id: e.target.value })}
                          placeholder="ABC123DEF4"
                          className="mt-1 text-sm font-mono"
                        />
                      </div>
                      <div>
                        <Label htmlFor="macos-api-issuer" className="text-xs">API Issuer ID</Label>
                        <Input
                          id="macos-api-issuer"
                          value={config.apple_api_issuer_id || ""}
                          onChange={(e) => handleChange({ apple_api_issuer_id: e.target.value })}
                          placeholder="12345678-1234-1234-1234-123456789012"
                          className="mt-1 text-sm font-mono"
                        />
                      </div>
                      <div>
                        <Label htmlFor="macos-api-key-file" className="text-xs">API Key File Path</Label>
                        <Input
                          id="macos-api-key-file"
                          value={config.apple_api_key_file || ""}
                          onChange={(e) => handleChange({ apple_api_key_file: e.target.value })}
                          placeholder="/path/to/AuthKey_ABC123DEF4.p8"
                          className="mt-1 text-sm"
                        />
                      </div>
                    </div>
                  </div>

                  {/* App-Specific Password Authentication */}
                  <div className="p-3 rounded-lg border border-slate-800 bg-slate-950/30">
                    <p className="text-xs font-medium text-slate-400 mb-2">
                      Or: Apple ID with App-Specific Password
                    </p>
                    <div className="space-y-3">
                      <div>
                        <Label htmlFor="macos-apple-id-env" className="text-xs">Apple ID Environment Variable</Label>
                        <Input
                          id="macos-apple-id-env"
                          value={config.apple_id_env || ""}
                          onChange={(e) => handleChange({ apple_id_env: e.target.value })}
                          placeholder="APPLE_ID"
                          className="mt-1 text-sm"
                        />
                      </div>
                      <div>
                        <Label htmlFor="macos-apple-password-env" className="text-xs">App Password Environment Variable</Label>
                        <Input
                          id="macos-apple-password-env"
                          value={config.apple_id_password_env || ""}
                          onChange={(e) => handleChange({ apple_id_password_env: e.target.value })}
                          placeholder="APPLE_APP_SPECIFIC_PASSWORD"
                          className="mt-1 text-sm"
                        />
                      </div>
                    </div>
                  </div>
                </div>
              )}
            </div>

            {/* Gatekeeper Assess */}
            <div className="flex items-center gap-2">
              <Checkbox
                id="macos-gatekeeper"
                checked={config.gatekeeper_assess ?? true}
                onChange={(e) => handleChange({ gatekeeper_assess: e.target.checked })}
              />
              <Label htmlFor="macos-gatekeeper" className="text-xs">
                Run Gatekeeper assessment after signing
              </Label>
            </div>
          </>
        ) : (
          <p className="text-xs text-slate-500 text-center py-4">
            Enable macOS signing to configure Developer ID and notarization.
          </p>
        )}
      </CardContent>
    </Card>
  );
}
