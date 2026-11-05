import { ExtractionHandler } from '../../../src/handlers/extraction';
import { createMockPage } from '../../helpers/playwright-mocks';
import { createTypedInstruction } from '../../helpers/instruction-factory';

// Arbitrary expressions can commit effects before their result becomes unavailable.
describe('evaluate during navigation', () => {
  const CONTEXT_DESTROYED =
    'page.evaluate: Execution context was destroyed, most likely because of a navigation';

  function setup() {
    const handler = new ExtractionHandler();
    const page = createMockPage();
    const logger = { info: jest.fn(), debug: jest.fn(), warn: jest.fn(), error: jest.fn() };
    const context = {
      page,
      logger,
      metrics: {},
      sessionId: 'test-session',
      config: { execution: {} },
    } as never;
    const instruction = createTypedInstruction(
      'evaluate',
      { expression: 'document.title' },
      { nodeId: 'read-title' }
    );
    return { handler, page, instruction, context };
  }

  it.each([
    CONTEXT_DESTROYED,
    'page.evaluate: Timeout 1000ms exceeded',
    'page.evaluate: Target page, context or browser has been closed',
    'page.evaluate: error after committed effect',
  ])('does not repeat an uncertain evaluation: %s', async (message) => {
    const { handler, page, instruction, context } = setup();
    page.evaluate = jest.fn().mockRejectedValue(new Error(message)) as never;
    page.waitForLoadState = jest.fn().mockResolvedValue(undefined) as never;

    const result = await handler.execute(instruction as never, context);

    expect(result.success).toBe(false);
    expect(result.error?.message).toBe(message);
    expect(result.error?.retryable).toBe(false);
    expect(page.evaluate).toHaveBeenCalledTimes(1);
    expect(page.waitForLoadState).not.toHaveBeenCalled();
  });

  it('preserves ordinary read-only extraction error classification', async () => {
    const { handler, page, context } = setup();
    page.textContent = jest.fn().mockRejectedValue(new Error(CONTEXT_DESTROYED)) as never;
    const result = await handler.execute(
      createTypedInstruction('extract', { selector: 'h1' }),
      context
    );
    expect(result.error?.retryable).toBe(true);
  });
});
