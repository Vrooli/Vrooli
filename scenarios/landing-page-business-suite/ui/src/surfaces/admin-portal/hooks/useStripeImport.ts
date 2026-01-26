import { useCallback, useMemo, useState } from 'react';
import {
  getStripeImportPreview,
  importStripePlans,
  type StripeImportPreview,
  type ImportPlanSelection,
  type StripeImportResult,
  type StripePriceImport,
} from '../../../shared/api/billing';

const buildSelectionMap = (
  preview: StripeImportPreview,
  selector: (price: StripePriceImport) => boolean
) => {
  const selections: Record<string, boolean> = {};
  preview.products.forEach((product) => {
    product.prices.forEach((price) => {
      selections[price.price_id] = selector(price);
    });
  });
  return selections;
};

const defaultSelectionsForPreview = (preview: StripeImportPreview) =>
  buildSelectionMap(preview, (price) => price.active && !price.exists_locally);

export interface UseStripeImportReturn {
  // Modal state
  isModalOpen: boolean;
  openModal: () => void;
  closeModal: () => void;

  // Preview data
  preview: StripeImportPreview | null;
  loading: boolean;
  error: string | null;

  // Selections
  selections: Record<string, boolean>;
  setPriceSelected: (priceId: string, selected: boolean) => void;
  setSelectionsForPrices: (priceIds: string[], selected: boolean) => void;
  selectNew: () => void;
  selectConflicts: () => void;
  clearSelections: () => void;

  // Import
  importing: boolean;
  importResult: StripeImportResult | null;
  handleImport: () => Promise<void>;
  resetImportResult: () => void;
}

/**
 * Hook for managing Stripe import modal state and operations.
 */
export function useStripeImport(onImportComplete?: () => void): UseStripeImportReturn {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [preview, setPreview] = useState<StripeImportPreview | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selections, setSelections] = useState<Record<string, boolean>>({});
  const [importing, setImporting] = useState(false);
  const [importResult, setImportResult] = useState<StripeImportResult | null>(null);

  const priceIndex = useMemo(() => {
    const index = new Map<string, StripePriceImport>();
    preview?.products.forEach((product) => {
      product.prices.forEach((price) => {
        index.set(price.price_id, price);
      });
    });
    return index;
  }, [preview]);

  const openModal = useCallback(async () => {
    setIsModalOpen(true);
    setLoading(true);
    setError(null);
    setSelections({});
    setImportResult(null);

    try {
      const data = await getStripeImportPreview();
      setPreview(data);
      setSelections(defaultSelectionsForPreview(data));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load Stripe products');
    } finally {
      setLoading(false);
    }
  }, []);

  const closeModal = useCallback(() => {
    setIsModalOpen(false);
    setPreview(null);
    setError(null);
    setSelections({});
    setImportResult(null);
  }, []);

  const setPriceSelected = useCallback((priceId: string, selected: boolean) => {
    setSelections((prev) => ({
      ...prev,
      [priceId]: selected,
    }));
  }, []);

  const setSelectionsForPrices = useCallback((priceIds: string[], selected: boolean) => {
    setSelections((prev) => {
      const next = { ...prev };
      priceIds.forEach((priceId) => {
        next[priceId] = selected;
      });
      return next;
    });
  }, []);

  const selectNew = useCallback(() => {
    if (!preview) return;
    setSelections(buildSelectionMap(preview, (price) => price.active && !price.exists_locally));
  }, [preview]);

  const selectConflicts = useCallback(() => {
    if (!preview) return;
    setSelections(buildSelectionMap(preview, (price) => price.active && price.exists_locally));
  }, [preview]);

  const clearSelections = useCallback(() => {
    if (!preview) return;
    setSelections(buildSelectionMap(preview, () => false));
  }, [preview]);

  const handleImport = useCallback(async () => {
    const selectionsList: ImportPlanSelection[] = Object.entries(selections).map(([priceId, selected]) => {
      const price = priceIndex.get(priceId);
      const action: ImportPlanSelection['action'] = selected
        ? price?.exists_locally
          ? 'overwrite'
          : 'import'
        : 'skip';
      return { price_id: priceId, action };
    });

    // Filter out skip actions for efficiency
    const toImport = selectionsList.filter((s) => s.action !== 'skip');
    if (toImport.length === 0) {
      setError('No prices selected for import');
      return;
    }

    setImporting(true);
    setError(null);

    try {
      const result = await importStripePlans({ selections: selectionsList });
      setImportResult(result);

      if (result.errors && result.errors.length > 0) {
        setError(`Import completed with ${result.errors.length} error(s)`);
      }

      // Refresh preview to update exists_locally flags
      const updatedPreview = await getStripeImportPreview();
      setPreview(updatedPreview);

      // Update selections for the new state
      setSelections(defaultSelectionsForPreview(updatedPreview));

      // Call completion callback
      onImportComplete?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Import failed');
    } finally {
      setImporting(false);
    }
  }, [selections, onImportComplete, priceIndex]);

  const resetImportResult = useCallback(() => {
    setImportResult(null);
  }, []);

  return {
    isModalOpen,
    openModal,
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
    resetImportResult,
  };
}
