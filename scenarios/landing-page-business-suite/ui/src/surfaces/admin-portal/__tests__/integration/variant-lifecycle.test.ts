/**
 * Integration tests for variant lifecycle: load → edit → save → verify
 *
 * Tests multi-service interactions across:
 * - variantEditorController
 * - variant.service
 * - section.service
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
  loadVariantEditorData,
  hydrateFormFromVariant,
  normalizeForm,
  validateVariantForm,
  type VariantFormState,
} from '../../controllers/variantEditorController';
import * as variantsApi from '../../../../shared/api/variants';
import * as sectionsApi from '../../../../shared/api/sections';
import type { Variant, ContentSection, VariantSpace } from '../../../../shared/api';

// Mock API modules
vi.mock('../../../../shared/api/variants');
vi.mock('../../../../shared/api/sections');

const mockGetVariant = vi.mocked(variantsApi.getVariant);
const mockGetAdminSections = vi.mocked(sectionsApi.getAdminSections);

const createMockVariant = (overrides: Partial<Variant> = {}): Variant => ({
  id: 1,
  slug: 'test-variant',
  name: 'Test Variant',
  description: 'A test variant',
  status: 'active',
  weight: 50,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  ...overrides,
});

const createMockSection = (overrides: Partial<ContentSection> = {}): ContentSection => ({
  id: 1,
  variant_id: 1,
  section_type: 'hero',
  content: {},
  order: 0,
  enabled: true,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  ...overrides,
});

describe('Variant Lifecycle Integration', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Load → Edit workflow', () => {
    it('loads variant and sections, then hydrates form state', async () => {
      const mockVariant = createMockVariant({
        slug: 'hero-test',
        name: 'Hero Test',
        weight: 75,
      });
      const mockSections = [
        createMockSection({ id: 1, section_type: 'hero' }),
        createMockSection({ id: 2, section_type: 'features' }),
      ];

      mockGetVariant.mockResolvedValue(mockVariant);
      mockGetAdminSections.mockResolvedValue({ sections: mockSections });

      // Load data
      const data = await loadVariantEditorData('hero-test');

      // Verify data loaded correctly
      expect(data.variant.slug).toBe('hero-test');
      expect(data.sections).toHaveLength(2);

      // Hydrate form
      const form = hydrateFormFromVariant(data.variant);

      // Verify form hydration
      expect(form.name).toBe('Hero Test');
      expect(form.slug).toBe('hero-test');
      expect(form.weight).toBe(75);
    });

    it('handles variant without sections gracefully', async () => {
      const mockVariant = createMockVariant();
      mockGetVariant.mockResolvedValue(mockVariant);
      mockGetAdminSections.mockResolvedValue({ sections: [] });

      const data = await loadVariantEditorData('test-variant');

      expect(data.variant).toBeDefined();
      expect(data.sections).toHaveLength(0);
    });

    it('throws error when variant has no ID', async () => {
      const invalidVariant = { ...createMockVariant(), id: undefined } as unknown as Variant;
      mockGetVariant.mockResolvedValue(invalidVariant);

      await expect(loadVariantEditorData('test')).rejects.toThrow('Variant payload missing ID');
    });
  });

  describe('Form validation workflow', () => {
    const mockVariantSpace: VariantSpace = {
      _name: 'Test Variant Space',
      _schemaVersion: 1,
      axes: {
        audience: {
          variants: [{ id: 'b2b', label: 'B2B' }, { id: 'b2c', label: 'B2C' }],
        },
      },
    };

    it('validates complete form successfully', () => {
      const form: VariantFormState = {
        name: 'Test Variant',
        slug: 'test-variant',
        description: 'Description',
        weight: 50,
      };

      const error = validateVariantForm({
        form,
        variantSpace: mockVariantSpace,
        axesSelection: { audience: 'b2b' },
        requireSlug: true,
      });

      expect(error).toBeNull();
    });

    it('fails validation without name', () => {
      const form: VariantFormState = {
        name: '',
        slug: 'test-variant',
        description: '',
        weight: 50,
      };

      const error = validateVariantForm({
        form,
        variantSpace: mockVariantSpace,
        axesSelection: { audience: 'b2b' },
        requireSlug: true,
      });

      expect(error).toBe('Name is required');
    });

    it('fails validation without required slug', () => {
      const form: VariantFormState = {
        name: 'Test',
        slug: '',
        description: '',
        weight: 50,
      };

      const error = validateVariantForm({
        form,
        variantSpace: mockVariantSpace,
        axesSelection: { audience: 'b2b' },
        requireSlug: true,
      });

      expect(error).toBe('Slug is required');
    });

    it('fails validation without axis selection', () => {
      const form: VariantFormState = {
        name: 'Test',
        slug: 'test',
        description: '',
        weight: 50,
      };

      const error = validateVariantForm({
        form,
        variantSpace: mockVariantSpace,
        axesSelection: {},
        requireSlug: true,
      });

      expect(error).toBe('Select a value for the audience axis');
    });
  });

  describe('Form normalization', () => {
    it('trims whitespace from fields', () => {
      const form: VariantFormState = {
        name: '  Test Name  ',
        slug: '  test-slug  ',
        description: '  Description  ',
        weight: 50,
      };

      const normalized = normalizeForm(form);

      expect(normalized.name).toBe('Test Name');
      expect(normalized.slug).toBe('test-slug');
      expect(normalized.description).toBe('Description');
    });

    it('sanitizes slug to lowercase alphanumeric with hyphens', () => {
      const form: VariantFormState = {
        name: 'Test',
        slug: 'Test_Slug!@#123',
        description: '',
        weight: 50,
      };

      const normalized = normalizeForm(form);

      expect(normalized.slug).toBe('testslug123');
    });
  });
});
