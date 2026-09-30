import { useCallback, useEffect, useMemo, useState } from 'react';
import type { SiteBranding, Asset } from '../../../shared/api';
import {
  loadBranding,
  saveBranding,
  clearField,
  brandingToForm,
  isBrandingDirty,
  computeBrandingHealth,
  selectLogoDerivatives,
  selectFaviconDerivatives,
  selectOgDerivatives,
  formatFieldName,
  DEFAULT_BRANDING_FORM,
  type BrandingFormState,
  type BrandingHealth,
} from '../services/branding.service';

/**
 * Reactive hook for branding settings form management
 *
 * Provides state and handlers for:
 * - Form state management
 * - Loading and saving branding
 * - Image upload with derivative auto-population
 * - Dirty detection and health checks
 */
export function useBrandingForm() {
  // Branding data state
  const [branding, setBranding] = useState<SiteBranding | null>(null);
  const [form, setForm] = useState<BrandingFormState>(DEFAULT_BRANDING_FORM);
  const [originalForm, setOriginalForm] = useState<BrandingFormState>(DEFAULT_BRANDING_FORM);

  // UI state
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  /**
   * Load branding data from API
   */
  const loadBrandingData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await loadBranding();
      setBranding(data);
      const formData = brandingToForm(data);
      setForm(formData);
      setOriginalForm(formData);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load branding');
    } finally {
      setLoading(false);
    }
  }, []);

  // Initial load
  useEffect(() => {
    void loadBrandingData();
  }, [loadBrandingData]);

  /**
   * Handle form field input changes
   */
  const handleInput = useCallback(
    (field: keyof BrandingFormState) =>
      (event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
        setForm((prev) => ({ ...prev, [field]: event.target.value }));
        setSuccessMessage(null);
      },
    []
  );

  /**
   * Handle direct value change for a field
   */
  const handleFieldChange = useCallback(
    (field: keyof BrandingFormState, value: string | boolean) => {
      setForm((prev) => ({ ...prev, [field]: value }));
      setSuccessMessage(null);
    },
    []
  );

  /**
   * Handle image URL changes
   */
  const handleImageChange = useCallback(
    (field: keyof BrandingFormState) => (url: string | null) => {
      setForm((prev) => ({ ...prev, [field]: url ?? '' }));
      setSuccessMessage(null);
    },
    []
  );

  /**
   * Apply logo derivatives when a logo image is uploaded
   */
  const applyLogoDerivatives = useCallback((asset: Asset) => {
    setForm((prev) => {
      const derivatives = selectLogoDerivatives(asset, prev);
      return {
        ...prev,
        logo_url: derivatives.logo_url,
        logo_icon_url: derivatives.logo_icon_url,
        favicon_url: derivatives.favicon_url,
        apple_touch_icon_url: derivatives.apple_touch_icon_url,
      };
    });
    setSuccessMessage(null);
  }, []);

  /**
   * Apply favicon derivatives when a favicon image is uploaded
   */
  const applyFaviconDerivatives = useCallback((asset: Asset) => {
    setForm((prev) => {
      const derivatives = selectFaviconDerivatives(asset, prev);
      return {
        ...prev,
        favicon_url: derivatives.favicon_url,
        apple_touch_icon_url: derivatives.apple_touch_icon_url,
      };
    });
    setSuccessMessage(null);
  }, []);

  /**
   * Apply OG image derivatives when an OG image is uploaded
   */
  const applyOgDerivatives = useCallback((asset: Asset) => {
    const derivatives = selectOgDerivatives(asset);
    setForm((prev) => ({
      ...prev,
      default_og_image_url: derivatives.default_og_image_url,
    }));
    setSuccessMessage(null);
  }, []);

  /**
   * Clear a specific branding field via API
   */
  const handleClearField = useCallback(async (field: keyof BrandingFormState) => {
    try {
      const data = await clearField(field);
      setBranding(data);
      const formData = brandingToForm(data);
      setForm(formData);
      setOriginalForm(formData);
      setSuccessMessage(`Cleared ${formatFieldName(field)}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to clear field');
    }
  }, []);

  /**
   * Save branding form
   */
  const handleSubmit = useCallback(
    async (event: React.FormEvent) => {
      event.preventDefault();
      setSaving(true);
      setError(null);
      setSuccessMessage(null);

      try {
        const updated = await saveBranding(form, originalForm);
        setBranding(updated);
        const formData = brandingToForm(updated);
        setForm(formData);
        setOriginalForm(formData);
        setSuccessMessage('Branding updated successfully');
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to update branding');
      } finally {
        setSaving(false);
      }
    },
    [form, originalForm]
  );

  /**
   * Toggle coming soon mode
   */
  const toggleComingSoon = useCallback(() => {
    setForm((prev) => ({ ...prev, coming_soon_enabled: !prev.coming_soon_enabled }));
    setSuccessMessage(null);
  }, []);

  /**
   * Preview public landing page
   */
  const previewPublicLanding = useCallback(() => {
    window.open('/', '_blank', 'noopener,noreferrer');
  }, []);

  // Computed values
  const isDirty = useMemo(() => isBrandingDirty(form, originalForm), [form, originalForm]);
  const brandingHealth: BrandingHealth = useMemo(() => computeBrandingHealth(form), [form]);

  return {
    // Data state
    branding,
    form,
    originalForm,

    // UI state
    loading,
    saving,
    error,
    successMessage,

    // Computed
    isDirty,
    brandingHealth,

    // Actions
    loadBrandingData,
    handleInput,
    handleFieldChange,
    handleImageChange,
    applyLogoDerivatives,
    applyFaviconDerivatives,
    applyOgDerivatives,
    handleClearField,
    handleSubmit,
    toggleComingSoon,
    previewPublicLanding,
  };
}
