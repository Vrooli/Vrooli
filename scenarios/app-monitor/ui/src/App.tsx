import ErrorBoundary from '@/components/ErrorBoundary';
import Shell from '@/components/Shell';
import AppPreviewView from '@/components/views/AppPreviewView';
import HomeView from '@/components/views/HomeView';
import ResourceDetailView from '@/components/views/ResourceDetailView';
import { isIosSafariUserAgent, primePreviewGuardForNavigation } from '@/components/views/useIosAutobackGuard';
import { useAppWebSocket } from '@/hooks/useWebSocket';
import { logger } from '@/services/logger';
import { useAppsStore } from '@/state/appsStore';
import { useResourcesStore } from '@/state/resourcesStore';
import { useEffect, useState } from 'react';
import {
  Navigate,
  Route,
  BrowserRouter as Router,
  Routes,
  useLocation,
  useParams,
} from 'react-router-dom';
import './App.css';

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
  params.set('overlay', 'logs');
  const search = params.toString();
  const targetSearch = search ? `?${search}` : '?overlay=logs';
  const targetPath = appId ? `/apps/${encodeURIComponent(appId)}/preview${targetSearch}` : '';

  useEffect(() => {
    if (!appId || !isIosSafariUserAgent()) {
      return;
    }
    primePreviewGuardForNavigation({
      appId,
      recoverPath: targetPath,
    });
  }, [appId, targetPath]);

  if (!appId) {
    return <TabOverlayRedirect segment="apps" />;
  }

  const navigationState = {
    fromAppsList: true,
    originAppId: appId,
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
  const loadResources = useResourcesStore(state => state.loadResources);
  const [isConnected, setIsConnected] = useState(false);

  const { connectionState } = useAppWebSocket({
    onAppUpdate: (update) => {
      logger.debug('Received app update payload', update);
      updateAppInStore(update);
    },
    onMetricUpdate: () => {
    },
    onLogEntry: (log) => {
      logger.debug('Received live log entry', log);
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
    logger.debug('WebSocket connection state changed', { state: connectionState });
  }, [connectionState]);

  useEffect(() => {
    void loadApps();
    void loadResources();
  }, [loadApps, loadResources]);

  return (
    <ErrorBoundary>
      <Router>
        <div className="app">
          <Routes>
            <Route element={<Shell isConnected={isConnected} />}>
              <Route index element={<HomeView />} />
              <Route path="apps" element={<TabOverlayRedirect segment="apps" />} />
              <Route path="resources" element={<TabOverlayRedirect segment="resources" />} />
              <Route path="tabs" element={<TabOverlayRedirect />} />
              <Route path="apps/:appId/preview" element={<AppPreviewView />} />
              <Route path="resources/:resourceId" element={<ResourceDetailView />} />
              <Route path="logs" element={<TabOverlayRedirect segment="apps" />} />
              <Route path="logs/:appId" element={<LogsAppRedirect />} />
              <Route path="*" element={<Navigate to="/" replace />} />
            </Route>
          </Routes>
        </div>
      </Router>
    </ErrorBoundary>
  );
}

export default App;
