import { describe, it, expect, vi, beforeEach } from 'vitest';

const {
  mockCreateVariant,
  mockGetAdminSections,
  mockGetVariant,
  mockGetVariantSpace,
  mockExportVariantSnapshot,
  mockImportVariantSnapshot,
  mockUpdateVariant,
} = vi.hoisted(() => ({
  mockCreateVariant: vi.fn(),
  mockGetAdminSections: vi.fn(),
  mockGetVariant: vi.fn(),
  mockGetVariantSpace: vi.fn(),
  mockExportVariantSnapshot: vi.fn(),
  mockImportVariantSnapshot: vi.fn(),
  mockUpdateVariant: vi.fn(),
}));

vi.mock('../../../shared/api', () => ({
  createVariant: mockCreateVariant,
  getAdminSections: mockGetAdminSections,
  getVariant: mockGetVariant,
  getVariantSpace: mockGetVariantSpace,
  exportVariantSnapshot: mockExportVariantSnapshot,
  importVariantSnapshot: mockImportVariantSnapshot,
  updateVariant: mockUpdateVariant,
}));

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
} from './variantEditorController';

const variantSpace = {
  axes: {
    tone: {
      _note: 'Tone axis',
      variants: [
        { id: 'bold', label: 'Bold' },
        { id: 'calm', label: 'Calm' },
      ],
    },
    audience: {
      variants: [
        { id: 'devs', label: 'Developers' },
      ],
    },
  },
} as never;

const baseVariant = {
  id: 7n,
  slug: 'hero',
  name: 'Hero',
  description: 'The hero variant',
  weight: 40,
  axes: { tone: 'calm' },
} as never;

beforeEach(() => {
  vi.clearAllMocks();
});

describe('sanitizeSlugInput', () => {
  it('lowercases and strips characters outside [a-z0-9-]', () => {
    expect(sanitizeSlugInput('My Cool_Variant!!')).toBe('mycoolvariant');
    expect(sanitizeSlugInput('Keep-Me-123')).toBe('keep-me-123');
  });
});

describe('normalizeForm', () => {
  it('trims fields and sanitizes the slug while preserving weight', () => {
    expect(
      normalizeForm({ name: '  Hero  ', slug: '  Hero Slug ', description: '  hi ', weight: 25 }),
    ).toEqual({ name: 'Hero', slug: 'heroslug', description: 'hi', weight: 25 });
  });
});

describe('hydrateFormFromVariant', () => {
  it('maps a variant onto form state', () => {
    expect(hydrateFormFromVariant(baseVariant)).toEqual({
      name: 'Hero',
      slug: 'hero',
      description: 'The hero variant',
      weight: 40,
    });
  });

  it('falls back to defaults for missing fields', () => {
    expect(hydrateFormFromVariant({} as never)).toEqual({
      name: '',
      slug: '',
      description: '',
      weight: 50,
    });
  });
});

describe('buildAxesSelection', () => {
  it('keeps an existing selection and defaults the rest to the first variant', () => {
    const selection = buildAxesSelection(variantSpace, { tone: 'bold' });
    expect(selection).toEqual({ tone: 'bold', audience: 'devs' });
  });

  it('defaults every axis when no existing selection is provided', () => {
    expect(buildAxesSelection(variantSpace)).toEqual({ tone: 'bold', audience: 'devs' });
  });
});

describe('validateVariantForm', () => {
  const axesSelection = { tone: 'bold', audience: 'devs' };

  it('requires a name', () => {
    expect(
      validateVariantForm({
        form: { name: '  ', slug: 'x', description: '', weight: 10 },
        variantSpace,
        axesSelection,
        requireSlug: true,
      }),
    ).toBe('Name is required');
  });

  it('requires a slug when requireSlug is set', () => {
    expect(
      validateVariantForm({
        form: { name: 'Hero', slug: '', description: '', weight: 10 },
        variantSpace,
        axesSelection,
        requireSlug: true,
      }),
    ).toBe('Slug is required');
  });

  it('flags a missing variant space registry', () => {
    expect(
      validateVariantForm({
        form: { name: 'Hero', slug: 'hero', description: '', weight: 10 },
        variantSpace: null,
        axesSelection,
        requireSlug: false,
      }),
    ).toBe('Variant axes registry not loaded yet');
  });

  it('reports the first axis missing a selection', () => {
    expect(
      validateVariantForm({
        form: { name: 'Hero', slug: 'hero', description: '', weight: 10 },
        variantSpace,
        axesSelection: { tone: 'bold' },
        requireSlug: false,
      }),
    ).toBe('Select a value for the audience axis');
  });

  it('returns null for a fully valid form', () => {
    expect(
      validateVariantForm({
        form: { name: 'Hero', slug: 'hero', description: '', weight: 10 },
        variantSpace,
        axesSelection,
        requireSlug: true,
      }),
    ).toBeNull();
  });
});

describe('loadVariantEditorData', () => {
  it('loads the variant and its admin sections', async () => {
    mockGetVariant.mockResolvedValue(baseVariant);
    mockGetAdminSections.mockResolvedValue([{ id: 1n }]);

    const data = await loadVariantEditorData('hero');

    expect(mockGetVariant).toHaveBeenCalledWith('hero');
    expect(mockGetAdminSections).toHaveBeenCalledWith(7n);
    expect(data.variant).toBe(baseVariant);
    expect(data.sections).toHaveLength(1);
  });

  it('throws when the variant is not found', async () => {
    mockGetVariant.mockResolvedValue(null);
    await expect(loadVariantEditorData('missing')).rejects.toThrow('Variant not found');
    expect(mockGetAdminSections).not.toHaveBeenCalled();
  });
});

describe('loadVariantSpaceDefinition', () => {
  it('delegates to getVariantSpace', () => {
    mockGetVariantSpace.mockReturnValue('space');
    expect(loadVariantSpaceDefinition()).toBe('space');
  });
});

describe('persistVariant', () => {
  it('creates a new variant from normalized form values', async () => {
    mockCreateVariant.mockResolvedValue({ slug: 'hero' });
    const result = await persistVariant({
      isNew: true,
      form: { name: ' Hero ', slug: 'Hero', description: ' about ', weight: 30 },
      axesSelection: { tone: 'bold' },
      headerConfig: {} as never,
    });

    expect(mockCreateVariant).toHaveBeenCalledWith({
      name: 'Hero',
      slug: 'hero',
      description: 'about',
      weight: 30,
      axes: { tone: 'bold' },
    });
    expect(result).toEqual({ slug: 'hero' });
  });

  it('omits an empty description when creating', async () => {
    mockCreateVariant.mockResolvedValue({});
    await persistVariant({
      isNew: true,
      form: { name: 'Hero', slug: 'hero', description: '   ', weight: 10 },
      axesSelection: {},
      headerConfig: {} as never,
    });
    expect(mockCreateVariant).toHaveBeenCalledWith(
      expect.objectContaining({ description: undefined }),
    );
  });

  it('updates an existing variant and returns undefined', async () => {
    mockUpdateVariant.mockResolvedValue(undefined);
    const headerConfig = { showNav: true } as never;
    const result = await persistVariant({
      isNew: false,
      slugFromRoute: 'hero',
      form: { name: 'Hero', slug: 'hero', description: 'about', weight: 55 },
      axesSelection: { tone: 'calm' },
      headerConfig,
    });

    expect(mockUpdateVariant).toHaveBeenCalledWith('hero', {
      name: 'Hero',
      description: 'about',
      weight: 55,
      axes: { tone: 'calm' },
      headerConfig,
    });
    expect(result).toBeUndefined();
  });

  it('throws when updating without a route slug', async () => {
    await expect(
      persistVariant({
        isNew: false,
        form: { name: 'Hero', slug: 'hero', description: '', weight: 10 },
        axesSelection: {},
        headerConfig: {} as never,
      }),
    ).rejects.toThrow('Variant slug missing');
  });
});

describe('snapshot helpers', () => {
  it('exports a snapshot by slug', () => {
    mockExportVariantSnapshot.mockReturnValue('snap');
    expect(loadVariantSnapshot('hero')).toBe('snap');
    expect(mockExportVariantSnapshot).toHaveBeenCalledWith('hero');
  });

  it('imports a snapshot payload', () => {
    const payload = { sections: [] } as never;
    mockImportVariantSnapshot.mockReturnValue('done');
    expect(persistVariantSnapshot('hero', payload)).toBe('done');
    expect(mockImportVariantSnapshot).toHaveBeenCalledWith('hero', payload);
  });
});
