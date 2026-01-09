import { describe, it, expect } from 'vitest';
import type { SiteBranding, VariantSEOResponse } from '../../../shared/api';
import { buildEditableSEOConfig } from './seoController';

describe('seoController', () => {
  const branding: SiteBranding = {
    site_name: 'Landing Suite',
    default_title: 'Default Title',
    default_description: 'Default description',
    default_og_image_url: 'https://cdn.example.com/default.png',
    theme_primary_color: '#000',
  };

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
    const response: VariantSEOResponse = {
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

    const config = buildEditableSEOConfig(response, branding);

    expect(config.title).toBe('Custom Title');
    expect(config.description).toBe('Custom description');
    expect(config.og_title).toBe('OG Title');
    expect(config.og_description).toBe('OG Description');
    expect(config.og_image_url).toBe('https://cdn.example.com/custom.png');
    expect(config.twitter_card).toBe('summary');
    expect(config.noindex).toBe(true);
    expect(config.structured_data).toEqual({ '@type': 'Product', name: 'Suite' });
  });
});
