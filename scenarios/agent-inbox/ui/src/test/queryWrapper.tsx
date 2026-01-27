/**
 * Test utilities for React Query testing.
 *
 * Provides a pre-configured QueryClient wrapper for use in tests,
 * ensuring consistent query client configuration across test suites.
 *
 * @example
 * import { createQueryWrapper, createQueryClient } from '@/test/queryWrapper';
 *
 * const { result } = renderHook(() => useMyHook(), {
 *   wrapper: createQueryWrapper(),
 * });
 */

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";

/**
 * Create a QueryClient configured for testing.
 *
 * Configuration:
 * - Retry disabled to make tests deterministic
 * - No cache time to prevent test pollution
 * - Errors not logged to console
 */
export function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
        staleTime: 0,
      },
      mutations: {
        retry: false,
      },
    },
  });
}

/**
 * Create a wrapper component for renderHook tests.
 *
 * @param queryClient - Optional custom QueryClient. Creates a new one if not provided.
 * @returns A wrapper component for renderHook
 *
 * @example
 * const { result } = renderHook(() => useChats(), {
 *   wrapper: createQueryWrapper(),
 * });
 */
export function createQueryWrapper(queryClient?: QueryClient) {
  const client = queryClient ?? createQueryClient();
  return ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client }, children);
}

/**
 * Create a wrapper component with multiple providers.
 * Useful when hooks need additional context providers.
 *
 * @param providers - Array of provider components to wrap with
 * @returns A wrapper component that nests all providers
 *
 * @example
 * const wrapper = createMultiWrapper([
 *   (children) => <QueryClientProvider client={client}>{children}</QueryClientProvider>,
 *   (children) => <CompletionClientProvider value={mockClient}>{children}</CompletionClientProvider>,
 * ]);
 */
export function createMultiWrapper(
  providers: ((children: ReactNode) => ReactNode)[]
) {
  return ({ children }: { children: ReactNode }) => {
    return providers.reduceRight((acc, provider) => provider(acc), children);
  };
}
