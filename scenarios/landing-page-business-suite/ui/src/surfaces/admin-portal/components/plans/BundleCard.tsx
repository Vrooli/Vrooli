import { GripVertical } from 'lucide-react';
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
import type { BundleCatalogEntry } from '../../../../shared/api';
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
}: BundleCardProps) {
  const visiblePrices = filterPricesByTab(entry.prices, pricingTab, includeDemoPlaceholders);
  const demoHidden = entry.prices.some(isDemoPlanOption) && !includeDemoPlaceholders;
  const previewData = buildPricingPreviewData(entry, priceForms, includeDemoPlaceholders);

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
          <span className="text-xs text-slate-400">Bundle key: {entry.bundle.bundle_key}</span>
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
          <div style={leftColumnStyle} className="divide-y divide-slate-700/50 min-w-0 overflow-y-auto max-h-[700px]">
            {visiblePrices.map((price) => {
              const priceIdentifier = getPriceIdentifier(price);
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

              return (
                <PriceFormCard
                  key={key}
                  bundleKey={entry.bundle.bundle_key}
                  priceIdentifier={priceIdentifier}
                  price={price}
                  formState={formState}
                  priceCheck={priceChecks[key]}
                  onPriceChange={onPriceChange}
                  onSavePrice={onSavePrice}
                  onVerifyPrice={onVerifyPrice}
                  onRemoveDemoPlan={onRemoveDemoPlan}
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
            {visiblePrices.map((price) => {
              const priceIdentifier = getPriceIdentifier(price);
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

              return (
                <PriceFormCard
                  key={key}
                  bundleKey={entry.bundle.bundle_key}
                  priceIdentifier={priceIdentifier}
                  price={price}
                  formState={formState}
                  priceCheck={priceChecks[key]}
                  onPriceChange={onPriceChange}
                  onSavePrice={onSavePrice}
                  onVerifyPrice={onVerifyPrice}
                  onRemoveDemoPlan={onRemoveDemoPlan}
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
