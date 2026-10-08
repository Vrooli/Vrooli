import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils';
import userEvent from '@testing-library/user-event';
import { create } from '@bufbuild/protobuf';
import {
  DownloadAppSchema,
  DownloadAssetSchema,
  DownloadStorefrontSchema,
} from '@vrooli/proto-types/landing-page-react-vite/v1/download_pb';
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

const entHolder = vi.hoisted(() => ({
  value: {
    email: '',
    setEmail: vi.fn(),
    entitlements: null as unknown,
    loading: false,
    error: null as string | null,
    refresh: vi.fn(),
  },
}));
vi.mock('../../../shared/hooks/useEntitlements', () => ({
  useEntitlements: () => entHolder.value,
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
    const asset: DownloadAsset = create(DownloadAssetSchema, {
      id: 42n,
      bundleKey: 'bundle',
      appKey: 'app',
      platform: 'windows',
      artifactUrl: 'https://example.com/app.exe',
      releaseVersion: '1.0.0',
      requiresEntitlement: true,
    });

    expect(getDownloadAssetKey(asset)).toBe('asset-42');
  });

  it('falls back to composite key when id missing', () => {
    const asset: DownloadAsset = create(DownloadAssetSchema, {
      bundleKey: 'bundle',
      appKey: 'studio',
      platform: 'mac',
      artifactUrl: 'https://example.com/app.dmg',
      releaseVersion: '1.0.0',
      requiresEntitlement: false,
    });

    expect(getDownloadAssetKey(asset)).toContain('app-studio-mac-1.0.0-https://example.com/app.dmg');
  });
});

describe('DownloadSection', () => {
  const originalWindowOpen = window.open;

  afterEach(() => {
    requestDownloadMock.mockReset();
    window.open = originalWindowOpen;
    entHolder.value = { email: '', setEmail: vi.fn(), entitlements: null, loading: false, error: null, refresh: vi.fn() };
  });

  const gatedApp = () =>
    create(DownloadAppSchema, {
      bundleKey: 'bundle',
      appKey: 'automation',
      name: 'Automation Studio',
      storefronts: [],
      displayOrder: 0,
      platforms: [
        create(DownloadAssetSchema, {
          bundleKey: 'bundle',
          appKey: 'automation',
          platform: 'windows',
          artifactUrl: 'https://example.com/app.exe',
          releaseVersion: '1.0.0',
          requiresEntitlement: true,
        }),
      ],
    });

  it('renders the full app detail matrix (steps, storefronts, installers)', () => {
    const richApp = create(DownloadAppSchema, {
      bundleKey: 'bundle',
      appKey: 'suite',
      name: 'Suite',
      tagline: 'Ship faster',
      description: 'A rich desktop bundle.',
      installOverview: 'Install in one click.',
      installSteps: ['Download', 'Run', 'Sign in'],
      displayOrder: 0,
      storefronts: [
        create(DownloadStorefrontSchema, { store: 'app_store', label: 'App Store', url: 'https://apple/x', badge: 'New' }),
        create(DownloadStorefrontSchema, { store: 'play_store', label: 'Google Play', url: 'https://play/x', badge: '' }),
      ],
      platforms: [
        create(DownloadAssetSchema, { appKey: 'suite', platform: 'mac', artifactUrl: 'https://cdn/mac.dmg', releaseVersion: '1.0.0', releaseNotes: 'First release', requiresEntitlement: false, metadata: { size_mb: 88 } }),
        create(DownloadAssetSchema, { appKey: 'suite', platform: 'windows', artifactUrl: 'https://cdn/win.exe', releaseVersion: '1.0.0', requiresEntitlement: true }),
      ],
    });
    render(<DownloadSection downloads={[richApp]} />);
    expect(screen.getByText('A rich desktop bundle.')).toBeInTheDocument();
    expect(screen.getByTestId('install-steps-suite')).toBeInTheDocument();
    expect(screen.getByTestId('storefront-suite-app_store')).toHaveTextContent('New');
    // Empty badge falls back to the store label.
    expect(screen.getByTestId('storefront-suite-play_store')).toHaveTextContent('Google Play');
    expect(screen.getByText('88 MB')).toBeInTheDocument();
    expect(screen.getByText('First release')).toBeInTheDocument();
    expect(screen.getByText('Release notes coming soon.')).toBeInTheDocument();
  });

  it('formats a large credit balance and tolerates missing version/artifact', () => {
    entHolder.value = {
      ...entHolder.value,
      email: 'user@example.com',
      entitlements: {
        status: 'active',
        planTier: 'pro',
        priceId: 'price_1',
        credits: { balanceCredits: 5_000_000_000n },
      },
    };
    const app = create(DownloadAppSchema, {
      appKey: 'suite',
      name: 'Suite',
      storefronts: [],
      platforms: [
        create(DownloadAssetSchema, { appKey: 'suite', platform: 'mac', artifactUrl: '', releaseVersion: '', requiresEntitlement: true }),
      ],
    });
    render(<DownloadSection downloads={[app]} />);
    expect(screen.getByText('active')).toBeInTheDocument();
  });

  it('shows the entitlement checking state while status loads', () => {
    entHolder.value = { ...entHolder.value, email: 'user@example.com', loading: true };
    render(<DownloadSection downloads={[gatedApp()]} />);
    expect(screen.getByText(/Checking status/i)).toBeInTheDocument();
  });

  it('surfaces an entitlement lookup error', () => {
    entHolder.value = { ...entHolder.value, email: 'user@example.com', error: 'lookup failed' };
    render(<DownloadSection downloads={[gatedApp()]} />);
    expect(screen.getByText('lookup failed')).toBeInTheDocument();
  });

  it('prompts for an email on an entitlement-gated download', async () => {
    const user = userEvent.setup();
    render(<DownloadSection downloads={[gatedApp()]} />);
    await user.click(screen.getByRole('button', { name: /download/i }));
    expect(
      await screen.findByText(/Enter the email tied to your subscription/i),
    ).toBeInTheDocument();
    expect(requestDownloadMock).not.toHaveBeenCalled();
  });

  it('blocks a gated download when the subscription is inactive', async () => {
    entHolder.value = {
      ...entHolder.value,
      email: 'user@example.com',
      entitlements: { status: 'canceled' },
    };
    const user = userEvent.setup();
    render(<DownloadSection downloads={[gatedApp()]} />);
    await user.click(screen.getByRole('button', { name: /download/i }));
    expect(await screen.findByText(/Subscription status is canceled/i)).toBeInTheDocument();
    expect(requestDownloadMock).not.toHaveBeenCalled();
  });

  it('treats a trialing subscription as active and shows credits + subscription id', () => {
    entHolder.value = {
      ...entHolder.value,
      email: 'user@example.com',
      entitlements: {
        status: 'trialing',
        planTier: 'pro',
        priceId: 'price_1',
        credits: { balanceCredits: 5000n },
        subscription: { subscriptionId: 'sub_123', cachedAt: undefined },
      },
    };
    render(<DownloadSection downloads={[gatedApp()]} />);
    expect(screen.getByText('trialing')).toBeInTheDocument();
    expect(screen.getByText(/sub_123/)).toBeInTheDocument();
  });

  it('allows a gated download with an active subscription', async () => {
    entHolder.value = {
      ...entHolder.value,
      email: 'user@example.com',
      entitlements: {
        status: 'active',
        planTier: 'pro',
        priceId: 'price_1',
        credits: { balanceCredits: 1000n },
      },
    };
    requestDownloadMock.mockResolvedValueOnce(
      create(DownloadAssetSchema, {
        appKey: 'automation',
        platform: 'windows',
        releaseVersion: '1.0.0',
        artifactUrl: 'https://example.com/app.exe',
      }),
    );
    window.open = vi.fn().mockReturnValue({});
    const user = userEvent.setup();
    render(<DownloadSection downloads={[gatedApp()]} />);
    // The entitlement summary reflects the active subscription.
    expect(screen.getByText('active')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /download/i }));
    expect(await screen.findByText('Download started in a new tab.')).toBeInTheDocument();
  });

  const buildApp = (overrides?: Partial<DownloadApp>, platforms?: DownloadAsset[]): DownloadApp =>
    create(DownloadAppSchema, {
      bundleKey: 'bundle',
      appKey: 'automation',
      name: 'Automation Studio',
      tagline: 'Desktop automation suite',
      description: 'Default description',
      installOverview: 'Download and sign in.',
      installSteps: ['Download installer', 'Launch setup', 'Sign in'],
      storefronts: [],
      displayOrder: 0,
      platforms: platforms ?? [
        create(DownloadAssetSchema, {
          bundleKey: 'bundle',
          appKey: 'automation',
          platform: 'windows',
          artifactUrl: 'https://example.com/app.exe',
          releaseVersion: '1.0.0',
          requiresEntitlement: false,
        }),
      ],
      ...overrides,
    });

  it('renders unique cards even when platforms repeat within an app', () => {
    const platforms: DownloadAsset[] = [
      create(DownloadAssetSchema, {
        bundleKey: 'bundle',
        appKey: 'automation',
        platform: 'windows',
        artifactUrl: 'https://example.com/app.exe',
        releaseVersion: '1.0.0',
        requiresEntitlement: false,
      }),
      create(DownloadAssetSchema, {
        bundleKey: 'bundle',
        appKey: 'automation',
        platform: 'windows',
        artifactUrl: 'https://example.com/app-beta.exe',
        releaseVersion: '1.1.0-beta',
        requiresEntitlement: false,
      }),
    ];

    render(<DownloadSection downloads={[buildApp(undefined, platforms)]} />);

    const cards = screen.getAllByTestId(/download-card-/);
    expect(cards).toHaveLength(2);
    expect(cards[0]).not.toBe(cards[1]);
  });

  it('shows a helpful message when an artifact URL is missing', async () => {
    const apps: DownloadApp[] = [
      buildApp(undefined, [
        create(DownloadAssetSchema, {
          bundleKey: 'bundle',
          appKey: 'automation',
          platform: 'windows',
          artifactUrl: 'not-used',
          releaseVersion: '1.0.0',
          requiresEntitlement: false,
        }),
      ]),
    ];

    requestDownloadMock.mockResolvedValueOnce(
      create(DownloadAssetSchema, {
        bundleKey: 'bundle',
        appKey: 'automation',
        platform: 'windows',
        releaseVersion: '1.0.0',
        requiresEntitlement: false,
        artifactUrl: '   ',
      }),
    );

    window.open = vi.fn();

    render(<DownloadSection downloads={apps} />);

    const user = userEvent.setup();
    await user.click(screen.getByRole('button', { name: /download/i }));

    expect(await screen.findByText('Download artifact is not available yet. Please try again later.')).toBeInTheDocument();
    expect(window.open).not.toHaveBeenCalled();
  });

  it('warns when the browser blocks pop-ups', async () => {
    const app = buildApp(undefined, [
      create(DownloadAssetSchema, {
        bundleKey: 'bundle',
        appKey: 'automation',
        platform: 'mac',
        artifactUrl: 'https://example.com/app.dmg',
        releaseVersion: '1.2.3',
        requiresEntitlement: false,
      }),
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
      create(DownloadAssetSchema, {
        bundleKey: 'bundle',
        appKey: 'automation',
        platform: 'linux',
        artifactUrl: '/downloads/app.tar.gz',
        releaseVersion: '0.9.0',
        requiresEntitlement: false,
      }),
    ]);

    requestDownloadMock.mockResolvedValueOnce(
      create(DownloadAssetSchema, {
        bundleKey: 'bundle',
        appKey: 'automation',
        platform: 'linux',
        releaseVersion: '0.9.0',
        requiresEntitlement: false,
        artifactUrl: '/downloads/app.tar.gz',
      }),
    );

    window.open = vi.fn(() => ({} as Window));

    render(<DownloadSection downloads={[app]} />);

    const user = userEvent.setup();
    await user.click(screen.getByRole('button', { name: /download/i }));

    expect(window.open).toHaveBeenCalledWith('/downloads/app.tar.gz', '_blank', 'noopener,noreferrer');
    expect(await screen.findByText('Download started in a new tab.')).toBeInTheDocument();
  });

  it('rejects dangerous artifact URL schemes before opening a new window', async () => {
    const app = buildApp(undefined, [
      create(DownloadAssetSchema, {
        bundleKey: 'bundle',
        appKey: 'automation',
        platform: 'mac',
        artifactUrl: 'placeholder',
        releaseVersion: '2.0.0',
        requiresEntitlement: false,
      }),
    ]);

    requestDownloadMock.mockResolvedValueOnce(
      create(DownloadAssetSchema, {
        bundleKey: 'bundle',
        appKey: 'automation',
        platform: 'mac',
        releaseVersion: '2.0.0',
        requiresEntitlement: false,
        artifactUrl: 'javascript:alert(1)',
      }),
    );

    window.open = vi.fn(() => ({} as Window));

    render(<DownloadSection downloads={[app]} />);

    const user = userEvent.setup();
    await user.click(screen.getByRole('button', { name: /download/i }));

    expect(await screen.findByText('Download artifact is not available yet. Please try again later.')).toBeInTheDocument();
    expect(window.open).not.toHaveBeenCalled();
  });
});
