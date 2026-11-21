import { describe, it, expect, vi, afterEach } from 'vitest';
import {
  ApiError,
  decodeApiError,
  uploadFile,
  jsonValueToJs,
  jsonMapToRecord,
  jsToJsonValue,
  recordToJsonMap,
} from './client';

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('json value converters', () => {
  it('round-trips every JS value kind through jsToJsonValue/jsonValueToJs', () => {
    const record = {
      flag: true,
      count: 3,
      big: 10n,
      name: 'hi',
      nothing: null,
      list: [1, 'two', false],
      nested: { a: 1, b: ['x'] },
    };
    const map = recordToJsonMap(record);
    const back = jsonMapToRecord(map);
    expect(back.flag).toBe(true);
    expect(back.count).toBe(3);
    // bigint collapses to number for display ergonomics.
    expect(back.big).toBe(10);
    expect(back.name).toBe('hi');
    expect(back.nothing).toBeNull();
    expect(back.list).toEqual([1, 'two', false]);
    expect(back.nested).toEqual({ a: 1, b: ['x'] });
  });

  it('returns undefined for an empty JsonValue and empty containers', () => {
    expect(jsonValueToJs(undefined)).toBeUndefined();
    expect(jsonMapToRecord(undefined)).toEqual({});
  });

  it('maps a bigint value to an int JsonValue', () => {
    const v = jsToJsonValue(9007199254740993n);
    expect(v.kind.case).toBe('intValue');
  });
});

describe('decodeApiError', () => {
  it('uses the response body text as the message', async () => {
    const res = new Response('boom detail', { status: 400, statusText: 'Bad Request' });
    const err = await decodeApiError(res);
    expect(err).toBeInstanceOf(ApiError);
    expect(err.status).toBe(400);
    expect(err.message).toBe('boom detail');
  });

  it('falls back to the status message when the body is unreadable', async () => {
    const res = { status: 503, statusText: 'Service Unavailable', text: () => Promise.reject(new Error('no body')) } as unknown as Response;
    const err = await decodeApiError(res);
    expect(err.status).toBe(503);
    expect(err.message).toBe('Service Unavailable');
  });
});

describe('uploadFile', () => {
  it('POSTs multipart form data with credentials', async () => {
    const fetchSpy = vi.fn().mockResolvedValue(new Response('{}', { status: 200 }));
    vi.stubGlobal('fetch', fetchSpy);
    const fd = new FormData();
    fd.append('file', new File(['x'], 'a.png', { type: 'image/png' }));
    await uploadFile('/admin/assets/upload', fd);
    const [, init] = fetchSpy.mock.calls[0] as [string, RequestInit];
    expect(init).toMatchObject({ method: 'POST', credentials: 'include', cache: 'no-store' });
  });
});
