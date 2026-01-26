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
      ...createGestureInstruction({ gestureType: GestureType.ZOOM, scale: 1.2 }),
      params: {},
    };

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(mockPage.evaluate).toHaveBeenCalled();
    expect(result.extracted_data?.zoom?.applied).toBe('page');
  });
});
