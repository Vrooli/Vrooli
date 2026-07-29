import { afterEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '../api/common';
import {
  assertDefined,
  createApiErrorMock,
  createDownloadLinkMock,
  createFetchMock,
  createObjectURLMock,
  createWindowOpenMock,
  expectApiError,
  getCall,
  getFetchCall,
  getFirstCall,
  installFetchMock,
  mockResponses,
  parseJsonBody,
} from './api-mocks';

describe('API test mocks', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('builds successful, empty, and typed error responses', async () => {
    const success = mockResponses.success({ id: 'plan-1' }, 201);
    expect(success).toMatchObject({ ok: true, status: 201, statusText: 'OK' });
    await expect(success.json?.()).resolves.toEqual({ id: 'plan-1' });
    await expect(success.text?.()).resolves.toBe('{"id":"plan-1"}');

    const empty = mockResponses.empty();
    expect(empty).toMatchObject({ ok: true, status: 204, statusText: 'No Content' });
    await expect(empty.json?.()).resolves.toBeUndefined();

    const validation = mockResponses.validationError({ email: 'Required' });
    expect(validation).toMatchObject({ ok: false, status: 400, statusText: 'Bad Request' });
    await expect(validation.json?.()).resolves.toEqual({ error: 'Validation failed', errors: { email: 'Required' } });

    expect(mockResponses.unauthorized('Sign in')).toMatchObject({ status: 401, statusText: 'Unauthorized' });
    expect(mockResponses.forbidden()).toMatchObject({ status: 403, statusText: 'Forbidden' });
    expect(mockResponses.notFound()).toMatchObject({ status: 404, statusText: 'Not Found' });
    expect(mockResponses.rateLimited(12)).toMatchObject({ status: 429, statusText: 'Too Many Requests' });
    expect(mockResponses.serverError()).toMatchObject({ status: 500, statusText: 'Internal Server Error' });
    expect(mockResponses.error(418, 'Teapot')).toMatchObject({ status: 418, statusText: 'Unknown Status' });
    expect(mockResponses.networkError()).toBeInstanceOf(TypeError);
    expect(mockResponses.timeout()).toMatchObject({ name: 'AbortError', message: 'Aborted' });
  });

  it('installs fetch mocks and exposes typed request calls', () => {
    const fetchMock = createFetchMock();
    installFetchMock(fetchMock);
    void fetchMock('https://example.test/plans', { method: 'POST' });
    void fetchMock(new URL('https://example.test/credits'));
    void fetchMock(new Request('https://example.test/account'));

    expect(getFetchCall(fetchMock)).toEqual(['https://example.test/plans', { method: 'POST' }]);
    expect(getFetchCall(fetchMock, 1)).toEqual(['https://example.test/credits', {}]);
    expect(getFetchCall(fetchMock, 2)).toEqual(['https://example.test/account', {}]);
    expect(() => getFetchCall(fetchMock, 3)).toThrow(/call at index 3/);
  });

  it('provides clear assertion helpers for mocks and JSON bodies', () => {
    const callback = vi.fn<(value: string) => number>((value) => value.length);
    callback('first');
    callback('second');
    expect(getFirstCall(callback)).toEqual(['first']);
    expect(getCall(callback, 1)).toEqual(['second']);
    expect(() => getCall(callback, 2)).toThrow(/call at index 2/);
    expect(() => getFirstCall(vi.fn<() => undefined>())).toThrow(/at least once/);

    expect(parseJsonBody('{"name":"Suite"}')).toEqual({ name: 'Suite' });
    expect(() => parseJsonBody(null)).toThrow(/JSON string/);
    expect(() => parseJsonBody('"not an object"')).toThrow(/JSON body to be an object/);
    expect(() => { assertDefined(undefined, 'plan'); }).toThrow(/Expected plan/);
    const value: string | undefined = 'defined';
    assertDefined(value, 'value');
    expect(value).toBe('defined');
  });

  it('creates and validates API errors with stable defaults', () => {
    const error = createApiErrorMock('rate_limited');
    expect(error).toBeInstanceOf(ApiError);
    expectApiError(error, 'rate_limited', 429);
    expect(createApiErrorMock('network', 'Offline', undefined, 'Try again')).toMatchObject({
      type: 'network', message: 'Offline', userMessage: 'Try again',
    });
    expect(() => { expectApiError(new Error('wrong'), 'network'); }).toThrow(/Expected ApiError/);
    expect(() => { expectApiError(error, 'server_error'); }).toThrow(/type "server_error"/);
    expect(() => { expectApiError(error, 'rate_limited', 500); }).toThrow(/status 500/);
  });

  it('restores browser download and URL mocks after use', () => {
    const originalOpen = window.open;
    const open = createWindowOpenMock();
    open.mock('https://example.test/export');
    expect(open.mock).toHaveBeenCalledWith('https://example.test/export');
    open.restore();
    expect(window.open).toBe(originalOpen);

    const objectUrl = createObjectURLMock();
    expect(URL.createObjectURL(new Blob(['export']))).toBe('blob:test-url');
    objectUrl.restore();

    const download = createDownloadLinkMock();
    const anchor = document.createElement('a');
    document.body.appendChild(anchor);
    anchor.click();
    document.body.removeChild(anchor);
    expect(download.clickSpy).toHaveBeenCalledOnce();
    expect(download.appendChildSpy).toHaveBeenCalledWith(anchor);
    expect(download.removeChildSpy).toHaveBeenCalledWith(anchor);
    download.restore();
  });
});
