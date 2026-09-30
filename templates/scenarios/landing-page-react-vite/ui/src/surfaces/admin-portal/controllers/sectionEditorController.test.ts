import { describe, it, expect, vi, beforeEach } from 'vitest';

const { mockGetSection, mockUpdateSection, mockGetVariant, mockGetVariantSpace } = vi.hoisted(() => ({
  mockGetSection: vi.fn(),
  mockUpdateSection: vi.fn(),
  mockGetVariant: vi.fn(),
  mockGetVariantSpace: vi.fn(),
}));

vi.mock('../../../shared/api', () => ({
  getSection: mockGetSection,
  updateSection: mockUpdateSection,
  getVariant: mockGetVariant,
  getVariantSpace: mockGetVariantSpace,
}));

import {
  buildFormFields,
  loadSectionEditor,
  persistExistingSectionContent,
  loadVariantContext,
} from './sectionEditorController';

const section = {
  id: 3n,
  sectionType: 'hero',
  enabled: true,
  order: 2,
  content: { title: 'Welcome' },
} as never;

beforeEach(() => {
  vi.clearAllMocks();
});

describe('buildFormFields', () => {
  it('projects a section onto editable form fields', () => {
    expect(buildFormFields(section)).toEqual({
      sectionType: 'hero',
      enabled: true,
      order: 2,
      content: { title: 'Welcome' },
    });
  });

  it('defaults content to an empty object when absent', () => {
    expect(buildFormFields({ ...(section as Record<string, unknown>), content: undefined } as never).content).toEqual({});
  });
});

describe('loadSectionEditor', () => {
  it('loads a section and derives its form', async () => {
    mockGetSection.mockResolvedValue(section);
    const state = await loadSectionEditor(3n);
    expect(mockGetSection).toHaveBeenCalledWith(3n);
    expect(state.section).toBe(section);
    expect(state.form.sectionType).toBe('hero');
  });

  it('throws when the section is missing', async () => {
    mockGetSection.mockResolvedValue(null);
    await expect(loadSectionEditor(99n)).rejects.toThrow('Section not found');
  });
});

describe('persistExistingSectionContent', () => {
  it('updates content then reloads the editor state', async () => {
    mockUpdateSection.mockResolvedValue(undefined);
    mockGetSection.mockResolvedValue(section);
    const content = { title: 'Updated' };

    const state = await persistExistingSectionContent(3n, content);

    expect(mockUpdateSection).toHaveBeenCalledWith(3n, content);
    expect(mockGetSection).toHaveBeenCalledWith(3n);
    expect(state.section).toBe(section);
  });
});

describe('loadVariantContext', () => {
  const variant = {
    id: 1n,
    slug: 'hero',
    name: 'Hero',
    axes: { jtbd: 'ship', tonePreference: 'bold' },
  } as never;

  const space = {
    _name: 'Conversion space',
    _note: 'space note',
    _agentGuidelines: ['be concise'],
    constraints: { _note: 'no dark patterns' },
    axes: {
      jtbd: {
        _note: 'Job to be done',
        variants: [
          {
            id: 'ship',
            label: 'Ship faster',
            description: 'Help teams ship',
            agentHints: ['emphasize speed'],
            examples: { headline: 'Ship today' },
          },
        ],
      },
      tonePreference: {
        variants: [{ id: 'bold', label: 'Bold' }],
      },
    },
  } as never;

  it('throws when slug is empty', async () => {
    await expect(loadVariantContext('')).rejects.toThrow('Variant slug is required');
  });

  it('throws when the variant is missing', async () => {
    mockGetVariant.mockResolvedValue(null);
    mockGetVariantSpace.mockResolvedValue(space);
    await expect(loadVariantContext('missing')).rejects.toThrow('Variant not found');
  });

  it('builds axis context with humanized labels and selection detail', async () => {
    mockGetVariant.mockResolvedValue(variant);
    mockGetVariantSpace.mockResolvedValue(space);

    const context = await loadVariantContext('hero');

    expect(context.variant).toBe(variant);
    expect(context.variantSpace).toEqual({
      name: 'Conversion space',
      note: 'space note',
      agentGuidelines: ['be concise'],
      constraintsNote: 'no dark patterns',
    });

    const jtbd = context.axes.find((axis) => axis.axisId === 'jtbd');
    expect(jtbd?.axisLabel).toBe('JTBD');
    expect(jtbd?.selectionLabel).toBe('Ship faster');
    expect(jtbd?.agentHints).toEqual(['emphasize speed']);

    const tone = context.axes.find((axis) => axis.axisId === 'tonePreference');
    expect(tone?.axisLabel).toBe('Tone Preference');
    expect(tone?.selectionLabel).toBe('Bold');
  });

  it('returns an empty axis list when the space has no axes', async () => {
    mockGetVariant.mockResolvedValue(variant);
    mockGetVariantSpace.mockResolvedValue({ _name: 'empty' } as never);
    const context = await loadVariantContext('hero');
    expect(context.axes).toEqual([]);
  });
});
