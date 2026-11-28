import { useState, useEffect, useCallback } from 'react';
import type { VariantSEOConfig, SiteBranding } from '../../../shared/api';
import { loadVariantSEOConfig, saveVariantSEOConfig } from '../controllers/seoController';

interface UseVariantSEOEditorProps {
  variantSlug: string;
  siteBranding?: SiteBranding | null;
  onSave?: () => void;
}

interface UseVariantSEOEditorReturn {
  seoConfig: VariantSEOConfig;
  loading: boolean;
  saving: boolean;
  error: string | null;
  success: boolean;
  fetchSEO: () => Promise<void>;
  handleSave: () => Promise<void>;
  updateField: <K extends keyof VariantSEOConfig>(field: K, value: VariantSEOConfig[K]) => void;
}

/**
 * Hook for managing SEO configuration state and operations for a variant.
 * Extracts state management logic from VariantSEOEditor component.
 */
export function useVariantSEOEditor({
  variantSlug,
  siteBranding,
  onSave,
}: UseVariantSEOEditorProps): UseVariantSEOEditorReturn {
  const [seoConfig, setSeoConfig] = useState<VariantSEOConfig>({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const fetchSEO = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const editableConfig = await loadVariantSEOConfig(variantSlug, siteBranding);
      setSeoConfig(editableConfig);
    } catch {
      setError('Failed to load SEO settings');
    } finally {
      setLoading(false);
    }
  }, [variantSlug, siteBranding]);

  useEffect(() => {
    void fetchSEO();
  }, [fetchSEO]);

  const handleSave = useCallback(async () => {
    setSaving(true);
    setError(null);
    setSuccess(false);

    try {
      await saveVariantSEOConfig(variantSlug, seoConfig);
      setSuccess(true);
      setTimeout(() => { setSuccess(false); }, 3000);
      onSave?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save');
    } finally {
      setSaving(false);
    }
  }, [variantSlug, seoConfig, onSave]);

  const updateField = useCallback(<K extends keyof VariantSEOConfig>(
    field: K,
    value: VariantSEOConfig[K]
  ) => {
    setSeoConfig((prev) => ({ ...prev, [field]: value }));
    setSuccess(false);
  }, []);

  return {
    seoConfig,
    loading,
    saving,
    error,
    success,
    fetchSEO,
    handleSave,
    updateField,
  };
}
