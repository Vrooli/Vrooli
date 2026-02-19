import { describe, it, expect, vi, beforeEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import App from './App';
import * as api from './lib/api';
import {
  createCheckHistoryResponse,
  createCheckInfo,
  createHealthResult,
  renderWithProviders,
  createStatusResponse,
  createTimelineResponse,
  createUptimeStatsResponse,
} from './test-utils';

vi.mock("./shared/contexts/CheckMetadataContext", async () => {
  const { useMockCheckMetadata } = await import(
    "./test-utils/mocks/checkMetadataContext"
  );
  return {
    useCheckMetadata: useMockCheckMetadata,
  };
});

// Mock API calls used by App while preserving the rest of the module exports.
vi.mock('./lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import("./lib/api")>();
  return {
    ...actual,
    fetchStatus: vi.fn(),
    fetchChecks: vi.fn(),
    runTick: vi.fn(),
    fetchTimeline: vi.fn(),
    fetchUptimeStats: vi.fn(),
    fetchCheckHistory: vi.fn(),
  };
});

const mockTimelineResponse = createTimelineResponse({
  events: [
    { checkId: 'infra-network', status: 'ok', message: 'Network OK', timestamp: new Date().toISOString() },
    { checkId: 'infra-dns', status: 'ok', message: 'DNS OK', timestamp: new Date().toISOString() },
  ],
  summary: { ok: 2, warning: 0, critical: 0 },
});

const mockUptimeStatsResponse = createUptimeStatsResponse({
  okEvents: 90,
  warningEvents: 10,
  criticalEvents: 0,
  uptimePercentage: 90.0,
});

const mockChecksMetadata: api.CheckInfo[] = [
  createCheckInfo({ id: 'infra-network' }),
  createCheckInfo({
    id: 'infra-dns',
    title: 'DNS Resolution',
    description: 'DNS resolution check',
    importance: 'Required for hostname resolution',
  }),
  createCheckInfo({
    id: 'infra-docker',
    title: 'Docker Engine',
    description: 'Docker daemon health',
    importance: 'Required for containers',
    intervalSeconds: 60,
  }),
  createCheckInfo({
    id: 'infra-cloudflared',
    title: 'Cloudflare Tunnel',
    description: 'Cloudflared tunnel health',
    importance: 'Required for external access',
    intervalSeconds: 60,
  }),
  createCheckInfo({
    id: 'infra-rdp',
    title: 'Remote Desktop',
    description: 'Remote desktop service health',
    importance: 'Required for RDP access',
    intervalSeconds: 60,
  }),
];

const mockStatusResponse = createStatusResponse({
  summary: {
    total: 5,
    ok: 4,
    warning: 1,
    critical: 0,
  },
  checks: [
    createHealthResult({
      checkId: 'infra-network',
      message: 'Network connectivity OK',
      duration: 10,
    }),
    createHealthResult({
      checkId: 'infra-dns',
      message: 'DNS resolution OK',
      duration: 15,
    }),
    createHealthResult({
      checkId: 'infra-docker',
      message: 'Docker daemon is healthy',
      duration: 30,
    }),
    createHealthResult({
      checkId: 'infra-cloudflared',
      message: 'Cloudflared is healthy',
      duration: 5,
    }),
    createHealthResult({
      checkId: 'infra-rdp',
      status: 'warning',
      message: 'xrdp service not active',
      duration: 3,
    }),
  ],
});

describe('App', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Default mocks for all API calls - can be overridden in specific tests
    vi.mocked(api.fetchChecks).mockResolvedValue(mockChecksMetadata);
    vi.mocked(api.fetchTimeline).mockResolvedValue(mockTimelineResponse);
    vi.mocked(api.fetchUptimeStats).mockResolvedValue(mockUptimeStatsResponse);
    vi.mocked(api.fetchCheckHistory).mockResolvedValue(createCheckHistoryResponse({ checkId: 'test' }));
  });

  it('[REQ:UI-HEALTH-001] renders loading state initially', () => {
    vi.mocked(api.fetchStatus).mockImplementation(() => new Promise(() => {}));

    renderWithProviders(<App />);

    expect(screen.getByText(/loading health status/i)).toBeInTheDocument();
  });

  it('[REQ:UI-HEALTH-001] displays health status when data is loaded', async () => {
    vi.mocked(api.fetchStatus).mockResolvedValue(mockStatusResponse);

    renderWithProviders(<App />);

    // Wait for content to load
    await waitFor(() => {
      expect(screen.getByText('Vrooli Autoheal')).toBeInTheDocument();
    }, { timeout: 2000 });

    // Check summary cards are displayed - use getAllByText since 'Healthy' appears multiple times
    expect(screen.getByText('Total Checks')).toBeInTheDocument();
    expect(screen.getAllByText('Healthy').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Warnings').length).toBeGreaterThan(0);
  });

  it('[REQ:UI-HEALTH-002] displays status badge with correct status', async () => {
    vi.mocked(api.fetchStatus).mockResolvedValue(mockStatusResponse);

    renderWithProviders(<App />);

    await waitFor(() => {
      expect(screen.getByText('OK')).toBeInTheDocument();
    });
  });

  it('[REQ:UI-HEALTH-002] groups checks by severity', async () => {
    vi.mocked(api.fetchStatus).mockResolvedValue(mockStatusResponse);

    renderWithProviders(<App />);

    await waitFor(() => {
      expect(screen.getByText('Vrooli Autoheal')).toBeInTheDocument();
    }, { timeout: 2000 });

    // Check that warnings section appears (since we have one warning)
    expect(screen.getAllByText('Warnings').length).toBeGreaterThan(0);
  });

  it('[REQ:UI-REFRESH-001] removes header auto-refresh toggle and keeps settings access', async () => {
    vi.mocked(api.fetchStatus).mockResolvedValue(mockStatusResponse);

    renderWithProviders(<App />);

    await waitFor(() => {
      expect(screen.getByTestId('settings-button')).toBeInTheDocument();
      expect(
        screen.queryByRole('button', { name: /enable auto refresh|disable auto refresh/i }),
      ).not.toBeInTheDocument();
    });
  });

  it('shows run tick button', async () => {
    vi.mocked(api.fetchStatus).mockResolvedValue(mockStatusResponse);

    renderWithProviders(<App />);

    await waitFor(() => {
      expect(screen.getByText('Run Tick')).toBeInTheDocument();
    });
  });

  // Platform info display
  it('displays platform information', async () => {
    vi.mocked(api.fetchStatus).mockResolvedValue(mockStatusResponse);

    renderWithProviders(<App />);

    await waitFor(() => {
      expect(screen.getByText('Platform')).toBeInTheDocument();
      expect(screen.getByText('linux')).toBeInTheDocument();
    });
  });

  it('shows error state when API fails', async () => {
    vi.mocked(api.fetchStatus).mockRejectedValue(new Error('Network error'));

    renderWithProviders(<App />);

    await waitFor(() => {
      expect(screen.getByText(/connection error/i)).toBeInTheDocument();
    });
  });

  it('shows retry button on error', async () => {
    vi.mocked(api.fetchStatus).mockRejectedValue(new Error('Network error'));

    renderWithProviders(<App />);

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /retry/i })).toBeInTheDocument();
    });
  });

  it('[REQ:UI-RESPONSIVE-001] renders with responsive grid layout', async () => {
    vi.mocked(api.fetchStatus).mockResolvedValue(mockStatusResponse);

    renderWithProviders(<App />);

    await waitFor(() => {
      expect(screen.getByText('Vrooli Autoheal')).toBeInTheDocument();
    }, { timeout: 2000 });

    // Verify responsive container exists with min-h-screen (responsive) class
    const container = screen.getByTestId('autoheal-dashboard');
    expect(container).toHaveClass('min-h-screen');
  });

  it('[REQ:UI-EVENTS-001] shows events timeline section', async () => {
    vi.mocked(api.fetchStatus).mockResolvedValue(mockStatusResponse);

    renderWithProviders(<App />);

    await waitFor(() => {
      expect(screen.getByText('Recent Events')).toBeInTheDocument();
    }, { timeout: 2000 });
  });

  it('[REQ:PERSIST-HISTORY-001] shows uptime statistics', async () => {
    vi.mocked(api.fetchStatus).mockResolvedValue(mockStatusResponse);

    renderWithProviders(<App />);

    await waitFor(() => {
      expect(screen.getByText(/Uptime/)).toBeInTheDocument();
    }, { timeout: 2000 });

    // Check that uptime percentage is displayed
    await waitFor(() => {
      expect(screen.getByText('90.0%')).toBeInTheDocument();
    });
  });

  it('[REQ:UI-EVENTS-001] shows events filter button', async () => {
    vi.mocked(api.fetchStatus).mockResolvedValue(mockStatusResponse);

    renderWithProviders(<App />);

    await waitFor(() => {
      expect(screen.getByText('All')).toBeInTheDocument();
    }, { timeout: 2000 });
  });
});
