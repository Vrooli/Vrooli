import { create } from '@bufbuild/protobuf';
import {
  ActionDefinitionSchema,
  ActionType,
  RotateParamsSchema,
  DeviceOrientation,
} from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import { DeviceHandler } from '../../../src/handlers/device';
import type { HandlerContext, HandlerResult } from '../../../src/handlers/base';
import { createMockContext, createMockPage, createTestConfig } from '../../helpers';
import { logger, metrics } from '../../../src/utils';

type RotateInstructionParams = {
  orientation?: DeviceOrientation;
  angle?: number;
};

function createRotateInstruction(params: RotateInstructionParams): { type: string; action: ReturnType<typeof create> } {
  return {
    type: 'rotate',
    action: create(ActionDefinitionSchema, {
      type: ActionType.ROTATE,
      params: {
        case: 'rotate',
        value: create(RotateParamsSchema, {
          orientation: params.orientation ?? DeviceOrientation.UNSPECIFIED,
          angle: params.angle,
        }),
      },
    }),
  };
}

describe('DeviceHandler', () => {
  let handler: DeviceHandler;

  beforeEach(() => {
    handler = new DeviceHandler();
  });

  it('rotates to portrait when currently landscape', async () => {
    const mockPage = createMockPage({
      viewportSize: jest.fn().mockReturnValue({ width: 1200, height: 800 }),
      setViewportSize: jest.fn().mockResolvedValue(undefined),
      evaluate: jest.fn().mockResolvedValue(undefined),
    });
    const mockContext = createMockContext();
    const context: HandlerContext = {
      page: mockPage,
      browserContext: mockContext,
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'device-session',
    };

    const instruction = {
      index: 0,
      nodeId: 'node-1',
      ...createRotateInstruction({ orientation: DeviceOrientation.PORTRAIT }),
      params: {},
    };

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(mockPage.setViewportSize).toHaveBeenCalledWith({ width: 800, height: 1200 });
    expect(mockPage.evaluate).toHaveBeenCalledWith(expect.any(Function));
    expect(result.extracted_data?.device).toEqual(
      expect.objectContaining({
        orientation: 'portrait',
        angle: 0,
        viewport: { width: 800, height: 1200 },
      })
    );
  });

  it('returns an error when viewport size is unavailable', async () => {
    const mockPage = createMockPage({
      viewportSize: jest.fn().mockReturnValue(null),
      setViewportSize: jest.fn().mockResolvedValue(undefined),
      evaluate: jest.fn().mockResolvedValue(undefined),
    });
    const mockContext = createMockContext();
    const context: HandlerContext = {
      page: mockPage,
      browserContext: mockContext,
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'device-session',
    };

    const instruction = {
      index: 0,
      nodeId: 'node-2',
      ...createRotateInstruction({ orientation: DeviceOrientation.LANDSCAPE }),
      params: {},
    };

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(false);
    expect(result.error?.code).toBe('NO_VIEWPORT');
  });

  it('applies rotation angle changes when invoked directly', async () => {
    const mockPage = createMockPage({
      viewportSize: jest.fn().mockReturnValue({ width: 800, height: 1200 }),
      setViewportSize: jest.fn().mockResolvedValue(undefined),
      evaluate: jest.fn().mockResolvedValue(undefined),
    });

    const result = await (handler as unknown as {
      handleRotate: (
        params: { orientation: string; angle?: number },
        page: typeof mockPage,
        logger: typeof logger
      ) => Promise<HandlerResult>;
    }).handleRotate({ orientation: '', angle: 180 }, mockPage, logger);

    expect(result.success).toBe(true);
    expect(mockPage.setViewportSize).toHaveBeenCalledWith({ width: 800, height: 1200 });
    expect(mockPage.evaluate).toHaveBeenCalledWith(expect.any(Function), 180);
    expect(result.extracted_data?.device).toEqual(
      expect.objectContaining({
        orientation: 'portrait',
        angle: 180,
      })
    );
  });

  it('prefers angle over orientation when both are provided', async () => {
    const mockPage = createMockPage({
      viewportSize: jest.fn().mockReturnValue({ width: 1200, height: 800 }),
      setViewportSize: jest.fn().mockResolvedValue(undefined),
      evaluate: jest.fn().mockResolvedValue(undefined),
    });
    const mockContext = createMockContext();
    const context: HandlerContext = {
      page: mockPage,
      browserContext: mockContext,
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'device-session',
    };

    const instruction = {
      index: 0,
      nodeId: 'node-3',
      ...createRotateInstruction({ orientation: DeviceOrientation.PORTRAIT, angle: 180 }),
      params: {},
    };

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(result.extracted_data?.device?.angle).toBe(180);
    expect(mockPage.evaluate).toHaveBeenCalledWith(expect.any(Function), 180);
  });

  it('rejects invalid rotation angles when invoked directly', async () => {
    const mockPage = createMockPage({
      viewportSize: jest.fn().mockReturnValue({ width: 800, height: 1200 }),
      setViewportSize: jest.fn().mockResolvedValue(undefined),
      evaluate: jest.fn().mockResolvedValue(undefined),
    });

    const result = await (handler as unknown as {
      handleRotate: (
        params: { orientation: string; angle?: number },
        page: typeof mockPage,
        logger: typeof logger
      ) => Promise<HandlerResult>;
    }).handleRotate({ orientation: '', angle: 45 }, mockPage, logger);

    expect(result.success).toBe(false);
    expect(result.error?.code).toBe('INVALID_ANGLE');
  });
});
