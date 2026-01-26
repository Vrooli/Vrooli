import { ServiceWorkerHandler } from '../../../src/handlers/service-worker';
import type { HandlerContext } from '../../../src/handlers/base';
import type { ActionDefinition } from '../../../src/types';
import { createMockContext, createMockPage, createTestConfig, createTestInstruction } from '../../helpers';
import { logger, metrics } from '../../../src/utils';

type ServiceWorkerController = {
  getWorkers: jest.Mock;
  unregister: jest.Mock;
  unregisterAll: jest.Mock;
  stopAll: jest.Mock;
};

function createServiceWorkerInstruction(params: { operation?: string; scopeURL?: string }) {
  return {
    ...createTestInstruction({ type: 'service-worker' }),
    action: {
      params: {
        value: {
          ...params,
        },
      },
    } as unknown as ActionDefinition,
  };
}

describe('ServiceWorkerHandler', () => {
  let handler: ServiceWorkerHandler;

  beforeEach(() => {
    handler = new ServiceWorkerHandler();
  });

  it('lists registered service workers', async () => {
    const swController: ServiceWorkerController = {
      getWorkers: jest.fn().mockReturnValue([{ scopeURL: 'https://example.com/' }]),
      unregister: jest.fn(),
      unregisterAll: jest.fn(),
      stopAll: jest.fn(),
    };

    const context: HandlerContext & { serviceWorkerController: ServiceWorkerController } = {
      page: createMockPage(),
      browserContext: createMockContext(),
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'sw-session',
      serviceWorkerController: swController,
    };

    const instruction = createServiceWorkerInstruction({ operation: 'list' });
    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(result.extracted_data?.workers).toEqual([{ scopeURL: 'https://example.com/' }]);
    expect(swController.getWorkers).toHaveBeenCalled();
  });

  it('unregisters a specific service worker', async () => {
    const swController: ServiceWorkerController = {
      getWorkers: jest.fn(),
      unregister: jest.fn().mockResolvedValue(true),
      unregisterAll: jest.fn(),
      stopAll: jest.fn(),
    };

    const context: HandlerContext & { serviceWorkerController: ServiceWorkerController } = {
      page: createMockPage(),
      browserContext: createMockContext(),
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'sw-session',
      serviceWorkerController: swController,
    };

    const instruction = createServiceWorkerInstruction({
      operation: 'unregister',
      scopeURL: 'https://example.com/app',
    });
    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(result.extracted_data).toEqual({ unregistered: true, scopeURL: 'https://example.com/app' });
    expect(swController.unregister).toHaveBeenCalledWith('https://example.com/app');
  });

  it('rejects unregister without scope URL', async () => {
    const swController: ServiceWorkerController = {
      getWorkers: jest.fn(),
      unregister: jest.fn(),
      unregisterAll: jest.fn(),
      stopAll: jest.fn(),
    };

    const context: HandlerContext & { serviceWorkerController: ServiceWorkerController } = {
      page: createMockPage(),
      browserContext: createMockContext(),
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'sw-session',
      serviceWorkerController: swController,
    };

    const instruction = createServiceWorkerInstruction({ operation: 'unregister' });
    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(false);
    expect(result.error?.code).toBe('MISSING_PARAM');
  });

  it('unregisters all service workers', async () => {
    const swController: ServiceWorkerController = {
      getWorkers: jest.fn(),
      unregister: jest.fn(),
      unregisterAll: jest.fn().mockResolvedValue(3),
      stopAll: jest.fn(),
    };

    const context: HandlerContext & { serviceWorkerController: ServiceWorkerController } = {
      page: createMockPage(),
      browserContext: createMockContext(),
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'sw-session',
      serviceWorkerController: swController,
    };

    const instruction = createServiceWorkerInstruction({ operation: 'unregister-all' });
    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(result.extracted_data?.unregisteredCount).toBe(3);
    expect(swController.unregisterAll).toHaveBeenCalled();
  });

  it('stops all service workers', async () => {
    const swController: ServiceWorkerController = {
      getWorkers: jest.fn(),
      unregister: jest.fn(),
      unregisterAll: jest.fn(),
      stopAll: jest.fn().mockResolvedValue(undefined),
    };

    const context: HandlerContext & { serviceWorkerController: ServiceWorkerController } = {
      page: createMockPage(),
      browserContext: createMockContext(),
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'sw-session',
      serviceWorkerController: swController,
    };

    const instruction = createServiceWorkerInstruction({ operation: 'stop-all' });
    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(swController.stopAll).toHaveBeenCalled();
  });

  it('returns an error when controller is missing', async () => {
    const context: HandlerContext = {
      page: createMockPage(),
      browserContext: createMockContext(),
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'sw-session',
    };

    const instruction = createServiceWorkerInstruction({ operation: 'list' });
    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(false);
    expect(result.error?.code).toBe('SW_NOT_INITIALIZED');
  });

  it('rejects missing operation', async () => {
    const swController: ServiceWorkerController = {
      getWorkers: jest.fn(),
      unregister: jest.fn(),
      unregisterAll: jest.fn(),
      stopAll: jest.fn(),
    };

    const context: HandlerContext & { serviceWorkerController: ServiceWorkerController } = {
      page: createMockPage(),
      browserContext: createMockContext(),
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'sw-session',
      serviceWorkerController: swController,
    };

    const instruction = createServiceWorkerInstruction({});
    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(false);
    expect(result.error?.code).toBe('MISSING_PARAM');
  });

  it('rejects unsupported operations', async () => {
    const swController: ServiceWorkerController = {
      getWorkers: jest.fn(),
      unregister: jest.fn(),
      unregisterAll: jest.fn(),
      stopAll: jest.fn(),
    };

    const context: HandlerContext & { serviceWorkerController: ServiceWorkerController } = {
      page: createMockPage(),
      browserContext: createMockContext(),
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'sw-session',
      serviceWorkerController: swController,
    };

    const instruction = createServiceWorkerInstruction({ operation: 'noop' });
    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(false);
    expect(result.error?.code).toBe('UNSUPPORTED_OPERATION');
  });
});
