import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { DownloadApp, DownloadAsset } from '../../../shared/api';
import { DownloadSection, getDownloadAssetKey } from './DownloadSection';

const requestDownloadMock = vi.hoisted(() => vi.fn());

vi.mock('../../../shared/api', async () => {
  const actual = await vi.importActual<typeof import('../../../shared/api')>('../../../shared/api');
  return {
    ...actual,
    requestDownload: requestDownloadMock,
  };
});

vi.mock('../../../shared/hooks/useEntitlements', () => ({
  useEntitlements: () => ({
    email: '',
    setEmail: vi.fn(),
    entitlements: null,
    loading: false,
    error: null,
    refresh: vi.fn(),
  }),
}));

vi.mock('../../../shared/hooks/useMetrics', () => ({
  useMetrics: () => ({
    trackDownload: vi.fn(),
    trackCTAClick: vi.fn(),
    trackFormSubmit: vi.fn(),
    trackConversion: vi.fn(),
    trackEvent: vi.fn(),
  }),
}));

describe('getDownloadAssetKey', () => {
  it('uses numeric id when present', () => {
    const asset: DownloadAsset = {
      id: 42,
      bundle_key: 'bundle',
      app_key: 'app',
      platform: 'windows',
      artifact_url: 'https://example.com/app.exe',
      release_version: '1.0.0',
      requires_entitlement: true,
    };

    expect(getDownloadAssetKey(asset)).toBe('asset-42');
  });

  it('falls back to composite key when id missing', () => {
    const asset: DownloadAsset = {
      bundle_key: 'bundle',
      app_key: 'studio',
      platform: 'mac',
      artifact_url: 'https://example.com/app.dmg',
      release_version: '1.0.0',
      requires_entitlement: false,
    };

    expect(getDownloadAssetKey(asset)).toContain('app-studio-mac-1.0.0-https://example.com/app.dmg');
  });
});

describe('DownloadSection', () => {
  const originalWindowOpen = window.open;

  afterEach(() => {
    requestDownloadMock.mockReset();
    window.open = originalWindowOpen;
  });

  const buildApp = (overrides?: Partial<DownloadApp>, platforms?: DownloadAsset[]): DownloadApp => ({
    bundle_key: 'bundle',
    app_key: 'automation',
    name: 'Automation Studio',
    tagline: 'Desktop automation suite',
    description: 'Default description',
    install_overview: 'Download and sign in.',
    install_steps: ['Download installer', 'Launch setup', 'Sign in'],
    storefronts: [],
    display_order: 0,
    platforms: platforms ?? [
      {
        bundle_key: 'bundle',
        app_key: 'automation',
        platform: 'windows',
        artifact_url: 'https://example.com/app.exe',
        release_version: '1.0.0',
        requires_entitlement: false,
      },
    ],
    ...overrides,
  });

  it('renders unique cards even when platforms repeat within an app', () => {
    const platforms: DownloadAsset[] = [
      {
        bundle_key: 'bundle',
        app_key: 'automation',
        platform: 'windows',
        artifact_url: 'https://example.com/app.exe',
        release_version: '1.0.0',
        requires_entitlement: false,
      },
      {
        bundle_key: 'bundle',
        app_key: 'automation',
        platform: 'windows',
        artifact_url: 'https://example.com/app-beta.exe',
        release_version: '1.1.0-beta',
        requires_entitlement: false,
      },
    ];

    render(<DownloadSection downloads={[buildApp(undefined, platforms)]} />);

    const cards = screen.getAllByTestId(/download-card-/);
    expect(cards).toHaveLength(2);
    expect(cards[0]).not.toBe(cards[1]);
  });

  it('shows a helpful message when an artifact URL is missing', async () => {
    const apps: DownloadApp[] = [
      buildApp(undefined, [
        {
          bundle_key: 'bundle',
          app_key: 'automation',
          platform: 'windows',
          artifact_url: 'not-used',
          release_version: '1.0.0',
          requires_entitlement: false,
        },
      ]),
    ];

    requestDownloadMock.mockResolvedValueOnce({
      bundle_key: 'bundle',
      app_key: 'automation',
      platform: 'windows',
      release_version: '1.0.0',
      requires_entitlement: false,
      artifact_url: '   ',
    });

    window.open = vi.fn();

    render(<DownloadSection downloads={apps} />);

    const user = userEvent.setup();
    await user.click(screen.getByRole('button', { name: /download/i }));

    expect(await screen.findByText('Download artifact is not available yet. Please try again later.')).toBeInTheDocument();
    expect(window.open).not.toHaveBeenCalled();
  });

  it('warns when the browser blocks pop-ups', async () => {
    const app = buildApp(undefined, [
      {
        bundle_key: 'bundle',
        app_key: 'automation',
        platform: 'mac',
        artifact_url: 'https://example.com/app.dmg',
        release_version: '1.2.3',
        requires_entitlement: false,
      },
    ]);

    requestDownloadMock.mockResolvedValueOnce(app.platforms[0]);

    window.open = vi.fn(() => null);

    render(<DownloadSection downloads={[app]} />);

    const user = userEvent.setup();
    await user.click(screen.getByRole('button', { name: /download/i }));

    expect(await screen.findByText('Unable to open download. Allow pop-ups and try again.')).toBeInTheDocument();
    expect(window.open).toHaveBeenCalledWith('https://example.com/app.dmg', '_blank', 'noopener,noreferrer');
  });

  it('allows safe relative artifact URLs returned by the API', async () => {
    const app = buildApp(undefined, [
      {
        bundle_key: 'bundle',
        app_key: 'automation',
        platform: 'linux',
        artifact_url: '/downloads/app.tar.gz',
        release_version: '0.9.0',
        requires_entitlement: false,
      },
    ]);

    requestDownloadMock.mockResolvedValueOnce({
      ...app.platforms[0],
      artifact_url: '/downloads/app.tar.gz',
    });

    window.open = vi.fn(() => ({} as Window));

    render(<DownloadSection downloads={[app]} />);

    const user = userEvent.setup();
    await user.click(screen.getByRole('button', { name: /download/i }));

    expect(window.open).toHaveBeenCalledWith('/downloads/app.tar.gz', '_blank', 'noopener,noreferrer');
    expect(await screen.findByText('Download started in a new tab.')).toBeInTheDocument();
  });

  it('rejects dangerous artifact URL schemes before opening a new window', async () => {
    const app = buildApp(undefined, [
      {
        bundle_key: 'bundle',
        app_key: 'automation',
        platform: 'mac',
        artifact_url: 'placeholder',
        release_version: '2.0.0',
        requires_entitlement: false,
      },
    ]);

    requestDownloadMock.mockResolvedValueOnce({
      ...app.platforms[0],
      artifact_url: 'javascript:alert(1)',
    });

    window.open = vi.fn(() => ({} as Window));

    render(<DownloadSection downloads={[app]} />);

    const user = userEvent.setup();
    await user.click(screen.getByRole('button', { name: /download/i }));

    expect(await screen.findByText('Download artifact is not available yet. Please try again later.')).toBeInTheDocument();
    expect(window.open).not.toHaveBeenCalled();
  });
});
