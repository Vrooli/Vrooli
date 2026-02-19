import { useState } from "react";
import { AlertCircle, AlertTriangle, Database, Loader2, Monitor, Plus, Trash2, X, Zap } from "lucide-react";
import { Badge, Button, Input, Switch } from "../../ui/primitives";
import { Notice, NoticeTitle } from "../../ui/composites";
import type { MonitoringConfig } from "../../../lib/api";

interface MonitoringSettingsProps {
  monitoring: MonitoringConfig | undefined;
  isLoading: boolean;
  onAddScenario: (name: string, critical: boolean) => void;
  onRemoveScenario: (name: string) => void;
  onSetCritical: (name: string, critical: boolean) => void;
  onAddResource: (name: string) => void;
  onRemoveResource: (name: string) => void;
  isUpdating: boolean;
}

export function MonitoringSettings({
  monitoring,
  isLoading,
  onAddScenario,
  onRemoveScenario,
  onSetCritical,
  onAddResource,
  onRemoveResource,
  isUpdating,
}: MonitoringSettingsProps) {
  const [newScenario, setNewScenario] = useState("");
  const [newScenarioCritical, setNewScenarioCritical] = useState(false);
  const [newResource, setNewResource] = useState("");

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="animate-spin text-accent-primary" size={32} />
      </div>
    );
  }

  const handleAddScenario = () => {
    if (!newScenario.trim()) return;
    onAddScenario(newScenario.trim(), newScenarioCritical);
    setNewScenario("");
    setNewScenarioCritical(false);
  };

  const handleAddResource = () => {
    if (!newResource.trim()) return;
    onAddResource(newResource.trim());
    setNewResource("");
  };

  const scenarios = monitoring?.scenarios ? Object.entries(monitoring.scenarios) : [];
  const resources = monitoring?.resources || [];
  const sortedResources = [...resources].sort((a, b) => a.localeCompare(b));

  return (
    <div className="min-w-0 space-y-6">
      <Notice tone="info" className="p-4">
        <div className="flex items-start gap-3">
          <Monitor className="mt-0.5 shrink-0 text-accent-primary" size={20} />
          <div>
            <NoticeTitle tone="info">Monitoring Configuration</NoticeTitle>
            <p className="mt-1 text-sm text-text-muted">
              Configure which scenarios and resources should be monitored. Critical scenarios will show as
              errors when unhealthy, while non-critical scenarios show as warnings.
            </p>
          </div>
        </div>
      </Notice>

      <div className="rounded-lg border border-border-default/70 bg-surface-overlay/40 p-4">
        <div className="mb-4 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Zap className="text-accent-warning" size={18} />
            <h3 className="text-lg font-medium">Monitored Scenarios</h3>
            <span className="text-xs text-text-muted">({scenarios.length})</span>
          </div>
        </div>

        <div className="mb-4 flex flex-col gap-2 sm:flex-row">
          <Input
            type="text"
            value={newScenario}
            onChange={(e) => setNewScenario(e.target.value)}
            placeholder="Scenario name (e.g., app-monitor)"
            className="flex-1"
            onKeyDown={(e) => e.key === "Enter" && handleAddScenario()}
          />
          <label className="flex items-center gap-2 rounded-lg border border-border-default/70 bg-surface-overlay/50 px-3 py-2 sm:w-fit">
            <input
              type="checkbox"
              checked={newScenarioCritical}
              onChange={(e) => setNewScenarioCritical(e.target.checked)}
              className="h-4 w-4 rounded border-border-default bg-surface-base text-accent-primary focus:ring-accent-primary"
            />
            <span className="text-sm text-text-muted">Critical</span>
          </label>
          <Button onClick={handleAddScenario} disabled={!newScenario.trim() || isUpdating} size="sm">
            <Plus size={16} className="mr-1" />
            Add
          </Button>
        </div>

        <div className="space-y-2">
          {scenarios.length === 0 ? (
            <div className="py-4 text-center text-sm text-text-muted">No scenarios configured for monitoring</div>
          ) : (
            scenarios
              .sort((a, b) => {
                if (a[1].critical !== b[1].critical) return b[1].critical ? 1 : -1;
                return a[0].localeCompare(b[0]);
              })
              .map(([name, config]) => (
                <div
                  key={name}
                  className="flex min-w-0 flex-wrap items-center justify-between gap-2 rounded-lg bg-surface-overlay/50 px-3 py-2 hover:bg-surface-overlay/70"
                >
                  <div className="flex min-w-0 items-center gap-2 sm:gap-3">
                    {config.critical ? (
                      <AlertCircle className="text-accent-danger" size={16} />
                    ) : (
                      <AlertTriangle className="text-accent-warning" size={16} />
                    )}
                    <span className="truncate text-sm font-medium">{name}</span>
                    <Badge tone={config.critical ? "danger" : "warning"} size="sm" className="shrink-0">
                      {config.critical ? "Critical" : "Non-Critical"}
                    </Badge>
                  </div>
                  <div className="flex shrink-0 items-center gap-2">
                    <Switch
                      checked={config.critical}
                      onCheckedChange={(checked) => onSetCritical(name, checked)}
                      disabled={isUpdating}
                      title={config.critical ? "Make non-critical" : "Make critical"}
                      size="sm"
                      tone="danger"
                    />
                    <button
                      onClick={() => onRemoveScenario(name)}
                      disabled={isUpdating}
                      className="rounded-lg p-1.5 text-text-muted transition-colors hover:bg-accent-danger/10 hover:text-accent-danger"
                      title="Remove scenario"
                    >
                      <Trash2 size={16} />
                    </button>
                  </div>
                </div>
              ))
          )}
        </div>
      </div>

      <div className="rounded-lg border border-border-default/70 bg-surface-overlay/40 p-4">
        <div className="mb-4 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Database className="text-accent-success" size={18} />
            <h3 className="text-lg font-medium">Monitored Resources</h3>
            <span className="text-xs text-text-muted">({resources.length})</span>
          </div>
        </div>

        <div className="mb-4 flex flex-col gap-2 sm:flex-row">
          <Input
            type="text"
            value={newResource}
            onChange={(e) => setNewResource(e.target.value)}
            placeholder="Resource name (e.g., postgres, redis)"
            className="flex-1"
            onKeyDown={(e) => e.key === "Enter" && handleAddResource()}
          />
          <Button onClick={handleAddResource} disabled={!newResource.trim() || isUpdating} size="sm">
            <Plus size={16} className="mr-1" />
            Add
          </Button>
        </div>

        <div className="flex flex-wrap gap-2">
          {resources.length === 0 ? (
            <div className="w-full py-4 text-center text-sm text-text-muted">No resources configured for monitoring</div>
          ) : (
            sortedResources.map((resource) => (
              <div
                key={resource}
                className="flex min-w-0 items-center gap-2 rounded-lg border border-accent-success/20 bg-accent-success/10 px-3 py-1.5"
              >
                <Database className="text-accent-success" size={14} />
                <span className="break-all text-sm font-medium">{resource}</span>
                <button
                  onClick={() => onRemoveResource(resource)}
                  disabled={isUpdating}
                  className="rounded p-0.5 text-text-muted transition-colors hover:text-accent-danger"
                  title="Remove resource"
                >
                  <X size={14} />
                </button>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
}
