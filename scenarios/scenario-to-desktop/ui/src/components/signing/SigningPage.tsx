import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Shield, AlertCircle, CheckCircle, XCircle, RefreshCw, Save, Trash2 } from "lucide-react";
import { useState, useEffect } from "react";
import {
  fetchSigningConfig,
  saveSigningConfig,
  validateSigningConfig,
  checkSigningReadiness,
  fetchSigningPrerequisites,
  deleteSigningConfig,
  type SigningConfig,
  type SigningReadinessResponse,
  type ValidationResult,
  type ToolDetectionResult
} from "../../lib/api";
import type { ScenariosResponse } from "../scenario-inventory/types";
import { fetchScenarioDesktopStatus } from "../../lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";
import { Button } from "../ui/button";
import { Label } from "../ui/label";
import { Checkbox } from "../ui/checkbox";
import { Select } from "../ui/select";
import { WindowsSigningForm } from "./WindowsSigningForm";
import { MacOSSigningForm } from "./MacOSSigningForm";
import { LinuxSigningForm } from "./LinuxSigningForm";
import { PrerequisitesPanel } from "./PrerequisitesPanel";
import { cn } from "../../lib/utils";

export function SigningPage() {
  const queryClient = useQueryClient();
  const [selectedScenario, setSelectedScenario] = useState<string>("");
  const [localConfig, setLocalConfig] = useState<SigningConfig>({
    enabled: false
  });
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false);

  // Fetch scenarios
  const { data: scenariosData } = useQuery<ScenariosResponse>({
    queryKey: ["scenarios-desktop-status"],
    queryFn: fetchScenarioDesktopStatus,
    refetchInterval: 30000
  });

  // Fetch signing config for selected scenario
  const { data: configData, isLoading: configLoading, refetch: refetchConfig } = useQuery({
    queryKey: ["signing-config", selectedScenario],
    queryFn: () => fetchSigningConfig(selectedScenario),
    enabled: !!selectedScenario
  });

  // Fetch readiness status
  const { data: readinessData, refetch: refetchReadiness } = useQuery<SigningReadinessResponse>({
    queryKey: ["signing-readiness", selectedScenario],
    queryFn: () => checkSigningReadiness(selectedScenario),
    enabled: !!selectedScenario
  });

  // Fetch prerequisites
  const { data: prerequisitesData } = useQuery<{ tools: ToolDetectionResult[] }>({
    queryKey: ["signing-prerequisites"],
    queryFn: fetchSigningPrerequisites
  });

  // Validation mutation
  const validateMutation = useMutation({
    mutationFn: () => validateSigningConfig(selectedScenario),
  });

  // Save mutation
  const saveMutation = useMutation({
    mutationFn: (config: SigningConfig) => saveSigningConfig(selectedScenario, config),
    onSuccess: () => {
      setHasUnsavedChanges(false);
      queryClient.invalidateQueries({ queryKey: ["signing-config", selectedScenario] });
      queryClient.invalidateQueries({ queryKey: ["signing-readiness", selectedScenario] });
    }
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: () => deleteSigningConfig(selectedScenario),
    onSuccess: () => {
      setLocalConfig({ enabled: false });
      setHasUnsavedChanges(false);
      queryClient.invalidateQueries({ queryKey: ["signing-config", selectedScenario] });
      queryClient.invalidateQueries({ queryKey: ["signing-readiness", selectedScenario] });
    }
  });

  // Sync local state when config data changes
  useEffect(() => {
    if (configData?.config) {
      setLocalConfig(configData.config);
      setHasUnsavedChanges(false);
    } else if (configData && !configData.config) {
      setLocalConfig({ enabled: false });
      setHasUnsavedChanges(false);
    }
  }, [configData]);

  const handleConfigChange = (updates: Partial<SigningConfig>) => {
    setLocalConfig(prev => ({ ...prev, ...updates }));
    setHasUnsavedChanges(true);
  };

  const handleSave = () => {
    saveMutation.mutate(localConfig);
  };

  const handleValidate = () => {
    validateMutation.mutate();
  };

  const handleDelete = () => {
    if (confirm("Are you sure you want to delete this signing configuration?")) {
      deleteMutation.mutate();
    }
  };

  const scenarios = scenariosData?.scenarios || [];

  return (
    <div className="space-y-6">
      {/* Header */}
      <Card className="border-slate-800/80 bg-slate-900/70">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Shield className="h-6 w-6 text-blue-400" />
            Code Signing Configuration
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-slate-300">
            Configure code signing for your desktop applications. Signed apps are trusted by operating systems
            and won&apos;t trigger security warnings during installation.
          </p>

          {/* Scenario Selector */}
          <div className="flex items-end gap-4">
            <div className="flex-1">
              <Label htmlFor="scenario-select">Scenario</Label>
              <Select
                id="scenario-select"
                value={selectedScenario}
                onChange={(e) => setSelectedScenario(e.target.value)}
                className="mt-1"
              >
                <option value="">Select a scenario...</option>
                {scenarios.map(scenario => (
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
                  <RefreshCw className={cn("h-4 w-4 mr-1", configLoading && "animate-spin")} />
                  Refresh
                </Button>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {selectedScenario && (
        <>
          {/* Readiness Status */}
          <ReadinessCard readiness={readinessData} />

          {/* Main Configuration */}
          <Card className="border-slate-800/80 bg-slate-900/70">
            <CardHeader className="flex flex-row items-center justify-between">
              <CardTitle className="text-lg">Configuration</CardTitle>
              <div className="flex items-center gap-3">
                <div className="flex items-center gap-2">
                  <Checkbox
                    id="signing-enabled"
                    checked={localConfig.enabled}
                    onChange={(e) => handleConfigChange({ enabled: e.target.checked })}
                  />
                  <Label htmlFor="signing-enabled" className="text-sm font-medium">
                    Enable Signing
                  </Label>
                </div>
              </div>
            </CardHeader>
            <CardContent className="space-y-6">
              {localConfig.enabled ? (
                <>
                  {/* Platform Tabs */}
                  <div className="grid gap-6 lg:grid-cols-3">
                    {/* Windows */}
                    <WindowsSigningForm
                      config={localConfig.windows}
                      onChange={(windows) => handleConfigChange({ windows })}
                    />

                    {/* macOS */}
                    <MacOSSigningForm
                      config={localConfig.macos}
                      onChange={(macos) => handleConfigChange({ macos })}
                    />

                    {/* Linux */}
                    <LinuxSigningForm
                      config={localConfig.linux}
                      onChange={(linux) => handleConfigChange({ linux })}
                    />
                  </div>

                  {/* Validation Results */}
                  {validateMutation.data && (
                    <ValidationResultsCard result={validateMutation.data} />
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
                    disabled={!localConfig.enabled || validateMutation.isPending}
                  >
                    {validateMutation.isPending ? (
                      <RefreshCw className="h-4 w-4 mr-1 animate-spin" />
                    ) : (
                      <CheckCircle className="h-4 w-4 mr-1" />
                    )}
                    Validate
                  </Button>
                  {configData?.config && (
                    <Button
                      variant="outline"
                      onClick={handleDelete}
                      disabled={deleteMutation.isPending}
                      className="text-red-400 hover:text-red-300 hover:border-red-800"
                    >
                      <Trash2 className="h-4 w-4 mr-1" />
                      Delete Config
                    </Button>
                  )}
                </div>

                <div className="flex items-center gap-2">
                  {hasUnsavedChanges && (
                    <span className="text-xs text-amber-400">Unsaved changes</span>
                  )}
                  <Button
                    onClick={handleSave}
                    disabled={!hasUnsavedChanges || saveMutation.isPending}
                  >
                    {saveMutation.isPending ? (
                      <RefreshCw className="h-4 w-4 mr-1 animate-spin" />
                    ) : (
                      <Save className="h-4 w-4 mr-1" />
                    )}
                    Save Configuration
                  </Button>
                </div>
              </div>

              {/* Error Display */}
              {(saveMutation.error || deleteMutation.error || validateMutation.error) && (
                <div className="p-3 rounded-lg bg-red-950/50 border border-red-800 text-red-200 text-sm">
                  {(saveMutation.error || deleteMutation.error || validateMutation.error)?.message}
                </div>
              )}
            </CardContent>
          </Card>

          {/* Prerequisites Panel */}
          <PrerequisitesPanel tools={prerequisitesData?.tools || []} />
        </>
      )}

      {!selectedScenario && (
        <Card className="border-slate-800/80 bg-slate-900/70">
          <CardContent className="py-12 text-center">
            <Shield className="h-12 w-12 mx-auto mb-4 text-slate-600" />
            <p className="text-slate-400">Select a scenario to configure code signing.</p>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

function ReadinessCard({ readiness }: { readiness?: SigningReadinessResponse }) {
  if (!readiness) return null;

  const platformOrder = ["windows", "macos", "linux"] as const;

  return (
    <Card className={cn(
      "border-slate-800/80 bg-slate-900/70",
      readiness.ready ? "border-green-800/50" : "border-amber-800/50"
    )}>
      <CardHeader className="pb-3">
        <CardTitle className="text-base flex items-center gap-2">
          {readiness.ready ? (
            <CheckCircle className="h-5 w-5 text-green-400" />
          ) : (
            <AlertCircle className="h-5 w-5 text-amber-400" />
          )}
          Signing Readiness
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-3 gap-4">
          {platformOrder.map(platform => {
            const status = readiness.platforms[platform];
            return (
              <div
                key={platform}
                className={cn(
                  "p-3 rounded-lg border",
                  status?.ready
                    ? "border-green-800/50 bg-green-950/30"
                    : "border-slate-800 bg-slate-950/30"
                )}
              >
                <div className="flex items-center gap-2 mb-1">
                  {status?.ready ? (
                    <CheckCircle className="h-4 w-4 text-green-400" />
                  ) : (
                    <XCircle className="h-4 w-4 text-slate-500" />
                  )}
                  <span className="font-medium capitalize">{platform === "macos" ? "macOS" : platform}</span>
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
      </CardContent>
    </Card>
  );
}

function ValidationResultsCard({ result }: { result: ValidationResult }) {
  return (
    <Card className={cn(
      "border",
      result.valid
        ? "border-green-800/50 bg-green-950/20"
        : "border-red-800/50 bg-red-950/20"
    )}>
      <CardHeader className="pb-3">
        <CardTitle className="text-base flex items-center gap-2">
          {result.valid ? (
            <CheckCircle className="h-5 w-5 text-green-400" />
          ) : (
            <XCircle className="h-5 w-5 text-red-400" />
          )}
          Validation {result.valid ? "Passed" : "Failed"}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {result.errors.length > 0 && (
          <div>
            <p className="text-sm font-medium text-red-300 mb-2">Errors:</p>
            <ul className="space-y-2">
              {result.errors.map((error, i) => (
                <li key={i} className="text-sm p-2 rounded bg-red-950/50 border border-red-800/50">
                  <div className="flex items-start gap-2">
                    <XCircle className="h-4 w-4 mt-0.5 text-red-400 flex-shrink-0" />
                    <div>
                      <p className="text-red-200">{error.message}</p>
                      {error.remediation && (
                        <p className="text-xs text-red-300/70 mt-1">{error.remediation}</p>
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
                <li key={i} className="text-sm p-2 rounded bg-amber-950/50 border border-amber-800/50">
                  <div className="flex items-start gap-2">
                    <AlertCircle className="h-4 w-4 mt-0.5 text-amber-400 flex-shrink-0" />
                    <p className="text-amber-200">{warning.message}</p>
                  </div>
                </li>
              ))}
            </ul>
          </div>
        )}

        {result.valid && result.errors.length === 0 && result.warnings.length === 0 && (
          <p className="text-sm text-green-300">All checks passed successfully.</p>
        )}
      </CardContent>
    </Card>
  );
}
