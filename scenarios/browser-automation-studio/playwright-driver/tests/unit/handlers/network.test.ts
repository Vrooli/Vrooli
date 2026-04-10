import { createMockContext, createMockPage, createTestConfig } from '../../helpers';
import { NetworkHandler, clearSessionRoutes } from '../../../src/handlers/network';
import type { HandlerContext } from '../../../src/handlers/base';
import { logger, metrics } from '../../../src/utils';
import { create } from '@bufbuild/protobuf';
import {
  ActionDefinitionSchema,
  ActionType,
  NetworkMockOperation,
  NetworkMockParamsSchema,
} from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';

type MockRoute = {
  request: () => { method: () => string; url: () => string; headers: () => Record<string, string> };
  fulfill: jest.Mock;
  continue: jest.Mock;
  abort: jest.Mock;
  fetch: jest.Mock;
};

function createNetworkInstruction(params: {
  operation: NetworkMockOperation;
  urlPattern: string;
  method?: string;
  statusCode?: number;
  headers?: Record<string, string>;
  body?: string;
}): { type: string; action: ReturnType<typeof create> } {
  return {
    type: 'network-mock',
    action: create(ActionDefinitionSchema, {
      type: ActionType.NETWORK_MOCK,
      params: {
        case: 'networkMock',
        value: create(NetworkMockParamsSchema, {
          operation: params.operation,
          urlPattern: params.urlPattern,
          method: params.method,
          statusCode: params.statusCode,
          headers: params.headers ?? {},
          body: params.body,
        }),
      },
    }),
  };
}

describe('NetworkHandler', () => {
  let handler: NetworkHandler;
  let mockPage: ReturnType<typeof createMockPage>;
  let context: HandlerContext;

  beforeEach(() => {
    handler = new NetworkHandler();
    mockPage = createMockPage();

    context = {
      page: mockPage,
      browserContext: createMockContext(),
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'network-session',
    };
  });

  afterEach(() => {
    clearSessionRoutes('network-session');
    jest.clearAllMocks();
  });

  it('registers a mock response route and fulfills with serialized body', async () => {
    const instruction = {
      index: 0,
      nodeId: 'node-1',
      ...createNetworkInstruction({
        operation: NetworkMockOperation.MOCK,
        urlPattern: 'example.com/api',
        method: 'GET',
        statusCode: 201,
        headers: { 'x-test': '1' },
        body: JSON.stringify({ ok: true }),
      }),
      params: {},
    };

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(mockPage.route).toHaveBeenCalledTimes(1);

    const routeHandler = mockPage.route.mock.calls[0]?.[1];
    const mockRoute: MockRoute = {
      request: () => ({
        method: () => 'GET',
        url: () => 'https://example.com/api',
        headers: () => ({ accept: 'application/json' }),
      }),
      fulfill: jest.fn().mockResolvedValue(undefined),
      continue: jest.fn().mockResolvedValue(undefined),
      abort: jest.fn().mockResolvedValue(undefined),
      fetch: jest.fn(),
    };

    await routeHandler?.(mockRoute as unknown as Parameters<typeof routeHandler>[0]);

    expect(mockRoute.fulfill).toHaveBeenCalledWith(expect.objectContaining({
      status: 201,
      headers: { 'x-test': '1' },
      contentType: 'application/json',
      body: JSON.stringify({ ok: true }),
    }));
  });

  it('returns idempotent result when registering same mock twice', async () => {
    const instruction = {
      index: 0,
      nodeId: 'node-2',
      ...createNetworkInstruction({
        operation: NetworkMockOperation.MOCK,
        urlPattern: 'example.com/api',
      }),
      params: {},
    };

    const first = await handler.execute(instruction, context);
    const second = await handler.execute(instruction, context);

    expect(first.success).toBe(true);
    expect(second.success).toBe(true);
    expect(second.extracted_data?.network?.idempotent).toBe(true);
    expect(mockPage.route).toHaveBeenCalledTimes(1);
  });

  it('modifies response headers and body when configured', async () => {
    const instruction = {
      index: 0,
      nodeId: 'node-3',
      ...createNetworkInstruction({
        operation: NetworkMockOperation.MODIFY_RESPONSE,
        urlPattern: 'example.com/api',
        method: 'POST',
        statusCode: 202,
        headers: { 'x-override': 'true' },
        body: JSON.stringify({ patched: true }),
      }),
      params: {},
    };

    await handler.execute(instruction, context);

    const routeHandler = mockPage.route.mock.calls[0]?.[1];
    const mockRoute: MockRoute = {
      request: () => ({
        method: () => 'POST',
        url: () => 'https://example.com/api',
        headers: () => ({}) ,
      }),
      fulfill: jest.fn().mockResolvedValue(undefined),
      continue: jest.fn().mockResolvedValue(undefined),
      abort: jest.fn().mockResolvedValue(undefined),
      fetch: jest.fn().mockResolvedValue({
        headers: () => ({ 'content-type': 'application/json', existing: '1' }),
        status: () => 200,
        body: () => Promise.resolve(Buffer.from('orig')),
      }),
    };

    await routeHandler?.(mockRoute as unknown as Parameters<typeof routeHandler>[0]);

    expect(mockRoute.fulfill).toHaveBeenCalledWith({
      status: 202,
      headers: { 'content-type': 'application/json', existing: '1', 'x-override': 'true' },
      body: JSON.stringify({ patched: true }),
    });
  });

  it('clears network mocks and unroutes handlers', async () => {
    const instruction = {
      index: 0,
      nodeId: 'node-4',
      ...createNetworkInstruction({
        operation: NetworkMockOperation.CLEAR,
        urlPattern: 'ignored',
      }),
      params: {},
    };

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(mockPage.unroute).toHaveBeenCalledWith('**/*');
    expect(result.extracted_data?.network?.clearedRouteCount).toBeDefined();
  });
});
