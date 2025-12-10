import { createTestInstruction, createMockPage, createTestConfig } from '../../helpers';
import { ExtractionHandler } from '../../../src/handlers/extraction';
import type { HandlerContext } from '../../../src/handlers/base';
import { logger, metrics } from '../../../src/utils';

describe('ExtractionHandler', () => {
  let handler: ExtractionHandler;
  let mockPage: ReturnType<typeof createMockPage>;
  let context: HandlerContext;

  beforeEach(() => {
    handler = new ExtractionHandler();
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

  it('should extract text from element', async () => {
    const instruction = createTestInstruction({
      type: 'extract',
      params: { selector: '#content' },
      node_id: 'node-1',
    });

    mockPage.textContent.mockResolvedValue('Extracted text');

    const result = await handler.execute(instruction, context);

    expect(mockPage.textContent).toHaveBeenCalledWith('#content', expect.any(Object));
    expect(result.success).toBe(true);
    expect(result.extracted_data).toEqual({ '#content': 'Extracted text' });
  });

  it('should handle empty text content', async () => {
    const instruction = createTestInstruction({
      type: 'extract',
      params: { selector: '#empty' },
      node_id: 'node-1',
    });

    mockPage.textContent.mockResolvedValue(null);

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(result.extracted_data).toEqual({ '#empty': '' });
  });

  it('should evaluate JavaScript', async () => {
    const instruction = createTestInstruction({
      type: 'evaluate',
      params: { script: 'return document.title' },
      node_id: 'node-1',
    });

    mockPage.evaluate.mockResolvedValue('Page Title');

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(result.extracted_data).toEqual({ result: 'Page Title' });
  });
});
