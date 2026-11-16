import type { Page, Route } from 'rebrowser-playwright';
import type winston from 'winston';
import { createEventRouteManager } from '../../../src/recording/io/event-route';
import { acknowledgeTimelineEntries, bufferTimelineEntry, getTimelineEntries, initRecordingBuffer, removeRecordingBuffer } from '../../../src/recording/io/buffer';
import { rawBrowserEventToTimelineEntry } from '../../../src/proto/recording';
import type { RawBrowserEvent } from '../../../src/recording/types';

describe('recording event route retries', () => {
  it('retries a failed commit even when Playwright can no longer resolve its original frame', async () => {
    const event: RawBrowserEvent = {
      id: 'event-1',
      recordingId: 'recording-1',
      actionType: 'click',
      timestamp: 1234,
      selector: { primary: '#open', candidates: [] },
      elementMeta: { tagName: 'button', isVisible: true, isEnabled: true },
      url: 'https://example.test/',
      payload: { text: 'Open' },
    };
    const body = JSON.stringify(event);
    const frameElement = {
      evaluate: jest.fn().mockResolvedValueOnce('#frame-one').mockRejectedValueOnce(new Error('detached frame')),
      dispose: jest.fn(async () => undefined),
    };
    const parentFrame = { parentFrame: () => null };
    const frame = {
      parentFrame: () => parentFrame,
      frameElement: async () => frameElement,
    };
    let routeHandler: ((route: Route) => Promise<void>) | undefined;
    const page = {
      route: jest.fn(async (_pattern: string, handler: (route: Route) => Promise<void>) => {
        routeHandler = handler;
      }),
      url: () => 'https://example.test/',
    } as unknown as Page;
    const eventHandler = jest
      .fn()
      .mockRejectedValueOnce(new Error('temporary persistence failure'))
      .mockResolvedValue(undefined);
    const pageIdentity = jest.fn()
      .mockReturnValueOnce('page-one')
      .mockImplementationOnce(() => { throw new Error('page identity unavailable'); });
    const manager = createEventRouteManager({
      logger: { debug: jest.fn(), info: jest.fn(), warn: jest.fn(), error: jest.fn() } as unknown as winston.Logger,
      getEventHandler: () => eventHandler,
      getDriverPageId: pageIdentity,
    });

    await manager.setupPageEventRoute(page);
    expect(routeHandler).toBeDefined();

    const invoke = async () => {
      const fulfill = jest.fn(async () => undefined);
      const route = {
        request: () => ({
          postData: () => body,
          frame: () => frame,
          url: () => 'http://localhost/__vrooli_recording_event__',
          method: () => 'POST',
        }),
        fulfill,
      } as unknown as Route;
      await routeHandler!(route);
      return fulfill.mock.calls[0][0];
    };

    expect(await invoke()).toMatchObject({ status: 500 });
    expect(await invoke()).toMatchObject({ status: 200 });
    expect(frameElement.evaluate).toHaveBeenCalledTimes(1);
    expect(pageIdentity).toHaveBeenCalledTimes(1);
    expect(eventHandler).toHaveBeenCalledTimes(2);
    expect(eventHandler.mock.calls[0][0]).toMatchObject({
      framePath: ['#frame-one'],
      driverPageId: 'page-one',
    });
    expect(eventHandler.mock.calls[1][0]).toMatchObject({
      framePath: ['#frame-one'],
      driverPageId: 'page-one',
    });
  });

  it('deduplicates the retained canonical entry when a frame changes during a failed commit retry', async () => {
    const sessionId = 'event-route-buffer-retry';
    initRecordingBuffer(sessionId);
    try {
      const event: RawBrowserEvent = {
        id: 'event-buffer-retry',
        recordingId: 'recording-buffer-retry',
        actionType: 'click',
        timestamp: 1234,
        selector: { primary: '#open', candidates: [] },
        elementMeta: { tagName: 'button', isVisible: true, isEnabled: true },
        url: 'https://example.test/',
        payload: { text: 'Open' },
      };
      const body = JSON.stringify(event);
      let selector = '#frame-one';
      const frameElement = {
        evaluate: jest.fn(async () => selector),
        dispose: jest.fn(async () => undefined),
      };
      const frame = { parentFrame: () => ({ parentFrame: () => null }), frameElement: async () => frameElement };
      let routeHandler: ((route: Route) => Promise<void>) | undefined;
      const page = {
        route: jest.fn(async (_pattern: string, handler: (route: Route) => Promise<void>) => { routeHandler = handler; }),
        url: () => 'https://example.test/',
      } as unknown as Page;
      const insertions: boolean[] = [];
      let failFirstDelivery = true;
      const eventHandler = jest.fn(async (raw: RawBrowserEvent) => {
        const entry = rawBrowserEventToTimelineEntry(raw, { sessionId, sequenceNum: 0 });
        insertions.push(bufferTimelineEntry(sessionId, entry));
        if (failFirstDelivery) {
          failFirstDelivery = false;
          throw new Error('temporary persistence failure');
        }
      });
      const manager = createEventRouteManager({
        logger: { debug: jest.fn(), info: jest.fn(), warn: jest.fn(), error: jest.fn() } as unknown as winston.Logger,
        getEventHandler: () => eventHandler,
        getDriverPageId: jest.fn().mockReturnValueOnce('page-one').mockReturnValueOnce('page-two'),
      });

      await manager.setupPageEventRoute(page);
      const invoke = async () => {
        const fulfill = jest.fn(async () => undefined);
        const route = {
          request: () => ({
            postData: () => body,
            frame: () => frame,
            url: () => 'http://localhost/__vrooli_recording_event__',
            method: () => 'POST',
          }),
          fulfill,
        } as unknown as Route;
        await routeHandler!(route);
        return fulfill.mock.calls[0][0];
      };

      expect(await invoke()).toMatchObject({ status: 500 });
      selector = '#frame-two';
      expect(await invoke()).toMatchObject({ status: 200 });
      expect(insertions).toEqual([true, false]);
      expect(getTimelineEntries(sessionId)).toHaveLength(1);
      expect(getTimelineEntries(sessionId)[0]?.telemetry?.framePath).toEqual(['#frame-one']);
    } finally {
      acknowledgeTimelineEntries(sessionId, getTimelineEntries(sessionId).map(({ id }) => id));
      removeRecordingBuffer(sessionId);
    }
  });
});
