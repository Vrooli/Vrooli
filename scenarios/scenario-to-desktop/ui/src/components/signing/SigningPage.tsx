import {
  Shield,
  AlertCircle,
  CheckCircle,
  XCircle,
  RefreshCw,
  Save,
  Trash2,
  Info,
  Wand2,
} from "lucide-react";
import type {
  DiscoveredCertificate,
  SigningReadinessResponse,
  SigningValidationResult,
} from "../../domain/signing";
import { useSigningPage } from "../../hooks/useSigningPage";
import { SectionCard } from "../sections/shared";
import { Button } from "../ui/button";
import { Label } from "../ui/label";
import { Checkbox } from "../ui/checkbox";
import { Select } from "../ui/select";
import { WindowsSigningForm } from "./WindowsSigningForm";
import { MacOSSigningForm } from "./MacOSSigningForm";
import { LinuxSigningForm } from "./LinuxSigningForm";
import { selectors } from "../../consts/selectors";
import { PrerequisitesPanel } from "./PrerequisitesPanel";
import { cn } from "../../lib/utils";

interface SigningPageProps {
  initialScenario?: string;
  onScenarioChange?: (name: string) => void;
}

export function SigningPage({
  initialScenario,
  onScenarioChange,
}: SigningPageProps) {
  const {
    scenarios,
    selectedScenario,
    setSelectedScenario,
    localConfig,
    hasUnsavedChanges,
    configLoading,
    serverConfig,
    readinessData,
    prerequisitesData,
    prerequisitesLoading,
    refetchPrerequisites,
    discoverPlatform,
    setDiscoverPlatform,
    discovered,
    discoverPending,
    onDiscover,
    keygenMessage,
    keygenPending,
    validationResult,
    validatePending,
    savePending,
    saveError,
    deletePending,
    deleteError,
    validateError,
    handleConfigChange,
    handleSave,
    handleValidate,
    handleDelete,
    handleGenerateKey,
    applyCertificate,
    refetchConfig,
    refetchReadiness,
  } = useSigningPage({ initialScenario, onScenarioChange });

  return (
    <div className="space-y-6">
      {/* Header */}
      <SectionCard
        title="Code Signing Configuration"
        icon={Shield}
        contentClassName="space-y-4"
      >
        <p className="text-sm text-slate-300">
          Configure code signing for your desktop applications. Signed apps are
          trusted by operating systems and won&apos;t trigger security warnings
          during installation.
        </p>

        {/* Scenario Selector */}
        <div className="flex items-end gap-4">
          <div className="flex-1">
            <Label htmlFor="scenario-select">Scenario</Label>
            <Select
              id="scenario-select"
              data-testid={selectors.signing.scenarioSelect}
              value={selectedScenario}
              onChange={(e) => {
                setSelectedScenario(e.target.value);
              }}
              className="mt-1"
            >
              <option value="">Select a scenario...</option>
              {scenarios.map((scenario) => (
                <option key={scenario.name} value={scenario.name}>
                  {scenario.display_name || scenario.name}
                </option>
              ))}
            </Select>
          </div>

          {selectedScenario && (
            <div className="flex gap-2">
              <Button
                variant="outline"
                onClick={() => {
                  refetchConfig();
                  refetchReadiness();
                }}
                disabled={configLoading}
              >
                <RefreshCw
                  className={cn(
                    "h-4 w-4 mr-1",
                    configLoading && "animate-spin",
                  )}
                />
                Refresh
              </Button>
            </div>
          )}
        </div>
      </SectionCard>

      <SigningPrimer />

      {selectedScenario && (
        <>
          {/* Readiness Status */}
          <ReadinessCard readiness={readinessData} />

          {/* Main Configuration */}
          <SectionCard title="Configuration" contentClassName="space-y-6">
            {/* Enable signing toggle in header area */}
            <div className="flex items-center justify-end gap-2 -mt-2 mb-4">
              <Checkbox
                id="signing-enabled"
                data-testid={selectors.signing.enabled}
                checked={localConfig.enabled}
                onChange={(e) => {
                  handleConfigChange({ enabled: e.target.checked });
                }}
              />
              <Label htmlFor="signing-enabled" className="text-sm font-medium">
                Enable Signing
              </Label>
            </div>
            {localConfig.enabled ? (
              <>
                {/* Platform Tabs */}
                <div className="grid gap-6 lg:grid-cols-3">
                  {/* Windows */}
                  <WindowsSigningForm
                    config={localConfig.windows}
                    onChange={(windows) => {
                      handleConfigChange({ windows });
                    }}
                    discovered={discovered.filter(
                      (c) => c.platform === "windows",
                    )}
                    onApplyDiscovered={applyCertificate}
                  />

                  {/* macOS */}
                  <MacOSSigningForm
                    config={localConfig.macos}
                    onChange={(macos) => {
                      handleConfigChange({ macos });
                    }}
                    discovered={discovered.filter(
                      (c) => c.platform === "macos",
                    )}
                    onApplyDiscovered={applyCertificate}
                  />

                  {/* Linux */}
                  <LinuxSigningForm
                    config={localConfig.linux}
                    onChange={(linux) => {
                      handleConfigChange({ linux });
                    }}
                    discovered={discovered.filter(
                      (c) => c.platform === "linux",
                    )}
                    onApplyDiscovered={applyCertificate}
                    onGenerate={() => {
                      void handleGenerateKey();
                    }}
                    generating={keygenPending}
                    generationMessage={keygenMessage}
                  />
                </div>

                {/* Validation Results */}
                {validationResult && (
                  <ValidationResultsCard result={validationResult} />
                )}
              </>
            ) : (
              <p className="text-sm text-slate-400 text-center py-8">
                Enable signing to configure platform-specific settings.
              </p>
            )}

            {/* Actions */}
            <div className="flex items-center justify-between pt-4 border-t border-slate-800">
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  onClick={handleValidate}
                  disabled={!localConfig.enabled || validatePending}
                  data-testid={selectors.signing.validate}
                >
                  {validatePending ? (
                    <RefreshCw className="h-4 w-4 mr-1 animate-spin" />
                  ) : (
                    <CheckCircle className="h-4 w-4 mr-1" />
                  )}
                  Validate
                </Button>
                {serverConfig && (
                  <Button
                    variant="outline"
                    onClick={handleDelete}
                    disabled={deletePending}
                    className="text-red-400 hover:text-red-300 hover:border-red-800"
                  >
                    <Trash2 className="h-4 w-4 mr-1" />
                    Delete Config
                  </Button>
                )}
              </div>

              <div className="flex items-center gap-2">
                {hasUnsavedChanges && (
                  <span className="text-xs text-amber-400">
                    Unsaved changes
                  </span>
                )}
                <Button
                  onClick={handleSave}
                  disabled={!hasUnsavedChanges || savePending}
                >
                  {savePending ? (
                    <RefreshCw className="h-4 w-4 mr-1 animate-spin" />
                  ) : (
                    <Save className="h-4 w-4 mr-1" />
                  )}
                  Save Configuration
                </Button>
              </div>
            </div>

            {/* Error Display */}
            {(saveError || deleteError || validateError) && (
              <div className="p-3 rounded-lg bg-red-950/50 border border-red-800 text-red-200 text-sm">
                {(saveError || deleteError || validateError)?.message}
              </div>
            )}
          </SectionCard>

          <CertificateDiscovery
            platform={discoverPlatform}
            onPlatformChange={setDiscoverPlatform}
            onDiscover={onDiscover}
            loading={discoverPending}
            certificates={discovered}
            onApply={applyCertificate}
          />

          {/* Prerequisites Panel */}
          <PrerequisitesPanel
            tools={prerequisitesData}
            onRefresh={refetchPrerequisites}
            refreshing={prerequisitesLoading}
          />
        </>
      )}

      {!selectedScenario && (
        <SectionCard title="Code Signing" icon={Shield} showHeader={false}>
          <div className="py-8 text-center">
            <Shield className="h-12 w-12 mx-auto mb-4 text-slate-600" />
            <p className="text-slate-400">
              Select a scenario to configure code signing.
            </p>
          </div>
        </SectionCard>
      )}
    </div>
  );
}

function SigningPrimer() {
  return (
    <SectionCard
      title="Signing Quickstart"
      icon={Info}
      contentClassName="grid gap-6 md:grid-cols-2"
    >
      <div className="space-y-3">
        <p className="text-sm text-slate-300">
          You can ship unsigned installers for local testing. Enable signing
          only when you&apos;re ready for users or app stores.
        </p>
        <ul className="list-disc space-y-2 pl-5 text-sm text-slate-200">
          <li>
            Select a scenario, toggle <strong>Enable Signing</strong>, and fill
            only the platform you care about.
          </li>
          <li>
            Use <strong>Validate</strong> to see missing items, then{" "}
            <strong>Save</strong>. The Signing Tools panel below shows which
            CLIs are installed.
          </li>
          <li>
            Return to <em>Generate Desktop App</em> and enable signing for that
            build to package with these settings.
          </li>
          <li>
            Need more detail? Read the full signing guide:{" "}
            <a
              className="text-blue-300 underline"
              href="/?view=docs&doc=SIGNING.md"
              onClick={(e) => {
                if (typeof window === "undefined") return;
                e.preventDefault();
                const url = new URL(window.location.href);
                url.searchParams.set("view", "docs");
                url.searchParams.set("doc", "SIGNING.md");
                window.history.pushState(null, "", url.toString());
                window.dispatchEvent(new PopStateEvent("popstate"));
              }}
            >
              scenarios/scenario-to-desktop/docs/SIGNING.md
            </a>
          </li>
        </ul>
      </div>
      <div className="space-y-3">
        <p className="text-sm font-semibold text-slate-100">
          Smallest thing you need per platform
        </p>
        <div className="rounded-lg border border-slate-800 bg-slate-950/50 p-3 space-y-2 text-xs text-slate-300">
          <p>
            <strong>Windows:</strong> A .pfx/.p12 file path and an env var name
            for the password (e.g. <code>WIN_CERT_PASSWORD</code>). If your cert
            lives in the Windows store, paste its thumbprint instead.
          </p>
          <p>
            <strong>macOS:</strong> Developer ID identity (see{" "}
            <code>security find-identity -v -p codesigning</code>) and Team ID.
            Notarization is optional until you publish.
          </p>
          <p>
            <strong>Linux:</strong> GPG key ID or fingerprint from{" "}
            <code>gpg --list-secret-keys</code>; optional keyring path if not
            default.
          </p>
          <p className="text-slate-400">
            If you don&apos;t have these yet, leave signing off—the build will
            still work but installers will prompt users.
          </p>
        </div>
      </div>
    </SectionCard>
  );
}

function ReadinessCard({
  readiness,
}: {
  readiness?: SigningReadinessResponse;
}) {
  if (!readiness) return null;

  const platformOrder = ["windows", "macos", "linux"] as const;

  return (
    <SectionCard
      title="Signing Readiness"
      icon={readiness.ready ? CheckCircle : AlertCircle}
      className={
        readiness.ready ? "border-green-800/50" : "border-amber-800/50"
      }
    >
      <div className="grid grid-cols-3 gap-4">
        {platformOrder.map((platform) => {
          const status = readiness.platforms?.[platform];
          return (
            <div
              key={platform}
              className={cn(
                "p-3 rounded-lg border",
                status?.ready
                  ? "border-green-800/50 bg-green-950/30"
                  : "border-slate-800 bg-slate-950/30",
              )}
            >
              <div className="flex items-center gap-2 mb-1">
                {status?.ready ? (
                  <CheckCircle className="h-4 w-4 text-green-400" />
                ) : (
                  <XCircle className="h-4 w-4 text-slate-500" />
                )}
                <span className="font-medium capitalize">
                  {platform === "macos" ? "macOS" : platform}
                </span>
              </div>
              <p className="text-xs text-slate-400">
                {status?.ready ? "Ready" : status?.reason || "Not configured"}
              </p>
            </div>
          );
        })}
      </div>

      {readiness.issues && readiness.issues.length > 0 && (
        <div className="mt-4 p-3 rounded-lg bg-amber-950/30 border border-amber-800/50">
          <p className="text-sm font-medium text-amber-300 mb-2">Issues:</p>
          <ul className="text-sm text-amber-200 space-y-1">
            {readiness.issues.map((issue, i) => (
              <li key={i} className="flex items-start gap-2">
                <AlertCircle className="h-4 w-4 mt-0.5 flex-shrink-0" />
                {issue}
              </li>
            ))}
          </ul>
        </div>
      )}
    </SectionCard>
  );
}

function CertificateDiscovery({
  platform,
  onPlatformChange,
  onDiscover,
  loading,
  certificates,
  onApply,
}: {
  platform: "windows" | "macos" | "linux";
  onPlatformChange: (value: "windows" | "macos" | "linux") => void;
  onDiscover: () => void;
  loading: boolean;
  certificates: DiscoveredCertificate[];
  onApply: (cert: DiscoveredCertificate) => void;
}) {
  return (
    <SectionCard
      title="Discover Certificates"
      icon={Wand2}
      contentClassName="space-y-3"
    >
      {/* Platform selector in content area */}
      <div className="flex items-center justify-end gap-2 -mt-2 mb-4">
        <select
          aria-label="Certificate discovery platform"
          className="rounded border border-slate-700 bg-slate-950 px-2 py-1 text-sm"
          value={platform}
          onChange={(e) => {
            onPlatformChange(e.target.value as "windows" | "macos" | "linux");
          }}
        >
          <option value="windows">Windows</option>
          <option value="macos">macOS</option>
          <option value="linux">Linux</option>
        </select>
        <Button
          variant="outline"
          size="sm"
          onClick={onDiscover}
          disabled={loading}
        >
          {loading ? (
            <RefreshCw className="h-4 w-4 animate-spin" />
          ) : (
            <RefreshCw className="h-4 w-4" />
          )}
          Scan
        </Button>
      </div>
      <p className="text-sm text-slate-400">
        Finds certificates or identities already on this machine so you can
        apply them without copy/paste.
      </p>
      {certificates.some(
        (c) => (c.days_to_expiry ?? Infinity) <= 30 && !c.is_expired,
      ) && (
        <div className="rounded border border-amber-800 bg-amber-950/30 p-2 text-xs text-amber-200 flex items-center gap-2">
          <AlertCircle className="h-4 w-4" />
          <span>
            One or more certificates expire within 30 days. Apply a newer one
            before publishing.
          </span>
        </div>
      )}
      {certificates.length === 0 ? (
        <div className="rounded border border-slate-800 bg-slate-950/50 p-3 text-sm text-slate-400">
          {loading ? "Scanning…" : "No certificates found for this platform."}
        </div>
      ) : (
        <div className="grid gap-3 md:grid-cols-2">
          {certificates.map((cert) => (
            <div
              key={cert.id}
              className="rounded border border-slate-800 bg-slate-950/50 p-3 space-y-2"
            >
              <div
                className="text-sm text-slate-100 font-semibold truncate"
                title={cert.name || cert.subject}
              >
                {cert.name || cert.subject || cert.id}
              </div>
              <div className="text-xs text-slate-400 space-y-1">
                {cert.subject && (
                  <p className="truncate" title={cert.subject}>
                    Subject: {cert.subject}
                  </p>
                )}
                {cert.issuer && (
                  <p className="truncate" title={cert.issuer}>
                    Issuer: {cert.issuer}
                  </p>
                )}
                {cert.expires_at && (
                  <p
                    className={cn(
                      cert.is_expired || (cert.days_to_expiry ?? Infinity) <= 7
                        ? "text-red-300"
                        : (cert.days_to_expiry ?? Infinity) <= 30
                          ? "text-amber-300"
                          : "text-slate-400",
                    )}
                  >
                    Expires: {cert.expires_at} ({cert.days_to_expiry ?? "?"}{" "}
                    days)
                  </p>
                )}
                {cert.usage_hint && <p>{cert.usage_hint}</p>}
                {!cert.is_code_sign && (
                  <p className="text-amber-300">Not marked for code signing</p>
                )}
              </div>
              <Button
                size="sm"
                variant="outline"
                onClick={() => {
                  onApply(cert);
                }}
                className="w-full"
              >
                Apply to {platform}
              </Button>
            </div>
          ))}
        </div>
      )}
    </SectionCard>
  );
}

function ValidationResultsCard({
  result,
}: {
  result: SigningValidationResult;
}) {
  return (
    <SectionCard
      title={`Validation ${result.valid ? "Passed" : "Failed"}`}
      icon={result.valid ? CheckCircle : XCircle}
      className={
        result.valid
          ? "border-green-800/50 bg-green-950/20"
          : "border-red-800/50 bg-red-950/20"
      }
      contentClassName="space-y-3"
    >
      {result.errors.length > 0 && (
        <div>
          <p className="text-sm font-medium text-red-300 mb-2">Errors:</p>
          <ul className="space-y-2">
            {result.errors.map((error, i) => (
              <li
                key={i}
                className="text-sm p-2 rounded bg-red-950/50 border border-red-800/50"
              >
                <div className="flex items-start gap-2">
                  <XCircle className="h-4 w-4 mt-0.5 text-red-400 flex-shrink-0" />
                  <div>
                    <p className="text-red-200">{error.message}</p>
                    {error.remediation && (
                      <p className="text-xs text-red-300/70 mt-1">
                        {error.remediation}
                      </p>
                    )}
                  </div>
                </div>
              </li>
            ))}
          </ul>
        </div>
      )}

      {result.warnings.length > 0 && (
        <div>
          <p className="text-sm font-medium text-amber-300 mb-2">Warnings:</p>
          <ul className="space-y-2">
            {result.warnings.map((warning, i) => (
              <li
                key={i}
                className="text-sm p-2 rounded bg-amber-950/50 border border-amber-800/50"
              >
                <div className="flex items-start gap-2">
                  <AlertCircle className="h-4 w-4 mt-0.5 text-amber-400 flex-shrink-0" />
                  <p className="text-amber-200">{warning.message}</p>
                </div>
              </li>
            ))}
          </ul>
        </div>
      )}

      {result.valid &&
        result.errors.length === 0 &&
        result.warnings.length === 0 && (
          <p className="text-sm text-green-300">
            All checks passed successfully.
          </p>
        )}
    </SectionCard>
  );
}
