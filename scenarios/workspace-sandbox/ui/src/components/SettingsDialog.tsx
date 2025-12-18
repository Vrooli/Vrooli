import { useState, useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import {
  Settings,
  RefreshCw,
  Check,
  X,
  ChevronDown,
  ChevronRight,
  Info,
  Cpu,
  HardDrive,
  Shield,
  Loader2,
  Copy,
  CheckCircle2,
  AlertTriangle,
  Play,
  FileStack,
  Container,
  Wifi,
  FolderOpen,
  Minus,
  Plus,
  Trash2,
  Save,
  Pencil,
  X as XIcon,
  Clock,
  MemoryStick,
  Hash,
  File,
} from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogClose,
} from "./ui/dialog";
import { Button } from "./ui/button";
import { Badge } from "./ui/badge";
import { Input } from "./ui/input";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "./ui/tabs";
import {
  useDriverOptions,
  useSelectDriver,
  useExecutionConfig,
  useUpdateExecutionConfig,
  useProfiles,
  useSaveProfile,
  useDeleteProfile,
  queryKeys,
} from "../lib/hooks";
import type {
  DriverOption,
  DriverRequirement,
  SelectDriverResponse,
  ExecutionConfig,
  ResourceLimitsConfig,
  IsolationProfile,
  NetworkAccess,
} from "../lib/api";

interface SettingsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

// --- Helper Components ---

function RequirementItem({ requirement }: { requirement: DriverRequirement }) {
  const [copied, setCopied] = useState(false);

  const copyToClipboard = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      const textarea = document.createElement("textarea");
      textarea.value = text;
      document.body.appendChild(textarea);
      textarea.select();
      document.execCommand("copy");
      document.body.removeChild(textarea);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const isOptionalUnmet = requirement.optional && !requirement.met;

  return (
    <div className="flex items-start gap-2 py-1.5">
      {requirement.met ? (
        <Check className="h-4 w-4 text-emerald-400 mt-0.5 flex-shrink-0" />
      ) : requirement.optional ? (
        <Minus className="h-4 w-4 text-amber-400 mt-0.5 flex-shrink-0" />
      ) : (
        <X className="h-4 w-4 text-red-400 mt-0.5 flex-shrink-0" />
      )}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 flex-wrap">
          <span
            className={`text-sm ${
              requirement.met ? "text-slate-300" : isOptionalUnmet ? "text-slate-400" : "text-slate-400"
            }`}
          >
            {requirement.name}
          </span>
          {requirement.optional && (
            <Badge variant="default" className="text-[9px] bg-slate-700 text-slate-400">
              Optional
            </Badge>
          )}
          {requirement.current && (
            <span className="text-xs text-slate-500 font-mono">
              ({requirement.current})
            </span>
          )}
        </div>
        {!requirement.met && requirement.howToFix && (
          <div className="mt-1.5 flex items-start gap-2 group">
            <code className={`flex-1 text-xs bg-slate-800 px-2 py-1 rounded font-mono break-all ${
              requirement.optional ? "text-blue-300" : "text-amber-300"
            }`}>
              {requirement.howToFix}
            </code>
            <Button
              variant="ghost"
              size="sm"
              className="h-6 w-6 p-0 opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0"
              onClick={() => copyToClipboard(requirement.howToFix!)}
              title="Copy to clipboard"
            >
              {copied ? (
                <CheckCircle2 className="h-3.5 w-3.5 text-emerald-400" />
              ) : (
                <Copy className="h-3.5 w-3.5 text-slate-400" />
              )}
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}

function CapabilityIndicator({ enabled, label }: { enabled: boolean; label: string }) {
  return (
    <div className="flex items-center gap-1.5" title={label}>
      {enabled ? (
        <Check className="h-3.5 w-3.5 text-emerald-400" />
      ) : (
        <X className="h-3.5 w-3.5 text-slate-600" />
      )}
    </div>
  );
}

function CapabilityMatrix({ options }: { options: DriverOption[] }) {
  const capabilities = [
    { key: "filesystemIsolation", label: "Filesystem", icon: FileStack, description: "Copy-on-write protection" },
    { key: "processIsolation", label: "Process", icon: Container, description: "Namespace isolation via bwrap" },
    { key: "networkIsolation", label: "Network", icon: Wifi, description: "Network access control" },
    { key: "directAccess", label: "Direct Access", icon: FolderOpen, description: "Merged dir accessible to tools" },
  ] as const;

  return (
    <div className="rounded-lg border border-slate-700 bg-slate-800/30 overflow-hidden">
      <table className="w-full text-xs">
        <thead>
          <tr className="border-b border-slate-700">
            <th className="text-left py-2 px-3 text-slate-400 font-medium">Driver</th>
            {capabilities.map((cap) => (
              <th key={cap.key} className="py-2 px-2 text-center" title={cap.description}>
                <div className="flex flex-col items-center gap-0.5">
                  <cap.icon className="h-3.5 w-3.5 text-slate-500" />
                  <span className="text-slate-500 font-normal">{cap.label}</span>
                </div>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {options.map((option) => (
            <tr key={option.id} className="border-b border-slate-800 last:border-0">
              <td className="py-2 px-3">
                <span className={`font-medium ${option.available ? "text-slate-300" : "text-slate-500"}`}>
                  {option.name.replace(" (Fallback)", "")}
                </span>
              </td>
              {capabilities.map((cap) => (
                <td key={cap.key} className="py-2 px-2 text-center">
                  <div className="flex justify-center">
                    <CapabilityIndicator
                      enabled={option.capabilities?.[cap.key] ?? false}
                      label={cap.description}
                    />
                  </div>
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function DriverOptionCard({
  option,
  isCurrentDriver,
  onSelect,
  isSelecting,
}: {
  option: DriverOption;
  isCurrentDriver: boolean;
  onSelect: (driverId: string) => void;
  isSelecting: boolean;
}) {
  const [expanded, setExpanded] = useState(!option.available || isCurrentDriver);

  const metCount = option.requirements.filter((r) => r.met).length;
  const totalCount = option.requirements.length;
  const allMet = metCount === totalCount;

  return (
    <div
      className={`rounded-lg border ${
        isCurrentDriver
          ? "border-emerald-700 bg-emerald-950/30"
          : option.available
          ? "border-slate-700 bg-slate-800/50"
          : "border-slate-800 bg-slate-900/50"
      }`}
    >
      <button
        type="button"
        className="w-full px-4 py-3 flex items-center gap-3 text-left"
        onClick={() => setExpanded(!expanded)}
      >
        {expanded ? (
          <ChevronDown className="h-4 w-4 text-slate-500 flex-shrink-0" />
        ) : (
          <ChevronRight className="h-4 w-4 text-slate-500 flex-shrink-0" />
        )}

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="font-medium text-slate-200">{option.name}</span>
            {isCurrentDriver && (
              <Badge variant="success" className="text-[10px]">
                Current
              </Badge>
            )}
            {option.recommended && !isCurrentDriver && (
              <Badge variant="default" className="text-[10px] bg-blue-900/50 text-blue-300 border-blue-700">
                Recommended
              </Badge>
            )}
          </div>
          <p className="text-xs text-slate-500 mt-0.5 line-clamp-2">
            {option.description}
          </p>
        </div>

        <div className="flex-shrink-0">
          {totalCount === 0 ? (
            <Badge variant="success" className="text-[10px]">
              Always Available
            </Badge>
          ) : allMet ? (
            <Badge variant="success" className="text-[10px]">
              {metCount}/{totalCount} Ready
            </Badge>
          ) : (
            <Badge variant="warning" className="text-[10px]">
              {metCount}/{totalCount} Met
            </Badge>
          )}
        </div>
      </button>

      {expanded && (
        <div className="px-4 pb-3 pt-0 border-t border-slate-700/50">
          {option.requirements.length > 0 && (
            <div className="mt-3 space-y-1">
              {option.requirements.map((req, i) => (
                <RequirementItem key={i} requirement={req} />
              ))}
            </div>
          )}

          {option.available && !isCurrentDriver && (
            <div className="mt-3 pt-3 border-t border-slate-700/50">
              <Button
                size="sm"
                variant="outline"
                onClick={(e) => {
                  e.stopPropagation();
                  onSelect(option.id);
                }}
                disabled={isSelecting}
                className="w-full"
              >
                {isSelecting ? (
                  <>
                    <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
                    Selecting...
                  </>
                ) : (
                  <>
                    <Play className="h-3.5 w-3.5 mr-1.5" />
                    Use this driver
                  </>
                )}
              </Button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// --- Limit Input Component ---

function LimitInput({
  label,
  icon: Icon,
  value,
  onChange,
  unit,
  description,
}: {
  label: string;
  icon: React.ElementType;
  value: number;
  onChange: (value: number) => void;
  unit?: string;
  description?: string;
}) {
  return (
    <div className="space-y-1">
      <label className="flex items-center gap-1.5 text-xs text-slate-400">
        <Icon className="h-3.5 w-3.5" />
        {label}
      </label>
      <div className="flex items-center gap-2">
        <Input
          type="number"
          min={0}
          value={value}
          onChange={(e) => onChange(parseInt(e.target.value) || 0)}
          className="h-8 text-sm"
        />
        {unit && <span className="text-xs text-slate-500">{unit}</span>}
      </div>
      {description && (
        <p className="text-[10px] text-slate-600">{description}</p>
      )}
    </div>
  );
}

// --- Driver Tab ---

function DriverTab() {
  const queryClient = useQueryClient();
  const driverOptionsQuery = useDriverOptions();
  const selectDriverMutation = useSelectDriver();
  const data = driverOptionsQuery.data;
  const [selectResult, setSelectResult] = useState<SelectDriverResponse | null>(null);

  const handleRefresh = () => {
    queryClient.invalidateQueries({ queryKey: queryKeys.driverOptions });
    setSelectResult(null);
  };

  const handleSelectDriver = (driverId: string) => {
    setSelectResult(null);
    selectDriverMutation.mutate(driverId, {
      onSuccess: (result) => {
        setSelectResult(result);
      },
    });
  };

  return (
    <div className="space-y-6">
      {/* System Info */}
      {data && (
        <section>
          <h3 className="text-sm font-medium text-slate-300 mb-3 flex items-center gap-2">
            <Cpu className="h-4 w-4 text-slate-500" />
            System Information
          </h3>
          <div className="grid grid-cols-2 gap-3">
            <div className="rounded-md bg-slate-800/50 px-3 py-2">
              <div className="text-[10px] uppercase tracking-wider text-slate-500">
                Operating System
              </div>
              <div className="text-sm text-slate-200 font-medium capitalize">
                {data.os}
              </div>
            </div>
            {data.kernel && (
              <div className="rounded-md bg-slate-800/50 px-3 py-2">
                <div className="text-[10px] uppercase tracking-wider text-slate-500">
                  Kernel Version
                </div>
                <div className="text-sm text-slate-200 font-medium font-mono">
                  {data.kernel}
                </div>
              </div>
            )}
            <div className="rounded-md bg-slate-800/50 px-3 py-2">
              <div className="text-[10px] uppercase tracking-wider text-slate-500">
                User Namespace
              </div>
              <div className="text-sm text-slate-200 font-medium flex items-center gap-1.5">
                {data.inUserNamespace ? (
                  <>
                    <Shield className="h-3.5 w-3.5 text-emerald-400" />
                    <span>Enabled</span>
                  </>
                ) : (
                  <>
                    <Shield className="h-3.5 w-3.5 text-slate-500" />
                    <span>Not Active</span>
                  </>
                )}
              </div>
            </div>
            <div className="rounded-md bg-slate-800/50 px-3 py-2">
              <div className="text-[10px] uppercase tracking-wider text-slate-500">
                Current Driver
              </div>
              <div className="text-sm text-emerald-400 font-medium font-mono">
                {data.currentDriver}
              </div>
            </div>
          </div>
        </section>
      )}

      {/* Capability Matrix */}
      {data?.options && data.options.length > 0 && (
        <section>
          <h3 className="text-sm font-medium text-slate-300 mb-3 flex items-center gap-2">
            <Shield className="h-4 w-4 text-slate-500" />
            Capability Comparison
          </h3>
          <CapabilityMatrix options={data.options} />
        </section>
      )}

      {/* Driver Options */}
      <section>
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-medium text-slate-300 flex items-center gap-2">
            <HardDrive className="h-4 w-4 text-slate-500" />
            Driver Options
          </h3>
          <Button
            variant="ghost"
            size="sm"
            onClick={handleRefresh}
            disabled={driverOptionsQuery.isLoading || driverOptionsQuery.isFetching}
            className="h-7 px-2"
          >
            <RefreshCw
              className={`h-3.5 w-3.5 mr-1.5 ${
                driverOptionsQuery.isFetching ? "animate-spin" : ""
              }`}
            />
            Refresh
          </Button>
        </div>

        {selectResult && (
          <div
            className={`rounded-lg border p-3 mb-4 ${
              selectResult.requiresRestart
                ? "border-amber-700 bg-amber-950/30 text-amber-200"
                : "border-emerald-700 bg-emerald-950/30 text-emerald-200"
            }`}
          >
            <div className="flex items-start gap-2">
              {selectResult.requiresRestart ? (
                <AlertTriangle className="h-4 w-4 mt-0.5 flex-shrink-0" />
              ) : (
                <CheckCircle2 className="h-4 w-4 mt-0.5 flex-shrink-0" />
              )}
              <div className="text-sm">{selectResult.message}</div>
            </div>
          </div>
        )}

        {selectDriverMutation.error && (
          <div className="rounded-lg border border-red-800 bg-red-950/50 p-3 mb-4 text-red-300 text-sm">
            <div className="flex items-center gap-2">
              <AlertTriangle className="h-4 w-4" />
              {(selectDriverMutation.error as Error).message}
            </div>
          </div>
        )}

        {driverOptionsQuery.isLoading ? (
          <div className="flex items-center justify-center py-8 text-slate-500">
            <Loader2 className="h-5 w-5 animate-spin mr-2" />
            Loading driver options...
          </div>
        ) : driverOptionsQuery.error ? (
          <div className="rounded-lg border border-red-800 bg-red-950/50 p-4 text-red-300 text-sm">
            <AlertTriangle className="h-4 w-4 inline mr-2" />
            Failed to load driver options
          </div>
        ) : data?.options ? (
          <div className="space-y-3">
            {data.options.map((option) => (
              <DriverOptionCard
                key={option.id}
                option={option}
                isCurrentDriver={option.id === data.currentDriver}
                onSelect={handleSelectDriver}
                isSelecting={selectDriverMutation.isPending}
              />
            ))}
          </div>
        ) : null}
      </section>
    </div>
  );
}

// --- Execution Tab ---

function ExecutionTab() {
  const configQuery = useExecutionConfig();
  const updateMutation = useUpdateExecutionConfig();
  const profilesQuery = useProfiles();
  const [localConfig, setLocalConfig] = useState<ExecutionConfig | null>(null);
  const [hasChanges, setHasChanges] = useState(false);

  // Initialize local config when data loads
  useEffect(() => {
    if (configQuery.data && !localConfig) {
      setLocalConfig(configQuery.data);
    }
  }, [configQuery.data, localConfig]);

  const updateLimits = (
    section: "defaultResourceLimits" | "maxResourceLimits",
    field: keyof ResourceLimitsConfig,
    value: number
  ) => {
    if (!localConfig) return;
    setLocalConfig({
      ...localConfig,
      [section]: {
        ...localConfig[section],
        [field]: value,
      },
    });
    setHasChanges(true);
  };

  const updateDefaultProfile = (profileId: string) => {
    if (!localConfig) return;
    setLocalConfig({
      ...localConfig,
      defaultIsolationProfile: profileId,
    });
    setHasChanges(true);
  };

  const handleSave = () => {
    if (!localConfig) return;
    updateMutation.mutate(localConfig, {
      onSuccess: () => {
        setHasChanges(false);
      },
    });
  };

  // Check error first to avoid showing loading forever on API failure
  if (configQuery.error) {
    return (
      <div className="rounded-lg border border-red-800 bg-red-950/50 p-4 text-red-300 text-sm">
        <AlertTriangle className="h-4 w-4 inline mr-2" />
        Failed to load execution configuration: {(configQuery.error as Error).message || "Unknown error"}
      </div>
    );
  }

  // Check if localConfig has the required nested structure
  const isConfigValid = localConfig &&
    localConfig.defaultResourceLimits &&
    localConfig.maxResourceLimits;

  if (configQuery.isLoading || !isConfigValid) {
    return (
      <div className="flex items-center justify-center py-8 text-slate-500">
        <Loader2 className="h-5 w-5 animate-spin mr-2" />
        Loading configuration...
      </div>
    );
  }

  // Safe access helper - localConfig is guaranteed valid here
  const defaults = localConfig.defaultResourceLimits;
  const maxes = localConfig.maxResourceLimits;

  return (
    <div className="space-y-6">
      {/* Default Resource Limits */}
      <section>
        <h3 className="text-sm font-medium text-slate-300 mb-2">
          Default Resource Limits
        </h3>
        <p className="text-xs text-slate-500 mb-4">
          Applied when no limits are specified per-request. Zero means unlimited.
        </p>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          <LimitInput
            label="Memory"
            icon={MemoryStick}
            value={defaults.memoryLimitMB}
            onChange={(v) => updateLimits("defaultResourceLimits", "memoryLimitMB", v)}
            unit="MB"
          />
          <LimitInput
            label="CPU Time"
            icon={Cpu}
            value={defaults.cpuTimeSec}
            onChange={(v) => updateLimits("defaultResourceLimits", "cpuTimeSec", v)}
            unit="sec"
          />
          <LimitInput
            label="Timeout"
            icon={Clock}
            value={defaults.timeoutSec}
            onChange={(v) => updateLimits("defaultResourceLimits", "timeoutSec", v)}
            unit="sec"
          />
          <LimitInput
            label="Max Processes"
            icon={Hash}
            value={defaults.maxProcesses}
            onChange={(v) => updateLimits("defaultResourceLimits", "maxProcesses", v)}
          />
          <LimitInput
            label="Max Open Files"
            icon={File}
            value={defaults.maxOpenFiles}
            onChange={(v) => updateLimits("defaultResourceLimits", "maxOpenFiles", v)}
          />
        </div>
      </section>

      {/* Maximum Limits */}
      <section>
        <h3 className="text-sm font-medium text-slate-300 mb-2">
          Maximum Limits (Enforcement Ceiling)
        </h3>
        <p className="text-xs text-slate-500 mb-4">
          Users cannot request limits exceeding these values. Zero means no maximum.
        </p>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          <LimitInput
            label="Memory"
            icon={MemoryStick}
            value={maxes.memoryLimitMB}
            onChange={(v) => updateLimits("maxResourceLimits", "memoryLimitMB", v)}
            unit="MB"
          />
          <LimitInput
            label="CPU Time"
            icon={Cpu}
            value={maxes.cpuTimeSec}
            onChange={(v) => updateLimits("maxResourceLimits", "cpuTimeSec", v)}
            unit="sec"
          />
          <LimitInput
            label="Timeout"
            icon={Clock}
            value={maxes.timeoutSec}
            onChange={(v) => updateLimits("maxResourceLimits", "timeoutSec", v)}
            unit="sec"
          />
          <LimitInput
            label="Max Processes"
            icon={Hash}
            value={maxes.maxProcesses}
            onChange={(v) => updateLimits("maxResourceLimits", "maxProcesses", v)}
          />
          <LimitInput
            label="Max Open Files"
            icon={File}
            value={maxes.maxOpenFiles}
            onChange={(v) => updateLimits("maxResourceLimits", "maxOpenFiles", v)}
          />
        </div>
      </section>

      {/* Default Isolation Profile */}
      <section>
        <h3 className="text-sm font-medium text-slate-300 mb-2">
          Default Isolation Profile
        </h3>
        <p className="text-xs text-slate-500 mb-4">
          Profile used when none is specified per-request.
        </p>
        <select
          value={localConfig.defaultIsolationProfile}
          onChange={(e) => updateDefaultProfile(e.target.value)}
          className="w-full h-9 px-3 rounded-md bg-slate-800 border border-slate-700 text-sm text-slate-200 focus:outline-none focus:ring-2 focus:ring-emerald-500"
        >
          {profilesQuery.data?.map((profile) => (
            <option key={profile.id} value={profile.id}>
              {profile.name} {profile.builtin && "(Built-in)"}
            </option>
          ))}
        </select>
      </section>

      {/* Save Button */}
      {hasChanges && (
        <div className="flex justify-end">
          <Button
            onClick={handleSave}
            disabled={updateMutation.isPending}
            className="bg-emerald-600 hover:bg-emerald-500"
          >
            {updateMutation.isPending ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                Saving...
              </>
            ) : (
              <>
                <Save className="h-4 w-4 mr-2" />
                Save Changes
              </>
            )}
          </Button>
        </div>
      )}

      {updateMutation.error && (
        <div className="rounded-lg border border-red-800 bg-red-950/50 p-3 text-red-300 text-sm">
          <AlertTriangle className="h-4 w-4 inline mr-2" />
          {(updateMutation.error as Error).message}
        </div>
      )}

      {updateMutation.isSuccess && !hasChanges && (
        <div className="rounded-lg border border-emerald-700 bg-emerald-950/30 p-3 text-emerald-200 text-sm">
          <CheckCircle2 className="h-4 w-4 inline mr-2" />
          Configuration saved successfully
        </div>
      )}
    </div>
  );
}

// --- Profiles Tab ---

// Key-Value Editor component for editing binds and environment variables
function KeyValueEditor({
  label,
  entries,
  onChange,
  keyPlaceholder,
  valuePlaceholder,
  keyLabel,
  valueLabel,
}: {
  label: string;
  entries: Record<string, string>;
  onChange: (entries: Record<string, string>) => void;
  keyPlaceholder?: string;
  valuePlaceholder?: string;
  keyLabel?: string;
  valueLabel?: string;
}) {
  const [newKey, setNewKey] = useState("");
  const [newValue, setNewValue] = useState("");

  const handleAdd = () => {
    if (!newKey.trim()) return;
    onChange({ ...entries, [newKey.trim()]: newValue });
    setNewKey("");
    setNewValue("");
  };

  const handleRemove = (key: string) => {
    const updated = { ...entries };
    delete updated[key];
    onChange(updated);
  };

  const handleUpdate = (oldKey: string, newKeyName: string, value: string) => {
    const updated = { ...entries };
    if (oldKey !== newKeyName) {
      delete updated[oldKey];
    }
    updated[newKeyName] = value;
    onChange(updated);
  };

  return (
    <div className="space-y-2">
      <label className="text-xs text-slate-400 block">{label}</label>
      <div className="space-y-1.5">
        {Object.entries(entries).map(([key, value]) => (
          <div key={key} className="flex gap-2 items-center">
            <Input
              value={key}
              onChange={(e) => handleUpdate(key, e.target.value, value)}
              className="h-7 text-xs font-mono flex-1"
              placeholder={keyPlaceholder}
            />
            <span className="text-slate-600 text-xs">:</span>
            <Input
              value={value}
              onChange={(e) => handleUpdate(key, key, e.target.value)}
              className="h-7 text-xs font-mono flex-1"
              placeholder={valuePlaceholder}
            />
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="h-7 w-7 p-0 text-red-400 hover:text-red-300 hover:bg-red-950/30"
              onClick={() => handleRemove(key)}
            >
              <XIcon className="h-3.5 w-3.5" />
            </Button>
          </div>
        ))}
        <div className="flex gap-2 items-center mt-2 pt-2 border-t border-slate-800">
          <Input
            value={newKey}
            onChange={(e) => setNewKey(e.target.value)}
            className="h-7 text-xs font-mono flex-1"
            placeholder={keyPlaceholder || (keyLabel ? `New ${keyLabel.toLowerCase()}` : "Key")}
          />
          <span className="text-slate-600 text-xs">:</span>
          <Input
            value={newValue}
            onChange={(e) => setNewValue(e.target.value)}
            className="h-7 text-xs font-mono flex-1"
            placeholder={valuePlaceholder || (valueLabel ? `New ${valueLabel.toLowerCase()}` : "Value")}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                handleAdd();
              }
            }}
          />
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="h-7 px-2"
            onClick={handleAdd}
            disabled={!newKey.trim()}
          >
            <Plus className="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>
    </div>
  );
}

// Profile Editor component for creating/editing profiles
function ProfileEditor({
  profile,
  onSave,
  onCancel,
  isSaving,
  error,
  isNew,
}: {
  profile: Partial<IsolationProfile>;
  onSave: (profile: IsolationProfile) => void;
  onCancel: () => void;
  isSaving: boolean;
  error?: Error | null;
  isNew: boolean;
}) {
  const [localProfile, setLocalProfile] = useState<Partial<IsolationProfile>>({
    name: profile.name || "",
    description: profile.description || "",
    networkAccess: profile.networkAccess || "none",
    readOnlyBinds: profile.readOnlyBinds || {},
    readWriteBinds: profile.readWriteBinds || {},
    environment: profile.environment || {},
    hostname: profile.hostname || "sandbox",
  });

  const handleSave = () => {
    if (!localProfile.name) return;

    const fullProfile: IsolationProfile = {
      id: isNew
        ? localProfile.name.toLowerCase().replace(/[^a-z0-9-]/g, "-")
        : profile.id || localProfile.name.toLowerCase().replace(/[^a-z0-9-]/g, "-"),
      name: localProfile.name,
      description: localProfile.description || "",
      builtin: false,
      networkAccess: localProfile.networkAccess || "none",
      readOnlyBinds: localProfile.readOnlyBinds || {},
      readWriteBinds: localProfile.readWriteBinds || {},
      environment: localProfile.environment || {},
      hostname: localProfile.hostname || "sandbox",
    };

    onSave(fullProfile);
  };

  return (
    <div className="rounded-lg border border-blue-700 bg-blue-950/20 p-4 space-y-4">
      <div className="flex items-center justify-between">
        <h4 className="text-sm font-medium text-slate-200">
          {isNew ? "New Custom Profile" : `Edit Profile: ${profile.name}`}
        </h4>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-7 w-7 p-0"
          onClick={onCancel}
        >
          <XIcon className="h-4 w-4" />
        </Button>
      </div>

      <div className="space-y-4">
        {/* Basic Info */}
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="text-xs text-slate-400 block mb-1">Name</label>
            <Input
              value={localProfile.name}
              onChange={(e) => setLocalProfile({ ...localProfile, name: e.target.value })}
              placeholder="my-custom-profile"
              className="h-8"
              disabled={!isNew}
            />
          </div>
          <div>
            <label className="text-xs text-slate-400 block mb-1">Hostname</label>
            <Input
              value={localProfile.hostname}
              onChange={(e) => setLocalProfile({ ...localProfile, hostname: e.target.value })}
              placeholder="sandbox"
              className="h-8"
            />
          </div>
        </div>

        <div>
          <label className="text-xs text-slate-400 block mb-1">Description</label>
          <Input
            value={localProfile.description}
            onChange={(e) => setLocalProfile({ ...localProfile, description: e.target.value })}
            placeholder="What this profile is for..."
            className="h-8"
          />
        </div>

        <div>
          <label className="text-xs text-slate-400 block mb-1">Network Access</label>
          <select
            value={localProfile.networkAccess}
            onChange={(e) => setLocalProfile({ ...localProfile, networkAccess: e.target.value as NetworkAccess })}
            className="w-full h-8 px-2 rounded-md bg-slate-800 border border-slate-700 text-sm text-slate-200"
          >
            <option value="none">None (isolated)</option>
            <option value="localhost">Localhost only</option>
            <option value="full">Full network</option>
          </select>
        </div>

        {/* Read-Only Binds */}
        <KeyValueEditor
          label="Read-Only Binds (host path : sandbox path)"
          entries={localProfile.readOnlyBinds || {}}
          onChange={(entries) => setLocalProfile({ ...localProfile, readOnlyBinds: entries })}
          keyPlaceholder="$HOME/.config"
          valuePlaceholder="/config"
          keyLabel="Host path"
          valueLabel="Sandbox path"
        />

        {/* Read-Write Binds */}
        <KeyValueEditor
          label="Read-Write Binds (host path : sandbox path)"
          entries={localProfile.readWriteBinds || {}}
          onChange={(entries) => setLocalProfile({ ...localProfile, readWriteBinds: entries })}
          keyPlaceholder="/tmp/shared"
          valuePlaceholder="/shared"
          keyLabel="Host path"
          valueLabel="Sandbox path"
        />

        {/* Environment Variables */}
        <KeyValueEditor
          label="Environment Variables"
          entries={localProfile.environment || {}}
          onChange={(entries) => setLocalProfile({ ...localProfile, environment: entries })}
          keyPlaceholder="MY_VAR"
          valuePlaceholder="value"
          keyLabel="Variable name"
          valueLabel="Value"
        />
      </div>

      <div className="flex gap-2 pt-2 border-t border-slate-800">
        <Button
          size="sm"
          onClick={handleSave}
          disabled={!localProfile.name || isSaving}
          className="bg-emerald-600 hover:bg-emerald-500"
        >
          {isSaving ? (
            <Loader2 className="h-4 w-4 mr-1.5 animate-spin" />
          ) : (
            <Save className="h-4 w-4 mr-1.5" />
          )}
          {isNew ? "Create Profile" : "Save Changes"}
        </Button>
        <Button size="sm" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
      </div>

      {error && (
        <div className="text-xs text-red-400 mt-2">
          {error.message}
        </div>
      )}
    </div>
  );
}

function ProfileCard({
  profile,
  onEdit,
  onCopy,
  onDelete,
}: {
  profile: IsolationProfile;
  onEdit: (profile: IsolationProfile) => void;
  onCopy: (profile: IsolationProfile) => void;
  onDelete: (id: string) => void;
}) {
  const [expanded, setExpanded] = useState(false);

  const networkBadgeColor = {
    none: "bg-red-900/50 text-red-300 border-red-700",
    localhost: "bg-amber-900/50 text-amber-300 border-amber-700",
    full: "bg-emerald-900/50 text-emerald-300 border-emerald-700",
  };

  return (
    <div
      className={`rounded-lg border p-4 ${
        profile.builtin
          ? "border-slate-700 bg-slate-800/30"
          : "border-blue-800 bg-blue-950/20"
      }`}
    >
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="font-medium text-slate-200">{profile.name}</span>
            {profile.builtin && (
              <Badge variant="default" className="text-[10px] bg-slate-700">
                Built-in
              </Badge>
            )}
            <Badge
              variant="default"
              className={`text-[10px] ${networkBadgeColor[profile.networkAccess]}`}
            >
              <Wifi className="h-2.5 w-2.5 mr-1" />
              {profile.networkAccess}
            </Badge>
          </div>
          <p className="text-xs text-slate-500 mt-1">{profile.description}</p>
        </div>
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="sm"
            className="h-7 w-7 p-0 text-slate-400 hover:text-emerald-400"
            onClick={() => onCopy(profile)}
            title="Copy profile"
          >
            <Copy className="h-3.5 w-3.5" />
          </Button>
          {!profile.builtin && (
            <Button
              variant="ghost"
              size="sm"
              className="h-7 w-7 p-0 text-slate-400 hover:text-blue-400"
              onClick={() => onEdit(profile)}
              title="Edit profile"
            >
              <Pencil className="h-3.5 w-3.5" />
            </Button>
          )}
          <Button
            variant="ghost"
            size="sm"
            className="h-7 w-7 p-0"
            onClick={() => setExpanded(!expanded)}
          >
            {expanded ? (
              <ChevronDown className="h-4 w-4" />
            ) : (
              <ChevronRight className="h-4 w-4" />
            )}
          </Button>
        </div>
      </div>

      {expanded && (
        <div className="mt-4 pt-4 border-t border-slate-700 space-y-3">
          {/* Hostname */}
          {profile.hostname && (
            <div>
              <div className="text-xs text-slate-500 mb-1">Hostname</div>
              <div className="text-xs font-mono text-slate-400 bg-slate-800/50 px-2 py-1 rounded inline-block">
                {profile.hostname}
              </div>
            </div>
          )}

          {/* Read-only binds */}
          {Object.keys(profile.readOnlyBinds).length > 0 && (
            <div>
              <div className="text-xs text-slate-500 mb-1.5">Read-Only Binds</div>
              <div className="text-xs font-mono text-slate-400 space-y-0.5 bg-slate-800/50 p-2 rounded">
                {Object.entries(profile.readOnlyBinds).map(([src, dst]) => (
                  <div key={src}>
                    {src} <span className="text-slate-600">-&gt;</span> {dst}
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Read-write binds */}
          {Object.keys(profile.readWriteBinds).length > 0 && (
            <div>
              <div className="text-xs text-slate-500 mb-1.5">Read-Write Binds</div>
              <div className="text-xs font-mono text-slate-400 space-y-0.5 bg-slate-800/50 p-2 rounded">
                {Object.entries(profile.readWriteBinds).map(([src, dst]) => (
                  <div key={src}>
                    {src} <span className="text-slate-600">-&gt;</span> {dst}
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Environment */}
          {Object.keys(profile.environment).length > 0 && (
            <div>
              <div className="text-xs text-slate-500 mb-1.5">Environment</div>
              <div className="text-xs font-mono text-slate-400 space-y-0.5 bg-slate-800/50 p-2 rounded">
                {Object.entries(profile.environment).map(([k, v]) => (
                  <div key={k}>
                    {k}=<span className="text-emerald-400">{v}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Action buttons */}
          <div className="pt-2 flex gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => onCopy(profile)}
              className="text-emerald-400 hover:text-emerald-300 hover:bg-emerald-950/30 border-emerald-800"
            >
              <Copy className="h-3.5 w-3.5 mr-1.5" />
              Copy
            </Button>
            {!profile.builtin && (
              <>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => onEdit(profile)}
                  className="text-blue-400 hover:text-blue-300 hover:bg-blue-950/30 border-blue-800"
                >
                  <Pencil className="h-3.5 w-3.5 mr-1.5" />
                  Edit
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => onDelete(profile.id)}
                  className="text-red-400 hover:text-red-300 hover:bg-red-950/30 border-red-800"
                >
                  <Trash2 className="h-3.5 w-3.5 mr-1.5" />
                  Delete
                </Button>
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

function ProfilesTab() {
  const profilesQuery = useProfiles();
  const deleteMutation = useDeleteProfile();
  const saveMutation = useSaveProfile();
  const [editingProfile, setEditingProfile] = useState<IsolationProfile | null>(null);
  const [copyingProfile, setCopyingProfile] = useState<Partial<IsolationProfile> | null>(null);
  const [isCreatingNew, setIsCreatingNew] = useState(false);

  const handleDelete = (id: string) => {
    if (confirm("Are you sure you want to delete this profile?")) {
      deleteMutation.mutate(id);
    }
  };

  const handleCopy = (profile: IsolationProfile) => {
    // Create a copy with modified name
    const copiedProfile: Partial<IsolationProfile> = {
      name: `${profile.name} (Copy)`,
      description: profile.description,
      networkAccess: profile.networkAccess,
      readOnlyBinds: { ...profile.readOnlyBinds },
      readWriteBinds: { ...profile.readWriteBinds },
      environment: { ...profile.environment },
      hostname: profile.hostname,
    };
    setCopyingProfile(copiedProfile);
    setEditingProfile(null);
    setIsCreatingNew(false);
  };

  const handleSave = (profile: IsolationProfile) => {
    saveMutation.mutate(profile, {
      onSuccess: () => {
        setEditingProfile(null);
        setCopyingProfile(null);
        setIsCreatingNew(false);
      },
    });
  };

  const handleEdit = (profile: IsolationProfile) => {
    setEditingProfile(profile);
    setCopyingProfile(null);
    setIsCreatingNew(false);
  };

  const handleCancel = () => {
    setEditingProfile(null);
    setCopyingProfile(null);
    setIsCreatingNew(false);
  };

  const handleStartCreate = () => {
    setIsCreatingNew(true);
    setEditingProfile(null);
    setCopyingProfile(null);
  };

  if (profilesQuery.isLoading) {
    return (
      <div className="flex items-center justify-center py-8 text-slate-500">
        <Loader2 className="h-5 w-5 animate-spin mr-2" />
        Loading profiles...
      </div>
    );
  }

  if (profilesQuery.error) {
    return (
      <div className="rounded-lg border border-red-800 bg-red-950/50 p-4 text-red-300 text-sm">
        <AlertTriangle className="h-4 w-4 inline mr-2" />
        Failed to load isolation profiles
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Info */}
      <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-3">
        <div className="flex items-start gap-2 text-xs text-slate-400">
          <Info className="h-4 w-4 text-blue-400 flex-shrink-0 mt-0.5" />
          <p>
            Isolation profiles define what resources are accessible inside sandboxes.
            Use placeholders like <code className="bg-slate-800 px-1 rounded">$HOME</code>,
            <code className="bg-slate-800 px-1 rounded ml-1">$VROOLI_ROOT</code> in paths.
          </p>
        </div>
      </div>

      {/* Profile List */}
      {profilesQuery.data?.map((profile) => (
        <ProfileCard
          key={profile.id}
          profile={profile}
          onEdit={handleEdit}
          onCopy={handleCopy}
          onDelete={handleDelete}
        />
      ))}

      {/* Profile Editor (for new, copying, or editing) */}
      {(isCreatingNew || copyingProfile || editingProfile) ? (
        <ProfileEditor
          profile={copyingProfile || editingProfile || {}}
          onSave={handleSave}
          onCancel={handleCancel}
          isSaving={saveMutation.isPending}
          error={saveMutation.error as Error | null}
          isNew={isCreatingNew || !!copyingProfile}
        />
      ) : (
        <Button
          variant="outline"
          className="w-full"
          onClick={handleStartCreate}
        >
          <Plus className="h-4 w-4 mr-2" />
          Add Custom Profile
        </Button>
      )}
    </div>
  );
}

// --- Main Settings Dialog ---

export function SettingsDialog({ open, onOpenChange }: SettingsDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-6xl max-h-[85vh] flex flex-col">
        <DialogClose onClose={() => onOpenChange(false)} />

        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Settings className="h-5 w-5 text-slate-400" />
            Settings
          </DialogTitle>
          <DialogDescription>
            Configure sandbox drivers, execution defaults, and isolation profiles.
          </DialogDescription>
        </DialogHeader>

        <Tabs defaultValue="driver" className="flex-1 flex flex-col overflow-hidden mt-4">
          <TabsList>
            <TabsTrigger value="driver">
              <HardDrive className="h-4 w-4 mr-1.5" />
              Driver
            </TabsTrigger>
            <TabsTrigger value="execution">
              <Cpu className="h-4 w-4 mr-1.5" />
              Execution
            </TabsTrigger>
            <TabsTrigger value="profiles">
              <Shield className="h-4 w-4 mr-1.5" />
              Isolation Profiles
            </TabsTrigger>
          </TabsList>

          <div className="flex-1 overflow-y-auto pr-2 mt-2">
            <TabsContent value="driver">
              <DriverTab />
            </TabsContent>

            <TabsContent value="execution">
              <ExecutionTab />
            </TabsContent>

            <TabsContent value="profiles">
              <ProfilesTab />
            </TabsContent>
          </div>
        </Tabs>

        <div className="flex justify-end pt-4 border-t border-slate-800 mt-4">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Close
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
