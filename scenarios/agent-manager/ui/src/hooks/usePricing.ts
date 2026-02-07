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
    const errorData = await response.json().catch(() => ({})) as Record<string, unknown>;
    const message =
      typeof errorData.message === "string" ? errorData.message
        : typeof errorData.error === "string" ? errorData.error
        : `Request failed: ${response.status}`;
    throw new Error(message);
  }

  if (response.status === 204) {
    return {} as T;
  }

  const json: unknown = await response.json();
  return json as T;
}

// =============================================================================
// Model Pricing Hooks
// =============================================================================

export function useModelPricing() {
  const { data, loading, error, setData, setLoading, setError } = useApiState<ModelPricingListItem[]>([]);

  const fetchModels = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiRequest<ModelPricingListResponse>("/pricing/models");
      setData(data.models ?? []);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [setData, setLoading, setError]);

  useEffect(() => {
    fetchModels();
  }, [fetchModels]);

  return { data, loading, error, refetch: fetchModels };
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
  const { data, loading, error, setData, setLoading, setError } = useApiState<PriceOverride[]>([]);

  const fetchOverrides = useCallback(async () => {
    if (!model) {
      setData([]);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const data = await apiRequest<OverridesResponse>(`/pricing/models/${encodeURIComponent(model)}/overrides`);
      setData(data.overrides ?? []);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [model, setData, setLoading, setError]);

  useEffect(() => {
    fetchOverrides();
  }, [fetchOverrides]);

  return { data, loading, error, refetch: fetchOverrides };
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
  const { data, loading, error, setData, setLoading, setError } = useApiState<ModelAlias[]>([]);

  const fetchAliases = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams();
      if (runnerType) {
        params.set("runner_type", runnerType);
      }
      const queryString = params.toString();
      const endpoint = "/pricing/aliases" + (queryString ? `?${queryString}` : "");
      const data = await apiRequest<AliasesResponse>(endpoint);
      setData(data.aliases ?? []);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [runnerType, setData, setLoading, setError]);

  useEffect(() => {
    fetchAliases();
  }, [fetchAliases]);

  return { data, loading, error, refetch: fetchAliases };
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
  const { data, loading, error, setData, setLoading, setError } = useApiState<PricingSettings | null>(null);

  const fetchSettings = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiRequest<PricingSettings>("/pricing/settings");
      setData(data);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [setData, setLoading, setError]);

  useEffect(() => {
    fetchSettings();
  }, [fetchSettings]);

  return { data, loading, error, refetch: fetchSettings };
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
  const { data, loading, error, setData, setLoading, setError } = useApiState<CacheStatusResponse | null>(null);

  const fetchStatus = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiRequest<CacheStatusResponse>("/pricing/cache");
      setData(data);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [setData, setLoading, setError]);

  useEffect(() => {
    fetchStatus();
  }, [fetchStatus]);

  return { data, loading, error, refetch: fetchStatus };
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
