// [REQ:UI-HEALTH-001] [REQ:UI-HEALTH-002] [REQ:UI-REFRESH-001] [REQ:UI-RESPONSIVE-001]
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import App from './App';
import * as api from './lib/api';

// Mock the API module - include helper functions that are used directly in App.tsx
vi.mock('./lib/api', () => ({
  fetchStatus: vi.fn(),
  fetchChecks: vi.fn(),
  runTick: vi.fn(),
  fetchTimeline: vi.fn(),
  fetchUptimeStats: vi.fn(),
  fetchCheckHistory: vi.fn(),
  // Include the helper functions that App.tsx uses
  groupChecksByStatus: (checks: { status: string }[]) => ({
    critical: checks.filter((c) => c.status === "critical"),
    warning: checks.filter((c) => c.status === "warning"),
    ok: checks.filter((c) => c.status === "ok"),
  }),
  statusToEmoji: (status: string) => {
    switch (status) {
      case "ok": return "\u2713";
      case "warning": return "\u26A0";
      case "critical": return "\u2717";
      default: return "\u2753";
    }
  },
}));

const mockTimelineResponse: api.TimelineResponse = {
  events: [
    { checkId: 'infra-network', status: 'ok', message: 'Network OK', timestamp: new Date().toISOString() },
    { checkId: 'infra-dns', status: 'ok', message: 'DNS OK', timestamp: new Date().toISOString() },
  ],
  count: 2,
  summary: { ok: 2, warning: 0, critical: 0 },
};

const mockUptimeStatsResponse: api.UptimeStatsResponse = {
  totalEvents: 100,
  okEvents: 90,
  warningEvents: 10,
  criticalEvents: 0,
  uptimePercentage: 90.0,
  windowHours: 24,
};

const mockChecksMetadata: api.CheckInfo[] = [
  { id: 'infra-network', description: 'Network connectivity check', intervalSeconds: 30 },
  { id: 'infra-dns', description: 'DNS resolution check', intervalSeconds: 30 },
  { id: 'infra-docker', description: 'Docker daemon health', intervalSeconds: 60 },
  { id: 'infra-cloudflared', description: 'Cloudflared tunnel health', intervalSeconds: 60 },
  { id: 'infra-rdp', description: 'Remote desktop service health', intervalSeconds: 60 },
];

const mockStatusResponse: api.StatusResponse = {
  status: 'ok',
  platform: {
    platform: 'linux',
    supportsRdp: false,
    supportsSystemd: true,
    supportsLaunchd: false,
    supportsWindowsServices: false,
    isHeadlessServer: false,
    hasDocker: true,
    isWsl: false,
    supportsCloudflared: true,
  },
  summary: {
    total: 5,
    ok: 4,
    warning: 1,
    critical: 0,
  },
  checks: [
    {
      checkId: 'infra-network',
      status: 'ok',
      message: 'Network connectivity OK',
      timestamp: new Date().toISOString(),
      duration: 10,
    },
    {
      checkId: 'infra-dns',
      status: 'ok',
      message: 'DNS resolution OK',
      timestamp: new Date().toISOString(),
      duration: 15,
    },
    {
      checkId: 'infra-docker',
      status: 'ok',
      message: 'Docker daemon is healthy',
      timestamp: new Date().toISOString(),
      duration: 30,
    },
    {
      checkId: 'infra-cloudflared',
      status: 'ok',
      message: 'Cloudflared is healthy',
      timestamp: new Date().toISOString(),
      duration: 5,
    },
    {
      checkId: 'infra-rdp',
      status: 'warning',
      message: 'xrdp service not active',
      timestamp: new Date().toISOString(),
      duration: 3,
    },
  ],
  timestamp: new Date().toISOString(),
};

function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
    },
  });
}

function renderWithProviders(ui: React.ReactElement) {
  const queryClient = createTestQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      {ui}
    </QueryClientProvider>
  );
}

describe('App', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Default mocks for all API calls - can be overridden in specific tests
    vi.mocked(api.fetchChecks).mockResolvedValue(mockChecksMetadata);
    vi.mocked(api.fetchTimeline).mockResolvedValue(mockTimelineResponse);
    vi.mocked(api.fetchUptimeStats).mockResolvedValue(mockUptimeStatsResponse);
    vi.mocked(api.fetchCheckHistory).mockResolvedValue({ checkId: 'test', history: [], count: 0 });
  });

  // [REQ:UI-HEALTH-001] Dashboard shows current status for all checks
  it('renders loading state initially', () => {
    vi.mocked(api.fetchStatus).mockImplementation(() => new Promise(() => {}));

    renderWithProviders(<App />);

    expect(screen.getByText(/loading health status/i)).toBeInTheDocument();
  });

  // [REQ:UI-HEALTH-001] Dashboard shows current status
  it('displays health status when data is loaded', async () => {
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

  // [REQ:UI-HEALTH-002] Status color coding (green for ok, amber for warning, red for critical)
  it('displays status badge with correct status', async () => {
    vi.mocked(api.fetchStatus).mockResolvedValue(mockStatusResponse);

    renderWithProviders(<App />);

    await waitFor(() => {
      expect(screen.getByText('OK')).toBeInTheDocument();
    });
  });

  // [REQ:UI-HEALTH-002] Color coding for different statuses
  it('groups checks by severity', async () => {
    vi.mocked(api.fetchStatus).mockResolvedValue(mockStatusResponse);

    renderWithProviders(<App />);

    await waitFor(() => {
      expect(screen.getByText('Vrooli Autoheal')).toBeInTheDocument();
    }, { timeout: 2000 });

    // Check that warnings section appears (since we have one warning)
    expect(screen.getAllByText('Warnings').length).toBeGreaterThan(0);
  });

  // [REQ:UI-REFRESH-001] Auto-refresh status
  it('shows auto-refresh toggle button', async () => {
    vi.mocked(api.fetchStatus).mockResolvedValue(mockStatusResponse);

    renderWithProviders(<App />);

    await waitFor(() => {
      expect(screen.getByText('Auto')).toBeInTheDocument();
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

  // [REQ:UI-RESPONSIVE-001] Responsive design
  it('renders with responsive grid layout', async () => {
    vi.mocked(api.fetchStatus).mockResolvedValue(mockStatusResponse);

    renderWithProviders(<App />);

    await waitFor(() => {
      expect(screen.getByText('Vrooli Autoheal')).toBeInTheDocument();
    }, { timeout: 2000 });

    // Verify responsive container exists with min-h-screen (responsive) class
    const container = screen.getByTestId('autoheal-dashboard');
    expect(container).toHaveClass('min-h-screen');
  });

  // [REQ:UI-EVENTS-001] Recent events timeline
  it('shows events timeline section', async () => {
    vi.mocked(api.fetchStatus).mockResolvedValue(mockStatusResponse);

    renderWithProviders(<App />);

    await waitFor(() => {
      expect(screen.getByText('Recent Events')).toBeInTheDocument();
    }, { timeout: 2000 });
  });

  // [REQ:PERSIST-HISTORY-001] Uptime statistics
  it('shows uptime statistics', async () => {
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

  // [REQ:UI-EVENTS-001] Events timeline filter
  it('shows events filter button', async () => {
    vi.mocked(api.fetchStatus).mockResolvedValue(mockStatusResponse);

    renderWithProviders(<App />);

    await waitFor(() => {
      expect(screen.getByText('All')).toBeInTheDocument();
    }, { timeout: 2000 });
  });
});
