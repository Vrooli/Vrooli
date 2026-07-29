import { describe, it, expect, vi, beforeEach } from 'vitest';
import type {
  ContentSection,
  Variant,
  VariantSpace,
  VariantAxes,
  getSection,
  updateSection,
  patchSection,
  getVariant,
  getVariantSpace,
} from '../../../shared/api';
import {
  buildFormFields,
  loadSectionEditor,
  persistExistingSectionContent,
  updateSectionOrder,
  loadVariantContext,
  type SectionFormFields,
} from './sectionEditorController';

// Mock the API module
type GetSectionFn = typeof getSection;
type UpdateSectionFn = typeof updateSection;
type PatchSectionFn = typeof patchSection;
type GetVariantFn = typeof getVariant;
type GetVariantSpaceFn = typeof getVariantSpace;

const getSectionMock = vi.fn<GetSectionFn>();
const updateSectionMock = vi.fn<UpdateSectionFn>();
const patchSectionMock = vi.fn<PatchSectionFn>();
const getVariantMock = vi.fn<GetVariantFn>();
const getVariantSpaceMock = vi.fn<GetVariantSpaceFn>();

vi.mock('../../../shared/api', async () => {
  const actual = await vi.importActual<typeof import('../../../shared/api')>('../../../shared/api');
  return {
    ...actual,
    getSection: (...args: Parameters<GetSectionFn>) => getSectionMock(...args),
    updateSection: (...args: Parameters<UpdateSectionFn>) => updateSectionMock(...args),
    patchSection: (...args: Parameters<PatchSectionFn>) => patchSectionMock(...args),
    getVariant: (...args: Parameters<GetVariantFn>) => getVariantMock(...args),
    getVariantSpace: (...args: Parameters<GetVariantSpaceFn>) => getVariantSpaceMock(...args),
  };
});

const mockSection: ContentSection = {
  id: 42,
  variant_id: 1,
  section_type: 'hero',
  content: { title: 'Welcome', subtitle: 'Get started today' },
  order: 1,
  enabled: true,
  created_at: '2025-01-01T00:00:00Z',
  updated_at: '2025-01-01T00:00:00Z',
};

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

const mockVariantSpace: VariantSpace = {
  _name: 'Test Space',
  _schemaVersion: 1,
  _note: 'Test variant space',
  _agentGuidelines: ['Guideline 1', 'Guideline 2'],
  axes: {
    jtbd: {
      _note: 'Job to be done',
      variants: [
        { id: 'automation', label: 'Automation', description: 'Automate tasks', agentHints: ['Focus on efficiency'] },
        { id: 'analytics', label: 'Analytics', description: 'Analyze data' },
      ],
    },
    industry: {
      _note: 'Target industry',
      variants: [
        { id: 'tech', label: 'Technology' },
        { id: 'finance', label: 'Finance' },
      ],
    },
  },
  constraints: {
    _note: 'Some constraints',
  },
};

describe('sectionEditorController', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('buildFormFields', () => {
    it('extracts form fields from section', () => {
      const result = buildFormFields(mockSection);

      expect(result.sectionType).toBe('hero');
      expect(result.enabled).toBe(true);
      expect(result.order).toBe(1);
      expect(result.content).toEqual({ title: 'Welcome', subtitle: 'Get started today' });
    });

    it('handles null content', () => {
      const sectionWithNullContent: ContentSection = {
        ...mockSection,
        content: null as unknown as Record<string, unknown>,
      };

      const result = buildFormFields(sectionWithNullContent);

      expect(result.content).toEqual({});
    });

    it('handles undefined content', () => {
      const sectionWithUndefinedContent: ContentSection = {
        ...mockSection,
        content: undefined as unknown as Record<string, unknown>,
      };

      const result = buildFormFields(sectionWithUndefinedContent);

      expect(result.content).toEqual({});
    });
  });

  describe('loadSectionEditor', () => {
    it('fetches section and returns editor state', async () => {
      getSectionMock.mockResolvedValue(mockSection);

      const result = await loadSectionEditor(42);

      expect(getSectionMock).toHaveBeenCalledWith(42);
      expect(result.section).toEqual(mockSection);
      expect(result.form.sectionType).toBe('hero');
      expect(result.form.enabled).toBe(true);
    });

    it('propagates API errors', async () => {
      getSectionMock.mockRejectedValue(new Error('Section not found'));

      await expect(loadSectionEditor(999)).rejects.toThrow('Section not found');
    });
  });

  describe('persistExistingSectionContent', () => {
    it('updates section and reloads editor state', async () => {
      updateSectionMock.mockResolvedValue({ success: true, message: 'Updated' });
      getSectionMock.mockResolvedValue({
        ...mockSection,
        content: { title: 'Updated', subtitle: 'New subtitle' },
      });

      const newContent = { title: 'Updated', subtitle: 'New subtitle' };
      const result = await persistExistingSectionContent(42, newContent);

      expect(updateSectionMock).toHaveBeenCalledWith(42, newContent);
      expect(getSectionMock).toHaveBeenCalledWith(42);
      expect(result.form.content.title).toBe('Updated');
    });

    it('propagates update errors', async () => {
      updateSectionMock.mockRejectedValue(new Error('Update failed'));

      await expect(persistExistingSectionContent(42, {})).rejects.toThrow('Update failed');
    });
  });

  describe('updateSectionOrder', () => {
    it('patches section with new order', async () => {
      patchSectionMock.mockResolvedValue({ success: true });

      await updateSectionOrder(42, 5);

      expect(patchSectionMock).toHaveBeenCalledWith(42, { order: 5 });
    });

    it('throws error when sectionId is falsy', async () => {
      await expect(updateSectionOrder(0, 5)).rejects.toThrow('Section ID and order are required');
    });

    it('throws error when order is NaN', async () => {
      await expect(updateSectionOrder(42, NaN)).rejects.toThrow('Section ID and order are required');
    });

    it('propagates API errors', async () => {
      patchSectionMock.mockRejectedValue(new Error('Patch failed'));

      await expect(updateSectionOrder(42, 5)).rejects.toThrow('Patch failed');
    });
  });

  describe('loadVariantContext', () => {
    it('fetches variant and space, builds context', async () => {
      getVariantMock.mockResolvedValue(mockVariant);
      getVariantSpaceMock.mockResolvedValue(mockVariantSpace);

      const result = await loadVariantContext('control');

      expect(getVariantMock).toHaveBeenCalledWith('control');
      expect(getVariantSpaceMock).toHaveBeenCalled();
      expect(result.variant).toEqual(mockVariant);
      expect(result.variantSpace.name).toBe('Test Space');
    });

    it('throws error when slug is empty', async () => {
      await expect(loadVariantContext('')).rejects.toThrow('Variant slug is required');
    });

    it('builds axis context with selections', async () => {
      getVariantMock.mockResolvedValue(mockVariant);
      getVariantSpaceMock.mockResolvedValue(mockVariantSpace);

      const result = await loadVariantContext('control');

      expect(result.axes).toHaveLength(2);

      const jtbdAxis = result.axes.find((a) => a.axisId === 'jtbd');
      expect(jtbdAxis?.axisLabel).toBe('JTBD');
      expect(jtbdAxis?.selectionId).toBe('automation');
      expect(jtbdAxis?.selectionLabel).toBe('Automation');
      expect(jtbdAxis?.selectionDescription).toBe('Automate tasks');
      expect(jtbdAxis?.agentHints).toEqual(['Focus on efficiency']);

      const industryAxis = result.axes.find((a) => a.axisId === 'industry');
      expect(industryAxis?.axisLabel).toBe('Industry');
      expect(industryAxis?.selectionId).toBe('tech');
    });

    it('formats axis labels correctly', async () => {
      getVariantMock.mockResolvedValue(mockVariant);
      const jtbdAxis = mockVariantSpace.axes.jtbd;
      if (!jtbdAxis) {
        throw new Error('Expected jtbd axis to be defined');
      }
      getVariantSpaceMock.mockResolvedValue({
        ...mockVariantSpace,
        axes: {
          jtbd: jtbdAxis,
          targetAudience: {
            _note: 'Target audience',
            variants: [{ id: 'developers', label: 'Developers' }],
          },
        },
      });

      const result = await loadVariantContext('control');

      const targetAxis = result.axes.find((a) => a.axisId === 'targetAudience');
      expect(targetAxis?.axisLabel).toBe('Target Audience');
    });

    it('handles missing variant axes', async () => {
      getVariantMock.mockResolvedValue({ ...mockVariant, axes: undefined });
      getVariantSpaceMock.mockResolvedValue(mockVariantSpace);

      const result = await loadVariantContext('control');

      expect(result.axes).toHaveLength(2);
      const jtbdAxis = result.axes.find((a) => a.axisId === 'jtbd');
      expect(jtbdAxis?.selectionId).toBeUndefined();
    });

    it('extracts variant space summary', async () => {
      getVariantMock.mockResolvedValue(mockVariant);
      getVariantSpaceMock.mockResolvedValue(mockVariantSpace);

      const result = await loadVariantContext('control');

      expect(result.variantSpace.name).toBe('Test Space');
      expect(result.variantSpace.note).toBe('Test variant space');
      expect(result.variantSpace.agentGuidelines).toEqual(['Guideline 1', 'Guideline 2']);
      expect(result.variantSpace.constraintsNote).toBe('Some constraints');
    });

    it('handles missing variant space fields', async () => {
      getVariantMock.mockResolvedValue(mockVariant);
      getVariantSpaceMock.mockResolvedValue({
        _name: 'Minimal Space',
        _schemaVersion: 1,
        axes: {},
      });

      const result = await loadVariantContext('control');

      expect(result.variantSpace.name).toBe('Minimal Space');
      expect(result.variantSpace.note).toBeUndefined();
      expect(result.variantSpace.agentGuidelines).toBeUndefined();
    });

    it('handles empty axes in variant space', async () => {
      getVariantMock.mockResolvedValue(mockVariant);
      getVariantSpaceMock.mockResolvedValue({
        ...mockVariantSpace,
        axes: {},
      });

      const result = await loadVariantContext('control');

      expect(result.axes).toEqual([]);
    });

    it('uses selection ID as label when variant not found', async () => {
      getVariantMock.mockResolvedValue({
        ...mockVariant,
        axes: { jtbd: 'unknown_value' },
      });
      getVariantSpaceMock.mockResolvedValue(mockVariantSpace);

      const result = await loadVariantContext('control');

      const jtbdAxis = result.axes.find((a) => a.axisId === 'jtbd');
      expect(jtbdAxis?.selectionId).toBe('unknown_value');
      expect(jtbdAxis?.selectionLabel).toBe('unknown_value');
    });
  });
});
