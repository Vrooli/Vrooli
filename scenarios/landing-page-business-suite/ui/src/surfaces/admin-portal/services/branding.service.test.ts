import { describe, it, expect, vi, beforeEach } from 'vitest';
import type {
  SiteBranding,
  Asset,
  getBranding,
  updateBranding,
  clearBrandingField,
} from '../../../shared/api';
import {
  brandingToForm,
  formToBrandingPayload,
  isBrandingDirty,
  computeBrandingHealth,
  selectLogoDerivatives,
  selectFaviconDerivatives,
  selectOgDerivatives,
  loadBranding,
  saveBranding,
  clearField,
  formatFieldName,
  DEFAULT_BRANDING_FORM,
  type BrandingFormState,
} from './branding.service';

// Mock the API module
type GetBrandingFn = typeof getBranding;
type UpdateBrandingFn = typeof updateBranding;
type ClearBrandingFieldFn = typeof clearBrandingField;

const getBrandingMock = vi.fn<GetBrandingFn>();
const updateBrandingMock = vi.fn<UpdateBrandingFn>();
const clearBrandingFieldMock = vi.fn<ClearBrandingFieldFn>();

vi.mock('../../../shared/api', async () => {
  const actual = await vi.importActual<typeof import('../../../shared/api')>('../../../shared/api');
  return {
    ...actual,
    getBranding: (...args: Parameters<GetBrandingFn>) => getBrandingMock(...args),
    updateBranding: (...args: Parameters<UpdateBrandingFn>) => updateBrandingMock(...args),
    clearBrandingField: (...args: Parameters<ClearBrandingFieldFn>) => clearBrandingFieldMock(...args),
  };
});

const mockBranding: SiteBranding = {
  id: 1,
  site_name: 'Test Site',
  tagline: 'A test tagline',
  logo_url: 'https://example.com/logo.png',
  logo_icon_url: 'https://example.com/icon.png',
  favicon_url: 'https://example.com/favicon.ico',
  apple_touch_icon_url: 'https://example.com/apple-touch.png',
  default_title: 'Test Site | Home',
  default_description: 'This is a test site description',
  default_og_image_url: 'https://example.com/og.png',
  theme_primary_color: '#3B82F6',
  theme_background_color: '#07090F',
  canonical_base_url: 'https://example.com',
  google_site_verification: 'verification-code',
  robots_txt: 'User-agent: *\nAllow: /',
  support_chat_url: 'https://chat.example.com',
  support_email: 'support@example.com',
  smtp_host: 'smtp.example.com',
  smtp_port: 587,
  smtp_username: 'smtp-user',
  smtp_password: 'smtp-pass',
  smtp_from: 'noreply@example.com',
  coming_soon_enabled: false,
  coming_soon_message: '',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

const mockAsset: Asset = {
  id: 1,
  filename: 'logo.png',
  original_filename: 'my-logo.png',
  mime_type: 'image/png',
  size_bytes: 12345,
  storage_path: '/uploads/logo.png',
  category: 'logo',
  created_at: '2024-01-01T00:00:00Z',
  url: 'https://example.com/uploads/logo.png',
  derivatives: {
    logo_512: 'https://example.com/uploads/logo-512.png',
    logo_256: 'https://example.com/uploads/logo-256.png',
    logo_128: 'https://example.com/uploads/logo-128.png',
    logo_icon: 'https://example.com/uploads/logo-icon.png',
    favicon: 'https://example.com/uploads/favicon.ico',
    favicon_32: 'https://example.com/uploads/favicon-32.png',
    favicon_64: 'https://example.com/uploads/favicon-64.png',
    apple_touch_180: 'https://example.com/uploads/apple-touch-180.png',
    og_image_1200x630: 'https://example.com/uploads/og-1200x630.png',
  },
};

describe('branding.service', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('DEFAULT_BRANDING_FORM', () => {
    it('has expected default values', () => {
      expect(DEFAULT_BRANDING_FORM.site_name).toBe('');
      expect(DEFAULT_BRANDING_FORM.smtp_port).toBe('587');
      expect(DEFAULT_BRANDING_FORM.coming_soon_enabled).toBe(false);
    });
  });

  describe('brandingToForm', () => {
    it('converts SiteBranding to form state', () => {
      const form = brandingToForm(mockBranding);

      expect(form.site_name).toBe('Test Site');
      expect(form.tagline).toBe('A test tagline');
      expect(form.logo_url).toBe('https://example.com/logo.png');
      expect(form.smtp_port).toBe('587');
      expect(form.coming_soon_enabled).toBe(false);
    });

    it('handles null values with defaults', () => {
      const partialBranding: SiteBranding = {
        id: 1,
        site_name: 'Minimal Site',
        tagline: null,
        logo_url: null,
        smtp_port: null,
      };

      const form = brandingToForm(partialBranding);

      expect(form.site_name).toBe('Minimal Site');
      expect(form.tagline).toBe('');
      expect(form.logo_url).toBe('');
      expect(form.smtp_port).toBe('587');
    });

    it('converts smtp_port number to string', () => {
      const branding: SiteBranding = {
        id: 1,
        site_name: 'Test',
        smtp_port: 465,
      };

      const form = brandingToForm(branding);

      expect(form.smtp_port).toBe('465');
    });

    it('handles coming_soon_enabled boolean', () => {
      const enabledBranding: SiteBranding = {
        id: 1,
        site_name: 'Test',
        coming_soon_enabled: true,
        coming_soon_message: 'Coming soon!',
      };

      const form = brandingToForm(enabledBranding);

      expect(form.coming_soon_enabled).toBe(true);
      expect(form.coming_soon_message).toBe('Coming soon!');
    });
  });

  describe('formToBrandingPayload', () => {
    it('returns only changed fields', () => {
      const original: BrandingFormState = { ...DEFAULT_BRANDING_FORM };
      const form: BrandingFormState = {
        ...DEFAULT_BRANDING_FORM,
        site_name: 'New Site',
        tagline: 'New tagline',
      };

      const payload = formToBrandingPayload(form, original);

      expect(payload).toEqual({
        site_name: 'New Site',
        tagline: 'New tagline',
      });
    });

    it('converts smtp_port to number', () => {
      const original: BrandingFormState = { ...DEFAULT_BRANDING_FORM };
      const form: BrandingFormState = {
        ...DEFAULT_BRANDING_FORM,
        smtp_port: '465',
      };

      const payload = formToBrandingPayload(form, original);

      expect(payload).toEqual({ smtp_port: 465 });
    });

    it('handles boolean fields (coming_soon_enabled)', () => {
      const original: BrandingFormState = { ...DEFAULT_BRANDING_FORM, coming_soon_enabled: false };
      const form: BrandingFormState = { ...DEFAULT_BRANDING_FORM, coming_soon_enabled: true };

      const payload = formToBrandingPayload(form, original);

      expect(payload).toEqual({ coming_soon_enabled: true });
    });

    it('excludes empty string changes', () => {
      const original: BrandingFormState = { ...DEFAULT_BRANDING_FORM, site_name: 'Old Name' };
      const form: BrandingFormState = { ...DEFAULT_BRANDING_FORM, site_name: '' };

      const payload = formToBrandingPayload(form, original);

      expect(payload).toEqual({});
    });

    it('trims whitespace from values', () => {
      const original: BrandingFormState = { ...DEFAULT_BRANDING_FORM };
      const form: BrandingFormState = {
        ...DEFAULT_BRANDING_FORM,
        site_name: '  New Site  ',
      };

      const payload = formToBrandingPayload(form, original);

      expect(payload).toEqual({ site_name: 'New Site' });
    });

    it('returns empty object when no changes', () => {
      const original: BrandingFormState = { ...DEFAULT_BRANDING_FORM, site_name: 'Same' };
      const form: BrandingFormState = { ...DEFAULT_BRANDING_FORM, site_name: 'Same' };

      const payload = formToBrandingPayload(form, original);

      expect(payload).toEqual({});
    });
  });

  describe('isBrandingDirty', () => {
    it('returns false when forms are identical', () => {
      const form: BrandingFormState = { ...DEFAULT_BRANDING_FORM, site_name: 'Test' };
      const original: BrandingFormState = { ...DEFAULT_BRANDING_FORM, site_name: 'Test' };

      expect(isBrandingDirty(form, original)).toBe(false);
    });

    it('returns true when forms differ', () => {
      const form: BrandingFormState = { ...DEFAULT_BRANDING_FORM, site_name: 'Changed' };
      const original: BrandingFormState = { ...DEFAULT_BRANDING_FORM, site_name: 'Original' };

      expect(isBrandingDirty(form, original)).toBe(true);
    });

    it('returns true when boolean field differs', () => {
      const form: BrandingFormState = { ...DEFAULT_BRANDING_FORM, coming_soon_enabled: true };
      const original: BrandingFormState = { ...DEFAULT_BRANDING_FORM, coming_soon_enabled: false };

      expect(isBrandingDirty(form, original)).toBe(true);
    });
  });

  describe('computeBrandingHealth', () => {
    it('returns all checks false for empty form', () => {
      const health = computeBrandingHealth(DEFAULT_BRANDING_FORM);

      expect(health.checks.identity).toBe(false);
      expect(health.checks.favicon).toBe(false);
      expect(health.checks.seo).toBe(false);
      expect(health.checks.ogImage).toBe(false);
      expect(health.configured).toBe(0);
      expect(health.total).toBe(4);
      expect(health.percentage).toBe(0);
    });

    it('returns identity true when site_name and logo_url are set', () => {
      const form: BrandingFormState = {
        ...DEFAULT_BRANDING_FORM,
        site_name: 'Test',
        logo_url: 'https://example.com/logo.png',
      };

      const health = computeBrandingHealth(form);

      expect(health.checks.identity).toBe(true);
      expect(health.configured).toBe(1);
      expect(health.percentage).toBe(25);
    });

    it('returns favicon true when favicon_url is set', () => {
      const form: BrandingFormState = {
        ...DEFAULT_BRANDING_FORM,
        favicon_url: 'https://example.com/favicon.ico',
      };

      const health = computeBrandingHealth(form);

      expect(health.checks.favicon).toBe(true);
    });

    it('returns seo true when both title and description are set', () => {
      const form: BrandingFormState = {
        ...DEFAULT_BRANDING_FORM,
        default_title: 'Page Title',
        default_description: 'Page description',
      };

      const health = computeBrandingHealth(form);

      expect(health.checks.seo).toBe(true);
    });

    it('returns seo false when only title is set', () => {
      const form: BrandingFormState = {
        ...DEFAULT_BRANDING_FORM,
        default_title: 'Page Title',
      };

      const health = computeBrandingHealth(form);

      expect(health.checks.seo).toBe(false);
    });

    it('returns ogImage true when og_image_url is set', () => {
      const form: BrandingFormState = {
        ...DEFAULT_BRANDING_FORM,
        default_og_image_url: 'https://example.com/og.png',
      };

      const health = computeBrandingHealth(form);

      expect(health.checks.ogImage).toBe(true);
    });

    it('returns 100% when all checks pass', () => {
      const form: BrandingFormState = {
        ...DEFAULT_BRANDING_FORM,
        site_name: 'Test',
        logo_url: 'https://example.com/logo.png',
        favicon_url: 'https://example.com/favicon.ico',
        default_title: 'Title',
        default_description: 'Description',
        default_og_image_url: 'https://example.com/og.png',
      };

      const health = computeBrandingHealth(form);

      expect(health.configured).toBe(4);
      expect(health.percentage).toBe(100);
    });
  });

  describe('selectLogoDerivatives', () => {
    it('selects logo_512 as primary logo when available', () => {
      const form: BrandingFormState = { ...DEFAULT_BRANDING_FORM };
      const result = selectLogoDerivatives(mockAsset, form);

      expect(result.logo_url).toBe('https://example.com/uploads/logo-512.png');
    });

    it('selects logo_icon for icon when available', () => {
      const form: BrandingFormState = { ...DEFAULT_BRANDING_FORM };
      const result = selectLogoDerivatives(mockAsset, form);

      expect(result.logo_icon_url).toBe('https://example.com/uploads/logo-icon.png');
    });

    it('selects favicon_32 for favicon when available', () => {
      const form: BrandingFormState = { ...DEFAULT_BRANDING_FORM };
      const result = selectLogoDerivatives(mockAsset, form);

      expect(result.favicon_url).toBe('https://example.com/uploads/favicon-32.png');
    });

    it('selects apple_touch_180 for touch icon', () => {
      const form: BrandingFormState = { ...DEFAULT_BRANDING_FORM };
      const result = selectLogoDerivatives(mockAsset, form);

      expect(result.apple_touch_icon_url).toBe('https://example.com/uploads/apple-touch-180.png');
    });

    it('falls back to asset.url when no logo derivatives', () => {
      const assetNoDerivatives: Asset = {
        ...mockAsset,
        derivatives: undefined,
      };
      const form: BrandingFormState = { ...DEFAULT_BRANDING_FORM };

      const result = selectLogoDerivatives(assetNoDerivatives, form);

      expect(result.logo_url).toBe('https://example.com/uploads/logo.png');
    });

    it('falls back to current form values when no derivatives', () => {
      const assetNoDerivatives: Asset = {
        ...mockAsset,
        derivatives: {},
      };
      const form: BrandingFormState = {
        ...DEFAULT_BRANDING_FORM,
        favicon_url: 'https://example.com/existing-favicon.ico',
        apple_touch_icon_url: 'https://example.com/existing-touch.png',
      };

      const result = selectLogoDerivatives(assetNoDerivatives, form);

      expect(result.favicon_url).toBe('https://example.com/existing-favicon.ico');
      expect(result.apple_touch_icon_url).toBe('https://example.com/existing-touch.png');
    });

    it('cascades fallbacks for icon (logo_256 -> logo_128)', () => {
      const assetPartialDerivatives: Asset = {
        ...mockAsset,
        derivatives: {
          logo_256: 'https://example.com/uploads/logo-256.png',
        },
      };
      const form: BrandingFormState = { ...DEFAULT_BRANDING_FORM };

      const result = selectLogoDerivatives(assetPartialDerivatives, form);

      expect(result.logo_icon_url).toBe('https://example.com/uploads/logo-256.png');
    });
  });

  describe('selectFaviconDerivatives', () => {
    it('selects favicon derivative when available', () => {
      const form: BrandingFormState = { ...DEFAULT_BRANDING_FORM };
      const result = selectFaviconDerivatives(mockAsset, form);

      expect(result.favicon_url).toBe('https://example.com/uploads/favicon.ico');
    });

    it('selects apple_touch_180 for touch icon', () => {
      const form: BrandingFormState = { ...DEFAULT_BRANDING_FORM };
      const result = selectFaviconDerivatives(mockAsset, form);

      expect(result.apple_touch_icon_url).toBe('https://example.com/uploads/apple-touch-180.png');
    });

    it('falls back to favicon_32 when favicon not available', () => {
      const assetWithFavicon32: Asset = {
        ...mockAsset,
        derivatives: {
          favicon_32: 'https://example.com/uploads/favicon-32.png',
        },
      };
      const form: BrandingFormState = { ...DEFAULT_BRANDING_FORM };

      const result = selectFaviconDerivatives(assetWithFavicon32, form);

      expect(result.favicon_url).toBe('https://example.com/uploads/favicon-32.png');
    });

    it('falls back to asset.url when no favicon derivatives', () => {
      const assetNoFavicons: Asset = {
        ...mockAsset,
        derivatives: {},
      };
      const form: BrandingFormState = { ...DEFAULT_BRANDING_FORM };

      const result = selectFaviconDerivatives(assetNoFavicons, form);

      expect(result.favicon_url).toBe('https://example.com/uploads/logo.png');
    });

    it('uses favicon for touch icon when apple_touch not available', () => {
      const assetNoTouch: Asset = {
        ...mockAsset,
        derivatives: {
          favicon: 'https://example.com/uploads/favicon.ico',
        },
      };
      const form: BrandingFormState = { ...DEFAULT_BRANDING_FORM };

      const result = selectFaviconDerivatives(assetNoTouch, form);

      expect(result.apple_touch_icon_url).toBe('https://example.com/uploads/favicon.ico');
    });
  });

  describe('selectOgDerivatives', () => {
    it('selects og_image_1200x630 when available', () => {
      const result = selectOgDerivatives(mockAsset);

      expect(result.default_og_image_url).toBe('https://example.com/uploads/og-1200x630.png');
    });

    it('falls back to asset.url when no og derivative', () => {
      const assetNoOg: Asset = {
        ...mockAsset,
        derivatives: {},
      };

      const result = selectOgDerivatives(assetNoOg);

      expect(result.default_og_image_url).toBe('https://example.com/uploads/logo.png');
    });

    it('returns empty string when asset.url is undefined', () => {
      const assetNoUrl: Asset = {
        ...mockAsset,
        url: undefined as unknown as string,
        derivatives: {},
      };

      const result = selectOgDerivatives(assetNoUrl);

      expect(result.default_og_image_url).toBe('');
    });
  });

  describe('loadBranding', () => {
    it('calls getBranding and returns the response', async () => {
      getBrandingMock.mockResolvedValue(mockBranding);

      const result = await loadBranding();

      expect(getBrandingMock).toHaveBeenCalledTimes(1);
      expect(result).toEqual(mockBranding);
    });

    it('propagates errors from the API', async () => {
      getBrandingMock.mockRejectedValue(new Error('API failure'));

      await expect(loadBranding()).rejects.toThrow('API failure');
    });
  });

  describe('saveBranding', () => {
    it('calls updateBranding with changed fields', async () => {
      updateBrandingMock.mockResolvedValue(mockBranding);

      const original: BrandingFormState = { ...DEFAULT_BRANDING_FORM };
      const form: BrandingFormState = { ...DEFAULT_BRANDING_FORM, site_name: 'Updated Site' };

      await saveBranding(form, original);

      expect(updateBrandingMock).toHaveBeenCalledWith({ site_name: 'Updated Site' });
    });

    it('throws error when no changes to save', async () => {
      const form: BrandingFormState = { ...DEFAULT_BRANDING_FORM };
      const original: BrandingFormState = { ...DEFAULT_BRANDING_FORM };

      await expect(saveBranding(form, original)).rejects.toThrow('No changes to save');
      expect(updateBrandingMock).not.toHaveBeenCalled();
    });

    it('returns updated branding data', async () => {
      const updatedBranding = { ...mockBranding, site_name: 'Updated Site' };
      updateBrandingMock.mockResolvedValue(updatedBranding);

      const original: BrandingFormState = { ...DEFAULT_BRANDING_FORM };
      const form: BrandingFormState = { ...DEFAULT_BRANDING_FORM, site_name: 'Updated Site' };

      const result = await saveBranding(form, original);

      expect(result.site_name).toBe('Updated Site');
    });

    it('propagates API errors', async () => {
      updateBrandingMock.mockRejectedValue(new Error('Save failed'));

      const original: BrandingFormState = { ...DEFAULT_BRANDING_FORM };
      const form: BrandingFormState = { ...DEFAULT_BRANDING_FORM, site_name: 'Test' };

      await expect(saveBranding(form, original)).rejects.toThrow('Save failed');
    });
  });

  describe('clearField', () => {
    it('calls clearBrandingField with field name', async () => {
      clearBrandingFieldMock.mockResolvedValue(mockBranding);

      await clearField('tagline');

      expect(clearBrandingFieldMock).toHaveBeenCalledWith('tagline');
    });

    it('returns updated branding data', async () => {
      const clearedBranding = { ...mockBranding, tagline: null };
      clearBrandingFieldMock.mockResolvedValue(clearedBranding);

      const result = await clearField('tagline');

      expect(result.tagline).toBeNull();
    });

    it('propagates API errors', async () => {
      clearBrandingFieldMock.mockRejectedValue(new Error('Clear failed'));

      await expect(clearField('tagline')).rejects.toThrow('Clear failed');
    });
  });

  describe('formatFieldName', () => {
    it('converts snake_case to space-separated', () => {
      expect(formatFieldName('site_name')).toBe('site name');
      expect(formatFieldName('theme_primary_color')).toBe('theme primary color');
    });

    it('handles single word', () => {
      expect(formatFieldName('tagline')).toBe('tagline');
    });

    it('handles multiple underscores', () => {
      expect(formatFieldName('default_og_image_url')).toBe('default og image url');
    });
  });
});
