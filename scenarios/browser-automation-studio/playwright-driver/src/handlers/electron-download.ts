import { randomUUID } from 'node:crypto';
import { constants } from 'node:fs';
import { lstat, open, rm } from 'node:fs/promises';
import path from 'node:path';
import { pipeline } from 'node:stream/promises';
import type { CDPSession } from 'rebrowser-playwright';
import type { AppTargetSpec } from '../types/session';
import { configureAppTargetDownloadBehavior, validateAppTargetSpec } from '../session/electron-target';

type DownloadStart = { guid: string; url: string; suggestedFilename: string };
type DownloadProgress = { guid: string; state: string };

export interface ElectronDownloadObserver {
  allowDownloads(directory: string): Promise<void>;
  onWillBegin(listener: (event: DownloadStart) => void): () => void;
  onProgress(listener: (event: DownloadProgress) => void): () => void;
  detach(): Promise<void>;
}

export function createElectronDownloadObserver(session: CDPSession, browserContextId?: string): ElectronDownloadObserver {
  return {
    allowDownloads: async directory => {
      await configureAppTargetDownloadBehavior(
        { download_path: directory },
        params => session.send('Browser.setDownloadBehavior', params),
        browserContextId
      );
    },
    onWillBegin: listener => {
      const handle = (event: { guid: string; url: string; suggestedFilename: string }): void => listener(event);
      session.on('Browser.downloadWillBegin', handle);
      return () => session.off('Browser.downloadWillBegin', handle);
    },
    onProgress: listener => {
      const handle = (event: { guid: string; state: string }): void => listener(event);
      session.on('Browser.downloadProgress', handle);
      return () => session.off('Browser.downloadProgress', handle);
    },
    detach: () => session.detach(),
  };
}

export async function captureElectronDownload(
  target: AppTargetSpec,
  observer: ElectronDownloadObserver,
  trigger: () => Promise<unknown>,
  timeoutMs: number
): Promise<{ downloadPath: string; filename: string; url: string }> {
  validateAppTargetSpec(target);
  const directory = target.download_path;
  if (!directory) throw new Error('Electron download target has no private download directory');

  let beginResolve: ((event: DownloadStart) => void) | undefined;
  let beginReject: ((error: Error) => void) | undefined;
  let progressResolve: (() => void) | undefined;
  let progressReject: ((error: Error) => void) | undefined;
  const begin = new Promise<DownloadStart>((resolve, reject) => {
    beginResolve = resolve;
    beginReject = reject;
  });
  const completed = new Promise<void>((resolve, reject) => {
    progressResolve = resolve;
    progressReject = reject;
  });
  void completed.catch(() => undefined);
  let activeGuid = '';
  let destination = '';
  const stopOnBegin = observer.onWillBegin(event => {
    if (activeGuid) return;
    activeGuid = event.guid;
    beginResolve?.(event);
  });
  const stopOnProgress = observer.onProgress(event => {
    if (event.guid !== activeGuid) return;
    if (event.state === 'completed') progressResolve?.();
    if (event.state === 'canceled') progressReject?.(new Error('Electron download was canceled'));
  });
  let timer: NodeJS.Timeout | undefined;
  const timeout = new Promise<never>((_, reject) => {
    timer = setTimeout(() => {
      const error = new Error(`Electron download did not complete within ${timeoutMs}ms`);
      beginReject?.(error);
      progressReject?.(error);
      reject(error);
    }, timeoutMs);
  });
  void timeout.catch(() => undefined);

  try {
    await observer.allowDownloads(directory);
    await Promise.race([trigger(), timeout]);
    const download = await Promise.race([begin, timeout]);
    if (!/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(download.guid)) {
      throw new Error('Electron download returned an invalid Chromium GUID');
    }
    await Promise.race([completed, timeout]);

    const source = path.join(directory, download.guid);
    const sourceInfo = await lstat(source);
    if (!sourceInfo.isFile() || sourceInfo.isSymbolicLink()) {
      throw new Error('Completed Electron download is not a regular private file');
    }
    const filename = path.basename(download.suggestedFilename.replaceAll('\\', '/')) || 'download.bin';
    destination = path.join(directory, `download-${randomUUID()}-${filename}`);
    const input = await open(source, constants.O_RDONLY | constants.O_NOFOLLOW);
    let output: Awaited<ReturnType<typeof open>> | undefined;
    try {
      output = await open(
        destination,
        constants.O_WRONLY | constants.O_CREAT | constants.O_EXCL | constants.O_NOFOLLOW,
        0o600
      );
      if (!(await input.stat()).isFile()) throw new Error('Completed Electron download changed before copy');
      await pipeline(input.createReadStream(), output.createWriteStream());
    } finally {
      const closes = [input.close()];
      if (output) closes.push(output.close());
      await Promise.allSettled(closes);
    }
    return { downloadPath: destination, filename, url: download.url };
  } catch (error) {
    if (destination) await rm(destination, { force: true }).catch(() => undefined);
    throw error;
  } finally {
    if (timer) clearTimeout(timer);
    stopOnBegin();
    stopOnProgress();
    await observer.detach().catch(() => undefined);
  }
}
