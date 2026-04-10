import { useState, useCallback } from 'react';
import {
  DEFAULT_AGENT_FORM,
  DEFAULT_SCENARIO_ID,
  validateBrief,
  parseAssets,
  submitAgentCustomization,
  type AgentFormState,
  type AgentResult,
} from '../services/agent.service';

export interface UseAgentFormReturn {
  // Data state
  form: AgentFormState;
  result: AgentResult | null;

  // UI state
  submitting: boolean;
  error: string | null;
  validationError: { message: string; title?: string } | null;

  // Actions
  setBrief: (brief: string) => void;
  setAssets: (assets: string) => void;
  setPreview: (preview: boolean) => void;
  handleSubmit: () => Promise<{ success: boolean; message?: string }>;
  resetForm: () => void;
  clearResult: () => void;
  clearError: () => void;
  clearValidationError: () => void;
}

/**
 * Reactive hook for agent customization form management
 *
 * Provides state and handlers for:
 * - Managing form state (brief, assets, preview)
 * - Submitting customization requests
 * - Handling results and errors
 */
export function useAgentForm(): UseAgentFormReturn {
  // Data state
  const [form, setForm] = useState<AgentFormState>(DEFAULT_AGENT_FORM);
  const [result, setResult] = useState<AgentResult | null>(null);

  // UI state
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [validationError, setValidationError] = useState<{
    message: string;
    title?: string;
  } | null>(null);

  /**
   * Update brief field
   */
  const setBrief = useCallback((brief: string) => {
    setForm((prev) => ({ ...prev, brief }));
    // Clear validation error when user starts typing
    setValidationError(null);
  }, []);

  /**
   * Update assets field
   */
  const setAssets = useCallback((assets: string) => {
    setForm((prev) => ({ ...prev, assets }));
  }, []);

  /**
   * Update preview field
   */
  const setPreview = useCallback((preview: boolean) => {
    setForm((prev) => ({ ...prev, preview }));
  }, []);

  /**
   * Submit the customization request
   */
  const handleSubmit = useCallback(async (): Promise<{
    success: boolean;
    message?: string;
  }> => {
    // Validate brief
    const briefError = validateBrief(form.brief);
    if (briefError) {
      setValidationError({ message: briefError, title: 'Missing Input' });
      return { success: false, message: briefError };
    }

    // Clear validation error if valid
    setValidationError(null);

    try {
      setSubmitting(true);
      setError(null);

      const assetList = parseAssets(form.assets);

      const response = await submitAgentCustomization(
        DEFAULT_SCENARIO_ID,
        form.brief.trim(),
        assetList,
        form.preview
      );

      setResult(response);
      return { success: true };
    } catch (err) {
      const message =
        err instanceof Error ? err.message : 'Failed to trigger agent customization';
      setError(message);
      console.error('Agent customization error:', err);
      return { success: false, message };
    } finally {
      setSubmitting(false);
    }
  }, [form]);

  /**
   * Reset form to default values
   */
  const resetForm = useCallback(() => {
    setForm(DEFAULT_AGENT_FORM);
    setValidationError(null);
    setError(null);
  }, []);

  /**
   * Clear the result (to show form again)
   */
  const clearResult = useCallback(() => {
    setResult(null);
    resetForm();
  }, [resetForm]);

  /**
   * Clear error state
   */
  const clearError = useCallback(() => {
    setError(null);
  }, []);

  /**
   * Clear validation error state
   */
  const clearValidationError = useCallback(() => {
    setValidationError(null);
  }, []);

  return {
    // Data state
    form,
    result,

    // UI state
    submitting,
    error,
    validationError,

    // Actions
    setBrief,
    setAssets,
    setPreview,
    handleSubmit,
    resetForm,
    clearResult,
    clearError,
    clearValidationError,
  };
}
