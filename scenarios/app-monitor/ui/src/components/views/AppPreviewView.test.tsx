import { render, screen, waitFor } from '@testing-library/react';
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

vi.mock('@/state/scenarioEngagementStore', () => ({
  useScenarioEngagementStore: (selector: (state: { beginSession: (...args: unknown[]) => void; endSession: (...args: unknown[]) => void }) => unknown) => selector({
    beginSession: vi.fn(),
    endSession: vi.fn(),
  }),
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

vi.mock('@/hooks/useAutoNextScenario', () => ({
  useAutoNextScenario: () => ({
    prepareAutoNext: vi.fn().mockResolvedValue(undefined),
  }),
}));

vi.mock('@/hooks/useOverlayRouter', () => ({
  useOverlayRouter: () => ({
    openOverlay: vi.fn(),
    closeOverlay: vi.fn(),
  }),
}));

vi.mock('@/hooks/useScheduledTimeout', () => ({
  useScheduledTimeout: () => ({
    schedule: vi.fn(),
    clear: vi.fn(),
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

vi.mock('./useIosAutobackGuard', () => ({
  useIosAutobackGuard: vi.fn(),
  isIosSafariUserAgent: () => false,
}));

vi.mock('@/hooks/usePreviewNavigationSession', () => ({
  usePreviewNavigationSession: () => ({
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
    previewUrl: null,
    setPreviewUrl: vi.fn(),
    previewUrlInput: '',
    hasCustomPreviewUrl: false,
    initialPreviewUrlRef: { current: null },
    clearNavigationSession: vi.fn(),
    canGoBack: false,
    canGoForward: false,
    history: [],
    handleUrlInputChange: vi.fn(),
    handleUrlInputKeyDown: vi.fn(),
    handleUrlInputBlur: vi.fn(),
    handleGoBack: vi.fn(),
    handleGoForward: vi.fn(),
    resetPreviewState: vi.fn(),
    applyDefaultPreviewUrl: vi.fn(),
    applyPreviewUrlValue: vi.fn(),
  }),
}));

vi.mock('@/hooks/usePreviewUrlOrchestration', () => ({
  usePreviewUrlOrchestration: () => (({ appForPreview }: { appForPreview: unknown }) => ({
    hasPreviewCandidate: Boolean(appForPreview),
    defaultPreviewUrl: null,
  })),
}));

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

vi.mock('@/hooks/useProxyMetadataSynchronization', () => ({
  useProxyMetadataSynchronization: () => ({ proxyMetadata: null, localhostReport: null }),
}));

vi.mock('@/hooks/useAppViewRecording', () => ({
  useAppViewRecording: vi.fn(),
}));

vi.mock('@/hooks/useAppDiagnostics', () => ({
  useAppDiagnostics: () => ({ diagnostics: null, loading: false, error: null, refetch: vi.fn() }),
}));

vi.mock('@/hooks/useLighthouseHistory', () => ({
  useLighthouseHistory: () => ({ history: [], loading: false, error: null, refetch: vi.fn() }),
}));

vi.mock('@/hooks/useAppCompleteness', () => ({
  useAppCompleteness: () => ({ completeness: null, loading: false, error: null, refetch: vi.fn() }),
}));

vi.mock('@/hooks/usePreviewCapture', () => ({
  usePreviewCapture: vi.fn(),
}));

vi.mock('@/hooks/usePreviewOverlay', () => ({
  usePreviewOverlay: () => ({ previewOverlay: null, setPreviewOverlay: vi.fn() }),
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
  default: () => <div data-testid="app-preview-toolbar">Toolbar</div>,
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
});
