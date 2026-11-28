import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { spreadsheetEngine } from '../engine.js';
import type { WorkbookResult } from '../../preview/workbookData.js';

class TestWorker {
  static instances: TestWorker[] = [];
  onmessage: ((event: MessageEvent<WorkbookResult>) => void) | null = null;
  onerror: ((event: ErrorEvent) => void) | null = null;
  postMessage = vi.fn(); terminate = vi.fn();
  constructor() { TestWorker.instances.push(this); }
}
beforeEach(() => { TestWorker.instances = []; vi.stubGlobal('Worker', TestWorker); });
afterEach(() => { vi.unstubAllGlobals(); });
function worker() { const value = TestWorker.instances.at(-1); if (!value) throw Error('Missing worker'); return value; }

describe('spreadsheet engine worker lifecycle', () => {
  it('hands the blob to a worker and resolves with its result, then destroys it', async () => {
    const pending = spreadsheetEngine.parse('/blob', new AbortController().signal); const parser = worker();
    expect(parser.postMessage).toHaveBeenCalledWith({ href: '/blob' });
    parser.onmessage?.(new MessageEvent('message', { data: { error: 'encrypted' } }));
    await expect(pending).resolves.toEqual({ error: 'encrypted' }); expect(parser.terminate).toHaveBeenCalled();
  });

  it('terminates the worker and rejects when the caller aborts', async () => {
    const controller = new AbortController(); const pending = spreadsheetEngine.parse('/blob', controller.signal); const parser = worker();
    controller.abort(); await expect(pending).rejects.toBeDefined(); expect(parser.terminate).toHaveBeenCalled();
  });

  it('does not start a worker for a signal that is already aborted', async () => {
    const controller = new AbortController(); controller.abort();
    await expect(spreadsheetEngine.parse('/blob', controller.signal)).rejects.toBeDefined(); expect(TestWorker.instances).toHaveLength(0);
  });

  it('reports a crashed worker as invalid and an unavailable one as unavailable', async () => {
    const crashed = spreadsheetEngine.parse('/blob', new AbortController().signal); const error = new ErrorEvent('error', { cancelable: true });
    worker().onerror?.(error); expect(error.defaultPrevented).toBe(true); await expect(crashed).resolves.toEqual({ error: 'invalid' });
    vi.stubGlobal('Worker', undefined); await expect(spreadsheetEngine.parse('/blob', new AbortController().signal)).resolves.toEqual({ error: 'unavailable' });
  });
});
