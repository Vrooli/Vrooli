import { useCallback, useEffect, useRef, useState } from "react";
import { getApiBaseUrl } from "../lib/utils";
import type {
  AgentProfile,
  ApprovalResult,
  ApproveRequest,
  CreateProfileRequest,
  CreateRunRequest,
  CreateTaskRequest,
  DiffResult,
  HealthStatus,
  RejectRequest,
  Run,
  RunEvent,
  RunnerStatus,
  Task,
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
    throw new Error(
      errorData.message || errorData.userMessage || "Request failed: " + response.status
    );
  }

  if (response.status === 204) {
    return {} as T;
  }

  return response.json();
}

// Health hook
export function useHealth() {
  const state = useApiState<HealthStatus>();
  const abortRef = useRef<AbortController | null>(null);

  const fetchHealth = useCallback(async () => {
    if (abortRef.current) {
      abortRef.current.abort();
    }
    const controller = new AbortController();
    abortRef.current = controller;

    state.setLoading(true);
    state.setError(null);

    try {
      const data = await apiRequest<HealthStatus>("/health", {
        signal: controller.signal,
      });
      state.setData(data);
    } catch (err) {
      if ((err as Error).name !== "AbortError") {
        state.setError((err as Error).message);
      }
    } finally {
      state.setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchHealth();
    const interval = setInterval(fetchHealth, 30000);
    return () => {
      clearInterval(interval);
      abortRef.current?.abort();
    };
  }, [fetchHealth]);

  return { ...state, refetch: fetchHealth };
}

// Profiles hook
export function useProfiles() {
  const state = useApiState<AgentProfile[]>([]);

  const fetchProfiles = useCallback(async () => {
    state.setLoading(true);
    state.setError(null);
    try {
      const data = await apiRequest<AgentProfile[]>("/profiles");
      state.setData(data || []);
    } catch (err) {
      state.setError((err as Error).message);
    } finally {
      state.setLoading(false);
    }
  }, []);

  const createProfile = useCallback(
    async (profile: CreateProfileRequest): Promise<AgentProfile> => {
      const created = await apiRequest<AgentProfile>("/profiles", {
        method: "POST",
        body: JSON.stringify(profile),
      });
      await fetchProfiles();
      return created;
    },
    [fetchProfiles]
  );

  const updateProfile = useCallback(
    async (id: string, profile: Partial<CreateProfileRequest>): Promise<AgentProfile> => {
      const updated = await apiRequest<AgentProfile>("/profiles/" + id, {
        method: "PUT",
        body: JSON.stringify(profile),
      });
      await fetchProfiles();
      return updated;
    },
    [fetchProfiles]
  );

  const deleteProfile = useCallback(
    async (id: string): Promise<void> => {
      await apiRequest<void>("/profiles/" + id, { method: "DELETE" });
      await fetchProfiles();
    },
    [fetchProfiles]
  );

  useEffect(() => {
    fetchProfiles();
  }, [fetchProfiles]);

  return {
    ...state,
    refetch: fetchProfiles,
    createProfile,
    updateProfile,
    deleteProfile,
  };
}

// Tasks hook
export function useTasks() {
  const state = useApiState<Task[]>([]);

  const fetchTasks = useCallback(async () => {
    state.setLoading(true);
    state.setError(null);
    try {
      const data = await apiRequest<Task[]>("/tasks");
      state.setData(data || []);
    } catch (err) {
      state.setError((err as Error).message);
    } finally {
      state.setLoading(false);
    }
  }, []);

  const createTask = useCallback(
    async (task: CreateTaskRequest): Promise<Task> => {
      const created = await apiRequest<Task>("/tasks", {
        method: "POST",
        body: JSON.stringify(task),
      });
      await fetchTasks();
      return created;
    },
    [fetchTasks]
  );

  const getTask = useCallback(async (id: string): Promise<Task> => {
    return apiRequest<Task>("/tasks/" + id);
  }, []);

  const cancelTask = useCallback(
    async (id: string): Promise<void> => {
      await apiRequest<void>("/tasks/" + id + "/cancel", { method: "POST" });
      await fetchTasks();
    },
    [fetchTasks]
  );

  useEffect(() => {
    fetchTasks();
  }, [fetchTasks]);

  return {
    ...state,
    refetch: fetchTasks,
    createTask,
    getTask,
    cancelTask,
  };
}

// Runs hook
export function useRuns() {
  const state = useApiState<Run[]>([]);

  const fetchRuns = useCallback(async () => {
    state.setLoading(true);
    state.setError(null);
    try {
      const data = await apiRequest<Run[]>("/runs");
      state.setData(data || []);
    } catch (err) {
      state.setError((err as Error).message);
    } finally {
      state.setLoading(false);
    }
  }, []);

  const createRun = useCallback(
    async (run: CreateRunRequest): Promise<Run> => {
      const created = await apiRequest<Run>("/runs", {
        method: "POST",
        body: JSON.stringify(run),
      });
      await fetchRuns();
      return created;
    },
    [fetchRuns]
  );

  const getRun = useCallback(async (id: string): Promise<Run> => {
    return apiRequest<Run>("/runs/" + id);
  }, []);

  const stopRun = useCallback(
    async (id: string): Promise<void> => {
      await apiRequest<void>("/runs/" + id + "/stop", { method: "POST" });
      await fetchRuns();
    },
    [fetchRuns]
  );

  const getRunEvents = useCallback(
    async (id: string): Promise<RunEvent[]> => {
      return apiRequest<RunEvent[]>("/runs/" + id + "/events");
    },
    []
  );

  const getRunDiff = useCallback(async (id: string): Promise<DiffResult> => {
    return apiRequest<DiffResult>("/runs/" + id + "/diff");
  }, []);

  const approveRun = useCallback(
    async (id: string, req: ApproveRequest): Promise<ApprovalResult> => {
      const result = await apiRequest<ApprovalResult>("/runs/" + id + "/approve", {
        method: "POST",
        body: JSON.stringify(req),
      });
      await fetchRuns();
      return result;
    },
    [fetchRuns]
  );

  const rejectRun = useCallback(
    async (id: string, req: RejectRequest): Promise<void> => {
      await apiRequest<void>("/runs/" + id + "/reject", {
        method: "POST",
        body: JSON.stringify(req),
      });
      await fetchRuns();
    },
    [fetchRuns]
  );

  useEffect(() => {
    fetchRuns();
  }, [fetchRuns]);

  return {
    ...state,
    refetch: fetchRuns,
    createRun,
    getRun,
    stopRun,
    getRunEvents,
    getRunDiff,
    approveRun,
    rejectRun,
  };
}

// Runners hook
export function useRunners() {
  const state = useApiState<Record<string, RunnerStatus>>({});

  const fetchRunners = useCallback(async () => {
    state.setLoading(true);
    state.setError(null);
    try {
      const data = await apiRequest<Record<string, RunnerStatus>>("/runners");
      state.setData(data || {});
    } catch (err) {
      state.setError((err as Error).message);
    } finally {
      state.setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchRunners();
  }, [fetchRunners]);

  return { ...state, refetch: fetchRunners };
}
