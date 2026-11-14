import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '@vrooli/api-base/testing';
import { MemoryRouter } from 'react-router-dom';
import { create } from '@bufbuild/protobuf';
import { LandingConfigResponseSchema } from '@vrooli/proto-types/landing-page-business-suite/v1/config_pb';
import { decodeLandingConfig } from '../../../shared/api/landing';
import { publicConfig } from './publicTestFixtures';
import { safeOwnerHref } from './publicIntegration';
import { DownloadPage } from './DownloadPage';
import { DownloadChooser } from './DownloadChooser';
import { PublicLanding } from '../routes/PublicLanding';
import { LandingVariantContext, type LandingVariantContextType } from '../../../app/providers/LandingVariantContext';
import { UserAuthContext, type UserAuthContextValue } from '../../../app/providers/UserAuthContext';
vi.unmock('../../../app/providers/useLandingVariant');
vi.unmock('../../../app/providers/useUserAuth');
vi.mock('../../../shared/hooks/useMetricsHook', () => ({ useMetrics: () => ({ trackCTAClick: vi.fn() }) }));
const auth: UserAuthContextValue = { user: null, isAuthenticated: false, isSessionLoading: false, logout: vi.fn(), refreshSession: vi.fn().mockResolvedValue(undefined) };
function fixture(url = '/app/example-app') {
  const config = publicConfig('/apps/example'); const p = config.presentation;
  const closing = p.page?.blocks[0]?.content?.value; const shell = p.page?.display?.shell;
  if (!p.diagnostics || closing?.case !== 'closingAction' || !closing.value.actions[0] || !shell) throw new Error('fixture');
  const action = closing.value.actions[0]; Object.assign(action, { kind: 'open', appKey: 'example-app', planRef: '', label: 'Configured web launch', accessibleLabel: 'Configured web launch' });
  shell.headerAction = { ...action };
  closing.value.actions.push({ ...action, kind: 'download', label: 'Configured installer', accessibleLabel: 'Configured installer' });
  p.appKey = 'example-app'; p.diagnostics.bundleKey = 'example'; p.diagnostics.eligibleAppKeys = ['example-app'];
  p.actions = [
    { $typeName: 'vrooli.landing_page_business_suite.v1.shared.ResolvedPresentationAction', key: JSON.stringify(['open', 'example-app', '', '']), status: 'ready', href: url, reason: '', appKey: 'example-app', planRef: '' },
    { $typeName: 'vrooli.landing_page_business_suite.v1.shared.ResolvedPresentationAction', key: JSON.stringify(['download', 'example-app', '', '']), status: 'unavailable', href: '', reason: 'No released installers', appKey: 'example-app', planRef: '' },
  ];
  // Exercise the real generated public response/Struct normalization, not a sidecar.
  return decodeLandingConfig(create(LandingConfigResponseSchema, { presentation: p, downloads: [{ bundleKey: 'example', appKey: 'example-app', name: 'Unrelated delivery marketing', metadata: { enabled: true, web_url: url }, platforms: [] }] }));
}
function mount(config = fixture(), download = true, canonicalBaseUrl?: string) {
  const value: LandingVariantContextType = { config, variant: { slug: config.presentation.diagnostics?.resolvedVariant ?? '' }, loading: false, error: null, resolution: 'api_select', statusNote: null, lastUpdated: null, refresh: vi.fn(), request: { route: '/apps/example', locale: '', variant: '' }, canonicalBaseUrl };
  return render(<MemoryRouter basename="/proxy" initialEntries={['/proxy/apps/example' + (download ? '/download' : '')]}><LandingVariantContext.Provider value={value}><UserAuthContext.Provider value={auth}>{download ? <DownloadPage /> : <PublicLanding />}</UserAuthContext.Provider></LandingVariantContext.Provider></MemoryRouter>, { withoutRouter: true });
}
beforeEach(() => { vi.mocked(auth.refreshSession).mockClear(); });
afterEach(cleanup);
describe('owner-backed web-only launch', () => {
  it('uses configured detail canonical authority while keeping the workflow noindex', () => {
    mount(fixture(), true, 'https://canonical.example/site');
    expect(document.querySelector('link[rel="canonical"]')).toHaveAttribute('href', 'https://canonical.example/site/apps/example');
    expect(document.querySelector('meta[property="og:url"]')).toHaveAttribute('content', 'https://canonical.example/site/apps/example');
    expect(document.querySelector('meta[name="robots"]')).toHaveAttribute('content', 'noindex, nofollow');
    expect(document.title).toBe('Configured /apps/example');
  });
  it('does not guess canonical authority for the workflow', () => {
    mount(); expect(document.querySelector('link[rel="canonical"]')).toBeNull(); expect(document.querySelector('meta[property="og:url"]')).toBeNull();
  });
  it('keeps a configured web-only preview effect-free and unavailable without an owner join', () => {
    render(<DownloadChooser title="Configured preview" description="Configured description" options={[]} selected="" onSelect={vi.fn()} state={{ status: 'idle' }} unavailableReason="Owner unavailable" launchActions={[{ kind: 'open', app_key: 'example-app', label: 'Configured launch', accessible_label: 'Configured launch' }]} />);
    expect(screen.getByRole('button', { name: 'Configured launch' })).toBeDisabled();
    expect(auth.refreshSession).not.toHaveBeenCalled();
  });
  it('preserves generated metadata and exposes configured web launch with truthful installer state', () => {
    const config = fixture(); expect(config.downloads[0]?.metadata?.web_url).toBe('/app/example-app'); mount(config);
    expect(screen.getByRole('link', { name: 'Configured web launch' })).toHaveAttribute('href', '/proxy/app/example-app');
    expect(screen.getByText('No installers are currently available for this app.')).toBeInTheDocument();
    expect(screen.queryByRole('combobox')).not.toBeInTheDocument(); expect(screen.queryByRole('link', { name: 'Sign in' })).not.toBeInTheDocument(); expect(auth.refreshSession).not.toHaveBeenCalled();
    expect(document.body).not.toHaveTextContent('Unrelated delivery marketing');
  });
  it('enables every configured web-launch CTA while keeping installer CTAs unavailable', async () => {
    mount(fixture(), false); const links = screen.getAllByRole('link', { name: 'Configured web launch' }); expect(links).toHaveLength(2);
    for (const link of links) expect(link).toHaveAttribute('href', '/proxy/app/example-app');
    expect(screen.getByRole('button', { name: 'Configured installer' })).toBeDisabled();
    await waitFor(() => { expect(document.documentElement.dataset.captureReady).toBe('true'); });
  });
  it('does not prefix a validated external HTTPS launch with the proxy basename', () => {
    mount(fixture('https://launch.example.test/session?view=main')); expect(screen.getByRole('link', { name: 'Configured web launch' })).toHaveAttribute('href', 'https://launch.example.test/session?view=main');
  });
  it.each(['wrong-bundle', 'missing-row', 'duplicate', 'disabled', 'private', 'mismatch', 'owner-unavailable', 'missing-url', 'marketing', 'absolute-marketing'])('fails closed for %s without promoting a marketing target or installing fallback data', failure => {
    const config = fixture(failure === 'marketing' ? '/apps/example' : failure === 'absolute-marketing' ? 'https://marketing.test/apps/example' : undefined);
    const row = config.downloads[0]; const p = config.presentation; if (!row || !p.diagnostics || !p.actions[0]) throw new Error('fixture');
    if (failure === 'wrong-bundle') row.bundle_key = 'private-bundle';
    if (failure === 'missing-row') config.downloads = [];
    if (failure === 'duplicate') config.downloads.push({ ...row });
    if (failure === 'disabled') row.metadata = { ...row.metadata, enabled: false };
    if (failure === 'private') p.diagnostics.eligibleAppKeys = [];
    if (failure === 'mismatch') p.actions[0].href = '/invented';
    if (failure === 'owner-unavailable') p.actions[0].status = 'unavailable';
    if (failure === 'missing-url') row.metadata = {};
    mount(config); expect(screen.queryByRole('link', { name: 'Configured web launch' })).not.toBeInTheDocument(); expect(screen.getByRole('button', { name: 'Configured web launch' })).toBeDisabled();
  });
});
describe('strict owner URL policy', () => {
  it.each(['//evil.test', '/\\evil.test', 'https:\\evil.test', 'javascript:alert(1)', 'data:text/plain,x', 'http://launch.test', 'https://user:pass@launch.test', '/%2f%2fevil.test', '/%5cevil.test', '/a/../admin', 'https://launch.test/a/../admin', '/%2e%2e/admin', '/%00app', '/app\u007f', '/app\n', '/app name', '#launch', 'app/example'])('rejects %s', value => {
    expect(safeOwnerHref(value, '/proxy')).toBeUndefined();
  });
  it('preserves opaque query tokens and already-based app paths', () => {
    expect(safeOwnerHref('/proxy/app/example?token=a%2Fb', '/proxy')).toBe('/proxy/app/example?token=a%2Fb');
    expect(safeOwnerHref('https://launch.test/file?token=a%2Fb', '/proxy')).toBe('https://launch.test/file?token=a%2Fb');
  });
});
