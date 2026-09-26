import { createMockContext, createMockPage, createTestConfig } from '../../helpers';
import { GestureHandler } from '../../../src/handlers/gesture';
import type { HandlerContext } from '../../../src/handlers/base';
import { logger, metrics } from '../../../src/utils';
import { create } from '@bufbuild/protobuf';
import {
  ActionDefinitionSchema,
  ActionType,
  DragDropParamsSchema,
  GestureParamsSchema,
  GestureType,
  SwipeDirection,
} from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';

jest.mock('../../../src/telemetry', () => ({
  captureElementContext: jest.fn().mockResolvedValue({
    selector: '#source',
    boundingBox: { x: 10, y: 20, width: 40, height: 20 },
  }),
}));

jest.mock('../../../src/handlers/behavior-utils', () => ({
  getBehaviorFromContext: jest.fn().mockReturnValue(null),
  applyPreActionDelay: jest.fn().mockResolvedValue(undefined),
  applyPostActionPause: jest.fn().mockResolvedValue(undefined),
  sleep: jest.fn().mockResolvedValue(undefined),
}));

function createDragDropInstruction(params: {
  sourceSelector: string;
  targetSelector?: string;
  offsetX?: number;
  offsetY?: number;
  targetOffsetX?: number;
  targetOffsetY?: number;
  steps?: number;
}): { type: string; action: ReturnType<typeof create> } {
  return {
    type: 'drag-drop',
    action: create(ActionDefinitionSchema, {
      type: ActionType.DRAG_DROP,
      params: {
        case: 'dragDrop',
        value: create(DragDropParamsSchema, {
          sourceSelector: params.sourceSelector,
          targetSelector: params.targetSelector,
          offsetX: params.offsetX,
          offsetY: params.offsetY,
          targetOffsetX: params.targetOffsetX,
          targetOffsetY: params.targetOffsetY,
          steps: params.steps,
        }),
      },
    }),
  };
}

function createGestureInstruction(params: {
  gestureType: GestureType;
  direction?: SwipeDirection;
  selector?: string;
  distance?: number;
  scale?: number;
  durationMs?: number;
  steps?: number;
  stepDelayMs?: number;
  traceLabel?: string;
  idleAfterMs?: number;
  wheelDeltaY?: number;
  ctrlKey?: boolean;
}): { type: string; action: ReturnType<typeof create> } {
  return {
    type: 'gesture',
    action: create(ActionDefinitionSchema, {
      type: ActionType.GESTURE,
      params: {
        case: 'gesture',
        value: create(GestureParamsSchema, {
          gestureType: params.gestureType,
          direction: params.direction,
          selector: params.selector,
          distance: params.distance,
          scale: params.scale,
          durationMs: params.durationMs,
          steps: params.steps,
          stepDelayMs: params.stepDelayMs,
          traceLabel: params.traceLabel,
          idleAfterMs: params.idleAfterMs,
          wheelDeltaY: params.wheelDeltaY,
          ctrlKey: params.ctrlKey,
        }),
      },
    }),
  };
}

describe('GestureHandler', () => {
  let handler: GestureHandler;
  let mockPage: ReturnType<typeof createMockPage>;
  let context: HandlerContext;

  beforeEach(() => {
    handler = new GestureHandler();
    mockPage = createMockPage({
      mouse: {
        move: jest.fn().mockResolvedValue(undefined),
        down: jest.fn().mockResolvedValue(undefined),
        up: jest.fn().mockResolvedValue(undefined),
        click: jest.fn().mockResolvedValue(undefined),
        wheel: jest.fn().mockResolvedValue(undefined),
      } as unknown as ReturnType<typeof createMockPage>['mouse'],
      keyboard: {
        down: jest.fn().mockResolvedValue(undefined),
        up: jest.fn().mockResolvedValue(undefined),
      } as unknown as ReturnType<typeof createMockPage>['keyboard'],
    });

    const mockElement = {
      boundingBox: jest.fn().mockResolvedValue({ x: 10, y: 20, width: 40, height: 20 }),
    };
    mockPage.waitForSelector = jest.fn().mockResolvedValue(mockElement as never);

    context = {
      page: mockPage,
      browserContext: createMockContext(),
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'gesture-session',
    };
  });

  it('performs drag-drop by offset with linear movement', async () => {
    const instruction = {
      index: 0,
      nodeId: 'node-1',
      ...createDragDropInstruction({ sourceSelector: '#source', offsetX: 20, offsetY: -10 }),
      params: {},
    };

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(mockPage.mouse.down).toHaveBeenCalled();
    expect(mockPage.mouse.up).toHaveBeenCalled();
    expect(result.extracted_data?.target?.position).toEqual({ x: 50, y: 20 });
  });

  it('uses HTML5 drag events for draggable source and target selectors', async () => {
    mockPage.evaluate.mockResolvedValue(true);
    const instruction = {
      index: 0,
      nodeId: 'node-html5',
      ...createDragDropInstruction({ sourceSelector: '#palette-card', targetSelector: '#canvas' }),
      params: {},
    };

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(result.extracted_data?.dragDrop?.mode).toBe('html5');
    expect(mockPage.mouse.down).not.toHaveBeenCalled();
    expect(mockPage.evaluate).toHaveBeenCalledWith(
      expect.any(Function),
      expect.objectContaining({ sourceX: 30, sourceY: 30, targetX: 30, targetY: 30 }),
    );
  });

  it('uses bounded target offsets for canvas-style drops', async () => {
    const sourceElement = { boundingBox: jest.fn().mockResolvedValue({ x: 10, y: 20, width: 40, height: 20 }) };
    const canvasElement = { boundingBox: jest.fn().mockResolvedValue({ x: 100, y: 100, width: 400, height: 300 }) };
    mockPage.waitForSelector = jest.fn()
      .mockResolvedValueOnce(sourceElement as never)
      .mockResolvedValueOnce(canvasElement as never);
    mockPage.evaluate.mockResolvedValue(true);
    const instruction = {
      index: 0,
      nodeId: 'node-target-offset',
      ...createDragDropInstruction({
        sourceSelector: '#palette-card',
        targetSelector: '#canvas',
        targetOffsetX: 100,
        targetOffsetY: 50,
      }),
      params: {},
    };

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(result.extracted_data?.dragDrop?.target).toEqual({ x: 400, y: 300 });
    expect(mockPage.evaluate).toHaveBeenCalledWith(
      expect.any(Function),
      expect.objectContaining({ targetX: 400, targetY: 300 }),
    );
  });

  it('rejects target offsets outside the target bounds', async () => {
    const sourceElement = { boundingBox: jest.fn().mockResolvedValue({ x: 10, y: 20, width: 40, height: 20 }) };
    const canvasElement = { boundingBox: jest.fn().mockResolvedValue({ x: 100, y: 100, width: 400, height: 300 }) };
    mockPage.waitForSelector = jest.fn()
      .mockResolvedValueOnce(sourceElement as never)
      .mockResolvedValueOnce(canvasElement as never);
    const instruction = {
      index: 0,
      nodeId: 'node-invalid-target-offset',
      ...createDragDropInstruction({ sourceSelector: '#palette-card', targetSelector: '#canvas', targetOffsetX: 500 }),
      params: {},
    };

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(false);
    expect(result.error?.code).toBe('INVALID_TARGET_OFFSET');
  });

  it('returns an error when drag-drop has no target or offset', async () => {
    const instruction = {
      index: 0,
      nodeId: 'node-2',
      ...createDragDropInstruction({ sourceSelector: '#source' }),
      params: {},
    };

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(false);
    expect(result.error?.code).toBe('MISSING_PARAMS');
  });

  it('rejects swipe with missing direction', async () => {
    const instruction = {
      index: 0,
      nodeId: 'node-3',
      ...createGestureInstruction({ gestureType: GestureType.SWIPE }),
      params: {},
    };

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(false);
    expect(result.error?.code).toBe('INVALID_DIRECTION');
  });

  it('applies zoom to the page when no selector is provided', async () => {
    const instruction = {
      index: 0,
      nodeId: 'node-4',
      ...createGestureInstruction({ gestureType: GestureType.ZOOM, scale: 1.2, steps: 3, wheelDeltaY: -120 }),
      params: {},
    };

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(mockPage.mouse.wheel).toHaveBeenCalledTimes(3);
    expect(result.extracted_data?.zoom?.applied).toBe('wheel');
  });

  it('wraps sustained gestures with trace markers and cadence', async () => {
    const instruction = {
      index: 0,
      nodeId: 'node-5',
      ...createGestureInstruction({
        gestureType: GestureType.SWIPE,
        direction: SwipeDirection.RIGHT,
        selector: '#canvas',
        distance: 320,
        steps: 4,
        stepDelayMs: 16,
        traceLabel: 'graph-pan',
      }),
      params: {},
    };

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(mockPage.$).toHaveBeenCalledWith('#canvas');
    expect(mockPage.evaluate).toHaveBeenCalledWith(expect.any(Function), 'bas.gesture.graph-pan.start');
    expect(mockPage.evaluate).toHaveBeenCalledWith(expect.any(Function), 'bas.gesture.graph-pan.end');
    expect(mockPage.mouse.move).toHaveBeenCalledWith(expect.any(Number), expect.any(Number));
    expect(result.extracted_data?.swipe?.steps).toBe(4);
    expect(result.extracted_data?.swipe?.traceLabel).toBe('graph-pan');
  });

  it('holds Control while performing wheel zoom when requested', async () => {
    const instruction = {
      index: 0,
      nodeId: 'node-6',
      ...createGestureInstruction({
        gestureType: GestureType.ZOOM,
        selector: '#canvas',
        steps: 2,
        wheelDeltaY: -240,
        ctrlKey: true,
        traceLabel: 'graph-wheel-zoom',
      }),
      params: {},
    };

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(mockPage.keyboard.down).toHaveBeenCalledWith('Control');
    expect(mockPage.mouse.wheel).toHaveBeenCalledTimes(2);
    expect(mockPage.mouse.wheel).toHaveBeenCalledWith(0, -240);
    expect(mockPage.keyboard.up).toHaveBeenCalledWith('Control');
    expect(mockPage.evaluate).toHaveBeenCalledWith(expect.any(Function), 'bas.gesture.graph-wheel-zoom.start');
    expect(mockPage.evaluate).toHaveBeenCalledWith(expect.any(Function), 'bas.gesture.graph-wheel-zoom.end');
  });
});
