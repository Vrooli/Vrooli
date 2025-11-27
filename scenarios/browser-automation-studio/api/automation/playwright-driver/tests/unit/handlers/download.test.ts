import { DownloadHandler } from '../../../src/handlers/download';
import type { CompiledInstruction, HandlerContext } from '../../../src/types';
import { createMockPage, createTestConfig } from '../../helpers';
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
    const instruction: CompiledInstruction = {
      type: 'download',
      params: { selector: '#download-link' },
      node_id: 'node-1',
    };

    const mockDownload = {
      path: jest.fn().mockResolvedValue('/tmp/download.pdf'),
      suggestedFilename: jest.fn().mockReturnValue('file.pdf'),
    };

    mockPage.waitForEvent = jest.fn().mockResolvedValue(mockDownload);

    const result = await handler.execute(instruction, context);

    expect(result.success).toBe(true);
    expect(result.extracted_data?.downloadPath).toBe('/tmp/download.pdf');
  });
});
