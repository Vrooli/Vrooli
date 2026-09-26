import {
  createTypedInstruction,
  createMockPage,
  createMockContext,
  createTestConfig,
} from '../../helpers';
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
      browserContext: createMockContext(),
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'test-session',
    };
  });

  it('should handle download', async () => {
    const instruction = createTypedInstruction(
      'download',
      {
        selector: '#download-link',
      },
      { nodeId: 'node-1' }
    );

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
  it('performs repeated new instructions and allocates distinct files even at the same clock time', async () => {
    const download = {
      suggestedFilename: () => 'same.txt',
      saveAs: jest.fn().mockResolvedValue(undefined),
      url: () => 'https://example.com/same.txt',
    };
    mockPage.waitForEvent = jest.fn().mockResolvedValue(download);
    const now = jest.spyOn(Date, 'now').mockReturnValue(1000);
    try {
      const instruction = createTypedInstruction('download', { selector: '#download-link' });
      const first = await handler.execute(instruction, context);
      const second = await handler.execute(instruction, context);
      expect(first.success).toBe(true);
      expect(second.success).toBe(true);
      expect(mockPage.click).toHaveBeenCalledTimes(2);
      expect(first.extracted_data?.download_path).not.toBe(second.extracted_data?.download_path);
      expect(download.saveAs).toHaveBeenCalledTimes(2);
    } finally {
      now.mockRestore();
    }
  });

  it.each([
    ['URL navigation', { url: 'https://example.com/file' }, 'net::ERR_NAME_NOT_RESOLVED'],
    ['selector click', { selector: '#download-link' }, 'net::ERR_ABORTED'],
  ])('preserves unexpected errors from %s', async (_label, params, message) => {
    const download = { saveAs: jest.fn() };
    mockPage.waitForEvent = jest.fn().mockResolvedValue(download);
    mockPage.goto = jest.fn().mockRejectedValue(new Error(message));
    mockPage.click = jest.fn().mockRejectedValue(new Error(message));
    const result = await handler.execute(createTypedInstruction('download', params), context);
    expect(result.success).toBe(false);
    expect(result.error?.message).toBe(message);
    expect(download.saveAs).not.toHaveBeenCalled();
  });
});
