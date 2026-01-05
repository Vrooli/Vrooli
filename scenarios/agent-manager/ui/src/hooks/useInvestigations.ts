import { useCallback, useEffect, useRef, useState } from "react";
import { getApiBaseUrl } from "../lib/utils";
import type {
  ApplyFixesRequest,
  CreateInvestigationRequest,
  Investigation,
  InvestigationStatus,
} from "../types";

interface ApiState<T> {
  data: T | null;
  loading: boolean;
  error: string | null;
}

function useApiState<T>(initialData: T | null = null): ApiState<T> & {
  setData: (data: T | null) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
} {
  const [data, setData] = useState<T | null>(initialData);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  return { data, loading, error, setData, setLoading, setError };
}

async function apiRequest<T>(
  endpoint: string,
  options: RequestInit = {}
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
    const errorData = await response.json().catch(() => ({}));
    const message = errorData.message || errorData.error || `Request failed: ${response.status}`;
    throw new Error(message);
  }

  if (response.status === 204) {
    return {} as T;
  }

  return response.json();
}

interface ListInvestigationsOptions {
  status?: InvestigationStatus;
  limit?: number;
}

export function useInvestigations(options?: ListInvestigationsOptions) {
  const state = useApiState<Investigation[]>([]);

  const fetchInvestigations = useCallback(async () => {
    state.setLoading(true);
    state.setError(null);
    try {
      const params = new URLSearchParams();
      if (options?.status) {
        params.set("status", options.status);
      }
      if (options?.limit) {
        params.set("limit", String(options.limit));
      }
      const queryString = params.toString();
      const endpoint = "/investigations" + (queryString ? `?${queryString}` : "");
      const data = await apiRequest<{ investigations: Investigation[] }>(endpoint);
      state.setData(data.investigations ?? []);
    } catch (err) {
      state.setError((err as Error).message);
    } finally {
      state.setLoading(false);
    }
  }, [options?.status, options?.limit]);

  useEffect(() => {
    fetchInvestigations();
  }, [fetchInvestigations]);

  return { ...state, refetch: fetchInvestigations };
}

export function useInvestigation(id: string | null) {
  const state = useApiState<Investigation>(null);
  const pollIntervalRef = useRef<number | null>(null);

  const fetchInvestigation = useCallback(async () => {
    if (!id) {
      state.setData(null);
      return;
    }
    state.setLoading(true);
    state.setError(null);
    try {
      const data = await apiRequest<{ investigation: Investigation }>(`/investigations/${id}`);
      state.setData(data.investigation);
    } catch (err) {
      state.setError((err as Error).message);
    } finally {
      state.setLoading(false);
    }
  }, [id]);

  // Poll while investigation is running
  useEffect(() => {
    if (!id) return;

    fetchInvestigation();

    // Set up polling if investigation is pending/running
    const poll = () => {
      const inv = state.data;
      if (inv && (inv.status === "pending" || inv.status === "running")) {
        pollIntervalRef.current = window.setTimeout(async () => {
          await fetchInvestigation();
          poll();
        }, 2000);
      }
    };

    poll();

    return () => {
      if (pollIntervalRef.current) {
        clearTimeout(pollIntervalRef.current);
      }
    };
  }, [id, fetchInvestigation, state.data?.status]);

  return { ...state, refetch: fetchInvestigation };
}

export function useActiveInvestigation() {
  const state = useApiState<Investigation | null>(null);
  const pollIntervalRef = useRef<number | null>(null);

  const fetchActiveInvestigation = useCallback(async () => {
    state.setLoading(true);
    state.setError(null);
    try {
      const data = await apiRequest<{ investigation: Investigation | null }>("/investigations/active");
      state.setData(data.investigation ?? null);
    } catch (err) {
      // 404 means no active investigation, which is fine
      if ((err as Error).message.includes("404") || (err as Error).message.includes("not found")) {
        state.setData(null);
      } else {
        state.setError((err as Error).message);
      }
    } finally {
      state.setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchActiveInvestigation();

    // Poll while there's an active investigation
    const poll = () => {
      const inv = state.data;
      if (inv && (inv.status === "pending" || inv.status === "running")) {
        pollIntervalRef.current = window.setTimeout(async () => {
          await fetchActiveInvestigation();
          poll();
        }, 2000);
      }
    };

    poll();

    return () => {
      if (pollIntervalRef.current) {
        clearTimeout(pollIntervalRef.current);
      }
    };
  }, [fetchActiveInvestigation, state.data?.status]);

  return { ...state, refetch: fetchActiveInvestigation };
}

export function useTriggerInvestigation() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const trigger = useCallback(async (request: CreateInvestigationRequest): Promise<Investigation> => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiRequest<{ investigation: Investigation }>("/investigations", {
        method: "POST",
        body: JSON.stringify(request),
      });
      return data.investigation;
    } catch (err) {
      const message = (err as Error).message;
      setError(message);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  return { trigger, loading, error };
}

export function useStopInvestigation() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const stop = useCallback(async (id: string): Promise<void> => {
    setLoading(true);
    setError(null);
    try {
      await apiRequest<void>(`/investigations/${id}/stop`, {
        method: "POST",
      });
    } catch (err) {
      const message = (err as Error).message;
      setError(message);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  return { stop, loading, error };
}

export function useDeleteInvestigation() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const deleteInvestigation = useCallback(async (id: string): Promise<void> => {
    setLoading(true);
    setError(null);
    try {
      await apiRequest<void>(`/investigations/${id}`, {
        method: "DELETE",
      });
    } catch (err) {
      const message = (err as Error).message;
      setError(message);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  return { deleteInvestigation, loading, error };
}

export function useApplyFixes() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const applyFixes = useCallback(async (
    investigationId: string,
    request: ApplyFixesRequest
  ): Promise<Investigation> => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiRequest<{ investigation: Investigation }>(
        `/investigations/${investigationId}/apply-fixes`,
        {
          method: "POST",
          body: JSON.stringify(request),
        }
      );
      return data.investigation;
    } catch (err) {
      const message = (err as Error).message;
      setError(message);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  return { applyFixes, loading, error };
}
