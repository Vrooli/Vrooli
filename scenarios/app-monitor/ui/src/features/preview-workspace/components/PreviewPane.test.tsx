import { beforeEach, describe, expect, it, vi } from 'vitest';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import type { MouseEvent as ReactMouseEvent, ReactNode } from 'react';
import type { App } from '@/types';
import { usePreviewWorkspaceStore } from '../state/previewWorkspaceStore';
import PreviewPane from './PreviewPane';

const {
  openOverlayMock,
  getAppMock,
  controlAppMock,
  bridgeHarness,
} = vi.hoisted(() => ({
  openOverlayMock: vi.fn(),
  getAppMock: vi.fn(),
  controlAppMock: vi.fn(),
  bridgeHarness: {
    onLocation: null as ((message: { href: string; title?: string | null }) => void) | null,
  },
}));

vi.mock('@/hooks/useOverlayRouter', () => ({
  useOverlayRouter: () => ({
    overlay: null,
    openOverlay: openOverlayMock,
    closeOverlay: vi.fn(),
  }),
}));

const setAppsStateMock = vi.fn();
vi.mock('@/state/appsStore', () => ({
  useAppsStore: (selector: (state: { setAppsState: typeof setAppsStateMock }) => unknown) => selector({
    setAppsState: setAppsStateMock,
  }),
}));

vi.mock('@/services/logger', () => ({
  logger: {
    debug: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
  },
}));

vi.mock('@/services/api', () => ({
  appService: {
    getApp: getAppMock,
    controlApp: controlAppMock,
  },
}));

vi.mock('@/hooks/useIframeBridge', () => ({
  useIframeBridge: (options: { onLocation?: (message: { href: string; title?: string | null }) => void }) => {
    bridgeHarness.onLocation = options.onLocation ?? null;
    return ({
    state: {
      isSupported: false,
      href: null,
      canGoBack: false,
      canGoForward: false,
      isReady: false,
      caps: [],
    },
    childOrigin: null,
    sendNav: vi.fn(),
    runComplianceCheck: vi.fn().mockResolvedValue({ ok: true, failures: [], checkedAt: Date.now() }),
    resetState: vi.fn(),
    requestScreenshot: vi.fn(),
    logState: null,
    networkState: null,
    subscribeLogs: vi.fn(() => () => {}),
    getRecentLogs: vi.fn(() => []),
    requestLogBatch: vi.fn().mockResolvedValue([]),
    configureLogs: vi.fn(() => false),
    subscribeNetwork: vi.fn(() => () => {}),
    getRecentNetworkEvents: vi.fn(() => []),
    requestNetworkBatch: vi.fn().mockResolvedValue([]),
    configureNetwork: vi.fn(() => false),
    inspectState: {
      supported: false,
      active: false,
      lastReason: null,
      hover: null,
      result: null,
      error: null,
    },
    startInspect: vi.fn(() => false),
    stopInspect: vi.fn(() => false),
    setInspectTargetIndex: vi.fn(() => false),
    shiftInspectTarget: vi.fn(() => false),
    });
  },
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

vi.mock('@/components/views/usePreviewInspector', () => ({
  default: () => ({
    handleToggleInspectMode: vi.fn(),
  }),
}));

vi.mock('@/components/AppModal', () => ({
  default: () => null,
}));

vi.mock('@/components/views/PreviewInspectorPanel', () => ({
  default: () => null,
}));

vi.mock('@/components/report/ReportIssueDialog', () => ({
  default: () => null,
}));

vi.mock('@/components/logs/AppLogsPanel', () => ({
  default: () => <div data-testid="app-logs-panel">Logs panel</div>,
}));

vi.mock('@/components/device-emulation/DeviceEmulationToolbar', () => ({
  default: () => null,
}));

vi.mock('@/components/device-emulation/DeviceEmulationViewport', () => ({
  default: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

vi.mock('@/components/device-emulation/DeviceVisionFilterDefs', () => ({
  default: () => null,
}));

vi.mock('@/components/AppPreviewToolbar', () => ({
  default: (props: {
    onRefresh: () => void;
    onToggleLogs: () => void;
    areLogsVisible: boolean;
    previewUrlInput: string;
    onPreviewUrlInputChange: (event: { target: { value: string } }) => void;
    onPreviewUrlInputKeyDown: (event: { key: string; preventDefault: () => void }) => void;
    onPreviewUrlInputBlur: () => void;
    onOpenScenarioSelector: () => void;
    onSelectUrlSuggestion?: (value: string) => void;
    onToggleApp: () => void;
    onOpenInNewTab: (event: ReactMouseEvent<HTMLButtonElement>) => void;
    hasCurrentApp: boolean;
    isAppRunning: boolean;
    appStatusLabel: string;
    urlStatusTitle: string;
    rightInlineActions?: ReactNode;
  }) => (
    <div className="preview-toolbar">
      <button type="button" aria-label="refresh preview" onClick={props.onRefresh}>
        Refresh
      </button>
      <button type="button" aria-label="toggle logs" onClick={props.onToggleLogs}>
        Toggle Logs
      </button>
      <button type="button" aria-label="open scenario selector" onClick={props.onOpenScenarioSelector}>
        Open Scenario Selector
      </button>
      <input
        aria-label="Preview URL"
        value={props.previewUrlInput}
        onChange={(event) => props.onPreviewUrlInputChange({ target: { value: event.target.value } })}
        onKeyDown={(event) => {
          if (event.key === 'Enter') {
            props.onPreviewUrlInputKeyDown({
              key: 'Enter',
              preventDefault: () => event.preventDefault(),
            });
          }
        }}
        onBlur={() => props.onPreviewUrlInputBlur()}
      />
      <button
        type="button"
        aria-label="select url suggestion"
        onClick={() => props.onSelectUrlSuggestion?.('https://example.com/path')}
      >
        Select URL Suggestion
      </button>
      <button type="button" aria-label="toggle app action" onClick={props.onToggleApp}>
        Toggle App
      </button>
      <button type="button" aria-label="open in new tab" onClick={props.onOpenInNewTab}>
        Open in New Tab
      </button>
      <span data-testid="toolbar-has-current-app">{String(props.hasCurrentApp)}</span>
      <span data-testid="toolbar-is-running">{String(props.isAppRunning)}</span>
      <span data-testid="toolbar-status-label">{props.appStatusLabel}</span>
      <span data-testid="toolbar-url-status-title">{props.urlStatusTitle}</span>
      {props.rightInlineActions}
      <span>{props.areLogsVisible ? 'logs-visible' : 'logs-hidden'}</span>
    </div>
  ),
}));

const createApp = (): App => ({
  id: 'scenario-1',
  name: 'Scenario One',
  scenario_name: 'scenario-one',
  path: '/tmp/scenario-one',
  created_at: '2026-02-07T00:00:00Z',
  updated_at: '2026-02-07T00:00:00Z',
  status: 'running',
  port_mappings: { UI_PORT: 4310 },
  environment: {},
  config: {},
});

const createAppWithId = (id: string): App => ({
  ...createApp(),
  id,
  name: id,
  scenario_name: id,
  path: `/tmp/${id}`,
  port_mappings: { UI_PORT: 4310 },
});

const renderPane = (options?: {
  app?: App;
  appId?: string | null;
  paneId?: string;
  onFocus?: (paneId: string) => void;
  onRemove?: (paneId: string) => void;
}) => {
  const app = options?.app ?? createApp();
  const paneId = options?.paneId ?? usePreviewWorkspaceStore.getState().panes[0]?.id ?? 'pane-1';
  return render(
    <MemoryRouter>
      <PreviewPane
        paneId={paneId}
        appId={options?.appId ?? app.id}
        apps={[app]}
        isFocused={true}
        isArrangeMode={false}
        isBeingDragged={false}
        canRemove={true}
        onFocus={options?.onFocus ?? vi.fn()}
        onRemove={options?.onRemove ?? vi.fn()}
        onArrangeDragStart={vi.fn()}
      />
    </MemoryRouter>,
  );
};

describe('PreviewPane', () => {
  const storageKey = 'app-monitor:preview-workspace-v1';

  beforeEach(async () => {
    openOverlayMock.mockReset();
    getAppMock.mockReset();
    controlAppMock.mockReset();
    bridgeHarness.onLocation = null;
    controlAppMock.mockResolvedValue(true);
    getAppMock.mockResolvedValue(createApp());
    setAppsStateMock.mockReset();
    await usePreviewWorkspaceStore.persist.clearStorage();
    await usePreviewWorkspaceStore.persist.rehydrate();
    usePreviewWorkspaceStore.getState().reset();
  });

  it('renders toolbar and supports removing pane', async () => {
    const user = userEvent.setup();
    const onRemove = vi.fn();
    const paneId = usePreviewWorkspaceStore.getState().panes[0]?.id ?? 'pane-1';

    const { container } = renderPane({ paneId, onRemove });

    expect(container.querySelector('.preview-toolbar')).not.toBeNull();
    await user.click(screen.getByRole('button', { name: /remove pane/i }));
    expect(onRemove).toHaveBeenCalledWith(paneId);
  });

  it('persists and restores pane-local logs visibility', async () => {
    const user = userEvent.setup();
    const paneId = usePreviewWorkspaceStore.getState().panes[0]?.id;
    expect(paneId).toBeTruthy();
    if (!paneId) {
      return;
    }

    const firstRender = renderPane({ paneId });
    await user.click(screen.getByRole('button', { name: /toggle logs/i }));

    await waitFor(() => {
      expect(screen.getByTestId('app-logs-panel')).toBeInTheDocument();
      expect(usePreviewWorkspaceStore.getState().paneViewState[paneId]?.isLogsVisible).toBe(true);
    });

    firstRender.unmount();
    renderPane({ paneId });

    expect(screen.getByTestId('app-logs-panel')).toBeInTheDocument();
  });

  it('invokes lifecycle control and refreshes app state on toggle action', async () => {
    const user = userEvent.setup();
    const app = createApp();
    renderPane({ app, paneId: usePreviewWorkspaceStore.getState().panes[0]?.id ?? 'pane-1' });

    await user.click(screen.getByRole('button', { name: /toggle app action/i }));

    await waitFor(() => {
      expect(controlAppMock).toHaveBeenCalledWith(app.id, 'stop');
      expect(getAppMock).toHaveBeenCalledWith(app.id);
    });
  });

  it('hydrates partial app metadata so running status is accurate per pane', async () => {
    const partialApp: App = {
      ...createApp(),
      status: 'unknown',
      is_partial: true,
    };
    getAppMock.mockResolvedValueOnce({
      ...partialApp,
      status: 'running',
      is_partial: false,
    });

    renderPane({ app: partialApp, paneId: usePreviewWorkspaceStore.getState().panes[0]?.id ?? 'pane-1' });

    await waitFor(() => {
      expect(getAppMock).toHaveBeenCalledWith(partialApp.id);
    });
    await waitFor(() => {
      expect(screen.getByTestId('toolbar-is-running')).toHaveTextContent('true');
      expect(screen.getByTestId('toolbar-status-label')).toHaveTextContent('Running');
    });
  });

  it('hydrates using canonical app id when pane app id is scenario name', async () => {
    const appFromStore: App = {
      ...createApp(),
      id: 'canonical-app-id',
      scenario_name: 'scenario-name-id',
      name: 'Scenario Name',
      status: 'unknown',
      is_partial: true,
    };
    getAppMock.mockImplementation(async (identifier: string) => {
      if (identifier === 'canonical-app-id') {
        return {
          ...appFromStore,
          status: 'running',
          is_partial: false,
        };
      }
      return null;
    });

    renderPane({
      app: appFromStore,
      appId: 'scenario-name-id',
      paneId: usePreviewWorkspaceStore.getState().panes[0]?.id ?? 'pane-1',
    });

    await waitFor(() => {
      expect(getAppMock).toHaveBeenCalled();
    });
    await waitFor(() => {
      expect(screen.getByTestId('toolbar-is-running')).toHaveTextContent('true');
      expect(screen.getByTestId('toolbar-status-label')).toHaveTextContent('Running');
    });
  });

  it('does not lose hydration when apps list refreshes during metadata fetch', async () => {
    const partialApp: App = {
      ...createApp(),
      id: 'canonical-app-id',
      scenario_name: 'scenario-name-id',
      status: 'unknown',
      is_partial: true,
    };

    let resolveFetch: ((value: App | null) => void) | null = null;
    getAppMock.mockImplementationOnce(() => new Promise<App | null>((resolve) => {
      resolveFetch = resolve;
    }));
    getAppMock.mockImplementation(() => Promise.resolve(null));

    const result = renderPane({
      app: partialApp,
      appId: 'scenario-name-id',
      paneId: usePreviewWorkspaceStore.getState().panes[0]?.id ?? 'pane-1',
    });

    await waitFor(() => {
      expect(getAppMock).toHaveBeenCalled();
    });

    result.rerender(
      <MemoryRouter>
        <PreviewPane
          paneId={usePreviewWorkspaceStore.getState().panes[0]?.id ?? 'pane-1'}
          appId="scenario-name-id"
          apps={[{ ...partialApp, updated_at: '2026-02-08T00:00:00Z' }]}
          isFocused={true}
          isArrangeMode={false}
          isBeingDragged={false}
          canRemove={true}
          onFocus={vi.fn()}
          onRemove={vi.fn()}
          onArrangeDragStart={vi.fn()}
        />
      </MemoryRouter>,
    );

    expect(resolveFetch).not.toBeNull();
    resolveFetch?.({
      ...partialApp,
      status: 'running',
      is_partial: false,
    });

    await waitFor(() => {
      expect(screen.getByTestId('toolbar-is-running')).toHaveTextContent('true');
      expect(screen.getByTestId('toolbar-status-label')).toHaveTextContent('Running');
    });
  });

  it('throttles repeated metadata hydration requests during rapid app snapshot churn', async () => {
    const partialApp: App = {
      ...createApp(),
      status: 'unknown',
      is_partial: true,
    };
    getAppMock.mockResolvedValue(null);

    const paneId = usePreviewWorkspaceStore.getState().panes[0]?.id ?? 'pane-1';
    const { rerender } = render(
      <MemoryRouter>
        <PreviewPane
          paneId={paneId}
          appId={partialApp.id}
          apps={[partialApp]}
          isFocused={true}
          isArrangeMode={false}
          isBeingDragged={false}
          canRemove={true}
          onFocus={vi.fn()}
          onRemove={vi.fn()}
          onArrangeDragStart={vi.fn()}
        />
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(getAppMock).toHaveBeenCalledTimes(1);
      expect(getAppMock).toHaveBeenCalledWith(partialApp.id);
    });

    rerender(
      <MemoryRouter>
        <PreviewPane
          paneId={paneId}
          appId={partialApp.id}
          apps={[{ ...partialApp, updated_at: '2026-02-08T00:00:01Z' }]}
          isFocused={true}
          isArrangeMode={false}
          isBeingDragged={false}
          canRemove={true}
          onFocus={vi.fn()}
          onRemove={vi.fn()}
          onArrangeDragStart={vi.fn()}
        />
      </MemoryRouter>,
    );

    rerender(
      <MemoryRouter>
        <PreviewPane
          paneId={paneId}
          appId={partialApp.id}
          apps={[{ ...partialApp, updated_at: '2026-02-08T00:00:02Z' }]}
          isFocused={true}
          isArrangeMode={false}
          isBeingDragged={false}
          canRemove={true}
          onFocus={vi.fn()}
          onRemove={vi.fn()}
          onArrangeDragStart={vi.fn()}
        />
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(getAppMock).toHaveBeenCalledTimes(1);
    });
  });

  it('does not downgrade hydrated running status from later unknown store snapshots', async () => {
    const partialApp: App = {
      ...createApp(),
      id: 'scenario-1',
      status: 'unknown',
      is_partial: true,
    };

    getAppMock.mockResolvedValueOnce({
      ...partialApp,
      status: 'running',
      is_partial: false,
    });

    const result = renderPane({
      app: partialApp,
      appId: partialApp.id,
      paneId: usePreviewWorkspaceStore.getState().panes[0]?.id ?? 'pane-1',
    });

    await waitFor(() => {
      expect(screen.getByTestId('toolbar-is-running')).toHaveTextContent('true');
      expect(screen.getByTestId('toolbar-status-label')).toHaveTextContent('Running');
    });

    result.rerender(
      <MemoryRouter>
        <PreviewPane
          paneId={usePreviewWorkspaceStore.getState().panes[0]?.id ?? 'pane-1'}
          appId={partialApp.id}
          apps={[{ ...partialApp, updated_at: '2026-02-08T00:00:00Z' }]}
          isFocused={true}
          isArrangeMode={false}
          isBeingDragged={false}
          canRemove={true}
          onFocus={vi.fn()}
          onRemove={vi.fn()}
          onArrangeDragStart={vi.fn()}
        />
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(screen.getByTestId('toolbar-is-running')).toHaveTextContent('true');
      expect(screen.getByTestId('toolbar-status-label')).toHaveTextContent('Running');
    });
  });

  it('persists hydrated app data back into apps store', async () => {
    const partialApp: App = {
      ...createApp(),
      status: 'unknown',
      is_partial: true,
    };
    getAppMock.mockResolvedValueOnce({
      ...partialApp,
      status: 'running',
      is_partial: false,
    });

    renderPane({ app: partialApp, paneId: usePreviewWorkspaceStore.getState().panes[0]?.id ?? 'pane-1' });

    await waitFor(() => {
      expect(setAppsStateMock).toHaveBeenCalled();
    });
  });

  it('falls back to proxy URL scenario identifier when pane app id does not resolve', async () => {
    const paneId = usePreviewWorkspaceStore.getState().panes[0]?.id ?? 'pane-1';
    usePreviewWorkspaceStore.getState().setPaneViewState(paneId, {
      previewUrlInput: '/apps/canonical-app-id/proxy/',
      previewUrl: '/apps/canonical-app-id/proxy/',
    });

    const app = {
      ...createApp(),
      id: 'canonical-app-id',
      status: 'running' as const,
      is_partial: false,
    };

    getAppMock.mockImplementation(async (identifier: string) => {
      if (identifier === 'canonical-app-id') {
        return app;
      }
      return null;
    });

    render(
      <MemoryRouter>
        <PreviewPane
          paneId={paneId}
          appId="unresolved-pane-id"
          apps={[createApp()]}
          isFocused={true}
          isArrangeMode={false}
          isBeingDragged={false}
          canRemove={true}
          onFocus={vi.fn()}
          onRemove={vi.fn()}
          onArrangeDragStart={vi.fn()}
        />
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(getAppMock).toHaveBeenCalledWith('unresolved-pane-id');
      expect(getAppMock).toHaveBeenCalledWith('canonical-app-id');
    });
    await waitFor(() => {
      expect(screen.getByTestId('toolbar-is-running')).toHaveTextContent('true');
      expect(screen.getByTestId('toolbar-status-label')).toHaveTextContent('Running');
    });
  });

  it('prioritizes scenario identifier from custom proxy URL over stale pane app id', async () => {
    const paneId = usePreviewWorkspaceStore.getState().panes[0]?.id ?? 'pane-1';
    usePreviewWorkspaceStore.getState().setPaneViewState(paneId, {
      hasCustomPreviewUrl: true,
      previewUrlInput: '/apps/url-scenario/proxy/',
      previewUrl: '/apps/url-scenario/proxy/',
    });

    const fetchedApp = {
      ...createApp(),
      id: 'url-scenario',
      status: 'running' as const,
      is_partial: false,
    };

    getAppMock.mockImplementation(async (identifier: string) => {
      if (identifier === 'url-scenario') {
        return fetchedApp;
      }
      return null;
    });

    render(
      <MemoryRouter>
        <PreviewPane
          paneId={paneId}
          appId="scenario-1"
          apps={[createApp()]}
          isFocused={true}
          isArrangeMode={false}
          isBeingDragged={false}
          canRemove={true}
          onFocus={vi.fn()}
          onRemove={vi.fn()}
          onArrangeDragStart={vi.fn()}
        />
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(getAppMock).toHaveBeenCalledWith('url-scenario');
    });
    expect(getAppMock).not.toHaveBeenCalledWith('scenario-1');
  });

  it('opens tab switcher in focused-pane mode from scenario selector action', async () => {
    const user = userEvent.setup();
    const onFocus = vi.fn();
    const paneId = usePreviewWorkspaceStore.getState().panes[0]?.id;
    expect(paneId).toBeTruthy();
    if (!paneId) {
      return;
    }

    renderPane({ paneId, onFocus });
    await user.click(screen.getByRole('button', { name: /open scenario selector/i }));

    expect(onFocus).toHaveBeenCalledWith(paneId);
    expect(openOverlayMock).toHaveBeenCalledWith('tabs', {
      params: {
        segment: 'apps',
        appOpenMode: 'replace-focused',
      },
    });
  });

  it('clears loading state after iframe load', async () => {
    renderPane({ paneId: usePreviewWorkspaceStore.getState().panes[0]?.id ?? 'pane-1' });

    const iframe = await waitFor(() => {
      const found = document.querySelector('iframe');
      expect(found).not.toBeNull();
      return found as HTMLIFrameElement;
    });

    fireEvent.load(iframe);
    await waitFor(() => {
      expect(screen.queryByText('Loading preview...')).not.toBeInTheDocument();
    });
  });

  it('uses eager iframe loading for pane previews', async () => {
    renderPane({ paneId: usePreviewWorkspaceStore.getState().panes[0]?.id ?? 'pane-1' });

    const iframe = await waitFor(() => {
      const found = document.querySelector('iframe');
      expect(found).not.toBeNull();
      return found as HTMLIFrameElement;
    });

    expect(iframe.getAttribute('loading')).toBe('eager');
  });

  it('persists bridge-resolved URL so refresh keeps the current in-app route', async () => {
    const user = userEvent.setup();
    const paneId = usePreviewWorkspaceStore.getState().panes[0]?.id;
    expect(paneId).toBeTruthy();
    if (!paneId) {
      return;
    }

    renderPane({ paneId });

    await waitFor(() => {
      expect(bridgeHarness.onLocation).not.toBeNull();
    });

    act(() => {
      bridgeHarness.onLocation?.({ href: 'http://localhost:4310/settings' });
    });

    await waitFor(() => {
      expect(usePreviewWorkspaceStore.getState().paneViewState[paneId]?.previewUrl).toBe(
        'http://localhost:3000/apps/scenario-1/proxy/',
      );
      expect(usePreviewWorkspaceStore.getState().paneViewState[paneId]?.previewUrlInput).toBe('http://localhost:4310/settings');
      const iframe = document.querySelector('iframe');
      expect(iframe?.getAttribute('src')).toBe('http://localhost:3000/apps/scenario-1/proxy/');
    });

    await waitFor(() => {
      const persisted = window.localStorage.getItem(storageKey);
      expect(persisted).toBeTruthy();
      if (!persisted) {
        return;
      }
      const parsed = JSON.parse(persisted) as {
        state?: { paneViewState?: Record<string, { previewUrl?: string; previewUrlInput?: string }> };
      };
      expect(parsed.state?.paneViewState?.[paneId]?.previewUrl).toBe('http://localhost:3000/apps/scenario-1/proxy/');
      expect(parsed.state?.paneViewState?.[paneId]?.previewUrlInput).toBe('http://localhost:4310/settings');
    });

    await user.click(screen.getByRole('button', { name: /refresh preview/i }));

    await waitFor(() => {
      const iframe = document.querySelector('iframe');
      expect(iframe?.getAttribute('src')).toBe('http://localhost:4310/settings');
    });

    await waitFor(() => {
      expect(usePreviewWorkspaceStore.getState().paneViewState[paneId]?.previewUrl).toBe('http://localhost:4310/settings');
      const persisted = window.localStorage.getItem(storageKey);
      expect(persisted).toBeTruthy();
      if (!persisted) {
        return;
      }
      const parsed = JSON.parse(persisted) as {
        state?: { paneViewState?: Record<string, { previewUrl?: string; previewUrlInput?: string }> };
      };
      expect(parsed.state?.paneViewState?.[paneId]?.previewUrl).toBe('http://localhost:4310/settings');
      expect(parsed.state?.paneViewState?.[paneId]?.previewUrlInput).toBe('http://localhost:4310/settings');
    });
  });

  it('blocks shell URL entry from replacing iframe src with app-monitor workspace route', async () => {
    const user = userEvent.setup();
    const paneId = usePreviewWorkspaceStore.getState().panes[0]?.id ?? 'pane-1';

    renderPane({ paneId });

    const iframe = await waitFor(() => {
      const found = document.querySelector('iframe');
      expect(found).not.toBeNull();
      return found as HTMLIFrameElement;
    });
    expect(iframe.getAttribute('src')).toBe('http://localhost:3000/apps/scenario-1/proxy/');

    const urlInput = screen.getByLabelText('Preview URL');
    await user.clear(urlInput);
    await user.type(urlInput, '/apps/workspace{enter}');

    await waitFor(() => {
      const currentIframe = document.querySelector('iframe');
      expect(currentIframe?.getAttribute('src')).toBe('http://localhost:3000/apps/scenario-1/proxy/');
      expect(screen.getByTestId('toolbar-url-status-title')).toHaveTextContent(
        'Preview URL points to App Monitor shell. Use a scenario proxy URL like /apps/<scenario>/proxy/.',
      );
    });
  });

  it('resets stale custom URL state when pane app assignment changes', async () => {
    const user = userEvent.setup();
    const paneId = usePreviewWorkspaceStore.getState().panes[0]?.id ?? 'pane-1';
    const appOne = createAppWithId('scenario-1');
    const appTwo = createAppWithId('scenario-2');

    const { rerender } = render(
      <MemoryRouter>
        <PreviewPane
          paneId={paneId}
          appId={appOne.id}
          apps={[appOne, appTwo]}
          isFocused={true}
          isArrangeMode={false}
          isBeingDragged={false}
          canRemove={true}
          onFocus={vi.fn()}
          onRemove={vi.fn()}
          onArrangeDragStart={vi.fn()}
        />
      </MemoryRouter>,
    );

    const initialIframe = await waitFor(() => {
      const found = document.querySelector('iframe');
      expect(found).not.toBeNull();
      return found as HTMLIFrameElement;
    });
    expect(initialIframe.getAttribute('src')).toBe('http://localhost:3000/apps/scenario-1/proxy/');

    const urlInput = screen.getByLabelText('Preview URL');
    await user.clear(urlInput);
    await user.type(urlInput, '/apps/missing-scenario/proxy/{enter}');

    await waitFor(() => {
      const updatedIframe = document.querySelector('iframe');
      expect(updatedIframe?.getAttribute('src')).toBe('http://localhost:3000/apps/missing-scenario/proxy/');
      expect(usePreviewWorkspaceStore.getState().paneViewState[paneId]?.hasCustomPreviewUrl).toBe(true);
    });

    rerender(
      <MemoryRouter>
        <PreviewPane
          paneId={paneId}
          appId={appTwo.id}
          apps={[appOne, appTwo]}
          isFocused={true}
          isArrangeMode={false}
          isBeingDragged={false}
          canRemove={true}
          onFocus={vi.fn()}
          onRemove={vi.fn()}
          onArrangeDragStart={vi.fn()}
        />
      </MemoryRouter>,
    );

    await waitFor(() => {
      const updatedIframe = document.querySelector('iframe');
      expect(updatedIframe?.getAttribute('src')).toBe('http://localhost:3000/apps/scenario-2/proxy/');
      const paneState = usePreviewWorkspaceStore.getState().paneViewState[paneId];
      expect(paneState?.hasCustomPreviewUrl).toBe(false);
      expect(paneState?.previewUrl).toBe('http://localhost:3000/apps/scenario-2/proxy/');
      expect(paneState?.previewUrlInput).toBe('http://localhost:3000/apps/scenario-2/proxy/');
    });
  });

  it('preserves manually selected URL in empty pane across apps list updates', async () => {
    const user = userEvent.setup();
    const paneId = usePreviewWorkspaceStore.getState().panes[0]?.id ?? 'pane-1';
    const app = createApp();
    const { rerender } = render(
      <MemoryRouter>
        <PreviewPane
          paneId={paneId}
          appId={null}
          apps={[app]}
          isFocused={true}
          isArrangeMode={false}
          isBeingDragged={false}
          canRemove={true}
          onFocus={vi.fn()}
          onRemove={vi.fn()}
          onArrangeDragStart={vi.fn()}
        />
      </MemoryRouter>,
    );

    await user.click(screen.getByRole('button', { name: /select url suggestion/i }));

    await waitFor(() => {
      const iframe = document.querySelector('iframe');
      expect(iframe).not.toBeNull();
      expect(iframe?.getAttribute('src')).toBe('https://example.com/path');
    });

    const updatedApps: App[] = [
      app,
      {
        ...app,
        id: 'scenario-2',
        name: 'Scenario Two',
        scenario_name: 'scenario-two',
        path: '/tmp/scenario-two',
      },
    ];

    rerender(
      <MemoryRouter>
        <PreviewPane
          paneId={paneId}
          appId={null}
          apps={updatedApps}
          isFocused={true}
          isArrangeMode={false}
          isBeingDragged={false}
          canRemove={true}
          onFocus={vi.fn()}
          onRemove={vi.fn()}
          onArrangeDragStart={vi.fn()}
        />
      </MemoryRouter>,
    );

    await waitFor(() => {
      const iframe = document.querySelector('iframe');
      expect(iframe).not.toBeNull();
      expect(iframe?.getAttribute('src')).toBe('https://example.com/path');
    });
  });

  it('keeps hasCurrentApp when metadata refresh fails for an already-resolved pane app', async () => {
    const app = createApp();
    const paneId = usePreviewWorkspaceStore.getState().panes[0]?.id ?? 'pane-1';

    const { rerender } = render(
      <MemoryRouter>
        <PreviewPane
          paneId={paneId}
          appId={app.id}
          apps={[app]}
          isFocused={true}
          isArrangeMode={false}
          isBeingDragged={false}
          canRemove={true}
          onFocus={vi.fn()}
          onRemove={vi.fn()}
          onArrangeDragStart={vi.fn()}
        />
      </MemoryRouter>,
    );

    expect(screen.getByTestId('toolbar-has-current-app')).toHaveTextContent('true');

    getAppMock.mockRejectedValueOnce(new Error('temporary metadata failure'));

    rerender(
      <MemoryRouter>
        <PreviewPane
          paneId={paneId}
          appId={app.id}
          apps={[]}
          isFocused={true}
          isArrangeMode={false}
          isBeingDragged={false}
          canRemove={true}
          onFocus={vi.fn()}
          onRemove={vi.fn()}
          onArrangeDragStart={vi.fn()}
        />
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(getAppMock).toHaveBeenCalledWith(app.id);
      expect(screen.getByTestId('toolbar-has-current-app')).toHaveTextContent('true');
    });
  });

  it('treats pane app identifier as scenario context when metadata is unavailable', async () => {
    const paneId = usePreviewWorkspaceStore.getState().panes[0]?.id ?? 'pane-1';
    getAppMock.mockRejectedValueOnce(new Error('temporary metadata failure'));

    render(
      <MemoryRouter>
        <PreviewPane
          paneId={paneId}
          appId="git-control-tower"
          apps={[]}
          isFocused={true}
          isArrangeMode={false}
          isBeingDragged={false}
          canRemove={true}
          onFocus={vi.fn()}
          onRemove={vi.fn()}
          onArrangeDragStart={vi.fn()}
        />
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(getAppMock).toHaveBeenCalledWith('git-control-tower');
      expect(screen.getByTestId('toolbar-has-current-app')).toHaveTextContent('true');
    });
  });

  it('treats scenario proxy URL input as scenario context even without pane app id', async () => {
    const paneId = usePreviewWorkspaceStore.getState().panes[0]?.id ?? 'pane-1';
    usePreviewWorkspaceStore.getState().setPaneViewState(paneId, {
      previewUrl: 'https://app-monitor.itsagitime.com/apps/git-control-tower/proxy/',
      previewUrlInput: 'https://app-monitor.itsagitime.com/apps/git-control-tower/proxy/',
      hasCustomPreviewUrl: true,
      history: ['https://app-monitor.itsagitime.com/apps/git-control-tower/proxy/'],
      historyIndex: 0,
      initialPreviewUrl: 'https://app-monitor.itsagitime.com/apps/git-control-tower/proxy/',
    });

    render(
      <MemoryRouter>
        <PreviewPane
          paneId={paneId}
          appId={null}
          apps={[]}
          isFocused={true}
          isArrangeMode={false}
          isBeingDragged={false}
          canRemove={true}
          onFocus={vi.fn()}
          onRemove={vi.fn()}
          onArrangeDragStart={vi.fn()}
        />
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(getAppMock).toHaveBeenCalledWith('git-control-tower');
      expect(screen.getByTestId('toolbar-has-current-app')).toHaveTextContent('true');
      expect(screen.getByTestId('toolbar-is-running')).toHaveTextContent('true');
      expect(screen.getByTestId('toolbar-status-label')).toHaveTextContent('Running');
    });
  });

});
