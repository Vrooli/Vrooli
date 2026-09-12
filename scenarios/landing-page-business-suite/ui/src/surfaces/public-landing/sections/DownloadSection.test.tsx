/* eslint-disable @typescript-eslint/unbound-method -- assertions exercise Vitest/browser mocks, not detached production methods. */
import { describe, expect, it, vi, afterEach } from 'vitest';
import { renderWithProviders as render } from "@vrooli/api-base/testing";
import { fireEvent, screen } from "@testing-library/react";
import userEvent from '@testing-library/user-event';
import type { DownloadApp, DownloadAsset } from '../../../shared/api';
import { DownloadSection } from './DownloadSection';
import { getDownloadAssetKey } from '../services/downloads.service';

const requestDownloadMock = vi.hoisted(() => vi.fn());
const createBillingPortalSessionMock = vi.hoisted(() => vi.fn());
const entitlementState = vi.hoisted(() => ({
  email: '', setEmail: vi.fn(), entitlements: null as { status: string } | null,
  loading: false, error: null as string | null, refresh: vi.fn(),
}));

vi.mock('../../../shared/api', async () => {
  const actual = await vi.importActual<typeof import('../../../shared/api')>('../../../shared/api');
  return {
    ...actual,
    requestDownload: requestDownloadMock,
    createBillingPortalSession: createBillingPortalSessionMock,
  };
});

vi.mock('../../../shared/hooks/useEntitlements', () => ({
  useEntitlements: () => entitlementState,
}));

vi.mock('../../../shared/hooks/useMetricsHook', () => ({
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
  const originalWindowOpen = window.open.bind(window);
  const originalNavigator = window.navigator;

  afterEach(() => {
    requestDownloadMock.mockReset();
    createBillingPortalSessionMock.mockReset();
    Object.assign(entitlementState, { email: '', entitlements: null, loading: false, error: null });
    entitlementState.setEmail.mockReset();
    entitlementState.refresh.mockReset();
    window.open = originalWindowOpen;
    Object.defineProperty(window, 'navigator', {
      value: originalNavigator,
      writable: true,
    });
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

  it('renders primary download card for first platform', () => {
    render(<DownloadSection downloads={[buildApp()]} />);

    expect(screen.getByTestId('download-card-primary')).toBeInTheDocument();
    expect(screen.getByTestId('download-btn-primary')).toBeInTheDocument();
  });

  it('shows other platforms toggle when multiple platforms exist', async () => {
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
        platform: 'mac',
        artifact_url: 'https://example.com/app.dmg',
        release_version: '1.0.0',
        requires_entitlement: false,
      },
    ];

    render(<DownloadSection downloads={[buildApp(undefined, platforms)]} />);

    const toggleButton = screen.getByTestId('toggle-other-platforms');
    expect(toggleButton).toBeInTheDocument();

    const user = userEvent.setup();
    await user.click(toggleButton);

    expect(screen.getByTestId('other-platforms-list')).toBeInTheDocument();
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
    await user.click(screen.getByTestId('download-btn-primary'));

    expect(await screen.findByText('Download not available yet. Check back soon.')).toBeInTheDocument();
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
    await user.click(screen.getByTestId('download-btn-primary'));

    expect(await screen.findByText('Pop-up blocked. Allow pop-ups and try again.')).toBeInTheDocument();
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
    await user.click(screen.getByTestId('download-btn-primary'));

    expect(window.open).toHaveBeenCalledWith('/downloads/app.tar.gz', '_blank', 'noopener,noreferrer');
    expect(await screen.findByText('Download started')).toBeInTheDocument();
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
    await user.click(screen.getByTestId('download-btn-primary'));

    expect(await screen.findByText('Download not available yet. Check back soon.')).toBeInTheDocument();
    expect(window.open).not.toHaveBeenCalled();
  });

  it('shows post-download instructions after successful download', async () => {
    const app = buildApp(undefined, [
      {
        bundle_key: 'bundle',
        app_key: 'automation',
        platform: 'windows',
        artifact_url: 'https://example.com/app.exe',
        release_version: '1.0.0',
        requires_entitlement: false,
      },
    ]);

    requestDownloadMock.mockResolvedValueOnce(app.platforms[0]);
    window.open = vi.fn(() => ({} as Window));

    render(<DownloadSection downloads={[app]} />);

    const user = userEvent.setup();
    await user.click(screen.getByTestId('download-btn-primary'));

    expect(await screen.findByTestId('post-download-instructions')).toBeInTheDocument();
    expect(screen.getByText('Download started')).toBeInTheDocument();
  });

  it('shows subscription input toggle', async () => {
    render(<DownloadSection downloads={[buildApp()]} />);

    const toggleButton = screen.getByTestId('toggle-subscription-input');
    expect(toggleButton).toBeInTheDocument();
    expect(toggleButton).toHaveTextContent('Already have a subscription?');

    const user = userEvent.setup();
    await user.click(toggleButton);

    expect(screen.getByTestId('subscription-input-panel')).toBeInTheDocument();
  });

  it('renders branded download content and falls back safely when an icon fails to load', () => {
    const app = buildApp({ icon_url: '/assets/icon.png', screenshot_url: '/assets/preview.png' });
    render(<DownloadSection content={{ title: 'Get the desktop app', subtitle: 'Choose your installer' }} downloads={[app]} supportEmail="help@example.com" />);

    expect(screen.getByRole('heading', { name: 'Get the desktop app' })).toBeInTheDocument();
    expect(screen.getByText('Choose your installer')).toBeInTheDocument();
    const icon = screen.getByAltText('Automation Studio icon');
    fireEvent.error(icon);
    expect(icon).toHaveStyle({ display: 'none' });
    expect(screen.getByAltText('Automation Studio screenshot')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Contact support' })).toHaveAttribute('href', 'mailto:help@example.com');
  });

  it('does not render when every app lacks an install target', () => {
    render(<DownloadSection downloads={[buildApp(undefined, [])]} />);
    expect(screen.queryByTestId('downloads-section')).not.toBeInTheDocument();
  });

  it('reports request failures and prevents downloads when app configuration has no application key', async () => {
    const user = userEvent.setup();
    const app = buildApp({ app_key: '' }, [{
      bundle_key: 'bundle', app_key: '', platform: 'windows', artifact_url: 'https://example.com/app.exe',
      release_version: '1.0.0', requires_entitlement: false,
    }]);
    render(<DownloadSection downloads={[app]} />);
    await user.click(screen.getByTestId('download-btn-primary'));
    expect(await screen.findByText('App configuration error.')).toBeInTheDocument();
    expect(requestDownloadMock).not.toHaveBeenCalled();

    requestDownloadMock.mockRejectedValueOnce(new Error('Service unavailable'));
    render(<DownloadSection downloads={[buildApp()]} />);
    await user.click(screen.getAllByTestId('download-btn-primary')[1]!);
    expect(await screen.findByText('Service unavailable')).toBeInTheDocument();
  });

  it('shows subscription states and validates billing portal requests before opening a window', async () => {
    Object.assign(entitlementState, { email: 'member@example.com', entitlements: { status: 'trialing' } });
    const user = userEvent.setup();
    render(<DownloadSection downloads={[buildApp()]} />);
    await user.click(screen.getByTestId('toggle-subscription-input'));
    expect(screen.getByText('Trial active')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Refresh' }));
    expect(entitlementState.refresh).toHaveBeenCalledOnce();
    fireEvent.change(screen.getByPlaceholderText('your@email'), { target: { value: 'next@example.com' } });
    fireEvent.blur(screen.getByPlaceholderText('your@email'));
    expect(entitlementState.setEmail).toHaveBeenCalledWith('next@example.com');
    expect(entitlementState.refresh).toHaveBeenCalledTimes(2);

    createBillingPortalSessionMock.mockResolvedValueOnce({ url: 'https://billing.example.test/portal' });
    window.open = vi.fn(() => null);
    await user.click(screen.getByRole('button', { name: 'Billing portal' }));
    expect(createBillingPortalSessionMock).toHaveBeenCalledWith(undefined, 'member@example.com');
    expect(await screen.findByText('Pop-up blocked. Allow pop-ups to open the billing portal.')).toBeInTheDocument();
  });

  it('makes unavailable platform groups harmless and renders an unverified subscription state', async () => {
    const app = buildApp(undefined, [
      { bundle_key: 'bundle', app_key: 'automation', platform: 'windows', artifact_url: 'https://example.com/app.exe', release_version: '1.0.0', requires_entitlement: false },
      { bundle_key: 'bundle', app_key: 'automation', platform: 'solaris', artifact_url: 'https://example.com/app.pkg', release_version: '1.0.0', requires_entitlement: false },
    ]);
    const user = userEvent.setup();
    render(<DownloadSection downloads={[app]} />);
    await user.click(screen.getByTestId('toggle-other-platforms'));
    expect(screen.queryByTestId('download-card-solaris')).not.toBeInTheDocument();
    await user.click(screen.getByTestId('toggle-subscription-input'));
    expect(screen.getByText('Enter email to verify')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Billing portal' }));
    expect(screen.getByText('Enter your subscription email first.')).toBeInTheDocument();
  });
});
