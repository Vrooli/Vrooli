import { ChevronDown, ChevronRight } from "lucide-react";
import { Button, Switch } from "../../ui/primitives";
import type { CategoryIcon, CheckWithConfig } from "./types";

interface ChecksSettingsProps {
  checksByCategory: Record<string, CheckWithConfig[]>;
  expandedCategories: Record<string, boolean>;
  toggleCategory: (category: string) => void;
  categoryLabels: Record<string, string>;
  categoryIcons: Record<string, CategoryIcon>;
  onToggleEnabled: (checkId: string, enabled: boolean) => void;
  onToggleAutoHeal: (checkId: string, autoHeal: boolean) => void;
  onBulkUpdate: (action: "enableAll" | "disableAll" | "autoHealAll" | "disableAutoHealAll") => void;
  isUpdating: boolean;
}

export function ChecksSettings({
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
      <div className="mb-4 flex flex-wrap gap-2">
        <Button variant="outline" size="sm" onClick={() => onBulkUpdate("enableAll")} disabled={isUpdating}>
          Enable All
        </Button>
        <Button variant="outline" size="sm" onClick={() => onBulkUpdate("disableAll")} disabled={isUpdating}>
          Disable All
        </Button>
        <Button variant="outline" size="sm" onClick={() => onBulkUpdate("autoHealAll")} disabled={isUpdating}>
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

      {Object.entries(checksByCategory).map(([category, checks]) => {
        if (checks.length === 0) return null;
        const Icon = categoryIcons[category];
        const isExpanded = expandedCategories[category];

        return (
          <div key={category} className="overflow-hidden rounded-lg border border-border-default/70">
            <button
              onClick={() => toggleCategory(category)}
              className="flex w-full items-center justify-between bg-surface-overlay/40 px-4 py-3 transition-colors hover:bg-surface-overlay/70"
            >
              <div className="flex items-center gap-3">
                {Icon ? <Icon size={18} className="text-accent-primary" /> : null}
                <span className="font-medium">{categoryLabels[category]}</span>
                <span className="text-xs text-text-muted">({checks.length} checks)</span>
              </div>
              {isExpanded ? <ChevronDown size={18} /> : <ChevronRight size={18} />}
            </button>

            {isExpanded && (
              <div className="divide-y divide-border-default/30">
                {checks.map((check) => (
                  <div
                    key={check.id}
                    className="flex items-center justify-between px-4 py-3 hover:bg-surface-overlay/30"
                  >
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <span className="text-sm font-medium">{check.title}</span>
                        <span className="text-xs text-text-muted">({check.intervalSeconds}s)</span>
                      </div>
                      <p className="truncate text-xs text-text-muted">{check.description}</p>
                    </div>

                    <div className="ml-4 flex items-center gap-4">
                      <label className="cursor-pointer items-center gap-2">
                        <span className="text-xs text-text-muted">Enabled</span>
                        <Switch
                          checked={check.config.enabled}
                          onCheckedChange={(checked) => onToggleEnabled(check.id, checked)}
                          disabled={isUpdating}
                          size="sm"
                          tone="success"
                          className="ml-2"
                        />
                      </label>

                      <label className="cursor-pointer items-center gap-2">
                        <span className="text-xs text-text-muted">Auto-Heal</span>
                        <Switch
                          checked={check.config.autoHeal && check.config.enabled}
                          onCheckedChange={(checked) => onToggleAutoHeal(check.id, checked)}
                          disabled={isUpdating || !check.config.enabled}
                          size="sm"
                          tone="primary"
                          className="ml-2"
                        />
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
