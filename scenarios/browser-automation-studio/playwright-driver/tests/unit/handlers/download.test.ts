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
  it('saves admitted Electron target downloads inside its private download directory', async () => {
    const root = await mkdtemp(path.join(os.tmpdir(), 'bas-handler-electron-download-'));
    await chmod(root, 0o700);
    const downloadPath = path.join(root, 'downloads');
    await mkdir(downloadPath, { mode: 0o700 });
    await chmod(downloadPath, 0o700);
    context.electronTarget = {
      target_id: 'target',
      cdp_endpoint: 'http://127.0.0.1:9222',
      renderer_id: 'renderer',
      renderer_url: 'file:///app/index.html',
      scenario_name: 'device-studio',
      artifact_digest: 'sha256:artifact',
      context_id: 'validation-cell',
      cdp_transport: 'loopback-authenticated',
      download_path: downloadPath,
      test_database_root: root,
    };
    const guid = randomUUID();
    const bytes = Buffer.from('handler-private-download');
    const events = new EventEmitter();
    const cdp = {
      send: jest.fn().mockResolvedValue(undefined),
      on: events.on.bind(events),
      off: events.off.bind(events),
      detach: jest.fn().mockResolvedValue(undefined),
    } as unknown as CDPSession;
    const browser = { newBrowserCDPSession: jest.fn().mockResolvedValue(cdp) } as unknown as Browser;
    Object.defineProperty(context.browserContext, 'browser', { value: () => browser });
    mockPage.click = jest.fn(async () => {
      await writeFile(path.join(downloadPath, guid), bytes, { mode: 0o600 });
      events.emit('Browser.downloadWillBegin', {
        guid,
        url: 'blob:http://127.0.0.1/output',
        suggestedFilename: 'result.png',
      });
      events.emit('Browser.downloadProgress', { guid, state: 'completed' });
    });
    try {
      const result = await handler.execute(createTypedInstruction('download', { selector: '#download' }), context);

      expect(result.success).toBe(true);
      expect(cdp.send).toHaveBeenCalledWith('Browser.setDownloadBehavior', {
        behavior: 'allowAndName',
        downloadPath,
        eventsEnabled: true,
      });
      expect(path.dirname(result.extracted_data?.download_path as string)).toBe(downloadPath);
      expect(result.extracted_data?.filename).toBe('result.png');
      await expect(readFile(result.extracted_data?.download_path as string)).resolves.toEqual(bytes);
      expect(mockPage.waitForEvent).not.toHaveBeenCalled();
    } finally {
      await rm(root, { recursive: true, force: true });
    }
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
import { EventEmitter } from 'node:events';
import { chmod, mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { randomUUID } from 'node:crypto';
import os from 'node:os';
import path from 'node:path';
import type { Browser, CDPSession } from 'rebrowser-playwright';
