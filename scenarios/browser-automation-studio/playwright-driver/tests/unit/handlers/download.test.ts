import { createTestInstruction, createMockPage, createTestConfig } from '../../helpers';
import { DownloadHandler } from '../../../src/handlers/download';
import type { HandlerContext } from '../../../src/handlers/base';
import { logger, metrics } from '../../../src/utils';

describe('DownloadHandler', () => {
  let handler: DownloadHandler;
  let mockPage: ReturnType<typeof createMockPage>;
  let context: HandlerContext;

  beforeEach(() => {
    handler = new DownloadHandler();
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

  it('should handle download', async () => {
    const instruction = createTestInstruction({
      type: 'download',
      params: { selector: '#download-link' },
      node_id: 'node-1',
    });

    const mockDownload = {
      suggestedFilename: jest.fn().mockReturnValue('file.pdf'),
      saveAs: jest.fn().mockResolvedValue(undefined),
      url: jest.fn().mockReturnValue('https://example.com/file.pdf'),
    };

    mockPage.waitForEvent = jest.fn().mockResolvedValue(mockDownload);

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(result.extracted_data?.download_path).toBeDefined();
    expect(result.extracted_data?.filename).toBe('file.pdf');
  });
});
