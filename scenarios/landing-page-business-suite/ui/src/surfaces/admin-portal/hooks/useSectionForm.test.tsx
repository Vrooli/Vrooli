import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import type {
  ContentSection,
  LandingConfigResponse,
  LandingSection,
  Variant,
  getLandingConfig,
  listVariants,
} from '../../../shared/api';
import { useSectionForm } from './useSectionForm';
import type {
  loadSectionEditor,
  persistExistingSectionContent,
  loadVariantContext,
  updateSectionOrder,
} from '../controllers/sectionEditorController';
import type { loadComparePreference, saveComparePreference } from '../services/section.service';

// Mock react-router-dom
const navigateMock = vi.fn();
vi.mock('react-router-dom', () => ({
  useNavigate: () => navigateMock,
}));

vi.mock('../../../shared/hooks/useDebounce', () => ({
  useDebounce: <T,>(value: T) => value,
}));

// Mock the controller
type LoadSectionEditorFn = typeof loadSectionEditor;
type PersistExistingSectionContentFn = typeof persistExistingSectionContent;
type LoadVariantContextFn = typeof loadVariantContext;
type UpdateSectionOrderFn = typeof updateSectionOrder;

const loadSectionEditorMock = vi.fn<Parameters<LoadSectionEditorFn>, ReturnType<LoadSectionEditorFn>>();
const persistExistingSectionContentMock = vi.fn<
  Parameters<PersistExistingSectionContentFn>,
  ReturnType<PersistExistingSectionContentFn>
>();
const loadVariantContextMock = vi.fn<Parameters<LoadVariantContextFn>, ReturnType<LoadVariantContextFn>>();
const updateSectionOrderMock = vi.fn<Parameters<UpdateSectionOrderFn>, ReturnType<UpdateSectionOrderFn>>();

vi.mock('../controllers/sectionEditorController', async () => {
  const actual = await vi.importActual<typeof import('../controllers/sectionEditorController')>('../controllers/sectionEditorController');
  return {
    ...actual,
    loadSectionEditor: (...args: Parameters<LoadSectionEditorFn>) => loadSectionEditorMock(...args),
    persistExistingSectionContent: (...args: Parameters<PersistExistingSectionContentFn>) => persistExistingSectionContentMock(...args),
    loadVariantContext: (...args: Parameters<LoadVariantContextFn>) => loadVariantContextMock(...args),
    updateSectionOrder: (...args: Parameters<UpdateSectionOrderFn>) => updateSectionOrderMock(...args),
  };
});

// Mock API functions
type GetLandingConfigFn = typeof getLandingConfig;
type ListVariantsFn = typeof listVariants;

const getLandingConfigMock = vi.fn<Parameters<GetLandingConfigFn>, ReturnType<GetLandingConfigFn>>();
const listVariantsMock = vi.fn<Parameters<ListVariantsFn>, ReturnType<ListVariantsFn>>();

vi.mock('../../../shared/api', async () => {
  const actual = await vi.importActual<typeof import('../../../shared/api')>('../../../shared/api');
  return {
    ...actual,
    getLandingConfig: (...args: Parameters<GetLandingConfigFn>) => getLandingConfigMock(...args),
    listVariants: (...args: Parameters<ListVariantsFn>) => listVariantsMock(...args),
  };
});

// Mock section service
vi.mock('../services/section.service', async () => {
  const actual = await vi.importActual<typeof import('../services/section.service')>('../services/section.service');
  type LoadComparePreferenceFn = typeof loadComparePreference;
  type SaveComparePreferenceFn = typeof saveComparePreference;
  const loadComparePreferenceMock = vi.fn<Parameters<LoadComparePreferenceFn>, ReturnType<LoadComparePreferenceFn>>()
    .mockReturnValue(null);
  const saveComparePreferenceMock = vi.fn<Parameters<SaveComparePreferenceFn>, ReturnType<SaveComparePreferenceFn>>();
  return {
    ...actual,
    loadComparePreference: (...args: Parameters<LoadComparePreferenceFn>) => loadComparePreferenceMock(...args),
    saveComparePreference: (...args: Parameters<SaveComparePreferenceFn>) => { saveComparePreferenceMock(...args); },
  };
});

// Mock admin experience
vi.mock('../../../shared/lib/adminExperience', () => ({
  rememberVariantSession: vi.fn(),
}));

const mockSection: ContentSection = {
  id: 42,
  variant_id: 1,
  section_type: 'hero',
  content: { title: 'Welcome', subtitle: 'Get started' },
  order: 1,
  enabled: true,
  created_at: '2025-01-01T00:00:00Z',
  updated_at: '2025-01-01T00:00:00Z',
};

const mockLandingSections: LandingSection[] = [
  { id: 42, section_type: 'hero', content: { title: 'Welcome' }, order: 1, enabled: true },
  { id: 43, section_type: 'features', content: {}, order: 2, enabled: true },
];

const mockLandingConfig: LandingConfigResponse = {
  variant: { id: 1, slug: 'control', name: 'Control' },
  sections: mockLandingSections,
  downloads: [],
  header: {
    branding: { mode: 'logo' },
    nav: { links: [] },
    ctas: { primary: { mode: 'inherit_hero' }, secondary: { mode: 'hidden' } },
    behavior: { sticky: true, hide_on_scroll: false },
  },
  fallback: false,
};

const mockVariantContext = {
  variant: { id: 1, slug: 'control', name: 'Control' },
  axes: [],
  variantSpace: { name: 'Test Space' },
};

const mockVariants: Variant[] = [
  { id: 1, slug: 'control', name: 'Control' },
  { id: 2, slug: 'test-a', name: 'Test A' },
];

type SectionFormResult = ReturnType<typeof useSectionForm>;

const waitForNewFormReady = async (result: { current: SectionFormResult }) => {
  await waitFor(() => {
    expect(result.current.variantOptions).toEqual(mockVariants);
  });
  await waitFor(() => {
    expect(result.current.previewConfigError).toBe('Variant slug missing for preview');
  });
};

const waitForExistingVariantReady = async (result: { current: SectionFormResult }) => {
  await waitFor(() => {
    expect(result.current.variantOptions).toEqual(mockVariants);
  });
  await waitFor(() => {
    expect(result.current.previewConfig).toEqual(mockLandingConfig);
  });
  await waitFor(() => {
    expect(result.current.variantContext).toEqual(mockVariantContext);
  });
};

describe('useSectionForm', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    loadSectionEditorMock.mockResolvedValue({
      section: mockSection,
      form: {
        sectionType: 'hero',
        enabled: true,
        order: 1,
        content: mockSection.content,
      },
    });
    loadVariantContextMock.mockResolvedValue(mockVariantContext);
    getLandingConfigMock.mockResolvedValue(mockLandingConfig);
    listVariantsMock.mockResolvedValue({ variants: mockVariants });
  });

  describe('initial state', () => {
    it('loads section data when sectionId is provided', async () => {
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: '42' })
      );

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(loadSectionEditorMock).toHaveBeenCalledWith(42);
      expect(result.current.section).toEqual(mockSection);
      expect(result.current.sectionType).toBe('hero');
    });

    it('does not load section when sectionId is "new"', async () => {
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: undefined, sectionId: 'new' })
      );

      await waitForNewFormReady(result);

      expect(result.current.isNew).toBe(true);
      expect(loadSectionEditorMock).not.toHaveBeenCalled();
    });

    it('loads variant context when variantSlug is provided', async () => {
      renderHook(() => useSectionForm({ variantSlug: 'control', sectionId: '42' }));

      await waitFor(() => {
        expect(loadVariantContextMock).toHaveBeenCalledWith('control');
      });
    });

    it('loads preview config', async () => {
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: '42' })
      );

      await waitFor(() => {
        expect(result.current.previewConfigLoading).toBe(false);
      });

      expect(getLandingConfigMock).toHaveBeenCalledWith('control');
      expect(result.current.previewConfig).toEqual(mockLandingConfig);
    });

    it('loads variant options for comparison', async () => {
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: '42' })
      );

      await waitFor(() => {
        expect(result.current.variantOptionsLoading).toBe(false);
      });

      expect(listVariantsMock).toHaveBeenCalled();
      expect(result.current.variantOptions).toEqual(mockVariants);
    });

    it('handles section load error', async () => {
      const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);
      loadSectionEditorMock.mockRejectedValue(new Error('Section not found'));

      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: '999' })
      );

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.error).toBe('Section not found');
      expect(consoleError).toHaveBeenCalledWith('Section fetch error:', expect.any(Error));
      consoleError.mockRestore();
    });
  });

  describe('form state', () => {
    it('updates content field', async () => {
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: '42' })
      );

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.updateContentField('title', 'New Title');
      });

      expect(result.current.content.title).toBe('New Title');
    });

    it('provides setSectionType', async () => {
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: undefined, sectionId: 'new' })
      );

      await waitForNewFormReady(result);

      act(() => {
        result.current.setSectionType('features');
      });

      expect(result.current.sectionType).toBe('features');
    });

    it('provides setEnabled', async () => {
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: undefined, sectionId: 'new' })
      );

      await waitForNewFormReady(result);

      act(() => {
        result.current.setEnabled(false);
      });

      expect(result.current.enabled).toBe(false);
    });
  });

  describe('handleSave', () => {
    it('persists section content', async () => {
      persistExistingSectionContentMock.mockResolvedValue({
        section: { ...mockSection, content: { title: 'Updated' } },
        form: {
          sectionType: 'hero',
          enabled: true,
          order: 1,
          content: { title: 'Updated' },
        },
      });

      const onSuccess = vi.fn();
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: '42', onSuccess })
      );

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.updateContentField('title', 'Updated');
      });

      await act(async () => {
        await result.current.handleSave();
      });

      expect(persistExistingSectionContentMock).toHaveBeenCalledWith(42, expect.objectContaining({ title: 'Updated' }));
      expect(onSuccess).toHaveBeenCalled();
    });

    it('calls onError when variantSlug is missing', async () => {
      const onError = vi.fn();
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: undefined, sectionId: '42', onError })
      );

      await act(async () => {
        await result.current.handleSave();
      });

      expect(onError).toHaveBeenCalledWith('Variant slug is required to save section');
    });

    it('reports unsupported new-section saves and malformed section identifiers', async () => {
      const onError = vi.fn();
      const { result: newResult } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: 'new', onError })
      );
      await waitForExistingVariantReady(newResult);
      await act(async () => { await newResult.current.handleSave(); });
      expect(onError).toHaveBeenCalledWith('Creating new sections requires variant ID. This is a placeholder.');

      const { result: invalidResult } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: 'not-a-number', onError })
      );
      await waitForExistingVariantReady(invalidResult);
      await act(async () => { await invalidResult.current.handleSave(); });
      expect(invalidResult.current.error).toBe('Section ID is missing.');
      expect(onError).toHaveBeenCalledWith('Cannot save: section ID is missing');
    });

    it('handles save error', async () => {
      const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);
      persistExistingSectionContentMock.mockRejectedValue(new Error('Save failed'));

      const onError = vi.fn();
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: '42', onError })
      );

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      await act(async () => {
        await result.current.handleSave();
      });

      expect(result.current.error).toBe('Save failed');
      expect(onError).toHaveBeenCalledWith('Failed to save section changes');
      expect(consoleError).toHaveBeenCalledWith('Section save error:', expect.any(Error));
      consoleError.mockRestore();
    });
  });

  describe('navigation', () => {
    it('navigates to section', async () => {
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: '42' })
      );

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.handleNavigateSection({ id: 43, section_type: 'features', content: {}, order: 2 });
      });

      expect(navigateMock).toHaveBeenCalledWith('/admin/customization/variants/control/sections/43');
    });

    it('navigates to new section route when id is missing', async () => {
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: '42' })
      );

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.handleNavigateSection({ section_type: 'cta', content: {}, order: 3 });
      });

      expect(navigateMock).toHaveBeenCalledWith('/admin/customization/variants/control/sections/new');
    });

    it('handleAddSection navigates to new section', async () => {
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: '42' })
      );

      await waitForExistingVariantReady(result);

      act(() => {
        result.current.handleAddSection();
      });

      expect(navigateMock).toHaveBeenCalledWith('/admin/customization/variants/control/sections/new');
    });
  });

  describe('section reordering', () => {
    it('reorders section up', async () => {
      updateSectionOrderMock.mockResolvedValue(undefined);

      const onSuccess = vi.fn();
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: '42', onSuccess })
      );

      await waitFor(() => {
        expect(result.current.previewConfigLoading).toBe(false);
      });

      await act(async () => {
        await result.current.handleReorderSection(
          { id: 43, section_type: 'features', content: {}, order: 2 },
          'up'
        );
      });

      expect(updateSectionOrderMock).toHaveBeenCalled();
    });

    it('handles reorder error', async () => {
      updateSectionOrderMock.mockRejectedValue(new Error('Reorder failed'));

      const onError = vi.fn();
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: '42', onError })
      );

      await waitFor(() => {
        expect(result.current.previewConfigLoading).toBe(false);
      });

      await act(async () => {
        await result.current.handleReorderSection(
          { id: 43, section_type: 'features', content: {}, order: 2 },
          'up'
        );
      });

      expect(result.current.reorderError).toBe('Reorder failed');
      expect(onError).toHaveBeenCalledWith('Failed to reorder sections');
    });

    it('does not persist impossible reorders and explains missing neighbor data', async () => {
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: '42' })
      );
      await waitForExistingVariantReady(result);

      await act(async () => {
        await result.current.handleReorderSection({ id: 42, section_type: 'hero', content: {}, order: 1 }, 'up');
      });
      expect(result.current.reorderError).toBe('Unable to move section. Missing neighbor information.');
      expect(updateSectionOrderMock).not.toHaveBeenCalled();

      await act(async () => {
        await result.current.handleReorderSection({ section_type: 'hero', content: {}, order: 1 }, 'down');
      });
      expect(updateSectionOrderMock).not.toHaveBeenCalled();
    });
  });

  describe('comparison', () => {
    it('handles comparison variant change', async () => {
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: '42' })
      );

      await waitFor(() => {
        expect(result.current.previewConfigLoading).toBe(false);
      });

      await act(async () => {
        await result.current.handleCompareVariantChange('test-a');
      });

      expect(result.current.compareVariantSlug).toBe('test-a');
      expect(getLandingConfigMock).toHaveBeenCalledWith('test-a');
    });

    it('clears comparison when slug is empty', async () => {
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: '42' })
      );

      await waitFor(() => {
        expect(result.current.previewConfigLoading).toBe(false);
      });

      await act(async () => {
        await result.current.handleCompareVariantChange('test-a');
      });

      await act(async () => {
        await result.current.handleCompareVariantChange('');
      });

      expect(result.current.compareVariantSlug).toBe('');
      expect(result.current.compareConfig).toBeNull();
    });

    it('uses cache for repeated comparison loads', async () => {
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: '42' })
      );

      await waitFor(() => {
        expect(result.current.previewConfigLoading).toBe(false);
      });

      getLandingConfigMock.mockClear();

      await act(async () => {
        await result.current.handleCompareVariantChange('test-a');
      });

      expect(getLandingConfigMock).toHaveBeenCalledTimes(1);

      await act(async () => {
        await result.current.handleCompareVariantChange('');
      });

      await act(async () => {
        await result.current.handleCompareVariantChange('test-a');
      });

      // Should use cache, not call API again
      expect(getLandingConfigMock).toHaveBeenCalledTimes(1);
    });

    it('surfaces comparison load failures without retaining stale content', async () => {
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: '42' })
      );
      await waitForExistingVariantReady(result);

      getLandingConfigMock.mockRejectedValueOnce(new Error('Comparison unavailable'));

      await act(async () => { await result.current.handleCompareVariantChange('test-a'); });
      expect(result.current.compareError).toBe('Comparison unavailable');
      expect(result.current.compareConfig).toBeNull();
      expect(result.current.comparisonVariantLabel).toBe('Test A');
    });
  });

  describe('computed values', () => {
    it('computes timelineSections from preview config', async () => {
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: '42' })
      );

      await waitFor(() => {
        expect(result.current.previewConfigLoading).toBe(false);
      });

      expect(result.current.timelineSections).toHaveLength(2);
      expect(result.current.timelineSections[0]?.section_type).toBe('hero');
    });

    it('computes previewVariantLabel', async () => {
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: '42' })
      );

      await waitFor(() => {
        expect(result.current.previewConfigLoading).toBe(false);
      });

      expect(result.current.previewVariantLabel).toBe('Control');
    });

    it('computes comparisonVariantLabel', async () => {
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: '42' })
      );

      await waitFor(() => {
        expect(result.current.variantOptionsLoading).toBe(false);
      });

      await act(async () => {
        await result.current.handleCompareVariantChange('test-a');
      });

      expect(result.current.comparisonVariantLabel).toBe('Test A');
    });
  });

  describe('concurrent operations', () => {
    it('handles sequential save and reorder operations', async () => {
      persistExistingSectionContentMock.mockResolvedValue({
        section: { ...mockSection, content: { title: 'Updated' } },
        form: {
          sectionType: 'hero',
          enabled: true,
          order: 1,
          content: { title: 'Updated' },
        },
      });
      updateSectionOrderMock.mockResolvedValue(undefined);

      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: '42' })
      );

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
        expect(result.current.previewConfigLoading).toBe(false);
      });

      act(() => {
        result.current.updateContentField('title', 'Updated');
      });

      // Save operation
      await act(async () => {
        await result.current.handleSave();
      });

      expect(persistExistingSectionContentMock).toHaveBeenCalled();

      // Reorder operation (sequential, not concurrent to avoid overlapping act() calls)
      await act(async () => {
        await result.current.handleReorderSection(
          { id: 43, section_type: 'features', content: {}, order: 2 },
          'up'
        );
      });

      expect(updateSectionOrderMock).toHaveBeenCalled();
    });
  });

  describe('error recovery', () => {
    it('can retry after transient save failure', async () => {
      const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);
      let callCount = 0;
      persistExistingSectionContentMock.mockImplementation(() => {
        callCount++;
        if (callCount === 1) {
          return Promise.reject(new Error('Network error'));
        }
        return Promise.resolve({
          section: { ...mockSection, content: { title: 'Updated' } },
          form: {
            sectionType: 'hero',
            enabled: true,
            order: 1,
            content: { title: 'Updated' },
          },
        });
      });

      const onError = vi.fn();
      const onSuccess = vi.fn();
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: '42', onError, onSuccess })
      );

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.updateContentField('title', 'Updated');
      });

      // First save attempt fails
      await act(async () => {
        await result.current.handleSave();
      });

      expect(result.current.error).toBe('Network error');
      expect(onError).toHaveBeenCalledWith('Failed to save section changes');

      // Retry save - should succeed
      await act(async () => {
        await result.current.handleSave();
      });

      expect(onSuccess).toHaveBeenCalled();
      expect(consoleError).toHaveBeenCalledWith('Section save error:', expect.any(Error));
      consoleError.mockRestore();
    });

    it('can retry after reorder failure', async () => {
      let callCount = 0;
      updateSectionOrderMock.mockImplementation(() => {
        callCount++;
        if (callCount === 1) {
          return Promise.reject(new Error('Reorder failed'));
        }
        return Promise.resolve();
      });

      const onError = vi.fn();
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: '42', onError })
      );

      await waitFor(() => {
        expect(result.current.previewConfigLoading).toBe(false);
      });

      const section = { id: 43, section_type: 'features', content: {}, order: 2 };

      // First reorder attempt fails
      await act(async () => {
        await result.current.handleReorderSection(section, 'up');
      });

      expect(result.current.reorderError).toBe('Reorder failed');

      // Retry reorder - should succeed
      await act(async () => {
        await result.current.handleReorderSection(section, 'up');
      });

      expect(result.current.reorderError).toBeNull();
    });

    it('maintains form state after failed save', async () => {
      const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);
      persistExistingSectionContentMock.mockRejectedValue(new Error('Save failed'));

      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: '42' })
      );

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.updateContentField('title', 'My Custom Title');
      });

      await act(async () => {
        await result.current.handleSave();
      });

      // Form state should be preserved after failure
      expect(result.current.content.title).toBe('My Custom Title');
      expect(result.current.error).toBe('Save failed');
      expect(consoleError).toHaveBeenCalledWith('Section save error:', expect.any(Error));
      consoleError.mockRestore();
    });

    it('clears error state when load succeeds after failure', async () => {
      const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);
      loadSectionEditorMock
        .mockRejectedValueOnce(new Error('Initial load failed'))
        .mockResolvedValueOnce({
          section: mockSection,
          form: {
            sectionType: 'hero',
            enabled: true,
            order: 1,
            content: mockSection.content,
          },
        });

      const { result, rerender } = renderHook(
        ({ sectionId }) => useSectionForm({ variantSlug: 'control', sectionId }),
        { initialProps: { sectionId: '42' } }
      );

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.error).toBe('Initial load failed');

      // Reload by changing sectionId and back
      rerender({ sectionId: '43' });
      rerender({ sectionId: '42' });

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
        expect(result.current.error).toBeNull();
      });
      expect(consoleError).toHaveBeenCalledWith('Section fetch error:', expect.any(Error));
      consoleError.mockRestore();
    });
  });

  describe('validation edge cases', () => {
    // Use 'new' sectionId to avoid needing API mocks for section loading
    it('handles content with unicode characters', async () => {
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: undefined, sectionId: 'new' })
      );

      await waitForNewFormReady(result);

      // For new sections, loading should be false immediately
      expect(result.current.loading).toBe(false);
      expect(result.current.isNew).toBe(true);

      act(() => {
        result.current.updateContentField('title', '日本語タイトル 🚀');
        result.current.updateContentField('subtitle', 'Émojis and spëcial çharacters');
      });

      expect(result.current.content.title).toBe('日本語タイトル 🚀');
      expect(result.current.content.subtitle).toBe('Émojis and spëcial çharacters');
    });

    it('handles very long content strings', async () => {
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: undefined, sectionId: 'new' })
      );

      await waitForNewFormReady(result);

      const longText = 'A'.repeat(10000);

      act(() => {
        result.current.updateContentField('title', longText);
      });

      expect(result.current.content.title).toBe(longText);
      expect((result.current.content.title as string).length).toBe(10000);
    });

    it('handles empty string content', async () => {
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: undefined, sectionId: 'new' })
      );

      await waitForNewFormReady(result);

      // Set initial title then clear it
      act(() => {
        result.current.updateContentField('title', 'Initial');
      });

      act(() => {
        result.current.updateContentField('title', '');
      });

      expect(result.current.content.title).toBe('');
    });

    it('handles content with HTML-like strings', async () => {
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: undefined, sectionId: 'new' })
      );

      await waitForNewFormReady(result);

      act(() => {
        result.current.updateContentField('title', '<script>alert("xss")</script>');
      });

      // The hook should preserve the string as-is (sanitization happens elsewhere)
      expect(result.current.content.title).toBe('<script>alert("xss")</script>');
    });

    it('handles whitespace-only content', async () => {
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: undefined, sectionId: 'new' })
      );

      await waitForNewFormReady(result);

      act(() => {
        result.current.updateContentField('title', '   ');
      });

      expect(result.current.content.title).toBe('   ');
    });
  });
});
