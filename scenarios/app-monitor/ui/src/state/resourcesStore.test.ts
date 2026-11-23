import { beforeEach, describe, expect, it, vi } from 'vitest';

import { resourceService } from '@/services/api';
import type { Resource } from '@/types';
import { useResourcesStore } from './resourcesStore';

vi.mock('@/services/api', () => ({
  resourceService: {
    getResources: vi.fn(),
    startResource: vi.fn(),
    stopResource: vi.fn(),
    getResourceStatus: vi.fn(),
  },
}));

vi.mock('@/services/logger', () => ({ logger: { warn: vi.fn() } }));

const initialResource: Resource = { id: 'ssh', name: 'SSH', type: 'service', status: 'online' };
const recoveredResource: Resource = { ...initialResource, status: 'offline' };

describe('resourcesStore', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useResourcesStore.setState({ resources: [], loading: false, error: null, loadError: null, hasInitialized: false });
  });

  it('keeps the last known resource list and exposes retryable refresh loss', async () => {
    vi.mocked(resourceService.getResources).mockResolvedValueOnce([initialResource]);
    await useResourcesStore.getState().loadResources();

    vi.mocked(resourceService.getResources).mockRejectedValueOnce(new Error('temporary inventory outage'));
    await useResourcesStore.getState().loadResources({ force: true });
    expect(useResourcesStore.getState().resources).toEqual([initialResource]);
    expect(String(useResourcesStore.getState().loadError)).toMatch(/temporary inventory outage/i);

    vi.mocked(resourceService.getResources).mockResolvedValueOnce([recoveredResource]);
    await useResourcesStore.getState().loadResources({ force: true });
    expect(useResourcesStore.getState().resources).toEqual([recoveredResource]);
    expect(useResourcesStore.getState().loadError).toBeNull();
  });
});
