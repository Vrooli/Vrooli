/**
 * Hook to fetch the external URL for an embedded service.
 *
 * Uses React Query with a short stale window because the peer's port can
 * change after a restart. React Query refreshes stale values when the window
 * regains focus. Silently returns null on failure
 * (the service may not be available).
 */

import { useQuery } from "@tanstack/react-query";
import { embeddedService } from "../services";

export interface UseEmbeddedServiceUrlResult {
  url: string | null;
  isLoading: boolean;
}

export function useEmbeddedServiceUrl(serviceName: string): UseEmbeddedServiceUrlResult {
  const { data, isLoading } = useQuery<string | null>({
    queryKey: ["embedded-service-url", serviceName],
    queryFn: () => embeddedService.getExternalUrl(serviceName),
    staleTime: 2_000,
    gcTime: Infinity,
    refetchOnWindowFocus: true,
    retry: false,
    // Silently return null on error — the embedded service may not be available
    placeholderData: null,
  });

  return {
    url: data ?? null,
    isLoading,
  };
}
