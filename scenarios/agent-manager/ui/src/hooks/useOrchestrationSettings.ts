import { useCallback, useEffect, useState } from "react";
import { getApiBaseUrl } from "../lib/utils";

export interface OrchestrationSettings {
  runExecution: {
    runTimeoutMinutes: number;
    maxConcurrentRuns: number;
    maxTurns: number;
  };
  safetyIsolation: {
    requireSandbox: boolean;
    requireApproval: boolean;
    networkAccess: "none" | "localhost" | "full";
  };
  healthDetection: {
    heartbeatIntervalSeconds: number;
    staleThresholdSeconds: number;
    maxRecoveryAgeSeconds: number;
    reconcilerIntervalSeconds: number;
  };
  processTermination: {
    gracePeriodSeconds: number;
    killProcessGroup: boolean;
    killOrphans: boolean;
    orphanGracePeriodSeconds: number;
    terminationMaxRetries: number;
  };
}

async function apiRequest<T>(
  endpoint: string,
  options: RequestInit = {},
): Promise<T> {
  const baseUrl = getApiBaseUrl();
  const url = endpoint.startsWith("http") ? endpoint : baseUrl + endpoint;

  const response = await fetch(url, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options.headers,
    },
  });

  if (!response.ok) {
    const errorData = (await response.json().catch(() => ({}))) as Record<
      string,
      unknown
    >;
    const message =
      typeof errorData.message === "string"
        ? errorData.message
        : typeof errorData.error === "string"
          ? errorData.error
          : `Request failed: ${response.status}`;
    throw new Error(message);
  }

  if (response.status === 204) {
    return {} as T;
  }

  const json: unknown = await response.json();
  return json as T;
}

export function useOrchestrationSettings() {
  const [data, setData] = useState<OrchestrationSettings | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchSettings = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await apiRequest<OrchestrationSettings>(
        "/orchestration-settings",
      );
      setData(result);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, []);

  const updateSettings = useCallback(
    async (settings: OrchestrationSettings): Promise<OrchestrationSettings> => {
      setLoading(true);
      setError(null);
      try {
        const result = await apiRequest<OrchestrationSettings>(
          "/orchestration-settings",
          {
            method: "PUT",
            body: JSON.stringify(settings),
          },
        );
        setData(result);
        return result;
      } catch (err) {
        setError((err as Error).message);
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [],
  );

  const resetSettings =
    useCallback(async (): Promise<OrchestrationSettings> => {
      setLoading(true);
      setError(null);
      try {
        const result = await apiRequest<OrchestrationSettings>(
          "/orchestration-settings/reset",
          {
            method: "POST",
          },
        );
        setData(result);
        return result;
      } catch (err) {
        setError((err as Error).message);
        throw err;
      } finally {
        setLoading(false);
      }
    }, []);

  useEffect(() => {
    fetchSettings();
  }, [fetchSettings]);

  return { data, loading, error, refetch: fetchSettings, updateSettings, resetSettings };
}
