import { useEffect, useState } from 'react';
import {
  BrowserRouter as Router,
  Routes,
  Route,
  Navigate,
  useLocation,
  useNavigate,
  useParams,
} from 'react-router-dom';
import Shell from '@/components/Shell';
import AppPreviewView from '@/components/views/AppPreviewView';
import ResourceDetailView from '@/components/views/ResourceDetailView';
import HomeView from '@/components/views/HomeView';
import ErrorBoundary from '@/components/ErrorBoundary';
import { useAppWebSocket } from '@/hooks/useWebSocket';
import './App.css';
import { useAppsStore } from '@/state/appsStore';
import { useResourcesStore } from '@/state/resourcesStore';

function LegacyLogsRedirect() {
  const navigate = useNavigate();
  const location = useLocation();
  const { appId } = useParams<{ appId?: string }>();

  useEffect(() => {
    const nextSearch = new URLSearchParams(location.search);
    if (appId) {
      nextSearch.set('overlay', 'logs');
      const searchString = nextSearch.toString();
      navigate({
        pathname: `/apps/${encodeURIComponent(appId)}/preview`,
        search: searchString ? `?${searchString}` : '?overlay=logs',
      }, { replace: true });
      return;
    }

    nextSearch.set('overlay', 'tabs');
    nextSearch.set('segment', 'apps');
    navigate({
      pathname: '/',
      search: `?${nextSearch.toString()}`,
    }, { replace: true });
  }, [appId, location.search, navigate]);

  return null;
}

function TabOverlayRedirect({ segment }: { segment?: 'apps' | 'resources' }) {
  const navigate = useNavigate();
  const location = useLocation();

  useEffect(() => {
    const nextParams = new URLSearchParams(location.search);
    nextParams.set('overlay', 'tabs');
    if (segment) {
      nextParams.set('segment', segment);
    } else {
      nextParams.delete('segment');
    }

    const search = nextParams.toString();
    navigate({
      pathname: '/',
      search: search ? `?${search}` : '?overlay=tabs',
    }, { replace: true });
  }, [navigate, location.search, segment]);

  return null;
}

function App() {
  const loadApps = useAppsStore(state => state.loadApps);
  const updateAppInStore = useAppsStore(state => state.updateApp);
  const loadResources = useResourcesStore(state => state.loadResources);
  const [isConnected, setIsConnected] = useState(false);

  const { connectionState } = useAppWebSocket({
    onAppUpdate: (update) => {
      console.log('App update received:', update);
      updateAppInStore(update);
    },
    onMetricUpdate: () => {
    },
    onLogEntry: (log) => {
      console.log('Log entry received:', log);
    },
    onConnection: (connected) => {
      setIsConnected(connected);
    },
    onError: (error) => {
      console.error('WebSocket error:', error);
    },
  });

  useEffect(() => {
    console.log(`WebSocket connection state: ${connectionState}`);
  }, [connectionState]);

  useEffect(() => {
    void loadApps();
    void loadResources();
  }, [loadApps, loadResources]);

  return (
    <ErrorBoundary>
      <Router>
        <div className="app">
          <div className="matrix-rain"></div>
          <Routes>
            <Route element={<Shell isConnected={isConnected} />}>
              <Route index element={<HomeView />} />
              <Route path="apps" element={<TabOverlayRedirect segment="apps" />} />
              <Route path="resources" element={<TabOverlayRedirect segment="resources" />} />
              <Route path="tabs" element={<TabOverlayRedirect />} />
              <Route path="apps/:appId/preview" element={<AppPreviewView />} />
              <Route path="resources/:resourceId" element={<ResourceDetailView />} />
              <Route path="logs" element={<LegacyLogsRedirect />} />
              <Route path="logs/:appId" element={<LegacyLogsRedirect />} />
              <Route path="*" element={<Navigate to="/" replace />} />
            </Route>
          </Routes>
        </div>
      </Router>
    </ErrorBoundary>
  );
}

export default App;
