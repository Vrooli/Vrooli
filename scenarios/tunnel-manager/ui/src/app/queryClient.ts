import { QueryClient } from "@tanstack/react-query";

/**
 * Production query policy: capability failures should become visible state
 * immediately. Retrying every unavailable remote capability with React
 * Query's default exponential backoff leaves the operator staring at a
 * skeleton and makes a healthy partial/local-mode dashboard look broken.
 */
export function createAppQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        refetchOnWindowFocus: false,
      },
      mutations: {
        retry: false,
      },
    },
  });
}
