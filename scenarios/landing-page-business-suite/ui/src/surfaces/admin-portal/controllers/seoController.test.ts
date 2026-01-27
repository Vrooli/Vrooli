import { describe, it, expect, vi, beforeEach } from 'vitest';
import type {
  SiteBranding,
  VariantSEOConfig,
  VariantSEOResponse,
  getVariantSEO,
  updateVariantSEO,
} from '../../../shared/api';
import { buildEditableSEOConfig, loadVariantSEOConfig, saveVariantSEOConfig } from './seoController';

// Mock the API functions
type GetVariantSEOFn = typeof getVariantSEO;
type UpdateVariantSEOFn = typeof updateVariantSEO;

const getVariantSEOMock = vi.fn<Parameters<GetVariantSEOFn>, ReturnType<GetVariantSEOFn>>();
const updateVariantSEOMock = vi.fn<Parameters<UpdateVariantSEOFn>, ReturnType<UpdateVariantSEOFn>>();

vi.mock('../../../shared/api', async () => {
  const actual = await vi.importActual<typeof import('../../../shared/api')>('../../../shared/api');
  return {
    ...actual,
    getVariantSEO: (...args: Parameters<GetVariantSEOFn>) => getVariantSEOMock(...args),
    updateVariantSEO: (...args: Parameters<UpdateVariantSEOFn>) => updateVariantSEOMock(...args),
  };
});

describe('seoController', () => {
  const branding: SiteBranding = {
    id: 1,
    site_name: 'Landing Suite',
    default_title: 'Default Title',
    default_description: 'Default description',
    default_og_image_url: 'https://cdn.example.com/default.png',
    theme_primary_color: '#000',
  };

  const mockSEOResponse: VariantSEOResponse = {
    site_name: 'Landing Suite',
    title: 'Custom Title',
    description: 'Custom description',
    og_title: 'OG Title',
    og_description: 'OG Description',
    og_image_url: 'https://cdn.example.com/custom.png',
    twitter_card: 'summary',
    canonical_url: 'https://example.com/custom',
    noindex: true,
    structured_data: { '@type': 'Product', name: 'Suite' },
  };

  beforeEach(() => {
    vi.clearAllMocks();
    getVariantSEOMock.mockResolvedValue(mockSEOResponse);
    updateVariantSEOMock.mockResolvedValue({});
  });

  describe('buildEditableSEOConfig', () => {
    it('strips branding defaults from editable config', () => {
      const response: VariantSEOResponse = {
        site_name: 'Landing Suite',
        title: 'Default Title',
        description: 'Default description',
        og_title: '',
        og_description: '',
        og_image_url: 'https://cdn.example.com/default.png',
        twitter_card: 'summary_large_image',
        canonical_url: 'https://example.com/',
        noindex: false,
        structured_data: { '@type': 'WebPage' },
      };

      const config = buildEditableSEOConfig(response, branding);

      expect(config.title).toBeUndefined();
      expect(config.description).toBeUndefined();
      expect(config.og_image_url).toBeUndefined();
      expect(config.noindex).toBe(false);
    });

    it('keeps overrides when provided', () => {
      const config = buildEditableSEOConfig(mockSEOResponse, branding);

      expect(config.title).toBe('Custom Title');
      expect(config.description).toBe('Custom description');
      expect(config.og_title).toBe('OG Title');
      expect(config.og_description).toBe('OG Description');
      expect(config.og_image_url).toBe('https://cdn.example.com/custom.png');
      expect(config.twitter_card).toBe('summary');
      expect(config.noindex).toBe(true);
      expect(config.structured_data).toEqual({ '@type': 'Product', name: 'Suite' });
    });

    it('handles null branding', () => {
      const response: VariantSEOResponse = {
        site_name: 'Landing Suite',
        title: 'My Title',
        description: 'My description',
        og_title: '',
        og_description: '',
        canonical_url: 'https://example.com/',
        noindex: false,
      };

      const config = buildEditableSEOConfig(response, null);

      expect(config.title).toBe('My Title');
      expect(config.description).toBe('My description');
    });

    it('handles empty strings as undefined', () => {
      const response: VariantSEOResponse = {
        site_name: 'Landing Suite',
        title: '  ',
        description: '',
        og_title: '',
        og_description: '',
        canonical_url: 'https://example.com/',
        noindex: false,
      };

      const config = buildEditableSEOConfig(response, null);

      expect(config.title).toBeUndefined();
      expect(config.description).toBeUndefined();
    });

    it('strips whitespace from values', () => {
      const response: VariantSEOResponse = {
        site_name: 'Landing Suite',
        title: '  Custom Title  ',
        description: '  Custom description  ',
        og_title: '',
        og_description: '',
        canonical_url: 'https://example.com/',
        noindex: false,
      };

      const config = buildEditableSEOConfig(response, branding);

      expect(config.title).toBe('Custom Title');
      expect(config.description).toBe('Custom description');
    });

    it('returns undefined for canonical_path (not exposed by API)', () => {
      const config = buildEditableSEOConfig(mockSEOResponse, branding);

      expect(config.canonical_path).toBeUndefined();
    });

    it('preserves structured_data when present', () => {
      const response: VariantSEOResponse = {
        site_name: 'Landing Suite',
        title: 'Title',
        description: '',
        og_title: '',
        og_description: '',
        canonical_url: 'https://example.com/',
        noindex: false,
        structured_data: { '@type': 'Organization', name: 'My Org' },
      };

      const config = buildEditableSEOConfig(response, branding);

      expect(config.structured_data).toEqual({ '@type': 'Organization', name: 'My Org' });
    });

    it('returns undefined for structured_data when null', () => {
      const response: VariantSEOResponse = {
        site_name: 'Landing Suite',
        title: 'Title',
        description: '',
        og_title: '',
        og_description: '',
        canonical_url: 'https://example.com/',
        noindex: false,
        structured_data: null as unknown as Record<string, unknown> | undefined,
      };

      const config = buildEditableSEOConfig(response, branding);

      expect(config.structured_data).toBeUndefined();
    });
  });

  describe('loadVariantSEOConfig', () => {
    it('fetches SEO config and transforms it', async () => {
      const result = await loadVariantSEOConfig('control', branding);

      expect(getVariantSEOMock).toHaveBeenCalledWith('control');
      expect(result.title).toBe('Custom Title');
      expect(result.description).toBe('Custom description');
    });

    it('passes branding to strip defaults', async () => {
      const matchingBranding: SiteBranding = {
        id: 1,
        site_name: 'Landing Suite',
        default_title: 'Custom Title',
        default_description: 'Custom description',
        theme_primary_color: '#000',
      };

      const result = await loadVariantSEOConfig('control', matchingBranding);

      expect(result.title).toBeUndefined();
      expect(result.description).toBeUndefined();
    });

    it('handles API errors', async () => {
      getVariantSEOMock.mockRejectedValue(new Error('Network error'));

      await expect(loadVariantSEOConfig('control', branding)).rejects.toThrow('Network error');
    });

    it('works without branding', async () => {
      const result = await loadVariantSEOConfig('control');

      expect(getVariantSEOMock).toHaveBeenCalledWith('control');
      expect(result.title).toBe('Custom Title');
    });
  });

  describe('saveVariantSEOConfig', () => {
    it('calls updateVariantSEO with slug and config', async () => {
      const config: VariantSEOConfig = {
        title: 'New Title',
        description: 'New Description',
        noindex: false,
      };

      await saveVariantSEOConfig('control', config);

      expect(updateVariantSEOMock).toHaveBeenCalledWith('control', config);
    });

    it('handles API errors', async () => {
      updateVariantSEOMock.mockRejectedValue(new Error('Save failed'));

      const config: VariantSEOConfig = {
        title: 'New Title',
        noindex: false,
      };

      await expect(saveVariantSEOConfig('control', config)).rejects.toThrow('Save failed');
    });

    it('saves empty config', async () => {
      const config: VariantSEOConfig = {
        noindex: false,
      };

      await saveVariantSEOConfig('control', config);

      expect(updateVariantSEOMock).toHaveBeenCalledWith('control', config);
    });

    it('saves full config', async () => {
      const config: VariantSEOConfig = {
        title: 'Full Title',
        description: 'Full Description',
        og_title: 'OG Full',
        og_description: 'OG Desc',
        og_image_url: 'https://example.com/og.png',
        twitter_card: 'summary_large_image',
        canonical_path: '/custom',
        noindex: true,
        structured_data: { '@type': 'WebPage' },
      };

      await saveVariantSEOConfig('test-variant', config);

      expect(updateVariantSEOMock).toHaveBeenCalledWith('test-variant', config);
    });
  });
});
