import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, cleanup, fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '@vrooli/api-base/testing';
import { MemoryRouter } from 'react-router-dom';
import { Code, ConnectError } from '@connectrpc/connect';
import { DownloadPage } from './DownloadPage';
import { DownloadChooser } from './DownloadChooser';
import { publicConfig } from './publicTestFixtures';
import { downloadOptions } from './commerceFixtures';
import { LandingVariantContext, type LandingVariantContextType } from '../../../app/providers/LandingVariantContext';
import { UserAuthContext, type UserAuthContextValue } from '../../../app/providers/UserAuthContext';
import type { DownloadAsset } from '../../../shared/api/types';
const mocks = vi.hoisted(() => ({ request: vi.fn<typeof import('../../../shared/api/downloads').requestDownload>() }));
vi.mock('../../../shared/api/downloads', () => ({ requestDownload: mocks.request }));
vi.unmock('../../../app/providers/useLandingVariant');
vi.unmock('../../../app/providers/useUserAuth');
const user = { id: 'user-1', email: 'private@example.test', email_verified: true };
const auth: UserAuthContextValue = { user, isAuthenticated: true, isSessionLoading: false, logout: vi.fn(), refreshSession: vi.fn().mockResolvedValue(undefined) };
function mount(options = downloadOptions, session = auth, overrides: Partial<LandingVariantContextType> = {}) {
  const config = publicConfig('/apps/example'); if (!config.presentation.diagnostics) throw new Error('fixture'); config.presentation.appKey = 'example-app'; config.presentation.diagnostics.bundleKey = 'example';
  config.downloads = [{ bundle_key: 'example', app_key: 'example-app', name: 'DO NOT COPY DELIVERY MARKETING', platforms: options, display_order: 0 }];
  const value: LandingVariantContextType = { config, variant: { slug: config.presentation.diagnostics.resolvedVariant }, loading: false, error: null, resolution: 'api_select', statusNote: null, lastUpdated: null, refresh: vi.fn(), request: { route: '/apps/example', locale: '', variant: '' }, ...overrides };
  return render(<MemoryRouter basename="/proxy" initialEntries={['/proxy/apps/example/download']}><LandingVariantContext.Provider value={value}><UserAuthContext.Provider value={session}><DownloadPage /></UserAuthContext.Provider></LandingVariantContext.Provider></MemoryRouter>, { withoutRouter: true });
}
beforeEach(() => { mocks.request.mockReset(); vi.mocked(auth.refreshSession).mockClear(); mocks.request.mockResolvedValue({ ...downloadOptions[0]!, artifact_url: '/authorized/file?token=opaque' }); });
afterEach(() => { cleanup(); vi.restoreAllMocks(); });
const select = (index = '0') => fireEvent.change(screen.getByRole('combobox'), { target: { value: index } });
describe('configured authorized download workflow', () => {
  it('shows all platform/release choices and preselects the detected platform without copying delivery marketing', () => {
    mount(); expect(screen.getByRole('combobox')).toHaveValue('0');
    expect(screen.getByRole('option', { name: /Linux · 2.4.1/ })).toBeInTheDocument(); expect(screen.getByRole('option', { name: /macOS · 2.4.0/ })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Prepare download' })).toBeEnabled(); expect(mocks.request).not.toHaveBeenCalled();
    expect(document.body).not.toHaveTextContent(/DO NOT COPY|private@example|Aquila/);
    expect(screen.getByRole('link', { name: 'Back to app' })).toHaveAttribute('href', '/proxy/apps/example');
    expect(document.querySelector('meta[name="robots"]')).toHaveAttribute('content', 'noindex, nofollow');
  });
  it('authorizes the explicit selection with the existing service and only then exposes its safe URL', async () => {
    mount(); select(); fireEvent.click(screen.getByRole('button', { name: 'Prepare download' }));
    expect(mocks.request).toHaveBeenCalledExactlyOnceWith('example-app', 'linux', undefined, { assetId: 11 });
    expect(await screen.findByRole('link', { name: 'Download file' })).toHaveAttribute('href', '/proxy/authorized/file?token=opaque');
    expect(document.body).not.toHaveTextContent('Download started');
  });
  it('uses the existing sign-in state and proxy-safe login, without authorizing a gated guest', () => {
    mount(downloadOptions, { ...auth, user: null, isAuthenticated: false }); select('1');
    expect(screen.getByRole('button', { name: 'Prepare download' })).toBeDisabled();
    expect(screen.getByRole('link', { name: 'Sign in' })).toHaveAttribute('href', '/proxy/auth/login');
    expect(mocks.request).not.toHaveBeenCalled(); fireEvent.click(screen.getByRole('button', { name: 'Recheck session' })); expect(auth.refreshSession).toHaveBeenCalledTimes(2);
  });
  it('authorizes the second Linux row by delivery ID, not its ArtifactID or the first platform row', async () => {
    const second = { ...downloadOptions[0]!, id: 13, artifact_id: 900, release_version: '2.3.0' };
    mocks.request.mockResolvedValue({ ...second, artifact_url: '/authorized/second' });
    mount([...downloadOptions, second]); select('2');
    fireEvent.click(screen.getByRole('button', { name: 'Prepare download' }));
    expect(mocks.request).toHaveBeenCalledExactlyOnceWith('example-app', 'linux', undefined, { assetId: 13 });
    expect(await screen.findByRole('link', { name: 'Download file' })).toHaveAttribute('href', '/proxy/authorized/second');
  });
  it.each([undefined, 0, -1, 1.5, Number.MAX_SAFE_INTEGER + 1, NaN, Infinity])('fails closed for missing/invalid delivery row ID %s', id => {
    mount([{ ...downloadOptions[0]!, id, artifact_id: 900 }]); select();
    expect(screen.getByText('This release has no valid download identifier. Reload the page before downloading.')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Prepare download' })).toBeDisabled(); expect(mocks.request).not.toHaveBeenCalled();
  });
  it('fails closed when two catalog rows reuse one delivery ID', () => {
    mount([...downloadOptions, { ...downloadOptions[0]!, release_version: '2.3.0' }]); select('2');
    expect(screen.getByText('This release has a duplicate download identifier. Reload the page before downloading.')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Prepare download' })).toBeDisabled(); expect(mocks.request).not.toHaveBeenCalled();
  });
  it.each(['first-row', 'stale-version', 'owner-stale'])('denies %s after selecting the second Linux row', async kind => {
    const second = { ...downloadOptions[0]!, id: 13, release_version: '2.3.0' };
    if (kind === 'owner-stale') mocks.request.mockRejectedValue(new ConnectError('PRIVATE STALE ROW', Code.FailedPrecondition));
    else mocks.request.mockResolvedValue({ ...(kind === 'first-row' ? downloadOptions[0]! : second), release_version: '2.4.1', artifact_url: '/never-open-wrong' });
    mount([...downloadOptions, second]); select('2'); fireEvent.click(screen.getByRole('button', { name: 'Prepare download' }));
    await waitFor(() => { expect(screen.getByRole('button', { name: 'Prepare download' })).toBeEnabled(); });
    expect(mocks.request).toHaveBeenCalledExactlyOnceWith('example-app', 'linux', undefined, { assetId: 13 });
    expect(screen.queryByRole('link', { name: 'Download file' })).not.toBeInTheDocument();
    expect(document.body).not.toHaveTextContent('PRIVATE STALE ROW');
  });
  it.each(['bundle', 'app', 'version', 'id', 'checksum', 'unsafe', 'self-loop'])('refuses %s authorization mismatch instead of opening a catalog URL', async kind => {
    const asset = { ...downloadOptions[0]!, artifact_url: '/authorized' };
    if (kind === 'bundle') asset.bundle_key = 'private-bundle';
    if (kind === 'app') asset.app_key = 'private'; if (kind === 'version') asset.release_version = '9'; if (kind === 'id') asset.id = 90; if (kind === 'checksum') asset.checksum = 'b';
    if (kind === 'unsafe') asset.artifact_url = 'javascript:alert(1)'; if (kind === 'self-loop') asset.artifact_url = '/apps/example/download';
    mocks.request.mockResolvedValue(asset); mount(); select(); fireEvent.click(screen.getByRole('button', { name: 'Prepare download' }));
    await waitFor(() => { expect(screen.getByRole('button', { name: 'Prepare download' })).toBeEnabled(); }); expect(screen.queryByRole('link', { name: 'Download file' })).not.toBeInTheDocument();
  });
  it.each([Code.PermissionDenied, Code.Unauthenticated, Code.Unavailable])('sanitizes owner failure %s and keeps download unavailable', async code => {
    mocks.request.mockRejectedValue(new ConnectError('PRIVATE-OWNER-DETAILS', code)); mount(); select(); fireEvent.click(screen.getByRole('button', { name: 'Prepare download' }));
    await waitFor(() => { expect(screen.getByRole('button', { name: 'Prepare download' })).toBeEnabled(); });
    expect(document.body).not.toHaveTextContent('PRIVATE-OWNER-DETAILS'); expect(screen.queryByRole('link', { name: 'Download file' })).not.toBeInTheDocument();
  });
  it('discards a late authorization when the selected release changes', async () => {
    let finish: ((asset: DownloadAsset) => void) | undefined;
    mocks.request.mockImplementation(() => new Promise(resolve => { finish = resolve; }));
    mount(); select(); fireEvent.click(screen.getByRole('button', { name: 'Prepare download' })); select('1');
    await act(async () => { finish?.({ ...downloadOptions[0]!, artifact_url: '/authorized-old' }); await Promise.resolve(); });
    expect(screen.queryByRole('link', { name: 'Download file' })).not.toBeInTheDocument();
  });
  it('keeps unknown/private app responses not-found and empty scoped catalogs unavailable', () => {
    const view = mount([], auth, { notFound: true }); expect(screen.getByRole('heading', { name: 'Page not found' })).toBeInTheDocument(); view.unmount();
    mount([]); expect(screen.getByText('No installers are currently available for this app.')).toBeInTheDocument(); expect(mocks.request).not.toHaveBeenCalled();
  });
  it('renders an effect-free chooser preview without auth/providers or transaction handlers', () => {
    render(<DownloadChooser title="Configured preview" description="Configured description" options={downloadOptions} selected="0" onSelect={vi.fn()} state={{ status: 'idle' }} unavailableReason="Configured unavailable" />);
    expect(screen.getByRole('button', { name: 'Prepare download' })).toBeDisabled(); expect(mocks.request).not.toHaveBeenCalled(); expect(auth.refreshSession).not.toHaveBeenCalled();
  });
});
