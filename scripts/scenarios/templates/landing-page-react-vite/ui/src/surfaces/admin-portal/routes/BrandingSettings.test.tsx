import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { BrandingSettings } from './BrandingSettings';
import { AdminAuthProvider } from '../../../app/providers/AdminAuthProvider';
import * as brandingApi from '../../../shared/api/branding';
import * as commonApi from '../../../shared/api/common';

vi.mock('../../../shared/api/branding', () => ({
  getBranding: vi.fn(),
  updateBranding: vi.fn(),
  clearBrandingField: vi.fn(),
}));

vi.mock('../../../shared/api/common', () => ({
  apiCall: vi.fn(),
}));

vi.mock('../../../shared/api', async () => {
  const actual = await vi.importActual<typeof import('../../../shared/api')>('../../../shared/api');
  return {
    ...actual,
    checkAdminSession: vi.fn().mockResolvedValue({ authenticated: true, email: 'test@example.com' }),
    listVariants: vi.fn().mockResolvedValue({ variants: [] }),
  };
});

const mockedGetBranding = vi.mocked(brandingApi.getBranding);
const mockedUpdateBranding = vi.mocked(brandingApi.updateBranding);

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
      <AdminAuthProvider>
        {component}
      </AdminAuthProvider>
    </BrowserRouter>
  );
};

describe('BrandingSettings', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockedGetBranding.mockResolvedValue(mockBranding);
    mockedUpdateBranding.mockResolvedValue(mockBranding);
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('renders the branding settings page with all sections', async () => {
    renderWithProviders(<BrandingSettings />);

    await waitFor(() => {
      expect(screen.getByText('Branding & SEO')).toBeInTheDocument();
    });

    // Check for main sections
    expect(screen.getByText('Site Identity')).toBeInTheDocument();
    expect(screen.getByText('Default SEO')).toBeInTheDocument();
    expect(screen.getByText('Theme Colors')).toBeInTheDocument();
    expect(screen.getByText('Advanced')).toBeInTheDocument();
  });

  it('loads and displays branding data', async () => {
    renderWithProviders(<BrandingSettings />);

    await waitFor(() => {
      expect(mockedGetBranding).toHaveBeenCalled();
    });

    // Check that form fields are populated
    await waitFor(() => {
      const siteNameInput = screen.getByLabelText(/site name/i);
      expect(siteNameInput).toHaveValue('Test Site');
    });
  });

  it('allows editing site name', async () => {
    const user = userEvent.setup();
    renderWithProviders(<BrandingSettings />);

    await waitFor(() => {
      expect(screen.getByLabelText(/site name/i)).toBeInTheDocument();
    });

    const siteNameInput = screen.getByLabelText(/site name/i);
    await user.clear(siteNameInput);
    await user.type(siteNameInput, 'Updated Site Name');

    expect(siteNameInput).toHaveValue('Updated Site Name');
  });

  it('saves branding changes when save button is clicked', async () => {
    const user = userEvent.setup();
    mockedUpdateBranding.mockResolvedValue({
      ...mockBranding,
      site_name: 'Updated Site Name',
    });

    renderWithProviders(<BrandingSettings />);

    await waitFor(() => {
      expect(screen.getByLabelText(/site name/i)).toBeInTheDocument();
    });

    const siteNameInput = screen.getByLabelText(/site name/i);
    await user.clear(siteNameInput);
    await user.type(siteNameInput, 'Updated Site Name');

    const saveButton = screen.getByRole('button', { name: /save/i });
    await user.click(saveButton);

    await waitFor(() => {
      expect(mockedUpdateBranding).toHaveBeenCalledWith(
        expect.objectContaining({
          site_name: 'Updated Site Name',
        })
      );
    });
  });

  it('displays error message when loading fails', async () => {
    mockedGetBranding.mockRejectedValue(new Error('Failed to load'));

    renderWithProviders(<BrandingSettings />);

    await waitFor(() => {
      expect(screen.getByText(/failed to load/i)).toBeInTheDocument();
    });
  });

  it('displays success message after saving', async () => {
    const user = userEvent.setup();
    renderWithProviders(<BrandingSettings />);

    await waitFor(() => {
      expect(screen.getByLabelText(/site name/i)).toBeInTheDocument();
    });

    const saveButton = screen.getByRole('button', { name: /save/i });
    await user.click(saveButton);

    await waitFor(() => {
      expect(screen.getByText(/saved/i)).toBeInTheDocument();
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
      expect(screen.getByText('Advanced')).toBeInTheDocument();
    });

    // Should have robots.txt textarea
    expect(screen.getByLabelText(/robots\.txt/i)).toBeInTheDocument();
  });
});
