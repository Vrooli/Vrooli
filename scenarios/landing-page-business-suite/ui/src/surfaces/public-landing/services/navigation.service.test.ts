import { describe, expect, it } from 'vitest';
import {
  buildNavItems,
  formatDownloadPlatform,
  getDownloadButtonLabel,
  hasDownloadTargets,
} from './navigation.service';
import type { DownloadApp, LandingSection } from '../../../shared/api';

const app = (overrides: Partial<DownloadApp> = {}): DownloadApp => ({
  bundle_key: 'business-suite',
  app_key: 'desktop',
  name: 'Desktop App',
  platforms: [],
  storefronts: [],
  ...overrides,
});

describe('public landing navigation service', () => {
  it('recognizes platform and storefront download targets', () => {
    expect(hasDownloadTargets(app())).toBe(false);
    expect(hasDownloadTargets(app({ platforms: [{ platform: 'windows' }] as DownloadApp['platforms'] }))).toBe(true);
    expect(hasDownloadTargets(app({ storefronts: [{ store: 'app_store', label: 'App Store', url: 'https://example.test' }] }))).toBe(true);
  });

  it('treats a malformed missing platforms field as having no download target', () => {
    const malformedApp = { ...app(), platforms: undefined } as unknown as DownloadApp;

    expect(hasDownloadTargets(malformedApp)).toBe(false);
    expect(getDownloadButtonLabel([malformedApp])).toBe('View Desktop App');
  });

  it('chooses specific labels only when a single download target is available', () => {
    expect(getDownloadButtonLabel([])).toBe('Downloads');
    expect(getDownloadButtonLabel([app({ platforms: [{ platform: 'windows' }] as DownloadApp['platforms'] })])).toBe('Download Windows');
    expect(getDownloadButtonLabel([app({ storefronts: [{ store: 'app_store', url: 'https://example.test', label: 'App Store' }] })])).toBe('Open App Store');
    expect(getDownloadButtonLabel([app(), app({ app_key: 'mobile' })])).toBe('View downloads');
  });

  it('builds ordered section navigation and preserves download insertion', () => {
    const sections = [
      { id: 2, section_type: 'pricing', content: {}, order: 2 },
      { id: 1, section_type: 'hero', content: {}, order: 1 },
    ] satisfies LandingSection[];

    expect(buildNavItems(sections, true, 'downloads')).toEqual([
      { id: 'hero-1', label: 'Overview' },
      { id: 'pricing-2', label: 'Pricing' },
      { id: 'downloads', label: 'Downloads' },
    ]);
    expect(formatDownloadPlatform()).toBe('Download');
    expect(formatDownloadPlatform('mac')).toBe('macOS');
  });
});
