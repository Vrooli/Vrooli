import { describe, it, expect, vi, beforeEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '../../../test-utils';
import { VariantSEOEditor } from './VariantSEOEditor';

const { mockGetVariantSEO, mockUpdateVariantSEO } = vi.hoisted(() => ({
  mockGetVariantSEO: vi.fn(),
  mockUpdateVariantSEO: vi.fn(),
}));

vi.mock('../../../shared/api', () => ({
  getVariantSEO: mockGetVariantSEO,
  updateVariantSEO: mockUpdateVariantSEO,
  // Consumed by ImageUploader / SEOPreview when they render.
  uploadAsset: vi.fn(),
  getAssetUrl: (path: string) => path,
}));

const siteBranding = {
  siteName: 'Acme',
  defaultTitle: 'Acme | Home',
  defaultDescription: 'Site default description',
  defaultOgImageUrl: 'og/default.png',
  canonicalBaseUrl: 'https://acme.test',
  faviconUrl: 'favicon.png',
} as never;

const seoResponse = {
  title: 'Variant Title',
  description: 'Variant description',
  ogTitle: 'OG title',
  ogDescription: 'OG description',
  ogImageUrl: 'og/variant.png',
  twitterCard: 'summary_large_image',
  noindex: false,
};

beforeEach(() => {
  vi.clearAllMocks();
  mockGetVariantSEO.mockResolvedValue(seoResponse);
  mockUpdateVariantSEO.mockResolvedValue(undefined);
});

describe('VariantSEOEditor', () => {
  it('shows a loading spinner then the SEO form once data resolves', async () => {
    renderWithProviders(
      <VariantSEOEditor variantSlug="hero" variantName="Hero" siteBranding={siteBranding} />,
    );
    await waitFor(() => expect(mockGetVariantSEO).toHaveBeenCalledWith('hero'));
    expect(await screen.findByText('SEO Settings')).toBeInTheDocument();
    expect(screen.getByText(/Customize meta tags for the "Hero" variant/)).toBeInTheDocument();
    expect(screen.getByDisplayValue('Variant Title')).toBeInTheDocument();
  });

  it('surfaces a load error when fetching SEO fails', async () => {
    mockGetVariantSEO.mockRejectedValue(new Error('boom'));
    renderWithProviders(<VariantSEOEditor variantSlug="hero" variantName="Hero" />);
    expect(await screen.findByText('Failed to load SEO settings')).toBeInTheDocument();
  });

  it('drops variant fields that equal the site defaults so placeholders show', async () => {
    mockGetVariantSEO.mockResolvedValue({
      ...seoResponse,
      title: 'Acme | Home',
      description: 'Site default description',
      ogImageUrl: 'og/default.png',
    });
    renderWithProviders(
      <VariantSEOEditor variantSlug="hero" variantName="Hero" siteBranding={siteBranding} />,
    );
    const titleInput = await screen.findByPlaceholderText('Acme | Home');
    expect(titleInput).toHaveValue('');
  });

  it('updates the title and reflects the live character count', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <VariantSEOEditor variantSlug="hero" variantName="Hero" siteBranding={siteBranding} />,
    );
    const titleInput = await screen.findByDisplayValue('Variant Title');
    await user.clear(titleInput);
    await user.type(titleInput, 'New');
    expect(screen.getByText(/Current: 3/)).toBeInTheDocument();
  });

  it('saves the config, calls onSave, and shows a saved confirmation', async () => {
    const user = userEvent.setup();
    const onSave = vi.fn();
    renderWithProviders(
      <VariantSEOEditor
        variantSlug="hero"
        variantName="Hero"
        siteBranding={siteBranding}
        onSave={onSave}
      />,
    );
    await screen.findByText('SEO Settings');
    await user.click(screen.getByRole('button', { name: /Save SEO/i }));

    await waitFor(() => expect(mockUpdateVariantSEO).toHaveBeenCalledWith('hero', expect.any(Object)));
    expect(await screen.findByText('Saved')).toBeInTheDocument();
    expect(onSave).toHaveBeenCalled();
  });

  it('surfaces a save error from the API', async () => {
    const user = userEvent.setup();
    mockUpdateVariantSEO.mockRejectedValue(new Error('save failed'));
    renderWithProviders(
      <VariantSEOEditor variantSlug="hero" variantName="Hero" siteBranding={siteBranding} />,
    );
    await screen.findByText('SEO Settings');
    await user.click(screen.getByRole('button', { name: /Save SEO/i }));
    expect(await screen.findByText('save failed')).toBeInTheDocument();
  });

  it('edits the social and canonical override fields', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <VariantSEOEditor variantSlug="hero" variantName="Hero" siteBranding={siteBranding} />,
    );
    await screen.findByText('SEO Settings');

    await user.type(screen.getByDisplayValue('Variant description'), ' extended');
    await user.type(screen.getByDisplayValue('OG title'), '!');
    await user.type(screen.getByDisplayValue('OG description'), '!');
    await user.type(screen.getByPlaceholderText('/'), 'landing');
    expect(screen.getByDisplayValue('landing')).toBeInTheDocument();
  });

  it('toggles noindex and switches the twitter card type', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <VariantSEOEditor variantSlug="hero" variantName="Hero" siteBranding={siteBranding} />,
    );
    await screen.findByText('SEO Settings');

    const noindex = screen.getByLabelText(/Hide from search engines/i);
    await user.click(noindex);
    expect(noindex).toBeChecked();

    const twitterCard = screen.getByDisplayValue('Large Image Card');
    await user.selectOptions(twitterCard, 'summary');
    expect((twitterCard as HTMLSelectElement).value).toBe('summary');
  });
});
