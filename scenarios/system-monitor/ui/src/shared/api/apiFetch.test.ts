import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiErrorException, apiFetch, extractErrorMessage, isApiError, protoFetch, toApiError } from './apiFetch';

const fetchMock = vi.fn<typeof fetch>();

const ok = (body: unknown) => new Response(JSON.stringify(body), { status: 200, headers: { 'content-type': 'application/json' } });

describe('apiFetch and protoFetch', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', fetchMock);
    fetchMock.mockReset();
    fetchMock.mockResolvedValue(ok({ value: 42 }));
  });

  it('parses successful JSON and preserves request options', async () => {
    await expect(apiFetch<{ value: number }>('/metrics/current', { headers: { Authorization: 'Bearer test' } })).resolves.toEqual({ value: 42 });
    expect(fetchMock).toHaveBeenCalledWith(expect.stringContaining('/metrics/current'), { headers: { Authorization: 'Bearer test' } });
  });

  it('normalizes structured, legacy, text, malformed, and network failures', async () => {
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ error: { code: 'busy', message: 'Try later', retryable: true } }), { status: 429, statusText: 'Busy' }));
    await expect(apiFetch('/x')).rejects.toMatchObject({ error: 'Try later', detail: { code: 'busy' } });

    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ error: 'legacy failure' }), { status: 500, statusText: 'Server Error' }));
    await expect(apiFetch('/x')).rejects.toMatchObject({ error: 'legacy failure' });

    fetchMock.mockResolvedValueOnce(new Response('not-json', { status: 500, statusText: 'Server Error' }));
    await expect(apiFetch('/x')).rejects.toMatchObject({ error: 'HTTP 500: Server Error' });

    fetchMock.mockResolvedValueOnce(new Response('{bad', { status: 200 }));
    await expect(apiFetch('/x')).rejects.toMatchObject({ error: 'Invalid response from server' });

    fetchMock.mockRejectedValueOnce(new Error('offline'));
    await expect(apiFetch('/x')).rejects.toMatchObject({ error: 'Unable to reach the server. Check your connection.' });
  });

  it('passes parsed proto data through ordinary API routes', async () => {
    fetchMock.mockResolvedValueOnce(ok({ metrics: { cpu: 1 } }));
    const parser = vi.fn((value: unknown) => value as { cpu: number });
    await expect(protoFetch('/metrics/current', parser)).resolves.toEqual({ cpu: 1 });
    expect(parser).toHaveBeenCalledWith({ cpu: 1 });
    const firstCall = fetchMock.mock.calls[0];
    if (!firstCall) throw new Error('proto request was not issued');
    expect(firstCall[0]).toEqual(expect.stringContaining('GetCurrentMetrics'));
    const firstInit = firstCall[1];
    if (!firstInit || typeof firstInit !== 'object' || !('body' in firstInit) || typeof firstInit.body !== 'string') {
      throw new Error('proto request body was not serialized');
    }
    expect(JSON.parse(firstInit.body)).toEqual({ fresh: false });
  });

  it('maps Connect routes and query/body variants to typed procedures', async () => {
    const routes: Array<[string, RequestInit | undefined, string, Record<string, unknown>]> = [
      ['/metrics/detailed', undefined, 'GetDetailedMetrics', {}],
      ['/metrics/timeline?window=10&interval=2', undefined, 'GetMetricsTimeline', { windowSeconds: 10, sampleIntervalSeconds: 2 }],
      ['/metrics/processes/timeline?window=2m&owner=alice&top=5', undefined, 'GetProcessTimeline', { windowSeconds: 120, owner: 'alice', top: 5 }],
      ['/settings', undefined, 'GetSettings', {}],
      ['/settings', { method: 'PUT', body: JSON.stringify({ enabled: true }) }, 'UpdateSettings', { settings: { enabled: true } }],
      ['/settings/reset', { method: 'POST' }, 'ResetSettings', {}],
      ['/maintenance/state', undefined, 'GetMaintenanceState', {}],
      ['/maintenance/state', { method: 'POST', body: JSON.stringify({ active: true }) }, 'SetMaintenanceState', { active: true }],
      ['/reports', undefined, 'ListReports', {}],
      ['/reports/generate', { method: 'POST', body: JSON.stringify({ format: 'json' }) }, 'GenerateReport', { format: 'json' }],
      ['/capacity/overview', undefined, 'GetCapacityOverview', {}],
      ['/capacity/reconcile', { method: 'POST' }, 'ReconcileCapacity', {}],
      ['/capacity/policy', undefined, 'GetCapacityPolicy', {}],
      ['/capacity/policy', { method: 'PUT', body: JSON.stringify({ limit: 5 }) }, 'SetCapacityPolicy', { limit: 5 }],
      ['/investigations?limit=3', undefined, 'ListInvestigations', { limit: 3 }],
      ['/investigations/latest', undefined, 'GetLatestInvestigation', {}],
      ['/investigations/trigger', { method: 'POST', body: JSON.stringify({ auto_fix: true, note: 'why' }) }, 'TriggerInvestigation', { autoFix: true, note: 'why' }],
      ['/investigations/cooldown', undefined, 'GetCooldownStatus', {}],
      ['/investigations/cooldown/reset', { method: 'POST' }, 'ResetCooldown', {}],
      ['/investigations/cooldown/period', { method: 'PUT', body: JSON.stringify({ cooldown_period_seconds: 30 }) }, 'UpdateCooldownPeriod', { cooldownPeriodSeconds: 30 }],
      ['/investigations/triggers', undefined, 'GetTriggers', {}],
      ['/investigations/scripts', undefined, 'ListScripts', {}],
      ['/reports/a%2Fb', undefined, 'GetReport', { id: 'a/b' }],
      ['/investigations/agent/a%2Fb/status', undefined, 'GetInvestigation', { id: 'a/b' }],
      ['/investigations/agent/a%2Fb/stop', { method: 'POST' }, 'StopAgent', { id: 'a/b' }],
      ['/investigations/scripts/a%2Fb', undefined, 'GetScript', { id: 'a/b' }],
      ['/investigations/scripts/a%2Fb/execute', { method: 'POST', body: JSON.stringify({ content: 'echo hi' }) }, 'ExecuteScript', { id: 'a/b', content: 'echo hi' }],
      ['/investigations/scripts/a%2Fb', { method: 'PUT', body: JSON.stringify({ content: 'echo changed' }) }, 'UpdateScript', { id: 'a/b', content: 'echo changed' }],
    ];

    for (const [path, options, procedure, expectedBody] of routes) {
      fetchMock.mockResolvedValueOnce(ok({ value: true }));
      await protoFetch(path, value => value, options);
      const lastCall = fetchMock.mock.calls.at(-1);
      if (!lastCall) throw new Error(`request for ${path} was not issued`);
      const [url, init] = lastCall;
      if (typeof url !== 'string') throw new Error(`request URL for ${path} was not a string`);
      expect(url).toContain(procedure);
      if (!init || typeof init !== 'object' || !('body' in init) || typeof init.body !== 'string') {
        throw new Error(`request body for ${path} was not serialized`);
      }
      expect(JSON.parse(init.body)).toEqual(expectedBody);
    }
  });

  it('supports header shapes, parser failures, aborts, and utility guards', async () => {
    fetchMock.mockResolvedValueOnce(ok({ value: 1 }));
    await protoFetch('/settings', value => value, { headers: [['X-Test', 'yes']] });
    expect((fetchMock.mock.calls[0]?.[1] as RequestInit).headers).toMatchObject({ 'X-Test': 'yes', 'Content-Type': 'application/json' });

    fetchMock.mockResolvedValueOnce(ok({ value: 1 }));
    const headers = new Headers({ 'X-Header': 'yes' });
    await protoFetch('/settings', value => value, { headers });
    expect((fetchMock.mock.calls[1]?.[1] as RequestInit).headers).toMatchObject({ 'x-header': 'yes', 'Content-Type': 'application/json' });

    fetchMock.mockResolvedValueOnce(ok({ value: 1 }));
    await expect(protoFetch('/settings', () => { throw new Error('bad shape'); })).rejects.toMatchObject({ error: 'Invalid response format' });
    fetchMock.mockResolvedValueOnce(new Response('{bad', { status: 200 }));
    await expect(protoFetch('/settings', value => value)).rejects.toMatchObject({ error: 'Invalid response from server' });
    fetchMock.mockRejectedValueOnce(new DOMException('aborted', 'AbortError'));
    await expect(protoFetch('/settings', value => value)).rejects.toMatchObject({ name: 'AbortError' });

    const apiError = new ApiErrorException({ error: 'bad' });
    expect(isApiError(apiError)).toBe(true);
    expect(extractErrorMessage(apiError)).toBe('bad');
    expect(extractErrorMessage('unknown', 'fallback')).toBe('fallback');
    expect(toApiError(new Error('ordinary'))).toMatchObject({ error: 'ordinary', detail: { code: 'internal' } });
    expect(toApiError('network')).toMatchObject({ detail: { code: 'network' } });
  });

  it('covers raw routes, malformed bodies, window units, and connect failures', async () => {
    fetchMock.mockResolvedValueOnce(ok('primitive'));
    await expect(apiFetch('/raw')).resolves.toBe('primitive');

    const routeBodies: Array<[string, RequestInit, Record<string, unknown>]> = [
      ['/metrics/processes/timeline?window=500ms&top=0', {}, { windowSeconds: 1, owner: '', top: undefined }],
      ['/metrics/processes/timeline?window=2s', {}, { windowSeconds: 2, owner: '', top: undefined }],
      ['/metrics/processes/timeline?window=3h&owner=', {}, { windowSeconds: 10800, owner: '', top: undefined }],
      ['/metrics/processes/timeline?window=bad&top=nope', {}, { windowSeconds: undefined, owner: '', top: undefined }],
      ['/metrics/timeline?window=bad&interval=0', {}, { windowSeconds: undefined, sampleIntervalSeconds: undefined }],
      ['/metrics/current?fresh=true', {}, { fresh: true }],
      ['/settings', { method: 'PUT', body: '' }, { settings: {} }],
      ['/settings', { method: 'PUT', body: '{bad' }, { settings: {} }],
      ['/investigations?limit=0', {}, { limit: undefined }],
      ['/investigations/agent/current', {}, {}],
      ['/investigations/agent/spawn', { method: 'POST', body: JSON.stringify({ auto_fix: 1 }) }, { autoFix: true, note: '' }],
      ['/unknown-connect-route', {}, {}],
    ];
    for (const [path, options, expected] of routeBodies) {
      fetchMock.mockResolvedValueOnce(ok({ value: true }));
      await protoFetch(path, value => value, options);
      const call = fetchMock.mock.calls.at(-1);
      if (!call || !call[1] || typeof call[1] !== 'object') throw new Error(`request for ${path} was not issued`);
      if (!('body' in call[1]) || typeof call[1].body !== 'string') {
        expect(Object.keys(expected).length).toBe(0);
        continue;
      }
      const definedExpected = Object.fromEntries(Object.entries(expected).filter(([, value]) => value !== undefined));
      expect(JSON.parse(call[1].body)).toMatchObject(definedExpected);
    }

    fetchMock.mockRejectedValueOnce(new Error('proto offline'));
    await expect(protoFetch('/unknown', value => value)).rejects.toMatchObject({ error: 'Unable to reach the server. Check your connection.' });
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify({ error: { code: 'busy', message: 'busy' } }), { status: 503, statusText: 'Busy' }));
    await expect(protoFetch('/unknown', value => value)).rejects.toMatchObject({ error: 'busy' });
    fetchMock.mockResolvedValueOnce(new Response('{bad', { status: 200 }));
    await expect(protoFetch('/unknown', value => value)).rejects.toMatchObject({ error: 'Invalid response from server' });
    fetchMock.mockResolvedValueOnce(ok({ value: true }));
    await expect(protoFetch('/unknown', () => { throw new Error('decode'); })).rejects.toMatchObject({ error: 'Invalid response format' });
  });
});
