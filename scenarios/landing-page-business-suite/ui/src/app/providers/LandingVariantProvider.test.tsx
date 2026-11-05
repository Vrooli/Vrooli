import { act, cleanup, fireEvent, renderHook, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '@vrooli/api-base/testing';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, useNavigate } from 'react-router-dom';
import type { ReactNode } from 'react';
import { ConnectError, Code } from '@connectrpc/connect';
import { LandingVariantProvider } from './LandingVariantProvider';
import { useLandingVariant } from './useLandingVariant';
import { publicConfig } from '../../surfaces/public-landing/presentation/publicTestFixtures';
import type { LandingConfigResponse } from '../../shared/api/types';
import { create, toJsonString } from '@bufbuild/protobuf';
import { LandingConfigResponseSchema } from '@vrooli/proto-types/landing-page-business-suite/v1/config_pb';
const mocks = vi.hoisted(() => ({ get: vi.fn<typeof import('../../shared/api/landing').getLandingConfig>() }));
vi.unmock('./useLandingVariant');
vi.mock('../../shared/api/landing', async original => ({ ...await original<typeof import('../../shared/api/landing')>(), getLandingConfig: mocks.get }));
beforeEach(() => { mocks.get.mockReset().mockResolvedValue(publicConfig()); localStorage.clear(); document.cookie = 'metrics_visitor_id=; Path=/; Max-Age=0'; });
afterEach(() => { cleanup(); document.getElementById('lpbs-presentation-bootstrap')?.remove(); vi.restoreAllMocks(); });
const wrapper = ({ children }: { children: ReactNode }) => <MemoryRouter><LandingVariantProvider>{children}</LandingVariantProvider></MemoryRouter>;
function Consumer() {
  const value = useLandingVariant(); const navigate = useNavigate();
  return <><span data-testid="state">{value.loading ? 'loading' : value.config?.presentation.diagnostics?.resolvedVariant || 'unavailable'}</span>
    <button onClick={() => { navigate('/apps/example?variant=next&locale=fr'); }}>Detail</button>
    <button onClick={() => { navigate('/admin/presentation/control'); }}>Admin</button></>;
}
describe('route-owned public assignment', () => {
  it('requests only the canonical detail document for the explicit download workflow', async () => {
    render(<MemoryRouter initialEntries={['/apps/example/download?variant=chosen&locale=fr']}><LandingVariantProvider><Consumer /></LandingVariantProvider></MemoryRouter>, { withoutRouter: true });
    await waitFor(() => { expect(mocks.get).toHaveBeenCalledWith('chosen', expect.any(String), expect.objectContaining({ route: '/apps/example', locale: 'fr' })); });
  });
  const bootstrap = (route = '/', preview = false) => {
    const config = publicConfig(route); if (!config.presentation.diagnostics) throw new Error('fixture');
    config.presentation.diagnostics.preview = preview;
    const script = document.createElement('script'); script.id = 'lpbs-presentation-bootstrap'; script.type = 'application/json';
    const raw: unknown = JSON.parse(toJsonString(LandingConfigResponseSchema, create(LandingConfigResponseSchema, { presentation: config.presentation })));
    script.textContent = JSON.stringify({ request: { route, locale: '', variant: '' }, config: raw, canonicalBaseUrl: 'https://canonical.example/site', visitorId: 'server_visitor' }); document.head.append(script);
  };
  it('uses an exact public bootstrap without a second selection or assignment', async () => {
    bootstrap(); localStorage.setItem('metrics_visitor_id', 'old_local'); document.cookie = 'metrics_visitor_id=old_cookie; Path=/';
    const { result } = renderHook(useLandingVariant, { wrapper });
    expect(result.current.config?.presentation.diagnostics?.resolvedRevision).toBe('revision-1');
    expect(result.current.canonicalBaseUrl).toBe('https://canonical.example/site');
    expect(result.current.loading).toBe(false); expect(mocks.get).not.toHaveBeenCalled();
    expect(result.current.visitorId).toBe('server_visitor');
    expect(localStorage.getItem('metrics_visitor_id')).toBe('server_visitor');
    expect(document.cookie).toContain('metrics_visitor_id=server_visitor');
    expect(result.current.refreshable).toBe(true);
    await act(async () => { await result.current.refresh(); });
    expect(mocks.get).toHaveBeenCalledWith(undefined, 'server_visitor', expect.objectContaining({ route: '/', locale: '' }));
    expect(result.current.config?.presentation.diagnostics?.resolvedRevision).toBe('revision-1');
  });
  it('consumes the detail-keyed server bootstrap on a download workflow refresh without reselecting', () => {
    bootstrap('/apps/example');
    const wrapper = ({ children }: { children: ReactNode }) => <MemoryRouter initialEntries={['/apps/example/download']}><LandingVariantProvider>{children}</LandingVariantProvider></MemoryRouter>;
    const { result } = renderHook(useLandingVariant, { wrapper });
    expect(result.current.request?.route).toBe('/apps/example');
    expect(result.current.config?.presentation.diagnostics?.resolvedRevision).toBe('revision-1');
    expect(result.current.canonicalBaseUrl).toBe('https://canonical.example/site');
    expect(mocks.get).not.toHaveBeenCalled();
  });
  it.each(['wrong-route', 'preview'])('rejects %s bootstrap and requests current public scope', async kind => {
    bootstrap(kind === 'wrong-route' ? '/apps/example' : '/', kind === 'preview');
    renderHook(useLandingVariant, { wrapper });
    await waitFor(() => { expect(mocks.get).toHaveBeenCalledTimes(1); });
  });
  it('sends exact route, locale and explicit variant through the typed request', async () => {
    render(<MemoryRouter initialEntries={['/apps/example?variant=chosen&locale=fr']}><LandingVariantProvider><Consumer /></LandingVariantProvider></MemoryRouter>, { withoutRouter: true });
    await waitFor(() => { expect(mocks.get).toHaveBeenCalledWith('chosen', expect.any(String), expect.objectContaining({ route: '/apps/example', locale: 'fr' })); });
    expect(mocks.get.mock.calls[0]?.[2]?.signal).toBeInstanceOf(AbortSignal);
  });
  it('keeps an API outage unavailable without product fallback', async () => {
    mocks.get.mockRejectedValue(new Error('PRIVATE-ERROR'));
    const { result } = renderHook(useLandingVariant, { wrapper });
    await waitFor(() => { expect(result.current.loading).toBe(false); });
    expect(result.current.config).toBeNull(); expect(result.current.variant).toBeNull();
    expect(result.current.error).toBe('This page is currently unavailable.');
  });
  it('recovers from a transient presentation owner outage automatically', async () => {
    mocks.get.mockRejectedValueOnce(new Error('temporary 503')).mockResolvedValueOnce(publicConfig());
    const { result } = renderHook(useLandingVariant, { wrapper });
    await waitFor(() => { expect(result.current.config?.presentation.diagnostics?.resolvedVariant).toBe('control'); }, { timeout: 2000 });
    expect(mocks.get).toHaveBeenCalledTimes(2);
  });
  it('retains not-found classification for unknown/private app routes', async () => {
    mocks.get.mockRejectedValue(new ConnectError('private details', Code.NotFound));
    const routeWrapper = ({ children }: { children: ReactNode }) => <MemoryRouter initialEntries={['/apps/unknown']}><LandingVariantProvider>{children}</LandingVariantProvider></MemoryRouter>;
    const { result } = renderHook(useLandingVariant, { wrapper: routeWrapper });
    await waitFor(() => { expect(result.current.notFound).toBe(true); });
    expect(result.current.config).toBeNull();
  });
  it.each(['/admin/presentation/control', '/admin/presentation/control?revision=secret', '/account', '/not-a-page'])('does not fetch or assign on %s', route => {
    const storage = vi.spyOn(Storage.prototype, 'setItem');
    render(<MemoryRouter initialEntries={[route]}><LandingVariantProvider><Consumer /></LandingVariantProvider></MemoryRouter>, { withoutRouter: true });
    expect(mocks.get).not.toHaveBeenCalled(); expect(storage).not.toHaveBeenCalled();
  });
  it('ignores a late root response after detail navigation and clears on protected navigation', async () => {
    let finish: ((config: LandingConfigResponse) => void) | undefined;
    mocks.get.mockImplementationOnce(() => new Promise<LandingConfigResponse>(resolve => { finish = resolve; }));
    mocks.get.mockResolvedValueOnce(publicConfig('/apps/example', 'fr', 'next'));
    render(<MemoryRouter><LandingVariantProvider><Consumer /></LandingVariantProvider></MemoryRouter>, { withoutRouter: true });
    await waitFor(() => { expect(mocks.get).toHaveBeenCalledTimes(1); });
    fireEvent.click(screen.getByText('Detail'));
    await waitFor(() => { expect(screen.getByTestId('state')).toHaveTextContent('next'); });
    await act(async () => { finish?.(publicConfig()); await Promise.resolve(); });
    expect(screen.getByTestId('state')).toHaveTextContent('next');
    expect(mocks.get.mock.calls[0]?.[2]?.signal?.aborted).toBe(true);
    fireEvent.click(screen.getByText('Admin'));
    expect(screen.getByTestId('state')).toHaveTextContent('unavailable');
    expect(mocks.get).toHaveBeenCalledTimes(2);
  });
  it('supports explicit refresh without accepting a preview response', async () => {
    const { result } = renderHook(useLandingVariant, { wrapper });
    await waitFor(() => { expect(result.current.loading).toBe(false); });
    const config = publicConfig(); if (!config.presentation.diagnostics) throw new Error('fixture');
    config.presentation.diagnostics.preview = true; mocks.get.mockResolvedValueOnce(config);
    await act(async () => { await result.current.refresh(); });
    expect(result.current.config).toBeNull();
  });
});
