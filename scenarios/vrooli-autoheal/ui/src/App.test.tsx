import { describe, it, expect, vi, beforeEach } from 'vitest';
import { cleanup, fireEvent, screen, waitFor } from '@testing-library/react';
import App from './App';
import * as api from './lib/api';
import { selectors } from './consts/selectors';
import {
  createCheckHistoryResponse,
  createCheckInfo,
  createHealthResult,
  renderWithProviders,
  createStatusResponse,
  createTimelineResponse,
  createUptimeStatsResponse,
  expectNoA11yViolations,
} from './test-utils';

vi.mock("./shared/contexts/CheckMetadataContext", async () => {
  const { useMockCheckMetadata } = await import(
    "./test-utils/mocks/checkMetadataContext"
  );
  return {
    useCheckMetadata: useMockCheckMetadata,
  };
});

vi.mock("./surfaces/trends", () => ({ TrendsSurface: () => <div>Trends surface loaded</div> }));
vi.mock("./surfaces/timeline", () => ({ TimelineSurface: () => <div>Timeline surface loaded</div> }));
vi.mock("./surfaces/incidents", () => ({ IncidentsSurface: () => <div>Incidents surface loaded</div> }));
vi.mock("./surfaces/docs", () => ({ DocsSurface: () => <div>Docs surface loaded</div> }));

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
    vi.mocked(api.fetchStatus).mockResolvedValue(mockStatusResponse);
    vi.mocked(api.fetchChecks).mockResolvedValue(mockChecksMetadata);
    vi.mocked(api.fetchTimeline).mockResolvedValue(mockTimelineResponse);
    vi.mocked(api.fetchUptimeStats).mockResolvedValue(mockUptimeStatsResponse);
    vi.mocked(api.fetchCheckHistory).mockResolvedValue(createCheckHistoryResponse({ checkId: 'test' }));
    vi.mocked(api.runTick).mockResolvedValue({
      success: true,
      status: 'ok',
      summary: { total: 1, ok: 1, warning: 0, critical: 0 },
      results: [createHealthResult()],
      timestamp: new Date().toISOString(),
    });
  });

  it('has no axe-core violations in the default dashboard state', async () => {
    const { container } = renderWithProviders(<App />);
    await waitFor(() => expect(screen.getByRole('main')).toBeInTheDocument());
    await expectNoA11yViolations(container);
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
      expect(screen.getByTestId(selectors.settingsButton)).toBeInTheDocument();
      expect(
        screen.queryByRole('button', { name: /enable auto refresh|disable auto refresh/i }),
      ).not.toBeInTheDocument();
    });
  });

  it('shows run tick button', async () => {
    vi.mocked(api.fetchStatus).mockResolvedValue(mockStatusResponse);

    renderWithProviders(<App />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.runTickButton)).toBeInTheDocument();
      expect(screen.getByTestId(selectors.runTickButtonDesktop)).toHaveTextContent('Run Tick');
    });
  });

  it('shows running indicator when a tick is active externally', async () => {
    vi.mocked(api.fetchStatus).mockResolvedValue(
      createStatusResponse({ ...mockStatusResponse, tickRunning: true, tickStartedAt: new Date().toISOString() }),
    );

    renderWithProviders(<App />);

    await waitFor(() => {
      expect(screen.getByText('Health check cycle is currently running.')).toBeInTheDocument();
      expect(screen.getByText('Tick Running')).toBeInTheDocument();
    });
  });

  it('shows conflict feedback when tick is already running', async () => {
    vi.mocked(api.fetchStatus).mockResolvedValue(mockStatusResponse);
    vi.mocked(api.runTick).mockRejectedValue(
      new api.APIError(
        'A health check cycle is already running. Please wait for it to complete.',
        'CONFLICT',
        409,
        'req-1',
        { action: 'wait', retryable: true, hint: 'Wait for the current operation to complete, then try again.' }
      ),
    );

    renderWithProviders(<App />);

    await waitFor(() => {
      expect(screen.getByText('Run Tick')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId(selectors.runTickButton));

    await waitFor(() => {
      expect(screen.getByText('A health check cycle is already running.')).toBeInTheDocument();
      expect(screen.getByText('Wait for the current operation to complete, then try again.')).toBeInTheDocument();
    });
  });

  it('shows success feedback after a completed tick', async () => {
    renderWithProviders(<App />);
    await waitFor(() => expect(screen.getByTestId(selectors.runTickButton)).toBeInTheDocument());
    fireEvent.click(screen.getByTestId(selectors.runTickButton));
    await waitFor(() => expect(screen.getByText(/health check completed/i)).toBeInTheDocument());
  });

  it('explains gateway, API, and unexpected tick failures', async () => {
    const cases = [
      {
        error: new api.APIError("gateway", "BAD_GATEWAY", 502, "req-502", { action: "retry", retryable: true }),
        message: /upstream gateway error/i,
      },
      {
        error: new api.APIError("bad request", "BAD_REQUEST", 400, "req-400", { action: "report", retryable: false }),
        message: /something went wrong/i,
      },
      { error: new Error("unexpected failure"), message: /failed to run health check cycle/i },
    ] as const;

    for (const { error, message } of cases) {
      vi.mocked(api.runTick).mockRejectedValueOnce(error);
      renderWithProviders(<App />);
      await waitFor(() => expect(screen.getByTestId(selectors.runTickButton)).toBeInTheDocument());
      fireEvent.click(screen.getByTestId(selectors.runTickButton));
      await waitFor(() => expect(screen.getByText(message)).toBeInTheDocument());
      cleanup();
    }
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
    fireEvent.click(screen.getByRole('button', { name: /retry/i }));
  });

  it('[REQ:UI-RESPONSIVE-001] renders shell and summary content in a mobile-safe layout contract', async () => {
    vi.mocked(api.fetchStatus).mockResolvedValue(mockStatusResponse);

    renderWithProviders(<App />);

    await waitFor(() => {
      expect(screen.getByText('Vrooli Autoheal')).toBeInTheDocument();
    }, { timeout: 2000 });

    const container = screen.getByTestId(selectors.dashboard);
    const header = screen.getByTestId(selectors.shell.header);
    const content = screen.getByTestId(selectors.shell.content);
    const summaryGrid = screen.getByTestId(selectors.summary.grid);

    expect(container).toHaveClass('min-h-full', 'overflow-x-hidden');
    expect(header).toHaveClass('sticky', 'top-0');
    expect(content).toHaveClass('min-w-0');
    expect(summaryGrid).toHaveClass('grid-cols-2', 'md:grid-cols-4');
    expect(summaryGrid.compareDocumentPosition(header) & Node.DOCUMENT_POSITION_PRECEDING).toBeTruthy();
  });

  it('[REQ:UI-RESPONSIVE-001] keeps all primary tabs in the selector registry and scrollable shell nav', async () => {
    vi.mocked(api.fetchStatus).mockResolvedValue(mockStatusResponse);

    renderWithProviders(<App />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.shell.nav)).toBeInTheDocument();
    }, { timeout: 2000 });

    expect(screen.getByTestId(selectors.shell.nav)).toHaveClass('overflow-x-auto');
    expect(screen.getByTestId(selectors.tabs.dashboard)).toHaveTextContent('Dashboard');
    expect(screen.getByTestId(selectors.tabs.trends)).toHaveTextContent('Trends');
    expect(screen.getByTestId(selectors.tabs.timeline)).toHaveTextContent('Timeline');
    expect(screen.getByTestId(selectors.tabs.incidents)).toHaveTextContent('Incidents');
    expect(screen.getByTestId(selectors.tabs.docs)).toHaveTextContent('Docs');
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

  it('switches through every lazy tab surface', async () => {
    renderWithProviders(<App />);
    await waitFor(() => expect(screen.getByTestId(selectors.tabs.trends)).toBeInTheDocument());
    fireEvent.click(screen.getByTestId(selectors.settingsButton));
    expect(screen.getByTestId("settings-dialog")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("settings-close"));
    fireEvent.click(screen.getByTestId(selectors.uptimeStats));
    await waitFor(() => expect(screen.getByText("Trends surface loaded")).toBeInTheDocument());
    for (const [selector, text] of [
      [selectors.tabs.trends, 'Trends surface loaded'],
      [selectors.tabs.timeline, 'Timeline surface loaded'],
      [selectors.tabs.incidents, 'Incidents surface loaded'],
      [selectors.tabs.docs, 'Docs surface loaded'],
      [selectors.tabs.dashboard, 'Vrooli Autoheal'],
    ] as const) {
      fireEvent.click(screen.getByTestId(selector));
      await waitFor(() => expect(screen.getByText(text)).toBeInTheDocument());
    }
  });
});
