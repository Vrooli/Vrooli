import { useState, useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import Layout from '@/components/Layout';
import AppsView from '@/components/views/AppsView';
import MetricsView from '@/components/views/MetricsView';
import LogsView from '@/components/views/LogsView';
import ResourcesView from '@/components/views/ResourcesView';
import TerminalView from '@/components/views/TerminalView';
import { useAppWebSocket } from '@/hooks/useWebSocket';
import { metricsService, appService } from '@/services/api';
import type { App, SystemMetrics } from '@/types';
import './App.css';

function App() {
  const [apps, setApps] = useState<App[]>([]);
  const [metrics, setMetrics] = useState<SystemMetrics | null>(null);
  const [isConnected, setIsConnected] = useState(false);

  // WebSocket connection for real-time updates
  const { connectionState } = useAppWebSocket({
    onAppUpdate: (update) => {
      console.log('App update received:', update);
      setApps(prev => {
        const index = prev.findIndex(app => app.id === update.id);
        if (index !== -1) {
          const updated = [...prev];
          updated[index] = { ...updated[index], ...update };
          return updated;
        }
        return prev;
      });
    },
    onMetricUpdate: (newMetrics) => {
      console.log('Metrics update received:', newMetrics);
      setMetrics(newMetrics);
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
    const fetchInitialData = async () => {
      console.log('[App] Fetching initial data...');
      
      try {
        // Fetch initial apps
        const initialApps = await appService.getApps();
        console.log('[App] Initial apps loaded:', initialApps.length);
        setApps(initialApps);
        
        // Fetch initial metrics
        const initialMetrics = await metricsService.getSystemMetrics();
        console.log('[App] Initial metrics loaded:', initialMetrics);
        if (initialMetrics) {
          setMetrics(initialMetrics);
        }
      } catch (error) {
        console.error('[App] Failed to fetch initial data:', error);
      }
    };

    fetchInitialData();
  }, []);

  return (
    <Router>
      <div className="app">
        <div className="matrix-rain"></div>
        <Layout isConnected={isConnected}>
          <Routes>
            <Route path="/" element={<Navigate to="/apps" replace />} />
            <Route path="/apps" element={<AppsView apps={apps} setApps={setApps} />} />
            <Route path="/metrics" element={<MetricsView metrics={metrics} />} />
            <Route path="/logs" element={<LogsView />} />
            <Route path="/logs/:appId" element={<LogsView />} />
            <Route path="/resources" element={<ResourcesView />} />
            <Route path="/terminal" element={<TerminalView />} />
            <Route path="*" element={<Navigate to="/apps" replace />} />
          </Routes>
        </Layout>
      </div>
    </Router>
  );
}

export default App;