import React from 'react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { BrandingSettings } from './BrandingSettings';
import { AdminAuthProvider } from '../../../app/providers/AdminAuthProvider';
import { LandingVariantProvider } from '../../../app/providers/LandingVariantProvider';

vi.mock('../components/AdminLayout', () => ({
  AdminLayout: ({ children }: { children: React.ReactNode }) => <div data-testid="admin-layout">{children}</div>,
}));
vi.mock('../../../app/providers/LandingVariantProvider', () => ({
  useLandingVariant: () => ({
    variant: { slug: 'control', name: 'Control' },
    config: null,
    loading: false,
    error: null,
    resolution: 'api_select',
    statusNote: null,
    lastUpdated: Date.now(),
    refresh: vi.fn(),
  }),
  LandingVariantProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

const { mockGetBranding, mockUpdateBranding, mockClearBrandingField, mockUploadAsset } = vi.hoisted(() => ({
  mockGetBranding: vi.fn(),
  mockUpdateBranding: vi.fn(),
  mockClearBrandingField: vi.fn(),
  mockUploadAsset: vi.fn(),
}));

vi.mock('../../../shared/api', async () => {
  const { getFallbackLandingConfig } = await import('../../../shared/lib/fallbackLandingConfig');
  const fallbackConfig = getFallbackLandingConfig();
  return {
    getBranding: mockGetBranding,
    updateBranding: mockUpdateBranding,
    clearBrandingField: mockClearBrandingField,
    uploadAsset: mockUploadAsset,
    getAssetUrl: (path: string) => path,
    checkAdminSession: vi.fn().mockResolvedValue({ authenticated: true, email: 'test@example.com', reset_enabled: true }),
    listVariants: vi.fn().mockResolvedValue({ variants: [] }),
    getLandingConfig: vi.fn().mockResolvedValue(fallbackConfig),
    adminLogout: vi.fn(),
    adminLogin: vi.fn().mockResolvedValue({ authenticated: true, email: 'test@example.com', reset_enabled: true }),
  };
});

const mockBranding = {
  id: 1,
  site_name: 'Test Site',
  tagline: 'A test tagline',
  logo_url: null,
  logo_icon_url: null,
  favicon_url: null,
  apple_touch_icon_url: null,
  default_title: 'Test Site | Home',
  default_description: 'Welcome to Test Site',
  default_og_image_url: null,
  theme_primary_color: '#6366f1',
  theme_background_color: '#07090F',
  canonical_base_url: 'https://test.example.com',
  google_site_verification: null,
  robots_txt: 'User-agent: *\nAllow: /',
  created_at: '2025-01-01T00:00:00Z',
  updated_at: '2025-01-01T00:00:00Z',
};

const renderWithProviders = (component: React.ReactElement) => {
  return render(
    <BrowserRouter>
      <LandingVariantProvider>
        <AdminAuthProvider>
          {component}
        </AdminAuthProvider>
      </LandingVariantProvider>
    </BrowserRouter>
  );
};

describe('BrandingSettings', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetBranding.mockResolvedValue(mockBranding);
    mockUpdateBranding.mockResolvedValue(mockBranding);
    mockUploadAsset.mockReset();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('renders the branding settings page with all sections', async () => {
    renderWithProviders(<BrandingSettings />);

    await waitFor(() => {
      expect(screen.getByTestId('branding-header')).toBeInTheDocument();
    });

    // Check for main sections
    expect(screen.getByText('Site Identity')).toBeInTheDocument();
    expect(screen.getByText('Default SEO')).toBeInTheDocument();
    expect(screen.getByText('Theme Colors')).toBeInTheDocument();
    expect(screen.getByText('Technical Settings')).toBeInTheDocument();
  });

  it('loads and displays branding data', async () => {
    renderWithProviders(<BrandingSettings />);

    await waitFor(() => {
      expect(mockGetBranding).toHaveBeenCalled();
    });

    // Check that form fields are populated
    await waitFor(() => {
      expect(screen.getByDisplayValue('Test Site')).toBeInTheDocument();
    });
  });

  it('allows editing site name', async () => {
    const user = userEvent.setup();
    renderWithProviders(<BrandingSettings />);

    await waitFor(() => {
      expect(screen.getByPlaceholderText(/my landing page/i)).toBeInTheDocument();
    });

    const siteNameInput = screen.getByPlaceholderText(/my landing page/i);
    await user.clear(siteNameInput);
    await user.type(siteNameInput, 'Updated Site Name');

    expect(siteNameInput).toHaveValue('Updated Site Name');
  });

  it('saves branding changes when save button is clicked', async () => {
    const user = userEvent.setup();
    mockUpdateBranding.mockResolvedValue({
      ...mockBranding,
      site_name: 'Updated Site Name',
    });

    renderWithProviders(<BrandingSettings />);

    await waitFor(() => {
      expect(screen.getByPlaceholderText(/my landing page/i)).toBeInTheDocument();
    });

    const siteNameInput = screen.getByPlaceholderText(/my landing page/i);
    await user.clear(siteNameInput);
    await user.type(siteNameInput, 'Updated Site Name');

    const saveButton = await screen.findByRole('button', { name: /save changes/i });
    await user.click(saveButton);

    await waitFor(() => {
      expect(mockUpdateBranding).toHaveBeenCalledWith(
        expect.objectContaining({
          site_name: 'Updated Site Name',
        })
      );
    });
  });

  it('displays error message when loading fails', async () => {
    mockGetBranding.mockRejectedValue(new Error('Failed to load'));

    renderWithProviders(<BrandingSettings />);

    await waitFor(() => {
      expect(screen.getByText(/failed to load/i)).toBeInTheDocument();
    });
  });

  it('displays success message after saving', async () => {
    const user = userEvent.setup();
    renderWithProviders(<BrandingSettings />);

    await waitFor(() => {
      expect(screen.getByPlaceholderText(/my landing page/i)).toBeInTheDocument();
    });

    const siteNameInput = screen.getByPlaceholderText(/my landing page/i);
    await user.clear(siteNameInput);
    await user.type(siteNameInput, 'Updated Site Name');

    const saveButton = await screen.findByRole('button', { name: /save changes/i });
    await user.click(saveButton);

    await waitFor(() => {
      expect(screen.getByText(/updated successfully/i)).toBeInTheDocument();
    });
  });

  it('renders color picker for theme colors', async () => {
    renderWithProviders(<BrandingSettings />);

    await waitFor(() => {
      expect(screen.getByText('Theme Colors')).toBeInTheDocument();
    });

    // Should have color input fields
    const colorInputs = screen.getAllByRole('textbox').filter(
      input => input.getAttribute('type') === 'text' &&
               (input.getAttribute('placeholder')?.includes('#') ||
                (input as HTMLInputElement).value?.startsWith('#'))
    );
    expect(colorInputs.length).toBeGreaterThan(0);
  });

  it('renders robots.txt editor in advanced section', async () => {
    renderWithProviders(<BrandingSettings />);

    await waitFor(() => {
      expect(screen.getByText('Technical Settings')).toBeInTheDocument();
    });

    // Should have robots.txt textarea
    expect(screen.getByPlaceholderText(/User-agent/i)).toBeInTheDocument();
  });

  it('uses generated derivatives from a single upload to populate related branding fields', async () => {
    const user = userEvent.setup();
    mockUploadAsset.mockResolvedValue({
      id: 10,
      filename: 'logo.png',
      original_filename: 'logo.png',
      mime_type: 'image/png',
      size_bytes: 1024,
      storage_path: 'logos/logo.png',
      url: '/api/v1/uploads/logos/logo.png',
      category: 'logo',
      uploaded_by: null,
      created_at: new Date().toISOString(),
      derivatives: {
        logo_512: 'logos/logo-logo_512.png',
        logo_icon: 'logos/logo-logo_icon.png',
        favicon_32: 'logos/logo-favicon_32.png',
        apple_touch_180: 'logos/logo-apple_touch_180.png',
      },
    });
    mockUpdateBranding.mockResolvedValue({
      ...mockBranding,
      logo_url: 'logos/logo-logo_512.png',
      logo_icon_url: 'logos/logo-logo_icon.png',
      favicon_url: 'logos/logo-favicon_32.png',
      apple_touch_icon_url: 'logos/logo-apple_touch_180.png',
    });

    const { container } = renderWithProviders(<BrandingSettings />);

    await waitFor(() => {
      expect(screen.getByTestId('branding-header')).toBeInTheDocument();
    });

    const fileInputs = container.querySelectorAll('input[type="file"]');
    const logoInput = fileInputs.item(0);
    const file = new File(['dummy'], 'logo.png', { type: 'image/png' });
    await user.upload(logoInput, file);

    const saveButton = screen.getByRole('button', { name: /save/i });
    await user.click(saveButton);

    await waitFor(() => {
      expect(mockUpdateBranding).toHaveBeenCalledWith(
        expect.objectContaining({
          logo_url: 'logos/logo-logo_512.png',
          logo_icon_url: 'logos/logo-logo_icon.png',
          favicon_url: 'logos/logo-favicon_32.png',
          apple_touch_icon_url: 'logos/logo-apple_touch_180.png',
        })
      );
    });
  });

  it('applies favicon upload to favicon and touch icon fields', async () => {
    const user = userEvent.setup();
    mockUploadAsset.mockResolvedValue({
      id: 11,
      filename: 'favicon.png',
      original_filename: 'favicon.png',
      mime_type: 'image/png',
      size_bytes: 512,
      storage_path: 'favicons/favicon.png',
      url: '/api/v1/uploads/favicons/favicon.png',
      category: 'favicon',
      uploaded_by: null,
      created_at: new Date().toISOString(),
      derivatives: {
        favicon_32: 'favicons/favicon-favicon_32.png',
        apple_touch_180: 'favicons/favicon-apple_touch_180.png',
      },
    });
    mockUpdateBranding.mockResolvedValue({
      ...mockBranding,
      favicon_url: 'favicons/favicon-favicon_32.png',
      apple_touch_icon_url: 'favicons/favicon-apple_touch_180.png',
    });

    const { container } = renderWithProviders(<BrandingSettings />);
    await waitFor(() => expect(screen.getByTestId('branding-header')).toBeInTheDocument());

    const fileInputs = container.querySelectorAll('input[type="file"]');
    const faviconInput = fileInputs.item(2);
    const file = new File(['dummy'], 'favicon.png', { type: 'image/png' });
    await user.upload(faviconInput, file);

    const saveButton = screen.getByRole('button', { name: /save/i });
    await user.click(saveButton);

    await waitFor(() => {
      expect(mockUpdateBranding).toHaveBeenCalledWith(
        expect.objectContaining({
          favicon_url: 'favicons/favicon-favicon_32.png',
          apple_touch_icon_url: 'favicons/favicon-apple_touch_180.png',
        })
      );
    });
  });

  it('applies og upload to default og image field', async () => {
    const user = userEvent.setup();
    mockUploadAsset.mockResolvedValue({
      id: 12,
      filename: 'og.png',
      original_filename: 'og.png',
      mime_type: 'image/png',
      size_bytes: 512,
      storage_path: 'og-images/og.png',
      url: '/api/v1/uploads/og-images/og.png',
      category: 'og_image',
      uploaded_by: null,
      created_at: new Date().toISOString(),
      derivatives: {
        og_image_1200x630: 'og-images/og-og_image_1200x630.png',
      },
    });
    mockUpdateBranding.mockResolvedValue({
      ...mockBranding,
      default_og_image_url: 'og-images/og-og_image_1200x630.png',
    });

    const { container } = renderWithProviders(<BrandingSettings />);
    await waitFor(() => expect(screen.getByTestId('branding-header')).toBeInTheDocument());

    const fileInputs = container.querySelectorAll('input[type="file"]');
    const ogInput = fileInputs.item(4);
    const file = new File(['dummy'], 'og.png', { type: 'image/png' });
    await user.upload(ogInput, file);

    const saveButton = screen.getByRole('button', { name: /save/i });
    await user.click(saveButton);

    await waitFor(() => {
      expect(mockUpdateBranding).toHaveBeenCalledWith(
        expect.objectContaining({
          default_og_image_url: 'og-images/og-og_image_1200x630.png',
        })
      );
    });
  });
});
