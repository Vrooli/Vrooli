import type { Page } from 'rebrowser-playwright';
import { chmodSync, mkdtempSync, mkdirSync, rmSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import {
  configureAppTargetDownloadBehavior,
  selectAppTargetPage,
  validateAppTargetCapabilities,
  validateAppTargetSpec,
  verifyAppTargetRenderer,
} from '../../../src/session/electron-target';

const target = {
  target_id: 'target-1',
  cdp_endpoint: 'http://127.0.0.1:43123',
  renderer_id: 'renderer-1',
  renderer_url: 'file:///controlled/index.html',
  renderer_title: 'Controlled Desktop',
  scenario_name: 'controlled-scenario',
  artifact_digest: 'sha256:controlled',
  context_id: 'ctx-1',
  cdp_transport: 'loopback-authenticated' as const,
};

describe('Electron target admission', () => {
  afterEach(() => jest.restoreAllMocks());

  it('rejects non-loopback endpoints and embedded credentials', () => {
    expect(() =>
      validateAppTargetSpec({ ...target, cdp_endpoint: 'http://example.test:9222' })
    ).toThrow('loopback');
    expect(() =>
      validateAppTargetSpec({ ...target, cdp_endpoint: 'http://user:pass@127.0.0.1:9222' })
    ).toThrow('credentials');
  });

  it('refuses evidence capabilities that require context creation control', () => {
    expect(() => validateAppTargetCapabilities({ video: true })).toThrow('video');
    expect(() => validateAppTargetCapabilities({ har: true, performance_trace: true })).toThrow(
      'har'
    );
    expect(() => validateAppTargetCapabilities(undefined)).not.toThrow();
  });

  it('requires the target to carry scenario and artifact identity', () => {
    expect(() => validateAppTargetSpec({ ...target, scenario_name: '' })).toThrow('scenario_name');
    expect(() => validateAppTargetSpec({ ...target, artifact_digest: '' })).toThrow(
      'artifact_digest'
    );
  });

  it('binds downloads to a private target-owned directory and configures Chromium', async () => {
    const root = mkdtempSync(path.join(os.tmpdir(), 'bas-app-target-'));
    chmodSync(root, 0o700);
    const downloadPath = path.join(root, 'downloads');
    mkdirSync(downloadPath, { mode: 0o700 });
    chmodSync(downloadPath, 0o700);
    try {
      const targetWithDownloads = { ...target, test_database_root: root, download_path: downloadPath };
      expect(() => validateAppTargetSpec(targetWithDownloads)).not.toThrow();
      const send = jest.fn().mockResolvedValue({});
      await configureAppTargetDownloadBehavior(targetWithDownloads, send);
      expect(send).toHaveBeenCalledWith({
        behavior: 'allowAndName',
        downloadPath,
        eventsEnabled: true,
      });
      await configureAppTargetDownloadBehavior(targetWithDownloads, send, 'browser-context-1');
      expect(send).toHaveBeenLastCalledWith({
        behavior: 'allowAndName',
        downloadPath,
        eventsEnabled: true,
        browserContextId: 'browser-context-1',
      });
      await configureAppTargetDownloadBehavior(target, send);
      expect(send).toHaveBeenCalledTimes(2);
      expect(() => validateAppTargetSpec({ ...targetWithDownloads, download_path: path.join(root, '..', 'outside') })).toThrow('downloads child');
      chmodSync(downloadPath, 0o755);
      expect(() => validateAppTargetSpec(targetWithDownloads)).toThrow('mode 0700');
    } finally {
      rmSync(root, { recursive: true, force: true });
    }
  });

  it('requires the admitted renderer id and origin', async () => {
    const response = (): Response =>
      new Response(
        JSON.stringify([
          {
            id: 'renderer-1',
            type: 'page',
            url: target.renderer_url,
            title: target.renderer_title,
          },
        ]),
        { status: 200, headers: { 'content-type': 'application/json' } }
      );
    jest.spyOn(global, 'fetch').mockImplementation(() => Promise.resolve(response()));
    await expect(verifyAppTargetRenderer(target)).resolves.toBeUndefined();
    await expect(verifyAppTargetRenderer({ ...target, renderer_id: 'wrong' })).rejects.toThrow(
      'missing'
    );
    const transitioned = (): Response =>
      new Response(
        JSON.stringify([
          {
            id: 'renderer-1',
            type: 'page',
            url: 'file:///controlled/route',
            title: target.renderer_title,
          },
        ]),
        { status: 200, headers: { 'content-type': 'application/json' } }
      );
    jest.spyOn(global, 'fetch').mockImplementation(() => Promise.resolve(transitioned()));
    await expect(verifyAppTargetRenderer(target)).resolves.toBeUndefined();
  });

  it('does not treat a normal renderer title transition as target replacement', async () => {
    const response = (): Response =>
      new Response(
        JSON.stringify([
          {
            id: target.renderer_id,
            type: 'page',
            url: target.renderer_url,
            title: 'Hello Desktop',
          },
        ]),
        { status: 200, headers: { 'content-type': 'application/json' } }
      );
    jest.spyOn(global, 'fetch').mockImplementation(() => Promise.resolve(response()));
    await expect(verifyAppTargetRenderer(target)).resolves.toBeUndefined();

    const page = {
      url: () => target.renderer_url,
      title: jest.fn().mockResolvedValue('Hello Desktop'),
    } as unknown as Page;
    await expect(selectAppTargetPage([page], target)).resolves.toBe(page);
  });

  it('refuses a stale or unavailable CDP endpoint', async () => {
    jest.spyOn(global, 'fetch').mockRejectedValue(new TypeError('fetch failed'));
    await expect(verifyAppTargetRenderer(target)).rejects.toThrow('fetch failed');
  });

  it('refuses ambiguous or missing Playwright page identity', async () => {
    const page = (url: string): Page => ({ url: () => url }) as unknown as Page;
    await expect(
      selectAppTargetPage([page(target.renderer_url), page(target.renderer_url)], target)
    ).rejects.toThrow('ambiguous');
    const androidTarget = { ...target, target_kind: 'android-webview' as const };
    const androidPages = [page(androidTarget.renderer_url), page(androidTarget.renderer_url)];
    await expect(selectAppTargetPage(androidPages, androidTarget)).resolves.toBe(androidPages[0]);
    await expect(
      selectAppTargetPage([page('file:///controlled/route')], target)
    ).resolves.toBeTruthy();
    await expect(selectAppTargetPage([page('about:blank')], target)).rejects.toThrow('missing');
    const matchingPage = page(target.renderer_url);
    matchingPage.title = jest.fn().mockResolvedValue(target.renderer_title);
    await expect(selectAppTargetPage([matchingPage], target)).resolves.toBe(matchingPage);
  });
});
