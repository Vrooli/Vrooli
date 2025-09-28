import { useEffect, useState } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import Layout from '@/components/Layout';
import AppsView from '@/components/views/AppsView';
import AppPreviewView from '@/components/views/AppPreviewView';
import LogsView from '@/components/views/LogsView';
import ResourcesView from '@/components/views/ResourcesView';
import ErrorBoundary from '@/components/ErrorBoundary';
import { useAppWebSocket } from '@/hooks/useWebSocket';
import './App.css';
import { useAppsStore } from '@/state/appsStore';
import { useResourcesStore } from '@/state/resourcesStore';

function App() {
  const loadApps = useAppsStore(state => state.loadApps);
  const updateAppInStore = useAppsStore(state => state.updateApp);
  const loadResources = useResourcesStore(state => state.loadResources);
  const [isConnected, setIsConnected] = useState(false);

  // WebSocket connection for real-time updates
  const { connectionState } = useAppWebSocket({
    onAppUpdate: (update) => {
      console.log('App update received:', update);
      updateAppInStore(update);
    },
    onMetricUpdate: () => {
      // Metrics are now handled by system-monitor iframe
    },
    onLogEntry: (log) => {
      console.log('Log entry received:', log);
      // Logs are handled directly in LogsView component
    },
    onConnection: (connected) => {
      setIsConnected(connected);
    },
    onError: (error) => {
      console.error('WebSocket error:', error);
    },
  });

  // Log connection state changes
  useEffect(() => {
    console.log(`WebSocket connection state: ${connectionState}`);
  }, [connectionState]);

  // Fetch initial data on component mount
  useEffect(() => {
    void loadApps();
    void loadResources();
  }, [loadApps, loadResources]);

  return (
    <ErrorBoundary>
      <Router>
        <div className="app">
          <div className="matrix-rain"></div>
          <Layout isConnected={isConnected}>
            <Routes>
              <Route path="/" element={<Navigate to="/apps" replace />} />
              <Route path="/apps" element={<AppsView />} />
              <Route path="/apps/:appId/preview" element={<AppPreviewView />} />
              <Route path="/logs" element={<LogsView />} />
              <Route path="/logs/:appId" element={<LogsView />} />
              <Route path="/resources" element={<ResourcesView />} />
              <Route path="*" element={<Navigate to="/apps" replace />} />
            </Routes>
          </Layout>
        </div>
      </Router>
    </ErrorBoundary>
  );
}

export default App;
