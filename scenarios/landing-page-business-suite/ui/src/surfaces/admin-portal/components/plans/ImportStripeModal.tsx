import { AlertCircle, Check, Download, Loader2, X } from 'lucide-react';
import { Button } from '../../../../shared/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../../../../shared/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../../../../shared/ui/select';
import { Callout } from '../Callout';
import type { UseStripeImportReturn, ImportAction } from '../../hooks/useStripeImport';

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
    handleSelectionChange,
    selectAll,
    importing,
    importResult,
    handleImport,
    resetImportResult,
  } = stripeImport;

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

  const countByAction = (action: ImportAction) => {
    return Object.values(selections).filter((a) => a === action).length;
  };

  return (
    <Dialog open={isModalOpen} onOpenChange={(open: boolean) => !open && closeModal()}>
      <DialogContent className="max-w-2xl max-h-[80vh] flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Download className="h-5 w-5" />
            Import Plans from Stripe
          </DialogTitle>
          <DialogDescription>
            Select which prices to import from your Stripe account into the local plan configuration.
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
              {/* Summary stats */}
              <div className="grid grid-cols-3 gap-4 text-center">
                <div className="p-3 rounded-lg bg-slate-800/50">
                  <div className="text-2xl font-bold text-white">{preview.total_prices}</div>
                  <div className="text-xs text-slate-400">Total Prices</div>
                </div>
                <div className="p-3 rounded-lg bg-emerald-500/10">
                  <div className="text-2xl font-bold text-emerald-400">{preview.new_count}</div>
                  <div className="text-xs text-slate-400">New</div>
                </div>
                <div className="p-3 rounded-lg bg-amber-500/10">
                  <div className="text-2xl font-bold text-amber-400">{preview.conflict_count}</div>
                  <div className="text-xs text-slate-400">Already Exists</div>
                </div>
              </div>

              {/* Bulk actions */}
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => selectAll('import')}
                  className="text-xs"
                >
                  Select All: Import
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => selectAll('skip')}
                  className="text-xs"
                >
                  Select All: Skip
                </Button>
              </div>

              {/* Products and prices list */}
              <div className="space-y-4">
                {preview.products.map((product) => (
                  <div key={product.product_id} className="border border-slate-700 rounded-lg overflow-hidden">
                    <div className="bg-slate-800/50 px-4 py-2 border-b border-slate-700">
                      <h3 className="font-medium text-white">{product.product_name}</h3>
                      <p className="text-xs text-slate-400">{product.product_id}</p>
                    </div>
                    <div className="divide-y divide-slate-700/50">
                      {product.prices.map((price) => (
                        <div
                          key={price.price_id}
                          className="px-4 py-3 flex items-center justify-between gap-4"
                        >
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2">
                              <span className="font-mono text-sm text-white">
                                {formatPrice(price.amount_cents, price.currency)}
                                <span className="text-slate-400">{formatInterval(price.interval)}</span>
                              </span>
                              {price.exists_locally && (
                                <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-amber-500/20 text-amber-300">
                                  Exists
                                </span>
                              )}
                              {!price.active && (
                                <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-red-500/20 text-red-300">
                                  Inactive
                                </span>
                              )}
                            </div>
                            <p className="text-xs text-slate-500 font-mono truncate">{price.price_id}</p>
                          </div>
                          <Select
                            value={selections[price.price_id] || 'skip'}
                            onValueChange={(value) => handleSelectionChange(price.price_id, value as ImportAction)}
                          >
                            <SelectTrigger className="w-32">
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="import">Import</SelectItem>
                              {price.exists_locally && (
                                <SelectItem value="overwrite">Overwrite</SelectItem>
                              )}
                              <SelectItem value="skip">Skip</SelectItem>
                            </SelectContent>
                          </Select>
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
              </div>
            </>
          )}
        </div>

        <DialogFooter className="border-t border-slate-700 pt-4">
          <div className="flex items-center gap-2 text-sm text-slate-400 mr-auto">
            <Check className="h-4 w-4 text-emerald-400" />
            <span>{countByAction('import')} to import</span>
            {countByAction('overwrite') > 0 && (
              <>
                <span className="text-slate-600">|</span>
                <AlertCircle className="h-4 w-4 text-amber-400" />
                <span>{countByAction('overwrite')} to overwrite</span>
              </>
            )}
          </div>
          <Button variant="outline" onClick={closeModal} disabled={importing}>
            Cancel
          </Button>
          <Button
            onClick={handleImport}
            disabled={importing || loading || countByAction('import') + countByAction('overwrite') === 0}
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
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
