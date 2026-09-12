import { describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({ apiFetch: vi.fn(), protoFetch: vi.fn() }));
vi.mock('./capacity/../shared/api/apiFetch', () => ({ apiFetch: mocks.apiFetch, protoFetch: mocks.protoFetch }));
vi.mock('../shared/api/apiFetch', () => ({ apiFetch: mocks.apiFetch, protoFetch: mocks.protoFetch }));

import { fetchCapacityOverview, fetchCapacityPolicy, fetchCapacityReconciliation, setCapacityPolicy } from './capacity/api';
import { fetchBootHistory, fetchForensicsSummary, fetchMCE, fetchPstore } from './forensics/api';
import { buildLogsQueryString, fetchBoots, fetchLogs, fetchUnits } from './logs/api';

describe('feature API surfaces', () => {
  it('forwards capacity requests and unwraps policy levers', async () => {
    mocks.protoFetch
      .mockResolvedValueOnce({ overview: true })
      .mockResolvedValueOnce({ findings: [] })
      .mockResolvedValueOnce({ levers: [{ key: 'gpu', value: '1' }] })
      .mockResolvedValueOnce({ levers: [{ key: 'gpu', value: '2' }] });
    const signal = new AbortController().signal;
    await expect(fetchCapacityOverview(signal)).resolves.toEqual({ overview: true });
    await expect(fetchCapacityReconciliation(signal)).resolves.toEqual({ findings: [] });
    await expect(fetchCapacityPolicy(signal)).resolves.toEqual([{ key: 'gpu', value: '1' }]);
    await expect(setCapacityPolicy('gpu', '2', signal)).resolves.toEqual([{ key: 'gpu', value: '2' }]);
    expect(mocks.protoFetch).toHaveBeenCalledWith('/api/v1/capacity/policy', expect.anything(), expect.objectContaining({ method: 'POST', body: JSON.stringify({ key: 'gpu', value: '2' }), signal }));
  });

  it('exposes all forensics endpoints with their optional signal', async () => {
    mocks.apiFetch.mockResolvedValue({ available: true });
    const signal = new AbortController().signal;
    await fetchForensicsSummary(signal);
    await fetchPstore(signal);
    await fetchBootHistory(signal);
    await fetchMCE(signal);
    // Paths passed to `apiFetch` are RELATIVE to the API base. `buildUrl` is
    // built from `resolveApiBase({ appendSuffix: true })`, which already ends
    // in `/api/v1`, so a path that repeats the prefix resolves to
    // `/api/v1/api/v1/...` and 404s. These assertions previously required the
    // doubled form, which is why the whole Crash Forensics page was dead.
    expect(mocks.apiFetch).toHaveBeenNthCalledWith(1, '/forensics/summary', { signal });
    expect(mocks.apiFetch).toHaveBeenNthCalledWith(2, '/forensics/pstore', { signal });
    expect(mocks.apiFetch).toHaveBeenNthCalledWith(3, '/forensics/boot-history', { signal });
    expect(mocks.apiFetch).toHaveBeenNthCalledWith(4, '/forensics/mce', { signal });
  });

  it('builds bounded log queries and forwards log, unit, and boot requests', async () => {
    const query = buildLogsQueryString({ filters: { units: ['api', 'ui'], kernel: true, since: '1h', until: 'now', priority: 'warning', grep: 'disk', boot: 'boot-1', limit: 99999 } });
    expect(query).toContain('unit=api');
    expect(query).toContain('kernel=true');
    expect(query).toContain('limit=500');
    expect(buildLogsQueryString({ filters: { units: [], limit: 0 }, cursor: 'next', direction: 'forward' })).toContain('limit=200');
    mocks.apiFetch.mockResolvedValue({ entries: [] });
    const signal = new AbortController().signal;
    await fetchLogs({ filters: { units: [], limit: 10 }, cursor: 'next', direction: 'backward', signal });
    await fetchUnits(signal);
    await fetchBoots(signal);
    expect(mocks.apiFetch).toHaveBeenCalledWith(expect.stringContaining('/logs?'), { signal });
    expect(mocks.apiFetch).toHaveBeenCalledWith('/logs/units', { signal });
    expect(mocks.apiFetch).toHaveBeenCalledWith('/logs/boots', { signal });

    // Guard the class of bug rather than the four instances of it: no path
    // handed to `apiFetch` may carry the base prefix itself.
    for (const [path] of mocks.apiFetch.mock.calls) {
      expect(String(path)).not.toContain('/api/v1');
    }
  });
});
