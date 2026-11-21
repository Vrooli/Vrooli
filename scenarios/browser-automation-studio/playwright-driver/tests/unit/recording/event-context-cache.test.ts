import { RecordingEventContextCache } from '../../../src/recording/io/event-context-cache';
import type { RawBrowserEvent } from '../../../src/recording/types';

const event: RawBrowserEvent = {
  id: 'event-1',
  recordingId: 'recording-1',
  timestamp: 1234,
  actionType: 'click',
  selector: { primary: '#open', candidates: [] },
  elementMeta: { tagName: 'button', isVisible: true, isEnabled: true },
  url: 'https://example.test/',
  payload: { text: 'Open' },
};

describe('RecordingEventContextCache', () => {
  it('pins Playwright-derived attribution while a browser retries the same payload', async () => {
    const cache = new RecordingEventContextCache();

    await expect(cache.resolve(event, () => ({ framePath: ['html > iframe:nth-of-type(1)'], driverPageId: 'page-1' }))).resolves.toEqual({
      framePath: ['html > iframe:nth-of-type(1)'],
      driverPageId: 'page-1',
    });
    await expect(cache.resolve(event, () => ({ framePath: ['html > iframe:nth-of-type(2)'], driverPageId: 'page-2' }))).resolves.toEqual({
      framePath: ['html > iframe:nth-of-type(1)'],
      driverPageId: 'page-1',
    });
  });

  it('rejects reuse of an event id for a different browser payload', async () => {
    const cache = new RecordingEventContextCache();
    await cache.resolve(event, () => ({ framePath: [], driverPageId: 'page-1' }));

    await expect(cache.resolve({ ...event, payload: { text: 'Delete' } }, () => ({ framePath: [], driverPageId: 'page-1' })))
      .rejects.toThrow('Recording event identity conflicts with its original browser payload');
  });

  it('expires old retry metadata so long recordings stay bounded', async () => {
    let now = 1_000;
    const cache = new RecordingEventContextCache(() => now);
    await cache.resolve(event, () => ({ framePath: [], driverPageId: 'page-1' }));
    now += 5 * 60 * 1000;

    await expect(cache.resolve(event, () => ({ framePath: ['new-path'], driverPageId: 'page-2' }))).resolves.toEqual({
      framePath: ['new-path'],
      driverPageId: 'page-2',
    });
  });

  it('sweeps retained metadata at a bounded cadence during event bursts', async () => {
    let now = 1_000;
    const cache = new RecordingEventContextCache(() => now);
    const prune = jest.spyOn(cache as unknown as { prune: (at: number) => void }, 'prune');

    await cache.resolve(event, () => ({ framePath: [], driverPageId: 'page-1' }));
    await cache.resolve({ ...event, id: 'event-2' }, () => ({ framePath: [], driverPageId: 'page-1' }));
    expect(prune).toHaveBeenCalledTimes(1);

    now += 999;
    await cache.resolve({ ...event, id: 'event-3' }, () => ({ framePath: [], driverPageId: 'page-1' }));
    expect(prune).toHaveBeenCalledTimes(1);

    now += 1;
    await cache.resolve({ ...event, id: 'event-4' }, () => ({ framePath: [], driverPageId: 'page-1' }));
    expect(prune).toHaveBeenCalledTimes(2);
  });

  it('does not consult a detached Playwright context while retrying a pinned event', async () => {
    const cache = new RecordingEventContextCache();
    const derive = jest.fn(() => ({ framePath: ['#stable-frame'], driverPageId: 'page-1' }));
    await cache.resolve(event, derive);

    await expect(cache.resolve(event, () => { throw new Error('frame detached'); })).resolves.toEqual({
      framePath: ['#stable-frame'],
      driverPageId: 'page-1',
    });
    expect(derive).toHaveBeenCalledTimes(1);
  });

  it('coalesces simultaneous retries while the first Playwright context is being resolved', async () => {
    const cache = new RecordingEventContextCache();
    let release!: (value: { framePath: string[]; driverPageId: string }) => void;
    const derive = jest.fn(() => new Promise<{ framePath: string[]; driverPageId: string }>((resolve) => {
      release = resolve;
    }));
    const first = cache.resolve(event, derive);
    const retry = cache.resolve(event, () => Promise.resolve({ framePath: ['#different'], driverPageId: 'page-2' }));
    release({ framePath: ['#stable-frame'], driverPageId: 'page-1' });

    await expect(Promise.all([first, retry])).resolves.toEqual([
      { framePath: ['#stable-frame'], driverPageId: 'page-1' },
      { framePath: ['#stable-frame'], driverPageId: 'page-1' },
    ]);
    expect(derive).toHaveBeenCalledTimes(1);
  });
});
