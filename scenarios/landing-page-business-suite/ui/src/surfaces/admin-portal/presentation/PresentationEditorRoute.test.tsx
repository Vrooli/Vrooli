import { afterEach, describe, expect, it, vi } from 'vitest';
import { act, cleanup, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { renderWithProviders as render } from '@vrooli/api-base/testing';
import { Code, ConnectError } from '@connectrpc/connect';
import { AdminAuthContext } from '../../../app/providers/AdminAuthContext';
import { formatPresentationDocument, parsePresentationDocument } from '../../../shared/api/productPresentation';
import * as legacyAdapter from '../../../shared/lib/presentationLegacyImport';
import { PresentationEditorRoute, type PresentationEditorRouteProps } from './PresentationEditorRoute';
import { admin, clientFixture, snapshot } from './testFixtures';

afterEach(() => { cleanup(); vi.restoreAllMocks(); });
function mount(props: Partial<PresentationEditorRouteProps> = {}, auth = admin) {
  const client = props.client ?? clientFixture();
  const element = <AdminAuthContext.Provider value={auth}><PresentationEditorRoute variantSlug="control" {...props} client={client} /></AdminAuthContext.Provider>;
  return { client, ...render(element) };
}
async function source() { return screen.findByRole<HTMLTextAreaElement>('textbox', { name: 'Complete document JSON' }); }
async function edit() {
  const input = await source();
  fireEvent.change(input, { target: { value: input.value.replace('Configured first page', 'Changed first page') } });
  return input;
}

describe('authenticated presentation editor', () => {
  it('imports into the current dirty private draft through the real local adapter without save, publish or rollback', async () => {
    const client = clientFixture(); const response = snapshot();
    const app = response.document?.apps[0]; if (!app) throw new Error('Fixture app missing');
    app.enabled = false; app.visibility = 'private'; app.publication = 'draft'; app.pageId = 'z-page';
    client.getPresentation.mockResolvedValue(response); mount({ client }); const input = await edit();
    const before = parsePresentationDocument(input.value);
    fireEvent.change(screen.getByRole('combobox', { name: 'Import target app' }), { target: { value: app.key } });
    fireEvent.change(screen.getByRole('combobox', { name: 'Import target locale' }), { target: { value: 'fr' } });
    fireEvent.change(screen.getByRole('textbox', { name: 'Legacy snapshot JSON' }), { target: { value: JSON.stringify({ variant: {}, sections: [{ key: 'imported', section_type: 'faq', content: { title: 'Recovered questions', items: [{ question: 'Recovered question?', answer: 'Recovered answer.' }] } }] }) } });
    fireEvent.click(screen.getByRole('button', { name: 'Import into local document' }));
    await screen.findByRole('region', { name: 'Local import receipt' });
    const after = parsePresentationDocument(input.value);
    expect(after.pages[0]?.title).toBe('Changed first page');
    expect(after.pages[0]?.blocks.slice(0, 2)).toEqual(before.pages[0]?.blocks);
    expect(after.pages[0]?.blocks).toHaveLength(3); expect(after.pages[1]).toEqual(before.pages[1]);
    expect(after.apps).toEqual(before.apps); expect(after.bundle).toEqual(before.bundle);
    expect(screen.getByRole('button', { name: 'Save draft' })).toBeEnabled();
    expect(screen.getByRole('button', { name: 'Publish draft…' })).toBeDisabled();
    expect(client.saveDraft).not.toHaveBeenCalled(); expect(client.publish).not.toHaveBeenCalled(); expect(client.rollback).not.toHaveBeenCalled();
  });
  it.each(['logout', 'route', 'document'] as const)('does not apply a late local import after mounted editor %s changes', async changed => {
    const client = clientFixture(); const response = snapshot(); const document = response.document;
    const app = document?.apps[0]; if (!document || !app) throw new Error('Fixture app missing');
    app.enabled = false; app.visibility = 'private'; app.publication = 'draft'; app.pageId = 'z-page';
    client.getPresentation.mockResolvedValue(response);
    const value = await legacyAdapter.importLegacyPresentationSnapshot(document, '{"variant":{},"sections":[]}', { adapterVersion: 1, appKey: app.key, locale: 'fr' });
    let finish: ((result: legacyAdapter.LegacyImportResult) => void) | undefined;
    vi.spyOn(legacyAdapter, 'importLegacyPresentationSnapshot').mockImplementation(() => new Promise(resolve => { finish = resolve; }));
    const view = mount({ client }); const input = await source();
    fireEvent.change(screen.getByRole('combobox', { name: 'Import target app' }), { target: { value: app.key } });
    fireEvent.change(screen.getByRole('combobox', { name: 'Import target locale' }), { target: { value: 'fr' } });
    fireEvent.change(screen.getByRole('textbox', { name: 'Legacy snapshot JSON' }), { target: { value: '{"variant":{},"sections":[]}' } });
    fireEvent.click(screen.getByRole('button', { name: 'Import into local document' }));
    if (changed === 'logout') view.rerender(<AdminAuthContext.Provider value={{ ...admin, isAuthenticated: false, user: null }}><PresentationEditorRoute variantSlug="control" client={client} /></AdminAuthContext.Provider>);
    if (changed === 'route') fireEvent.change(screen.getByRole('textbox', { name: 'Page route' }), { target: { value: '/apps/other' } });
    if (changed === 'document') await edit();
    const expectedText = input.value;
    await act(async () => { finish?.(value); await Promise.resolve(); });
    expect(screen.queryByRole('region', { name: 'Local import receipt' })).toBeNull();
    if (changed === 'logout') { expect(screen.queryByRole('textbox')).toBeNull(); expect(screen.getByRole('alert')).toHaveTextContent('Sign in'); }
    else { expect(input).toHaveValue(expectedText); expect(input.value).not.toBe(formatPresentationDocument(value.document)); }
    expect(client.saveDraft).not.toHaveBeenCalled(); expect(client.publish).not.toHaveBeenCalled(); expect(client.rollback).not.toHaveBeenCalled();
  });
  it('does not fetch private configuration while the session is still being checked', () => {
    const client = clientFixture();
    mount({ client }, { ...admin, isSessionLoading: true });
    expect(screen.getByRole('status')).toHaveTextContent('Checking administrator session');
    expect(client.getPresentation).not.toHaveBeenCalled();
    expect(screen.queryByRole('textbox')).toBeNull();
  });
  it('requires an explicit nonblank variant even for an authenticated administrator', () => {
    const client = clientFixture(); mount({ client, variantSlug: '  ' });
    expect(screen.getByRole('alert')).toHaveTextContent('Select an explicit presentation variant');
    expect(client.getPresentation).not.toHaveBeenCalled();
  });
  it.each(['state', 'variant', 'document'] as const)('rejects an incomplete or mis-scoped owner snapshot: %s', async missing => {
    const client = clientFixture(); const response = snapshot();
    if (missing === 'state') response.state = undefined;
    if (missing === 'variant') response.variantSlug = 'another-variant';
    if (missing === 'document') response.document = undefined;
    client.getPresentation.mockResolvedValue(response); mount({ client });
    await screen.findByText('Request not confirmed');
    expect(screen.queryByRole('textbox', { name: 'Complete document JSON' })).toBeNull();
    expect(document.body).not.toHaveTextContent('PRIVATE-REFERENCE');
    expect(client.saveDraft).not.toHaveBeenCalled(); expect(client.publish).not.toHaveBeenCalled();
  });
  it('edits the explicitly selected page title and description without changing other pages or saving', async () => {
    const client = clientFixture(); mount({ client }); const input = await source();
    const original = parsePresentationDocument(input.value);
    fireEvent.click(screen.getByRole('button', { name: 'a-page · en' }));
    fireEvent.change(screen.getByRole('textbox', { name: 'Page title' }), { target: { value: 'Localized second title' } });
    fireEvent.change(screen.getByRole('textbox', { name: 'Page description' }), { target: { value: 'Localized second description' } });
    const changed = parsePresentationDocument(input.value);
    expect(changed.pages.map(page => page.id)).toEqual(['z-page', 'a-page']);
    expect(changed.pages[0]).toEqual(original.pages[0]);
    expect(changed.pages[1]).toEqual({ ...original.pages[1], title: 'Localized second title', description: 'Localized second description' });
    expect(changed.bundle).toEqual(original.bundle); expect(changed.strings).toEqual(original.strings);
    expect(client.saveDraft).not.toHaveBeenCalled(); expect(client.publish).not.toHaveBeenCalled();
  });
  it.each(['publish', 'rollback', 'reload'] as const)('Escape dismisses %s confirmation without mutation or lost edits', async action => {
    const client = clientFixture(); mount({ client }); const input = await source();
    if (action === 'reload') {
      await edit(); fireEvent.click(screen.getByRole('button', { name: 'Reload draft' }));
    } else if (action === 'rollback') {
      fireEvent.change(screen.getByRole('combobox', { name: 'Retained published revision' }), { target: { value: 'older-revision' } });
      fireEvent.click(screen.getByRole('button', { name: 'Roll back…' }));
    } else fireEvent.click(screen.getByRole('button', { name: 'Publish draft…' }));
    const text = input.value;
    fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Escape' });
    await waitFor(() => { expect(screen.queryByRole('dialog')).toBeNull(); });
    expect(input).toHaveValue(text); expect(input).toBeEnabled();
    expect(client.getPresentation).toHaveBeenCalledTimes(1);
    expect(client.publish).not.toHaveBeenCalled(); expect(client.rollback).not.toHaveBeenCalled(); expect(client.saveDraft).not.toHaveBeenCalled();
  });
  it('does not fetch or disclose document data before administrator authentication', () => {
    const client = clientFixture();
    mount({ client }, { ...admin, isAuthenticated: false, user: null });
    expect(screen.getByRole('alert')).toHaveTextContent('Sign in');
    expect(client.getPresentation).not.toHaveBeenCalled();
    expect(screen.queryByRole('textbox')).toBeNull();
  });
  it('keeps a complete dirty document and saves only the draft using the exact bigint generation', async () => {
    const client = clientFixture();
    const changed = vi.fn();
    mount({ client, onDirtyChange: changed });
    await edit();
    expect(screen.getByRole('status')).toHaveTextContent('Unsaved document changes');
    expect(changed).toHaveBeenLastCalledWith(true);
    expect(screen.getByRole('button', { name: 'Publish draft…' })).toBeDisabled();
    fireEvent.click(screen.getByRole('button', { name: 'Save draft' }));
    await waitFor(() => { expect(client.saveDraft).toHaveBeenCalledTimes(1); });
    const request = client.saveDraft.mock.calls[0]?.[0];
    expect(request?.expectedGeneration).toBe(9007199254740993n);
    expect(request?.document?.pages?.[0]?.title).toBe('Changed first page');
    expect(request?.document?.pages?.[0]?.display?.shell?.brandName).toBe('Configured brand');
    expect(request?.document?.bundle?.appOrder).toEqual(['second-app', 'first-app']);
    expect(request?.document?.apps?.[0]?.preservationRef).toBe('PRIVATE-REFERENCE');
    expect(request?.document?.strings?.fr?.values).toEqual({ 'custom.copy': 'Configured localized copy' });
    expect(client.publish).not.toHaveBeenCalled();
    expect(client.rollback).not.toHaveBeenCalled();
    await screen.findByText('Draft saved. The active publication was not changed.');
  });
  it('retains invalid JSON and rejects unknown generated fields without a server mutation', async () => {
    const client = clientFixture(); mount({ client });
    const input = await source();
    fireEvent.change(input, { target: { value: '{"unknown_field":true}' } });
    expect(input).toHaveValue('{"unknown_field":true}');
    expect(input).toHaveAttribute('aria-invalid', 'true');
    expect(screen.getByRole('button', { name: 'Save draft' })).toBeDisabled();
    expect(client.saveDraft).not.toHaveBeenCalled();
  });
  it('surfaces server validation and retains the edited source for correction', async () => {
    const client = clientFixture();
    client.saveDraft.mockRejectedValue(new ConnectError('pages[0].blocks[0]: unknown capability reference', Code.InvalidArgument));
    mount({ client }); await edit();
    fireEvent.click(screen.getByRole('button', { name: 'Save draft' }));
    await screen.findByText('Server validation rejected the request');
    expect(screen.getByRole('alert')).toHaveTextContent('unknown capability reference');
    expect((await source()).value).toContain('Changed first page');
    expect(client.publish).not.toHaveBeenCalled();
  });
  it('locks stale CAS writes, keeps local changes, and requires confirmed reload', async () => {
    const client = clientFixture(); client.saveDraft.mockRejectedValue(new ConnectError('stale generation', Code.Aborted));
    mount({ client }); await edit();
    fireEvent.click(screen.getByRole('button', { name: 'Save draft' }));
    await screen.findByText('Generation conflict');
    expect((await source()).value).toContain('Changed first page');
    expect(screen.getByRole('button', { name: 'Save draft' })).toBeDisabled();
    fireEvent.click(screen.getByRole('button', { name: 'Reload draft' }));
    expect(screen.getByRole('dialog')).toHaveTextContent('Discard local edits');
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(client.getPresentation).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByRole('button', { name: 'Reload draft' }));
    client.getPresentation.mockResolvedValue(snapshot(9007199254740999n));
    fireEvent.click(screen.getByRole('button', { name: 'Discard and reload' }));
    await screen.findByText('Draft reloaded.');
    expect((await source()).value).toContain('Configured first page');
    expect(client.getPresentation).toHaveBeenCalledTimes(2);
    expect(screen.queryByText('Generation conflict')).toBeNull();
  });
  it('publishes only after explicit confirmation of revision and generation', async () => {
    const client = clientFixture(); mount({ client }); await source();
    fireEvent.click(screen.getByRole('button', { name: 'Publish draft…' }));
    expect(client.publish).not.toHaveBeenCalled();
    expect(screen.getByRole('dialog')).toHaveTextContent('9007199254740993');
    fireEvent.click(screen.getByRole('button', { name: 'Confirm publish' }));
    await waitFor(() => { expect(client.publish).toHaveBeenCalledWith({ variantSlug: 'control', revision: 'draft-revision', expectedGeneration: 9007199254740993n }, expect.anything()); });
    expect(client.publish.mock.calls[0]?.[1]?.signal).toBeInstanceOf(AbortSignal);
    expect(client.saveDraft).not.toHaveBeenCalled();
  });
  it('rolls back only an explicitly selected retained revision, preserving server order', async () => {
    const client = clientFixture(); mount({ client }); await source();
    const select = screen.getByRole('combobox', { name: 'Retained published revision' });
    expect(within(select).getAllByRole('option').map(option => option.textContent)).toEqual(['Choose a revision', 'older-revision', 'active-revision (active)']);
    fireEvent.change(select, { target: { value: 'older-revision' } });
    fireEvent.click(screen.getByRole('button', { name: 'Roll back…' }));
    expect(client.rollback).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: 'Confirm rollback' }));
    await waitFor(() => { expect(client.rollback).toHaveBeenCalledWith({ variantSlug: 'control', revision: 'older-revision', expectedGeneration: 9007199254740993n }, expect.anything()); });
  });
  it('reorders only the explicitly moved pages and blocks', async () => {
    mount(); const input = await source();
    fireEvent.click(screen.getByRole('button', { name: 'Move block a-block up' }));
    expect(parsePresentationDocument(input.value).pages[0]?.blocks.map(block => block.id)).toEqual(['a-block', 'z-block']);
    fireEvent.click(screen.getByRole('button', { name: 'Move page a-page en up' }));
    const document = parsePresentationDocument(input.value);
    expect(document.pages.map(page => page.id)).toEqual(['a-page', 'z-page']);
    expect(document.bundle?.appOrder).toEqual(['second-app', 'first-app']);
  });
  it('previews the authorized saved revision without public navigation or demo substitution', async () => {
    const client = clientFixture(); const storage = vi.spyOn(Storage.prototype, 'setItem');
    mount({ client }); await source();
    fireEvent.click(screen.getByRole('radio', { name: 'Saved revision' }));
    fireEvent.change(screen.getByRole('textbox', { name: 'Page route' }), { target: { value: '/apps/configured-slug' } });
    fireEvent.click(screen.getByRole('button', { name: 'Preview saved revision' }));
    await screen.findByRole('heading', { name: 'Configured preview heading' });
    expect(client.preview).toHaveBeenCalledWith({ variantSlug: 'control', revision: 'draft-revision', route: '/apps/configured-slug', locale: '' }, expect.anything());
    expect(storage).not.toHaveBeenCalled();
    expect(document.head.querySelector('meta[name="robots"]')).toHaveAttribute('content', 'noindex, nofollow');
    expect(screen.queryByText(/All your agents|One clear view/)).toBeNull();
  });
  it('rejects a preview response without privacy diagnostics', async () => {
    const client = clientFixture();
    client.preview.mockResolvedValue({ $typeName: 'landing_page_business_suite.v1.PreviewPresentationResponse', presentation: undefined });
    mount({ client }); await source();
    fireEvent.click(screen.getByRole('radio', { name: 'Saved revision' }));
    fireEvent.click(screen.getByRole('button', { name: 'Preview saved revision' }));
    await screen.findByText('Request not confirmed');
    expect(screen.queryByRole('region', { name: 'Private revision preview' })).toBeNull();
  });
  it('hides private source when the server rejects authorization', async () => {
    const client = clientFixture(); client.saveDraft.mockRejectedValue(new ConnectError('expired', Code.Unauthenticated));
    mount({ client }); await edit(); fireEvent.click(screen.getByRole('button', { name: 'Save draft' }));
    await screen.findByText(/Private editor data has been hidden/);
    expect(screen.queryByRole('textbox')).toBeNull();
    expect(document.body).not.toHaveTextContent('PRIVATE-REFERENCE');
  });
  it('clears private state immediately when the authentication context changes', async () => {
    const client = clientFixture(); const view = mount({ client }); await source();
    view.rerender(<AdminAuthContext.Provider value={{ ...admin, isAuthenticated: false, user: null }}><PresentationEditorRoute variantSlug="control" client={client} /></AdminAuthContext.Provider>);
    expect(screen.queryByRole('textbox')).toBeNull();
    expect(client.getPresentation).toHaveBeenCalledTimes(1);
  });
  it('prevents duplicate saves while a request is in flight', async () => {
    const client = clientFixture(); let finish: ((value: ReturnType<typeof snapshot>) => void) | undefined;
    const pendingResult = new Promise<ReturnType<typeof snapshot>>(resolve => { finish = resolve; });
    client.saveDraft.mockReturnValue(pendingResult);
    mount({ client }); await edit(); const button = screen.getByRole('button', { name: 'Save draft' });
    fireEvent.click(button); fireEvent.click(button);
    expect(client.saveDraft).toHaveBeenCalledTimes(1);
    expect(await source()).toBeDisabled();
    await act(async () => { finish?.(snapshot(9007199254740994n)); await pendingResult; });
  });
  it('blocks another publish after a CAS conflict until the operator reloads', async () => {
    const client = clientFixture(); client.publish.mockRejectedValue(new ConnectError('another editor published', Code.Aborted));
    mount({ client }); await source();
    fireEvent.click(screen.getByRole('button', { name: 'Publish draft…' }));
    fireEvent.click(screen.getByRole('button', { name: 'Confirm publish' }));
    await screen.findByText('Generation conflict');
    expect(screen.getByRole('button', { name: 'Publish draft…' })).toBeDisabled();
    expect(client.getPresentation).toHaveBeenCalledTimes(1);
    expect(client.publish).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByRole('button', { name: 'Reload draft' }));
    await screen.findByText('Draft reloaded.');
    expect(screen.getByRole('button', { name: 'Publish draft…' })).toBeEnabled();
  });
  it('does not repeat a write when its network outcome is unknown', async () => {
    const client = clientFixture(); client.saveDraft.mockRejectedValue(new ConnectError('network unavailable', Code.Unavailable));
    mount({ client }); await edit(); fireEvent.click(screen.getByRole('button', { name: 'Save draft' }));
    await screen.findByText('Request not confirmed');
    expect((await source()).value).toContain('Changed first page');
    expect(screen.getByRole('button', { name: 'Save draft' })).toBeDisabled();
    expect(client.saveDraft).toHaveBeenCalledTimes(1);
    expect(client.getPresentation).toHaveBeenCalledTimes(1);
  });
  it('offers an empty schema-shaped document when no draft exists, not a demonstration app', async () => {
    const client = clientFixture(); const response = snapshot(0n);
    response.document = undefined; response.revision = '';
    if (!response.state) throw new Error('Test state missing');
    response.state.draftRevision = ''; response.state.activeRevision = ''; response.state.publishedRevisions = [];
    client.getPresentation.mockResolvedValue(response); mount({ client });
    const draft = parsePresentationDocument((await source()).value);
    expect(draft.schemaVersion).toBe(1); expect(draft.apps).toEqual([]); expect(draft.pages).toEqual([]);
    expect(screen.getByRole('button', { name: 'Publish draft…' })).toBeDisabled();
  });
  it('aborts pending work and ignores a late private response after logout', async () => {
    const client = clientFixture(); let finish: ((value: ReturnType<typeof snapshot>) => void) | undefined;
    const pendingResult = new Promise<ReturnType<typeof snapshot>>(resolve => { finish = resolve; });
    client.getPresentation.mockReturnValue(pendingResult);
    const view = mount({ client });
    await waitFor(() => { expect(client.getPresentation).toHaveBeenCalledTimes(1); });
    const signal = client.getPresentation.mock.calls[0]?.[1]?.signal;
    view.rerender(<AdminAuthContext.Provider value={{ ...admin, isAuthenticated: false, user: null }}><PresentationEditorRoute variantSlug="control" client={client} /></AdminAuthContext.Provider>);
    expect(signal?.aborted).toBe(true);
    await act(async () => { finish?.(snapshot()); await pendingResult; });
    expect(screen.queryByRole('textbox')).toBeNull();
    expect(screen.getByRole('alert')).toHaveTextContent('Sign in');
  });
  it('shows configured application copy without hardcoded public product names', async () => {
    mount(); const text = (await source()).value;
    expect(text).toContain('Configured private app');
    expect(document.body).not.toHaveTextContent(/Aquila|Backdrop Studio|Browser Automation Studio/);
  });
});
