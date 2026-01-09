import { useCallback, useEffect, useState } from "react";
import { getApiBaseUrl } from "../lib/utils";
import type {
  AliasesResponse,
  CacheStatusResponse,
  CreateAliasRequest,
  ModelAlias,
  ModelPricingListItem,
  ModelPricingListResponse,
  OverridesResponse,
  PriceOverride,
  PricingComponent,
  PricingSettings,
  SetOverrideRequest,
  UpdatePricingSettingsRequest,
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

// =============================================================================
// Model Pricing Hooks
// =============================================================================

export function useModelPricing() {
  const state = useApiState<ModelPricingListItem[]>([]);

  const fetchModels = useCallback(async () => {
    state.setLoading(true);
    state.setError(null);
    try {
      const data = await apiRequest<ModelPricingListResponse>("/pricing/models");
      state.setData(data.models ?? []);
    } catch (err) {
      state.setError((err as Error).message);
    } finally {
      state.setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchModels();
  }, [fetchModels]);

  return { ...state, refetch: fetchModels };
}

export function useRecalculateModelPricing() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const recalculate = useCallback(async (model: string): Promise<void> => {
    setLoading(true);
    setError(null);
    try {
      await apiRequest<{ status: string }>(`/pricing/models/${encodeURIComponent(model)}/recalculate`, {
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

  return { recalculate, loading, error };
}

// =============================================================================
// Override Hooks
// =============================================================================

export function useModelOverrides(model: string | null) {
  const state = useApiState<PriceOverride[]>([]);

  const fetchOverrides = useCallback(async () => {
    if (!model) {
      state.setData([]);
      return;
    }
    state.setLoading(true);
    state.setError(null);
    try {
      const data = await apiRequest<OverridesResponse>(`/pricing/models/${encodeURIComponent(model)}/overrides`);
      state.setData(data.overrides ?? []);
    } catch (err) {
      state.setError((err as Error).message);
    } finally {
      state.setLoading(false);
    }
  }, [model]);

  useEffect(() => {
    fetchOverrides();
  }, [fetchOverrides]);

  return { ...state, refetch: fetchOverrides };
}

export function useSetOverride() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const setOverride = useCallback(async (model: string, request: SetOverrideRequest): Promise<void> => {
    setLoading(true);
    setError(null);
    try {
      await apiRequest<{ status: string }>(`/pricing/models/${encodeURIComponent(model)}/overrides`, {
        method: "PUT",
        body: JSON.stringify(request),
      });
    } catch (err) {
      const message = (err as Error).message;
      setError(message);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  return { setOverride, loading, error };
}

export function useDeleteOverride() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const deleteOverride = useCallback(async (model: string, component: PricingComponent): Promise<void> => {
    setLoading(true);
    setError(null);
    try {
      await apiRequest<{ status: string }>(
        `/pricing/models/${encodeURIComponent(model)}/overrides/${encodeURIComponent(component)}`,
        { method: "DELETE" }
      );
    } catch (err) {
      const message = (err as Error).message;
      setError(message);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  return { deleteOverride, loading, error };
}

// =============================================================================
// Alias Hooks
// =============================================================================

export function useModelAliases(runnerType?: string) {
  const state = useApiState<ModelAlias[]>([]);

  const fetchAliases = useCallback(async () => {
    state.setLoading(true);
    state.setError(null);
    try {
      const params = new URLSearchParams();
      if (runnerType) {
        params.set("runner_type", runnerType);
      }
      const queryString = params.toString();
      const endpoint = "/pricing/aliases" + (queryString ? `?${queryString}` : "");
      const data = await apiRequest<AliasesResponse>(endpoint);
      state.setData(data.aliases ?? []);
    } catch (err) {
      state.setError((err as Error).message);
    } finally {
      state.setLoading(false);
    }
  }, [runnerType]);

  useEffect(() => {
    fetchAliases();
  }, [fetchAliases]);

  return { ...state, refetch: fetchAliases };
}

export function useCreateAlias() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const createAlias = useCallback(async (request: CreateAliasRequest): Promise<void> => {
    setLoading(true);
    setError(null);
    try {
      await apiRequest<{ status: string }>("/pricing/aliases", {
        method: "POST",
        body: JSON.stringify(request),
      });
    } catch (err) {
      const message = (err as Error).message;
      setError(message);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  return { createAlias, loading, error };
}

// =============================================================================
// Settings Hooks
// =============================================================================

export function usePricingSettings() {
  const state = useApiState<PricingSettings | null>(null);

  const fetchSettings = useCallback(async () => {
    state.setLoading(true);
    state.setError(null);
    try {
      const data = await apiRequest<PricingSettings>("/pricing/settings");
      state.setData(data);
    } catch (err) {
      state.setError((err as Error).message);
    } finally {
      state.setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchSettings();
  }, [fetchSettings]);

  return { ...state, refetch: fetchSettings };
}

export function useUpdatePricingSettings() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const updateSettings = useCallback(async (request: UpdatePricingSettingsRequest): Promise<PricingSettings> => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiRequest<PricingSettings>("/pricing/settings", {
        method: "PUT",
        body: JSON.stringify(request),
      });
      return data;
    } catch (err) {
      const message = (err as Error).message;
      setError(message);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  return { updateSettings, loading, error };
}

// =============================================================================
// Cache Status Hooks
// =============================================================================

export function usePricingCacheStatus() {
  const state = useApiState<CacheStatusResponse | null>(null);

  const fetchStatus = useCallback(async () => {
    state.setLoading(true);
    state.setError(null);
    try {
      const data = await apiRequest<CacheStatusResponse>("/pricing/cache");
      state.setData(data);
    } catch (err) {
      state.setError((err as Error).message);
    } finally {
      state.setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchStatus();
  }, [fetchStatus]);

  return { ...state, refetch: fetchStatus };
}

export function useRefreshAllPricing() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refreshAll = useCallback(async (): Promise<void> => {
    setLoading(true);
    setError(null);
    try {
      await apiRequest<{ status: string }>("/pricing/refresh", {
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

  return { refreshAll, loading, error };
}
