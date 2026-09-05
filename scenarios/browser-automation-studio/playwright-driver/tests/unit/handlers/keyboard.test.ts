import { create } from '@bufbuild/protobuf';
import {
  ActionDefinitionSchema,
  KeyboardParamsSchema,
  ShortcutParamsSchema,
  ActionType,
  KeyAction,
} from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import { KeyboardHandler } from '../../../src/handlers/keyboard';
import type { HandlerContext } from '../../../src/handlers/base';
import type { HandlerInstruction } from '../../../src/types';
import { createMockContext, createTestConfig } from '../../helpers';
import { logger, metrics } from '../../../src/utils';

function buildKeyboardInstruction(params: {
  key?: string;
  keys?: string[];
  action?: KeyAction;
}): HandlerInstruction {
  const action = create(ActionDefinitionSchema, {
    type: ActionType.KEYBOARD,
    params: {
      case: 'keyboard',
      value: create(KeyboardParamsSchema, {
        key: params.key,
        keys: params.keys ?? [],
        action: params.action,
      }),
    },
  });

  return {
    index: 0,
    nodeId: 'node-1',
    type: 'keyboard',
    params: params as Record<string, unknown>,
    action,
  };
}

function buildShortcutInstruction(shortcut?: string): HandlerInstruction {
  const action = create(ActionDefinitionSchema, {
    type: ActionType.SHORTCUT,
    params: {
      case: 'shortcut',
      value: create(ShortcutParamsSchema, {
        shortcut: shortcut ?? '',
      }),
    },
  });

  return {
    index: 0,
    nodeId: 'node-1',
    type: 'shortcut',
    params: { shortcut },
    action,
  };
}

describe('KeyboardHandler', () => {
  let handler: KeyboardHandler;
  let context: HandlerContext;
  const keyboard = {
    press: jest.fn().mockResolvedValue(undefined),
    down: jest.fn().mockResolvedValue(undefined),
    up: jest.fn().mockResolvedValue(undefined),
  };

  beforeEach(() => {
    handler = new KeyboardHandler();
    context = {
      page: { keyboard } as HandlerContext['page'],
      browserContext: createMockContext(),
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'test-session',
    };

    keyboard.press.mockClear();
    keyboard.down.mockClear();
    keyboard.up.mockClear();
  });

  it('presses all keys in a sequence', async () => {
    const instruction = buildKeyboardInstruction({
      keys: ['A', 'B'],
      action: KeyAction.PRESS,
    });

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(keyboard.press).toHaveBeenCalledTimes(2);
    expect(keyboard.press).toHaveBeenNthCalledWith(1, 'A');
    expect(keyboard.press).toHaveBeenNthCalledWith(2, 'B');
  });

  it('handles key down and up actions', async () => {
    const downInstruction = buildKeyboardInstruction({
      key: 'Shift',
      action: KeyAction.DOWN,
    });

    const upInstruction = buildKeyboardInstruction({
      key: 'Shift',
      action: KeyAction.UP,
    });

    expect((await handler.execute(downInstruction, context)).success).toBe(true);
    expect((await handler.execute(upInstruction, context)).success).toBe(true);

    expect(keyboard.down).toHaveBeenCalledWith('Shift');
    expect(keyboard.up).toHaveBeenCalledWith('Shift');
  });

  it('returns an error when no key is provided', async () => {
    const instruction = buildKeyboardInstruction({});

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(false);
    expect(result.error?.code).toBe('MISSING_PARAM');
  });

  it('executes keyboard shortcuts', async () => {
    const instruction = buildShortcutInstruction('Control+A');

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(keyboard.press).toHaveBeenCalledWith('Control+A');
  });

  it('returns an error when shortcut is missing', async () => {
    const instruction = buildShortcutInstruction('');

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(false);
    expect(result.error?.code).toBe('MISSING_PARAM');
  });

  it('rejects unsupported instruction types', async () => {
    const instruction: HandlerInstruction = {
      index: 0,
      nodeId: 'node-2',
      action: create(ActionDefinitionSchema, {
        type: ActionType.UNSPECIFIED,
      }),
    };

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(false);
    expect(result.error?.code).toBe('UNSUPPORTED_TYPE');
  });

  it('normalizes unexpected execution errors', async () => {
    const instruction = buildKeyboardInstruction({
      key: 'Enter',
      action: KeyAction.PRESS,
    });

    keyboard.press.mockRejectedValueOnce(new Error('boom'));

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(false);
    expect(result.error?.code).toBe('PLAYWRIGHT_ERROR');
  });
});
