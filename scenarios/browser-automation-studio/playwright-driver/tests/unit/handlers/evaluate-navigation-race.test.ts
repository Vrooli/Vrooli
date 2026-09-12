import { ExtractionHandler } from '../../../src/handlers/extraction';
import { createMockPage } from '../../helpers/playwright-mocks';
import { createTypedInstruction } from '../../helpers/instruction-factory';

/**
 * A navigation that commits mid-evaluate destroys the JS execution context.
 * That is transient, but it surfaced as a hard failure because the executor
 * defaults to MaxAttempts=1, so the driver's `retryable` flag never gets acted
 * on unless a workflow declares a per-node resilience block.
 *
 * These pin the retry so the class of flake stays fixed — and pin the limits so
 * the retry never hides a real failure.
 */
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

  it('retries once and succeeds when the context is destroyed', async () => {
    const { handler, page, instruction, context } = setup();
    page.evaluate = jest
      .fn()
      .mockRejectedValueOnce(new Error(CONTEXT_DESTROYED))
      .mockResolvedValueOnce('recovered') as never;
    page.waitForLoadState = jest.fn().mockResolvedValue(undefined) as never;

    const result = await handler.execute(instruction as never, context);

    expect(result.success).toBe(true);
    expect(result.extracted_data?.result).toBe('recovered');
    expect(page.evaluate).toHaveBeenCalledTimes(2);
    // The retry must wait for the new document, or it races the same way.
    expect(page.waitForLoadState).toHaveBeenCalled();
  });

  it('gives up after a second destruction rather than looping', async () => {
    const { handler, page, instruction, context } = setup();
    page.evaluate = jest.fn().mockRejectedValue(new Error(CONTEXT_DESTROYED)) as never;
    page.waitForLoadState = jest.fn().mockResolvedValue(undefined) as never;

    const result = await handler.execute(instruction as never, context);

    expect(result.success).toBe(false);
    expect(page.evaluate).toHaveBeenCalledTimes(2);
  });

  it('does not retry unrelated evaluate failures', async () => {
    const { handler, page, instruction, context } = setup();
    page.evaluate = jest.fn().mockRejectedValue(new SyntaxError('Unexpected token )')) as never;

    const result = await handler.execute(instruction as never, context);

    expect(result.success).toBe(false);
    expect(page.evaluate).toHaveBeenCalledTimes(1);
  });
});
