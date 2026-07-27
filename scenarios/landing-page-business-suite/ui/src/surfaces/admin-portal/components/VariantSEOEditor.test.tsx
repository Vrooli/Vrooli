import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { VariantSEOEditor } from './VariantSEOEditor';
import * as seoHook from '../hooks/useVariantSEOEditor';

vi.mock('../hooks/useVariantSEOEditor');
vi.mock('../../../shared/ui/ImageUploader', () => ({ ImageUploader: ({ onChange }: { onChange: (value: string) => void }) => <button type="button" onClick={() => { onChange('/og.png'); }}>Set OG image</button> }));
vi.mock('../../../shared/ui/SEOPreview', () => ({ SEOPreview: ({ title, description, url }: { title: string; description: string; url: string }) => <output>{`${title}|${description}|${url}`}</output> }));

function hookState(overrides: Partial<ReturnType<typeof seoHook.useVariantSEOEditor>> = {}) {
  return {
    seoConfig: {}, loading: false, saving: false, error: null, success: false,
    fetchSEO: vi.fn(), handleSave: vi.fn().mockResolvedValue(undefined), updateField: vi.fn(), ...overrides,
  };
}

describe('VariantSEOEditor', () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it('shows a loading state until the variant SEO configuration is ready', () => {
    vi.mocked(seoHook.useVariantSEOEditor).mockReturnValue(hookState({ loading: true }));
    render(<VariantSEOEditor variantSlug="control" variantName="Control" />);
    expect(screen.queryByText('SEO Settings')).not.toBeInTheDocument();
  });

  it('uses safe site branding defaults in its public-search preview', () => {
    vi.mocked(seoHook.useVariantSEOEditor).mockReturnValue(hookState());
    render(<VariantSEOEditor variantSlug="control" variantName="Control" siteBranding={{ id: 1, site_name: 'Acme', default_title: 'Acme default', default_description: 'Trusted workspace', canonical_base_url: 'https://acme.example' }} />);
    expect(screen.getByText('Acme default|Trusted workspace|https://acme.example')).toBeInTheDocument();
  });

  it('forwards SEO field edits, image selection, and save intent through the controlled hook', () => {
    const state = hookState({ seoConfig: { title: 'Campaign' }, error: 'Metadata rejected', success: true });
    vi.mocked(seoHook.useVariantSEOEditor).mockReturnValue(state);
    render(<VariantSEOEditor variantSlug="campaign" variantName="Campaign" siteBranding={{ id: 1, site_name: 'Acme' }} />);
    fireEvent.change(screen.getByDisplayValue('Campaign'), { target: { value: 'Campaign updated' } });
    fireEvent.click(screen.getByLabelText('Hide from search engines (noindex)'));
    fireEvent.click(screen.getByRole('button', { name: 'Set OG image' }));
    fireEvent.click(screen.getByRole('button', { name: 'Save SEO' }));
    expect(state.updateField).toHaveBeenCalledWith('title', 'Campaign updated');
    expect(state.updateField).toHaveBeenCalledWith('noindex', true);
    expect(state.updateField).toHaveBeenCalledWith('og_image_url', '/og.png');
    expect(state.handleSave).toHaveBeenCalledOnce();
    expect(screen.getByText('Metadata rejected')).toBeInTheDocument();
    expect(screen.getByText('Saved')).toBeInTheDocument();
  });

  it('forwards metadata, social overrides, card type, and canonical path edits', () => {
    const state = hookState();
    vi.mocked(seoHook.useVariantSEOEditor).mockReturnValue(state);
    render(<VariantSEOEditor variantSlug="campaign" variantName="Campaign" siteBranding={{ id: 1, site_name: 'Acme' }} />);

    fireEvent.change(screen.getAllByPlaceholderText('Use site default')[1]!, { target: { value: 'Campaign details' } });
    fireEvent.change(screen.getByPlaceholderText('Same as page title'), { target: { value: 'Share campaign' } });
    fireEvent.change(screen.getByPlaceholderText('Same as meta description'), { target: { value: 'Share details' } });
    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'summary' } });
    fireEvent.change(screen.getByPlaceholderText('/'), { target: { value: '/campaign' } });

    expect(state.updateField).toHaveBeenCalledWith('description', 'Campaign details');
    expect(state.updateField).toHaveBeenCalledWith('og_title', 'Share campaign');
    expect(state.updateField).toHaveBeenCalledWith('og_description', 'Share details');
    expect(state.updateField).toHaveBeenCalledWith('twitter_card', 'summary');
    expect(state.updateField).toHaveBeenCalledWith('canonical_path', '/campaign');
  });
});
