/**
 * Custom render function with providers for testing.
 * Wraps components with QueryClientProvider configured for testing.
 */

import React from "react";
import { render, type RenderOptions, type RenderResult } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

/** Options for renderWithProviders */
export interface RenderWithProvidersOptions extends Omit<RenderOptions, "wrapper"> {
  /** Custom QueryClient instance */
  queryClient?: QueryClient;
}

/**
 * Creates a QueryClient configured for testing (no retries, no garbage collection).
 */
export function createTestQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: Infinity,
      },
      mutations: {
        retry: false,
      },
    },
  });
}

/**
 * Creates a wrapper component with all required providers for testing.
 */
function createWrapper(queryClient: QueryClient) {
  return function TestWrapper({ children }: { children: React.ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        {children}
      </QueryClientProvider>
    );
  };
}

/**
 * Render with all necessary providers for testing.
 * Automatically wraps the component with QueryClientProvider.
 *
 * @example
 * ```tsx
 * import { renderWithProviders, screen } from "@/test-utils";
 *
 * it("renders correctly", () => {
 *   renderWithProviders(<MyComponent />);
 *   expect(screen.getByText("Hello")).toBeInTheDocument();
 * });
 * ```
 */
export function renderWithProviders(
  ui: React.ReactElement,
  options: RenderWithProvidersOptions = {}
): RenderResult & { queryClient: QueryClient } {
  const { queryClient = createTestQueryClient(), ...renderOptions } = options;
  const Wrapper = createWrapper(queryClient);

  const result = render(ui, { wrapper: Wrapper, ...renderOptions });

  return {
    ...result,
    queryClient,
  };
}

/**
 * Creates a wrapper function for use with renderHook.
 * Returns both the wrapper and the queryClient for assertions.
 *
 * @example
 * ```tsx
 * import { createHookWrapper } from "@/test-utils";
 * import { renderHook } from "@testing-library/react";
 *
 * it("uses the hook correctly", () => {
 *   const { wrapper } = createHookWrapper();
 *   const { result } = renderHook(() => useMyHook(), { wrapper });
 *   expect(result.current.value).toBe("expected");
 * });
 * ```
 */
export function createHookWrapper(queryClient?: QueryClient): {
  wrapper: React.ComponentType<{ children: React.ReactNode }>;
  queryClient: QueryClient;
} {
  const client = queryClient ?? createTestQueryClient();
  return {
    wrapper: createWrapper(client),
    queryClient: client,
  };
}
