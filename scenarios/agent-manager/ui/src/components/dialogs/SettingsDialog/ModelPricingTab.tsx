// Model Pricing Tab - displays and manages model pricing data with per-component sources

import { useCallback, useEffect, useMemo, useState } from "react";
import { Pencil, RefreshCw } from "lucide-react";
import { Badge } from "../../ui/badge";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../ui/card";
import { Input } from "../../ui/input";
import { Label } from "../../ui/label";
import { EditPricingDialog } from "../EditPricingDialog";
import { SortHeaderCell, compareSortValues, useTableSort } from "../../../features/stats/components/tables/SortableHeader";
import {
  useModelPricing,
  usePricingSettings,
  useUpdatePricingSettings,
  usePricingCacheStatus,
  useRefreshAllPricing,
  useRecalculateModelPricing,
} from "../../../hooks/usePricing";
import {
  formatPricingDisplay,
  formatPricingTimestamp,
  pricingSourceBadgeClass,
  pricingSourceLabel,
} from "../pricingDisplay";
import type {
  ModelPricingListItem,
  PricingSource,
  PricingSettings,
} from "../../../types";

// =============================================================================
// Types
// =============================================================================

type SortField = "model" | "input" | "output" | "cacheRead" | "fetched";

// =============================================================================
// Sub-Components
// =============================================================================

interface SourceBadgeProps {
  source: PricingSource;
  onClick?: () => void;
  isActive?: boolean;
}

function SourceBadge({ source, onClick, isActive }: SourceBadgeProps) {
  return (
    <Badge
      variant="outline"
      className={pricingSourceBadgeClass(source, { isActive, clickable: Boolean(onClick) })}
      onClick={onClick}
    >
      {pricingSourceLabel(source)}
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
          className="flex min-h-11 flex-col justify-center text-left transition-colors hover:text-primary sm:min-h-0"
        >
          <div className="font-medium text-sm">{model.model}</div>
          {model.canonicalName && model.canonicalName !== model.model && (
            <div className="text-xs text-muted-foreground">{model.canonicalName}</div>
          )}
        </button>
      </td>
      <td className="px-3 py-2 text-right">
        <div className="flex items-center justify-end gap-2">
          <span className="text-sm">{formatPricingDisplay(model.inputPricePer1M)}</span>
          <SourceBadge source={model.inputSource} />
        </div>
      </td>
      <td className="px-3 py-2 text-right">
        <div className="flex items-center justify-end gap-2">
          <span className="text-sm">{formatPricingDisplay(model.outputPricePer1M)}</span>
          <SourceBadge source={model.outputSource} />
        </div>
      </td>
      <td className="px-3 py-2 text-right">
        {model.cacheReadPricePer1M !== undefined && model.cacheReadSource && (
          <div className="flex items-center justify-end gap-2">
            <span className="text-sm">{formatPricingDisplay(model.cacheReadPricePer1M)}</span>
            <SourceBadge source={model.cacheReadSource} />
          </div>
        )}
      </td>
      <td className="px-3 py-2 text-center text-xs text-muted-foreground">
        {formatPricingTimestamp(model.fetchedAt)}
      </td>
      <td className="px-3 py-2">
        <div className="flex items-center justify-end gap-1">
          <Button
            variant="ghost"
            size="icon"
            className="h-11 w-11 sm:h-7 sm:w-7"
            onClick={() => onSelectModel(model.canonicalName || model.model)}
            title="Edit pricing overrides"
          >
            <Pencil className="h-3.5 w-3.5" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="h-11 w-11 sm:h-7 sm:w-7"
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
  const [ttlSeconds, setTtlSeconds] = useState(settings?.providerCacheTtlSeconds?.toString() || "21600");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!settings) return;
    setTtlSeconds(settings.providerCacheTtlSeconds.toString());
  }, [settings]);

  const handleSave = async () => {
    setSaving(true);
    try {
      await onUpdate({
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
          How long fetched provider prices are cached
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-2">
          <Label htmlFor="ttl" className="text-xs">Provider Cache TTL (seconds)</Label>
          <Input
            id="ttl"
            type="number"
            min={60}
            max={86400}
            value={ttlSeconds}
            onChange={(e) => setTtlSeconds(e.target.value)}
            className="h-11 sm:h-8"
          />
        </div>
        <Button size="sm" className="min-h-11 sm:min-h-9" onClick={handleSave} disabled={loading || saving}>
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
                  {p.modelCount} models | Last fetch: {formatPricingTimestamp(p.lastFetchedAt)}
                </div>
                {p.isStale && <Badge variant="outline" className="text-yellow-400 text-xs mt-1">Stale</Badge>}
              </div>
            ))}
          </>
        )}
        <Button size="sm" variant="secondary" className="min-h-11 sm:min-h-9" onClick={handleRefresh} disabled={loading || refreshing}>
          {refreshing ? "Refreshing..." : "Refresh All Pricing"}
        </Button>
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
  const [sourceFilter, setSourceFilter] = useState<Set<PricingSource>>(new Set());
  const sort = useTableSort<SortField>("model", { direction: "asc", textFields: ["model"] });

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

    const sortValue = (m: ModelPricingListItem): number | string => {
      switch (sort.field) {
        case "model": return m.model || "";
        case "input": return m.inputPricePer1M ?? 0;
        case "output": return m.outputPricePer1M ?? 0;
        case "cacheRead": return m.cacheReadPricePer1M ?? 0;
        case "fetched": return m.fetchedAt || "";
      }
    };
    result = [...result].sort((a, b) => compareSortValues(sortValue(a), sortValue(b), sort.direction));

    return result;
  }, [models, searchFilter, sourceFilter, sort.field, sort.direction]);

  if (loading) {
    return <div className="text-center py-8 text-muted-foreground">Loading pricing data...</div>;
  }

  if (error) {
    return (
      <div className="text-center py-8 text-red-400">
        Error loading pricing: {error}
        <Button variant="secondary" size="sm" onClick={refetch} className="ml-2">
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
            className="h-11 max-w-sm sm:h-8"
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
          {sourceFilter.size > 0 && (
            <Button
              variant="ghost"
              size="sm"
              className="min-h-11 px-2 text-xs sm:min-h-0 sm:h-5 sm:px-1"
              onClick={() => setSourceFilter(new Set())}
            >
              Clear
            </Button>
          )}
        </div>
      </div>

      {/* Main content grid */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-[1fr_280px]">
        {/* Model pricing table */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm">Model Pricing</CardTitle>
            <CardDescription className="text-xs">
              Prices per 1M tokens. Click a model to manage overrides.
            </CardDescription>
          </CardHeader>
          <CardContent className="p-0">
            {/* Scrolls both ways inside its own box: a narrow screen pans the table instead of clipping it. */}
            <div className="h-[400px] overflow-auto" data-testid="pricing-table-scroll" tabIndex={0} aria-label="Model pricing table">
              <table className="w-full min-w-[40rem] text-left">
                <thead className="sticky top-0 bg-card border-b">
                  <tr className="text-xs text-muted-foreground">
                    <SortHeaderCell field="model" label="Model" sort={sort} className="px-3 py-2" />
                    <SortHeaderCell field="input" label="Input" sort={sort} align="end" className="px-3 py-2" />
                    <SortHeaderCell field="output" label="Output" sort={sort} align="end" className="px-3 py-2" />
                    <SortHeaderCell field="cacheRead" label="Cache Read" sort={sort} align="end" className="px-3 py-2" />
                    <SortHeaderCell field="fetched" label="Fetched" sort={sort} align="center" className="px-3 py-2" />
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
            </div>
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
        onClose={() => setSelectedModel(null)}
        onPricingUpdated={refetch}
      />
    </div>
  );
}
