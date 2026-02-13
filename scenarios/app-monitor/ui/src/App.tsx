import ErrorBoundary from '@/components/ErrorBoundary';
import Shell from '@/components/Shell';
import {
  WORKSPACE_INTENT_APP_ID_KEY,
  WORKSPACE_INTENT_MODE_KEY,
} from '@/features/preview-workspace/utils/navigationIntent';
import { useAppWebSocket } from '@/hooks/useWebSocket';
import { KeyboardScopeProvider } from '@/hooks/useKeyboardScopes';
import { logger } from '@/services/logger';
import { useAppsStore } from '@/state/appsStore';
import { useResourcesStore } from '@/state/resourcesStore';
import { lazy, Suspense, useCallback, useEffect, useRef, useState } from 'react';
import type { App } from '@/types';
import {
  Navigate,
  Route,
  BrowserRouter as Router,
  Routes,
  useLocation,
  useParams,
} from 'react-router-dom';
import './App.css';

// AI_CHECK: APP_MONITOR_RENDER_PERF=1 | LAST: 2026-02-13
const ResourceDetailView = lazy(() => import('@/components/views/ResourceDetailView'));
const PreviewWorkspaceView = lazy(() => import('@/features/preview-workspace/components/PreviewWorkspaceView'));

function RouteLoadingFallback() {
  return (
    <main className="home-view" aria-busy="true" aria-live="polite">
      <section className="hero-section">
        <h1 className="visually-hidden">Loading view</h1>
      </section>
    </main>
  );
}

function TabOverlayRedirect({ segment }: { segment?: 'apps' | 'resources' }) {
  const location = useLocation();
  const nextParams = new URLSearchParams(location.search);
  nextParams.set('overlay', 'tabs');
  if (segment) {
    nextParams.set('segment', segment);
  } else {
    nextParams.delete('segment');
  }

  const search = nextParams.toString();
  const target = search ? `/?${search}` : '/?overlay=tabs';

  return <Navigate to={target} replace />;
}

function LogsAppRedirect() {
  const { appId } = useParams<{ appId?: string }>();
  const location = useLocation();

  const params = new URLSearchParams(location.search);
  const normalizedAppId = appId?.trim();
  if (normalizedAppId) {
    params.set(WORKSPACE_INTENT_APP_ID_KEY, normalizedAppId);
    params.set(WORKSPACE_INTENT_MODE_KEY, 'replace-focused');
    params.set('paneLogs', '1');
  }
  const search = params.toString();
  const targetPath = normalizedAppId ? `/apps/workspace?${search}` : '';

  if (!normalizedAppId) {
    return <TabOverlayRedirect segment="apps" />;
  }

  const navigationState = {
    fromAppsList: true,
    originAppId: normalizedAppId,
    navTimestamp: Date.now(),
    suppressedAutoBack: false,
  } as const;

  return (
    <Navigate
      to={targetPath}
      state={navigationState}
      replace
    />
  );
}

function AppPreviewRedirect() {
  const { appId } = useParams<{ appId?: string }>();
  const location = useLocation();
  const normalizedAppId = appId?.trim();

  if (!normalizedAppId) {
    return <Navigate to="/apps/workspace" replace />;
  }

  const nextParams = new URLSearchParams(location.search);
  nextParams.set(WORKSPACE_INTENT_APP_ID_KEY, normalizedAppId);
  nextParams.set(WORKSPACE_INTENT_MODE_KEY, 'replace-focused');
  const search = nextParams.toString();
  const targetPath = search ? `/apps/workspace?${search}` : '/apps/workspace';

  const navigationState = {
    fromAppsList: true,
    originAppId: normalizedAppId,
    navTimestamp: Date.now(),
    suppressedAutoBack: false,
  } as const;

  return (
    <Navigate
      to={targetPath}
      state={navigationState}
      replace
    />
  );
}

function App() {
  const loadApps = useAppsStore(state => state.loadApps);
  const updateAppInStore = useAppsStore(state => state.updateApp);
  const updateAppsBatchInStore = useAppsStore(state => state.updateAppsBatch);
  const loadResources = useResourcesStore(state => state.loadResources);
  const [isConnected, setIsConnected] = useState(false);
  const pendingAppUpdatesRef = useRef<Map<string, Partial<App> & { id?: string }>>(new Map());
  const flushFrameRef = useRef<number | null>(null);

  const flushPendingAppUpdates = useCallback(() => {
    flushFrameRef.current = null;
    const pending = pendingAppUpdatesRef.current;
    if (pending.size === 0) {
      return;
    }
    const updates = Array.from(pending.values());
    pending.clear();
    if (updates.length === 1) {
      const [single] = updates;
      if (single) {
        updateAppInStore(single);
      }
      return;
    }
    updateAppsBatchInStore(updates);
  }, [updateAppInStore, updateAppsBatchInStore]);

  const queueAppUpdate = useCallback((update: Partial<App> & { id?: string }) => {
    const key = update.id ?? update.scenario_name ?? update.name;
    if (!key || key.trim().length === 0) {
      updateAppInStore(update);
      return;
    }
    const normalizedKey = key.trim().toLowerCase();
    const existing = pendingAppUpdatesRef.current.get(normalizedKey);
    pendingAppUpdatesRef.current.set(normalizedKey, existing ? { ...existing, ...update } : update);

    if (flushFrameRef.current === null) {
      flushFrameRef.current = window.requestAnimationFrame(flushPendingAppUpdates);
    }
  }, [flushPendingAppUpdates, updateAppInStore]);

  const { connectionState } = useAppWebSocket({
    onAppUpdate: (update) => {
      queueAppUpdate(update);
    },
    onMetricUpdate: () => {
    },
    onLogEntry: () => {
    },
    onConnection: (connected) => {
      setIsConnected(connected);
      logger.info('App monitor websocket connection updated', { connected });
    },
    onError: (error) => {
      logger.error('App monitor websocket error', error);
    },
  });

  useEffect(() => {
    logger.info('WebSocket connection state changed', { state: connectionState });
  }, [connectionState]);

  useEffect(() => {
    void loadApps();
    void loadResources();
  }, [loadApps, loadResources]);

  useEffect(() => {
    return () => {
      if (flushFrameRef.current !== null) {
        window.cancelAnimationFrame(flushFrameRef.current);
        flushFrameRef.current = null;
      }
      pendingAppUpdatesRef.current.clear();
    };
  }, []);

  return (
    <ErrorBoundary>
      <KeyboardScopeProvider>
        <Router>
          <div className="app">
            <Routes>
              <Route element={<Shell isConnected={isConnected} />}>
                <Route index element={<Navigate to="/apps/workspace" replace />} />
                <Route path="apps" element={<TabOverlayRedirect segment="apps" />} />
                <Route path="resources" element={<TabOverlayRedirect segment="resources" />} />
                <Route path="tabs" element={<TabOverlayRedirect />} />
                <Route
                  path="apps/workspace"
                  element={(
                    <Suspense fallback={<RouteLoadingFallback />}>
                      <PreviewWorkspaceView />
                    </Suspense>
                  )}
                />
                <Route path="apps/:appId/preview" element={<AppPreviewRedirect />} />
                <Route
                  path="resources/:resourceId"
                  element={(
                    <Suspense fallback={<RouteLoadingFallback />}>
                      <ResourceDetailView />
                    </Suspense>
                  )}
                />
                <Route path="logs" element={<TabOverlayRedirect segment="apps" />} />
                <Route path="logs/:appId" element={<LogsAppRedirect />} />
                <Route path="*" element={<Navigate to="/" replace />} />
              </Route>
            </Routes>
          </div>
        </Router>
      </KeyboardScopeProvider>
    </ErrorBoundary>
  );
}

export default App;
