import { create } from '@bufbuild/protobuf';
import {
  ActionDefinitionSchema,
  ActionType,
  CookieStorageParamsSchema,
  CookieOperation,
  StorageType,
  CookieOptionsSchema,
  CookieSameSite,
} from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import { CookieStorageHandler } from '../../../src/handlers/cookie-storage';
import type { HandlerContext } from '../../../src/handlers/base';
import { createMockContext, createMockPage, createTestConfig } from '../../helpers';
import { logger, metrics } from '../../../src/utils';

type CookieInstructionParams = {
  operation: CookieOperation;
  storageType: StorageType;
  key?: string;
  value?: string;
  cookieOptions?: {
    domain?: string;
    path?: string;
    expires?: number;
    httpOnly?: boolean;
    secure?: boolean;
    sameSite?: CookieSameSite;
  };
};

function createCookieInstruction(params: CookieInstructionParams): { type: string; action: ReturnType<typeof create> } {
  return {
    type: 'cookie-storage',
    action: create(ActionDefinitionSchema, {
      type: ActionType.COOKIE_STORAGE,
      params: {
        case: 'cookieStorage',
        value: create(CookieStorageParamsSchema, {
          operation: params.operation,
          storageType: params.storageType,
          key: params.key,
          value: params.value,
          cookieOptions: params.cookieOptions
            ? create(CookieOptionsSchema, {
                domain: params.cookieOptions.domain,
                path: params.cookieOptions.path,
                expires: params.cookieOptions.expires,
                httpOnly: params.cookieOptions.httpOnly,
                secure: params.cookieOptions.secure,
                sameSite: params.cookieOptions.sameSite,
              })
            : undefined,
        }),
      },
    }),
  };
}

describe('CookieStorageHandler', () => {
  let handler: CookieStorageHandler;

  beforeEach(() => {
    handler = new CookieStorageHandler();
  });

  it('sets a cookie with provided options', async () => {
    const mockPage = createMockPage();
    const mockContext = createMockContext();
    const context: HandlerContext = {
      page: mockPage,
      browserContext: mockContext,
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'cookie-session',
    };

    const instruction = {
      index: 0,
      nodeId: 'node-1',
      ...createCookieInstruction({
        operation: CookieOperation.SET,
        storageType: StorageType.COOKIE,
        key: 'session',
        value: 'abc123',
        cookieOptions: {
          domain: 'example.com',
          path: '/',
          httpOnly: true,
          sameSite: CookieSameSite.STRICT,
        },
      }),
      params: {},
    };

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(mockContext.addCookies).toHaveBeenCalledWith([
      expect.objectContaining({
        name: 'session',
        value: 'abc123',
        domain: 'example.com',
        path: '/',
        httpOnly: true,
        sameSite: 'Strict',
      }),
    ]);
  });

  it('returns an error when setting cookie without name', async () => {
    const mockPage = createMockPage();
    const mockContext = createMockContext();
    const context: HandlerContext = {
      page: mockPage,
      browserContext: mockContext,
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'cookie-session',
    };

    const instruction = {
      index: 0,
      nodeId: 'node-2',
      ...createCookieInstruction({
        operation: CookieOperation.SET,
        storageType: StorageType.COOKIE,
        value: 'missing-name',
      }),
      params: {},
    };

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(false);
    expect(result.error?.code).toBe('MISSING_PARAM');
  });

  it('gets a specific cookie by name', async () => {
    const mockPage = createMockPage();
    const mockContext = createMockContext({
      cookies: jest.fn().mockResolvedValue([
        { name: 'session', value: 'abc123' },
        { name: 'other', value: 'xyz' },
      ]),
    });
    const context: HandlerContext = {
      page: mockPage,
      browserContext: mockContext,
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'cookie-session',
    };

    const instruction = {
      index: 0,
      nodeId: 'node-3',
      ...createCookieInstruction({
        operation: CookieOperation.GET,
        storageType: StorageType.COOKIE,
        key: 'session',
      }),
      params: {},
    };

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(result.extracted_data?.cookie).toBe('abc123');
  });

  it('gets all cookies when no name is provided', async () => {
    const mockPage = createMockPage();
    const mockContext = createMockContext({
      cookies: jest.fn().mockResolvedValue([
        { name: 'session', value: 'abc123' },
        { name: 'other', value: 'xyz' },
      ]),
    });
    const context: HandlerContext = {
      page: mockPage,
      browserContext: mockContext,
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'cookie-session',
    };

    const instruction = {
      index: 0,
      nodeId: 'node-4',
      ...createCookieInstruction({
        operation: CookieOperation.GET,
        storageType: StorageType.COOKIE,
      }),
      params: {},
    };

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(result.extracted_data?.cookie).toEqual({ session: 'abc123', other: 'xyz' });
  });

  it('deletes a cookie by name', async () => {
    const mockPage = createMockPage();
    const mockContext = createMockContext({
      cookies: jest.fn().mockResolvedValue([
        { name: 'session', value: 'abc123' },
        { name: 'other', value: 'xyz' },
      ]),
    });
    const context: HandlerContext = {
      page: mockPage,
      browserContext: mockContext,
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'cookie-session',
    };

    const instruction = {
      index: 0,
      nodeId: 'node-5',
      ...createCookieInstruction({
        operation: CookieOperation.DELETE,
        storageType: StorageType.COOKIE,
        key: 'session',
      }),
      params: {},
    };

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(mockContext.clearCookies).toHaveBeenCalled();
    expect(mockContext.addCookies).toHaveBeenCalledWith([{ name: 'other', value: 'xyz' }]);
  });

  it('clears all cookies', async () => {
    const mockPage = createMockPage();
    const mockContext = createMockContext();
    const context: HandlerContext = {
      page: mockPage,
      browserContext: mockContext,
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'cookie-session',
    };

    const instruction = {
      index: 0,
      nodeId: 'node-6',
      ...createCookieInstruction({
        operation: CookieOperation.CLEAR,
        storageType: StorageType.COOKIE,
      }),
      params: {},
    };

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(mockContext.clearCookies).toHaveBeenCalled();
  });

  it('sets and reads localStorage values', async () => {
    const mockPage = createMockPage({
      evaluate: jest.fn().mockResolvedValueOnce(undefined).mockResolvedValueOnce('dark'),
    });
    const mockContext = createMockContext();
    const context: HandlerContext = {
      page: mockPage,
      browserContext: mockContext,
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'storage-session',
    };

    const setInstruction = {
      index: 0,
      nodeId: 'node-7',
      ...createCookieInstruction({
        operation: CookieOperation.SET,
        storageType: StorageType.LOCAL_STORAGE,
        key: 'theme',
        value: 'dark',
      }),
      params: {},
    };

    const getInstruction = {
      index: 1,
      nodeId: 'node-8',
      ...createCookieInstruction({
        operation: CookieOperation.GET,
        storageType: StorageType.LOCAL_STORAGE,
        key: 'theme',
      }),
      params: {},
    };

    const setResult = await handler.execute(setInstruction, context);
    const getResult = await handler.execute(getInstruction, context);

    expect(setResult.success).toBe(true);
    expect(mockPage.evaluate).toHaveBeenCalledWith(expect.any(Function), {
      storage: 'localStorage',
      key: 'theme',
      value: 'dark',
    });

    expect(getResult.success).toBe(true);
    expect(getResult.extracted_data?.value).toBe('dark');
  });

  it('deletes sessionStorage values', async () => {
    const mockPage = createMockPage({
      evaluate: jest.fn().mockResolvedValue(undefined),
    });
    const mockContext = createMockContext();
    const context: HandlerContext = {
      page: mockPage,
      browserContext: mockContext,
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'storage-session',
    };

    const instruction = {
      index: 0,
      nodeId: 'node-9',
      ...createCookieInstruction({
        operation: CookieOperation.DELETE,
        storageType: StorageType.SESSION_STORAGE,
      }),
      params: {},
    };

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(mockPage.evaluate).toHaveBeenCalledWith(expect.any(Function), {
      storage: 'sessionStorage',
      key: undefined,
    });
  });
});
