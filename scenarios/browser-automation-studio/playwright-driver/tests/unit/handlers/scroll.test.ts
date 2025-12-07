import { createTestInstruction, createMockPage, createTestConfig } from '../../helpers';
import { ScrollHandler } from '../../../src/handlers/scroll';
import type { HandlerContext } from '../../../src/types';
import { logger, metrics } from '../../../src/utils';

describe('ScrollHandler', () => {
  let handler: ScrollHandler;
  let mockPage: ReturnType<typeof createMockPage>;
  let context: HandlerContext;

  beforeEach(() => {
    handler = new ScrollHandler();
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

  it('should scroll to origin', async () => {
    const instruction = createTestInstruction({
      type: 'scroll',
      params: { x: 0, y: 0 },
      node_id: 'node-1',
    });

    mockPage.evaluate.mockResolvedValue(undefined);

    const result = await handler.execute(instruction, context);

    expect(mockPage.evaluate).toHaveBeenCalled();
    expect(result.success).toBe(true);
  });

  it('should scroll to coordinates', async () => {
    const instruction = createTestInstruction({
      type: 'scroll',
      params: { x: 0, y: 500 },
      node_id: 'node-1',
    });

    mockPage.evaluate.mockResolvedValue(undefined);

    const result = await handler.execute(instruction, context);

    expect(mockPage.evaluate).toHaveBeenCalled();
    expect(result.success).toBe(true);
  });
});
