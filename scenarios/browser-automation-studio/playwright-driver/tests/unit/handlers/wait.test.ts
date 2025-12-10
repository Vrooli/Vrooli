import { createTestInstruction, createMockPage, createTestConfig } from '../../helpers';
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
      context: {} as any,
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'test-session',
    };
  });

  it('should wait for selector', async () => {
    const instruction = createTestInstruction({
      type: 'wait',
      params: { selector: '#element' },
      node_id: 'node-1',
    });

    const result = await handler.execute(instruction, context);

    expect(mockPage.waitForSelector).toHaveBeenCalledWith('#element', expect.any(Object));
    expect(result.success).toBe(true);
  });

  it('should wait for timeout when no selector', async () => {
    const instruction = createTestInstruction({
      type: 'wait',
      params: { timeoutMs: 1000 },
      node_id: 'node-1',
    });

    const result = await handler.execute(instruction, context);

    expect(mockPage.waitForTimeout).toHaveBeenCalledWith(1000);
    expect(result.success).toBe(true);
  });
});
