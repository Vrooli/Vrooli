import { vi } from "vitest";

export interface MockApiConfig {
  defaultLoading?: boolean;
  defaultError?: Error | null;
  defaultData?: any;
}

export function createApiMocks(config: MockApiConfig = {}) {
  const {
    defaultLoading = false,
    defaultError = null,
    defaultData = null,
  } = config;

  const mockFetch = vi.fn().mockResolvedValue(defaultData);
  const mockFetchData = vi.fn().mockResolvedValue({
    data: defaultData,
    errors: defaultError ? [defaultError] : undefined,
    __fetchTimestamp: Date.now(),
  });
  
  return {
    fetchData: mockFetchData,
    fetchLazyWrapper: mockFetch,
    useLazyFetch: vi.fn(() => [
      mockFetch,
      { 
        loading: defaultLoading, 
        error: defaultError,
        data: defaultData,
      },
    ]),
    useFetch: vi.fn(() => ({
      loading: defaultLoading,
      error: defaultError,
      data: defaultData,
      refetch: mockFetch,
    })),
  };
}

export const apiMocks = createApiMocks();
