import { chmod, mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { randomUUID } from 'node:crypto';
import os from 'node:os';
import path from 'node:path';
import {
  captureElectronDownload,
  type ElectronDownloadObserver,
} from '../../../src/handlers/electron-download';

describe('Electron target download capture', () => {
  it('copies only the completed Chromium GUID file from the private target directory', async () => {
    const root = await mkdtemp(path.join(os.tmpdir(), 'bas-electron-download-'));
    await chmod(root, 0o700);
    const downloadPath = path.join(root, 'downloads');
    await mkdir(downloadPath, { mode: 0o700 });
    await chmod(downloadPath, 0o700);
    const guid = randomUUID();
    const source = Buffer.from('completed-png-bytes');
    let begin: ((event: { guid: string; url: string; suggestedFilename: string }) => void) | undefined;
    let progress: ((event: { guid: string; state: string }) => void) | undefined;
    const observer: ElectronDownloadObserver = {
      allowDownloads: jest.fn().mockResolvedValue(undefined),
      onWillBegin: listener => {
        begin = listener;
        return () => { begin = undefined; };
      },
      onProgress: listener => {
        progress = listener;
        return () => { progress = undefined; };
      },
      detach: jest.fn().mockResolvedValue(undefined),
    };
    const target = {
      target_id: 'local-electron',
      cdp_endpoint: 'http://127.0.0.1:43123',
      renderer_id: 'renderer-1',
      renderer_url: 'file:///app/index.html',
      scenario_name: 'device-studio',
      artifact_digest: 'sha256:offline-appimage',
      context_id: 'context-1',
      cdp_transport: 'bridge-authenticated' as const,
      test_database_root: root,
      download_path: downloadPath,
    };

    try {
      const saved = await captureElectronDownload(
        target,
        observer,
        async () => {
          await writeFile(path.join(downloadPath, guid), source, { mode: 0o600 });
          begin?.({ guid, url: 'http://127.0.0.1:20000/api/export.png', suggestedFilename: 'export.png' });
          progress?.({ guid, state: 'completed' });
        },
        1000
      );

      expect(observer.allowDownloads).toHaveBeenCalledWith(downloadPath);
      expect(saved.filename).toBe('export.png');
      expect(saved.url).toBe('http://127.0.0.1:20000/api/export.png');
      expect(path.dirname(saved.downloadPath)).toBe(downloadPath);
      await expect(readFile(saved.downloadPath)).resolves.toEqual(source);
      expect(observer.detach).toHaveBeenCalledTimes(1);
    } finally {
      await rm(root, { recursive: true, force: true });
    }
  });

  it('rejects invalid GUIDs and detaches without creating an output file', async () => {
    const root = await mkdtemp(path.join(os.tmpdir(), 'bas-electron-download-invalid-'));
    await chmod(root, 0o700);
    const downloadPath = path.join(root, 'downloads');
    await mkdir(downloadPath, { mode: 0o700 });
    await chmod(downloadPath, 0o700);
    let begin: ((event: { guid: string; url: string; suggestedFilename: string }) => void) | undefined;
    const observer: ElectronDownloadObserver = {
      allowDownloads: jest.fn().mockResolvedValue(undefined),
      onWillBegin: listener => { begin = listener; return () => { begin = undefined; }; },
      onProgress: () => () => undefined,
      detach: jest.fn().mockResolvedValue(undefined),
    };
    const target = {
      target_id: 'local-electron', cdp_endpoint: 'http://127.0.0.1:43123', renderer_id: 'renderer-1',
      renderer_url: 'file:///app/index.html', scenario_name: 'device-studio', artifact_digest: 'sha256:offline-appimage',
      context_id: 'context-1', cdp_transport: 'bridge-authenticated' as const, test_database_root: root, download_path: downloadPath,
    };
    try {
      await expect(captureElectronDownload(target, observer, async () => {
        begin?.({ guid: '../escape', url: 'http://127.0.0.1/file', suggestedFilename: 'escape.png' });
      }, 1000)).rejects.toThrow('invalid Chromium GUID');
      expect(observer.detach).toHaveBeenCalledTimes(1);
    } finally {
      await rm(root, { recursive: true, force: true });
    }
  });
});
