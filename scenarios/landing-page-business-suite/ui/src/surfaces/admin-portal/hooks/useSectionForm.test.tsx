import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import type {
  ContentSection,
  LandingConfigResponse,
  LandingSection,
  Variant,
} from '../../../shared/api';
import { useSectionForm } from './useSectionForm';

// Mock react-router-dom
const navigateMock = vi.fn();
vi.mock('react-router-dom', () => ({
  useNavigate: () => navigateMock,
}));

// Mock the controller
const loadSectionEditorMock = vi.fn();
const persistExistingSectionContentMock = vi.fn();
const loadVariantContextMock = vi.fn();
const updateSectionOrderMock = vi.fn();

vi.mock('../controllers/sectionEditorController', async () => {
  const actual = await vi.importActual<typeof import('../controllers/sectionEditorController')>('../controllers/sectionEditorController');
  return {
    ...actual,
    loadSectionEditor: (...args: unknown[]) => loadSectionEditorMock(...args),
    persistExistingSectionContent: (...args: unknown[]) => persistExistingSectionContentMock(...args),
    loadVariantContext: (...args: unknown[]) => loadVariantContextMock(...args),
    updateSectionOrder: (...args: unknown[]) => updateSectionOrderMock(...args),
  };
});

// Mock API functions
const getLandingConfigMock = vi.fn();
const listVariantsMock = vi.fn();

vi.mock('../../../shared/api', async () => {
  const actual = await vi.importActual<typeof import('../../../shared/api')>('../../../shared/api');
  return {
    ...actual,
    getLandingConfig: (...args: unknown[]) => getLandingConfigMock(...args),
    listVariants: (...args: unknown[]) => listVariantsMock(...args),
  };
});

// Mock section service
vi.mock('../services/section.service', async () => {
  const actual = await vi.importActual<typeof import('../services/section.service')>('../services/section.service');
  return {
    ...actual,
    loadComparePreference: vi.fn().mockReturnValue(null),
    saveComparePreference: vi.fn(),
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
        useSectionForm({ variantSlug: 'control', sectionId: 'new' })
      );

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
      loadSectionEditorMock.mockRejectedValue(new Error('Section not found'));

      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: '999' })
      );

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.error).toBe('Section not found');
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
        useSectionForm({ variantSlug: 'control', sectionId: 'new' })
      );

      act(() => {
        result.current.setSectionType('features');
      });

      expect(result.current.sectionType).toBe('features');
    });

    it('provides setEnabled', async () => {
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: 'new' })
      );

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

    it('handles save error', async () => {
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
      expect(result.current.timelineSections[0].section_type).toBe('hero');
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
      let callCount = 0;
      persistExistingSectionContentMock.mockImplementation(async () => {
        callCount++;
        if (callCount === 1) {
          throw new Error('Network error');
        }
        return {
          section: { ...mockSection, content: { title: 'Updated' } },
          form: {
            sectionType: 'hero',
            enabled: true,
            order: 1,
            content: { title: 'Updated' },
          },
        };
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
    });

    it('can retry after reorder failure', async () => {
      let callCount = 0;
      updateSectionOrderMock.mockImplementation(async () => {
        callCount++;
        if (callCount === 1) {
          throw new Error('Reorder failed');
        }
        return undefined;
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
    });

    it('clears error state when load succeeds after failure', async () => {
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
    });
  });

  describe('validation edge cases', () => {
    // Use 'new' sectionId to avoid needing API mocks for section loading
    it('handles content with unicode characters', async () => {
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: 'new' })
      );

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
        useSectionForm({ variantSlug: 'control', sectionId: 'new' })
      );

      const longText = 'A'.repeat(10000);

      act(() => {
        result.current.updateContentField('title', longText);
      });

      expect(result.current.content.title).toBe(longText);
      expect((result.current.content.title as string).length).toBe(10000);
    });

    it('handles empty string content', async () => {
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: 'new' })
      );

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
        useSectionForm({ variantSlug: 'control', sectionId: 'new' })
      );

      act(() => {
        result.current.updateContentField('title', '<script>alert("xss")</script>');
      });

      // The hook should preserve the string as-is (sanitization happens elsewhere)
      expect(result.current.content.title).toBe('<script>alert("xss")</script>');
    });

    it('handles whitespace-only content', async () => {
      const { result } = renderHook(() =>
        useSectionForm({ variantSlug: 'control', sectionId: 'new' })
      );

      act(() => {
        result.current.updateContentField('title', '   ');
      });

      expect(result.current.content.title).toBe('   ');
    });
  });
});
