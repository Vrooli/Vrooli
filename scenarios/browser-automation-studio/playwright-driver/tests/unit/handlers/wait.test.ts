import {
  createTypedInstruction,
  createMockPage,
  createMockContext,
  createTestConfig,
} from '../../helpers';
import { WaitHandler } from '../../../src/handlers/wait';
import type { HandlerContext } from '../../../src/handlers/base';
import { logger, metrics } from '../../../src/utils';

describe('WaitHandler', () => {
  let handler: WaitHandler;
  let mockPage: ReturnType<typeof createMockPage>;
  let context: HandlerContext;

  beforeEach(() => {
    handler = new WaitHandler();
    mockPage = createMockPage();
    context = {
      page: mockPage,
      browserContext: createMockContext(),
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'test-session',
    };
  });

  it('should wait for selector', async () => {
    const instruction = createTypedInstruction('wait', { selector: '#element' }, { nodeId: 'node-1' });

    const result = await handler.execute(instruction, context);

    const [selector, options] = mockPage.waitForSelector.mock.calls[0] ?? [];
    expect(selector).toBe('#element');
    expect(options).toEqual(expect.any(Object));
    expect(result.success).toBe(true);
  });

  it('exports the observed experience lifecycle state for selector waits', async () => {
    const instruction = createTypedInstruction('wait', { selector: '[data-experience-surface="results"]' }, { nodeId: 'node-1' });
    const locator = mockPage.locator('[data-experience-surface="results"]');
    const first = locator.first() as unknown as { evaluate: jest.Mock };
    first.evaluate = jest.fn().mockResolvedValue({
      tagName: 'section',
      attributes: {
        'data-experience-surface': 'results',
        'data-experience-state': 'partial',
      },
      isVisible: true,
      isEnabled: true,
    });

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(result.extracted_data).toEqual({
      experience_surface_id: 'results',
      experience_surface_state: 'partial',
    });
  });

  it('should wait for timeout when no selector', async () => {
    const instruction = createTypedInstruction('wait', { ms: 1000 }, { nodeId: 'node-1' });

    const result = await handler.execute(instruction, context);

    const [ms] = mockPage.waitForTimeout.mock.calls[0] ?? [];
    expect(ms).toBe(1000);
    expect(result.success).toBe(true);
  });
});
