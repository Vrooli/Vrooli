import { describe, it, expect, vi, beforeEach } from 'vitest';
import type {
  ContentSection,
  Variant,
  VariantSpace,
  VariantAxes,
  LandingHeaderConfig,
  getVariant,
  getVariantSections,
  getVariantSpace,
  createVariant,
  updateVariant,
  exportVariantSnapshot,
  importVariantSnapshot,
} from '../../../shared/api';
import {
  loadVariantEditorData,
  loadVariantSpaceDefinition,
  buildAxesSelection,
  hydrateFormFromVariant,
  normalizeForm,
  sanitizeSlugInput,
  validateVariantForm,
  persistVariant,
  loadVariantSnapshot,
  persistVariantSnapshot,
  type VariantFormState,
} from './variantEditorController';

// Mock the API module
type GetVariantFn = typeof getVariant;
type GetVariantSectionsFn = typeof getVariantSections;
type GetVariantSpaceFn = typeof getVariantSpace;
type CreateVariantFn = typeof createVariant;
type UpdateVariantFn = typeof updateVariant;
type ExportVariantSnapshotFn = typeof exportVariantSnapshot;
type ImportVariantSnapshotFn = typeof importVariantSnapshot;

const getVariantMock = vi.fn<GetVariantFn>();
const getVariantSectionsMock = vi.fn<GetVariantSectionsFn>();
const getVariantSpaceMock = vi.fn<GetVariantSpaceFn>();
const createVariantMock = vi.fn<CreateVariantFn>();
const updateVariantMock = vi.fn<UpdateVariantFn>();
const exportVariantSnapshotMock = vi.fn<ExportVariantSnapshotFn>();
const importVariantSnapshotMock = vi.fn<ImportVariantSnapshotFn>();

vi.mock('../../../shared/api', async () => {
  const actual = await vi.importActual<typeof import('../../../shared/api')>('../../../shared/api');
  return {
    ...actual,
    getVariant: (...args: Parameters<GetVariantFn>) => getVariantMock(...args),
    getVariantSections: (...args: Parameters<GetVariantSectionsFn>) => getVariantSectionsMock(...args),
    getVariantSpace: (...args: Parameters<GetVariantSpaceFn>) => getVariantSpaceMock(...args),
    createVariant: (...args: Parameters<CreateVariantFn>) => createVariantMock(...args),
    updateVariant: (...args: Parameters<UpdateVariantFn>) => updateVariantMock(...args),
    exportVariantSnapshot: (...args: Parameters<ExportVariantSnapshotFn>) => exportVariantSnapshotMock(...args),
    importVariantSnapshot: (...args: Parameters<ImportVariantSnapshotFn>) => importVariantSnapshotMock(...args),
  };
});

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
};

const mockSections: ContentSection[] = [
  {
    id: 1,
    variant_id: 1,
    key: 'section-1-hero',
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

const mockHeaderConfig: LandingHeaderConfig = {
  branding: { mode: 'logo', label: 'Test' },
  nav: { links: [] },
  ctas: {
    primary: { mode: 'inherit_hero' },
    secondary: { mode: 'hidden' },
  },
  behavior: { sticky: true, hide_on_scroll: false },
};

describe('variantEditorController', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('loadVariantEditorData', () => {
    it('fetches variant and sections', async () => {
      getVariantMock.mockResolvedValue(mockVariant);
      getVariantSectionsMock.mockResolvedValue({ sections: mockSections });

      const result = await loadVariantEditorData('control');

      expect(getVariantMock).toHaveBeenCalledWith('control');
      expect(getVariantSectionsMock).toHaveBeenCalledWith('control');
      expect(result.variant).toEqual(mockVariant);
      expect(result.sections).toEqual(mockSections);
    });

    it('supports variants that omit a numeric database ID', async () => {
      getVariantMock.mockResolvedValue({ ...mockVariant, id: undefined });
      getVariantSectionsMock.mockResolvedValue({ sections: [] });

      await expect(loadVariantEditorData('control')).resolves.toMatchObject({ sections: [] });
    });

    it('propagates API errors', async () => {
      getVariantMock.mockRejectedValue(new Error('Variant not found'));

      await expect(loadVariantEditorData('invalid')).rejects.toThrow('Variant not found');
    });
  });

  describe('loadVariantSpaceDefinition', () => {
    it('calls getVariantSpace', async () => {
      getVariantSpaceMock.mockResolvedValue(mockVariantSpace);

      const result = await loadVariantSpaceDefinition();

      expect(getVariantSpaceMock).toHaveBeenCalled();
      expect(result).toEqual(mockVariantSpace);
    });
  });

  describe('buildAxesSelection', () => {
    it('builds selection from variant space with existing values', () => {
      const existing: VariantAxes = { jtbd: 'analytics' };

      const result = buildAxesSelection(mockVariantSpace, existing);

      expect(result.jtbd).toBe('analytics');
      expect(result.industry).toBe('tech'); // Falls back to first option
    });

    it('uses first variant as fallback when no existing value', () => {
      const result = buildAxesSelection(mockVariantSpace, undefined);

      expect(result.jtbd).toBe('automation');
      expect(result.industry).toBe('tech');
    });

    it('handles empty axes', () => {
      const emptySpace: VariantSpace = {
        _name: 'Empty',
        _schemaVersion: 1,
        axes: {},
      };

      const result = buildAxesSelection(emptySpace, undefined);

      expect(result).toEqual({});
    });

    it('handles axis with empty variants', () => {
      const spaceWithEmptyVariants: VariantSpace = {
        _name: 'Test',
        _schemaVersion: 1,
        axes: {
          emptyAxis: { variants: [] },
        },
      };

      const result = buildAxesSelection(spaceWithEmptyVariants, undefined);

      expect(result.emptyAxis).toBeUndefined();
    });
  });

  describe('hydrateFormFromVariant', () => {
    it('extracts form state from variant', () => {
      const result = hydrateFormFromVariant(mockVariant);

      expect(result.name).toBe('Control Variant');
      expect(result.slug).toBe('control');
      expect(result.description).toBe('The default variant');
      expect(result.weight).toBe(50);
    });

    it('handles missing optional fields', () => {
      const minimalVariant: Variant = {
        slug: 'minimal',
        name: 'Minimal',
      };

      const result = hydrateFormFromVariant(minimalVariant);

      expect(result.name).toBe('Minimal');
      expect(result.slug).toBe('minimal');
      expect(result.description).toBe('');
      expect(result.weight).toBe(50);
    });
  });

  describe('normalizeForm', () => {
    it('trims whitespace from string fields', () => {
      const form: VariantFormState = {
        name: '  Test Variant  ',
        slug: '  test-variant  ',
        description: '  Description  ',
        weight: 75,
      };

      const result = normalizeForm(form);

      expect(result.name).toBe('Test Variant');
      expect(result.slug).toBe('test-variant');
      expect(result.description).toBe('Description');
      expect(result.weight).toBe(75);
    });

    it('sanitizes slug', () => {
      const form: VariantFormState = {
        name: 'Test',
        slug: 'Test_Variant 123!',
        description: '',
        weight: 50,
      };

      const result = normalizeForm(form);

      expect(result.slug).toBe('testvariant123');
    });
  });

  describe('sanitizeSlugInput', () => {
    it('converts to lowercase', () => {
      expect(sanitizeSlugInput('TestSlug')).toBe('testslug');
    });

    it('removes non-alphanumeric characters except hyphens', () => {
      expect(sanitizeSlugInput('test_slug!@#')).toBe('testslug');
    });

    it('keeps hyphens', () => {
      expect(sanitizeSlugInput('test-slug-name')).toBe('test-slug-name');
    });

    it('handles empty string', () => {
      expect(sanitizeSlugInput('')).toBe('');
    });

    it('removes spaces', () => {
      expect(sanitizeSlugInput('test slug')).toBe('testslug');
    });
  });

  describe('validateVariantForm', () => {
    const validParams = {
      form: { name: 'Test', slug: 'test', description: '', weight: 50 },
      variantSpace: mockVariantSpace,
      axesSelection: { jtbd: 'automation', industry: 'tech' },
      requireSlug: true,
    };

    it('returns null for valid form', () => {
      const result = validateVariantForm(validParams);
      expect(result).toBeNull();
    });

    it('returns error when name is empty', () => {
      const result = validateVariantForm({
        ...validParams,
        form: { ...validParams.form, name: '' },
      });
      expect(result).toBe('Name is required');
    });

    it('returns error when slug is required but empty', () => {
      const result = validateVariantForm({
        ...validParams,
        form: { ...validParams.form, slug: '' },
        requireSlug: true,
      });
      expect(result).toBe('Slug is required');
    });

    it('allows empty slug when not required', () => {
      const result = validateVariantForm({
        ...validParams,
        form: { ...validParams.form, slug: '' },
        requireSlug: false,
      });
      expect(result).toBeNull();
    });

    it('returns error when variant space is not loaded', () => {
      const result = validateVariantForm({
        ...validParams,
        variantSpace: null,
      });
      expect(result).toBe('Variant axes registry not loaded yet');
    });

    it('returns error when axis selection is missing', () => {
      const result = validateVariantForm({
        ...validParams,
        axesSelection: { jtbd: 'automation' }, // Missing industry
      });
      expect(result).toBe('Select a value for the industry axis');
    });

    it('handles whitespace-only name', () => {
      const result = validateVariantForm({
        ...validParams,
        form: { ...validParams.form, name: '   ' },
      });
      expect(result).toBe('Name is required');
    });
  });

  describe('persistVariant', () => {
    it('creates new variant when isNew is true', async () => {
      const newVariant = { id: 2, slug: 'new-variant', name: 'New Variant' };
      createVariantMock.mockResolvedValue(newVariant);

      const result = await persistVariant({
        isNew: true,
        form: { name: 'New Variant', slug: 'new-variant', description: 'Desc', weight: 50 },
        axesSelection: { jtbd: 'automation' },
        headerConfig: mockHeaderConfig,
      });

      expect(createVariantMock).toHaveBeenCalledWith({
        name: 'New Variant',
        slug: 'new-variant',
        description: 'Desc',
        weight: 50,
        axes: { jtbd: 'automation' },
        header_config: mockHeaderConfig,
      });
      expect(result).toEqual(newVariant);
    });

    it('omits empty description when creating', async () => {
      createVariantMock.mockResolvedValue({ id: 2, slug: 'test', name: 'Test' });

      await persistVariant({
        isNew: true,
        form: { name: 'Test', slug: 'test', description: '', weight: 50 },
        axesSelection: {},
        headerConfig: mockHeaderConfig,
      });

      expect(createVariantMock).toHaveBeenCalledWith(
        expect.objectContaining({
          description: undefined,
        })
      );
    });

    it('updates existing variant when isNew is false', async () => {
      updateVariantMock.mockResolvedValue({
        id: 1,
        slug: 'control',
        name: 'Updated Name',
        status: 'active',
      });

      await persistVariant({
        isNew: false,
        slugFromRoute: 'control',
        form: { name: 'Updated Name', slug: 'control', description: 'Updated', weight: 75 },
        axesSelection: { jtbd: 'analytics' },
        headerConfig: mockHeaderConfig,
      });

      expect(updateVariantMock).toHaveBeenCalledWith('control', {
        name: 'Updated Name',
        description: 'Updated',
        weight: 75,
        axes: { jtbd: 'analytics' },
        header_config: mockHeaderConfig,
      });
    });

    it('throws error when updating without slugFromRoute', async () => {
      await expect(
        persistVariant({
          isNew: false,
          slugFromRoute: undefined,
          form: { name: 'Test', slug: 'test', description: '', weight: 50 },
          axesSelection: {},
          headerConfig: mockHeaderConfig,
        })
      ).rejects.toThrow('Variant slug missing');
    });
  });

  describe('loadVariantSnapshot', () => {
    it('calls exportVariantSnapshot', async () => {
      const mockSnapshot = { variant: { slug: 'control', name: 'Control', axes: {} }, sections: [] };
      exportVariantSnapshotMock.mockResolvedValue(mockSnapshot);

      const result = await loadVariantSnapshot('control');

      expect(exportVariantSnapshotMock).toHaveBeenCalledWith('control');
      expect(result).toEqual(mockSnapshot);
    });
  });

  describe('persistVariantSnapshot', () => {
    it('calls importVariantSnapshot', async () => {
      const mockPayload = { variant: { slug: 'control', name: 'Control', axes: {} }, sections: [] };
      const savedSnapshot = { ...mockPayload, _metadata: { updated_at: '2025-01-01' } };
      importVariantSnapshotMock.mockResolvedValue(savedSnapshot);

      const result = await persistVariantSnapshot('control', mockPayload);

      expect(importVariantSnapshotMock).toHaveBeenCalledWith('control', mockPayload);
      expect(result).toEqual(savedSnapshot);
    });
  });
});
