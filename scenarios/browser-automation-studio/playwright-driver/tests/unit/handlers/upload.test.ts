import { createTypedInstruction, createMockPage, createTestConfig } from '../../helpers';
import type { HandlerContext } from '../../../src/handlers/base';
import { logger, metrics } from '../../../src/utils';

describe('UploadHandler', () => {
  let UploadHandlerCtor: typeof import('../../../src/handlers/upload').UploadHandler;
  let handler: InstanceType<typeof import('../../../src/handlers/upload').UploadHandler>;
  let mockPage: ReturnType<typeof createMockPage>;
  let context: HandlerContext;

  beforeEach(async () => {
    jest.resetModules();
    (jest as any).unstable_mockModule('fs/promises', () => ({
      access: jest.fn().mockResolvedValue(undefined),
    }));

    const mod = await import('../../../src/handlers/upload');
    UploadHandlerCtor = mod.UploadHandler;
    handler = new UploadHandlerCtor();
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
