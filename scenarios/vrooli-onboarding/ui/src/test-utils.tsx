import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, type RenderOptions } from "@testing-library/react";
import { vi } from "vitest";
import type { ReactElement } from "react";

/** Creates a QueryClient configured for tests (no retries, no refetch). */
export function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false, refetchOnWindowFocus: false } },
  });
}

/** Renders a component wrapped in QueryClientProvider with a fresh test client. */
export function renderWithQueryClient(
  ui: ReactElement,
  options?: Omit<RenderOptions, "wrapper">,
) {
  const queryClient = createTestQueryClient();
  return render(ui, {
    wrapper: ({ children }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    ),
    ...options,
  });
}

/** Mocks globalThis.fetch to resolve with the given JSON body. */
export function mockFetchSuccess(body: unknown) {
  globalThis.fetch = vi.fn().mockResolvedValue({
    ok: true,
    json: () => Promise.resolve(body),
  });
}

/** Mocks globalThis.fetch to resolve with a non-ok status (simulates API error). */
export function mockFetchError(status = 500) {
  globalThis.fetch = vi.fn().mockResolvedValue({
    ok: false,
    status,
  });
}

/** Mocks globalThis.fetch to never resolve (simulates loading state). */
export function mockFetchPending() {
  globalThis.fetch = vi.fn().mockImplementation(() => new Promise(() => {}));
}
