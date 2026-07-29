import { beforeEach, describe, expect, it, vi } from 'vitest';
import { create } from '@bufbuild/protobuf';
import { VariantSchema as VariantMessageSchema, VariantSnapshotSchema as VariantSnapshotMessageSchema } from '@vrooli/proto-types/landing-page-business-suite/variant_pb';
import * as variants from './variants';

const { variantClient, variantSpaceClient, seoClient } = vi.hoisted(() => ({
  variantClient: {
    selectVariant: vi.fn(), getPublicVariant: vi.fn(), getVariant: vi.fn(), listVariants: vi.fn(), createVariant: vi.fn(),
    updateVariant: vi.fn(), archiveVariant: vi.fn(), deleteVariant: vi.fn(), exportVariantSnapshot: vi.fn(), importVariantSnapshot: vi.fn(),
  },
  variantSpaceClient: { getVariantSpace: vi.fn() },
  seoClient: { getVariantSEO: vi.fn(), updateVariantSEO: vi.fn() },
}));

vi.mock('@connectrpc/connect', () => ({
  createClient: vi.fn((service: { typeName?: string }) => {
    if (service.typeName?.endsWith('.VariantService')) return variantClient;
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
});
