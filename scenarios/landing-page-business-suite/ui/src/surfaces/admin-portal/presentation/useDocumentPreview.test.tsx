// provider-free-exception: this state hook receives its authorized read callback and has no context dependencies; API/auth integration is tested at the editor route.
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, cleanup, renderHook } from '@testing-library/react';
import { Code, ConnectError } from '@connectrpc/connect';
import { useDocumentPreview, type DocumentPreviewRequest } from './useDocumentPreview';
import { documentFixture, previewFixture } from './testFixtures';
import type { ProductPresentationDocument } from '../../../shared/api/productPresentation';

const response = () => {
  const p = previewFixture(); if (!p.diagnostics) throw new Error('fixture');
  Object.assign(p.diagnostics, { requestedRevision: 'a'.repeat(64), resolvedRevision: 'a'.repeat(64), requestedRoute: '/', resolvedRoute: '/', requestedVariant: 'control', resolvedVariant: 'control', locale: 'fr' });
  return p;
};
function input(document: ProductPresentationDocument | undefined = documentFixture()) { return { document, variantSlug: 'control', route: '/', locale: 'fr', enabled: true }; }
const advance = async (ms = 300) => { await act(async () => { await vi.advanceTimersByTimeAsync(ms); }); };
beforeEach(() => { vi.useFakeTimers(); });
afterEach(() => { cleanup(); vi.useRealTimers(); });
describe('read-only unsaved document preview', () => {
  it('debounces the complete unsaved document for exactly 300ms without a persistence operation', async () => {
    const request = vi.fn<DocumentPreviewRequest>().mockResolvedValue(response());
    const source = input();
    const hook = renderHook(() => useDocumentPreview(source, request));
    await advance(299); expect(request).not.toHaveBeenCalled();
    await advance(1); expect(request).toHaveBeenCalledTimes(1);
    expect(request.mock.calls[0]?.[0]).toEqual({ variantSlug: 'control', document: source.document, route: '/', locale: 'fr' });
    expect(hook.result.current.status).toBe('ready');
    expect(hook.result.current.presentation?.diagnostics?.resolvedRevision).toBe('a'.repeat(64));
  });
  it('coalesces rapid edits and immediately hides the obsolete preview', async () => {
    const request = vi.fn<DocumentPreviewRequest>().mockResolvedValue(response());
    const hook = renderHook(props => useDocumentPreview(props, request), { initialProps: input() });
    await advance(); expect(hook.result.current.status).toBe('ready');
    hook.rerender(input()); expect(hook.result.current.presentation).toBeUndefined();
    await advance(200); hook.rerender(input()); await advance(299);
    expect(request).toHaveBeenCalledTimes(1);
    await advance(1); expect(request).toHaveBeenCalledTimes(2);
  });
  it.each(['document', 'route', 'locale', 'variantSlug'] as const)('aborts and discards late responses when %s changes even if transport ignores abort', async field => {
    const late: ((p: ReturnType<typeof response>) => void)[] = [];
    const request = vi.fn<DocumentPreviewRequest>().mockImplementation(() => new Promise(resolve => { late.push(resolve); }));
    const source = input();
    const hook = renderHook(props => useDocumentPreview(props, request), { initialProps: source });
    await advance();
    const next = { ...source };
    if (field === 'document') next.document = documentFixture();
    if (field === 'route') next.route = '/apps/example';
    if (field === 'locale') next.locale = 'en';
    if (field === 'variantSlug') next.variantSlug = 'review';
    hook.rerender(next);
    expect(request.mock.calls[0]?.[1].aborted).toBe(true);
    await act(async () => { late[0]?.(response()); await Promise.resolve(); });
    expect(hook.result.current.presentation).toBeUndefined();
    await advance(); expect(request).toHaveBeenCalledTimes(2);
  });
  it('does not request invalid JSON, invalid routes or disabled retained-preview mode', async () => {
    const request = vi.fn<DocumentPreviewRequest>();
    const hook = renderHook(props => useDocumentPreview(props, request), { initialProps: { ...input(), document: undefined } });
    await advance(); expect(request).not.toHaveBeenCalled();
    hook.unmount();
    const invalid = renderHook(() => useDocumentPreview({ ...input(), route: '/admin/secret' }, request));
    await advance(); expect(request).not.toHaveBeenCalled(); expect(invalid.result.current.error).toContain('exact');
    invalid.unmount(); renderHook(() => useDocumentPreview({ ...input(), enabled: false }, request));
    await advance(); expect(request).not.toHaveBeenCalled();
  });
  it('ignores a late authorization failure from an aborted preview without hiding the newer result', async () => {
    let rejectOld: ((reason: Error) => void) | undefined;
    const request = vi.fn<DocumentPreviewRequest>()
      .mockImplementationOnce(() => new Promise((_resolve, reject) => { rejectOld = reject; }))
      .mockResolvedValue(response());
    const hook = renderHook(props => useDocumentPreview(props, request), { initialProps: input() });
    await advance();
    hook.rerender(input()); await advance();
    expect(request.mock.calls[0]?.[1].aborted).toBe(true);
    expect(hook.result.current.status).toBe('ready');
    const current = hook.result.current.presentation;
    await act(async () => { rejectOld?.(new ConnectError('Old session request expired', Code.PermissionDenied)); await Promise.resolve(); });
    expect(hook.result.current.status).toBe('ready');
    expect(hook.result.current.presentation).toBe(current);
    expect(hook.result.current.unauthorized).not.toBeTruthy();
    expect(hook.result.current.error).not.toBeTruthy();
  });
  it.each([Code.InvalidArgument, Code.PermissionDenied, Code.Unavailable])('reports preview failure %s without discarding source', async code => {
    const source = input(); const original = source.document;
    const request = vi.fn<DocumentPreviewRequest>().mockRejectedValue(new ConnectError('pages[0]: invalid capability', code));
    const hook = renderHook(() => useDocumentPreview(source, request)); await advance();
    expect(hook.result.current.status).toBe('error'); expect(hook.result.current.presentation).toBeUndefined();
    expect(hook.result.current.unauthorized).toBe(code === Code.PermissionDenied);
    expect(hook.result.current.error).toContain(code === Code.InvalidArgument ? 'pages[0]' : code === Code.PermissionDenied ? 'Administrator' : 'not been saved');
    expect(source.document).toBe(original);
  });
  it.each(['preview', 'noindex', 'noStore', 'fallback', 'requestedRevision', 'resolvedRevision', 'blockDigest', 'requestedRoute', 'resolvedRoute', 'resolvedVariant', 'locale'] as const)('fails closed for wrong %s diagnostics', async field => {
    const p = response(); if (!p.diagnostics) throw new Error('fixture');
    Object.assign(p.diagnostics, { [field]: ['preview', 'noindex', 'noStore'].includes(field) ? false : field === 'fallback' ? true : field === 'resolvedRevision' || field === 'blockDigest' ? '' : 'wrong' });
    const request = vi.fn<DocumentPreviewRequest>().mockResolvedValue(p);
    const source = input();
    const hook = renderHook(() => useDocumentPreview(source, request));
    await advance(); expect(hook.result.current.status).toBe('error'); expect(hook.result.current.presentation).toBeUndefined();
  });
  it('cleans debounce and in-flight requests on unmount', async () => {
    const request = vi.fn<DocumentPreviewRequest>().mockImplementation(() => new Promise(() => {}));
    const source = input();
    const first = renderHook(() => useDocumentPreview(source, request)); first.unmount();
    await advance(); expect(request).not.toHaveBeenCalled();
    const second = renderHook(() => useDocumentPreview(source, request)); await advance(); second.unmount();
    expect(request.mock.calls[0]?.[1].aborted).toBe(true); expect(vi.getTimerCount()).toBe(0);
  });
});
