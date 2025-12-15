import { createTypedInstruction, createMockPage, createTestConfig } from '../../helpers';
import { UploadHandler } from '../../../src/handlers/upload';
import type { HandlerContext } from '../../../src/handlers/base';
import { logger, metrics } from '../../../src/utils';

// Mock fs/promises to allow file access checks to pass in tests
jest.mock('fs/promises', () => ({
  access: jest.fn().mockResolvedValue(undefined),
  constants: { R_OK: 4 },
}));

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
    const instruction = createTypedInstruction('uploadfile', {
      selector: '#file-input',
      filePath: '/path/to/file.txt',
    }, { nodeId: 'node-1' });

    const result = await handler.execute(instruction, context);

    expect(mockPage.setInputFiles).toHaveBeenCalledWith('#file-input', '/path/to/file.txt', { timeout: 30000 });
    expect(result.success).toBe(true);
  });

  it('should upload multiple files', async () => {
    const instruction = createTypedInstruction('uploadfile', {
      selector: '#file-input',
      filePaths: ['/file1.txt', '/file2.txt'],
    }, { nodeId: 'node-1' });

    const result = await handler.execute(instruction, context);

    expect(mockPage.setInputFiles).toHaveBeenCalledWith('#file-input', ['/file1.txt', '/file2.txt'], { timeout: 30000 });
    expect(result.success).toBe(true);
  });
});
