import { afterEach, describe, expect, it, vi } from 'vitest';
import { act, cleanup, fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '@vrooli/api-base/testing';
import { create } from '@bufbuild/protobuf';
import { Code, ConnectError } from '@connectrpc/connect';
import { PreviewPresentationResponseSchema } from '@vrooli/proto-types/landing-page-business-suite/v1/product_presentation_pb';
import { AdminAuthContext } from '../../../app/providers/AdminAuthContext';
import { PresentationEditorRoute } from './PresentationEditorRoute';
import { admin, clientFixture, previewFixture } from './testFixtures';

afterEach(() => { cleanup(); vi.useRealTimers(); vi.restoreAllMocks(); });
const flush = async (ms = 0) => { await act(async () => { await vi.advanceTimersByTimeAsync(ms); }); };
const source = () => screen.getByRole<HTMLTextAreaElement>('textbox', { name: 'Complete document JSON' });
function result(title = 'Live private preview', route = '/', locale = 'fr', revision = 'a'.repeat(64)) {
  const p = previewFixture(); if (!p.diagnostics || !p.page) throw new Error('fixture');
  Object.assign(p.diagnostics, { requestedRevision: revision, resolvedRevision: revision, requestedVariant: 'control', resolvedVariant: 'control', requestedRoute: route, resolvedRoute: route, locale });
  p.page.locale = locale;
  const c = p.page.blocks[0]?.content?.value; if (c?.case !== 'closingAction') throw new Error('fixture'); c.value.heading = title;
  return create(PreviewPresentationResponseSchema, { presentation: p });
}
async function mount(client = clientFixture()) {
  vi.useFakeTimers();
  const view = render(<AdminAuthContext.Provider value={admin}><PresentationEditorRoute variantSlug="control" client={client} /></AdminAuthContext.Provider>);
  await flush(); return { client, ...view };
}
function noWrites(client: ReturnType<typeof clientFixture>) {
  expect(client.saveDraft).not.toHaveBeenCalled(); expect(client.publish).not.toHaveBeenCalled(); expect(client.rollback).not.toHaveBeenCalled();
}
describe('mounted unsaved private preview', () => {
  it('sends the complete local document after 300ms, renders it beside editing, and never saves automatically', async () => {
    const storage = vi.spyOn(Storage.prototype, 'setItem');
    const client = clientFixture();
    client.preview.mockImplementation(request => Promise.resolve(result(request.document?.pages?.[0]?.title)));
    const view = await mount(client);
    await flush(300); expect(screen.getByRole('heading', { name: 'Configured first page' })).toBeVisible();
    const before = source().value;
    fireEvent.change(source(), { target: { value: before.replace('Configured first page', 'Unsaved local title') } });
    expect(screen.queryByRole('heading', { name: 'Configured first page' })).toBeNull();
    await flush(299); expect(client.preview).toHaveBeenCalledTimes(1);
    await flush(1); expect(client.preview).toHaveBeenCalledTimes(2);
    expect(client.preview.mock.calls[1]?.[0].revision).toBeUndefined();
    expect(client.preview.mock.calls[1]?.[0].document?.pages?.[0]?.title).toBe('Unsaved local title');
    expect(screen.getByRole('heading', { name: 'Unsaved local title' })).toBeVisible();
    expect(screen.getByRole('region', { name: 'Unsaved document preview' })).toHaveTextContent('Document identity (not saved)');
    expect(view.container.querySelector('.presentation-page')).toHaveAttribute('data-presentation-preview', 'true');
    expect(view.container.querySelector('.presentation-editing-grid')?.children).toHaveLength(2);
    expect(screen.getByRole('button', { name: 'Publish draft…' })).toBeDisabled();
    expect(source().value).toContain('Unsaved local title'); noWrites(client); expect(storage).not.toHaveBeenCalled();
  });
  it('hides obsolete previews immediately for syntax errors and owner validation errors while retaining edits', async () => {
    const { client } = await mount(); await flush(300);
    expect(screen.getByRole('region', { name: 'Unsaved document preview' })).toBeVisible();
    const valid = source().value;
    fireEvent.change(source(), { target: { value: '{invalid' } });
    expect(screen.queryByRole('region', { name: 'Unsaved document preview' })).toBeNull();
    await flush(500); expect(client.preview).toHaveBeenCalledTimes(1); expect(source()).toHaveValue('{invalid');
    client.preview.mockRejectedValue(new ConnectError('pages[0].blocks: unknown reference', Code.InvalidArgument));
    fireEvent.change(source(), { target: { value: valid.replace('Configured first page', 'Keep my rejected edit') } });
    await flush(300);
    expect(screen.getByText('pages[0].blocks: unknown reference')).toBeVisible();
    expect(screen.queryByRole('region', { name: 'Unsaved document preview' })).toBeNull();
    expect(source().value).toContain('Keep my rejected edit'); noWrites(client);
    expect(screen.getByRole('button', { name: 'Save draft' })).toBeEnabled();
  });
  it('aborts superseded route/locale requests and ignores responses returned out of order', async () => {
    const client = clientFixture(); const finish: ((value: ReturnType<typeof result>) => void)[] = [];
    client.preview.mockImplementation(() => new Promise(resolve => { finish.push(resolve); }));
    await mount(client); await flush(300);
    fireEvent.change(screen.getByRole('textbox', { name: 'Page route' }), { target: { value: '/apps/example' } });
    await flush(300);
    fireEvent.change(screen.getByRole('textbox', { name: 'Locale' }), { target: { value: 'en' } });
    await flush(300);
    expect(client.preview.mock.calls[0]?.[1]?.signal?.aborted).toBe(true); expect(client.preview.mock.calls[1]?.[1]?.signal?.aborted).toBe(true);
    await act(async () => { finish[2]?.(result('Latest requested page', '/apps/example', 'en')); await Promise.resolve(); });
    expect(screen.getByRole('heading', { name: 'Latest requested page' })).toBeVisible();
    await act(async () => { finish[0]?.(result('Stale root')); finish[1]?.(result('Stale locale', '/apps/example')); await Promise.resolve(); });
    expect(screen.queryByRole('heading', { name: /Stale/ })).toBeNull(); expect(screen.getByRole('heading', { name: 'Latest requested page' })).toBeVisible();
    noWrites(client);
  });
  it('requires explicit retained-revision inspection and preserves dirty source while switching back to local preview', async () => {
    const { client } = await mount();
    fireEvent.change(source(), { target: { value: source().value.replace('Configured first page', 'Keep local editing') } });
    fireEvent.click(screen.getByRole('radio', { name: 'Saved revision' }));
    await flush(500); expect(client.preview).not.toHaveBeenCalled();
    fireEvent.change(screen.getByRole('combobox', { name: 'Revision to inspect' }), { target: { value: 'older-revision' } });
    client.preview.mockResolvedValueOnce(result('Retained inspection', '/', 'fr', 'older-revision'));
    fireEvent.click(screen.getByRole('button', { name: 'Preview saved revision' })); await flush();
    expect(client.preview.mock.calls[0]?.[0]).toEqual({ variantSlug: 'control', revision: 'older-revision', route: '/', locale: '' });
    expect(screen.getByRole('region', { name: 'Private revision preview' })).toBeVisible();
    expect(source().value).toContain('Keep local editing'); noWrites(client);
    fireEvent.click(screen.getByRole('radio', { name: 'Local document (read-only)' }));
    expect(screen.queryByRole('region', { name: 'Private revision preview' })).toBeNull();
    await flush(300); expect(client.preview.mock.calls[1]?.[0].document).toBeDefined(); noWrites(client);
  });
  it('does not lock typing while preview resolves, and auth loss aborts the private read', async () => {
    const client = clientFixture(); client.preview.mockImplementation(() => new Promise(() => {}));
    const view = await mount(client); await flush(300);
    expect(source()).toBeEnabled();
    fireEvent.change(source(), { target: { value: source().value.replace('Configured first page', 'Still editable') } });
    expect(client.preview.mock.calls[0]?.[1]?.signal?.aborted).toBe(true);
    await flush(300);
    view.rerender(<AdminAuthContext.Provider value={{ ...admin, isAuthenticated: false, user: null }}><PresentationEditorRoute variantSlug="control" client={client} /></AdminAuthContext.Provider>);
    expect(client.preview.mock.calls[1]?.[1]?.signal?.aborted).toBe(true);
    expect(screen.queryByRole('textbox')).toBeNull(); noWrites(client);
  });
  it('hides the editor when ephemeral preview authorization expires', async () => {
    const client = clientFixture(); client.preview.mockRejectedValue(new ConnectError('expired', Code.Unauthenticated));
    await mount(client); await flush(300);
    expect(screen.getByRole('alert')).toHaveTextContent('Administrator access is required');
    expect(screen.queryByRole('textbox')).toBeNull(); expect(document.body).not.toHaveTextContent('PRIVATE-REFERENCE'); noWrites(client);
  });
});
