import { Package, RefreshCw, Plus } from 'lucide-react';
import { Button } from '../../../../shared/ui/button';
import { Card, CardContent } from '../../../../shared/ui/card';
import { Callout } from '../Callout';
import { LAYOUT } from '../../config/layout.constants';
import { IntervalTabs, type IntervalTab } from './IntervalTabs';
import { BundleCard } from './BundleCard';
import type { BundleCatalogEntry } from '../../../../shared/api';
import type { PriceFormState, PriceFormValues } from '../../services/pricing.service';

interface PriceVerificationResult {
  status: string;
  message?: string;
}

export interface PlanDisplayManagerProps {
  mode: 'preview' | 'edit';
  bundles: BundleCatalogEntry[];
  priceForms: Record<string, PriceFormState>;
  activeTab: IntervalTab;
  onTabChange: (tab: IntervalTab) => void;
  showDemoPlaceholders: boolean;
  onToggleDemoPlaceholders: () => void;
  onReload: () => void;
  loading: boolean;
  error: string | null;
  defaultBundleKey?: string;
  // Edit-mode handlers (required in edit mode, optional in preview)
  onPriceChange: (bundleKey: string, priceId: string, field: keyof PriceFormValues, transformer?: (value: string) => string | number) => (event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => void;
  onSavePrice: (bundleKey: string, priceId: string) => Promise<void>;
  onVerifyPrice: (bundleKey: string, priceId: string) => Promise<void>;
  onRemoveDemoPlan: (bundleKey: string, priceId: string) => void;
  priceChecks: Record<string, PriceVerificationResult>;
  // Add plan handler
  onAddPlan?: (bundleKey: string) => void;
}

export function PlanDisplayManager({
  mode,
  bundles,
  priceForms,
  activeTab,
  onTabChange,
  showDemoPlaceholders,
  onToggleDemoPlaceholders,
  onReload,
  loading,
  error,
  defaultBundleKey,
  onPriceChange,
  onSavePrice,
  onVerifyPrice,
  onRemoveDemoPlan,
  priceChecks,
  onAddPlan,
}: PlanDisplayManagerProps) {
  const canAddPlan = Boolean(onAddPlan && defaultBundleKey);

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-xl font-semibold text-white">
          {mode === 'preview' ? 'Preview' : 'Plan Display Manager'}
        </h2>
        <div className="flex items-center gap-2">
          <IntervalTabs activeTab={activeTab} onTabChange={onTabChange} />
          <Button
            variant="ghost"
            size="sm"
            onClick={onToggleDemoPlaceholders}
            className="gap-2"
          >
            {showDemoPlaceholders ? 'Hide demo placeholders' : 'Show demo placeholders'}
          </Button>
          <Button variant="ghost" size="sm" onClick={onReload} className="gap-2">
            <RefreshCw className="h-4 w-4" /> Reload catalog
          </Button>
        </div>
      </div>
      <div className="space-y-6">
        {loading && <p className="text-slate-400">Loading bundle catalog...</p>}
        {error && <Callout type="error" message={error} />}
        {!loading && !error && bundles.length === 0 && (
          <Card className={LAYOUT.card.base}>
            <CardContent className="py-12 text-center">
              <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-slate-800">
                <Package className="h-8 w-8 text-slate-400" />
              </div>
              <h3 className="mb-2 text-lg font-semibold text-white">No pricing plans configured</h3>
              <p className="mb-6 max-w-md mx-auto text-sm text-slate-400">
                You don't have any pricing bundles set up yet. Enable "Show demo placeholders" to preview
                the layout with sample plans, or connect your Stripe account to create real pricing tiers.
              </p>
              <div className="flex flex-col items-center justify-center gap-3 sm:flex-row">
                {canAddPlan && (
                  <Button
                    onClick={() => onAddPlan?.(defaultBundleKey as string)}
                    className="gap-2"
                  >
                    <Plus className="h-4 w-4" />
                    Add plan
                  </Button>
                )}
                <Button
                  variant="outline"
                  onClick={onToggleDemoPlaceholders}
                  className="gap-2"
                >
                  {showDemoPlaceholders ? 'Hide demo placeholders' : 'Show demo placeholders'}
                </Button>
              </div>
            </CardContent>
          </Card>
        )}
        {!loading && !error && bundles.length > 0 && bundles.map((entry) => (
          <BundleCard
            key={entry.bundle.bundle_key}
            mode={mode}
            entry={entry}
            priceForms={priceForms}
            pricingTab={activeTab}
            includeDemoPlaceholders={showDemoPlaceholders}
            priceChecks={priceChecks}
            onPriceChange={onPriceChange}
            onSavePrice={onSavePrice}
            onVerifyPrice={onVerifyPrice}
            onRemoveDemoPlan={onRemoveDemoPlan}
            onAddPlan={onAddPlan}
          />
        ))}
      </div>
    </div>
  );
}
