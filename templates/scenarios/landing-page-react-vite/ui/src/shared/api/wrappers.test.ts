import { describe, it, expect, vi } from 'vitest';

// A single canned response object serves every Connect client method; each
// thin wrapper reads its own field off it. Mocking createClient keeps the
// tests focused on the wrapper mapping rather than the transport.
const { RESP } = vi.hoisted(() => ({
  RESP: {
    variant: { slug: 'x', name: 'X' },
    variants: [{ slug: 'x' }, { slug: 'y' }],
    section: { id: 1n, sectionType: 'hero' },
    sections: [{ id: 1n }],
    snapshot: { variant: { slug: 'x' }, sections: [] },
    deleted: true,
    success: true,
    apps: [{ appKey: 'a' }],
    app: { appKey: 'a' },
    branding: { siteName: 'Acme' },
    bundles: [{ id: 'solo' }],
    pricing: { plans: [] },
    entries: [{ path: 'a', name: 'a', isDir: false }],
    status: { active: true },
    variantStats: [{ variantSlug: 'x' }],
    stats: [{ variantSlug: 'x' }],
    assets: [{ id: 1n }],
    asset: { platform: 'mac', artifactUrl: 'https://cdn/a.dmg' },
    price: { id: 'price_1' },
    rawJson: new TextEncoder().encode('{"_name":"space","axes":{},"_schemaVersion":1}'),
  },
}));

vi.mock('@connectrpc/connect', async (importActual) => {
  const actual = await importActual<typeof import('@connectrpc/connect')>();
  return {
    ...actual,
    createClient: () => new Proxy({}, { get: () => () => Promise.resolve(RESP) }),
  };
});

import * as variants from './variants';
import * as sections from './sections';
import * as billing from './billing';
import * as downloads from './downloads';
import * as branding from './branding';
import * as seo from './seo';
import * as landing from './landing';
import * as docs from './docs';
import * as metrics from './metrics';
import * as account from './account';
import * as auth from './auth';
import * as admin from './admin';
import * as assets from './assets';

describe('variant API wrappers', () => {
  it('maps single-variant responses', async () => {
    expect(await variants.selectVariant()).toEqual(RESP.variant);
    expect(await variants.getPublicVariant('x')).toEqual(RESP.variant);
    expect(await variants.getVariant('x')).toEqual(RESP.variant);
    expect(await variants.createVariant({ name: 'X', slug: 'x', axes: {} })).toEqual(RESP.variant);
    expect(await variants.updateVariant('x', { name: 'X', axes: { tone: 'bold' } })).toEqual(RESP.variant);
    expect(await variants.importVariantSnapshot('x', RESP.snapshot as never)).toEqual(RESP.variant);
    expect(await variants.archiveVariant('x')).toEqual(RESP.variant);
  });

  it('maps list, snapshot, and boolean responses', async () => {
    expect(await variants.listVariants('active')).toEqual(RESP.variants);
    expect(await variants.listVariants()).toEqual(RESP.variants);
    expect(await variants.exportVariantSnapshot('x')).toEqual(RESP.snapshot);
    expect(await variants.deleteVariant('x')).toBe(true);
  });

  it('decodes the raw JSON variant space', async () => {
    const space = await variants.getVariantSpace();
    expect(space._name).toBe('space');
    expect(space.axes).toEqual({});
  });
});

describe('section API wrappers', () => {
  it('maps section responses', async () => {
    expect(await sections.getSections(1n)).toEqual(RESP.sections);
    expect(await sections.getAdminSections(1n)).toEqual(RESP.sections);
    expect(await sections.getSection(1n)).toEqual(RESP.section);
    expect(await sections.updateSection(1n, { title: 'hi' })).toBeDefined();
    expect(await sections.createSection({ variantId: 1n, sectionType: 'hero', content: {} } as never)).toBeDefined();
    expect(await sections.deleteSection(1n)).toBe(true);
  });
});

describe('billing / downloads / branding wrappers', () => {
  it('maps billing responses', async () => {
    expect(await billing.getStripeSettings()).toEqual(RESP);
    expect(await billing.getBundleCatalog()).toEqual(RESP.bundles);
    await expect(billing.updateStripeSettings({} as never)).resolves.toBeDefined();
    await expect(billing.updateBundlePrice('bundle', 'price', {} as never)).resolves.toBeDefined();
  });

  it('maps download responses', async () => {
    expect(await downloads.listDownloadAppsAdmin()).toEqual(RESP.apps);
    expect(await downloads.createDownloadAppAdmin({} as never)).toEqual(RESP.app);
    await expect(downloads.saveDownloadAppAdmin('a', {} as never)).resolves.toBeDefined();
    await expect(downloads.requestDownload('a', 'mac')).resolves.toBeDefined();
  });

  it('maps branding responses', async () => {
    expect(await branding.getBranding()).toEqual(RESP.branding);
    expect(await branding.updateBranding({} as never)).toEqual(RESP.branding);
    expect(await branding.clearBrandingField('logoUrl')).toEqual(RESP.branding);
    expect(await branding.getPublicBranding()).toEqual(RESP.branding);
  });
});

describe('seo / landing / docs / metrics wrappers', () => {
  it('maps seo and landing responses', async () => {
    await expect(seo.getVariantSEO('x')).resolves.toEqual(RESP);
    await expect(seo.updateVariantSEO('x', {} as never)).resolves.toEqual(RESP);
    await expect(landing.getLandingConfig('x')).resolves.toEqual(RESP);
    await expect(landing.getLandingConfig()).resolves.toEqual(RESP);
    expect(await landing.getPlans()).toEqual(RESP.pricing);
  });

  it('maps docs and metrics responses', async () => {
    expect(await docs.getDocsTree()).toEqual(RESP.entries);
    await expect(docs.getDocContent('a')).resolves.toEqual(RESP);
    await expect(metrics.trackMetric({ eventType: 'view' } as never)).resolves.toBeDefined();
    await expect(metrics.getMetricsSummary('2025-01-01', '2025-01-08')).resolves.toEqual(RESP);
    await expect(metrics.getMetricsSummary()).resolves.toEqual(RESP);
    await expect(metrics.getVariantMetrics('x', '2025-01-01', '2025-01-08')).resolves.toBeDefined();
  });
});

describe('account / auth / admin wrappers', () => {
  it('maps account responses', async () => {
    expect(await account.getSubscriptionInfo()).toEqual(RESP.status);
    await expect(account.getCreditInfo()).resolves.toEqual(RESP);
    await expect(account.getEntitlements()).resolves.toEqual(RESP);
  });

  it('maps auth responses', async () => {
    await expect(auth.adminLogin('e@x.com', 'pw')).resolves.toEqual(RESP);
    expect(await auth.adminLogout()).toBe(true);
    await expect(auth.checkAdminSession()).resolves.toEqual(RESP);
  });

  it('maps admin reset response', async () => {
    await expect(admin.resetDemoData()).resolves.toEqual(RESP);
  });
});

describe('asset wrappers', () => {
  it('maps connect-backed asset list and delete', async () => {
    expect(await assets.listAssets('logo')).toBeDefined();
    expect(await assets.deleteAsset(1n)).toBe(true);
  });

  it('resolves asset URLs across passthrough and prefix branches', () => {
    expect(assets.getAssetUrl('')).toBe('');
    expect(assets.getAssetUrl('https://cdn/a.png')).toBe('https://cdn/a.png');
    expect(assets.getAssetUrl('http://cdn/a.png')).toBe('http://cdn/a.png');
    expect(assets.getAssetUrl('data:image/png;base64,AAAA')).toBe('data:image/png;base64,AAAA');
    expect(assets.getAssetUrl('logos/a.png')).toContain('/uploads/logos/a.png');
    expect(assets.getAssetUrl('/logos/a.png')).toContain('/uploads/logos/a.png');
  });
});
