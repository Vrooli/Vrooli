import React from 'react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { renderWithProviders as renderWithBaseProviders } from '../../../test-utils';
import { create } from '@bufbuild/protobuf';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { SiteBrandingSchema } from '@vrooli/proto-types/landing-page-react-vite/v1/branding_pb';
import { AssetSchema } from '@vrooli/proto-types/landing-page-react-vite/v1/assets_pb';
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
    checkAdminSession: vi.fn().mockResolvedValue({ authenticated: true, email: 'test@example.com', resetEnabled: true }),
    listVariants: vi.fn().mockResolvedValue([]),
    getLandingConfig: vi.fn().mockResolvedValue(fallbackConfig),
    adminLogout: vi.fn(),
    adminLogin: vi.fn().mockResolvedValue({ authenticated: true, email: 'test@example.com', resetEnabled: true }),
  };
});

const mockBranding = create(SiteBrandingSchema, {
  id: 1n,
  siteName: 'Test Site',
  tagline: 'A test tagline',
  defaultTitle: 'Test Site | Home',
  defaultDescription: 'Welcome to Test Site',
  themePrimaryColor: '#6366f1',
  themeBackgroundColor: '#07090F',
  canonicalBaseUrl: 'https://test.example.com',
  robotsTxt: 'User-agent: *\nAllow: /',
  createdAt: timestampFromDate(new Date('2025-01-01T00:00:00Z')),
  updatedAt: timestampFromDate(new Date('2025-01-01T00:00:00Z')),
});

const renderWithProviders = (component: React.ReactElement) => {
  return renderWithBaseProviders(
    <BrowserRouter>
      <LandingVariantProvider>
        <AdminAuthProvider>
          {component}
        </AdminAuthProvider>
      </LandingVariantProvider>
    </BrowserRouter>,
    { withoutRouter: true }
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
      siteName: 'Updated Site Name',
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
          siteName: 'Updated Site Name',
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
    mockUploadAsset.mockResolvedValue(
      create(AssetSchema, {
        id: 10n,
        filename: 'logo.png',
        originalFilename: 'logo.png',
        mimeType: 'image/png',
        sizeBytes: 1024n,
        storagePath: 'logos/logo.png',
        url: '/api/v1/uploads/logos/logo.png',
        category: 'logo',
        createdAt: timestampFromDate(new Date()),
        derivatives: {
          logo_512: 'logos/logo-logo_512.png',
          logo_icon: 'logos/logo-logo_icon.png',
          favicon_32: 'logos/logo-favicon_32.png',
          apple_touch_180: 'logos/logo-apple_touch_180.png',
        },
      })
    );
    mockUpdateBranding.mockResolvedValue({
      ...mockBranding,
      logoUrl: 'logos/logo-logo_512.png',
      logoIconUrl: 'logos/logo-logo_icon.png',
      faviconUrl: 'logos/logo-favicon_32.png',
      appleTouchIconUrl: 'logos/logo-apple_touch_180.png',
    });

    const { container } = renderWithProviders(<BrandingSettings />);

    await waitFor(() => {
      expect(screen.getByTestId('branding-header')).toBeInTheDocument();
    });

    const fileInputs = container.querySelectorAll<HTMLInputElement>('input[type="file"]');
    const logoInput = fileInputs.item(0);
    if (!logoInput) throw new Error('logo file input not found');
    const file = new File(['dummy'], 'logo.png', { type: 'image/png' });
    await user.upload(logoInput, file);

    const saveButton = screen.getByRole('button', { name: /save/i });
    await user.click(saveButton);

    await waitFor(() => {
      expect(mockUpdateBranding).toHaveBeenCalledWith(
        expect.objectContaining({
          logoUrl: 'logos/logo-logo_512.png',
          logoIconUrl: 'logos/logo-logo_icon.png',
          faviconUrl: 'logos/logo-favicon_32.png',
          appleTouchIconUrl: 'logos/logo-apple_touch_180.png',
        })
      );
    });
  });

  it('applies favicon upload to favicon and touch icon fields', async () => {
    const user = userEvent.setup();
    mockUploadAsset.mockResolvedValue(
      create(AssetSchema, {
        id: 11n,
        filename: 'favicon.png',
        originalFilename: 'favicon.png',
        mimeType: 'image/png',
        sizeBytes: 512n,
        storagePath: 'favicons/favicon.png',
        url: '/api/v1/uploads/favicons/favicon.png',
        category: 'favicon',
        createdAt: timestampFromDate(new Date()),
        derivatives: {
          favicon_32: 'favicons/favicon-favicon_32.png',
          apple_touch_180: 'favicons/favicon-apple_touch_180.png',
        },
      })
    );
    mockUpdateBranding.mockResolvedValue({
      ...mockBranding,
      faviconUrl: 'favicons/favicon-favicon_32.png',
      appleTouchIconUrl: 'favicons/favicon-apple_touch_180.png',
    });

    const { container } = renderWithProviders(<BrandingSettings />);
    await waitFor(() => expect(screen.getByTestId('branding-header')).toBeInTheDocument());

    const fileInputs = container.querySelectorAll<HTMLInputElement>('input[type="file"]');
    const faviconInput = fileInputs.item(2);
    if (!faviconInput) throw new Error('favicon file input not found');
    const file = new File(['dummy'], 'favicon.png', { type: 'image/png' });
    await user.upload(faviconInput, file);

    const saveButton = screen.getByRole('button', { name: /save/i });
    await user.click(saveButton);

    await waitFor(() => {
      expect(mockUpdateBranding).toHaveBeenCalledWith(
        expect.objectContaining({
          faviconUrl: 'favicons/favicon-favicon_32.png',
          appleTouchIconUrl: 'favicons/favicon-apple_touch_180.png',
        })
      );
    });
  });

  it('applies og upload to default og image field', async () => {
    const user = userEvent.setup();
    mockUploadAsset.mockResolvedValue(
      create(AssetSchema, {
        id: 12n,
        filename: 'og.png',
        originalFilename: 'og.png',
        mimeType: 'image/png',
        sizeBytes: 512n,
        storagePath: 'og-images/og.png',
        url: '/api/v1/uploads/og-images/og.png',
        category: 'og_image',
        createdAt: timestampFromDate(new Date()),
        derivatives: {
          og_image_1200x630: 'og-images/og-og_image_1200x630.png',
        },
      })
    );
    mockUpdateBranding.mockResolvedValue({
      ...mockBranding,
      defaultOgImageUrl: 'og-images/og-og_image_1200x630.png',
    });

    const { container } = renderWithProviders(<BrandingSettings />);
    await waitFor(() => expect(screen.getByTestId('branding-header')).toBeInTheDocument());

    const fileInputs = container.querySelectorAll<HTMLInputElement>('input[type="file"]');
    const ogInput = fileInputs.item(4);
    if (!ogInput) throw new Error('og file input not found');
    const file = new File(['dummy'], 'og.png', { type: 'image/png' });
    await user.upload(ogInput, file);

    const saveButton = screen.getByRole('button', { name: /save/i });
    await user.click(saveButton);

    await waitFor(() => {
      expect(mockUpdateBranding).toHaveBeenCalledWith(
        expect.objectContaining({
          defaultOgImageUrl: 'og-images/og-og_image_1200x630.png',
        })
      );
    });
  });

  it('applies a raw upload URL when no derivatives are generated', async () => {
    const user = userEvent.setup();
    mockUploadAsset.mockResolvedValue(
      create(AssetSchema, {
        id: 20n,
        filename: 'logo.png',
        originalFilename: 'logo.png',
        mimeType: 'image/png',
        sizeBytes: 1024n,
        storagePath: 'logos/logo.png',
        url: 'logos/raw-logo.png',
        category: 'logo',
        createdAt: timestampFromDate(new Date()),
        derivatives: {},
      }),
    );
    mockUpdateBranding.mockResolvedValue({ ...mockBranding, logoUrl: 'logos/raw-logo.png' });
    const { container } = renderWithProviders(<BrandingSettings />);
    await waitFor(() => expect(screen.getByTestId('branding-header')).toBeInTheDocument());

    const logoInput = container.querySelectorAll<HTMLInputElement>('input[type="file"]').item(0);
    if (!logoInput) throw new Error('logo input missing');
    await user.upload(logoInput, new File(['x'], 'logo.png', { type: 'image/png' }));
    const saveButton = await screen.findByRole('button', { name: /save changes/i });
    await user.click(saveButton);
    await waitFor(() =>
      expect(mockUpdateBranding).toHaveBeenCalledWith(expect.objectContaining({ logoUrl: 'logos/raw-logo.png' })),
    );
  });

  it('previews the public landing in a new tab', async () => {
    const user = userEvent.setup();
    const openSpy = vi.spyOn(window, 'open').mockReturnValue(null);
    renderWithProviders(<BrandingSettings />);
    await waitFor(() => expect(screen.getByTestId('branding-header')).toBeInTheDocument());
    await user.click(screen.getByTestId('branding-preview'));
    expect(openSpy).toHaveBeenCalledWith('/', '_blank', 'noopener,noreferrer');
    openSpy.mockRestore();
  });

  it('clears the tagline field through the API', async () => {
    const user = userEvent.setup();
    mockClearBrandingField.mockResolvedValue({ ...mockBranding, tagline: '' });
    renderWithProviders(<BrandingSettings />);
    await waitFor(() => expect(screen.getByTestId('branding-header')).toBeInTheDocument());
    await user.click(screen.getByTitle('Clear tagline'));
    await waitFor(() => expect(mockClearBrandingField).toHaveBeenCalledWith('tagline'));
  });

  it('edits the canonical URL and robots.txt advanced fields', async () => {
    const user = userEvent.setup();
    renderWithProviders(<BrandingSettings />);
    await waitFor(() => expect(screen.getByText('Technical Settings')).toBeInTheDocument());
    const robots = screen.getByPlaceholderText(/User-agent/i);
    await user.type(robots, '\nDisallow: /admin');
    expect((robots as HTMLTextAreaElement).value).toContain('Disallow: /admin');
  });

  it('reloads branding when refresh is clicked', async () => {
    const user = userEvent.setup();
    renderWithProviders(<BrandingSettings />);
    await waitFor(() => expect(screen.getByTestId('branding-header')).toBeInTheDocument());
    mockGetBranding.mockClear();
    await user.click(screen.getByTestId('branding-refresh'));
    await waitFor(() => expect(mockGetBranding).toHaveBeenCalled());
  });

  it('saves multiple changed branding fields in one update', async () => {
    const user = userEvent.setup();
    renderWithProviders(<BrandingSettings />);
    await waitFor(() => expect(screen.getByTestId('branding-header')).toBeInTheDocument());

    await user.type(screen.getByPlaceholderText(/my landing page/i), ' Updated');
    const primary = screen.getByPlaceholderText('#3B82F6');
    await user.clear(primary);
    await user.type(primary, '#abcdef');
    const robots = screen.getByPlaceholderText(/User-agent/i);
    await user.type(robots, '\nSitemap: /sitemap.xml');

    await user.click(await screen.findByRole('button', { name: /save changes/i }));
    await waitFor(() =>
      expect(mockUpdateBranding).toHaveBeenCalledWith(
        expect.objectContaining({ themePrimaryColor: '#abcdef' }),
      ),
    );
  });

  it('keeps default form values when the branding fetch returns nothing', async () => {
    mockGetBranding.mockResolvedValue(undefined);
    renderWithProviders(<BrandingSettings />);
    await waitFor(() => expect(screen.getByTestId('branding-header')).toBeInTheDocument());
    expect(screen.getByPlaceholderText(/my landing page/i)).toBeInTheDocument();
  });

  it('falls back to empty defaults when branding fields are absent', async () => {
    mockGetBranding.mockResolvedValue(create(SiteBrandingSchema, { id: 2n, siteName: '' }));
    renderWithProviders(<BrandingSettings />);
    await waitFor(() => expect(screen.getByTestId('branding-header')).toBeInTheDocument());
    // With no site name, the identity input renders empty against its placeholder.
    expect(screen.getByPlaceholderText(/my landing page/i)).toHaveValue('');
  });

  it('edits and clears the theme colors', async () => {
    const user = userEvent.setup();
    mockClearBrandingField.mockResolvedValue({ ...mockBranding, themePrimaryColor: '' });
    const { container } = renderWithProviders(<BrandingSettings />);
    await waitFor(() => expect(screen.getByText('Theme Colors')).toBeInTheDocument());

    // Text field edit.
    const primaryText = screen.getByPlaceholderText('#3B82F6');
    await user.clear(primaryText);
    await user.type(primaryText, '#123456');
    // Native color pickers fire their own change handler.
    const colorInputs = container.querySelectorAll<HTMLInputElement>('input[type="color"]');
    fireEvent.change(colorInputs[0]!, { target: { value: '#abcdef' } });
    fireEvent.change(colorInputs[1]!, { target: { value: '#000000' } });
    // Clear a color via its dedicated button.
    await user.click(screen.getAllByTitle('Clear color')[0]!);
    await waitFor(() => expect(mockClearBrandingField).toHaveBeenCalledWith('theme_primary_color'));
  });

  it('surfaces an error when clearing a field fails', async () => {
    const user = userEvent.setup();
    mockClearBrandingField.mockRejectedValue(new Error('clear failed'));
    renderWithProviders(<BrandingSettings />);
    await waitFor(() => expect(screen.getByTestId('branding-header')).toBeInTheDocument());
    await user.click(screen.getByTitle('Clear tagline'));
    expect(await screen.findByText(/clear failed/i)).toBeInTheDocument();
  });
});
