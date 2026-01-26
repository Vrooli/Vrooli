import { useCallback, useState } from 'react';
import {
  getStripeImportPreview,
  importStripePlans,
  type StripeImportPreview,
  type ImportPlanSelection,
  type StripeImportResult,
} from '../../../shared/api/billing';

export type ImportAction = 'import' | 'overwrite' | 'skip';

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
  selections: Record<string, ImportAction>;
  handleSelectionChange: (priceId: string, action: ImportAction) => void;
  selectAll: (action: ImportAction) => void;

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
  const [selections, setSelections] = useState<Record<string, ImportAction>>({});
  const [importing, setImporting] = useState(false);
  const [importResult, setImportResult] = useState<StripeImportResult | null>(null);

  const openModal = useCallback(async () => {
    setIsModalOpen(true);
    setLoading(true);
    setError(null);
    setSelections({});
    setImportResult(null);

    try {
      const data = await getStripeImportPreview();
      setPreview(data);

      // Initialize selections - default new prices to 'import', existing to 'skip'
      const initialSelections: Record<string, ImportAction> = {};
      data.products.forEach((product) => {
        product.prices.forEach((price) => {
          initialSelections[price.price_id] = price.exists_locally ? 'skip' : 'import';
        });
      });
      setSelections(initialSelections);
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

  const handleSelectionChange = useCallback((priceId: string, action: ImportAction) => {
    setSelections((prev) => ({
      ...prev,
      [priceId]: action,
    }));
  }, []);

  const selectAll = useCallback((action: ImportAction) => {
    if (!preview) return;
    const newSelections: Record<string, ImportAction> = {};
    preview.products.forEach((product) => {
      product.prices.forEach((price) => {
        newSelections[price.price_id] = action;
      });
    });
    setSelections(newSelections);
  }, [preview]);

  const handleImport = useCallback(async () => {
    const selectionsList: ImportPlanSelection[] = Object.entries(selections).map(([priceId, action]) => ({
      price_id: priceId,
      action,
    }));

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
      const updatedSelections: Record<string, ImportAction> = {};
      updatedPreview.products.forEach((product) => {
        product.prices.forEach((price) => {
          updatedSelections[price.price_id] = price.exists_locally ? 'skip' : 'import';
        });
      });
      setSelections(updatedSelections);

      // Call completion callback
      onImportComplete?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Import failed');
    } finally {
      setImporting(false);
    }
  }, [selections, onImportComplete]);

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
    handleSelectionChange,
    selectAll,
    importing,
    importResult,
    handleImport,
    resetImportResult,
  };
}
