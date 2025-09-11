import { useState, useEffect, useCallback } from 'react';
import type {
  MetricsResponse,
  DetailedMetrics,
  ProcessMonitorData,
  InfrastructureMonitorData,
  Investigation,
  APIError
} from '../types';

interface UseSystemMonitorReturn {
  metrics: MetricsResponse | null;
  detailedMetrics: DetailedMetrics | null;
  processMonitorData: ProcessMonitorData | null;
  infrastructureData: InfrastructureMonitorData | null;
  investigations: Investigation[];
  isLoading: boolean;
  error: APIError | null;
  refresh: () => void;
  refreshMetrics: () => void;
}

const API_BASE = '';  // Using Vite proxy, so same origin

export const useSystemMonitor = (): UseSystemMonitorReturn => {
  const [metrics, setMetrics] = useState<MetricsResponse | null>(null);
  const [detailedMetrics, setDetailedMetrics] = useState<DetailedMetrics | null>(null);
  const [processMonitorData, setProcessMonitorData] = useState<ProcessMonitorData | null>(null);
  const [infrastructureData, setInfrastructureData] = useState<InfrastructureMonitorData | null>(null);
  const [investigations, setInvestigations] = useState<Investigation[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<APIError | null>(null);

  const handleApiCall = async <T>(url: string): Promise<T | null> => {
    try {
      const response = await fetch(`${API_BASE}${url}`);
      
      if (!response.ok) {
        const errorText = await response.text();
        let errorData: APIError;
        try {
          errorData = JSON.parse(errorText);
        } catch {
          errorData = {
            error: `HTTP ${response.status}: ${response.statusText}`,
            details: errorText,
            timestamp: new Date().toISOString()
          };
        }
        throw errorData;
      }

      const data = await response.json();
      return data;
    } catch (err) {
      console.error(`API call failed for ${url}:`, err);
      
      if (err && typeof err === 'object' && 'error' in err) {
        setError(err as APIError);
      } else {
        setError({
          error: 'Network or unknown error',
          details: err instanceof Error ? err.message : String(err),
          timestamp: new Date().toISOString()
        });
      }
      return null;
    }
  };

  const checkHealth = useCallback(async (): Promise<boolean> => {
    try {
      const response = await fetch(`${API_BASE}/health`);
      return response.ok;
    } catch {
      return false;
    }
  }, []);

  const fetchMetrics = useCallback(async () => {
    const data = await handleApiCall<MetricsResponse>('/api/metrics/current');
    if (data) {
      setMetrics(data);
      setError(null);
    }
  }, []);

  const fetchDetailedMetrics = useCallback(async () => {
    const data = await handleApiCall<DetailedMetrics>('/api/metrics/detailed');
    if (data) {
      setDetailedMetrics(data);
    }
  }, []);

  const fetchProcessMonitorData = useCallback(async () => {
    const data = await handleApiCall<ProcessMonitorData>('/api/metrics/processes');
    if (data) {
      setProcessMonitorData(data);
    }
  }, []);

  const fetchInfrastructureData = useCallback(async () => {
    const data = await handleApiCall<InfrastructureMonitorData>('/api/metrics/infrastructure');
    if (data) {
      setInfrastructureData(data);
    }
  }, []);

  const fetchInvestigations = useCallback(async () => {
    const data = await handleApiCall<Investigation>('/api/investigations/latest');
    if (data) {
      setInvestigations([data]); // For now, just latest investigation
    }
  }, []);

  const refreshMetrics = useCallback(async () => {
    setIsLoading(true);
    await Promise.all([
      fetchMetrics(),
      fetchDetailedMetrics()
    ]);
    setIsLoading(false);
  }, [fetchMetrics, fetchDetailedMetrics]);

  const refresh = useCallback(async () => {
    setIsLoading(true);
    
    // Check health first
    const isHealthy = await checkHealth();
    if (!isHealthy) {
      setError({
        error: 'API server is not responding',
        details: 'Health check failed - ensure the Go backend is running',
        timestamp: new Date().toISOString()
      });
      setIsLoading(false);
      return;
    }

    // Fetch all data
    await Promise.all([
      fetchMetrics(),
      fetchDetailedMetrics(),
      fetchProcessMonitorData(),
      fetchInfrastructureData(),
      fetchInvestigations()
    ]);
    
    setIsLoading(false);
  }, [
    checkHealth,
    fetchMetrics,
    fetchDetailedMetrics,
    fetchProcessMonitorData,
    fetchInfrastructureData,
    fetchInvestigations
  ]);

  // Initial load
  useEffect(() => {
    refresh();
  }, [refresh]);

  // Set up polling for metrics (every 30 seconds)
  useEffect(() => {
    const metricsInterval = setInterval(refreshMetrics, 30000);
    return () => clearInterval(metricsInterval);
  }, [refreshMetrics]);

  // Set up polling for detailed data (every 60 seconds)
  useEffect(() => {
    const detailedInterval = setInterval(() => {
      Promise.all([
        fetchProcessMonitorData(),
        fetchInfrastructureData(),
        fetchInvestigations()
      ]);
    }, 60000);
    
    return () => clearInterval(detailedInterval);
  }, [fetchProcessMonitorData, fetchInfrastructureData, fetchInvestigations]);

  return {
    metrics,
    detailedMetrics,
    processMonitorData,
    infrastructureData,
    investigations,
    isLoading,
    error,
    refresh,
    refreshMetrics
  };
};