// Model Pricing Tab - displays and manages model pricing data with per-component sources

import { useCallback, useMemo, useState } from "react";
import { ArrowDown, ArrowUp, ArrowUpDown, Pencil, RefreshCw, X } from "lucide-react";
import { Badge } from "../../ui/badge";
import { Button } from "../../ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../ui/card";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogBody,
} from "../../ui/dialog";
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
// Types
// =============================================================================

type SortField = "model" | "input" | "output" | "cacheRead" | "fetched";
type SortDirection = "asc" | "desc";

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
  onClick?: () => void;
  isActive?: boolean;
}

function SourceBadge({ source, onClick, isActive }: SourceBadgeProps) {
  const baseClasses = `text-xs ${sourceColor(source)}`;
  const activeClasses = isActive ? "ring-2 ring-offset-1 ring-offset-background" : "";
  const clickableClasses = onClick ? "cursor-pointer hover:opacity-80 transition-opacity" : "";

  return (
    <Badge
      variant="outline"
      className={`${baseClasses} ${activeClasses} ${clickableClasses}`}
      onClick={onClick}
    >
      {sourceLabel(source)}
    </Badge>
  );
}

interface SortableHeaderProps {
  label: string;
  field: SortField;
  currentSort: SortField;
  direction: SortDirection;
  onSort: (field: SortField) => void;
  align?: "left" | "center" | "right";
}

function SortableHeader({ label, field, currentSort, direction, onSort, align = "left" }: SortableHeaderProps) {
  const isActive = currentSort === field;
  const alignClass = align === "right" ? "justify-end" : align === "center" ? "justify-center" : "justify-start";

  return (
    <th className={`px-3 py-2 font-medium text-${align}`}>
      <button
        type="button"
        onClick={() => onSort(field)}
        className={`flex items-center gap-1 ${alignClass} w-full hover:text-foreground transition-colors`}
      >
        <span>{label}</span>
        {isActive ? (
          direction === "asc" ? <ArrowUp className="h-3 w-3" /> : <ArrowDown className="h-3 w-3" />
        ) : (
          <ArrowUpDown className="h-3 w-3 opacity-50" />
        )}
      </button>
    </th>
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
      <td className="px-3 py-2">
        <div className="flex items-center justify-end gap-1">
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            onClick={() => onSelectModel(model.canonicalName || model.model)}
            title="Edit pricing overrides"
          >
            <Pencil className="h-3.5 w-3.5" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            onClick={() => onRecalculate(model.canonicalName || model.model)}
            disabled={isRecalculating}
            title="Refresh pricing"
          >
            <RefreshCw className={`h-3.5 w-3.5 ${isRecalculating ? "animate-spin" : ""}`} />
          </Button>
        </div>
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

// All pricing components we support
const ALL_COMPONENTS: { key: PricingComponent; label: string }[] = [
  { key: "input_tokens", label: "Input Tokens" },
  { key: "output_tokens", label: "Output Tokens" },
  { key: "cache_read", label: "Cache Read" },
  { key: "cache_creation", label: "Cache Creation" },
  { key: "web_search", label: "Web Search" },
  { key: "server_tool_use", label: "Server Tool Use" },
];

// Helper to get effective price and source from model pricing for a component
function getComponentPricing(
  modelPricing: ModelPricingListItem | undefined,
  component: PricingComponent
): { price: number | undefined; source: PricingSource } {
  if (!modelPricing) {
    return { price: undefined, source: "unknown" };
  }
  switch (component) {
    case "input_tokens":
      return { price: modelPricing.inputPricePer1M, source: modelPricing.inputSource };
    case "output_tokens":
      return { price: modelPricing.outputPricePer1M, source: modelPricing.outputSource };
    case "cache_read":
      return { price: modelPricing.cacheReadPricePer1M, source: modelPricing.cacheReadSource || "unknown" };
    case "cache_creation":
      return { price: modelPricing.cacheCreatePricePer1M, source: modelPricing.cacheCreateSource || "unknown" };
    default:
      return { price: undefined, source: "unknown" };
  }
}

interface EditPricingDialogProps {
  model: string | null;
  modelPricing: ModelPricingListItem | undefined;
  onClose: () => void;
  onPricingUpdated: () => void;
}

function EditPricingDialog({ model, modelPricing, onClose, onPricingUpdated }: EditPricingDialogProps) {
  const { data: overrides, loading, refetch } = useModelOverrides(model);
  const { setOverride, loading: settingOverride } = useSetOverride();
  const { deleteOverride, loading: deletingOverride } = useDeleteOverride();

  // Track which component is being edited and its pending value
  const [editingComponent, setEditingComponent] = useState<PricingComponent | null>(null);
  const [pendingPrice, setPendingPrice] = useState("");

  // Build a map of overrides for quick lookup
  const overrideMap = useMemo(() => {
    const map = new Map<PricingComponent, PriceOverride>();
    if (overrides) {
      for (const o of overrides) {
        map.set(o.component, o);
      }
    }
    return map;
  }, [overrides]);

  const handleStartEdit = (component: PricingComponent) => {
    const existing = overrideMap.get(component);
    // Convert per-token price to per-1M for display
    setPendingPrice(existing ? (existing.priceUsd * 1_000_000).toString() : "");
    setEditingComponent(component);
  };

  const handleCancelEdit = () => {
    setEditingComponent(null);
    setPendingPrice("");
  };

  const handleSaveOverride = async (component: PricingComponent) => {
    if (!model || !pendingPrice) return;
    try {
      // Convert per-1M price back to per-token
      const pricePerToken = parseFloat(pendingPrice) / 1_000_000;
      await setOverride(model, {
        component,
        priceUsd: pricePerToken,
      });
      setEditingComponent(null);
      setPendingPrice("");
      refetch();
      onPricingUpdated();
    } catch {
      // Error handled by hook
    }
  };

  const handleDeleteOverride = async (component: PricingComponent) => {
    if (!model) return;
    try {
      await deleteOverride(model, component);
      refetch();
      onPricingUpdated();
    } catch {
      // Error handled by hook
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent, component: PricingComponent) => {
    if (e.key === "Enter") {
      handleSaveOverride(component);
    } else if (e.key === "Escape") {
      handleCancelEdit();
    }
  };

  return (
    <Dialog open={!!model} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-2xl">
        <DialogHeader onClose={onClose}>
          <DialogTitle>Edit Pricing: {model}</DialogTitle>
          <DialogDescription>
            View current prices and set manual overrides. Prices shown per 1M tokens.
          </DialogDescription>
        </DialogHeader>
        <DialogBody>
          {loading ? (
            <div className="text-sm text-muted-foreground py-4">Loading...</div>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b text-muted-foreground">
                  <th className="text-left py-2 font-medium">Component</th>
                  <th className="text-right py-2 font-medium">Effective Price</th>
                  <th className="text-center py-2 font-medium">Source</th>
                  <th className="text-right py-2 font-medium">Manual Override</th>
                  <th className="text-right py-2 font-medium w-20">Actions</th>
                </tr>
              </thead>
              <tbody>
                {ALL_COMPONENTS.map(({ key, label }) => {
                  const { price, source } = getComponentPricing(modelPricing, key);
                  const override = overrideMap.get(key);
                  const isEditing = editingComponent === key;
                  const hasOverride = !!override;

                  return (
                    <tr key={key} className="border-b border-border/50 hover:bg-muted/30">
                      <td className="py-2 font-medium">{label}</td>
                      <td className="py-2 text-right font-mono">
                        {price !== undefined && price > 0 ? formatPrice(price) : "-"}
                      </td>
                      <td className="py-2 text-center">
                        {price !== undefined && price > 0 ? (
                          <SourceBadge source={source} />
                        ) : (
                          <span className="text-muted-foreground">-</span>
                        )}
                      </td>
                      <td className="py-2 text-right">
                        {isEditing ? (
                          <div className="flex items-center justify-end gap-1">
                            <span className="text-muted-foreground text-xs">$</span>
                            <Input
                              type="number"
                              step="0.01"
                              placeholder="0.00"
                              value={pendingPrice}
                              onChange={(e) => setPendingPrice(e.target.value)}
                              onKeyDown={(e) => handleKeyDown(e, key)}
                              className="h-7 w-24 text-right font-mono"
                              autoFocus
                            />
                          </div>
                        ) : hasOverride ? (
                          <span className="font-mono text-purple-300">
                            {formatPrice(override.priceUsd * 1_000_000)}
                          </span>
                        ) : (
                          <span className="text-muted-foreground">-</span>
                        )}
                      </td>
                      <td className="py-2 text-right">
                        <div className="flex items-center justify-end gap-1">
                          {isEditing ? (
                            <>
                              <Button
                                variant="ghost"
                                size="icon"
                                className="h-7 w-7"
                                onClick={() => handleSaveOverride(key)}
                                disabled={settingOverride || !pendingPrice}
                                title="Save override"
                              >
                                <span className="text-green-400 text-xs font-bold">✓</span>
                              </Button>
                              <Button
                                variant="ghost"
                                size="icon"
                                className="h-7 w-7"
                                onClick={handleCancelEdit}
                                title="Cancel"
                              >
                                <X className="h-3.5 w-3.5" />
                              </Button>
                            </>
                          ) : (
                            <>
                              <Button
                                variant="ghost"
                                size="icon"
                                className="h-7 w-7"
                                onClick={() => handleStartEdit(key)}
                                title={hasOverride ? "Edit override" : "Set override"}
                              >
                                <Pencil className="h-3.5 w-3.5" />
                              </Button>
                              {hasOverride && (
                                <Button
                                  variant="ghost"
                                  size="icon"
                                  className="h-7 w-7 text-red-400 hover:text-red-300"
                                  onClick={() => handleDeleteOverride(key)}
                                  disabled={deletingOverride}
                                  title="Clear override"
                                >
                                  <X className="h-3.5 w-3.5" />
                                </Button>
                              )}
                            </>
                          )}
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          )}
          <div className="mt-4 text-xs text-muted-foreground">
            <p><strong>Manual Override</strong> takes priority over provider and historical pricing.</p>
            <p className="mt-1">Click the pencil icon to set an override, or X to clear it.</p>
          </div>
        </DialogBody>
      </DialogContent>
    </Dialog>
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
  const [sourceFilter, setSourceFilter] = useState<Set<PricingSource>>(new Set());
  const [sortField, setSortField] = useState<SortField>("model");
  const [sortDirection, setSortDirection] = useState<SortDirection>("asc");

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

  const toggleSourceFilter = useCallback((source: PricingSource) => {
    setSourceFilter((prev) => {
      const next = new Set(prev);
      if (next.has(source)) {
        next.delete(source);
      } else {
        next.add(source);
      }
      return next;
    });
  }, []);

  const handleSort = useCallback((field: SortField) => {
    setSortField((prevField) => {
      if (prevField === field) {
        setSortDirection((d) => (d === "asc" ? "desc" : "asc"));
        return field;
      }
      setSortDirection("asc");
      return field;
    });
  }, []);

  const filteredAndSortedModels = useMemo(() => {
    let result = models ?? [];

    // Apply search filter
    if (searchFilter) {
      const search = searchFilter.toLowerCase();
      result = result.filter(
        (m) =>
          m.model.toLowerCase().includes(search) ||
          (m.canonicalName?.toLowerCase().includes(search) ?? false)
      );
    }

    // Apply source filter (model must have at least one matching source)
    if (sourceFilter.size > 0) {
      result = result.filter((m) =>
        sourceFilter.has(m.inputSource) ||
        sourceFilter.has(m.outputSource) ||
        (m.cacheReadSource && sourceFilter.has(m.cacheReadSource))
      );
    }

    // Apply sorting
    result = [...result].sort((a, b) => {
      let cmp = 0;
      switch (sortField) {
        case "model":
          cmp = (a.model || "").localeCompare(b.model || "");
          break;
        case "input":
          cmp = (a.inputPricePer1M ?? 0) - (b.inputPricePer1M ?? 0);
          break;
        case "output":
          cmp = (a.outputPricePer1M ?? 0) - (b.outputPricePer1M ?? 0);
          break;
        case "cacheRead":
          cmp = (a.cacheReadPricePer1M ?? 0) - (b.cacheReadPricePer1M ?? 0);
          break;
        case "fetched":
          cmp = (a.fetchedAt || "").localeCompare(b.fetchedAt || "");
          break;
      }
      return sortDirection === "asc" ? cmp : -cmp;
    });

    return result;
  }, [models, searchFilter, sourceFilter, sortField, sortDirection]);

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
          <SourceBadge
            source="provider_api"
            onClick={() => toggleSourceFilter("provider_api")}
            isActive={sourceFilter.has("provider_api")}
          />
          <SourceBadge
            source="manual_override"
            onClick={() => toggleSourceFilter("manual_override")}
            isActive={sourceFilter.has("manual_override")}
          />
          <SourceBadge
            source="historical_average"
            onClick={() => toggleSourceFilter("historical_average")}
            isActive={sourceFilter.has("historical_average")}
          />
          {sourceFilter.size > 0 && (
            <Button
              variant="ghost"
              size="sm"
              className="h-5 px-1 text-xs"
              onClick={() => setSourceFilter(new Set())}
            >
              Clear
            </Button>
          )}
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
                    <SortableHeader
                      label="Model"
                      field="model"
                      currentSort={sortField}
                      direction={sortDirection}
                      onSort={handleSort}
                    />
                    <SortableHeader
                      label="Input"
                      field="input"
                      currentSort={sortField}
                      direction={sortDirection}
                      onSort={handleSort}
                      align="right"
                    />
                    <SortableHeader
                      label="Output"
                      field="output"
                      currentSort={sortField}
                      direction={sortDirection}
                      onSort={handleSort}
                      align="right"
                    />
                    <SortableHeader
                      label="Cache Read"
                      field="cacheRead"
                      currentSort={sortField}
                      direction={sortDirection}
                      onSort={handleSort}
                      align="right"
                    />
                    <SortableHeader
                      label="Fetched"
                      field="fetched"
                      currentSort={sortField}
                      direction={sortDirection}
                      onSort={handleSort}
                      align="center"
                    />
                    <th className="px-3 py-2 font-medium text-right">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredAndSortedModels.map((model) => (
                    <ModelPricingRow
                      key={model.canonicalName || model.model}
                      model={model}
                      onRecalculate={handleRecalculate}
                      onSelectModel={setSelectedModel}
                      recalculatingModel={recalculatingModel}
                    />
                  ))}
                  {filteredAndSortedModels.length === 0 && (
                    <tr>
                      <td colSpan={6} className="px-3 py-8 text-center text-muted-foreground">
                        {searchFilter || sourceFilter.size > 0 ? "No models match your filters." : "No pricing data available."}
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

      {/* Edit pricing dialog */}
      <EditPricingDialog
        model={selectedModel}
        modelPricing={models?.find((m) => (m.canonicalName || m.model) === selectedModel)}
        onClose={() => setSelectedModel(null)}
        onPricingUpdated={refetch}
      />
    </div>
  );
}
