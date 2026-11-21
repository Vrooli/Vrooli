import { describe, it, expect, afterEach, vi } from 'vitest';
import { fetchHealth } from './health';
import { ApiError } from './client';

describe('fetchHealth', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('fetches and returns the health payload from the REST /health probe', async () => {
    const fetchSpy = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({ status: 'healthy', service: 'landing-page', timestamp: '2026-01-01T00:00:00Z' }),
        { status: 200 },
      ),
    );
    vi.stubGlobal('fetch', fetchSpy);

    const result = await fetchHealth();

    expect(result).toEqual({
      status: 'healthy',
      service: 'landing-page',
      timestamp: '2026-01-01T00:00:00Z',
    });
    const [url, init] = fetchSpy.mock.calls[0] as [string, RequestInit];
    expect(url).toMatch(/\/health$/);
    expect(init).toMatchObject({ method: 'GET', cache: 'no-store' });
  });

  it('throws ApiError on a non-ok response', async () => {
    const fetchSpy = vi.fn().mockResolvedValue(
      new Response('service unavailable', { status: 503 }),
    );
    vi.stubGlobal('fetch', fetchSpy);

    const err = await fetchHealth().catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect((err as ApiError).status).toBe(503);
  });
});
