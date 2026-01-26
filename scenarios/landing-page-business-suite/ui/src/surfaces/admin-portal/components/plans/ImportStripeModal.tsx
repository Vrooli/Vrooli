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

type PriceFilter = 'all' | 'new' | 'conflicts' | 'inactive';
const PRICE_FILTERS: Array<{ key: PriceFilter; label: string }> = [
  { key: 'all', label: 'All' },
  { key: 'new', label: 'New' },
  { key: 'conflicts', label: 'Conflicts' },
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
    selections,
    setPriceSelected,
    setSelectionsForPrices,
    selectNew,
    selectConflicts,
    clearSelections,
    importing,
    importResult,
    handleImport,
  } = stripeImport;

  const [filter, setFilter] = useState<PriceFilter>('all');
  const [query, setQuery] = useState('');
  const [confirmOverwrite, setConfirmOverwrite] = useState(false);

  const formatPrice = (amountCents: number, currency: string) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: currency.toUpperCase(),
    }).format(amountCents / 100);
  };

  const formatInterval = (interval?: string) => {
    if (!interval) return 'one-time';
    return interval === 'month' ? '/mo' : interval === 'year' ? '/yr' : `/${interval}`;
  };

  const selectionStats = useMemo(() => {
    if (!preview) {
      return {
        selected: 0,
        selectedConflicts: 0,
        selectedNew: 0,
        selectedInactive: 0,
      };
    }

    let selected = 0;
    let selectedConflicts = 0;
    let selectedNew = 0;
    let selectedInactive = 0;

    preview.products.forEach((product) => {
      product.prices.forEach((price) => {
        if (!selections[price.price_id]) return;
        selected += 1;
        if (price.exists_locally) selectedConflicts += 1;
        if (!price.exists_locally) selectedNew += 1;
        if (!price.active) selectedInactive += 1;
      });
    });

    return { selected, selectedConflicts, selectedNew, selectedInactive };
  }, [preview, selections]);

  useEffect(() => {
    if (!isModalOpen) {
      setFilter('all');
      setQuery('');
      setConfirmOverwrite(false);
    }
  }, [isModalOpen]);

  useEffect(() => {
    if (selectionStats.selectedConflicts === 0) {
      setConfirmOverwrite(false);
    }
  }, [selectionStats.selectedConflicts]);

  const filteredProducts = useMemo(() => {
    if (!preview) return [];
    const normalizedQuery = query.trim().toLowerCase();

    return preview.products
      .map((product) => {
        const productText = `${product.product_name} ${product.product_id}`.toLowerCase();
        const productMatch = normalizedQuery ? productText.includes(normalizedQuery) : false;
        const prices = product.prices.filter((price) => {
          const matchesFilter = (() => {
            if (filter === 'new') return !price.exists_locally && price.active;
            if (filter === 'conflicts') return price.exists_locally;
            if (filter === 'inactive') return !price.active;
            return true;
          })();

          if (!matchesFilter) return false;
          if (!normalizedQuery) return true;
          const priceText = `${price.price_id} ${price.lookup_key ?? ''}`.toLowerCase();
          return productMatch || priceText.includes(normalizedQuery);
        });

        return { product, prices };
      })
      .filter((entry) => entry.prices.length > 0);
  }, [preview, filter, query]);

  const requiresConfirm = selectionStats.selectedConflicts > 0;
  const canImport = selectionStats.selected > 0 && (!requiresConfirm || confirmOverwrite);

  return (
    <Dialog open={isModalOpen} onOpenChange={(open: boolean) => !open && closeModal()}>
      <DialogContent className="max-w-3xl max-h-[80vh] flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Download className="h-5 w-5" />
            Import Plans from Stripe
          </DialogTitle>
          <DialogDescription>
            Select the prices you want to import into the local plan catalog. Conflicts will overwrite local plan data
            when selected.
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
              message={`Imported: ${importResult.imported} | Overwritten: ${importResult.overwritten} | Skipped: ${importResult.skipped}${
                importResult.errors?.length ? ` | Errors: ${importResult.errors.length}` : ''
              }`}
            />
          )}

          {preview && !loading && (
            <>
              <div className="rounded-lg border border-slate-700/60 bg-slate-900/50 p-4">
                <div className="grid grid-cols-2 gap-3 text-center md:grid-cols-4">
                  <div className="rounded-lg bg-slate-800/50 p-3">
                    <div className="text-2xl font-bold text-white">{preview.total_prices}</div>
                    <div className="text-xs text-slate-400">Total Prices</div>
                  </div>
                  <div className="rounded-lg bg-emerald-500/10 p-3">
                    <div className="text-2xl font-bold text-emerald-400">{preview.new_count}</div>
                    <div className="text-xs text-slate-400">New</div>
                  </div>
                  <div className="rounded-lg bg-amber-500/10 p-3">
                    <div className="text-2xl font-bold text-amber-400">{preview.conflict_count}</div>
                    <div className="text-xs text-slate-400">Conflicts</div>
                  </div>
                  <div className="rounded-lg bg-slate-800/50 p-3">
                    <div className="text-2xl font-bold text-white">
                      {preview.total_prices - preview.new_count - preview.conflict_count}
                    </div>
                    <div className="text-xs text-slate-400">Unchanged</div>
                  </div>
                </div>
                <div className="mt-3 flex flex-wrap items-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={selectNew}
                    className="text-xs"
                  >
                    Select New
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={selectConflicts}
                    className="text-xs"
                  >
                    Select Conflicts
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

              {preview.conflict_count > 0 && (
                <Callout
                  type="warning"
                  message="Conflicts will overwrite local plan data when selected. Use this only when Stripe is the source of truth."
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
                        onClick={() => setFilter(option.key)}
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
                    placeholder="Search products or price IDs"
                    value={query}
                    onChange={(event) => setQuery(event.target.value)}
                  />
                </div>
              </div>

              <div className="space-y-4">
                {filteredProducts.map(({ product, prices }) => (
                  <div key={product.product_id} className="border border-slate-700 rounded-lg overflow-hidden">
                    <ProductHeader
                      product={product}
                      prices={prices}
                      selections={selections}
                      onSelectionChange={(selected) => {
                        const priceIds = prices
                          .filter((price) => price.active)
                          .map((price) => price.price_id);
                        setSelectionsForPrices(priceIds, selected);
                      }}
                    />
                    <div className="divide-y divide-slate-700/50">
                      {prices.map((price) => (
                        <div
                          key={price.price_id}
                          className={`px-4 py-3 flex items-start gap-4 ${price.active ? '' : 'opacity-70'}`}
                        >
                          <input
                            type="checkbox"
                            className="mt-1 h-4 w-4 rounded border-slate-600 bg-slate-900 text-emerald-400 focus:ring-2 focus:ring-emerald-400/60"
                            checked={Boolean(selections[price.price_id])}
                            onChange={(event) => setPriceSelected(price.price_id, event.target.checked)}
                          />
                          <div className="flex-1 min-w-0 space-y-1">
                            <div className="flex flex-wrap items-center gap-2">
                              <span className="font-mono text-sm text-white">
                                {formatPrice(price.amount_cents, price.currency)}
                                <span className="text-slate-400">{formatInterval(price.interval)}</span>
                              </span>
                              {price.exists_locally && (
                                <span className="inline-flex items-center rounded px-2 py-0.5 text-xs font-medium bg-amber-500/20 text-amber-300">
                                  Conflict
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
                                Will overwrite local plan data
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
                ))}

                {preview.products.length === 0 && (
                  <div className="text-center py-8 text-slate-400">
                    No products found in your Stripe account.
                  </div>
                )}
                {preview.products.length > 0 && filteredProducts.length === 0 && (
                  <div className="text-center py-8 text-slate-400">
                    No prices match your filters.
                  </div>
                )}
              </div>
            </>
          )}
        </div>

        <DialogFooter className="border-t border-slate-700 pt-4 gap-3 flex-col sm:flex-row sm:items-center sm:justify-between">
          <div className="flex flex-col gap-2 text-sm text-slate-400">
            <div className="flex items-center gap-2">
              <Check className="h-4 w-4 text-emerald-400" />
              <span>{selectionStats.selected} selected</span>
              {selectionStats.selectedConflicts > 0 && (
                <>
                  <span className="text-slate-600">|</span>
                  <AlertCircle className="h-4 w-4 text-amber-400" />
                  <span>{selectionStats.selectedConflicts} conflicts will overwrite</span>
                </>
              )}
            </div>
            {requiresConfirm && (
              <label className="flex items-start gap-2 text-xs text-amber-200">
                <input
                  type="checkbox"
                  className="mt-0.5 h-4 w-4 rounded border-amber-400/60 bg-slate-900 text-amber-400 focus:ring-2 focus:ring-amber-400/60"
                  checked={confirmOverwrite}
                  onChange={(event) => setConfirmOverwrite(event.target.checked)}
                />
                <span>I understand selected conflicts will overwrite local plan data.</span>
              </label>
            )}
          </div>
          <div className="flex items-center gap-2">
            <Button variant="outline" onClick={closeModal} disabled={importing}>
              Cancel
            </Button>
            <Button
              onClick={handleImport}
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
    let newCount = 0;
    let conflictCount = 0;
    let inactiveCount = 0;
    let selectedActiveCount = 0;
    let selectableCount = 0;

    prices.forEach((price) => {
      if (!price.active) {
        inactiveCount += 1;
      } else {
        selectableCount += 1;
      }
      if (price.exists_locally) {
        conflictCount += 1;
      } else {
        newCount += 1;
      }
      if (selections[price.price_id] && price.active) {
        selectedActiveCount += 1;
      }
    });

    return { newCount, conflictCount, inactiveCount, selectedActiveCount, selectableCount };
  }, [prices, selections]);

  const allSelected = stats.selectableCount > 0 && stats.selectedActiveCount >= stats.selectableCount;
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
          onChange={(event) => onSelectionChange(event.target.checked)}
        />
        <div>
          <h3 className="font-medium text-white">{product.product_name}</h3>
          <p className="text-xs text-slate-400">{product.product_id}</p>
        </div>
      </div>
      <div className="flex flex-wrap gap-2 text-xs text-slate-400">
        <span>{prices.length} prices</span>
        <span className="text-emerald-300">{stats.newCount} new</span>
        <span className="text-amber-300">{stats.conflictCount} conflicts</span>
        {stats.inactiveCount > 0 && <span className="text-red-300">{stats.inactiveCount} inactive</span>}
      </div>
    </div>
  );
}
