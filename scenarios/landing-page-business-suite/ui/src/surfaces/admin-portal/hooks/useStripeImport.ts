import { useCallback, useMemo, useState } from 'react';
import {
  getStripeImportPreview,
  importStripePlans,
  type StripeImportPreview,
  type ImportPlanSelection,
  type StripeImportResult,
  type StripePriceImport,
  type StripeProductWithPrices,
} from '../../../shared/api/billing';
import { getApiErrorMessage } from '../../../shared/api';

const buildSelectionMapForProduct = (
  product: StripeProductWithPrices,
  selector: (price: StripePriceImport) => boolean
) => {
  const selections: Record<string, boolean> = {};
  product.prices.forEach((price) => {
    selections[price.price_id] = selector(price);
  });
  return selections;
};

const defaultSelectionsForProduct = (product: StripeProductWithPrices) =>
  buildSelectionMapForProduct(product, (price) => price.active);

const resolveSelectedProductId = (preview: StripeImportPreview, preferredId?: string) => {
  const products = preview.products;
  if (preferredId && products.some((product) => product.product_id === preferredId)) {
    return preferredId;
  }
  const bundleId = preview.bundle_product_id;
  if (bundleId && products.some((product) => product.product_id === bundleId)) {
    return bundleId;
  }
  if (products.length === 1) {
    const only = products[0];
    if (only) {
      return only.product_id;
    }
  }
  return '';
};

export interface UseStripeImportReturn {
  // Modal state
  isModalOpen: boolean;
  openModal: () => void;
  closeModal: () => void;

  // Preview data
  preview: StripeImportPreview | null;
  loading: boolean;
  error: string | null;

  // Product selection
  selectedProductId: string;
  selectedProduct: StripeProductWithPrices | null;
  selectProduct: (productId: string) => void;

  // Selections
  selections: Record<string, boolean>;
  setPriceSelected: (priceId: string, selected: boolean) => void;
  setSelectionsForPrices: (priceIds: string[], selected: boolean) => void;
  selectActive: () => void;
  selectExisting: () => void;
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
  const [selectedProductId, setSelectedProductId] = useState('');
  const [selections, setSelections] = useState<Record<string, boolean>>({});
  const [importing, setImporting] = useState(false);
  const [importResult, setImportResult] = useState<StripeImportResult | null>(null);

  const selectedProduct = useMemo(() => {
    if (!preview || !selectedProductId) return null;
    return preview.products.find((product) => product.product_id === selectedProductId) ?? null;
  }, [preview, selectedProductId]);

  const priceIndex = useMemo(() => {
    const index = new Map<string, StripePriceImport>();
    selectedProduct?.prices.forEach((price) => {
      index.set(price.price_id, price);
    });
    return index;
  }, [selectedProduct]);

  const selectProduct = useCallback(
    (productId: string) => {
      if (!preview) {
        setSelectedProductId(productId);
        setSelections({});
        return;
      }
      const product = preview.products.find((entry) => entry.product_id === productId);
      setSelectedProductId(productId);
      setImportResult(null);
      setError(null);
      if (!product) {
        setSelections({});
        return;
      }
      setSelections(defaultSelectionsForProduct(product));
    },
    [preview]
  );

  const openModal = useCallback(async () => {
    setIsModalOpen(true);
    setLoading(true);
    setError(null);
    setSelections({});
    setSelectedProductId('');
    setImportResult(null);

    try {
      const data = await getStripeImportPreview();
      setPreview(data);
      const nextProductId = resolveSelectedProductId(data);
      setSelectedProductId(nextProductId);
      if (nextProductId) {
        const product = data.products.find((entry) => entry.product_id === nextProductId);
        if (product) {
          setSelections(defaultSelectionsForProduct(product));
        }
      }
    } catch (err) {
      setError(getApiErrorMessage(err, 'Failed to load Stripe products'));
    } finally {
      setLoading(false);
    }
  }, []);

  const closeModal = useCallback(() => {
    setIsModalOpen(false);
    setPreview(null);
    setError(null);
    setSelections({});
    setSelectedProductId('');
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

  const selectActive = useCallback(() => {
    if (!selectedProduct) return;
    setSelections(buildSelectionMapForProduct(selectedProduct, (price) => price.active));
  }, [selectedProduct]);

  const selectExisting = useCallback(() => {
    if (!selectedProduct) return;
    setSelections(buildSelectionMapForProduct(selectedProduct, (price) => price.exists_locally));
  }, [selectedProduct]);

  const clearSelections = useCallback(() => {
    if (!selectedProduct) return;
    setSelections(buildSelectionMapForProduct(selectedProduct, () => false));
  }, [selectedProduct]);

  const handleImport = useCallback(async () => {
    if (!selectedProductId) {
      setError('Select a Stripe product to import');
      return;
    }

    const selectionsList: ImportPlanSelection[] = Object.entries(selections).map(([priceId, selected]) => {
      const price = priceIndex.get(priceId);
      if (!price) {
        return { price_id: priceId, action: 'skip' };
      }
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
      const result = await importStripePlans({
        bundle_product_id: selectedProductId,
        mode: 'replace',
        selections: selectionsList,
      });
      setImportResult(result);

      if (result.errors && result.errors.length > 0) {
        setError(`Import completed with ${result.errors.length} error(s)`);
      }

      // Refresh preview to update exists_locally flags
      const updatedPreview = await getStripeImportPreview();
      setPreview(updatedPreview);

      const nextProductId = resolveSelectedProductId(updatedPreview, selectedProductId);
      setSelectedProductId(nextProductId);

      // Update selections for the new state
      if (nextProductId) {
        const product = updatedPreview.products.find((entry) => entry.product_id === nextProductId);
        if (product) {
          setSelections(defaultSelectionsForProduct(product));
        } else {
          setSelections({});
        }
      } else {
        setSelections({});
      }

      // Call completion callback
      onImportComplete?.();
    } catch (err) {
      setError(getApiErrorMessage(err, 'Import failed'));
    } finally {
      setImporting(false);
    }
  }, [selections, onImportComplete, priceIndex, selectedProductId]);

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
    resetImportResult,
  };
}
