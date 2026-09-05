import { Monitor } from "lucide-react";
import type {
  DiscoveredCertificate,
  WindowsSigningConfig,
} from "../../domain/signing";
import { Label } from "../ui/label";
import { Input } from "../ui/input";
import { Checkbox } from "../ui/checkbox";
import { Select } from "../ui/select";
import { SigningFormWrapper } from "./SigningFormWrapper";
import { DiscoveredCertSelector } from "./DiscoveredCertSelector";
import { selectors } from "../../consts/selectors";

interface WindowsSigningFormProps {
  config?: WindowsSigningConfig;
  onChange: (config: WindowsSigningConfig | undefined) => void;
  discovered?: DiscoveredCertificate[];
  onApplyDiscovered?: (cert: DiscoveredCertificate) => void;
}

const TIMESTAMP_SERVERS = [
  { value: "http://timestamp.digicert.com", label: "DigiCert (Recommended)" },
  { value: "http://timestamp.sectigo.com", label: "Sectigo" },
  {
    value: "http://timestamp.globalsign.com/tsa/r6advanced1",
    label: "GlobalSign",
  },
];

export function WindowsSigningForm({
  config,
  onChange,
  discovered,
  onApplyDiscovered,
}: WindowsSigningFormProps) {
  const handleChange = (updates: Partial<WindowsSigningConfig>) => {
    onChange({
      certificate_source: "file",
      ...config,
      ...updates,
    });
  };

  const handleEnable = (enabled: boolean) => {
    if (enabled) {
      onChange({
        certificate_source: "file",
        timestamp_server: "http://timestamp.digicert.com",
        sign_algorithm: "sha256",
      });
    } else {
      onChange(undefined);
    }
  };

  return (
    <SigningFormWrapper
      platform="Windows"
      platformId="windows"
      icon={Monitor}
      iconClassName="h-4 w-4 text-blue-400"
      isConfigured={!!config}
      onToggle={handleEnable}
      disabledMessage="Enable Windows signing to configure certificate settings."
      testId={selectors.signing.windowsForm}
    >
      {discovered && discovered.length > 0 && onApplyDiscovered && (
        <DiscoveredCertSelector
          label="Discovered certificates"
          discovered={discovered}
          onSelect={onApplyDiscovered}
          expiryWarningText="Some certificates expire within 30 days."
        />
      )}

      {/* Certificate Source */}
      <div>
        <Label htmlFor="win-cert-source" className="text-xs">
          Certificate Source
        </Label>
        <Select
          id="win-cert-source"
          value={config?.certificate_source ?? "file"}
          onChange={(e) => {
            handleChange({
              certificate_source: e.target
                .value as WindowsSigningConfig["certificate_source"],
            });
          }}
          className="mt-1 text-sm"
        >
          <option value="file">File (.pfx/.p12)</option>
          <option value="store">Windows Certificate Store</option>
          <option value="azure_keyvault">Azure Key Vault</option>
          <option value="aws_kms">AWS KMS</option>
        </Select>
      </div>

      {/* File-based certificate fields */}
      {config?.certificate_source === "file" && (
        <>
          <div>
            <Label htmlFor="win-cert-file" className="text-xs">
              Certificate File Path
            </Label>
            <Input
              id="win-cert-file"
              value={config.certificate_file || ""}
              onChange={(e) => {
                handleChange({ certificate_file: e.target.value });
              }}
              placeholder="/path/to/certificate.pfx"
              className="mt-1 text-sm"
            />
          </div>
          <div>
            <Label htmlFor="win-cert-password-env" className="text-xs">
              Password Environment Variable
            </Label>
            <Input
              id="win-cert-password-env"
              value={config.certificate_password_env || ""}
              onChange={(e) => {
                handleChange({ certificate_password_env: e.target.value });
              }}
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
      {config?.certificate_source === "store" && (
        <div>
          <Label htmlFor="win-cert-thumbprint" className="text-xs">
            Certificate Thumbprint
          </Label>
          <Input
            id="win-cert-thumbprint"
            value={config.certificate_thumbprint || ""}
            onChange={(e) => {
              handleChange({ certificate_thumbprint: e.target.value });
            }}
            placeholder="SHA-1 thumbprint"
            className="mt-1 text-sm font-mono"
          />
        </div>
      )}

      {/* Timestamp Server */}
      <div>
        <Label htmlFor="win-timestamp" className="text-xs">
          Timestamp Server
        </Label>
        <Select
          id="win-timestamp"
          value={config?.timestamp_server || "http://timestamp.digicert.com"}
          onChange={(e) => {
            handleChange({ timestamp_server: e.target.value });
          }}
          className="mt-1 text-sm"
        >
          {TIMESTAMP_SERVERS.map((server) => (
            <option key={server.value} value={server.value}>
              {server.label}
            </option>
          ))}
        </Select>
      </div>

      {/* Sign Algorithm */}
      <div>
        <Label htmlFor="win-algorithm" className="text-xs">
          Signing Algorithm
        </Label>
        <Select
          id="win-algorithm"
          value={config?.sign_algorithm || "sha256"}
          onChange={(e) => {
            handleChange({
              sign_algorithm: e.target
                .value as WindowsSigningConfig["sign_algorithm"],
            });
          }}
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
          checked={config?.dual_sign || false}
          onChange={(e) => {
            handleChange({ dual_sign: e.target.checked });
          }}
        />
        <Label htmlFor="win-dual-sign" className="text-xs">
          Dual Sign (SHA-1 + SHA-256 for Windows 7)
        </Label>
      </div>
    </SigningFormWrapper>
  );
}
