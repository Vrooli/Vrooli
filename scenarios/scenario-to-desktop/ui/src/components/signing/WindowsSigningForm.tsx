import { Monitor } from "lucide-react";
import type { WindowsSigningConfig } from "../../lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";
import { Label } from "../ui/label";
import { Input } from "../ui/input";
import { Checkbox } from "../ui/checkbox";
import { Select } from "../ui/select";

interface WindowsSigningFormProps {
  config?: WindowsSigningConfig;
  onChange: (config: WindowsSigningConfig | undefined) => void;
}

const TIMESTAMP_SERVERS = [
  { value: "http://timestamp.digicert.com", label: "DigiCert (Recommended)" },
  { value: "http://timestamp.sectigo.com", label: "Sectigo" },
  { value: "http://timestamp.globalsign.com/tsa/r6advanced1", label: "GlobalSign" },
];

export function WindowsSigningForm({ config, onChange }: WindowsSigningFormProps) {
  const isConfigured = !!config;

  const handleChange = (updates: Partial<WindowsSigningConfig>) => {
    onChange({
      certificate_source: "file",
      ...config,
      ...updates
    });
  };

  const handleEnable = (enabled: boolean) => {
    if (enabled) {
      onChange({
        certificate_source: "file",
        timestamp_server: "http://timestamp.digicert.com",
        sign_algorithm: "sha256"
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
            <Monitor className="h-4 w-4 text-blue-400" />
            Windows
          </span>
          <div className="flex items-center gap-2">
            <Checkbox
              id="windows-enabled"
              checked={isConfigured}
              onChange={(e) => handleEnable(e.target.checked)}
            />
            <Label htmlFor="windows-enabled" className="text-xs text-slate-400">
              Configure
            </Label>
          </div>
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {isConfigured ? (
          <>
            {/* Certificate Source */}
            <div>
              <Label htmlFor="win-cert-source" className="text-xs">Certificate Source</Label>
              <Select
                id="win-cert-source"
                value={config.certificate_source}
                onChange={(e) => handleChange({
                  certificate_source: e.target.value as WindowsSigningConfig["certificate_source"]
                })}
                className="mt-1 text-sm"
              >
                <option value="file">File (.pfx/.p12)</option>
                <option value="store">Windows Certificate Store</option>
                <option value="azure_keyvault">Azure Key Vault</option>
                <option value="aws_kms">AWS KMS</option>
              </Select>
            </div>

            {/* File-based certificate fields */}
            {config.certificate_source === "file" && (
              <>
                <div>
                  <Label htmlFor="win-cert-file" className="text-xs">Certificate File Path</Label>
                  <Input
                    id="win-cert-file"
                    value={config.certificate_file || ""}
                    onChange={(e) => handleChange({ certificate_file: e.target.value })}
                    placeholder="/path/to/certificate.pfx"
                    className="mt-1 text-sm"
                  />
                </div>
                <div>
                  <Label htmlFor="win-cert-password-env" className="text-xs">Password Environment Variable</Label>
                  <Input
                    id="win-cert-password-env"
                    value={config.certificate_password_env || ""}
                    onChange={(e) => handleChange({ certificate_password_env: e.target.value })}
                    placeholder="WIN_CERT_PASSWORD"
                    className="mt-1 text-sm"
                  />
                  <p className="text-xs text-slate-500 mt-1">
                    Name of environment variable containing the certificate password
                  </p>
                </div>
              </>
            )}

            {/* Store-based certificate fields */}
            {config.certificate_source === "store" && (
              <div>
                <Label htmlFor="win-cert-thumbprint" className="text-xs">Certificate Thumbprint</Label>
                <Input
                  id="win-cert-thumbprint"
                  value={config.certificate_thumbprint || ""}
                  onChange={(e) => handleChange({ certificate_thumbprint: e.target.value })}
                  placeholder="SHA-1 thumbprint"
                  className="mt-1 text-sm font-mono"
                />
              </div>
            )}

            {/* Timestamp Server */}
            <div>
              <Label htmlFor="win-timestamp" className="text-xs">Timestamp Server</Label>
              <Select
                id="win-timestamp"
                value={config.timestamp_server || "http://timestamp.digicert.com"}
                onChange={(e) => handleChange({ timestamp_server: e.target.value })}
                className="mt-1 text-sm"
              >
                {TIMESTAMP_SERVERS.map(server => (
                  <option key={server.value} value={server.value}>
                    {server.label}
                  </option>
                ))}
              </Select>
            </div>

            {/* Sign Algorithm */}
            <div>
              <Label htmlFor="win-algorithm" className="text-xs">Signing Algorithm</Label>
              <Select
                id="win-algorithm"
                value={config.sign_algorithm || "sha256"}
                onChange={(e) => handleChange({
                  sign_algorithm: e.target.value as WindowsSigningConfig["sign_algorithm"]
                })}
                className="mt-1 text-sm"
              >
                <option value="sha256">SHA-256 (Recommended)</option>
                <option value="sha384">SHA-384</option>
                <option value="sha512">SHA-512</option>
              </Select>
            </div>

            {/* Dual Sign */}
            <div className="flex items-center gap-2">
              <Checkbox
                id="win-dual-sign"
                checked={config.dual_sign || false}
                onChange={(e) => handleChange({ dual_sign: e.target.checked })}
              />
              <Label htmlFor="win-dual-sign" className="text-xs">
                Dual Sign (SHA-1 + SHA-256 for Windows 7)
              </Label>
            </div>
          </>
        ) : (
          <p className="text-xs text-slate-500 text-center py-4">
            Enable Windows signing to configure certificate settings.
          </p>
        )}
      </CardContent>
    </Card>
  );
}
