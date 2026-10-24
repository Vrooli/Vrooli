import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, cleanup, fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '@vrooli/api-base/testing';
import { MemoryRouter } from 'react-router-dom';
import { PublicLanding, PRESENTATION_READY_TIMEOUT_MS } from './PublicLanding';
import { publicConfig } from '../presentation/publicTestFixtures';
import { LandingVariantContext, type LandingVariantContextType } from '../../../app/providers/LandingVariantContext';
import type { LandingConfigResponse } from '../../../shared/api/types';
import { fromJsonString } from '@bufbuild/protobuf';
import { ResolvedProductPresentationSchema } from '@vrooli/proto-types/landing-page-business-suite/v1/shared/product_presentation_pb';
import signal from '../presentation/fixtures/signal.json';
import { publicHref } from '../presentation/publicIntegration';
import { recordPresentationExposure } from '../../../shared/api/landing';
vi.mock('../../../shared/api/landing', () => ({ recordPresentationExposure: vi.fn().mockResolvedValue({ recorded: true }) }));
const fontDescriptor = Object.getOwnPropertyDescriptor(document, 'fonts');
function installFonts(statuses: FontFaceLoadStatus[] = ['loaded'], ready: Promise<unknown> = Promise.resolve(), family = 'PresentationSans') {
  const faces = statuses.map(status => ({ family, status }));
  // jsdom has no CSS font inheritance/loading. Supply the computed public family,
  // while retaining real generated page and readiness effects.
  const computedStyle = window.getComputedStyle.bind(window);
  vi.spyOn(window, 'getComputedStyle').mockImplementation((element, pseudo) => {
    const style = computedStyle(element, pseudo);
    if (element.closest('.presentation-page')) style.fontFamily = `"${family}", Arial, sans-serif`;
    return style;
  });
  const fonts = Object.assign(new EventTarget(), {
    ready,
    faces,
    load: vi.fn(async () => { await ready; return faces.filter(face => face.family === family); }),
    forEach: (visit: (face: Pick<FontFace, 'status' | 'family'>) => void) => { faces.forEach(visit); },
  });
  Object.defineProperty(document, 'fonts', { configurable: true, value: fonts });
  return fonts;
}
vi.unmock('../../../app/providers/useLandingVariant');
const metrics = vi.hoisted(() => ({ track: vi.fn(), mounted: vi.fn() }));
vi.mock('../../../shared/hooks/useMetricsHook', () => ({ useMetrics: () => { metrics.mounted(); return { trackCTAClick: metrics.track }; } }));
function mount(config: LandingConfigResponse | null = publicConfig(), path = '/', overrides: Partial<LandingVariantContextType> = {}) {
  const value: LandingVariantContextType = { config, variant: config?.presentation.diagnostics ? { slug: config.presentation.diagnostics.resolvedVariant } : null, loading: false, error: null, resolution: 'api_select', statusNote: null, lastUpdated: null, refresh: vi.fn(), request: { route: path, locale: '', variant: '' }, ...overrides };
  return render(<MemoryRouter basename="/proxy" initialEntries={['/proxy' + path]}><LandingVariantContext.Provider value={value}><PublicLanding /></LandingVariantContext.Provider></MemoryRouter>, { withoutRouter: true });
}
beforeEach(() => { metrics.mounted.mockClear(); metrics.track.mockClear(); vi.mocked(recordPresentationExposure).mockClear(); });
afterEach(() => { cleanup(); document.head.querySelector('meta[name="presentation-canonical-base"]')?.remove(); vi.restoreAllMocks(); vi.useRealTimers(); if (fontDescriptor) Object.defineProperty(document, 'fonts', fontDescriptor); else Reflect.deleteProperty(document, 'fonts'); });
describe('canonical public landing integration', () => {
  it('retains explicit locale/variant on configured detail links without modifying checkout', async () => {
    const config = publicConfig('/', 'fr', 'review'); if (!config.presentation.diagnostics) throw new Error('fixture');
    config.presentation.diagnostics.requestedVariant = 'review';
    config.presentation.actions = [{ $typeName: 'vrooli.landing_page_business_suite.v1.shared.ResolvedPresentationAction', key: JSON.stringify(['purchase', '', 'plan', '']), status: 'ready', href: '/checkout?owner=exact', reason: '', appKey: '', planRef: 'plan' }];
    mount(config, '/', { request: { route: '/', locale: 'fr', variant: 'review' } });
    expect(screen.getByRole('link', { name: 'Configured detail' })).toHaveAttribute('href', '/proxy/apps/example?locale=fr&variant=review');
    expect(screen.getByRole('link', { name: 'Configured purchase' })).toHaveAttribute('href', '/proxy/checkout?owner=exact');
    await waitFor(() => { expect(document.documentElement.dataset.captureReady).toBe('true'); });
    expect(recordPresentationExposure).not.toHaveBeenCalled();
  });
  function qualifiedArt() {
    const config = publicConfig(); config.presentation = fromJsonString(ResolvedProductPresentationSchema, JSON.stringify(signal.presentation));
    const d = config.presentation.diagnostics; if (!d) throw new Error('fixture');
    Object.assign(d, { preview: false, requestedRoute: '/', resolvedRoute: '/', requestedVariant: '', resolvedVariant: 'control', blockDigest: 'digest', locale: config.presentation.page?.locale });
    for (const asset of config.presentation.assets) { asset.releaseRef = 'release'; asset.contentHash = 'a'.repeat(64); }
    d.assetReleaseRefs = ['release'];
    return config;
  }
  it('clears global capture readiness immediately on a late lazy image failure', async () => {
    vi.spyOn(HTMLImageElement.prototype, 'complete', 'get').mockReturnValue(true);
    vi.spyOn(HTMLImageElement.prototype, 'naturalWidth', 'get').mockReturnValue(100);
    const view = mount(qualifiedArt());
    await waitFor(() => { expect(document.documentElement.dataset.captureReady).toBe('true'); });
    const lazy = view.container.querySelector('img[loading="lazy"]'); if (!lazy) throw new Error('Missing lazy image');
    fireEvent.error(lazy);
    expect(document.documentElement.dataset.captureReady).toBe('false');
    expect(document.documentElement.dataset.experienceState).toBe('unavailable');
    expect(screen.getByRole('heading')).toHaveTextContent('This page is currently unavailable');
  });
  it.each(['fonts', 'images'])('bounds unresolved %s readiness and cleans timers on unmount', async kind => {
    vi.useFakeTimers();
    if (kind === 'fonts') installFonts([], new Promise(() => {}));
    const view = mount(kind === 'images' ? qualifiedArt() : publicConfig());
    await act(async () => { await vi.advanceTimersByTimeAsync(PRESENTATION_READY_TIMEOUT_MS); });
    expect(document.documentElement.dataset.captureReady).toBe('false');
    expect(document.documentElement.dataset.experienceState).toBe('unavailable');
    view.unmount(); expect(vi.getTimerCount()).toBe(0);
  });
  it.each(['PresentationSans', 'PresentationArchivo', 'Presentation Mono'])('rejects a failed used %s face even when fonts.ready fulfills, without recording exposure', async family => {
    installFonts(['loaded', 'error'], Promise.resolve(), family);
    const config = publicConfig(); const d = config.presentation.diagnostics; if (!d) throw new Error('fixture');
    d.assignmentSource = 'weighted_visitor'; d.weightFingerprint = 'weights';
    mount(config, '/', { visitorId: 'font_failure_visitor' });
    await waitFor(() => { expect(document.documentElement.dataset.experienceState).toBe('unavailable'); });
    expect(document.documentElement.dataset.captureReady).toBe('false');
    expect(screen.getByRole('heading')).toHaveTextContent('This page is currently unavailable');
    expect(recordPresentationExposure).not.toHaveBeenCalled();
  });
  it('clears ready indicators on late font failure and removes the listener on unmount', async () => {
    const fonts = installFonts(['loaded']);
    const remove = vi.spyOn(fonts, 'removeEventListener');
    const view = mount();
    await waitFor(() => { expect(document.documentElement.dataset.captureReady).toBe('true'); });
    fonts.faces[0]!.status = 'error';
    act(() => { fonts.dispatchEvent(new Event('loadingerror')); });
    expect(document.documentElement.dataset.captureReady).toBe('false');
    expect(document.documentElement.dataset.experienceState).toBe('unavailable');
    expect(screen.getByRole('heading')).toHaveTextContent('This page is currently unavailable');
    act(() => { fonts.dispatchEvent(new Event('loadingdone')); });
    expect(document.documentElement.dataset.captureReady).toBe('false');
    view.unmount();
    expect(remove).toHaveBeenCalledWith('loadingerror', expect.any(Function));
    act(() => { fonts.dispatchEvent(new Event('loadingerror')); });
    expect(document.documentElement.dataset.captureReady).toBeUndefined();
  });
  it('supports test DOMs without the Font Loading API', async () => {
    Object.defineProperty(document, 'fonts', { configurable: true, value: undefined });
    mount();
    await waitFor(() => { expect(document.documentElement.dataset.captureReady).toBe('true'); });
  });
  it('ignores failed and late-failing global/admin faces without waiting on global fonts.ready', async () => {
    const fonts = installFonts();
    fonts.faces.push({ family: 'Space Grotesk', status: 'error' });
    fonts.faces.push({ family: 'Presentation Mono', status: 'error' });
    fonts.ready = new Promise(() => {});
    mount();
    await waitFor(() => { expect(document.documentElement.dataset.captureReady).toBe('true'); });
    act(() => { fonts.dispatchEvent(new Event('loadingerror')); });
    expect(screen.getByRole('heading', { name: 'Published /' })).toBeVisible();
    expect(document.documentElement.dataset.experienceState).toBe('ready');
    expect(document.documentElement.dataset.captureReady).toBe('true');
    expect(fonts.load).toHaveBeenCalledWith('normal 400 16px "PresentationSans"', expect.any(String));
  });
  it.each(['missing', 'rejected', 'unloaded'] as const)('fails closed when a required local face is %s', async kind => {
    const fonts = installFonts(kind === 'unloaded' ? ['unloaded'] : []);
    if (kind === 'rejected') fonts.load.mockRejectedValue(new Error('Local font failed'));
    mount();
    await waitFor(() => { expect(document.documentElement.dataset.experienceState).toBe('unavailable'); });
    expect(document.documentElement.dataset.captureReady).toBe('false');
  });
  it('records only the actually mounted ready weighted page with the bootstrap identity', async () => {
    const config = publicConfig(); const d = config.presentation.diagnostics; if (!d) throw new Error('fixture');
    d.assignmentSource = 'weighted_visitor'; d.weightFingerprint = 'weights';
    mount(config, '/', { visitorId: 'server_visitor' });
    expect(recordPresentationExposure).not.toHaveBeenCalled();
    await waitFor(() => { expect(recordPresentationExposure).toHaveBeenCalledTimes(1); });
    expect(recordPresentationExposure).toHaveBeenCalledWith(expect.objectContaining({ visitorId: 'server_visitor', weightFingerprint: 'weights', revision: 'revision-1' }));
  });
  it.each(['unavailable', 'unsafe-href', 'detail-download'])('keeps the page visible and actions disabled for %s', async failure => {
    const config = publicConfig('/apps/example'); const p = config.presentation;
    const block = p.page?.blocks[0]?.content?.value;
    if (block?.case !== 'closingAction' || !block.value.actions[0]) throw new Error('fixture');
    const action = block.value.actions[0];
    if (failure === 'detail-download') action.kind = 'download';
    p.actions = [{ $typeName: 'vrooli.landing_page_business_suite.v1.shared.ResolvedPresentationAction', key: JSON.stringify([action.kind, '', 'plan', '']), status: failure === 'unavailable' ? 'unavailable' : 'ready', href: failure === 'detail-download' ? '/apps/example' : 'javascript:alert(1)', reason: 'Delivery owner not ready', appKey: '', planRef: 'plan' }];
    mount(config, '/apps/example');
    expect(screen.getByRole('heading')).toHaveTextContent('Published /apps/example');
    expect(screen.getByRole('button', { name: 'Configured purchase' })).toBeDisabled();
    expect(screen.getByText('Delivery owner not ready')).toBeInTheDocument();
    await waitFor(() => { expect(document.documentElement.dataset.captureReady).toBe('true'); });
  });
  it('keeps all internal paths inside the proxy basename and rejects traversal', () => {
    expect(publicHref('/proxy/apps/example', '/proxy/')).toBe('/proxy/apps/example');
    expect(publicHref('/presentation/art.png', '/proxy/')).toBe('/proxy/presentation/art.png');
    expect(() => publicHref('/%2e%2e/admin', '/proxy')).toThrow('Unsafe presentation');
  });
  it.each(['/', '/apps/example'])('renders %s with route metadata, proxy links and capture diagnostics', async route => {
    mount(publicConfig(route), route);
    expect(screen.getByRole('heading', { name: 'Published ' + route })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Configured detail' })).toHaveAttribute('href', '/proxy/apps/example');
    expect(document.querySelector('link[rel="canonical"]')).toBeNull();
    expect(document.title).toBe('Configured ' + route);
    expect(document.querySelector('meta[name="description"]')).toHaveAttribute('content', 'Description en');
    await waitFor(() => { expect(document.documentElement.dataset.captureReady).toBe('true'); });
    expect(document.documentElement.dataset.presentationRevision).toBe('revision-1');
    expect(document.documentElement.dataset.presentationDigest).toBe('digest-1');
    expect(document.documentElement.dataset.variantSlug).toBe('control');
    expect(screen.getByRole('button', { name: 'Configured purchase' })).toBeDisabled();
  });
  it('uses a server-owned canonical base independently of the proxy served path', async () => {
    const meta = document.createElement('meta'); meta.name = 'presentation-canonical-base'; meta.content = 'https://canonical.example/site'; document.head.append(meta);
    mount(publicConfig('/apps/example'), '/apps/example');
    expect(document.querySelector('link[rel="canonical"]')).toHaveAttribute('href', 'https://canonical.example/site/apps/example');
    await waitFor(() => { expect(document.documentElement.dataset.captureReady).toBe('true'); });
  });
  it('joins exact owner action keys and keeps owner outage local to the action', async () => {
    const config = publicConfig();
    config.presentation.actions = [{ $typeName: 'vrooli.landing_page_business_suite.v1.shared.ResolvedPresentationAction', key: JSON.stringify(['purchase', '', 'plan', '']), status: 'ready', href: '/checkout?owner=exact', reason: '', appKey: '', planRef: 'plan' }];
    mount(config);
    expect(screen.getByRole('link', { name: 'Configured purchase' })).toHaveAttribute('href', '/proxy/checkout?owner=exact');
    const link = screen.getByRole('link', { name: 'Configured detail' }); link.addEventListener('click', event => { event.preventDefault(); }); fireEvent.click(link);
    expect(metrics.track).toHaveBeenCalled();
    await waitFor(() => { expect(document.documentElement.dataset.captureReady).toBe('true'); });
  });
  it('keeps missing typed content explicitly unavailable without legacy rendering or analytics', () => {
    mount(null);
    expect(screen.getByRole('heading')).toHaveTextContent('This page is currently unavailable');
    expect(document.body).not.toHaveTextContent('PRIVATE LEGACY PRODUCT');
    expect(metrics.mounted).not.toHaveBeenCalled();
    expect(document.documentElement.dataset.captureReady).toBe('false');
  });
  it('renders unknown/private app 404 state without navigation or narrative', () => {
    mount(null, '/apps/unknown', { notFound: true });
    expect(screen.getByRole('heading')).toHaveTextContent('Page not found');
    expect(screen.getByTestId('landing-experience-surface')).toHaveAttribute('data-http-status', '404');
    expect(document.querySelector('meta[name="robots"]')).toHaveAttribute('content', 'noindex, nofollow');
  });
  it.each(['preview', 'route', 'revision', 'display'])('fails closed for unqualified %s', field => {
    const config = publicConfig(); const p = config.presentation; if (!p.diagnostics || !p.page) throw new Error('fixture');
    if (field === 'preview') p.diagnostics.preview = true;
    if (field === 'route') p.diagnostics.resolvedRoute = '/apps/private';
    if (field === 'revision') p.diagnostics.resolvedRevision = '';
    if (field === 'display') p.page.display = undefined;
    mount(config);
    expect(screen.getByRole('heading')).toHaveTextContent('This page is currently unavailable');
    expect(metrics.mounted).not.toHaveBeenCalled();
  });
  it('rejects assets without owner release correlation', () => {
    const config = publicConfig(); config.presentation = fromJsonString(ResolvedProductPresentationSchema, JSON.stringify(signal.presentation));
    const d = config.presentation.diagnostics; if (!d) throw new Error('fixture');
    d.preview = false; d.requestedRoute = '/'; d.resolvedRoute = '/'; d.requestedVariant = ''; d.resolvedVariant = 'control'; d.blockDigest = 'digest';
    mount(config);
    expect(screen.getByRole('heading')).toHaveTextContent('This page is currently unavailable');
  });
});
