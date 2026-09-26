import { create } from '@bufbuild/protobuf';
import { errors } from 'rebrowser-playwright';
import { ActionDefinitionSchema, ActionType, ConditionalParamsSchema, ConditionalType, type ConditionalParams } from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import { ConditionalHandler } from '../../../src/handlers/conditional';
import { getActionType } from '../../../src/proto';
import { createMockPage, createMockContext, createTestConfig, createTestInstruction } from '../../helpers';
import { logger, metrics } from '../../../src/utils';

const instruction = (conditionType: ConditionalType, params: Partial<ConditionalParams>) => createTestInstruction({
  type: 'conditional', action: create(ActionDefinitionSchema, { type: ActionType.CONDITIONAL,
    params: { case: 'conditional', value: create(ConditionalParamsSchema, { ...params, conditionType }) } }),
});

describe('ConditionalHandler [REQ:BAS-RH-J24]', () => {
  const handler = new ConditionalHandler();
  let page: ReturnType<typeof createMockPage>;
  let context: Parameters<ConditionalHandler['execute']>[1];
  beforeEach(() => {
    page = createMockPage();
    context = { page, browserContext: createMockContext(), config: createTestConfig(), logger, metrics, sessionId: 'conditional-test' };
    page.evaluate.mockImplementation((fn: any, arg: any) => fn(arg));
  });

  it.each(['true', 'return true;', 'Promise.resolve(true)', 'return Promise.resolve(true);', 'false', 'return false;', '0', 'null'])
  ('evaluates expression/body syntax %s', async expression => {
    const expected = expression.includes('true');
    for (const negate of [false, true]) {
      const input = instruction(ConditionalType.EXPRESSION, { expression, negate });
      expect(getActionType(input)).toBe('conditional');
      const result = await handler.execute(input, context);
      expect(result.success).toBe(true);
      expect(result.condition).toMatchObject({ outcome: expected !== negate, negated: negate, expression });
    }
  });

  it.each([false, true])('does not reinterpret a runtime SyntaxError, negate=%s', async negate => {
    const expression = '(() => { globalThis.__basConditionalEffects++; throw new SyntaxError("runtime marker"); })()';
    Object.assign(globalThis, { __basConditionalEffects: 0 });
    try {
      const result = await handler.execute(instruction(ConditionalType.EXPRESSION, { expression, negate }), context);
      expect(result.success).toBe(false);
      expect(result.error).toMatchObject({ retryable: false });
      expect(result.error?.message).toContain('runtime marker');
      expect(result.condition).toBeUndefined();
      expect(Reflect.get(globalThis, '__basConditionalEffects')).toBe(1);
    } finally { Reflect.deleteProperty(globalThis, '__basConditionalEffects'); }
  });

  it.each([false, true].flatMap(present => [false, true].map(negate => ({ present, negate }))))
  ('observes timed presence=$present, negate=$negate', async ({ present, negate }) => {
    const dispose = jest.fn().mockResolvedValue(undefined);
    page.waitForFunction = present ? jest.fn().mockResolvedValue({ dispose })
      : jest.fn().mockRejectedValue(new errors.TimeoutError('presence timed out'));
    const result = await handler.execute(instruction(ConditionalType.ELEMENT, { selector: '#ready', negate, timeoutMs: 80, pollIntervalMs: 20 }), context);
    expect(result.success).toBe(true);
    expect(result.condition).toMatchObject({ outcome: present !== negate, actual: present, negated: negate, selector: '#ready' });
    expect(page.waitForFunction).toHaveBeenCalledWith(expect.any(Function), '#ready', { timeout: 80, polling: 20 });
    expect(dispose).toHaveBeenCalledTimes(present ? 1 : 0);
  });

  it.each(['closed page', 'invalid selector'])('keeps %s as evaluation failure under negation', async message => {
    page.waitForFunction = jest.fn().mockRejectedValue(new Error(message));
    const result = await handler.execute(instruction(ConditionalType.ELEMENT, { selector: '[', negate: true }), context);
    expect(result.success).toBe(false);
    expect(result.error?.message).toContain(message);
    expect(result.condition).toBeUndefined();
  });

  it.each([false, true])('samples immediately with timeout zero: %s', async present => {
    (page.locator('#ready').count as jest.Mock).mockResolvedValue(present ? 1 : 0);
    page.waitForFunction = jest.fn();
    const result = await handler.execute(instruction(ConditionalType.ELEMENT, { selector: '#ready', timeoutMs: 0 }), context);
    expect(result.condition?.outcome).toBe(present);
    expect(page.waitForFunction).not.toHaveBeenCalled();
  });

  it.each([
    [ConditionalType.EXPRESSION, {}], [ConditionalType.ELEMENT, {}],
    [ConditionalType.ELEMENT, { selector: '#ready', timeoutMs: -1 }],
    [ConditionalType.ELEMENT, { selector: '#ready', pollIntervalMs: 0 }],
    [ConditionalType.VARIABLE, { variable: 'answer' }], [ConditionalType.UNSPECIFIED, {}],
  ] as const)('rejects invalid or executor-owned predicate %s %j', async (type, params) => {
    const result = await handler.execute(instruction(type, params), context);
    expect(result.success).toBe(false);
    expect(result.error?.code).toBe('INVALID_INSTRUCTION');
    expect(result.condition).toBeUndefined();
  });
});
