import { useEffect, useState } from "react";
import { apiUrlBase } from "../utils/consts.js";
import type { SystemHealth } from "../components/stats/SystemHealthCard.js";
import type { SystemMetrics } from "../components/stats/SystemMetricsCards.js";

interface UseSystemHealthOptions {
    autoRefresh?: boolean;
    refreshInterval?: number;
}

export function useSystemHealth(options: UseSystemHealthOptions = {}) {
    const { autoRefresh = true, refreshInterval = 30000 } = options;
    
    const [healthData, setHealthData] = useState<SystemHealth | null>(null);
    const [metricsData, setMetricsData] = useState<SystemMetrics | null>(null);
    const [healthLoading, setHealthLoading] = useState(false);
    const [metricsLoading, setMetricsLoading] = useState(false);
    const [healthError, setHealthError] = useState<string | null>(null);
    const [metricsError, setMetricsError] = useState<string | null>(null);

    const fetchHealthData = async () => {
        setHealthLoading(true);
        setHealthError(null);
        try {
            // Remove the /api prefix since healthcheck is mounted directly on the app
            const url = apiUrlBase.replace("/api", "") + "/healthcheck";
            const response = await fetch(url, {
                credentials: "include",
            });
            if (!response.ok) {
                throw new Error(`Health check failed: ${response.statusText}`);
            }
            const data = await response.json();
            setHealthData(data);
        } catch (error) {
            setHealthError(error instanceof Error ? error.message : "Failed to fetch health data");
        } finally {
            setHealthLoading(false);
        }
    };

    const fetchMetricsData = async () => {
        setMetricsLoading(true);
        setMetricsError(null);
        try {
            // Remove the /api prefix since metrics is mounted directly on the app
            const url = apiUrlBase.replace("/api", "") + "/metrics";
            const response = await fetch(url, {
                credentials: "include",
            });
            if (!response.ok) {
                throw new Error(`Metrics fetch failed: ${response.statusText}`);
            }
            const data = await response.json();
            setMetricsData(data);
        } catch (error) {
            setMetricsError(error instanceof Error ? error.message : "Failed to fetch metrics data");
        } finally {
            setMetricsLoading(false);
        }
    };

    const refresh = () => {
        fetchHealthData();
        fetchMetricsData();
    };

    // Initial fetch
    useEffect(() => {
        refresh();
    }, []);

    // Auto-refresh
    useEffect(() => {
        if (!autoRefresh) return;

        const interval = setInterval(() => {
            if (!healthLoading && !metricsLoading) {
                refresh();
            }
        }, refreshInterval);

        return () => clearInterval(interval);
    }, [autoRefresh, refreshInterval, healthLoading, metricsLoading]);

    return {
        healthData,
        metricsData,
        healthLoading,
        metricsLoading,
        healthError,
        metricsError,
        refresh,
    };
}
