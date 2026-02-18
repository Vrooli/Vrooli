// Settings Dialog - Configuration management for vrooli-autoheal
// [REQ:CONFIG-*]
import { useState, useEffect, useRef, useCallback, useMemo, type ChangeEvent } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  X,
  Settings,
  Save,
  AlertTriangle,
  Power,
  Zap,
  Clock,
  Database,
  CheckCircle,
  XCircle,
  Loader2,
} from "lucide-react";
import { Button, ModalContent, ModalOverlay } from "../ui/primitives";
import { TabTrigger } from "../ui/composites";
import { useEscapeKey } from "../hooks/useEscapeKey";
import {
  fetchConfig,
  updateConfig,
  fetchDefaults,
  exportConfig,
  importConfig,
  setCheckEnabled,
  setCheckAutoHeal,
  bulkUpdateChecks,
  fetchChecks,
  fetchMonitoring,
  addScenario,
  removeScenario,
  setScenarioCritical,
  addResource,
  removeResource,
  type Config,
  type GlobalConfig,
} from "../../lib/api";
import {
  ChecksSettings,
  GeneralSettings,
  ImportExportSettings,
  MonitoringSettings,
  type CategoryIcon,
  type CheckWithConfig,
  type SettingsTab,
} from "./settings";

interface SettingsDialogProps {
  isOpen: boolean;
  onClose: () => void;
}

export function SettingsDialog({ isOpen, onClose }: SettingsDialogProps) {
  const queryClient = useQueryClient();
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

  const { data: config, isLoading: configLoading } = useQuery({
    queryKey: ["config"],
    queryFn: fetchConfig,
    enabled: isOpen,
  });

  const { data: defaults } = useQuery({
    queryKey: ["config-defaults"],
    queryFn: fetchDefaults,
    enabled: isOpen,
    staleTime: 60000,
  });

  const { data: checksMetadata } = useQuery({
    queryKey: ["checks-metadata"],
    queryFn: fetchChecks,
    enabled: isOpen,
    staleTime: 60000,
  });

  const { data: monitoring, isLoading: monitoringLoading } = useQuery({
    queryKey: ["monitoring"],
    queryFn: fetchMonitoring,
    enabled: isOpen,
    staleTime: 30000,
  });

  useEffect(() => {
    if (config && !localConfig) {
      setLocalConfig(config);
    }
  }, [config, localConfig]);

  useEffect(() => {
    if (!isOpen) {
      setLocalConfig(null);
      setHasChanges(false);
      setSaveStatus("idle");
    }
  }, [isOpen]);

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

  const addScenarioMutation = useMutation({
    mutationFn: ({ name, critical }: { name: string; critical: boolean }) => addScenario(name, critical),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["monitoring"] });
    },
  });

  const removeScenarioMutation = useMutation({
    mutationFn: (name: string) => removeScenario(name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["monitoring"] });
    },
  });

  const setCriticalMutation = useMutation({
    mutationFn: ({ name, critical }: { name: string; critical: boolean }) => setScenarioCritical(name, critical),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["monitoring"] });
    },
  });

  const addResourceMutation = useMutation({
    mutationFn: (name: string) => addResource(name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["monitoring"] });
    },
  });

  const removeResourceMutation = useMutation({
    mutationFn: (name: string) => removeResource(name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["monitoring"] });
    },
  });

  const updateGlobalConfig = useCallback(
    (key: keyof GlobalConfig, value: number) => {
      if (!localConfig) return;
      setLocalConfig({
        ...localConfig,
        global: { ...localConfig.global, [key]: value },
      });
      setHasChanges(true);
    },
    [localConfig]
  );

  const handleSave = useCallback(() => {
    if (!localConfig) return;
    setSaveStatus("saving");
    saveMutation.mutate(localConfig);
  }, [localConfig, saveMutation]);

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

  const handleImport = useCallback(
    async (e: ChangeEvent<HTMLInputElement>) => {
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

      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
    },
    [queryClient]
  );

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

  const checksByCategory = useMemo(() => {
    if (!checksMetadata || !config) return {};

    const groups: Record<string, CheckWithConfig[]> = {
      infrastructure: [],
      resource: [],
      system: [],
      scenario: [],
    };

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
        const fallbackGroup = groups.system;
        if (fallbackGroup) {
          fallbackGroup.push(enriched);
        }
      }
    }

    return groups;
  }, [checksMetadata, config, defaults]);

  const toggleCategory = useCallback((category: string) => {
    setExpandedCategories((prev) => ({
      ...prev,
      [category]: !prev[category],
    }));
  }, []);

  useEscapeKey(onClose, isOpen);

  if (!isOpen) return null;

  const categoryLabels: Record<string, string> = {
    infrastructure: "Infrastructure",
    resource: "Resources",
    system: "System",
    scenario: "Scenarios",
  };

  const categoryIcons: Record<string, CategoryIcon> = {
    infrastructure: Power,
    resource: Database,
    system: Clock,
    scenario: Zap,
  };

  return (
    <ModalOverlay onDismiss={onClose}>
      <ModalContent size="lg" data-testid="settings-dialog">
        <div className="flex items-center justify-between border-b border-border-default/70 px-6 py-4">
          <div className="flex items-center gap-3">
            <Settings className="text-accent-primary" size={24} />
            <h2 className="text-xl font-semibold">Settings</h2>
          </div>
          <Button
            onClick={onClose}
            variant="outline"
            size="icon"
            className="text-text-muted hover:text-text-primary"
            data-testid="settings-close"
          >
            <X size={20} />
          </Button>
        </div>

        <div className="flex border-b border-border-default/70 px-6">
          {[
            { id: "general", label: "General" },
            { id: "checks", label: "Health Checks" },
            { id: "monitoring", label: "Monitoring" },
            { id: "import-export", label: "Import / Export" },
          ].map((tab) => (
            <TabTrigger
              key={tab.id}
              onClick={() => setActiveTab(tab.id as SettingsTab)}
              active={activeTab === tab.id}
              size="regular"
            >
              {tab.label}
            </TabTrigger>
          ))}
        </div>

        <div className="overflow-y-auto p-6" style={{ maxHeight: "calc(90vh - 180px)" }}>
          {configLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="animate-spin text-accent-primary" size={32} />
            </div>
          ) : activeTab === "general" ? (
            <GeneralSettings config={localConfig} defaults={defaults} onChange={updateGlobalConfig} />
          ) : activeTab === "checks" ? (
            <ChecksSettings
              checksByCategory={checksByCategory}
              expandedCategories={expandedCategories}
              toggleCategory={toggleCategory}
              categoryLabels={categoryLabels}
              categoryIcons={categoryIcons}
              onToggleEnabled={(checkId, enabled) => toggleEnabledMutation.mutate({ checkId, enabled })}
              onToggleAutoHeal={(checkId, autoHeal) => toggleAutoHealMutation.mutate({ checkId, autoHeal })}
              onBulkUpdate={(action) => bulkMutation.mutate(action)}
              isUpdating={toggleEnabledMutation.isPending || toggleAutoHealMutation.isPending || bulkMutation.isPending}
            />
          ) : activeTab === "monitoring" ? (
            <MonitoringSettings
              monitoring={monitoring}
              isLoading={monitoringLoading}
              onAddScenario={(name, critical) => addScenarioMutation.mutate({ name, critical })}
              onRemoveScenario={(name) => removeScenarioMutation.mutate(name)}
              onSetCritical={(name, critical) => setCriticalMutation.mutate({ name, critical })}
              onAddResource={(name) => addResourceMutation.mutate(name)}
              onRemoveResource={(name) => removeResourceMutation.mutate(name)}
              isUpdating={
                addScenarioMutation.isPending ||
                removeScenarioMutation.isPending ||
                setCriticalMutation.isPending ||
                addResourceMutation.isPending ||
                removeResourceMutation.isPending
              }
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

        <div className="flex items-center justify-between border-t border-border-default/70 px-6 py-4">
          <div className="text-sm text-text-muted">
            {hasChanges && (
              <span className="flex items-center gap-2 text-accent-warning">
                <AlertTriangle size={16} />
                Unsaved changes
              </span>
            )}
            {saveStatus === "saved" && (
              <span className="flex items-center gap-2 text-accent-success">
                <CheckCircle size={16} />
                Saved successfully
              </span>
            )}
            {saveStatus === "error" && (
              <span className="flex items-center gap-2 text-accent-danger">
                <XCircle size={16} />
                Failed to save
              </span>
            )}
          </div>
          <div className="flex gap-3">
            <Button variant="outline" onClick={onClose}>
              Cancel
            </Button>
            <Button onClick={handleSave} disabled={!hasChanges || saveStatus === "saving"} data-testid="settings-save">
              {saveStatus === "saving" ? (
                <>
                  <Loader2 className="mr-2 animate-spin" size={16} />
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

        <input ref={fileInputRef} type="file" accept=".json" onChange={handleImport} className="hidden" />
      </ModalContent>
    </ModalOverlay>
  );
}

export default SettingsDialog;
