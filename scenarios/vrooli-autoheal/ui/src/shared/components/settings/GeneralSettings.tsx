import { Input, Switch } from "../../ui/primitives";
import type { Config, DefaultsResponse, GlobalConfig } from "../../../lib/api";

interface GeneralSettingsProps {
  config: Config | null;
  defaults: DefaultsResponse | undefined;
  onChange: (key: keyof GlobalConfig, value: number) => void;
  autoRefresh: boolean;
  onAutoRefreshChange: (enabled: boolean) => void;
}

export function GeneralSettings({
  config,
  defaults,
  onChange,
  autoRefresh,
  onAutoRefreshChange,
}: GeneralSettingsProps) {
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
      <div className="rounded-lg border border-border-default/70 bg-surface-overlay/40 p-4">
        <h3 className="mb-2 text-lg font-medium">Dashboard Refresh</h3>
        <p className="mb-4 text-sm text-text-muted">
          Controls whether dashboard health data refreshes automatically every 30 seconds.
        </p>
        <div className="flex items-center justify-between gap-3 rounded-md border border-border-default/70 bg-surface-elevated/70 px-3 py-2">
          <div className="min-w-0">
            <p className="text-sm font-medium text-text-primary">Auto-refresh</p>
            <p className="text-xs text-text-muted">{autoRefresh ? "Enabled" : "Manual mode"}</p>
          </div>
          <Switch
            checked={autoRefresh}
            onCheckedChange={onAutoRefreshChange}
            aria-label={autoRefresh ? "Disable auto-refresh" : "Enable auto-refresh"}
            size="sm"
          />
        </div>
      </div>

      <div className="rounded-lg border border-border-default/70 bg-surface-overlay/40 p-4">
        <h3 className="mb-4 text-lg font-medium">Global Settings</h3>
        <div className="space-y-4">
          {fields.map((field) => (
            <div key={field.key} className="grid items-start gap-3 sm:grid-cols-3 sm:gap-4">
              <div>
                <label className="text-sm font-medium text-text-primary">{field.label}</label>
                <p className="mt-1 text-xs text-text-muted">{field.description}</p>
              </div>
              <div className="flex flex-wrap items-center gap-2 sm:col-span-2 sm:gap-3">
                <Input
                  type="number"
                  min={field.min}
                  max={field.max}
                  value={config.global[field.key]}
                  onChange={(e) => onChange(field.key, parseInt(e.target.value, 10) || field.min)}
                  className="w-24"
                />
                <span className="text-sm text-text-muted">{field.unit}</span>
                {defaults && (
                  <span className="text-xs text-text-muted/80">(default: {defaults.global[field.key]})</span>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
