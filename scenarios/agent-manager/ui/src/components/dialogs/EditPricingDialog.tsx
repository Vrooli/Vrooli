// Reusable dialog for editing model pricing overrides

import { useMemo, useState } from "react";
import { Pencil, X } from "lucide-react";
import { Badge } from "../ui/badge";
import { Button } from "../ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogBody,
} from "../ui/dialog";
import { Input } from "../ui/input";
import {
  useModelOverrides,
  useSetOverride,
  useDeleteOverride,
  useModelPricing,
} from "../../hooks/usePricing";
import type {
  ModelPricingListItem,
  PricingComponent,
  PricingSource,
  PriceOverride,
} from "../../types";

// =============================================================================
// Types
// =============================================================================

export interface EditPricingDialogProps {
  model: string | null;
  onClose: () => void;
  onPricingUpdated?: () => void;
}

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

function SourceBadge({ source }: { source: PricingSource }) {
  return (
    <Badge variant="outline" className={`text-xs ${sourceColor(source)}`}>
      {sourceLabel(source)}
    </Badge>
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
      return {
        price: modelPricing.cacheReadPricePer1M,
        source: modelPricing.cacheReadSource || "unknown",
      };
    case "cache_creation":
      return {
        price: modelPricing.cacheCreatePricePer1M,
        source: modelPricing.cacheCreateSource || "unknown",
      };
    default:
      return { price: undefined, source: "unknown" };
  }
}

// =============================================================================
// Main Component
// =============================================================================

export function EditPricingDialog({ model, onClose, onPricingUpdated }: EditPricingDialogProps) {
  const { data: allModels, refetch: refetchModels } = useModelPricing();
  const { data: overrides, loading, refetch } = useModelOverrides(model);
  const { setOverride, loading: settingOverride } = useSetOverride();
  const { deleteOverride, loading: deletingOverride } = useDeleteOverride();

  // Find the model pricing data
  const modelPricing = useMemo(
    () => allModels?.find((m) => (m.canonicalName || m.model) === model),
    [allModels, model]
  );

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
      refetchModels();
      onPricingUpdated?.();
    } catch {
      // Error handled by hook
    }
  };

  const handleDeleteOverride = async (component: PricingComponent) => {
    if (!model) return;
    try {
      await deleteOverride(model, component);
      refetch();
      refetchModels();
      onPricingUpdated?.();
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
            <p>
              <strong>Manual Override</strong> takes priority over provider and historical pricing.
            </p>
            <p className="mt-1">Click the pencil icon to set an override, or X to clear it.</p>
          </div>
        </DialogBody>
      </DialogContent>
    </Dialog>
  );
}
