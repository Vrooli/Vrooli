import { useEffect, useMemo, useRef, useState } from 'react';
import { AlertCircle, Check, Download, Loader2 } from 'lucide-react';
import { Button } from '../../../../shared/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../../../../shared/ui/dialog';
import { Input } from '../../../../shared/ui/input';
import type { StripeProductWithPrices } from '../../../../shared/api/billing';
import { cn } from '../../../../shared/lib/utils';
import { Callout } from '../Callout';
import type { UseStripeImportReturn } from '../../hooks/useStripeImport';

type PriceFilter = 'all' | 'active' | 'existing' | 'inactive';
const PRICE_FILTERS: Array<{ key: PriceFilter; label: string }> = [
  { key: 'all', label: 'All' },
  { key: 'active', label: 'Active' },
  { key: 'existing', label: 'Existing' },
  { key: 'inactive', label: 'Inactive' },
];

interface ImportStripeModalProps {
  stripeImport: UseStripeImportReturn;
}

export function ImportStripeModal({ stripeImport }: ImportStripeModalProps) {
  const {
    isModalOpen,
    closeModal,
    preview,
    loading,
    error,
    selectedProductId,
    selectedProduct,
    selectProduct,
    selections,
    setPriceSelected,
    setSelectionsForPrices,
    selectActive,
    selectExisting,
    clearSelections,
    importing,
    importResult,
    handleImport,
  } = stripeImport;

  const [filter, setFilter] = useState<PriceFilter>('all');
  const [productQuery, setProductQuery] = useState('');
  const [priceQuery, setPriceQuery] = useState('');
  const [confirmReplace, setConfirmReplace] = useState(false);

  const formatPrice = (amountCents: number, currency: string) => {
    const safeAmount = Number.isFinite(amountCents) ? amountCents : 0;
    const safeCurrency = currency.toUpperCase() || 'USD';
    try {
      return new Intl.NumberFormat('en-US', {
        style: 'currency',
        currency: safeCurrency,
      }).format(safeAmount / 100);
    } catch {
      return `${(safeAmount / 100).toFixed(2)} ${safeCurrency}`;
    }
  };

  const formatInterval = (interval?: string) => {
    if (!interval) return 'one-time';
    return interval === 'month' ? '/mo' : interval === 'year' ? '/yr' : `/${interval}`;
  };

  const currentBundleProductId = preview?.bundle_product_id ?? '';
  const bundlePlanCount = preview?.bundle_plan_count ?? 0;
  const bundleProductFound = preview?.bundle_product_found ?? false;
  const isSwitchingProduct = Boolean(
    selectedProductId && currentBundleProductId && selectedProductId !== currentBundleProductId
  );

  const selectionStats = useMemo(() => {
    if (!selectedProduct) {
      return {
        total: 0,
        active: 0,
        inactive: 0,
        existing: 0,
        selected: 0,
        selectedActive: 0,
        selectedInactive: 0,
        selectedExisting: 0,
      };
    }

    let total = 0;
    let active = 0;
    let inactive = 0;
    let existing = 0;
    let selected = 0;
    let selectedActive = 0;
    let selectedInactive = 0;
    let selectedExisting = 0;

    selectedProduct.prices.forEach((price) => {
      total += 1;
      if (price.active) {
        active += 1;
      } else {
        inactive += 1;
      }
      if (price.exists_locally) {
        existing += 1;
      }

      if (!selections[price.price_id]) return;
      selected += 1;
      if (price.active) selectedActive += 1;
      if (!price.active) selectedInactive += 1;
      if (price.exists_locally) selectedExisting += 1;
    });

    return {
      total,
      active,
      inactive,
      existing,
      selected,
      selectedActive,
      selectedInactive,
      selectedExisting,
    };
  }, [selectedProduct, selections]);

  useEffect(() => {
    if (!isModalOpen) {
      setFilter('all');
      setProductQuery('');
      setPriceQuery('');
      setConfirmReplace(false);
    }
  }, [isModalOpen]);

  useEffect(() => {
    if (selectionStats.selected === 0) {
      setConfirmReplace(false);
    }
  }, [selectionStats.selected]);

  useEffect(() => {
    setConfirmReplace(false);
  }, [selectedProductId]);

  const filteredProducts = useMemo(() => {
    if (!preview) return [];
    const normalizedQuery = productQuery.trim().toLowerCase();
    if (!normalizedQuery) return preview.products;
    return preview.products.filter((product) => {
      const productText = `${product.product_name} ${product.product_id}`.toLowerCase();
      return productText.includes(normalizedQuery);
    });
  }, [preview, productQuery]);

  const filteredPrices = useMemo(() => {
    if (!selectedProduct) return [];
    const normalizedQuery = priceQuery.trim().toLowerCase();

    return selectedProduct.prices.filter((price) => {
      const matchesFilter = (() => {
        if (filter === 'active') return price.active;
        if (filter === 'existing') return price.exists_locally;
        if (filter === 'inactive') return !price.active;
        return true;
      })();

      if (!matchesFilter) return false;
      if (!normalizedQuery) return true;
      const priceText = `${price.price_id} ${price.lookup_key ?? ''}`.toLowerCase();
      return priceText.includes(normalizedQuery);
    });
  }, [selectedProduct, filter, priceQuery]);

  const requiresConfirm = selectionStats.selected > 0 && (bundlePlanCount > 0 || isSwitchingProduct);
  const canImport = selectionStats.selected > 0 && (!requiresConfirm || confirmReplace);

  return (
    <Dialog
      open={isModalOpen}
      onOpenChange={(open: boolean) => {
        if (!open) {
          closeModal();
        }
      }}
    >
      <DialogContent className="max-w-3xl max-h-[80vh] flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Download className="h-5 w-5" />
            Import Plans from Stripe
          </DialogTitle>
          <DialogDescription>
            Choose a Stripe product, then select the prices you want to keep. Import replaces existing plans for this
            bundle.
          </DialogDescription>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto space-y-4 py-4">
          {loading && (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-8 w-8 animate-spin text-primary" />
              <span className="ml-2 text-slate-400">Loading Stripe products...</span>
            </div>
          )}

          {error && (
            <Callout type="error" message={error} />
          )}

          {importResult && (
            <Callout
              type={importResult.errors?.length ? 'warning' : 'success'}
              message={`Imported: ${String(importResult.imported)} | Overwritten: ${String(importResult.overwritten)} | Skipped: ${String(importResult.skipped)}${
                importResult.errors?.length ? ` | Errors: ${String(importResult.errors.length)}` : ''
              }`}
            />
          )}

          {preview && !loading && (
            <>
              {preview.products.length === 0 ? (
                <div className="text-center py-8 text-slate-400">
                  No products found in your Stripe account.
                </div>
              ) : (
                <div className="space-y-6">
                  {!bundleProductFound && currentBundleProductId && (
                    <Callout
                      type="warning"
                      message={`Current bundle product ${currentBundleProductId} was not found in this Stripe account. Select a product below to link and import.`}
                    />
                  )}

                  <div className="space-y-3">
                    <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
                      <div>
                        <p className="text-sm font-semibold text-white">Step 1 · Choose a Stripe product</p>
                        <p className="text-xs text-slate-400">
                          This links the bundle to a single Stripe product.
                        </p>
                      </div>
                      <div className="w-full md:w-64">
                        <Input
                          placeholder="Search products"
                          value={productQuery}
                          onChange={(event) => { setProductQuery(event.target.value); }}
                        />
                      </div>
                    </div>
                    <div className="grid gap-3 md:grid-cols-2">
                      {filteredProducts.map((product) => (
                        <ProductSelectCard
                          key={product.product_id}
                          product={product}
                          selected={product.product_id === selectedProductId}
                          onSelect={() => { selectProduct(product.product_id); }}
                        />
                      ))}
                    </div>
                    {filteredProducts.length === 0 && (
                      <div className="text-center py-6 text-slate-400">
                        No products match your search.
                      </div>
                    )}
                  </div>

                  <div className="space-y-3">
                    <div>
                      <p className="text-sm font-semibold text-white">Step 2 · Select prices to keep</p>
                      <p className="text-xs text-slate-400">
                        Import replaces existing plans for this bundle.
                      </p>
                    </div>

                    {!selectedProduct ? (
                      <div className="text-center py-8 text-slate-400">
                        Select a product to view prices.
                      </div>
                    ) : (
                      <>
                        <div className="rounded-lg border border-slate-700/60 bg-slate-900/50 p-4">
                          <div className="grid grid-cols-2 gap-3 text-center md:grid-cols-4">
                            <div className="rounded-lg bg-slate-800/50 p-3">
                              <div className="text-2xl font-bold text-white">{selectionStats.total}</div>
                              <div className="text-xs text-slate-400">Total Prices</div>
                            </div>
                            <div className="rounded-lg bg-emerald-500/10 p-3">
                              <div className="text-2xl font-bold text-emerald-400">{selectionStats.active}</div>
                              <div className="text-xs text-slate-400">Active</div>
                            </div>
                            <div className="rounded-lg bg-amber-500/10 p-3">
                              <div className="text-2xl font-bold text-amber-400">{selectionStats.existing}</div>
                              <div className="text-xs text-slate-400">Existing</div>
                            </div>
                            <div className="rounded-lg bg-slate-800/50 p-3">
                              <div className="text-2xl font-bold text-white">{selectionStats.inactive}</div>
                              <div className="text-xs text-slate-400">Inactive</div>
                            </div>
                          </div>
                          <div className="mt-3 flex flex-wrap items-center gap-2">
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={selectActive}
                              className="text-xs"
                            >
                              Select Active
                            </Button>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={selectExisting}
                              className="text-xs"
                            >
                              Select Existing
                            </Button>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={clearSelections}
                              className="text-xs"
                            >
                              Clear Selection
                            </Button>
                            <span className="text-xs text-slate-500">Inactive prices are not auto-selected.</span>
                          </div>
                        </div>

                        {isSwitchingProduct && (
                          <Callout
                            type="warning"
                            message="Switching products will relink this bundle and replace all current plans with the selected prices."
                          />
                        )}

                        {bundlePlanCount > 0 && (
                          <Callout
                            type="warning"
                            message={`Import will replace ${String(bundlePlanCount)} existing plan${bundlePlanCount === 1 ? '' : 's'} in the catalog.`}
                          />
                        )}

                        <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
                          <div className="flex flex-wrap gap-2">
                            {PRICE_FILTERS.map((option) => {
                              const isActive = filter === option.key;
                              return (
                                <Button
                                  key={option.key}
                                  variant="outline"
                                  size="sm"
                                  onClick={() => { setFilter(option.key); }}
                                  className={cn(
                                    'text-xs',
                                    isActive
                                      ? 'border-slate-500 bg-slate-800 text-white'
                                      : 'border-slate-700 text-slate-400'
                                  )}
                                >
                                  {option.label}
                                </Button>
                              );
                            })}
                          </div>
                          <div className="w-full md:w-56">
                            <Input
                              placeholder="Search price IDs or lookup keys"
                              value={priceQuery}
                              onChange={(event) => { setPriceQuery(event.target.value); }}
                            />
                          </div>
                        </div>

                        <div className="space-y-4">
                          <div className="border border-slate-700 rounded-lg overflow-hidden">
                            <ProductHeader
                              product={selectedProduct}
                              prices={selectedProduct.prices}
                              selections={selections}
                              onSelectionChange={(selected) => {
                                const priceIds = selectedProduct.prices
                                  .filter((price) => price.active)
                                  .map((price) => price.price_id);
                                setSelectionsForPrices(priceIds, selected);
                              }}
                            />
                            <div className="divide-y divide-slate-700/50">
                              {filteredPrices.map((price) => (
                                <div
                                  key={price.price_id}
                                  className={`px-4 py-3 flex items-start gap-4 ${price.active ? '' : 'opacity-70'}`}
                                >
                                  <input
                                    type="checkbox"
                                    className="mt-1 h-4 w-4 rounded border-slate-600 bg-slate-900 text-emerald-400 focus:ring-2 focus:ring-emerald-400/60"
                                    checked={Boolean(selections[price.price_id])}
                                    onChange={(event) => { setPriceSelected(price.price_id, event.target.checked); }}
                                  />
                                  <div className="flex-1 min-w-0 space-y-1">
                                    <div className="flex flex-wrap items-center gap-2">
                                      <span className="font-mono text-sm text-white">
                                        {formatPrice(price.amount_cents, price.currency)}
                                        <span className="text-slate-400">{formatInterval(price.interval)}</span>
                                      </span>
                                      {price.exists_locally && (
                                        <span className="inline-flex items-center rounded px-2 py-0.5 text-xs font-medium bg-amber-500/20 text-amber-300">
                                          Existing
                                        </span>
                                      )}
                                      {!price.active && (
                                        <span className="inline-flex items-center rounded px-2 py-0.5 text-xs font-medium bg-red-500/20 text-red-300">
                                          Inactive
                                        </span>
                                      )}
                                    </div>
                                    <p className="text-xs text-slate-500 font-mono truncate">{price.price_id}</p>
                                    {price.lookup_key && (
                                      <p className="text-xs text-slate-400">lookup_key: {price.lookup_key}</p>
                                    )}
                                    {price.exists_locally && (
                                      <div className="flex items-center gap-1 text-xs text-amber-300">
                                        <AlertCircle className="h-3.5 w-3.5" />
                                        Already in the current plan catalog
                                      </div>
                                    )}
                                    {!price.active && (
                                      <p className="text-xs text-slate-500">Inactive in Stripe - not auto-selected.</p>
                                    )}
                                  </div>
                                </div>
                              ))}
                            </div>
                          </div>

                          {filteredPrices.length === 0 && (
                            <div className="text-center py-8 text-slate-400">
                              No prices match your filters.
                            </div>
                          )}
                        </div>
                      </>
                    )}
                  </div>
                </div>
              )}
            </>
          )}
        </div>

        <DialogFooter className="border-t border-slate-700 pt-4 gap-3 flex-col sm:flex-row sm:items-center sm:justify-between">
          <div className="flex flex-col gap-2 text-sm text-slate-400">
            <div className="flex items-center gap-2">
              <Check className="h-4 w-4 text-emerald-400" />
              <span>{selectionStats.selected} selected</span>
              {selectionStats.selectedExisting > 0 && (
                <>
                  <span className="text-slate-600">|</span>
                  <AlertCircle className="h-4 w-4 text-amber-400" />
                  <span>{selectionStats.selectedExisting} existing</span>
                </>
              )}
              {selectionStats.selectedInactive > 0 && (
                <>
                  <span className="text-slate-600">|</span>
                  <AlertCircle className="h-4 w-4 text-red-400" />
                  <span>{selectionStats.selectedInactive} inactive</span>
                </>
              )}
            </div>
            {requiresConfirm && (
              <label className="flex items-start gap-2 text-xs text-amber-200">
                <input
                  type="checkbox"
                  className="mt-0.5 h-4 w-4 rounded border-amber-400/60 bg-slate-900 text-amber-400 focus:ring-2 focus:ring-amber-400/60"
                  checked={confirmReplace}
                  onChange={(event) => { setConfirmReplace(event.target.checked); }}
                />
                <span>
                  {isSwitchingProduct
                    ? 'I understand this will relink the bundle to the selected Stripe product and replace existing plans.'
                    : 'I understand this will replace existing plans in this bundle.'}
                </span>
              </label>
            )}
          </div>
          <div className="flex items-center gap-2">
            <Button variant="outline" onClick={closeModal} disabled={importing}>
              Cancel
            </Button>
            <Button
              onClick={() => {
                void handleImport();
              }}
              disabled={importing || loading || !canImport}
            >
              {importing ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Importing...
                </>
              ) : (
                <>
                  <Download className="mr-2 h-4 w-4" />
                  Import Selected
                </>
              )}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

interface ProductHeaderProps {
  product: StripeProductWithPrices;
  prices: StripeProductWithPrices['prices'];
  selections: Record<string, boolean>;
  onSelectionChange: (selected: boolean) => void;
}

function ProductHeader({ product, prices, selections, onSelectionChange }: ProductHeaderProps) {
  const inputRef = useRef<HTMLInputElement>(null);

  const stats = useMemo(() => {
    let activeCount = 0;
    let existingCount = 0;
    let inactiveCount = 0;
    let selectedActiveCount = 0;

    prices.forEach((price) => {
      if (price.active) {
        activeCount += 1;
        if (selections[price.price_id]) {
          selectedActiveCount += 1;
        }
      } else {
        inactiveCount += 1;
      }
      if (price.exists_locally) {
        existingCount += 1;
      }
    });

    return { activeCount, existingCount, inactiveCount, selectedActiveCount };
  }, [prices, selections]);

  const allSelected = stats.activeCount > 0 && stats.selectedActiveCount >= stats.activeCount;
  const isIndeterminate = stats.selectedActiveCount > 0 && !allSelected;

  useEffect(() => {
    if (inputRef.current) {
      inputRef.current.indeterminate = isIndeterminate;
    }
  }, [isIndeterminate]);

  return (
    <div className="bg-slate-800/50 px-4 py-3 border-b border-slate-700 flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
      <div className="flex items-start gap-3">
        <input
          ref={inputRef}
          type="checkbox"
          className="mt-1 h-4 w-4 rounded border-slate-600 bg-slate-900 text-emerald-400 focus:ring-2 focus:ring-emerald-400/60"
          checked={allSelected}
          onChange={(event) => { onSelectionChange(event.target.checked); }}
        />
        <div>
          <h3 className="font-medium text-white">{product.product_name}</h3>
          <p className="text-xs text-slate-400">{product.product_id}</p>
        </div>
      </div>
      <div className="flex flex-wrap gap-2 text-xs text-slate-400">
        <span>{prices.length} prices</span>
        <span className="text-emerald-300">{stats.activeCount} active</span>
        <span className="text-amber-300">{stats.existingCount} existing</span>
        {stats.inactiveCount > 0 && <span className="text-red-300">{stats.inactiveCount} inactive</span>}
      </div>
    </div>
  );
}

interface ProductSelectCardProps {
  product: StripeProductWithPrices;
  selected: boolean;
  onSelect: () => void;
}

function ProductSelectCard({ product, selected, onSelect }: ProductSelectCardProps) {
  const stats = useMemo(() => {
    let activeCount = 0;
    let inactiveCount = 0;
    let existingCount = 0;

    product.prices.forEach((price) => {
      if (price.active) {
        activeCount += 1;
      } else {
        inactiveCount += 1;
      }
      if (price.exists_locally) {
        existingCount += 1;
      }
    });

    return { activeCount, inactiveCount, existingCount };
  }, [product.prices]);

  return (
    <button
      type="button"
      onClick={onSelect}
      className={cn(
        'rounded-lg border px-4 py-3 text-left transition',
        selected
          ? 'border-emerald-400/60 bg-emerald-500/10'
          : 'border-slate-700 bg-slate-900/40 hover:border-slate-500'
      )}
    >
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="font-medium text-white">{product.product_name}</h3>
            {product.is_current_bundle && (
              <span className="inline-flex items-center rounded px-2 py-0.5 text-xs font-medium bg-slate-700/60 text-slate-200">
                Current bundle
              </span>
            )}
          </div>
          <p className="text-xs text-slate-400 font-mono">{product.product_id}</p>
        </div>
        {selected && <Check className="h-4 w-4 text-emerald-400" />}
      </div>
      <div className="mt-2 flex flex-wrap gap-2 text-xs text-slate-400">
        <span>{product.prices.length} prices</span>
        <span className="text-emerald-300">{stats.activeCount} active</span>
        {stats.existingCount > 0 && <span className="text-amber-300">{stats.existingCount} existing</span>}
        {stats.inactiveCount > 0 && <span className="text-red-300">{stats.inactiveCount} inactive</span>}
      </div>
      {selected && (
        <div className="mt-2 text-xs text-emerald-300">Selected</div>
      )}
    </button>
  );
}
