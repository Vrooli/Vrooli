import { beforeEach, describe, expect, it, vi } from 'vitest';
import { create } from '@bufbuild/protobuf';
import { VariantSchema as VariantMessageSchema, VariantSnapshotSchema as VariantSnapshotMessageSchema } from '@vrooli/proto-types/landing-page-business-suite/v1/variant_pb';
import * as variants from './variants';

const { variantClient, variantSectionClient, variantSpaceClient, seoClient } = vi.hoisted(() => ({
  variantClient: {
    selectVariant: vi.fn(), getPublicVariant: vi.fn(), getVariant: vi.fn(), listVariants: vi.fn(), createVariant: vi.fn(),
    updateVariant: vi.fn(), archiveVariant: vi.fn(), deleteVariant: vi.fn(), exportVariantSnapshot: vi.fn(), importVariantSnapshot: vi.fn(),
  },
  variantSectionClient: {
    getVariantSections: vi.fn(), getVariantSection: vi.fn(), createVariantSection: vi.fn(),
    updateVariantSection: vi.fn(), deleteVariantSection: vi.fn(),
  },
  variantSpaceClient: { getVariantSpace: vi.fn() },
  seoClient: { getVariantSEO: vi.fn(), updateVariantSEO: vi.fn() },
}));

vi.mock('@connectrpc/connect', () => ({
  createClient: vi.fn((service: { typeName?: string }) => {
    if (service.typeName?.endsWith('.VariantService')) return variantClient;
    if (service.typeName?.endsWith('.VariantSectionService')) return variantSectionClient;
    if (service.typeName?.endsWith('.SeoService')) return seoClient;
    return variantSpaceClient;
  }),
}));
vi.mock('@vrooli/api-base', () => ({ createScenarioConnectTransport: vi.fn(), resolveApiBase: vi.fn(() => 'http://localhost:17691/api/v1'), DEFAULT_API_SUFFIX: '/api/v1' }));

const protoVariant = create(VariantMessageSchema, { slug: 'control', name: 'Control', weight: 100, status: 'active', axes: {} });
const protoSnapshot = create(VariantSnapshotMessageSchema, { slug: 'control', name: 'Control', weight: 100, status: 'active', axes: {}, sections: [] });

describe('variant API transport', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    variantClient.getPublicVariant.mockResolvedValue({ variant: protoVariant });
    variantClient.getVariant.mockResolvedValue({ variant: protoVariant });
    variantClient.listVariants.mockResolvedValue({ variants: [protoVariant] });
    variantClient.createVariant.mockResolvedValue({ variant: protoVariant });
    variantClient.updateVariant.mockResolvedValue({ variant: protoVariant });
    variantClient.archiveVariant.mockResolvedValue({ variant: create(VariantMessageSchema, { slug: 'control', name: 'Control', weight: 100, status: 'archived', axes: {} }) });
    variantClient.deleteVariant.mockResolvedValue({ deleted: true });
    variantClient.selectVariant.mockResolvedValue({ variant: protoVariant });
    variantClient.exportVariantSnapshot.mockResolvedValue({ snapshot: protoSnapshot });
    variantClient.importVariantSnapshot.mockResolvedValue({ snapshot: protoSnapshot });
    variantSectionClient.getVariantSections.mockResolvedValue({ sections: [] });
    variantSectionClient.getVariantSection.mockResolvedValue({});
    variantSectionClient.createVariantSection.mockResolvedValue({});
    variantSectionClient.updateVariantSection.mockResolvedValue({});
    variantSectionClient.deleteVariantSection.mockResolvedValue({ deleted: true });
    variantSpaceClient.getVariantSpace.mockResolvedValue({ rawJson: new TextEncoder().encode('{"_name":"Default","_schemaVersion":1,"axes":{}}') });
    seoClient.getVariantSEO.mockResolvedValue({ siteName: 'Example', title: 'Title', description: 'Description', ogTitle: 'OG Title', ogDescription: 'OG Description', noindex: false });
    seoClient.updateVariantSEO.mockResolvedValue({ success: true, updatedAt: '2026-07-28T00:00:00Z' });
  });

  it('uses the generated VariantService for every lifecycle operation', async () => {
    await expect(variants.getPublicVariant('control')).resolves.toMatchObject({ slug: 'control' });
    await expect(variants.getVariant('control')).resolves.toMatchObject({ name: 'Control' });
    await expect(variants.listVariants()).resolves.toMatchObject({ variants: [{ slug: 'control' }] });
    await expect(variants.createVariant({ name: 'Control', slug: 'control', axes: {} })).resolves.toMatchObject({ status: 'active' });
    await expect(variants.updateVariant('control', { weight: 50 })).resolves.toMatchObject({ weight: 100 });
    await expect(variants.archiveVariant('control')).resolves.toMatchObject({ status: 'archived' });
    await expect(variants.deleteVariant('control')).resolves.toEqual({ success: true });
    await expect(variants.selectVariant()).resolves.toMatchObject({ slug: 'control' });
    expect(variantClient.getPublicVariant).toHaveBeenCalledWith({ slug: 'control' });
    expect(variantClient.createVariant).toHaveBeenCalledWith(expect.objectContaining({ slug: 'control', name: 'Control' }));
    expect(variantClient.archiveVariant).toHaveBeenCalledWith({ slug: 'control' });
  });

  it('preserves portable snapshot import and export semantics', async () => {
    await expect(variants.exportVariantSnapshot('control')).resolves.toMatchObject({ variant: { slug: 'control' }, sections: [] });
    await expect(variants.importVariantSnapshot('control', { variant: { slug: 'control', name: 'Control', axes: {} }, sections: [] })).resolves.toMatchObject({ variant: { slug: 'control' } });
    expect(variantClient.exportVariantSnapshot).toHaveBeenCalledWith({ slug: 'control' });
    expect(variantClient.importVariantSnapshot).toHaveBeenCalledWith(expect.objectContaining({ slug: 'control' }));
  });

  it('continues to validate the generated variant-space and SEO responses', async () => {
    await expect(variants.getVariantSpace()).resolves.toMatchObject({ _name: 'Default', axes: {} });
    await expect(variants.getVariantSEO('control')).resolves.toMatchObject({ site_name: 'Example' });
    await expect(variants.updateVariantSEO('control', {})).resolves.toEqual({ success: true });
  });

  it('preserves complete header and SEO configuration when updating a variant', async () => {
    const header = {
      branding: { mode: 'logo_and_name' as const, label: 'Acme', subtitle: 'Launch faster', mobile_preference: 'stacked' as const },
      nav: { links: [{
        id: 'products', type: 'menu' as const, label: 'Products', section_type: 'features', section_id: 42,
        anchor: 'features', href: '/products', visible_on: { desktop: false, mobile: true },
        children: [{ id: 'download', type: 'downloads' as const, label: 'Download', visible_on: { desktop: true, mobile: false } }],
      }] },
      ctas: {
        primary: { mode: 'custom' as const, label: 'Start', href: '/start', variant: 'solid' as const },
        secondary: { mode: 'hidden' as const, label: 'Later', href: '/later', variant: 'ghost' as const },
      },
      behavior: { sticky: true, hide_on_scroll: false },
    };
    variantClient.updateVariant.mockResolvedValue({ variant: protoVariant });
    seoClient.getVariantSEO.mockResolvedValue({
      siteName: 'Acme', title: 'Launch', description: 'Description', ogTitle: 'OG', ogDescription: 'OG description',
      ogImageUrl: '', twitterCard: '', canonicalUrl: '', faviconUrl: '', appleTouchIconUrl: '', themePrimaryColor: '',
      noindex: true, structuredData: { '@type': 'SoftwareApplication' },
    });

    await expect(variants.updateVariant('control', { name: 'Control', description: 'Updated', weight: 15, axes: { persona: 'operator' }, header_config: header })).resolves.toMatchObject({ slug: 'control' });
    await expect(variants.getVariantSEO('control')).resolves.toMatchObject({
      site_name: 'Acme', noindex: true, structured_data: { '@type': 'SoftwareApplication' },
      og_image_url: undefined, twitter_card: undefined, canonical_url: undefined,
    });

    expect(variantClient.updateVariant).toHaveBeenCalledWith({
      slug: 'control', name: 'Control', description: 'Updated', weight: 15, axes: { values: { persona: 'operator' } },
      headerConfig: {
        branding: { mode: 'logo_and_name', label: 'Acme', subtitle: 'Launch faster', mobilePreference: 'stacked' },
        nav: { links: [{
          id: 'products', type: 'menu', label: 'Products', sectionType: 'features', sectionId: 42, anchor: 'features', href: '/products',
          visibleOn: { desktop: false, mobile: true }, children: [{ id: 'download', type: 'downloads', label: 'Download', sectionType: '', sectionId: undefined, anchor: '', href: '', visibleOn: { desktop: true, mobile: false }, children: [] }],
        }] },
        ctas: {
          primary: { mode: 'custom', label: 'Start', href: '/start', variant: 'solid' },
          secondary: { mode: 'hidden', label: 'Later', href: '/later', variant: 'ghost' },
        },
        behavior: { sticky: true, hideOnScroll: false },
      },
    });
  });

  it('maps every variant-section lifecycle operation through the typed client', async () => {
    const section = {
      key: 'hero', sectionType: 'hero', content: { title: 'Welcome', nested: { enabled: true } },
      order: 2, enabled: true,
    };
    variantSectionClient.getVariantSections.mockResolvedValue({ sections: [section] });
    variantSectionClient.getVariantSection.mockResolvedValue({ section });
    variantSectionClient.createVariantSection.mockResolvedValue({ section });
    variantSectionClient.updateVariantSection.mockResolvedValue({ section: { ...section, enabled: false } });

    await expect(variants.getVariantSections('control')).resolves.toEqual({ sections: [expect.objectContaining({ key: 'hero', section_type: 'hero' })] });
    await expect(variants.getVariantSection('control', 'hero')).resolves.toMatchObject({ key: 'hero', content: section.content });
    await expect(variants.createVariantSection('control', { key: 'hero', section_type: 'hero', content: section.content, order: 2, enabled: true })).resolves.toMatchObject({ key: 'hero' });
    await expect(variants.updateVariantSection('control', 'hero', { content: section.content, order: 3, enabled: false })).resolves.toMatchObject({ enabled: false });
    await expect(variants.deleteVariantSection('control', 'hero')).resolves.toEqual({ success: true });

    expect(variantSectionClient.getVariantSections).toHaveBeenCalledWith({ slug: 'control' });
    expect(variantSectionClient.getVariantSection).toHaveBeenCalledWith({ slug: 'control', sectionKey: 'hero' });
    expect(variantSectionClient.createVariantSection).toHaveBeenCalledWith(expect.objectContaining({ slug: 'control' }));
    expect(variantSectionClient.updateVariantSection).toHaveBeenCalledWith(expect.objectContaining({
      slug: 'control', sectionKey: 'hero', content: section.content, order: 3, enabled: false,
    }));
  });

  it('fails closed when a section response lacks its stable transport identity', async () => {
    variantSectionClient.getVariantSection.mockResolvedValue({ section: { key: 'hero' } });
    await expect(variants.getVariantSection('control', 'hero')).rejects.toThrow('stable key and section type');

    variantSectionClient.getVariantSection.mockResolvedValue({ section: { sectionType: 'hero' } });
    await expect(variants.getVariantSection('control', 'hero')).rejects.toThrow('stable key and section type');
  });

  it('rejects malformed generated payloads instead of exposing partial variant data', async () => {
    variantClient.getVariant.mockResolvedValue({});
    await expect(variants.getVariant('control')).rejects.toThrow('Variant response did not include a variant');

    variantClient.exportVariantSnapshot.mockResolvedValue({});
    await expect(variants.exportVariantSnapshot('control')).rejects.toThrow('Variant snapshot response did not include a snapshot');

    variantSpaceClient.getVariantSpace.mockResolvedValue({ rawJson: new TextEncoder().encode('not-json') });
    await expect(variants.getVariantSpace()).rejects.toThrow('Invalid variant space response from API');

    variantSectionClient.getVariantSection.mockResolvedValue({ section: { key: 'footer', sectionType: 'footer', order: 0, enabled: false } });
    await expect(variants.getVariantSection('control', 'footer')).resolves.toMatchObject({ content: {}, enabled: false });
  });

  it('rejects non-object section content before it reaches the transport', async () => {
    expect(() => variants.createVariantSection('control', {
      key: 'hero', section_type: 'hero', content: ['invalid'] as unknown as Record<string, unknown>, order: 1, enabled: true,
    })).toThrow('Section content must be a JSON object');
    variantSectionClient.updateVariantSection.mockResolvedValue({
      section: { key: 'hero', sectionType: 'hero', content: {}, order: 1, enabled: true },
    });
    await expect(variants.updateVariantSection('control', 'hero', {
      content: null as unknown as Record<string, unknown>,
    })).resolves.toMatchObject({ key: 'hero' });
    expect(variantSectionClient.updateVariantSection).toHaveBeenLastCalledWith({
      slug: 'control', sectionKey: 'hero', sectionType: undefined, content: undefined, order: undefined, enabled: undefined,
    });
    expect(variantSectionClient.createVariantSection).not.toHaveBeenCalled();
  });

  it('preserves explicit false/zero section patches and rejects non-JSON nested content', async () => {
    variantSectionClient.updateVariantSection.mockResolvedValue({
      section: { key: 'hero', sectionType: 'hero', content: {}, order: 0, enabled: false },
    });
    await expect(variants.updateVariantSection('control', 'hero', {
      section_type: 'hero', content: {}, order: 0, enabled: false,
    })).resolves.toMatchObject({ order: 0, enabled: false });
    expect(variantSectionClient.updateVariantSection).toHaveBeenLastCalledWith({
      slug: 'control', sectionKey: 'hero', sectionType: 'hero', content: {}, order: 0, enabled: false,
    });

    expect(() => variants.createVariantSection('control', {
      key: 'hero', section_type: 'hero', content: { bad: () => undefined }, order: 0, enabled: false,
    })).toThrow('Section content must be a JSON object');
  });

  it('fails closed for structurally invalid variant-space while preserving empty optional SEO fields', async () => {
    variantSpaceClient.getVariantSpace.mockResolvedValue({ rawJson: new TextEncoder().encode('[]') });
    await expect(variants.getVariantSpace()).rejects.toThrow('Invalid variant space response from API');

    seoClient.getVariantSEO.mockResolvedValue({ siteName: '', title: '', description: '', ogTitle: '', ogDescription: '', noindex: false });
    await expect(variants.getVariantSEO('control')).resolves.toMatchObject({ site_name: '', title: '', noindex: false });

    seoClient.updateVariantSEO.mockResolvedValue({ success: false, updatedAt: '' });
    await expect(variants.updateVariantSEO('control', {})).resolves.toEqual({ success: false });
  });
});
