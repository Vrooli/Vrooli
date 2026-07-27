import { afterEach, describe, expect, it, vi } from 'vitest';
import type { DownloadAsset } from '../../../shared/api';
import {
  detectPlatform,
  getDownloadAssetKey,
  getPlatformLabel,
  getVariantLabel,
  openDownloadWindow,
  sanitizeArtifactUrl,
} from './downloads.service';

const asset = (overrides: Partial<DownloadAsset> = {}): DownloadAsset => ({
  bundle_key: 'desktop',
  app_key: 'client',
  platform: 'windows',
  artifact_url: 'https://example.test/client.exe',
  release_version: '1.2.3',
  requires_entitlement: false,
  ...overrides,
} as DownloadAsset);

describe('public download service', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it.each([
    ['Windows NT 10.0', 'windows'],
    ['Macintosh; Intel Mac OS X', 'mac'],
    ['X11; Linux x86_64', 'linux'],
    ['Solaris', 'unknown'],
  ] as const)('detects %s as %s', (userAgent, expected) => {
    vi.spyOn(window.navigator, 'userAgent', 'get').mockReturnValue(userAgent);
    expect(detectPlatform()).toBe(expected);
  });

  it('builds stable keys for persisted and unpersisted assets', () => {
    expect(getDownloadAssetKey(asset({ id: 7 }))).toBe('asset-7');
    expect(getDownloadAssetKey(asset({ artifact_url: '', release_version: '' }))).toBe(
      'app-client-windows-unknown-na'
    );
  });

  it.each([
    [undefined, ''],
    ['  ', ''],
    ['javascript:alert(1)', ''],
    ['DATA:text/html,test', ''],
    ['vbscript:msgbox(1)', ''],
    ['https://cdn.example.test/app.exe', 'https://cdn.example.test/app.exe'],
    ['HTTP://cdn.example.test/app.exe', 'HTTP://cdn.example.test/app.exe'],
    ['/downloads/app.exe', '/downloads/app.exe'],
    ['../app.exe', '../app.exe'],
    ['app.exe', 'app.exe'],
    ['ftp://example.test/app.exe', ''],
  ])('sanitizes artifact URL %j', (input, expected) => {
    expect(sanitizeArtifactUrl(input)).toBe(expected);
  });

  it('only reports a download window as opened when the browser returns a handle', () => {
    const open = vi.spyOn(window, 'open');
    open.mockReturnValue(null);
    expect(openDownloadWindow('https://example.test/app.exe')).toBe(false);
    open.mockReturnValue({} as Window);
    expect(openDownloadWindow('https://example.test/app.exe')).toBe(true);
    expect(open).toHaveBeenLastCalledWith(
      'https://example.test/app.exe',
      '_blank',
      'noopener,noreferrer'
    );
  });

  it.each([
    ['release.AppImage', '', 'AppImage'],
    ['release.deb', '', '.deb'],
    ['release.rpm', '', '.rpm'],
    ['release.tar.gz', '', '.tar.gz'],
    ['release.dmg', '', '.dmg'],
    ['release.pkg', '', '.pkg'],
    ['release.exe', '', 'Installer'],
    ['release.msi', '', '.msi'],
    ['release.zip', '', '.zip'],
    ['release.bin', 'arm build', 'ARM64'],
    ['release-x64.bin', '', '64-bit'],
    ['release-x86.bin', '', '32-bit'],
    ['release.bin', '', 'v1.2.3'],
    ['release.bin', '', 'Download'],
  ])('labels installer artifacts by format, architecture, or fallback', (artifact_url, release_notes, expected) => {
    const release_version = expected === 'Download' ? '' : '1.2.3';
    expect(getVariantLabel(asset({ artifact_url, release_notes, release_version }))).toBe(expected);
  });

  it('uses known platform labels and leaves custom labels intact', () => {
    expect(getPlatformLabel('WINDOWS')).toBe('Windows');
    expect(getPlatformLabel('bsd')).toBe('bsd');
  });
});
