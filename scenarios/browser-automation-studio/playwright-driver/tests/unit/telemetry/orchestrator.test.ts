import type { Page } from 'rebrowser-playwright';
import { TelemetryOrchestrator } from '../../../src/telemetry/orchestrator';
import type { HandlerResult } from '../../../src/handlers/base';
import { createTestConfig } from '../../helpers';

const consoleCollector = {
  getAndClear: jest.fn().mockReturnValue([{ level: 'info', message: 'log', timestamp: 't' }]),
  getLogs: jest.fn().mockReturnValue([{ level: 'info', message: 'log', timestamp: 't' }]),
  clear: jest.fn(),
  dispose: jest.fn(),
};

const networkCollector = {
  getAndClear: jest.fn().mockReturnValue([{ method: 'GET', url: 'https://example.com' }]),
  getEvents: jest.fn().mockReturnValue([{ method: 'GET', url: 'https://example.com' }]),
  clear: jest.fn(),
  dispose: jest.fn(),
};

jest.mock('../../../src/telemetry/collector', () => ({
  ConsoleLogCollector: jest.fn().mockImplementation(() => consoleCollector),
  NetworkCollector: jest.fn().mockImplementation(() => networkCollector),
}));

const captureScreenshot = jest.fn().mockResolvedValue({
  format: 'jpeg',
  data: 'screenshot-data',
  width: 100,
  height: 100,
});
const captureDOMSnapshot = jest.fn().mockResolvedValue({
  format: 'html',
  data: '<html></html>',
  sizeBytes: 42,
});
const captureElementContext = jest.fn().mockResolvedValue({
  selector: '#target',
  boundingBox: { x: 0, y: 0, width: 10, height: 10 },
  confidence: 0.8,
});

jest.mock('../../../src/telemetry/screenshot', () => ({
  captureScreenshot: (...args: unknown[]) => captureScreenshot(...args),
}));

jest.mock('../../../src/telemetry/dom', () => ({
  captureDOMSnapshot: (...args: unknown[]) => captureDOMSnapshot(...args),
}));

jest.mock('../../../src/telemetry/element-context', () => ({
  captureElementContext: (...args: unknown[]) => captureElementContext(...args),
}));

describe('TelemetryOrchestrator', () => {
  const page = {} as Page;

  beforeEach(() => {
    captureScreenshot.mockClear();
    captureDOMSnapshot.mockClear();
    captureElementContext.mockClear();
    consoleCollector.getAndClear.mockClear();
    consoleCollector.getLogs.mockClear();
    consoleCollector.clear.mockClear();
    consoleCollector.dispose.mockClear();
    networkCollector.getAndClear.mockClear();
    networkCollector.getEvents.mockClear();
    networkCollector.clear.mockClear();
    networkCollector.dispose.mockClear();
  });

  it('initializes collectors only when enabled', () => {
    const config = createTestConfig({
      telemetry: {
        console: { enabled: true },
        network: { enabled: false },
      },
    });

    const orchestrator = new TelemetryOrchestrator(page, config);
    orchestrator.start();

    expect(orchestrator.isActive()).toBe(true);
  });

  it('throws if collectForStep is called before start', async () => {
    const config = createTestConfig();
    const orchestrator = new TelemetryOrchestrator(page, config);

    await expect(orchestrator.collectForStep()).rejects.toThrow('TelemetryOrchestrator not started');
  });

  it('collects telemetry and respects handler-provided data', async () => {
    const config = createTestConfig();
    const orchestrator = new TelemetryOrchestrator(page, config);
    orchestrator.start();

    const handlerResult: HandlerResult = {
      success: true,
      screenshot: { format: 'jpeg', data: 'from-handler', width: 10, height: 10 },
      domSnapshot: { format: 'html', data: '<html />', sizeBytes: 10 },
      consoleLogs: [{ level: 'info', message: 'from-handler', timestamp: 't' }],
      networkEvents: [{ method: 'POST', url: 'https://example.com' }],
    };

    const telemetry = await orchestrator.collectForStep(handlerResult, {
      includeConsoleLogs: false,
      includeNetworkEvents: false,
    });

    expect(telemetry.screenshot?.data).toBe('from-handler');
    expect(telemetry.domSnapshot?.data).toBe('<html />');
    expect(telemetry.consoleLogs?.[0]?.message).toBe('from-handler');
    expect(telemetry.networkEvents?.[0]?.method).toBe('POST');
    expect(captureScreenshot).not.toHaveBeenCalled();
    expect(captureDOMSnapshot).not.toHaveBeenCalled();
  });

  it('forces capture when requested', async () => {
    const config = createTestConfig();
    const orchestrator = new TelemetryOrchestrator(page, config);
    orchestrator.start();

    const handlerResult: HandlerResult = {
      success: true,
      screenshot: { format: 'jpeg', data: 'from-handler', width: 10, height: 10 },
      domSnapshot: { format: 'html', data: '<html />', sizeBytes: 10 },
    };

    const telemetry = await orchestrator.collectForStep(handlerResult, {
      forceScreenshot: true,
      forceDomSnapshot: true,
    });

    expect(captureScreenshot).toHaveBeenCalled();
    expect(captureDOMSnapshot).toHaveBeenCalled();
    expect(telemetry.screenshot?.data).toBe('screenshot-data');
    expect(telemetry.domSnapshot?.data).toBe('<html></html>');
  });

  it('captures console and network events from collectors', async () => {
    const config = createTestConfig();
    const orchestrator = new TelemetryOrchestrator(page, config);
    orchestrator.start();

    const telemetry = await orchestrator.collectForStep();

    expect(telemetry.consoleLogs?.length).toBe(1);
    expect(telemetry.networkEvents?.length).toBe(1);
  });

  it('captures element context for actions', async () => {
    const config = createTestConfig();
    const orchestrator = new TelemetryOrchestrator(page, config);
    orchestrator.start();

    const context = await orchestrator.captureElementContextForAction('#target', {
      timeout: 1000,
    });

    expect(captureElementContext).toHaveBeenCalledWith(page, '#target', { timeout: 1000 });
    expect(context.selector).toBe('#target');
  });

  it('returns undefined when screenshot or DOM capture is disabled', async () => {
    const config = createTestConfig({
      telemetry: {
        screenshot: { enabled: false },
        dom: { enabled: false },
      },
    });
    const orchestrator = new TelemetryOrchestrator(page, config);
    orchestrator.start();

    expect(await orchestrator.captureScreenshot()).toBeUndefined();
    expect(await orchestrator.captureDomSnapshot()).toBeUndefined();
  });

  it('clears and exposes collected events', () => {
    const config = createTestConfig();
    const orchestrator = new TelemetryOrchestrator(page, config);
    orchestrator.start();

    orchestrator.getConsoleLogs();
    orchestrator.getNetworkEvents();
    orchestrator.clearConsoleLogs();
    orchestrator.clearNetworkEvents();

    expect(consoleCollector.getLogs).toHaveBeenCalled();
    expect(networkCollector.getEvents).toHaveBeenCalled();
    expect(consoleCollector.clear).toHaveBeenCalled();
    expect(networkCollector.clear).toHaveBeenCalled();
  });

  it('disposes collectors and blocks further collection', async () => {
    const config = createTestConfig();
    const orchestrator = new TelemetryOrchestrator(page, config);
    orchestrator.start();
    orchestrator.dispose();

    expect(orchestrator.isActive()).toBe(false);
    expect(consoleCollector.dispose).toHaveBeenCalled();
    expect(networkCollector.dispose).toHaveBeenCalled();

    await expect(orchestrator.collectForStep()).rejects.toThrow('disposed');
  });
});
