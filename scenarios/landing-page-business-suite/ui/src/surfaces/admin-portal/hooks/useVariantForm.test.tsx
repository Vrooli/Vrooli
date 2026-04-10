import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import type {
  ContentSection,
  Variant,
  VariantSpace,
  LandingHeaderConfig,
} from '../../../shared/api';
import { useVariantForm } from './useVariantForm';
import type {
  loadVariantEditorData,
  loadVariantSpaceDefinition,
  loadVariantSnapshot,
  persistVariant,
  persistVariantSnapshot,
  validateVariantForm,
} from '../controllers/variantEditorController';

// Mock the controller
type LoadVariantEditorDataFn = typeof loadVariantEditorData;
type LoadVariantSpaceDefinitionFn = typeof loadVariantSpaceDefinition;
type LoadVariantSnapshotFn = typeof loadVariantSnapshot;
type PersistVariantFn = typeof persistVariant;
type PersistVariantSnapshotFn = typeof persistVariantSnapshot;
type ValidateVariantFormFn = typeof validateVariantForm;

const loadVariantEditorDataMock = vi.fn<Parameters<LoadVariantEditorDataFn>, ReturnType<LoadVariantEditorDataFn>>();
const loadVariantSpaceDefinitionMock = vi.fn<
  Parameters<LoadVariantSpaceDefinitionFn>,
  ReturnType<LoadVariantSpaceDefinitionFn>
>();
const loadVariantSnapshotMock = vi.fn<Parameters<LoadVariantSnapshotFn>, ReturnType<LoadVariantSnapshotFn>>();
const persistVariantMock = vi.fn<Parameters<PersistVariantFn>, ReturnType<PersistVariantFn>>();
const persistVariantSnapshotMock = vi.fn<Parameters<PersistVariantSnapshotFn>, ReturnType<PersistVariantSnapshotFn>>();
const validateVariantFormMock = vi.fn<Parameters<ValidateVariantFormFn>, ReturnType<ValidateVariantFormFn>>();

vi.mock('../controllers/variantEditorController', async () => {
  const actual = await vi.importActual<typeof import('../controllers/variantEditorController')>('../controllers/variantEditorController');
  return {
    ...actual,
    loadVariantEditorData: (...args: Parameters<LoadVariantEditorDataFn>) => loadVariantEditorDataMock(...args),
    loadVariantSpaceDefinition: (...args: Parameters<LoadVariantSpaceDefinitionFn>) => loadVariantSpaceDefinitionMock(...args),
    loadVariantSnapshot: (...args: Parameters<LoadVariantSnapshotFn>) => loadVariantSnapshotMock(...args),
    persistVariant: (...args: Parameters<PersistVariantFn>) => persistVariantMock(...args),
    persistVariantSnapshot: (...args: Parameters<PersistVariantSnapshotFn>) => persistVariantSnapshotMock(...args),
    validateVariantForm: (...args: Parameters<ValidateVariantFormFn>) => validateVariantFormMock(...args),
  };
});

// Mock header config lib
vi.mock('../../../shared/lib/headerConfig', () => ({
  buildDefaultHeaderConfig: (name: string) => ({
    branding: { mode: 'logo', label: name },
    nav: { links: [] },
    ctas: { primary: { mode: 'inherit_hero' }, secondary: { mode: 'hidden' } },
    behavior: { sticky: true, hide_on_scroll: false },
  }),
  normalizeHeaderConfig: (config: unknown) => config,
}));

// Mock admin experience
vi.mock('../../../shared/lib/adminExperience', () => ({
  rememberVariantSession: vi.fn(),
}));

const mockVariant: Variant = {
  id: 1,
  slug: 'control',
  name: 'Control Variant',
  description: 'The default variant',
  weight: 50,
  axes: {
    jtbd: 'automation',
    industry: 'tech',
  },
  header_config: {
    branding: { mode: 'logo', label: 'Control' },
    nav: { links: [] },
    ctas: { primary: { mode: 'inherit_hero' }, secondary: { mode: 'hidden' } },
    behavior: { sticky: true, hide_on_scroll: false },
  },
};

const mockSections: ContentSection[] = [
  {
    id: 1,
    variant_id: 1,
    section_type: 'hero',
    content: { title: 'Welcome' },
    order: 1,
    enabled: true,
    created_at: '2025-01-01T00:00:00Z',
    updated_at: '2025-01-01T00:00:00Z',
  },
];

const mockVariantSpace: VariantSpace = {
  _name: 'Test Space',
  _schemaVersion: 1,
  axes: {
    jtbd: {
      variants: [
        { id: 'automation', label: 'Automation' },
        { id: 'analytics', label: 'Analytics' },
      ],
    },
    industry: {
      variants: [
        { id: 'tech', label: 'Technology' },
        { id: 'finance', label: 'Finance' },
      ],
    },
  },
};

const mockSnapshot = {
  variant: { slug: 'control', name: 'Control', axes: {} },
  sections: [],
};

const defaultProps = {
  slug: 'control',
  isNew: false,
  monacoSchemaCatalog: [],
  variantSchemaUri: 'file://schema.json',
  editorModelPath: 'file://variant.json',
};

describe('useVariantForm', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    loadVariantEditorDataMock.mockResolvedValue({ variant: mockVariant, sections: mockSections });
    loadVariantSpaceDefinitionMock.mockResolvedValue(mockVariantSpace);
    loadVariantSnapshotMock.mockResolvedValue(mockSnapshot);
    validateVariantFormMock.mockReturnValue(null);
    persistVariantMock.mockResolvedValue(undefined);
  });

  describe('initial load', () => {
    it('loads variant data when not new', async () => {
      const { result } = renderHook(() => useVariantForm(defaultProps));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(loadVariantEditorDataMock).toHaveBeenCalledWith('control');
      expect(result.current.variant).toEqual(mockVariant);
      expect(result.current.sections).toEqual(mockSections);
    });

    it('does not load variant data when isNew', async () => {
      const { result } = renderHook(() =>
        useVariantForm({ ...defaultProps, isNew: true, slug: undefined })
      );

      expect(result.current.loading).toBe(false);
      expect(loadVariantEditorDataMock).not.toHaveBeenCalled();
    });

    it('loads variant space definition', async () => {
      renderHook(() => useVariantForm(defaultProps));

      await waitFor(() => {
        expect(loadVariantSpaceDefinitionMock).toHaveBeenCalled();
      });
    });

    it('hydrates form from variant', async () => {
      const { result } = renderHook(() => useVariantForm(defaultProps));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.form.name).toBe('Control Variant');
      expect(result.current.form.slug).toBe('control');
      expect(result.current.form.description).toBe('The default variant');
      expect(result.current.form.weight).toBe(50);
    });

    it('loads snapshot for JSON editor', async () => {
      const { result } = renderHook(() => useVariantForm(defaultProps));

      await waitFor(() => {
        expect(result.current.snapshotLoading).toBe(false);
      });

      expect(loadVariantSnapshotMock).toHaveBeenCalledWith('control');
      expect(result.current.snapshotDraft).toContain('control');
    });

    it('handles variant load error', async () => {
      loadVariantEditorDataMock.mockRejectedValue(new Error('Variant not found'));

      const { result } = renderHook(() => useVariantForm(defaultProps));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.error).toBe('Variant not found');
    });
  });

  describe('form state', () => {
    it('updates form field', async () => {
      const { result } = renderHook(() => useVariantForm(defaultProps));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.updateFormField('name', 'New Name');
      });

      expect(result.current.form.name).toBe('New Name');
    });

    it('clears validation error on form field change', async () => {
      validateVariantFormMock.mockReturnValue('Name is required');

      const { result } = renderHook(() => useVariantForm(defaultProps));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      // Trigger validation error
      await act(async () => {
        await result.current.handleSave();
      });

      expect(result.current.validationError).toBe('Name is required');

      // Update field should clear error
      validateVariantFormMock.mockReturnValue(null);
      act(() => {
        result.current.updateFormField('name', 'Valid Name');
      });

      expect(result.current.validationError).toBeNull();
    });

    it('updates axes selection', async () => {
      const { result } = renderHook(() => useVariantForm(defaultProps));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.updateAxesSelection('jtbd', 'analytics');
      });

      expect(result.current.axesSelection.jtbd).toBe('analytics');
    });

    it('sanitizes slug input', async () => {
      const { result } = renderHook(() => useVariantForm(defaultProps));

      const sanitized = result.current.sanitizeSlugInput('Test Slug 123!');

      expect(sanitized).toBe('testslug123');
    });
  });

  describe('tab state', () => {
    it('defaults to form tab', async () => {
      const { result } = renderHook(() => useVariantForm(defaultProps));

      expect(result.current.activeTab).toBe('form');
      expect(result.current.isJsonTab).toBe(false);
    });

    it('switches between tabs', async () => {
      const { result } = renderHook(() => useVariantForm(defaultProps));

      act(() => {
        result.current.setActiveTab('json');
      });

      expect(result.current.activeTab).toBe('json');
      expect(result.current.isJsonTab).toBe(true);
    });

    it('computes saving label based on tab', async () => {
      const { result } = renderHook(() => useVariantForm(defaultProps));

      expect(result.current.savingLabel).toBe('Save');

      act(() => {
        result.current.setActiveTab('json');
      });

      expect(result.current.savingLabel).toBe('Save JSON');
    });
  });

  describe('handleSave', () => {
    it('validates form before saving', async () => {
      validateVariantFormMock.mockReturnValue('Name is required');

      const { result } = renderHook(() => useVariantForm(defaultProps));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      const saveResult = await act(async () => {
        return await result.current.handleSave();
      });

      expect(saveResult.success).toBe(false);
      expect(result.current.validationError).toBe('Name is required');
      expect(persistVariantMock).not.toHaveBeenCalled();
    });

    it('persists variant on valid form', async () => {
      const onSuccess = vi.fn();
      const { result } = renderHook(() =>
        useVariantForm({ ...defaultProps, onSuccess })
      );

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      await act(async () => {
        await result.current.handleSave();
      });

      expect(persistVariantMock).toHaveBeenCalled();
      expect(onSuccess).toHaveBeenCalled();
    });

    it('handles save error', async () => {
      persistVariantMock.mockRejectedValue(new Error('Save failed'));

      const onError = vi.fn();
      const { result } = renderHook(() =>
        useVariantForm({ ...defaultProps, onError })
      );

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      const saveResult = await act(async () => {
        return await result.current.handleSave();
      });

      expect(saveResult.success).toBe(false);
      expect(result.current.error).toBe('Save failed');
      expect(onError).toHaveBeenCalledWith('Failed to save variant changes');
    });

    it('returns saved variant on create', async () => {
      const newVariant = { id: 2, slug: 'new-variant', name: 'New Variant' };
      persistVariantMock.mockResolvedValue(newVariant);

      const { result } = renderHook(() =>
        useVariantForm({ ...defaultProps, isNew: true, slug: undefined })
      );

      const saveResult = await act(async () => {
        return await result.current.handleSave();
      });

      expect(saveResult.success).toBe(true);
      expect(saveResult.savedVariant).toEqual(newVariant);
    });
  });

  describe('handleSaveJson', () => {
    it('parses and persists JSON snapshot', async () => {
      const savedSnapshot = { ...mockSnapshot, _metadata: { updated_at: '2025-01-01' } };
      persistVariantSnapshotMock.mockResolvedValue(savedSnapshot);

      const onSuccess = vi.fn();
      const { result } = renderHook(() =>
        useVariantForm({ ...defaultProps, onSuccess })
      );

      await waitFor(() => {
        expect(result.current.snapshotLoading).toBe(false);
      });

      const saveResult = await act(async () => {
        return await result.current.handleSaveJson();
      });

      expect(saveResult).toBe(true);
      expect(persistVariantSnapshotMock).toHaveBeenCalled();
      expect(onSuccess).toHaveBeenCalled();
    });

    it('handles invalid JSON', async () => {
      const onError = vi.fn();
      const { result } = renderHook(() =>
        useVariantForm({ ...defaultProps, onError })
      );

      await waitFor(() => {
        expect(result.current.snapshotLoading).toBe(false);
      });

      act(() => {
        result.current.setSnapshotDraft('not valid json');
      });

      const saveResult = await act(async () => {
        return await result.current.handleSaveJson();
      });

      expect(saveResult).toBe(false);
      expect(result.current.snapshotError).toContain('Invalid JSON');
      expect(onError).toHaveBeenCalledWith('Invalid JSON syntax');
    });

    it('handles missing slug', async () => {
      const { result } = renderHook(() =>
        useVariantForm({ ...defaultProps, isNew: true, slug: undefined })
      );

      const saveResult = await act(async () => {
        return await result.current.handleSaveJson();
      });

      expect(saveResult).toBe(false);
      expect(result.current.snapshotError).toBe('Variant slug missing');
    });

    it('handles save error', async () => {
      persistVariantSnapshotMock.mockRejectedValue(new Error('Snapshot save failed'));

      const onError = vi.fn();
      const { result } = renderHook(() =>
        useVariantForm({ ...defaultProps, onError })
      );

      await waitFor(() => {
        expect(result.current.snapshotLoading).toBe(false);
      });

      const saveResult = await act(async () => {
        return await result.current.handleSaveJson();
      });

      expect(saveResult).toBe(false);
      expect(result.current.snapshotError).toBe('Snapshot save failed');
    });
  });

  describe('header config', () => {
    it('provides header config state', async () => {
      const { result } = renderHook(() => useVariantForm(defaultProps));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.headerConfig).toBeDefined();
      expect(result.current.headerConfig.branding.mode).toBe('logo');
    });

    it('allows updating header config', async () => {
      const { result } = renderHook(() => useVariantForm(defaultProps));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.setHeaderConfig({
          ...result.current.headerConfig,
          branding: { mode: 'name', label: 'Updated' },
        });
      });

      expect(result.current.headerConfig.branding.mode).toBe('name');
    });
  });

  describe('axes seeding', () => {
    it('seeds axes from variant space for new variant', async () => {
      const { result } = renderHook(() =>
        useVariantForm({ ...defaultProps, isNew: true, slug: undefined })
      );

      await waitFor(() => {
        expect(result.current.variantSpace).not.toBeNull();
      });

      // Axes should be seeded with first options from variant space
      expect(result.current.axesSelection.jtbd).toBe('automation');
      expect(result.current.axesSelection.industry).toBe('tech');
    });

    it('preserves existing axes for existing variant', async () => {
      const { result } = renderHook(() => useVariantForm(defaultProps));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.axesSelection.jtbd).toBe('automation');
      expect(result.current.axesSelection.industry).toBe('tech');
    });
  });

  describe('clipboard operations', () => {
    const originalClipboard = navigator.clipboard;

    afterEach(() => {
      Object.defineProperty(navigator, 'clipboard', {
        value: originalClipboard,
        configurable: true,
      });
    });

    it('copies schema issues to clipboard', async () => {
      const writeTextMock = vi.fn().mockResolvedValue(undefined);
      Object.defineProperty(navigator, 'clipboard', {
        value: { writeText: writeTextMock },
        configurable: true,
      });

      const { result } = renderHook(() => useVariantForm(defaultProps));

      await act(async () => {
        await result.current.handleCopyIssues();
      });

      expect(result.current.copyStatus).toBe('No schema issues to copy');
    });

    it('copies schema to clipboard', async () => {
      const writeTextMock = vi.fn().mockResolvedValue(undefined);
      Object.defineProperty(navigator, 'clipboard', {
        value: { writeText: writeTextMock },
        configurable: true,
      });

      const { result } = renderHook(() => useVariantForm(defaultProps));

      const schema = { type: 'object' };
      await act(async () => {
        await result.current.handleCopySchema(schema);
      });

      expect(writeTextMock).toHaveBeenCalledWith(JSON.stringify(schema, null, 2));
      expect(result.current.copyStatus).toBe('Schema copied');
    });

    it('handles clipboard write failure', async () => {
      Object.defineProperty(navigator, 'clipboard', {
        value: { writeText: vi.fn().mockRejectedValue(new Error('Clipboard blocked')) },
        configurable: true,
      });

      const { result } = renderHook(() => useVariantForm(defaultProps));

      await act(async () => {
        await result.current.handleCopySchema({ type: 'object' });
      });

      expect(result.current.copyStatus).toBe('Copy failed');
    });
  });

  describe('fetchVariant', () => {
    it('allows manual refetch', async () => {
      const { result } = renderHook(() => useVariantForm(defaultProps));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      loadVariantEditorDataMock.mockClear();

      await act(async () => {
        await result.current.fetchVariant();
      });

      expect(loadVariantEditorDataMock).toHaveBeenCalledWith('control');
    });
  });
});
