import { Suspense } from 'react';
import { BrowserRouter, MemoryRouter, Route, Routes } from 'react-router-dom';
import { act, cleanup, fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '@vrooli/api-base/testing';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { create } from '@bufbuild/protobuf';
import { PreviewPresentationResponseSchema } from '@vrooli/proto-types/landing-page-business-suite/v1/product_presentation_pb';
import { AdminAuthContext } from '../../../app/providers/AdminAuthContext';
import { LandingVariantProvider } from '../../../app/providers/LandingVariantProvider';
import { adminRoutes } from '../../../app/routes/adminRoutes';
import { admin, previewFixture, snapshot } from './testFixtures';
const mocks = vi.hoisted(() => ({ list: vi.fn(), get: vi.fn(), save: vi.fn(), publish: vi.fn(), rollback: vi.fn(), preview: vi.fn(), public: vi.fn(), metric: vi.fn(), logout: vi.fn() }));
vi.unmock('../../../app/providers/useLandingVariant');
vi.mock('../../../shared/api/variants', () => ({ listVariants: mocks.list }));
vi.mock('../../../shared/api/landing', async original => ({ ...await original<typeof import('../../../shared/api/landing')>(), getLandingConfig: mocks.public }));
vi.mock('../../../shared/api/productPresentation', async original => ({ ...await original<typeof import('../../../shared/api/productPresentation')>(), productPresentationClient: {
  getPresentation: mocks.get, saveDraft: mocks.save, publish: mocks.publish, rollback: mocks.rollback, preview: mocks.preview,
} }));
vi.mock('../../../shared/api', async original => ({ ...await original<typeof import('../../../shared/api')>(), trackMetric: mocks.metric, adminLogout: mocks.logout }));
beforeEach(() => {
  vi.clearAllMocks();
  mocks.list.mockResolvedValue({ variants: [{ slug: 'control', name: 'Control' }, { slug: 'second', name: 'Second' }] });
  mocks.get.mockImplementation(({ variantSlug }: { variantSlug: string }) => Promise.resolve({ ...snapshot(), variantSlug }));
  mocks.preview.mockResolvedValue(create(PreviewPresentationResponseSchema, { presentation: previewFixture() }));
  mocks.logout.mockResolvedValue(undefined);
});
afterEach(() => { cleanup(); vi.restoreAllMocks(); });
function Tree({ authenticated = true }: { authenticated?: boolean }) {
  return <AdminAuthContext.Provider value={{ ...admin, isAuthenticated: authenticated }}><LandingVariantProvider><Suspense><Routes>{adminRoutes}<Route path="/admin/login" element={<p>Login destination</p>} /></Routes></Suspense></LandingVariantProvider></AdminAuthContext.Provider>;
}
function mount(route = '/admin/presentation/control', authenticated = true) {
  return render(<MemoryRouter initialEntries={[route]}><Tree authenticated={authenticated} /></MemoryRouter>, { withoutRouter: true });
}
const source = () => screen.findByRole('textbox', { name: 'Complete document JSON' });
describe('mounted authenticated presentation editor', () => {
  it('does not fetch a guessed document when the private variant catalog fails', async () => {
    mocks.list.mockRejectedValue(new Error('Catalog unavailable'));
    mount();
    expect(await screen.findByRole('alert')).toHaveTextContent('Variants could not be loaded');
    expect(screen.getByRole('combobox', { name: 'Presentation variant' })).toBeDisabled();
    expect(screen.queryByRole('textbox')).toBeNull();
    expect(mocks.get).not.toHaveBeenCalled(); expect(mocks.public).not.toHaveBeenCalled();
  });
  it('keeps an unknown selected variant unavailable instead of silently choosing the first variant', async () => {
    mount('/admin/presentation/removed');
    expect(await screen.findByRole('alert')).toHaveTextContent('selected variant is unavailable');
    expect(screen.getByRole('combobox', { name: 'Presentation variant' })).toHaveValue('');
    expect(mocks.get).not.toHaveBeenCalled();
    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'second' } });
    await source(); expect(mocks.get).toHaveBeenCalledWith({ variantSlug: 'second' }, expect.anything());
  });
  it('uses the owner slug when no variant display name exists and lets the operator clear the selection', async () => {
    mocks.list.mockResolvedValue({ variants: [{ slug: 'control', name: '' }] });
    mount(); await source();
    expect(screen.getByRole('option', { name: 'control' })).toHaveValue('control');
    fireEvent.change(screen.getByRole('combobox', { name: 'Presentation variant' }), { target: { value: '' } });
    await waitFor(() => { expect(screen.queryByRole('textbox')).toBeNull(); });
    expect(screen.getByRole('combobox')).toHaveValue('');
    expect(mocks.get).toHaveBeenCalledTimes(1);
  });
  it('requires confirmation before logging out with unsaved document edits', async () => {
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(false);
    mount(); const input = await source(); fireEvent.change(input, { target: { value: 'Unsaved private document' } });
    fireEvent.click(screen.getByTestId('nav-logout'));
    expect(confirm).toHaveBeenCalledOnce(); expect(mocks.logout).not.toHaveBeenCalled();
    expect(input).toHaveValue('Unsaved private document');
    confirm.mockReturnValue(true); fireEvent.click(screen.getByTestId('nav-logout'));
    expect(await screen.findByText('Login destination')).toBeInTheDocument();
    expect(mocks.logout).toHaveBeenCalledOnce();
    expect(document.body).not.toHaveTextContent('Unsaved private document');
    expect(mocks.save).not.toHaveBeenCalled(); expect(mocks.publish).not.toHaveBeenCalled();
  });
  it.each(['resolve', 'reject'] as const)('ignores a catalog request that completes with %s after authentication is lost', async outcome => {
    let finish: (() => void) | undefined;
    mocks.list.mockImplementationOnce(() => new Promise((resolve, reject) => { finish = () => { if (outcome === 'resolve') resolve({ variants: [{ slug: 'control' }] }); else reject(new Error('Late failure')); }; }));
    const view = mount();
    await screen.findByText('Loading variants…');
    view.rerender(<MemoryRouter initialEntries={['/admin/presentation/control']}><Tree authenticated={false} /></MemoryRouter>);
    await screen.findByText('Login destination');
    await act(async () => { finish?.(); await Promise.resolve(); });
    expect(screen.queryByRole('textbox')).toBeNull(); expect(screen.queryByText('Variants could not be loaded. Reload to try again.')).toBeNull();
    expect(mocks.get).not.toHaveBeenCalled(); expect(mocks.public).not.toHaveBeenCalled();
  });
  it.each(['/admin/presentation/control', '/admin/customization/variants/control/sections/42'])('requires authentication before fetching configuration on %s', async path => {
    mount(path, false);
    expect(await screen.findByText('Login destination')).toBeInTheDocument();
    expect(mocks.get).not.toHaveBeenCalled(); expect(mocks.list).not.toHaveBeenCalled(); expect(mocks.public).not.toHaveBeenCalled();
  });
  it('mounts the existing layout and explicit variant selection without public assignment or preview metrics', async () => {
    const storage = vi.spyOn(Storage.prototype, 'setItem');
    mount(); await source();
    expect(screen.getByRole('link', { name: 'Landing Page Business Suite' })).toHaveAttribute('href', '/admin');
    expect(screen.getByRole('combobox', { name: 'Presentation variant' })).toHaveValue('control');
    fireEvent.click(screen.getByRole('radio', { name: 'Saved revision' }));
    fireEvent.click(screen.getByRole('button', { name: 'Preview saved revision' }));
    expect(await screen.findByRole('heading', { name: 'Configured preview heading' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Configured action' })).toBeDisabled();
    expect(mocks.public).not.toHaveBeenCalled(); expect(mocks.metric).not.toHaveBeenCalled();
    expect(storage.mock.calls.every(([key]) => key === 'landing_admin_experience')).toBe(true);
    expect(document.querySelector('meta[name="robots"]')).toHaveAttribute('content', 'noindex, nofollow');
  });
  it.each(['42', 'new', 'hero'])('migrates the legacy %s bookmark to the same typed variant with private preview and publication controls', async section => {
    mount('/admin/customization/variants/second/sections/' + section); await source();
    expect(mocks.get).toHaveBeenCalledWith({ variantSlug: 'second' }, expect.anything());
    expect(screen.getByRole('combobox', { name: 'Presentation variant' })).toHaveValue('second');
    expect(screen.getByRole('button', { name: 'Publish draft…' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Roll back…' })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('radio', { name: 'Saved revision' }));
    fireEvent.click(screen.getByRole('button', { name: 'Preview saved revision' }));
    expect(await screen.findByRole('heading', { name: 'Configured preview heading' })).toBeInTheDocument();
    expect(mocks.public).not.toHaveBeenCalled(); expect(mocks.metric).not.toHaveBeenCalled();
    expect(mocks.save).not.toHaveBeenCalled(); expect(mocks.publish).not.toHaveBeenCalled(); expect(mocks.rollback).not.toHaveBeenCalled();
  });
  it('requires an explicit variant before mounting the document editor', async () => {
    mount('/admin/presentation');
    await waitFor(() => { expect(screen.getByRole('combobox')).toBeEnabled(); });
    expect(mocks.get).not.toHaveBeenCalled();
    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'second' } });
    await source(); expect(mocks.get).toHaveBeenCalledWith({ variantSlug: 'second' }, expect.anything());
  });
  it('guards dirty variant and admin navigation; confirmed navigation loads the chosen document', async () => {
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(false);
    mount(); const input = await source(); fireEvent.change(input, { target: { value: 'unsaved text' } });
    fireEvent.change(screen.getByRole('combobox', { name: 'Presentation variant' }), { target: { value: 'second' } });
    expect(confirm).toHaveBeenCalledOnce(); expect(input).toHaveValue('unsaved text');
    expect(mocks.get).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByRole('link', { name: 'Landing Page Business Suite' }));
    expect(confirm).toHaveBeenCalledTimes(2); expect(input).toHaveValue('unsaved text');
    confirm.mockReturnValue(true);
    fireEvent.change(screen.getByRole('combobox', { name: 'Presentation variant' }), { target: { value: 'second' } });
    await waitFor(() => { expect(mocks.get).toHaveBeenCalledTimes(2); });
    expect(screen.getByRole('combobox', { name: 'Presentation variant' })).toHaveValue('second');
  });
  it('restores the BrowserRouter history entry when dirty back navigation is declined', async () => {
    window.history.replaceState({ idx: 0 }, '', '/admin/presentation');
    window.history.pushState({ idx: 1 }, '', '/admin/presentation/control');
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(false);
    render(<BrowserRouter><Tree /></BrowserRouter>, { withoutRouter: true });
    const input = await source(); fireEvent.change(input, { target: { value: 'unsaved text' } });
    act(() => { window.history.back(); });
    await waitFor(() => { expect(confirm).toHaveBeenCalledOnce(); expect(window.location.pathname).toBe('/admin/presentation/control'); });
    expect(input).toHaveValue('unsaved text'); expect(mocks.get).toHaveBeenCalledTimes(1);
  });
});
