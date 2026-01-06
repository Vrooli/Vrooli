// Model Pricing Tab - displays and manages model pricing data with per-component sources

import { useCallback, useState } from "react";
import { Badge } from "../../ui/badge";
import { Button } from "../../ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../ui/card";
import { Input } from "../../ui/input";
import { Label } from "../../ui/label";
import { ScrollArea } from "../../ui/scroll-area";
import {
  useModelPricing,
  usePricingSettings,
  useUpdatePricingSettings,
  usePricingCacheStatus,
  useRefreshAllPricing,
  useRecalculateModelPricing,
  useModelOverrides,
  useSetOverride,
  useDeleteOverride,
} from "../../../hooks/usePricing";
import type {
  ModelPricingListItem,
  PricingComponent,
  PricingSource,
  PricingSettings,
  PriceOverride,
} from "../../../types";

// =============================================================================
// Helper Functions
// =============================================================================

function sourceColor(source: PricingSource): string {
  switch (source) {
    case "manual_override":
      return "bg-purple-500/20 text-purple-300 border-purple-500/30";
    case "provider_api":
      return "bg-green-500/20 text-green-300 border-green-500/30";
    case "historical_average":
      return "bg-blue-500/20 text-blue-300 border-blue-500/30";
    default:
      return "bg-gray-500/20 text-gray-300 border-gray-500/30";
  }
}

function sourceLabel(source: PricingSource): string {
  switch (source) {
    case "manual_override":
      return "Manual";
    case "provider_api":
      return "Provider";
    case "historical_average":
      return "Historical";
    default:
      return "Unknown";
  }
}

function formatPrice(price: number | undefined): string {
  if (price === undefined || price === 0) return "-";
  if (price < 0.01) return `$${price.toFixed(4)}`;
  return `$${price.toFixed(2)}`;
}

function formatTimestamp(ts: string | undefined): string {
  if (!ts) return "-";
  try {
    return new Date(ts).toLocaleString();
  } catch {
    return ts;
  }
}

// =============================================================================
// Sub-Components
// =============================================================================

interface SourceBadgeProps {
  source: PricingSource;
}

function SourceBadge({ source }: SourceBadgeProps) {
  return (
    <Badge variant="outline" className={`text-xs ${sourceColor(source)}`}>
      {sourceLabel(source)}
    </Badge>
  );
}

interface ModelPricingRowProps {
  model: ModelPricingListItem;
  onRecalculate: (model: string) => void;
  onSelectModel: (model: string) => void;
  recalculatingModel: string | null;
}

function ModelPricingRow({ model, onRecalculate, onSelectModel, recalculatingModel }: ModelPricingRowProps) {
  const isRecalculating = recalculatingModel === model.canonicalName;

  return (
    <tr className="border-b border-border/50 hover:bg-muted/30 transition-colors">
      <td className="px-3 py-2">
        <button
          type="button"
          onClick={() => onSelectModel(model.canonicalName || model.model)}
          className="text-left hover:text-primary transition-colors"
        >
          <div className="font-medium text-sm">{model.model}</div>
          {model.canonicalName && model.canonicalName !== model.model && (
            <div className="text-xs text-muted-foreground">{model.canonicalName}</div>
          )}
        </button>
      </td>
      <td className="px-3 py-2 text-right">
        <div className="flex items-center justify-end gap-2">
          <span className="text-sm">{formatPrice(model.inputPricePer1M)}</span>
          <SourceBadge source={model.inputSource} />
        </div>
      </td>
      <td className="px-3 py-2 text-right">
        <div className="flex items-center justify-end gap-2">
          <span className="text-sm">{formatPrice(model.outputPricePer1M)}</span>
          <SourceBadge source={model.outputSource} />
        </div>
      </td>
      <td className="px-3 py-2 text-right">
        {model.cacheReadPricePer1M !== undefined && model.cacheReadSource && (
          <div className="flex items-center justify-end gap-2">
            <span className="text-sm">{formatPrice(model.cacheReadPricePer1M)}</span>
            <SourceBadge source={model.cacheReadSource} />
          </div>
        )}
      </td>
      <td className="px-3 py-2 text-center text-xs text-muted-foreground">
        {formatTimestamp(model.fetchedAt)}
      </td>
      <td className="px-3 py-2 text-right">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => onRecalculate(model.canonicalName || model.model)}
          disabled={isRecalculating}
        >
          {isRecalculating ? "..." : "Refresh"}
        </Button>
      </td>
    </tr>
  );
}

interface SettingsCardProps {
  settings: PricingSettings | null;
  loading: boolean;
  onUpdate: (settings: Partial<PricingSettings>) => Promise<void>;
}

function SettingsCard({ settings, loading, onUpdate }: SettingsCardProps) {
  const [avgDays, setAvgDays] = useState(settings?.historicalAverageDays?.toString() || "7");
  const [ttlSeconds, setTtlSeconds] = useState(settings?.providerCacheTtlSeconds?.toString() || "21600");
  const [saving, setSaving] = useState(false);

  const handleSave = async () => {
    setSaving(true);
    try {
      await onUpdate({
        historicalAverageDays: parseInt(avgDays, 10),
        providerCacheTtlSeconds: parseInt(ttlSeconds, 10),
      });
    } finally {
      setSaving(false);
    }
  };

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm">Settings</CardTitle>
        <CardDescription className="text-xs">
          Configure pricing fallback behavior
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-2">
          <Label htmlFor="avg-days" className="text-xs">Historical Average Period (days)</Label>
          <Input
            id="avg-days"
            type="number"
            min={1}
            max={365}
            value={avgDays}
            onChange={(e) => setAvgDays(e.target.value)}
            className="h-8"
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="ttl" className="text-xs">Provider Cache TTL (seconds)</Label>
          <Input
            id="ttl"
            type="number"
            min={60}
            max={86400}
            value={ttlSeconds}
            onChange={(e) => setTtlSeconds(e.target.value)}
            className="h-8"
          />
        </div>
        <Button size="sm" onClick={handleSave} disabled={loading || saving}>
          {saving ? "Saving..." : "Save Settings"}
        </Button>
      </CardContent>
    </Card>
  );
}

interface CacheStatusCardProps {
  status: ReturnType<typeof usePricingCacheStatus>["data"];
  loading: boolean;
  onRefreshAll: () => Promise<void>;
}

function CacheStatusCard({ status, loading, onRefreshAll }: CacheStatusCardProps) {
  const [refreshing, setRefreshing] = useState(false);

  const handleRefresh = async () => {
    setRefreshing(true);
    try {
      await onRefreshAll();
    } finally {
      setRefreshing(false);
    }
  };

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm">Cache Status</CardTitle>
        <CardDescription className="text-xs">
          Provider pricing cache information
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">
        {status && (
          <>
            <div className="text-sm">
              <span className="text-muted-foreground">Total Models:</span>{" "}
              <span className="font-medium">{status.totalModels}</span>
            </div>
            <div className="text-sm">
              <span className="text-muted-foreground">Expired:</span>{" "}
              <span className={`font-medium ${status.expiredCount > 0 ? "text-yellow-400" : ""}`}>
                {status.expiredCount}
              </span>
            </div>
            {status.providers.map((p) => (
              <div key={p.provider} className="text-xs border-l-2 border-border pl-2">
                <div className="font-medium">{p.provider}</div>
                <div className="text-muted-foreground">
                  {p.modelCount} models | Last fetch: {formatTimestamp(p.lastFetchedAt)}
                </div>
                {p.isStale && <Badge variant="outline" className="text-yellow-400 text-xs mt-1">Stale</Badge>}
              </div>
            ))}
          </>
        )}
        <Button size="sm" variant="outline" onClick={handleRefresh} disabled={loading || refreshing}>
          {refreshing ? "Refreshing..." : "Refresh All Pricing"}
        </Button>
      </CardContent>
    </Card>
  );
}

interface ModelDetailPanelProps {
  model: string;
  onClose: () => void;
}

function ModelDetailPanel({ model, onClose }: ModelDetailPanelProps) {
  const { data: overrides, loading, refetch } = useModelOverrides(model);
  const { setOverride, loading: settingOverride } = useSetOverride();
  const { deleteOverride, loading: deletingOverride } = useDeleteOverride();

  const [newComponent, setNewComponent] = useState<PricingComponent>("input_tokens");
  const [newPrice, setNewPrice] = useState("");

  const handleAddOverride = async () => {
    if (!newPrice) return;
    try {
      await setOverride(model, {
        component: newComponent,
        priceUsd: parseFloat(newPrice),
      });
      setNewPrice("");
      refetch();
    } catch {
      // Error handled by hook
    }
  };

  const handleDeleteOverride = async (component: PricingComponent) => {
    try {
      await deleteOverride(model, component);
      refetch();
    } catch {
      // Error handled by hook
    }
  };

  return (
    <Card className="mt-4">
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm">Manual Overrides: {model}</CardTitle>
          <Button variant="ghost" size="sm" onClick={onClose}>Close</Button>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {loading ? (
          <div className="text-sm text-muted-foreground">Loading...</div>
        ) : (
          <>
            {overrides && overrides.length > 0 ? (
              <div className="space-y-2">
                {overrides.map((o: PriceOverride) => (
                  <div key={o.component} className="flex items-center justify-between text-sm border rounded px-3 py-2">
                    <span>{o.component}</span>
                    <div className="flex items-center gap-2">
                      <span className="font-mono">${o.priceUsd.toFixed(6)}</span>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleDeleteOverride(o.component)}
                        disabled={deletingOverride}
                      >
                        Remove
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-sm text-muted-foreground">No manual overrides configured.</div>
            )}

            <div className="border-t pt-4">
              <Label className="text-xs">Add Override</Label>
              <div className="flex gap-2 mt-2">
                <select
                  value={newComponent}
                  onChange={(e) => setNewComponent(e.target.value as PricingComponent)}
                  className="h-8 rounded border bg-background px-2 text-sm"
                >
                  <option value="input_tokens">Input Tokens</option>
                  <option value="output_tokens">Output Tokens</option>
                  <option value="cache_read">Cache Read</option>
                  <option value="cache_creation">Cache Creation</option>
                  <option value="web_search">Web Search</option>
                  <option value="server_tool_use">Server Tool Use</option>
                </select>
                <Input
                  type="number"
                  step="0.000001"
                  placeholder="Price per token"
                  value={newPrice}
                  onChange={(e) => setNewPrice(e.target.value)}
                  className="h-8 w-32"
                />
                <Button size="sm" onClick={handleAddOverride} disabled={settingOverride || !newPrice}>
                  Add
                </Button>
              </div>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}

// =============================================================================
// Main Component
// =============================================================================

export function ModelPricingTab() {
  const { data: models, loading, error, refetch } = useModelPricing();
  const { data: settings, loading: settingsLoading, refetch: refetchSettings } = usePricingSettings();
  const { updateSettings } = useUpdatePricingSettings();
  const { data: cacheStatus, loading: cacheLoading, refetch: refetchCache } = usePricingCacheStatus();
  const { refreshAll } = useRefreshAllPricing();
  const { recalculate } = useRecalculateModelPricing();

  const [recalculatingModel, setRecalculatingModel] = useState<string | null>(null);
  const [selectedModel, setSelectedModel] = useState<string | null>(null);
  const [searchFilter, setSearchFilter] = useState("");

  const handleRecalculate = useCallback(async (model: string) => {
    setRecalculatingModel(model);
    try {
      await recalculate(model);
      refetch();
    } finally {
      setRecalculatingModel(null);
    }
  }, [recalculate, refetch]);

  const handleRefreshAll = useCallback(async () => {
    await refreshAll();
    refetch();
    refetchCache();
  }, [refreshAll, refetch, refetchCache]);

  const handleUpdateSettings = useCallback(async (newSettings: Partial<PricingSettings>) => {
    await updateSettings(newSettings as Required<typeof newSettings>);
    refetchSettings();
  }, [updateSettings, refetchSettings]);

  const filteredModels = models?.filter((m) => {
    const search = searchFilter.toLowerCase();
    return (
      m.model.toLowerCase().includes(search) ||
      (m.canonicalName?.toLowerCase().includes(search) ?? false)
    );
  }) ?? [];

  if (loading) {
    return <div className="text-center py-8 text-muted-foreground">Loading pricing data...</div>;
  }

  if (error) {
    return (
      <div className="text-center py-8 text-red-400">
        Error loading pricing: {error}
        <Button variant="outline" size="sm" onClick={refetch} className="ml-2">
          Retry
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Header with search and legend */}
      <div className="flex items-center justify-between gap-4">
        <div className="flex-1">
          <Input
            placeholder="Search models..."
            value={searchFilter}
            onChange={(e) => setSearchFilter(e.target.value)}
            className="h-8 max-w-sm"
          />
        </div>
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <span>Sources:</span>
          <Badge variant="outline" className={sourceColor("provider_api")}>Provider</Badge>
          <Badge variant="outline" className={sourceColor("manual_override")}>Manual</Badge>
          <Badge variant="outline" className={sourceColor("historical_average")}>Historical</Badge>
        </div>
      </div>

      {/* Main content grid */}
      <div className="grid grid-cols-[1fr_280px] gap-4">
        {/* Model pricing table */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm">Model Pricing</CardTitle>
            <CardDescription className="text-xs">
              Prices per 1M tokens. Click a model to manage overrides.
            </CardDescription>
          </CardHeader>
          <CardContent className="p-0">
            <ScrollArea className="h-[400px]">
              <table className="w-full text-left">
                <thead className="sticky top-0 bg-card border-b">
                  <tr className="text-xs text-muted-foreground">
                    <th className="px-3 py-2 font-medium">Model</th>
                    <th className="px-3 py-2 font-medium text-right">Input</th>
                    <th className="px-3 py-2 font-medium text-right">Output</th>
                    <th className="px-3 py-2 font-medium text-right">Cache Read</th>
                    <th className="px-3 py-2 font-medium text-center">Fetched</th>
                    <th className="px-3 py-2 font-medium text-right">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredModels.map((model) => (
                    <ModelPricingRow
                      key={model.canonicalName || model.model}
                      model={model}
                      onRecalculate={handleRecalculate}
                      onSelectModel={setSelectedModel}
                      recalculatingModel={recalculatingModel}
                    />
                  ))}
                  {filteredModels.length === 0 && (
                    <tr>
                      <td colSpan={6} className="px-3 py-8 text-center text-muted-foreground">
                        {searchFilter ? "No models match your search." : "No pricing data available."}
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </ScrollArea>
          </CardContent>
        </Card>

        {/* Side panel with settings and cache status */}
        <div className="space-y-4">
          <SettingsCard
            settings={settings}
            loading={settingsLoading}
            onUpdate={handleUpdateSettings}
          />
          <CacheStatusCard
            status={cacheStatus}
            loading={cacheLoading}
            onRefreshAll={handleRefreshAll}
          />
        </div>
      </div>

      {/* Model detail panel (shown when a model is selected) */}
      {selectedModel && (
        <ModelDetailPanel
          model={selectedModel}
          onClose={() => setSelectedModel(null)}
        />
      )}
    </div>
  );
}
