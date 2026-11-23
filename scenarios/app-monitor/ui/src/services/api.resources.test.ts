import { afterEach, describe, expect, it, vi } from 'vitest';

const { apiGet } = vi.hoisted(() => ({ apiGet: vi.fn() }));

vi.mock('axios', () => {
  const instance = {
    get: apiGet,
    post: vi.fn(),
    interceptors: { request: { use: vi.fn() }, response: { use: vi.fn() } },
  };
  return { default: { create: () => instance, get: vi.fn() } };
});

import { resourceService } from './api';

describe('resourceService.getResources', () => {
  afterEach(() => vi.clearAllMocks());

  it('distinguishes a valid empty inventory from unavailable or malformed data', async () => {
    apiGet.mockResolvedValueOnce({ data: [] });
    await expect(resourceService.getResources()).resolves.toEqual([]);

    apiGet.mockRejectedValueOnce(new Error('inventory unavailable'));
    await expect(resourceService.getResources()).rejects.toThrow('inventory unavailable');

    apiGet.mockResolvedValueOnce({ data: { resources: [] } });
    await expect(resourceService.getResources()).rejects.toThrow('Resource inventory response must be an array.');
  });
});
