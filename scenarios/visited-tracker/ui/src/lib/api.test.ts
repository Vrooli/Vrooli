import { afterEach, describe, expect, it, vi } from 'vitest';
import * as api from './api';

vi.mock('@vrooli/api-base', () => ({
  resolveApiBase: () => '/api/v1',
  buildApiUrl: (path: string, options: { baseUrl: string }) => options.baseUrl + path,
}));

afterEach(() => vi.unstubAllGlobals());

const campaign = { name: 'review', from_agent: 'worker', patterns: ['src/*.ts'] };
const visit = { paths: ['src/a.ts'], notes: 'reviewed' };
const requests = [
  { name: 'health', run: () => api.fetchHealth(), path: '/health', method: 'GET', json: true },
  { name: 'campaign listing', run: () => api.fetchCampaigns(), path: '/campaigns', method: 'GET', json: true },
  { name: 'campaign details', run: () => api.fetchCampaign('id/segment'), path: '/campaigns/id%2Fsegment', method: 'GET', json: true },
  { name: 'create campaign', run: () => api.createCampaign(campaign), path: '/campaigns', method: 'POST', body: campaign, json: true },
  { name: 'delete campaign', run: () => api.deleteCampaign('id/segment'), path: '/campaigns/id%2Fsegment', method: 'DELETE', json: false },
  { name: 'record visit', run: () => api.recordVisit('id/segment', visit), path: '/campaigns/id%2Fsegment/visit', method: 'POST', body: visit, json: false },
  { name: 'least visited', run: () => api.fetchLeastVisited('id/segment', 5), path: '/campaigns/id%2Fsegment/prioritize/least-visited?limit=5', method: 'GET', json: true },
  { name: 'most stale', run: () => api.fetchMostStale('id/segment'), path: '/campaigns/id%2Fsegment/prioritize/most-stale?limit=10', method: 'GET', json: true },
];

describe('campaign HTTP contract', () => {
  it.each(requests)('sends $name to its declared route and preserves the response', async (request) => {
    const payload = { value: 'server response' };
    const json = vi.fn().mockResolvedValue(payload);
    const fetch = vi.fn().mockResolvedValue({ ok: true, status: request.json ? 200 : 204, json });
    vi.stubGlobal('fetch', fetch);
    expect(await request.run()).toEqual(request.json ? payload : undefined);
    expect(fetch).toHaveBeenCalledTimes(1);
    const [url, options] = fetch.mock.calls[0];
    expect(url).toBe('/api/v1' + request.path);
    expect(options.method ?? 'GET').toBe(request.method);
    if ('body' in request) expect(JSON.parse(options.body)).toEqual(request.body);
    if (request.method === 'GET') expect(options.cache).toBe('no-store');
    expect(json).toHaveBeenCalledTimes(request.json ? 1 : 0);
  });

  it.each(requests)('rejects a failed $name response before reading a success payload', async (request) => {
    const json = vi.fn();
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 503, json }));
    await expect(request.run()).rejects.toThrow('503');
    expect(json).not.toHaveBeenCalled();
  });

  it('distinguishes a duplicate campaign from a transient server failure', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 409 }));
    await expect(api.createCampaign(campaign)).rejects.toThrow('A campaign with this name already exists');
  });
});
