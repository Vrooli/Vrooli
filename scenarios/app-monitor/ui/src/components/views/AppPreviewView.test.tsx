import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import AppPreviewView from './AppPreviewView';

const {
  sendBeaconMock,
  getAppMock,
  setAppsStateMock,
  runComplianceCheckMock,
  registerOverlayHostMock,
  setSurfaceScreenshotMock,
} = vi.hoisted(() => ({
  sendBeaconMock: vi.fn(),
  getAppMock: vi.fn(),
  setAppsStateMock: vi.fn(),
  runComplianceCheckMock: vi.fn().mockResolvedValue({ ok: true, failures: [], checkedAt: Date.now() }),
  registerOverlayHostMock: vi.fn(),
  setSurfaceScreenshotMock: vi.fn(),
}));

vi.mock('@/services/logger', () => ({
  logger: {
    debug: vi.fn(),
    warn: vi.fn(),
    info: vi.fn(),
    error: vi.fn(),
  },
}));

vi.mock('@/services/api', () => ({
  appService: {
    getApp: getAppMock,
    controlApp: vi.fn().mockResolvedValue(true),
  },
}));

const appsStoreState = {
  apps: [{
    id: 'scenario-1',
    name: 'Scenario One',
    scenario_name: 'scenario-1',
    path: '/tmp/scenario-1',
    created_at: '2026-02-07T00:00:00Z',
    updated_at: '2026-02-07T00:00:00Z',
    status: 'running',
    port_mappings: { UI_PORT: 4310 },
    environment: {},
    config: {},
    is_partial: false,
  }],
  setAppsState: setAppsStateMock,
  loadApps: vi.fn().mockResolvedValue(undefined),
  loadingInitial: false,
  hasInitialized: true,
};

vi.mock('@/state/appsStore', () => ({
  useAppsStore: (selector: (state: typeof appsStoreState) => unknown) => selector(appsStoreState),
}));

vi.mock('@/state/shellOverlayStore', () => ({
  useShellOverlayStore: (selector: (state: { activeView: string | null; registerHost: (node: HTMLElement | null) => void }) => unknown) => selector({
    activeView: null,
    registerHost: registerOverlayHostMock,
  }),
}));

vi.mock('@/state/surfaceMediaStore', () => ({
  useSurfaceMediaStore: (selector: (state: { setScreenshot: (...args: unknown[]) => void }) => unknown) => selector({
    setScreenshot: setSurfaceScreenshotMock,
  }),
}));

vi.mock('@/hooks/useOverlayRouter', () => ({
  useOverlayRouter: () => ({
    openOverlay: vi.fn(),
    closeOverlay: vi.fn(),
  }),
}));

vi.mock('@/hooks/useTimeout', () => ({
  useScheduledTimeout: () => ({
    schedule: vi.fn(),
    clear: vi.fn(),
  }),
}));

vi.mock('@/hooks/useKeyboardScopes', () => ({
  useKeyboardScope: vi.fn(),
}));

vi.mock('@/hooks/usePreviewNavigationSession', async () => {
  const React = await import('react');
  return {
    usePreviewNavigationSession: () => {
    const [previewUrl, setPreviewUrl] = React.useState<string | null>('http://localhost:4310');
    const [previewUrlInput, setPreviewUrlInput] = React.useState<string>('http://localhost:4310');
    const [hasCustomPreviewUrl, setHasCustomPreviewUrl] = React.useState(false);
    const [history, setHistory] = React.useState<string[]>(['http://localhost:4310']);
    const [historyIndex, setHistoryIndex] = React.useState(0);
    const initialPreviewUrlRef = React.useRef<string | null>('http://localhost:4310');

    return {
      bridge: {
        state: {
          isSupported: true,
          isReady: true,
          href: null,
          caps: [],
        },
        runComplianceCheck: runComplianceCheckMock,
        resetState: vi.fn(),
        requestScreenshot: vi.fn(),
        logState: null,
        requestLogBatch: vi.fn(),
        getRecentLogs: vi.fn(() => []),
        configureLogs: vi.fn(),
        networkState: null,
        requestNetworkBatch: vi.fn(),
        getRecentNetworkEvents: vi.fn(() => []),
        configureNetwork: vi.fn(),
        inspectState: { supported: false, active: false },
        startInspect: vi.fn(),
        stopInspect: vi.fn(),
        setInspectTargetIndex: vi.fn(),
        shiftInspectTarget: vi.fn(),
      },
      previewUrl,
      setPreviewUrl,
      previewUrlInput,
      hasCustomPreviewUrl,
      initialPreviewUrlRef,
      clearNavigationSession: vi.fn(),
      canGoBack: false,
      canGoForward: false,
      history,
      handleUrlInputChange: (event: { target: { value: string } }) => {
        setPreviewUrlInput(event.target.value);
      },
      handleUrlInputKeyDown: vi.fn(),
      handleUrlInputBlur: () => {
        const trimmed = previewUrlInput.trim();
        if (!trimmed) {
          return;
        }
        setHasCustomPreviewUrl(true);
        setPreviewUrl(trimmed);
        setPreviewUrlInput(trimmed);
        const nextHistory = [...history.slice(0, historyIndex + 1), trimmed];
        setHistory(nextHistory);
        setHistoryIndex(nextHistory.length - 1);
      },
      handleGoBack: vi.fn(),
      handleGoForward: vi.fn(),
      resetPreviewState: vi.fn(),
      applyDefaultPreviewUrl: (url: string) => {
        setHasCustomPreviewUrl(false);
        setPreviewUrl(url);
        setPreviewUrlInput(url);
        initialPreviewUrlRef.current = url;
      },
      applyPreviewUrlValue: (value: string) => {
        const trimmed = value.trim();
        if (!trimmed) {
          return;
        }
        setHasCustomPreviewUrl(true);
        setPreviewUrl(trimmed);
        setPreviewUrlInput(trimmed);
      },
    };
  },
  };
});

vi.mock('@/hooks/usePreviewAppLifecycle', () => ({
  usePreviewAppLifecycle: () => ({
    runAction: vi.fn().mockResolvedValue(true),
    actionInProgress: false,
    isAppRunning: true,
    appStatusLabel: 'running',
    pendingAction: null,
    toggleActionLabel: 'Stop',
    restartActionLabel: 'Restart',
    urlStatusClass: 'ok',
  }),
}));

vi.mock('@/hooks/usePreviewInteractionTracking', () => ({
  usePreviewInteractionTracking: () => ({ previewInteractionSignal: 0 }),
}));

vi.mock('@/hooks/useAppViewRecording', () => ({
  useAppViewRecording: vi.fn(),
}));

vi.mock('@/hooks/useAppInsights', () => ({
  useAppInsights: () => ({
    diagnostics: null,
    diagnosticsLoading: false,
    diagnosticsError: null,
    lighthouseHistory: null,
    lighthouseLoading: false,
    lighthouseError: null,
    completeness: null,
    completenessLoading: false,
    completenessError: null,
    proxyMetadata: null,
    localhostReport: null,
    proxyLoading: false,
    proxyError: null,
    refetchDiagnostics: vi.fn(),
    refetchLighthouse: vi.fn(),
    refetchCompleteness: vi.fn(),
    refetchProxy: vi.fn(),
    prefetch: vi.fn(),
  }),
}));

vi.mock('@/hooks/usePreviewCapture', () => ({
  usePreviewCapture: vi.fn(),
}));

vi.mock('@/hooks/usePreviewOverlay', () => ({
  usePreviewOverlay: () => ({ previewOverlay: null, setPreviewOverlay: vi.fn(), fallbackState: null }),
}));

vi.mock('@/hooks/usePreviewBackgroundColor', () => ({
  usePreviewBackgroundColor: () => vi.fn(() => '#111'),
}));

vi.mock('@/hooks/useAppLifecycleMonitor', () => ({
  useAppLifecycleMonitor: () => ({
    beginLifecycleMonitor: vi.fn(),
    stopLifecycleMonitor: vi.fn(),
  }),
}));

vi.mock('@/hooks/useDeviceEmulation', () => ({
  useDeviceEmulation: () => ({
    isActive: false,
    toggleActive: vi.fn(),
    toolbar: {},
    viewport: {},
  }),
}));

vi.mock('@/hooks/useAppLogs', () => ({
  useAppLogs: () => ({}),
}));

vi.mock('./usePreviewInspector', () => ({
  default: () => ({
    handleToggleInspectMode: vi.fn(),
    handleInspectorDialogClose: vi.fn(),
  }),
}));

vi.mock('@/hooks/usePreviewToolbarSession', () => ({
  usePreviewToolbarSession: () => ({
    openPreviewTarget: '',
    urlSuggestions: [],
    handleOpenScenarioSelector: vi.fn(),
    handleOpenPreviewInNewTab: vi.fn(),
  }),
}));

vi.mock('../ErrorBoundary', () => ({
  default: ({ children }: { children: ReactNode }) => <>{children}</>,
  SectionErrorFallback: () => null,
}));

vi.mock('../AppModal', () => ({
  default: () => null,
}));

vi.mock('../AppPreviewToolbar', () => ({
  default: (props: {
    previewUrlInput: string;
    onPreviewUrlInputChange: (event: { target: { value: string } }) => void;
    onPreviewUrlInputBlur: () => void;
  }) => (
    <div data-testid="app-preview-toolbar">
      <input
        aria-label="Preview URL Input"
        value={props.previewUrlInput}
        onChange={(event) => props.onPreviewUrlInputChange({ target: { value: event.currentTarget.value } })}
        onBlur={() => props.onPreviewUrlInputBlur()}
      />
    </div>
  ),
}));

vi.mock('../report/ReportIssueDialog', () => ({
  default: () => null,
}));

vi.mock('./PreviewInspectorPanel', () => ({
  default: () => null,
}));

vi.mock('../device-emulation/DeviceEmulationToolbar', () => ({
  default: () => null,
}));

vi.mock('../device-emulation/DeviceEmulationViewport', () => ({
  default: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

vi.mock('../device-emulation/DeviceVisionFilterDefs', () => ({
  default: () => null,
}));

vi.mock('../logs/AppLogsPanel', () => ({
  default: () => <div data-testid="logs-panel">Logs</div>,
}));

const renderPreviewView = (initialPath: string) => render(
  <MemoryRouter initialEntries={[initialPath]}>
    <Routes>
      <Route path="/apps/:appId/preview" element={<AppPreviewView />} />
    </Routes>
  </MemoryRouter>,
);

describe('AppPreviewView', () => {
  beforeEach(() => {
    sendBeaconMock.mockReset();
    getAppMock.mockReset();
    setAppsStateMock.mockReset();
    runComplianceCheckMock.mockClear();
    registerOverlayHostMock.mockReset();
    setSurfaceScreenshotMock.mockReset();

    Object.defineProperty(navigator, 'sendBeacon', {
      configurable: true,
      value: sendBeaconMock,
    });

    window.localStorage.removeItem('app-monitor:debug-preview-events');
  });

  it('renders preview toolbar for a routed app', async () => {
    renderPreviewView('/apps/scenario-1/preview');
    await waitFor(() => {
      expect(screen.getByTestId('app-preview-toolbar')).toBeInTheDocument();
    });
  });

  it('does not emit debug beacons when preview debug mode is disabled', async () => {
    renderPreviewView('/apps/scenario-1/preview');

    await waitFor(() => {
      expect(screen.getByTestId('app-preview-toolbar')).toBeInTheDocument();
    });
    expect(sendBeaconMock).not.toHaveBeenCalled();
  });

  it('emits debug beacons when preview debug mode is enabled', async () => {
    window.localStorage.setItem('app-monitor:debug-preview-events', '1');
    renderPreviewView('/apps/scenario-1/preview');

    await waitFor(() => {
      expect(sendBeaconMock).toHaveBeenCalled();
    });
  });

  it('keeps custom URL after blur without snapping back to default URL', async () => {
    renderPreviewView('/apps/scenario-1/preview');

    const input = await screen.findByRole('textbox', { name: 'Preview URL Input' });

    await waitFor(() => {
      expect((input as HTMLInputElement).value).toContain('/apps/scenario-1/proxy/');
    });

    fireEvent.change(input, { target: { value: 'https://example.com/custom' } });
    fireEvent.blur(input);

    await waitFor(() => {
      expect((input as HTMLInputElement).value).toBe('https://example.com/custom');
    });
  });
});
