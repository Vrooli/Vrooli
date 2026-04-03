/**
 * Hook to fetch the external URL for an embedded service.
 *
 * Uses React Query with staleTime: Infinity since embedded service URLs
 * don't change during a session. Silently returns null on failure
 * (the service may not be available).
 */

import { useQuery } from "@tanstack/react-query";
import { embeddedService } from "../services";

export interface UseEmbeddedServiceUrlResult {
  url: string | null;
  isLoading: boolean;
}

export function useEmbeddedServiceUrl(serviceName: string): UseEmbeddedServiceUrlResult {
  const { data, isLoading } = useQuery<string | null, Error>({
    queryKey: ["embedded-service-url", serviceName],
    queryFn: () => embeddedService.getExternalUrl(serviceName),
    staleTime: Infinity,
    gcTime: Infinity,
    retry: false,
    // Silently return null on error — the embedded service may not be available
    placeholderData: null,
  });

  return {
    url: data ?? null,
    isLoading,
  };
}
