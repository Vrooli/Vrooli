import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import type { SiteBranding, Asset } from '../../../shared/api';
import { useBrandingForm } from './useBrandingForm';
import type { loadBranding, saveBranding, clearField } from '../services/branding.service';

// Mock the branding service
type LoadBrandingFn = typeof loadBranding;
type SaveBrandingFn = typeof saveBranding;
type ClearFieldFn = typeof clearField;

const loadBrandingMock = vi.fn<Parameters<LoadBrandingFn>, ReturnType<LoadBrandingFn>>();
const saveBrandingMock = vi.fn<Parameters<SaveBrandingFn>, ReturnType<SaveBrandingFn>>();
const clearFieldMock = vi.fn<Parameters<ClearFieldFn>, ReturnType<ClearFieldFn>>();

vi.mock('../services/branding.service', async () => {
  const actual = await vi.importActual<typeof import('../services/branding.service')>(
    '../services/branding.service'
  );
  return {
    ...actual,
    loadBranding: (...args: Parameters<LoadBrandingFn>) => loadBrandingMock(...args),
    saveBranding: (...args: Parameters<SaveBrandingFn>) => saveBrandingMock(...args),
    clearField: (...args: Parameters<ClearFieldFn>) => clearFieldMock(...args),
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
    logo_icon: 'https://example.com/uploads/logo-icon.png',
    favicon: 'https://example.com/uploads/favicon.ico',
    favicon_32: 'https://example.com/uploads/favicon-32.png',
    apple_touch_180: 'https://example.com/uploads/apple-touch-180.png',
    og_image_1200x630: 'https://example.com/uploads/og-1200x630.png',
  },
};

describe('useBrandingForm', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    loadBrandingMock.mockResolvedValue(mockBranding);
  });

  describe('initial load', () => {
    it('loads branding data on mount', async () => {
      const { result } = renderHook(() => useBrandingForm());

      expect(result.current.loading).toBe(true);

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(loadBrandingMock).toHaveBeenCalledTimes(1);
      expect(result.current.branding).toEqual(mockBranding);
      expect(result.current.form.site_name).toBe('Test Site');
    });

    it('populates form from branding data', async () => {
      const { result } = renderHook(() => useBrandingForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.form.site_name).toBe('Test Site');
      expect(result.current.form.tagline).toBe('A test tagline');
      expect(result.current.form.logo_url).toBe('https://example.com/logo.png');
      expect(result.current.form.smtp_port).toBe('587');
      expect(result.current.form.coming_soon_enabled).toBe(false);
    });

    it('handles load error', async () => {
      loadBrandingMock.mockRejectedValue(new Error('API failure'));

      const { result } = renderHook(() => useBrandingForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.error).toBe('API failure');
      expect(result.current.branding).toBeNull();
    });

    it('handles non-Error rejection', async () => {
      loadBrandingMock.mockRejectedValue('Unknown error');

      const { result } = renderHook(() => useBrandingForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.error).toBe('Failed to load branding');
    });
  });

  describe('form input handling', () => {
    it('updates form field on handleInput', async () => {
      const { result } = renderHook(() => useBrandingForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        const handler = result.current.handleInput('site_name');
        handler({ target: { value: 'New Site Name' } } as React.ChangeEvent<HTMLInputElement>);
      });

      expect(result.current.form.site_name).toBe('New Site Name');
    });

    it('clears success message on input', async () => {
      const { result } = renderHook(() => useBrandingForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      // Simulate a successful save first
      saveBrandingMock.mockResolvedValue(mockBranding);
      await act(async () => {
        result.current.handleFieldChange('site_name', 'Changed');
        await result.current.handleSubmit({ preventDefault: vi.fn() } as unknown as React.FormEvent);
      });

      expect(result.current.successMessage).toBe('Branding updated successfully');

      // Now make an input change
      act(() => {
        result.current.handleInput('tagline')({
          target: { value: 'New tagline' },
        } as React.ChangeEvent<HTMLInputElement>);
      });

      expect(result.current.successMessage).toBeNull();
    });

    it('updates form field on handleFieldChange', async () => {
      const { result } = renderHook(() => useBrandingForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.handleFieldChange('theme_primary_color', '#FF0000');
      });

      expect(result.current.form.theme_primary_color).toBe('#FF0000');
    });
  });

  describe('image handling', () => {
    it('updates form on handleImageChange', async () => {
      const { result } = renderHook(() => useBrandingForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        const handler = result.current.handleImageChange('logo_url');
        handler('https://example.com/new-logo.png');
      });

      expect(result.current.form.logo_url).toBe('https://example.com/new-logo.png');
    });

    it('handles null image URL', async () => {
      const { result } = renderHook(() => useBrandingForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.handleImageChange('logo_url')(null);
      });

      expect(result.current.form.logo_url).toBe('');
    });

    it('applies logo derivatives on upload', async () => {
      const { result } = renderHook(() => useBrandingForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.applyLogoDerivatives(mockAsset);
      });

      expect(result.current.form.logo_url).toBe('https://example.com/uploads/logo-512.png');
      expect(result.current.form.logo_icon_url).toBe('https://example.com/uploads/logo-icon.png');
      expect(result.current.form.favicon_url).toBe('https://example.com/uploads/favicon-32.png');
      expect(result.current.form.apple_touch_icon_url).toBe(
        'https://example.com/uploads/apple-touch-180.png'
      );
    });

    it('applies favicon derivatives on upload', async () => {
      const { result } = renderHook(() => useBrandingForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.applyFaviconDerivatives(mockAsset);
      });

      expect(result.current.form.favicon_url).toBe('https://example.com/uploads/favicon.ico');
      expect(result.current.form.apple_touch_icon_url).toBe(
        'https://example.com/uploads/apple-touch-180.png'
      );
    });

    it('applies OG derivatives on upload', async () => {
      const { result } = renderHook(() => useBrandingForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.applyOgDerivatives(mockAsset);
      });

      expect(result.current.form.default_og_image_url).toBe(
        'https://example.com/uploads/og-1200x630.png'
      );
    });
  });

  describe('dirty detection', () => {
    it('returns isDirty false when form matches original', async () => {
      const { result } = renderHook(() => useBrandingForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.isDirty).toBe(false);
    });

    it('returns isDirty true when form differs from original', async () => {
      const { result } = renderHook(() => useBrandingForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.handleFieldChange('site_name', 'Changed Site Name');
      });

      expect(result.current.isDirty).toBe(true);
    });
  });

  describe('branding health', () => {
    it('computes branding health from form', async () => {
      const { result } = renderHook(() => useBrandingForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      // Mock branding has site_name, logo_url, favicon_url, title, description, og_image
      expect(result.current.brandingHealth.checks.identity).toBe(true);
      expect(result.current.brandingHealth.checks.favicon).toBe(true);
      expect(result.current.brandingHealth.checks.seo).toBe(true);
      expect(result.current.brandingHealth.checks.ogImage).toBe(true);
      expect(result.current.brandingHealth.percentage).toBe(100);
    });

    it('updates health when form changes', async () => {
      loadBrandingMock.mockResolvedValue({
        id: 1,
        site_name: 'Test',
      });

      const { result } = renderHook(() => useBrandingForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.brandingHealth.checks.identity).toBe(false); // No logo

      act(() => {
        result.current.handleFieldChange('logo_url', 'https://example.com/logo.png');
      });

      expect(result.current.brandingHealth.checks.identity).toBe(true);
    });
  });

  describe('form submission', () => {
    it('saves branding on submit', async () => {
      saveBrandingMock.mockResolvedValue({ ...mockBranding, site_name: 'Updated Site' });

      const { result } = renderHook(() => useBrandingForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.handleFieldChange('site_name', 'Updated Site');
      });

      await act(async () => {
        await result.current.handleSubmit({ preventDefault: vi.fn() } as unknown as React.FormEvent);
      });

      expect(saveBrandingMock).toHaveBeenCalled();
      expect(result.current.successMessage).toBe('Branding updated successfully');
    });

    it('sets saving state during submission', async () => {
      let resolveSave: (value: SiteBranding) => void;
      saveBrandingMock.mockReturnValue(
        new Promise<SiteBranding>((resolve) => {
          resolveSave = resolve;
        })
      );

      const { result } = renderHook(() => useBrandingForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.handleFieldChange('site_name', 'Changed');
      });

      act(() => {
        result.current.handleSubmit({ preventDefault: vi.fn() } as unknown as React.FormEvent);
      });

      expect(result.current.saving).toBe(true);

      await act(async () => {
        resolveSave?.({ ...mockBranding, site_name: 'Changed' });
      });

      expect(result.current.saving).toBe(false);
    });

    it('handles save error', async () => {
      saveBrandingMock.mockRejectedValue(new Error('Save failed'));

      const { result } = renderHook(() => useBrandingForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.handleFieldChange('site_name', 'Changed');
      });

      await act(async () => {
        await result.current.handleSubmit({ preventDefault: vi.fn() } as unknown as React.FormEvent);
      });

      expect(result.current.error).toBe('Save failed');
      expect(result.current.saving).toBe(false);
    });

    it('updates original form after successful save', async () => {
      const updatedBranding = { ...mockBranding, site_name: 'New Name' };
      saveBrandingMock.mockResolvedValue(updatedBranding);

      const { result } = renderHook(() => useBrandingForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.handleFieldChange('site_name', 'New Name');
      });

      expect(result.current.isDirty).toBe(true);

      await act(async () => {
        await result.current.handleSubmit({ preventDefault: vi.fn() } as unknown as React.FormEvent);
      });

      expect(result.current.isDirty).toBe(false);
      expect(result.current.originalForm.site_name).toBe('New Name');
    });
  });

  describe('clear field', () => {
    it('clears field via API', async () => {
      const clearedBranding = { ...mockBranding, tagline: null };
      clearFieldMock.mockResolvedValue(clearedBranding);

      const { result } = renderHook(() => useBrandingForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      await act(async () => {
        await result.current.handleClearField('tagline');
      });

      expect(clearFieldMock).toHaveBeenCalledWith('tagline');
      expect(result.current.form.tagline).toBe('');
      expect(result.current.successMessage).toBe('Cleared tagline');
    });

    it('handles clear field error', async () => {
      clearFieldMock.mockRejectedValue(new Error('Clear failed'));

      const { result } = renderHook(() => useBrandingForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      await act(async () => {
        await result.current.handleClearField('tagline');
      });

      expect(result.current.error).toBe('Clear failed');
    });

    it('updates both form and original after clear', async () => {
      const clearedBranding = { ...mockBranding, tagline: null };
      clearFieldMock.mockResolvedValue(clearedBranding);

      const { result } = renderHook(() => useBrandingForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      await act(async () => {
        await result.current.handleClearField('tagline');
      });

      expect(result.current.form.tagline).toBe('');
      expect(result.current.originalForm.tagline).toBe('');
      expect(result.current.isDirty).toBe(false);
    });
  });

  describe('toggle coming soon', () => {
    it('toggles coming_soon_enabled', async () => {
      const { result } = renderHook(() => useBrandingForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.form.coming_soon_enabled).toBe(false);

      act(() => {
        result.current.toggleComingSoon();
      });

      expect(result.current.form.coming_soon_enabled).toBe(true);

      act(() => {
        result.current.toggleComingSoon();
      });

      expect(result.current.form.coming_soon_enabled).toBe(false);
    });

    it('clears success message on toggle', async () => {
      saveBrandingMock.mockResolvedValue(mockBranding);

      const { result } = renderHook(() => useBrandingForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      // Create success message
      act(() => {
        result.current.handleFieldChange('site_name', 'Changed');
      });
      await act(async () => {
        await result.current.handleSubmit({ preventDefault: vi.fn() } as unknown as React.FormEvent);
      });

      expect(result.current.successMessage).not.toBeNull();

      act(() => {
        result.current.toggleComingSoon();
      });

      expect(result.current.successMessage).toBeNull();
    });
  });

  describe('reload', () => {
    it('reloads branding data', async () => {
      const { result } = renderHook(() => useBrandingForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      loadBrandingMock.mockClear();
      loadBrandingMock.mockResolvedValue({ ...mockBranding, site_name: 'Reloaded' });

      await act(async () => {
        await result.current.loadBrandingData();
      });

      expect(loadBrandingMock).toHaveBeenCalledTimes(1);
      expect(result.current.form.site_name).toBe('Reloaded');
    });

    it('clears error on reload', async () => {
      loadBrandingMock.mockRejectedValue(new Error('Initial error'));

      const { result } = renderHook(() => useBrandingForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.error).toBe('Initial error');

      loadBrandingMock.mockResolvedValue(mockBranding);

      await act(async () => {
        await result.current.loadBrandingData();
      });

      expect(result.current.error).toBeNull();
    });
  });

  describe('preview public landing', () => {
    it('opens landing page in new window', async () => {
      const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null);

      const { result } = renderHook(() => useBrandingForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.previewPublicLanding();
      });

      expect(openSpy).toHaveBeenCalledWith('/', '_blank', 'noopener,noreferrer');

      openSpy.mockRestore();
    });
  });
});
