import { useEffect, useMemo, useState, type DragEvent } from 'react';
import { GripVertical, Plus } from 'lucide-react';
import { Button } from '../../../../shared/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../../shared/ui/card';
import { Callout } from '../Callout';
import { LAYOUT } from '../../config/layout.constants';
import { isDemoPlanOption } from '../../../../shared/lib/pricingPlaceholders';
import {
  filterPricesByTab,
  buildPricingPreviewData,
  getPriceIdentifier,
  type PriceFormState,
  type PriceFormValues,
} from '../../services/pricing.service';
import type { BundleCatalogEntry, StripeCoupon } from '../../../../shared/api';
import { PriceFormCard } from './PriceFormCard';
import { PriceReadOnlyCard } from './PriceReadOnlyCard';
import { PlanPreview } from './PlanPreview';
import { useResizableColumns } from '../../hooks/useResizableColumns';

interface PriceVerificationResult {
  status: string;
  message?: string;
}

export interface BundleCardProps {
  mode: 'preview' | 'edit';
  entry: BundleCatalogEntry;
  priceForms: Record<string, PriceFormState>;
  pricingTab: 'month' | 'year' | 'other';
  includeDemoPlaceholders: boolean;
  priceChecks: Record<string, PriceVerificationResult>;
  onPriceChange: (bundleKey: string, priceId: string, field: keyof PriceFormValues, transformer?: (value: string) => string | number) => (event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => void;
  onSavePrice: (bundleKey: string, priceId: string) => Promise<void>;
  onVerifyPrice: (bundleKey: string, priceId: string) => Promise<void>;
  onRemoveDemoPlan: (bundleKey: string, priceId: string) => void;
  onDeletePlan?: (bundleKey: string, priceId: string) => void;
  onReorderPlans?: (bundleKey: string, orderedPriceIds: string[]) => void;
  onAddPlan?: (bundleKey: string) => void;
  // Coupon mapping props
  availableCoupons?: StripeCoupon[];
  couponMappings?: Record<string, string>; // priceID -> couponID
  onAssignCoupon?: (priceId: string, couponId: string) => Promise<{ success: boolean; error?: string }>;
  onUnassignCoupon?: (priceId: string) => Promise<{ success: boolean; error?: string }>;
  couponSaving?: boolean;
}

export function BundleCard({
  mode,
  entry,
  priceForms,
  pricingTab,
  includeDemoPlaceholders,
  priceChecks,
  onPriceChange,
  onSavePrice,
  onVerifyPrice,
  onRemoveDemoPlan,
  onDeletePlan,
  onReorderPlans,
  onAddPlan,
  availableCoupons,
  couponMappings,
  onAssignCoupon,
  onUnassignCoupon,
  couponSaving,
}: BundleCardProps) {
  const visiblePrices = useMemo(
    () => filterPricesByTab(entry.prices, pricingTab, includeDemoPlaceholders),
    [entry.prices, pricingTab, includeDemoPlaceholders]
  );
  const demoHidden = entry.prices.some(isDemoPlanOption) && !includeDemoPlaceholders;
  const previewData = buildPricingPreviewData(entry, priceForms, includeDemoPlaceholders);
  const canReorder = mode === 'edit' && typeof onReorderPlans === 'function';

  const sortedVisiblePrices = useMemo(() => {
    const enriched = visiblePrices.map((price, index) => {
      const priceIdentifier = getPriceIdentifier(price);
      const key = `${entry.bundle.bundle_key}:${priceIdentifier}`;
      const formState = priceForms[key];
      const weight = formState?.values.displayWeight ?? price.display_weight ?? 0;
      const rank = typeof price.plan_rank === 'number' ? price.plan_rank : Number.MAX_SAFE_INTEGER;
      return { price, priceIdentifier, weight, rank, index };
    });

    return enriched.sort((a, b) => {
      if (a.weight === b.weight) {
        if (a.rank === b.rank) {
          return a.index - b.index;
        }
        return a.rank - b.rank;
      }
      return b.weight - a.weight;
    });
  }, [entry.bundle.bundle_key, priceForms, visiblePrices]);

  const [collapsedById, setCollapsedById] = useState<Record<string, boolean>>({});
  const [draggingId, setDraggingId] = useState<string | null>(null);
  const [dragOverId, setDragOverId] = useState<string | null>(null);

  useEffect(() => {
    if (sortedVisiblePrices.length === 0) return;
    setCollapsedById((prev) => {
      let changed = false;
      const next = { ...prev };
      const ids = new Set(sortedVisiblePrices.map((item) => item.priceIdentifier));

      sortedVisiblePrices.forEach((item) => {
        if (next[item.priceIdentifier] === undefined) {
          next[item.priceIdentifier] = true;
          changed = true;
        }
      });

      Object.keys(next).forEach((id) => {
        if (!ids.has(id)) {
          delete next[id];
          changed = true;
        }
      });

      return changed ? next : prev;
    });
  }, [sortedVisiblePrices]);

  const handleToggleCollapse = (priceId: string) => {
    setCollapsedById((prev) => {
      const isCollapsed = prev[priceId] ?? true;
      const next: Record<string, boolean> = {};
      sortedVisiblePrices.forEach((item) => {
        next[item.priceIdentifier] = item.priceIdentifier === priceId ? !isCollapsed : true;
      });
      return next;
    });
  };

  const moveItem = <T,>(items: T[], from: number, to: number): T[] => {
    const next = [...items];
    const [moved] = next.splice(from, 1);
    if (!moved) return items;
    next.splice(to, 0, moved);
    return next;
  };

  const handleDragStart = (priceId: string) => (event: DragEvent<HTMLElement>) => {
    if (!canReorder) return;
    setDraggingId(priceId);
    event.dataTransfer.effectAllowed = 'move';
    event.dataTransfer.setData('text/plain', priceId);
  };

  const handleDragOver = (priceId: string) => (event: DragEvent<HTMLElement>) => {
    if (!canReorder) return;
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
    if (draggingId && priceId !== draggingId) {
      setDragOverId(priceId);
    }
  };

  const handleDragLeave = () => {
    if (!canReorder) return;
    setDragOverId(null);
  };

  const handleDrop = (targetId: string) => (event: DragEvent<HTMLElement>) => {
    if (!canReorder) return;
    event.preventDefault();
    const sourceId = event.dataTransfer.getData('text/plain');
    if (!sourceId || sourceId === targetId) {
      setDraggingId(null);
      setDragOverId(null);
      return;
    }

    const ids = sortedVisiblePrices.map((item) => item.priceIdentifier);
    const sourceIndex = ids.indexOf(sourceId);
    const targetIndex = ids.indexOf(targetId);
    if (sourceIndex === -1 || targetIndex === -1) {
      setDraggingId(null);
      setDragOverId(null);
      return;
    }

    const reordered = moveItem(ids, sourceIndex, targetIndex);
    onReorderPlans?.(entry.bundle.bundle_key, reordered);
    setDraggingId(null);
    setDragOverId(null);
  };

  const handleDragEnd = () => {
    if (!canReorder) return;
    setDraggingId(null);
    setDragOverId(null);
  };

  // Resizable columns: left (form cards) starts at 40%, right (preview) at 60%
  const {
    isResizing,
    containerRef,
    handleResizeStart,
    leftColumnStyle,
    rightColumnStyle,
  } = useResizableColumns({
    defaultLeftRatio: 0.4, // Form cards 40%, preview 60%
    minRatio: 0.25,
    maxRatio: 0.6,
    storageKey: `lpbs.bundleColumns.${entry.bundle.bundle_key}`,
  });

  return (
    <Card className={LAYOUT.card.base}>
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <span>{entry.bundle.name}</span>
          <div className="flex items-center gap-3">
            {mode === 'edit' && onAddPlan && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => onAddPlan(entry.bundle.bundle_key)}
                className="gap-1"
              >
                <Plus className="h-4 w-4" />
                Add Plan
              </Button>
            )}
            <span className="text-xs text-slate-400">Bundle key: {entry.bundle.bundle_key}</span>
          </div>
        </CardTitle>
        <CardDescription>
          {mode === 'edit'
            ? 'Control which Stripe prices are surfaced on the landing page and customize their marketing copy.'
            : 'Preview of pricing plans synced from Stripe.'}
        </CardDescription>
      </CardHeader>
      <CardContent className={LAYOUT.contentSpacing}>
        {demoHidden && (
          <Callout
            type="warning"
            message="Demo placeholders hidden. Turn them back on to see filler tiers until you add real Stripe prices."
          />
        )}
        {/* Resizable two-column layout */}
        <div
          ref={containerRef}
          className={`hidden xl:flex gap-6 ${isResizing ? 'select-none' : ''}`}
        >
          {/* Left column: form cards */}
          <div style={leftColumnStyle} className="space-y-4 min-w-0 overflow-y-auto max-h-[700px] pr-1">
            {sortedVisiblePrices.map((item, index) => {
              const { price, priceIdentifier } = item;
              const key = `${entry.bundle.bundle_key}:${priceIdentifier}`;
              const formState = priceForms[key];
              if (!formState) return null;

              if (mode === 'preview') {
                return (
                  <PriceReadOnlyCard
                    key={key}
                    price={price}
                    formState={formState}
                  />
                );
              }

              const assignedCouponId = couponMappings?.[priceIdentifier];
              const assignedCoupon = assignedCouponId
                ? availableCoupons?.find((c) => c.id === assignedCouponId)
                : undefined;

              return (
                <PriceFormCard
                  key={key}
                  bundleKey={entry.bundle.bundle_key}
                  priceIdentifier={priceIdentifier}
                  price={price}
                  formState={formState}
                  priceCheck={priceChecks[key]}
                  isCollapsed={collapsedById[priceIdentifier] ?? true}
                  onToggleCollapse={() => handleToggleCollapse(priceIdentifier)}
                  onPriceChange={onPriceChange}
                  onSavePrice={onSavePrice}
                  onVerifyPrice={onVerifyPrice}
                  onRemoveDemoPlan={onRemoveDemoPlan}
                  onDeletePlan={onDeletePlan}
                  planIndex={index}
                  draggable={canReorder}
                  onDragStart={handleDragStart(priceIdentifier)}
                  onDragOver={handleDragOver(priceIdentifier)}
                  onDragLeave={handleDragLeave}
                  onDrop={handleDrop(priceIdentifier)}
                  onDragEnd={handleDragEnd}
                  isDragging={draggingId === priceIdentifier}
                  isDragOver={dragOverId === priceIdentifier}
                  availableCoupons={availableCoupons}
                  assignedCoupon={assignedCoupon}
                  onAssignCoupon={onAssignCoupon}
                  onUnassignCoupon={onUnassignCoupon}
                  couponSaving={couponSaving}
                />
              );
            })}
            {visiblePrices.length === 0 && (
              <div className="py-6 text-center">
                <p className="text-sm text-slate-400">
                  No plans found for this interval.
                </p>
              </div>
            )}
          </div>

          {/* Resize handle - thick and visible */}
          <div
            role="separator"
            aria-orientation="vertical"
            aria-label="Resize columns"
            tabIndex={0}
            onMouseDown={handleResizeStart}
            className="relative flex-shrink-0 w-6 cursor-col-resize flex items-center justify-center group"
          >
            {/* Visual indicator bar */}
            <div
              className={`absolute inset-y-0 left-1/2 -translate-x-1/2 w-1 rounded-full transition-colors ${
                isResizing ? 'bg-primary' : 'bg-slate-600 group-hover:bg-primary/70'
              }`}
            />
            <GripVertical
              className={`h-8 w-4 text-slate-400 opacity-60 group-hover:opacity-100 transition-opacity z-10 ${
                isResizing ? 'opacity-100 text-primary' : ''
              }`}
            />
          </div>

          {/* Right column: preview */}
          <div style={rightColumnStyle} className="min-w-0">
            <PlanPreview data={previewData} />
          </div>
        </div>

        {/* Mobile: stacked layout (no resize) */}
        <div className="xl:hidden space-y-6">
          <div className="space-y-4">
            {sortedVisiblePrices.map((item, index) => {
              const { price, priceIdentifier } = item;
              const key = `${entry.bundle.bundle_key}:${priceIdentifier}`;
              const formState = priceForms[key];
              if (!formState) return null;

              if (mode === 'preview') {
                return (
                  <PriceReadOnlyCard
                    key={key}
                    price={price}
                    formState={formState}
                  />
                );
              }

              const assignedCouponIdMobile = couponMappings?.[priceIdentifier];
              const assignedCouponMobile = assignedCouponIdMobile
                ? availableCoupons?.find((c) => c.id === assignedCouponIdMobile)
                : undefined;

              return (
                <PriceFormCard
                  key={key}
                  bundleKey={entry.bundle.bundle_key}
                  priceIdentifier={priceIdentifier}
                  price={price}
                  formState={formState}
                  priceCheck={priceChecks[key]}
                  isCollapsed={collapsedById[priceIdentifier] ?? true}
                  onToggleCollapse={() => handleToggleCollapse(priceIdentifier)}
                  onPriceChange={onPriceChange}
                  onSavePrice={onSavePrice}
                  onVerifyPrice={onVerifyPrice}
                  onRemoveDemoPlan={onRemoveDemoPlan}
                  onDeletePlan={onDeletePlan}
                  planIndex={index}
                  draggable={canReorder}
                  onDragStart={handleDragStart(priceIdentifier)}
                  onDragOver={handleDragOver(priceIdentifier)}
                  onDragLeave={handleDragLeave}
                  onDrop={handleDrop(priceIdentifier)}
                  onDragEnd={handleDragEnd}
                  isDragging={draggingId === priceIdentifier}
                  isDragOver={dragOverId === priceIdentifier}
                  availableCoupons={availableCoupons}
                  assignedCoupon={assignedCouponMobile}
                  onAssignCoupon={onAssignCoupon}
                  onUnassignCoupon={onUnassignCoupon}
                  couponSaving={couponSaving}
                />
              );
            })}
            {visiblePrices.length === 0 && (
              <div className="py-6 text-center">
                <p className="text-sm text-slate-400">
                  No plans found for this interval.
                </p>
              </div>
            )}
          </div>
          <PlanPreview data={previewData} />
        </div>
      </CardContent>
    </Card>
  );
}
