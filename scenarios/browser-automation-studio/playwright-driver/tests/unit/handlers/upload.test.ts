import {
  createTypedInstruction,
  createMockPage,
  createMockContext,
  createTestConfig,
} from '../../helpers';
import type { HandlerContext } from '../../../src/handlers/base';
import { logger, metrics } from '../../../src/utils';

describe('UploadHandler', () => {
  let UploadHandlerCtor: typeof import('../../../src/handlers/upload').UploadHandler;
  let handler: InstanceType<typeof import('../../../src/handlers/upload').UploadHandler>;
  let mockPage: ReturnType<typeof createMockPage>;
  let context: HandlerContext;

  beforeEach(async () => {
    jest.resetModules();
    const jestRuntime = jest as unknown as {
      unstable_mockModule: (moduleName: string, factory: () => unknown) => void;
    };
    jestRuntime.unstable_mockModule('fs/promises', () => ({
      access: jest.fn().mockResolvedValue(undefined),
    }));

    const mod = await import('../../../src/handlers/upload');
    UploadHandlerCtor = mod.UploadHandler;
    handler = new UploadHandlerCtor();
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

  it('should upload file', async () => {
    const instruction = createTypedInstruction('uploadfile', {
      selector: '#file-input',
      filePath: '/path/to/file.txt',
    }, { nodeId: 'node-1' });

    const result = await handler.execute(instruction, context);

    const [selector, files, options] = mockPage.setInputFiles.mock.calls[0] ?? [];
    expect(selector).toBe('#file-input');
    expect(files).toBe('/path/to/file.txt');
    expect(options).toEqual({ timeout: 30000 });
    expect(result.success).toBe(true);
  });

  it('should upload multiple files', async () => {
    const instruction = createTypedInstruction('uploadfile', {
      selector: '#file-input',
      filePaths: ['/file1.txt', '/file2.txt'],
    }, { nodeId: 'node-1' });

    const result = await handler.execute(instruction, context);

    const [selector, files, options] = mockPage.setInputFiles.mock.calls[0] ?? [];
    expect(selector).toBe('#file-input');
    expect(files).toEqual(['/file1.txt', '/file2.txt']);
    expect(options).toEqual({ timeout: 30000 });
    expect(result.success).toBe(true);
  });
});
