import {
  AppMonitorTraceCapture,
  APP_MONITOR_TRACE_CATEGORIES,
} from '../../../src/tracing/app-monitor-capture';
import type { CDPSession, Page } from 'rebrowser-playwright';

function fakeCDP(raw: string) {
  let complete: ((value: { stream: string }) => void) | undefined;
  let reads = 0;
  const cdp = {
    send: jest.fn(async (method: string) => {
      if (method === 'Tracing.start') return {};
      if (method === 'Tracing.end') {
        complete?.({ stream: 'stream-1' });
        return {};
      }
      if (method === 'IO.read') {
        reads += 1;
        return reads === 1 ? { data: raw, eof: true } : { data: '', eof: true };
      }
      return {};
    }),
    on: jest.fn((_event: string, callback: (value: { stream: string }) => void) => {
      complete = callback;
    }),
    off: jest.fn(),
    detach: jest.fn(async () => undefined),
  } as unknown as CDPSession & { send: jest.Mock; detach: jest.Mock };
  return cdp;
}

describe('AppMonitorTraceCapture', () => {
  test('uses the fixed category allowlist and strips args and page-derived strings', async () => {
    const cdp = fakeCDP(
      JSON.stringify({
        traceEvents: [
          {
            name: 'RunTask',
            cat: 'devtools.timeline',
            ph: 'X',
            ts: 12,
            dur: 3,
            pid: 1,
            tid: 2,
            args: { url: 'https://secret.example/path', text: 'private' },
          },
          {
            name: 'https://secret.example/private.js',
            cat: 'v8.execute',
            ph: 'X',
            ts: 20,
            dur: 2,
            args: { scriptName: 'secret' },
          },
          { name: 'Ignored', cat: 'unapproved.category', ph: 'X', ts: 30 },
        ],
      })
    );
    const capture = await AppMonitorTraceCapture.start(
      '550e8400-e29b-41d4-a716-446655440000',
      60_000,
      1024 * 1024,
      {} as Page,
      async () => cdp
    );
    const artifact = await capture.finish();
    expect(cdp.send).toHaveBeenCalledWith(
      'Tracing.start',
      expect.objectContaining({
        traceConfig: expect.objectContaining({
          includedCategories: [...APP_MONITOR_TRACE_CATEGORIES],
        }),
      })
    );
    expect(artifact.traceJson).toContain('RunTask');
    expect(artifact.traceJson).toContain('OtherEvent');
    expect(artifact.traceJson).not.toContain('secret.example');
    expect(artifact.traceJson).not.toContain('private');
    expect(artifact.traceJson).not.toContain('unapproved.category');
    expect(artifact.byteLength).toBe(Buffer.byteLength(artifact.traceJson));
    expect(artifact.sha256).toMatch(/^[0-9a-f]{64}$/);
    expect(cdp.detach).toHaveBeenCalledTimes(1);
  });

  test('rejects a trace over the caller-independent byte cap', async () => {
    const cdp = fakeCDP(
      JSON.stringify({
        traceEvents: [
          {
            name: 'RunTask',
            cat: 'devtools.timeline',
            ph: 'X',
            ts: 1,
            args: { noise: 'x'.repeat(1000) },
          },
        ],
      })
    );
    const capture = await AppMonitorTraceCapture.start(
      '550e8400-e29b-41d4-a716-446655440000',
      60_000,
      128,
      {} as Page,
      async () => cdp
    );
    await expect(capture.finish()).rejects.toThrow('authorized byte limit');
    expect(cdp.detach).toHaveBeenCalledTimes(1);
  });

  test('rejects duration overflow and still detaches the driver session', async () => {
    const cdp = fakeCDP(JSON.stringify({ traceEvents: [] }));
    const capture = await AppMonitorTraceCapture.start(
      '550e8400-e29b-41d4-a716-446655440000',
      1,
      1024,
      {} as Page,
      async () => cdp
    );
    await expect(capture.finish(Date.now() + 10)).rejects.toThrow('duration limit');
    expect(cdp.detach).toHaveBeenCalledTimes(1);
  });
});
