// Settings Dialog - Configuration management for vrooli-autoheal
// [REQ:CONFIG-*]
import { useState, useEffect, useRef, useCallback, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  X, Settings, Save, Upload, Download, RotateCcw, AlertTriangle,
  ChevronDown, ChevronRight, Power, Zap, Clock, Database,
  CheckCircle, XCircle, Loader2
} from "lucide-react";
import { Button } from "./ui/button";
import {
  fetchConfig, updateConfig, fetchDefaults, exportConfig, importConfig,
  setCheckEnabled, setCheckAutoHeal, bulkUpdateChecks, fetchChecks,
  Config, CheckConfig, GlobalConfig, DefaultsResponse, CheckInfo
} from "../lib/api";

interface SettingsDialogProps {
  isOpen: boolean;
  onClose: () => void;
}

type SettingsTab = "general" | "checks" | "import-export";

interface CheckWithConfig extends CheckInfo {
  config: {
    enabled: boolean;
    autoHeal: boolean;
  };
}

export function SettingsDialog({ isOpen, onClose }: SettingsDialogProps) {
  const queryClient = useQueryClient();
  const dialogRef = useRef<HTMLDivElement>(null);
  const [activeTab, setActiveTab] = useState<SettingsTab>("general");
  const [localConfig, setLocalConfig] = useState<Config | null>(null);
  const [hasChanges, setHasChanges] = useState(false);
  const [expandedCategories, setExpandedCategories] = useState<Record<string, boolean>>({
    infrastructure: true,
    resource: true,
    system: true,
    scenario: true,
  });
  const [saveStatus, setSaveStatus] = useState<"idle" | "saving" | "saved" | "error">("idle");
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Fetch current config
  const { data: config, isLoading: configLoading } = useQuery({
    queryKey: ["config"],
    queryFn: fetchConfig,
    enabled: isOpen,
  });

  // Fetch defaults for reference
  const { data: defaults } = useQuery({
    queryKey: ["config-defaults"],
    queryFn: fetchDefaults,
    enabled: isOpen,
    staleTime: 60000,
  });

  // Fetch check metadata
  const { data: checksMetadata } = useQuery({
    queryKey: ["checks-metadata"],
    queryFn: fetchChecks,
    enabled: isOpen,
    staleTime: 60000,
  });

  // Initialize local config when API data loads
  useEffect(() => {
    if (config && !localConfig) {
      setLocalConfig(config);
    }
  }, [config, localConfig]);

  // Reset local config when dialog closes
  useEffect(() => {
    if (!isOpen) {
      setLocalConfig(null);
      setHasChanges(false);
      setSaveStatus("idle");
    }
  }, [isOpen]);

  // Save mutation
  const saveMutation = useMutation({
    mutationFn: updateConfig,
    onSuccess: () => {
      setSaveStatus("saved");
      setHasChanges(false);
      queryClient.invalidateQueries({ queryKey: ["config"] });
      setTimeout(() => setSaveStatus("idle"), 2000);
    },
    onError: () => {
      setSaveStatus("error");
      setTimeout(() => setSaveStatus("idle"), 3000);
    },
  });

  // Check toggle mutations
  const toggleEnabledMutation = useMutation({
    mutationFn: ({ checkId, enabled }: { checkId: string; enabled: boolean }) =>
      setCheckEnabled(checkId, enabled),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["config"] });
    },
  });

  const toggleAutoHealMutation = useMutation({
    mutationFn: ({ checkId, autoHeal }: { checkId: string; autoHeal: boolean }) =>
      setCheckAutoHeal(checkId, autoHeal),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["config"] });
    },
  });

  const bulkMutation = useMutation({
    mutationFn: bulkUpdateChecks,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["config"] });
    },
  });

  // Handle global config changes
  const updateGlobalConfig = useCallback((key: keyof GlobalConfig, value: number) => {
    if (!localConfig) return;
    setLocalConfig({
      ...localConfig,
      global: { ...localConfig.global, [key]: value },
    });
    setHasChanges(true);
  }, [localConfig]);

  // Handle save
  const handleSave = useCallback(() => {
    if (!localConfig) return;
    setSaveStatus("saving");
    saveMutation.mutate(localConfig);
  }, [localConfig, saveMutation]);

  // Handle export
  const handleExport = useCallback(async () => {
    try {
      const blob = await exportConfig();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "autoheal-config.json";
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch {
      alert("Failed to export configuration");
    }
  }, []);

  // Handle import
  const handleImport = useCallback(async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    try {
      const text = await file.text();
      const result = await importConfig(text);
      if (result.success) {
        setLocalConfig(result.config);
        queryClient.invalidateQueries({ queryKey: ["config"] });
        alert("Configuration imported successfully");
      }
    } catch {
      alert("Failed to import configuration. Please check the file format.");
    }

    // Reset file input
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
  }, [queryClient]);

  // Handle reset to defaults
  const handleReset = useCallback(() => {
    if (!defaults) return;
    if (!confirm("Reset all settings to defaults? This cannot be undone.")) return;

    const resetConfig: Config = {
      version: "1.0",
      global: defaults.global,
      checks: {},
      ui: defaults.ui,
    };
    setLocalConfig(resetConfig);
    setHasChanges(true);
  }, [defaults]);

  // Group checks by category
  const checksByCategory = useMemo(() => {
    if (!checksMetadata || !config) return {};

    const groups: Record<string, CheckWithConfig[]> = {
      infrastructure: [],
      resource: [],
      system: [],
      scenario: [],
    };

    // Defensive: config.checks might be undefined
    const configChecks = config.checks || {};
    const defaultChecks = defaults?.checks || {};

    for (const check of checksMetadata) {
      const category = check.category || "system";
      const checkConfig = configChecks[check.id] || {};
      const defaultConfig = defaultChecks[check.id];

      const enriched: CheckWithConfig = {
        ...check,
        config: {
          enabled: checkConfig.enabled ?? defaultConfig?.enabled ?? true,
          autoHeal: checkConfig.autoHeal ?? defaultConfig?.autoHeal ?? false,
        },
      };

      if (groups[category]) {
        groups[category].push(enriched);
      } else {
        groups.system.push(enriched);
      }
    }

    return groups;
  }, [checksMetadata, config, defaults]);

  // Toggle category expansion
  const toggleCategory = useCallback((category: string) => {
    setExpandedCategories((prev) => ({
      ...prev,
      [category]: !prev[category],
    }));
  }, []);

  // Close on escape
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape" && isOpen) {
        onClose();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, onClose]);

  // Close on click outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (dialogRef.current && !dialogRef.current.contains(e.target as Node)) {
        onClose();
      }
    };
    if (isOpen) {
      document.addEventListener("mousedown", handleClickOutside);
    }
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const categoryLabels: Record<string, string> = {
    infrastructure: "Infrastructure",
    resource: "Resources",
    system: "System",
    scenario: "Scenarios",
  };

  const categoryIcons: Record<string, typeof Power> = {
    infrastructure: Power,
    resource: Database,
    system: Clock,
    scenario: Zap,
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div
        ref={dialogRef}
        className="relative w-full max-w-3xl max-h-[90vh] overflow-hidden rounded-xl border border-white/10 bg-slate-900 shadow-2xl"
        data-testid="settings-dialog"
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-white/10 px-6 py-4">
          <div className="flex items-center gap-3">
            <Settings className="text-blue-400" size={24} />
            <h2 className="text-xl font-semibold">Settings</h2>
          </div>
          <button
            onClick={onClose}
            className="rounded-lg p-2 text-slate-400 hover:bg-white/5 hover:text-slate-200"
            data-testid="settings-close"
          >
            <X size={20} />
          </button>
        </div>

        {/* Tab Navigation */}
        <div className="flex border-b border-white/10 px-6">
          {[
            { id: "general", label: "General" },
            { id: "checks", label: "Health Checks" },
            { id: "import-export", label: "Import / Export" },
          ].map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as SettingsTab)}
              className={`px-4 py-3 text-sm font-medium border-b-2 transition-colors ${
                activeTab === tab.id
                  ? "border-blue-400 text-blue-400"
                  : "border-transparent text-slate-400 hover:text-slate-200"
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>

        {/* Content */}
        <div className="overflow-y-auto p-6" style={{ maxHeight: "calc(90vh - 180px)" }}>
          {configLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="animate-spin text-blue-400" size={32} />
            </div>
          ) : activeTab === "general" ? (
            <GeneralSettings
              config={localConfig}
              defaults={defaults}
              onChange={updateGlobalConfig}
            />
          ) : activeTab === "checks" ? (
            <ChecksSettings
              checksByCategory={checksByCategory}
              expandedCategories={expandedCategories}
              toggleCategory={toggleCategory}
              categoryLabels={categoryLabels}
              categoryIcons={categoryIcons}
              onToggleEnabled={(checkId, enabled) =>
                toggleEnabledMutation.mutate({ checkId, enabled })
              }
              onToggleAutoHeal={(checkId, autoHeal) =>
                toggleAutoHealMutation.mutate({ checkId, autoHeal })
              }
              onBulkUpdate={(action) => bulkMutation.mutate(action)}
              isUpdating={toggleEnabledMutation.isPending || toggleAutoHealMutation.isPending || bulkMutation.isPending}
            />
          ) : (
            <ImportExportSettings
              onExport={handleExport}
              onImport={() => fileInputRef.current?.click()}
              onReset={handleReset}
              config={localConfig}
            />
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between border-t border-white/10 px-6 py-4">
          <div className="text-sm text-slate-400">
            {hasChanges && (
              <span className="flex items-center gap-2 text-amber-400">
                <AlertTriangle size={16} />
                Unsaved changes
              </span>
            )}
            {saveStatus === "saved" && (
              <span className="flex items-center gap-2 text-emerald-400">
                <CheckCircle size={16} />
                Saved successfully
              </span>
            )}
            {saveStatus === "error" && (
              <span className="flex items-center gap-2 text-red-400">
                <XCircle size={16} />
                Failed to save
              </span>
            )}
          </div>
          <div className="flex gap-3">
            <Button variant="outline" onClick={onClose}>
              Cancel
            </Button>
            <Button
              onClick={handleSave}
              disabled={!hasChanges || saveStatus === "saving"}
              data-testid="settings-save"
            >
              {saveStatus === "saving" ? (
                <>
                  <Loader2 className="animate-spin mr-2" size={16} />
                  Saving...
                </>
              ) : (
                <>
                  <Save className="mr-2" size={16} />
                  Save Changes
                </>
              )}
            </Button>
          </div>
        </div>

        {/* Hidden file input for import */}
        <input
          ref={fileInputRef}
          type="file"
          accept=".json"
          onChange={handleImport}
          className="hidden"
        />
      </div>
    </div>
  );
}

// General Settings Tab
interface GeneralSettingsProps {
  config: Config | null;
  defaults: DefaultsResponse | undefined;
  onChange: (key: keyof GlobalConfig, value: number) => void;
}

function GeneralSettings({ config, defaults, onChange }: GeneralSettingsProps) {
  if (!config) return null;

  const fields: Array<{
    key: keyof GlobalConfig;
    label: string;
    description: string;
    min: number;
    max: number;
    unit: string;
  }> = [
    {
      key: "gracePeriodSeconds",
      label: "Grace Period",
      description: "Wait time after boot before running health checks",
      min: 0,
      max: 600,
      unit: "seconds",
    },
    {
      key: "tickIntervalSeconds",
      label: "Tick Interval",
      description: "How often to run health check cycles",
      min: 10,
      max: 3600,
      unit: "seconds",
    },
    {
      key: "verifyDelaySeconds",
      label: "Verify Delay",
      description: "Wait time after restart before re-checking health",
      min: 5,
      max: 300,
      unit: "seconds",
    },
    {
      key: "maxRestartAttempts",
      label: "Max Restart Attempts",
      description: "Maximum restart attempts before giving up",
      min: 1,
      max: 10,
      unit: "attempts",
    },
    {
      key: "restartCooldownSeconds",
      label: "Restart Cooldown",
      description: "Minimum time between restarts of the same service",
      min: 60,
      max: 3600,
      unit: "seconds",
    },
    {
      key: "historyRetentionHours",
      label: "History Retention",
      description: "How long to keep check history",
      min: 1,
      max: 168,
      unit: "hours",
    },
  ];

  return (
    <div className="space-y-6">
      <div className="rounded-lg border border-white/10 bg-white/5 p-4">
        <h3 className="text-lg font-medium mb-4">Global Settings</h3>
        <div className="space-y-4">
          {fields.map((field) => (
            <div key={field.key} className="grid grid-cols-3 gap-4 items-start">
              <div>
                <label className="text-sm font-medium text-slate-200">{field.label}</label>
                <p className="text-xs text-slate-400 mt-1">{field.description}</p>
              </div>
              <div className="col-span-2 flex items-center gap-3">
                <input
                  type="number"
                  min={field.min}
                  max={field.max}
                  value={config.global[field.key]}
                  onChange={(e) => onChange(field.key, parseInt(e.target.value) || field.min)}
                  className="w-24 rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm text-slate-200 focus:border-blue-400 focus:outline-none"
                />
                <span className="text-sm text-slate-400">{field.unit}</span>
                {defaults && (
                  <span className="text-xs text-slate-500">
                    (default: {defaults.global[field.key]})
                  </span>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

// Checks Settings Tab
interface ChecksSettingsProps {
  checksByCategory: Record<string, CheckWithConfig[]>;
  expandedCategories: Record<string, boolean>;
  toggleCategory: (category: string) => void;
  categoryLabels: Record<string, string>;
  categoryIcons: Record<string, typeof Power>;
  onToggleEnabled: (checkId: string, enabled: boolean) => void;
  onToggleAutoHeal: (checkId: string, autoHeal: boolean) => void;
  onBulkUpdate: (action: "enableAll" | "disableAll" | "autoHealAll" | "disableAutoHealAll") => void;
  isUpdating: boolean;
}

function ChecksSettings({
  checksByCategory,
  expandedCategories,
  toggleCategory,
  categoryLabels,
  categoryIcons,
  onToggleEnabled,
  onToggleAutoHeal,
  onBulkUpdate,
  isUpdating,
}: ChecksSettingsProps) {
  return (
    <div className="space-y-4">
      {/* Bulk Actions */}
      <div className="flex flex-wrap gap-2 mb-4">
        <Button
          variant="outline"
          size="sm"
          onClick={() => onBulkUpdate("enableAll")}
          disabled={isUpdating}
        >
          Enable All
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={() => onBulkUpdate("disableAll")}
          disabled={isUpdating}
        >
          Disable All
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={() => onBulkUpdate("autoHealAll")}
          disabled={isUpdating}
        >
          Enable All Auto-Heal
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={() => onBulkUpdate("disableAutoHealAll")}
          disabled={isUpdating}
        >
          Disable All Auto-Heal
        </Button>
      </div>

      {/* Categories */}
      {Object.entries(checksByCategory).map(([category, checks]) => {
        if (checks.length === 0) return null;
        const Icon = categoryIcons[category] || Power;
        const isExpanded = expandedCategories[category];

        return (
          <div key={category} className="rounded-lg border border-white/10 overflow-hidden">
            <button
              onClick={() => toggleCategory(category)}
              className="w-full flex items-center justify-between px-4 py-3 bg-white/5 hover:bg-white/10 transition-colors"
            >
              <div className="flex items-center gap-3">
                <Icon size={18} className="text-blue-400" />
                <span className="font-medium">{categoryLabels[category]}</span>
                <span className="text-xs text-slate-400">({checks.length} checks)</span>
              </div>
              {isExpanded ? <ChevronDown size={18} /> : <ChevronRight size={18} />}
            </button>

            {isExpanded && (
              <div className="divide-y divide-white/5">
                {checks.map((check) => (
                  <div
                    key={check.id}
                    className="flex items-center justify-between px-4 py-3 hover:bg-white/5"
                  >
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-sm">{check.title}</span>
                        <span className="text-xs text-slate-500">({check.intervalSeconds}s)</span>
                      </div>
                      <p className="text-xs text-slate-400 truncate">{check.description}</p>
                    </div>

                    <div className="flex items-center gap-4 ml-4">
                      {/* Enabled Toggle */}
                      <label className="flex items-center gap-2 cursor-pointer">
                        <span className="text-xs text-slate-400">Enabled</span>
                        <button
                          onClick={() => onToggleEnabled(check.id, !check.config.enabled)}
                          disabled={isUpdating}
                          className={`relative w-10 h-5 rounded-full transition-colors ${
                            check.config.enabled ? "bg-emerald-500" : "bg-slate-600"
                          }`}
                        >
                          <span
                            className={`absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-white transition-transform ${
                              check.config.enabled ? "translate-x-5" : ""
                            }`}
                          />
                        </button>
                      </label>

                      {/* Auto-Heal Toggle */}
                      <label className="flex items-center gap-2 cursor-pointer">
                        <span className="text-xs text-slate-400">Auto-Heal</span>
                        <button
                          onClick={() => onToggleAutoHeal(check.id, !check.config.autoHeal)}
                          disabled={isUpdating || !check.config.enabled}
                          className={`relative w-10 h-5 rounded-full transition-colors ${
                            check.config.autoHeal && check.config.enabled
                              ? "bg-blue-500"
                              : "bg-slate-600"
                          } ${!check.config.enabled ? "opacity-50 cursor-not-allowed" : ""}`}
                        >
                          <span
                            className={`absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-white transition-transform ${
                              check.config.autoHeal && check.config.enabled ? "translate-x-5" : ""
                            }`}
                          />
                        </button>
                      </label>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}

// Import/Export Tab
interface ImportExportSettingsProps {
  onExport: () => void;
  onImport: () => void;
  onReset: () => void;
  config: Config | null;
}

function ImportExportSettings({ onExport, onImport, onReset, config }: ImportExportSettingsProps) {
  return (
    <div className="space-y-6">
      {/* Export */}
      <div className="rounded-lg border border-white/10 bg-white/5 p-4">
        <h3 className="text-lg font-medium mb-2">Export Configuration</h3>
        <p className="text-sm text-slate-400 mb-4">
          Download your current configuration as a JSON file. You can use this to backup your
          settings or transfer them to another installation.
        </p>
        <Button onClick={onExport}>
          <Download className="mr-2" size={16} />
          Export Configuration
        </Button>
      </div>

      {/* Import */}
      <div className="rounded-lg border border-white/10 bg-white/5 p-4">
        <h3 className="text-lg font-medium mb-2">Import Configuration</h3>
        <p className="text-sm text-slate-400 mb-4">
          Load a previously exported configuration file. This will replace your current settings.
        </p>
        <Button variant="outline" onClick={onImport}>
          <Upload className="mr-2" size={16} />
          Import Configuration
        </Button>
      </div>

      {/* Reset */}
      <div className="rounded-lg border border-amber-500/20 bg-amber-500/5 p-4">
        <h3 className="text-lg font-medium mb-2 text-amber-400">Reset to Defaults</h3>
        <p className="text-sm text-slate-400 mb-4">
          Reset all settings to their default values. This will clear all your customizations.
        </p>
        <Button
          variant="outline"
          onClick={onReset}
          className="border-amber-500/30 text-amber-400 hover:bg-amber-500/10"
        >
          <RotateCcw className="mr-2" size={16} />
          Reset to Defaults
        </Button>
      </div>

      {/* Current Config Preview */}
      {config && (
        <div className="rounded-lg border border-white/10 bg-white/5 p-4">
          <h3 className="text-lg font-medium mb-2">Current Configuration</h3>
          <pre className="text-xs text-slate-400 overflow-auto max-h-48 bg-black/20 rounded p-3">
            {JSON.stringify(config, null, 2)}
          </pre>
        </div>
      )}
    </div>
  );
}

export default SettingsDialog;
