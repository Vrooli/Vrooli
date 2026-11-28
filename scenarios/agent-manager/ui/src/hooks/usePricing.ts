import { apiRequest } from "./useApiTransport";
import { useApiMutation, useApiQuery } from "./useApiQuery";
import type {
  CacheStatusResponse,
  ModelPricingListItem,
  ModelPricingListResponse,
  OverridesResponse,
  PriceOverride,
  PricingComponent,
  PricingSettings,
  SetOverrideRequest,
  UpdatePricingSettingsRequest,
} from "../types";

// Pricing reads are shared by the tab and its edit dialog: both mount the same query, so a mount must not
// refetch what is fresh, and a background refresh must not read as a first load (which would unmount the dialog).
const READ = { staleTime: 30_000, firstLoadOnly: true } as const;

const modelPath = (model: string) => `/pricing/models/${encodeURIComponent(model)}`;

export function useModelPricing() {
  return useApiQuery<ModelPricingListItem[]>(
    ["pricing", "models"],
    async (signal) => (await apiRequest<ModelPricingListResponse>("/pricing/models", { signal })).models ?? [],
    { ...READ, fallback: [] },
  );
}

export function useRecalculateModelPricing() {
  const { run, loading, error } = useApiMutation((model: string) =>
    apiRequest<{ status: string }>(`${modelPath(model)}/recalculate`, { method: "POST" }));
  return { recalculate: async (model: string): Promise<void> => void (await run(model)), loading, error };
}

export function useModelOverrides(model: string | null) {
  return useApiQuery<PriceOverride[]>(
    ["pricing", "overrides", model],
    async (signal) => (await apiRequest<OverridesResponse>(`${modelPath(model ?? "")}/overrides`, { signal })).overrides ?? [],
    { ...READ, enabled: !!model, fallback: [] },
  );
}

export function useSetOverride() {
  const { run, loading, error } = useApiMutation((model: string, request: SetOverrideRequest) =>
    apiRequest<{ status: string }>(`${modelPath(model)}/overrides`, { method: "PUT", body: JSON.stringify(request) }));
  return { setOverride: async (model: string, request: SetOverrideRequest): Promise<void> => void (await run(model, request)), loading, error };
}

export function useDeleteOverride() {
  const { run, loading, error } = useApiMutation((model: string, component: PricingComponent) =>
    apiRequest<{ status: string }>(`${modelPath(model)}/overrides/${encodeURIComponent(component)}`, { method: "DELETE" }));
  return { deleteOverride: async (model: string, component: PricingComponent): Promise<void> => void (await run(model, component)), loading, error };
}

export function usePricingSettings() {
  return useApiQuery<PricingSettings>(["pricing", "settings"], (signal) => apiRequest<PricingSettings>("/pricing/settings", { signal }), READ);
}

export function useUpdatePricingSettings() {
  const { run, loading, error } = useApiMutation((request: UpdatePricingSettingsRequest) =>
    apiRequest<PricingSettings>("/pricing/settings", { method: "PUT", body: JSON.stringify(request) }));
  return { updateSettings: run, loading, error };
}

export function usePricingCacheStatus() {
  return useApiQuery<CacheStatusResponse>(["pricing", "cache"], (signal) => apiRequest<CacheStatusResponse>("/pricing/cache", { signal }), READ);
}

export function useRefreshAllPricing() {
  const { run, loading, error } = useApiMutation(() => apiRequest<{ status: string }>("/pricing/refresh", { method: "POST" }));
  return { refreshAll: async (): Promise<void> => void (await run()), loading, error };
}
