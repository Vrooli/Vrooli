import { createTestInstruction } from '../../helpers';
import { ExtractionHandler } from '../../../src/handlers/extraction';
import type { CompiledInstruction, HandlerContext } from '../../../src/types';
import { createMockPage, createTestConfig } from '../../helpers';
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
      params: { selector: '#content', attribute: 'text' },
      node_id: 'node-1',
    };

    const mockLocator = {
      textContent: jest.fn().mockResolvedValue('Extracted text'),
    };
    mockPage.locator.mockReturnValue(mockLocator as any);

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(result.extracted_data).toEqual({ value: 'Extracted text' });
  });

  it('should extract attribute value', async () => {
    const instruction = createTestInstruction({
      type: 'extract',
      params: { selector: '#link', attribute: 'href' },
      node_id: 'node-1',
    };

    const mockLocator = {
      getAttribute: jest.fn().mockResolvedValue('https://example.com'),
    };
    mockPage.locator.mockReturnValue(mockLocator as any);

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(result.extracted_data).toEqual({ value: 'https://example.com' });
  });

  it('should evaluate JavaScript', async () => {
    const instruction = createTestInstruction({
      type: 'evaluate',
      params: { script: 'return document.title' },
      node_id: 'node-1',
    };

    mockPage.evaluate.mockResolvedValue('Page Title');

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(result.extracted_data).toEqual({ result: 'Page Title' });
  });
});
