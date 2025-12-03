// SystemProtection component tests
// [REQ:WATCH-DETECT-001]
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { SystemProtection } from './SystemProtection';
import * as api from '../lib/api';

// Mock the API module
vi.mock('../lib/api', () => ({
  fetchWatchdogStatus: vi.fn(),
  fetchWatchdogTemplate: vi.fn(),
}));

const mockWatchdogStatusFull: api.WatchdogStatus = {
  loopRunning: true,
  watchdogType: 'systemd',
  watchdogInstalled: true,
  watchdogEnabled: true,
  watchdogRunning: true,
  bootProtectionActive: true,
  canInstall: true,
  servicePath: '/etc/systemd/system/vrooli-autoheal.service',
  protectionLevel: 'full',
};

const mockWatchdogStatusPartial: api.WatchdogStatus = {
  loopRunning: true,
  watchdogType: '',
  watchdogInstalled: false,
  watchdogEnabled: false,
  watchdogRunning: false,
  bootProtectionActive: false,
  canInstall: true,
  protectionLevel: 'partial',
};

const mockWatchdogStatusNone: api.WatchdogStatus = {
  loopRunning: false,
  watchdogType: '',
  watchdogInstalled: false,
  watchdogEnabled: false,
  watchdogRunning: false,
  bootProtectionActive: false,
  canInstall: false,
  protectionLevel: 'none',
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

describe('SystemProtection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // [REQ:WATCH-DETECT-001] Display protection status
  it('renders loading state initially', () => {
    vi.mocked(api.fetchWatchdogStatus).mockImplementation(() => new Promise(() => {}));

    renderWithProviders(<SystemProtection />);

    expect(screen.getByText(/loading/i)).toBeInTheDocument();
  });

  it('displays full protection status correctly', async () => {
    vi.mocked(api.fetchWatchdogStatus).mockResolvedValue(mockWatchdogStatusFull);

    renderWithProviders(<SystemProtection />);

    await waitFor(() => {
      expect(screen.getByText('Full Protection')).toBeInTheDocument();
    });

    // Check individual status indicators
    expect(screen.getByText('Autoheal Loop')).toBeInTheDocument();
    expect(screen.getByText('OS Watchdog')).toBeInTheDocument();
    expect(screen.getByText('Boot Recovery')).toBeInTheDocument();
  });

  it('displays partial protection status correctly', async () => {
    vi.mocked(api.fetchWatchdogStatus).mockResolvedValue(mockWatchdogStatusPartial);

    renderWithProviders(<SystemProtection />);

    await waitFor(() => {
      expect(screen.getByText('Partial Protection')).toBeInTheDocument();
    });

    // Should show install button when not installed but can install
    expect(screen.getByText('Set up OS Watchdog')).toBeInTheDocument();
  });

  it('displays no protection status correctly', async () => {
    vi.mocked(api.fetchWatchdogStatus).mockResolvedValue(mockWatchdogStatusNone);

    renderWithProviders(<SystemProtection />);

    await waitFor(() => {
      expect(screen.getByText('Not Protected')).toBeInTheDocument();
    });

    // Should not show install button when can't install
    expect(screen.queryByText('Set up OS Watchdog')).not.toBeInTheDocument();
    expect(screen.getByText(/not available on this platform/i)).toBeInTheDocument();
  });

  it('renders compact mode correctly', async () => {
    vi.mocked(api.fetchWatchdogStatus).mockResolvedValue(mockWatchdogStatusFull);

    renderWithProviders(<SystemProtection compact />);

    // Wait for data to load and compact element to appear
    await waitFor(() => {
      expect(screen.getByTestId('autoheal-system-protection-compact')).toBeInTheDocument();
    });

    // Compact mode should not show the full labels
    expect(screen.queryByText('Full Protection')).not.toBeInTheDocument();
  });

  it('shows service path when installed', async () => {
    vi.mocked(api.fetchWatchdogStatus).mockResolvedValue(mockWatchdogStatusFull);

    renderWithProviders(<SystemProtection />);

    await waitFor(() => {
      // Check for path display - there may be multiple systemd matches so use getAllByText
      const pathElements = screen.getAllByText(/systemd/);
      expect(pathElements.length).toBeGreaterThan(0);
    });
  });

  it('shows error when API fails', async () => {
    vi.mocked(api.fetchWatchdogStatus).mockRejectedValue(new Error('Network error'));

    renderWithProviders(<SystemProtection />);

    await waitFor(() => {
      expect(screen.getByText(/retry/i)).toBeInTheDocument();
    });
  });

  it('shows last error from status', async () => {
    const statusWithError: api.WatchdogStatus = {
      ...mockWatchdogStatusPartial,
      lastError: 'systemd not available',
    };
    vi.mocked(api.fetchWatchdogStatus).mockResolvedValue(statusWithError);

    renderWithProviders(<SystemProtection />);

    await waitFor(() => {
      expect(screen.getByText('systemd not available')).toBeInTheDocument();
    });
  });
});

describe('SystemProtection - compact mode', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows green indicator for full protection', async () => {
    vi.mocked(api.fetchWatchdogStatus).mockResolvedValue(mockWatchdogStatusFull);

    const { container } = renderWithProviders(<SystemProtection compact />);

    await waitFor(() => {
      const compactElement = container.querySelector('[data-testid="autoheal-system-protection-compact"]');
      expect(compactElement).toHaveClass('bg-emerald-500/20');
    });
  });

  it('shows amber indicator for partial protection', async () => {
    vi.mocked(api.fetchWatchdogStatus).mockResolvedValue(mockWatchdogStatusPartial);

    const { container } = renderWithProviders(<SystemProtection compact />);

    await waitFor(() => {
      const compactElement = container.querySelector('[data-testid="autoheal-system-protection-compact"]');
      expect(compactElement).toHaveClass('bg-amber-500/20');
    });
  });

  it('shows red indicator for no protection', async () => {
    vi.mocked(api.fetchWatchdogStatus).mockResolvedValue(mockWatchdogStatusNone);

    const { container } = renderWithProviders(<SystemProtection compact />);

    await waitFor(() => {
      const compactElement = container.querySelector('[data-testid="autoheal-system-protection-compact"]');
      expect(compactElement).toHaveClass('bg-red-500/20');
    });
  });
});
