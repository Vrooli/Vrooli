import { createTestInstruction, createMockPage, createTestConfig } from '../../helpers';
import { UploadHandler } from '../../../src/handlers/upload';
import type { HandlerContext } from '../../../src/types';
import { logger, metrics } from '../../../src/utils';

describe('UploadHandler', () => {
  let handler: UploadHandler;
  let mockPage: ReturnType<typeof createMockPage>;
  let context: HandlerContext;

  beforeEach(() => {
    handler = new UploadHandler();
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

  it('should upload file', async () => {
    const instruction = createTestInstruction({
      type: 'uploadfile',
      params: { selector: '#file-input', filePath: '/path/to/file.txt' },
      node_id: 'node-1',
    });

    const result = await handler.execute(instruction, context);

    expect(mockPage.setInputFiles).toHaveBeenCalledWith('#file-input', '/path/to/file.txt');
    expect(result.success).toBe(true);
  });

  it('should upload multiple files', async () => {
    const instruction = createTestInstruction({
      type: 'uploadfile',
      params: { selector: '#file-input', filePath: ['/file1.txt', '/file2.txt'] },
      node_id: 'node-1',
    });

    const result = await handler.execute(instruction, context);

    expect(mockPage.setInputFiles).toHaveBeenCalledWith('#file-input', ['/file1.txt', '/file2.txt']);
    expect(result.success).toBe(true);
  });
});
