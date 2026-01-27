import { useCallback, useEffect, useRef, useState } from 'react';
import type { IDisposable, editor as MonacoEditor, Uri as MonacoUri } from 'monaco-editor';
import {
  type Variant,
  type ContentSection,
  type VariantSpace,
  type VariantAxes,
  type LandingHeaderConfig,
} from '../../../shared/api';
import {
  buildAxesSelection,
  hydrateFormFromVariant,
  loadVariantEditorData,
  loadVariantSpaceDefinition,
  loadVariantSnapshot,
  persistVariant,
  persistVariantSnapshot,
  sanitizeSlugInput,
  validateVariantForm,
  type VariantFormState,
  type VariantSnapshotPayload,
} from '../controllers/variantEditorController';
import { buildDefaultHeaderConfig, normalizeHeaderConfig } from '../../../shared/lib/headerConfig';
import { VariantSnapshotSchema } from '../../../shared/api/schemas/variants.schema';
import { safeParseJson } from '../../../shared/lib/utils';
import { rememberVariantSession } from '../../../shared/lib/adminExperience';

/**
 * Monaco schema catalog item
 */
export interface MonacoSchemaCatalogItem {
  uri: string;
  schema: unknown;
}

/**
 * Props for the useVariantForm hook
 */
export interface UseVariantFormProps {
  slug: string | undefined;
  isNew: boolean;
  monacoSchemaCatalog: MonacoSchemaCatalogItem[];
  variantSchemaUri: string;
  editorModelPath: string;
  onSuccess?: (message: string, title?: string) => void;
  onError?: (message: string) => void;
}

type MonacoApi = typeof import('monaco-editor');

/**
 * Custom hook for managing variant editor form state and operations
 */
export function useVariantForm({
  slug,
  isNew,
  monacoSchemaCatalog,
  variantSchemaUri,
  editorModelPath,
  onSuccess,
  onError,
}: UseVariantFormProps) {
  // Variant data state
  const [variant, setVariant] = useState<Variant | null>(null);
  const [sections, setSections] = useState<ContentSection[]>([]);
  const [loading, setLoading] = useState(!isNew);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [validationError, setValidationError] = useState<string | null>(null);

  // Variant space and axes state
  const [variantSpace, setVariantSpace] = useState<VariantSpace | null>(null);
  const [axesSelection, setAxesSelection] = useState<VariantAxes>({});
  const [axesSeeded, setAxesSeeded] = useState(false);

  // Form state
  const [form, setForm] = useState<VariantFormState>({
    name: '',
    slug: '',
    description: '',
    weight: 50,
  });

  // Header config state
  const [headerConfig, setHeaderConfig] = useState<LandingHeaderConfig>(() => buildDefaultHeaderConfig(''));

  // Tab state
  const [activeTab, setActiveTab] = useState<'form' | 'json'>('form');

  // Snapshot (JSON editor) state
  const [snapshotDraft, setSnapshotDraft] = useState('');
  const [snapshotError, setSnapshotError] = useState<string | null>(null);
  const [snapshotLoading, setSnapshotLoading] = useState(!isNew);
  const [snapshotSaving, setSnapshotSaving] = useState(false);
  const [schemaIssues, setSchemaIssues] = useState<string[]>([]);

  // Clipboard status
  const [copyStatus, setCopyStatus] = useState<string | null>(null);

  // Monaco markers listener ref
  const markersListener = useRef<IDisposable | null>(null);

  // Computed values
  const isJsonTab = activeTab === 'json';
  const currentSaving = isJsonTab ? snapshotSaving : saving;
  const savingLabel = isJsonTab
    ? snapshotSaving
      ? 'Saving JSON...'
      : 'Save JSON'
    : saving
    ? 'Saving...'
    : 'Save';

  /**
   * Handle Monaco editor mount
   */
  const handleEditorMount = useCallback(
    (_editor: MonacoEditor.IStandaloneCodeEditor, monaco: MonacoApi) => {
    const jsonDefaults = (monaco.languages as typeof monaco.languages & {
      json?: { jsonDefaults?: { setDiagnosticsOptions: (options: unknown) => void } };
    }).json?.jsonDefaults;

    if (jsonDefaults) {
      jsonDefaults.setDiagnosticsOptions({
        validate: true,
        allowComments: false,
        schemas: monacoSchemaCatalog.map(({ uri, schema }) => ({
          uri,
          fileMatch: uri === variantSchemaUri ? [editorModelPath] : [uri],
          schema,
        })),
      });
    }

    const uri = monaco.Uri.parse(editorModelPath);
    const refreshMarkers = () => {
      const markers = monaco.editor.getModelMarkers({ resource: uri });
      setSchemaIssues(
        markers.map(
          (marker: MonacoEditor.IMarker) =>
            `${marker.message} (line ${marker.startLineNumber}:${marker.startColumn})`
        )
      );
    };

    markersListener.current?.dispose();
    markersListener.current = monaco.editor.onDidChangeMarkers((changed: readonly MonacoUri[]) => {
      const affected = changed.some((change: MonacoUri) => change.toString() === uri.toString());
      if (affected) {
        refreshMarkers();
      }
    });

    refreshMarkers();
  }, [monacoSchemaCatalog, variantSchemaUri, editorModelPath]);

  /**
   * Update a form field
   */
  const updateFormField = useCallback(<K extends keyof VariantFormState>(field: K, value: VariantFormState[K]) => {
    if (validationError) {
      setValidationError(null);
    }
    setForm((prev) => ({ ...prev, [field]: value }));
  }, [validationError]);

  /**
   * Apply axes selection from variant space
   */
  const applyAxesSelection = useCallback((space: VariantSpace, existing?: VariantAxes) => {
    setAxesSelection(buildAxesSelection(space, existing));
  }, []);

  /**
   * Update axes selection
   */
  const updateAxesSelection = useCallback((axisId: string, value: string) => {
    if (validationError) {
      setValidationError(null);
    }
    setAxesSelection((prev) => ({ ...prev, [axisId]: value }));
  }, [validationError]);

  /**
   * Fetch variant snapshot for JSON editor
   */
  const fetchSnapshot = useCallback(async () => {
    if (!slug || isNew) return;
    try {
      setSnapshotLoading(true);
      const snapshot = await loadVariantSnapshot(slug);
      setSnapshotDraft(JSON.stringify(snapshot, null, 2));
      setSnapshotError(null);
    } catch (err) {
      console.error('Variant snapshot fetch error:', err);
      setSnapshotError(err instanceof Error ? err.message : 'Failed to load variant JSON');
    } finally {
      setSnapshotLoading(false);
    }
  }, [slug, isNew]);

  /**
   * Fetch variant data
   */
  const fetchVariant = useCallback(async () => {
    if (!slug) return;

    try {
      setLoading(true);
      const data = await loadVariantEditorData(slug);
      setAxesSeeded(false);
      setVariant(data.variant);
      setForm(hydrateFormFromVariant(data.variant));
      setAxesSelection(data.variant.axes || {});
      setSections(data.sections);
      setHeaderConfig(normalizeHeaderConfig(data.variant.header_config, data.variant.name));
      rememberVariantSession({
        slug: data.variant.slug,
        name: data.variant.name,
        surface: 'variant',
      });
      await fetchSnapshot();
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load variant');
      console.error('Variant fetch error:', err);
      setSnapshotLoading(false);
    } finally {
      setLoading(false);
    }
  }, [slug, fetchSnapshot]);

  /**
   * Save variant form
   */
  const handleSave = useCallback(async () => {
    const validationMessage = validateVariantForm({
      form,
      variantSpace,
      axesSelection,
      requireSlug: isNew,
    });

    if (validationMessage) {
      setValidationError(validationMessage);
      return { success: false, savedVariant: null };
    }

    try {
      setSaving(true);
      setValidationError(null);

      const saved = await persistVariant({
        isNew,
        slugFromRoute: slug,
        form,
        axesSelection,
        headerConfig,
      });

      if (saved && isNew) {
        rememberVariantSession({
          slug: saved.slug,
          name: saved.name,
          surface: 'variant',
        });
        onSuccess?.(`Variant "${saved.name}" created`, 'Variant created');
        return { success: true, savedVariant: saved };
      } else if (slug) {
        await fetchVariant();
        onSuccess?.('Variant settings saved', 'Changes saved');
      }

      setError(null);
      return { success: true, savedVariant: null };
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save variant');
      onError?.('Failed to save variant changes');
      console.error('Variant save error:', err);
      return { success: false, savedVariant: null };
    } finally {
      setSaving(false);
    }
  }, [form, variantSpace, axesSelection, isNew, slug, headerConfig, fetchVariant, onSuccess, onError]);

  /**
   * Save JSON snapshot
   */
  const handleSaveJson = useCallback(async () => {
    if (!slug) {
      setSnapshotError('Variant slug missing');
      return false;
    }
    try {
      setSnapshotSaving(true);
      setSnapshotError(null);

      const parsed = safeParseJson(snapshotDraft);
      if (parsed === undefined) {
        throw new SyntaxError('Invalid JSON');
      }
      const validated = VariantSnapshotSchema.safeParse(parsed);
      if (!validated.success) {
        throw new Error('Invalid JSON structure for variant snapshot');
      }
      const saved = await persistVariantSnapshot(slug, validated.data as VariantSnapshotPayload);
      setSnapshotDraft(JSON.stringify(saved, null, 2));
      await fetchVariant();
      onSuccess?.('Variant JSON applied successfully', 'JSON saved');
      return true;
    } catch (err) {
      if (err instanceof SyntaxError) {
        setSnapshotError(`Invalid JSON: ${err.message}`);
        onError?.('Invalid JSON syntax');
      } else {
        setSnapshotError(err instanceof Error ? err.message : 'Failed to save variant JSON');
        onError?.('Failed to apply variant JSON');
      }
      return false;
    } finally {
      setSnapshotSaving(false);
    }
  }, [slug, snapshotDraft, fetchVariant, onSuccess, onError]);

  /**
   * Copy schema issues to clipboard
   */
  const handleCopyIssues = useCallback(async () => {
    if (schemaIssues.length === 0) {
      setCopyStatus('No schema issues to copy');
      setTimeout(() => setCopyStatus(null), 2000);
      return;
    }
    const text = schemaIssues.join('\n');
    try {
      await navigator.clipboard.writeText(text);
      setCopyStatus('Copied schema issues');
    } catch {
      const textarea = document.createElement('textarea');
      textarea.value = text;
      textarea.style.position = 'fixed';
      textarea.style.left = '-9999px';
      document.body.appendChild(textarea);
      textarea.select();
      try {
        document.execCommand('copy');
        setCopyStatus('Copied schema issues');
      } catch {
        setCopyStatus('Copy failed (clipboard blocked)');
      } finally {
        document.body.removeChild(textarea);
      }
    } finally {
      setTimeout(() => setCopyStatus(null), 2000);
    }
  }, [schemaIssues]);

  /**
   * Copy schema to clipboard
   */
  const handleCopySchema = useCallback(async (schema: unknown) => {
    try {
      await navigator.clipboard.writeText(JSON.stringify(schema, null, 2));
      setCopyStatus('Schema copied');
    } catch (err) {
      setCopyStatus('Copy failed');
      console.error('Schema copy failed', err);
    } finally {
      setTimeout(() => setCopyStatus(null), 2000);
    }
  }, []);

  // Initial load effect
  useEffect(() => {
    if (!isNew && slug) {
      fetchVariant();
    }
  }, [isNew, slug, fetchVariant]);

  // Load variant space
  useEffect(() => {
    const fetchVariantSpaceData = async () => {
      try {
        const space = await loadVariantSpaceDefinition();
        setVariantSpace(space);
      } catch (err) {
        console.error('Variant space fetch error:', err);
        setError(err instanceof Error ? err.message : 'Failed to load variant axes');
      }
    };
    fetchVariantSpaceData();
  }, []);

  // Seed axes selection when variant space is loaded
  useEffect(() => {
    if (!variantSpace || axesSeeded) return;

    if (isNew) {
      applyAxesSelection(variantSpace);
      setAxesSeeded(true);
      return;
    }

    if (variant?.axes) {
      applyAxesSelection(variantSpace, variant.axes);
      setAxesSeeded(true);
    }
  }, [variantSpace, variant, isNew, axesSeeded, applyAxesSelection]);

  // Cleanup markers listener
  useEffect(() => {
    return () => {
      markersListener.current?.dispose();
    };
  }, []);

  return {
    // Data state
    variant,
    sections,
    loading,
    saving,
    error,
    validationError,

    // Variant space
    variantSpace,
    axesSelection,
    updateAxesSelection,

    // Form state
    form,
    updateFormField,
    sanitizeSlugInput,

    // Header config
    headerConfig,
    setHeaderConfig,

    // Tab state
    activeTab,
    setActiveTab,
    isJsonTab,
    currentSaving,
    savingLabel,

    // Snapshot state
    snapshotDraft,
    setSnapshotDraft,
    snapshotError,
    snapshotLoading,
    snapshotSaving,
    schemaIssues,
    copyStatus,

    // Actions
    handleSave,
    handleSaveJson,
    handleEditorMount,
    handleCopyIssues,
    handleCopySchema,
    fetchVariant,
  };
}
