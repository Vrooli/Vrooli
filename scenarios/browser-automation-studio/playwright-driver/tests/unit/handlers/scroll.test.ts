import {
  createTypedInstruction,
  createMockPage,
  createMockContext,
  createTestConfig,
} from '../../helpers';
import { ScrollHandler } from '../../../src/handlers/scroll';
import type { HandlerContext } from '../../../src/handlers/base';
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
      browserContext: createMockContext(),
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'test-session',
    };
  });

  it('releases its target after a browser-side failure', async () => {
    const dispose = jest.fn().mockResolvedValue(undefined);
    const evaluate = jest.fn().mockRejectedValue(new Error('document was replaced'));
    mockPage.waitForSelector.mockResolvedValue({ evaluate, dispose } as never);
    const instruction = createTypedInstruction('scroll', { selector: '#pane', x: 20, y: 30 });
    const result = await handler.execute(instruction, context);
    expect(result.success).toBe(false);
    expect(dispose).toHaveBeenCalledTimes(1);
  });
});
