/**
 * renderWithProviders — the canonical render helper for component tests.
 *
 * Wraps the rendered tree in every provider a real `<App />` mount needs:
 *
 *   - QueryClientProvider with retry disabled (tests should not retry on
 *     simulated 5xx; they should fail fast)
 *   - I18nextProvider bound to the same singleton `App.tsx` consumes, so
 *     `useTranslation()` resolves the same catalogs in tests as in
 *     production (cimode by default — see `src/test-setup.ts`)
 *
 * Usage:
 *
 *   import { renderWithProviders } from "@/test-utils";
 *   const { queryClient } = renderWithProviders(<MyComponent />);
 *
 * The returned `queryClient` is exposed for tests that need to seed the
 * cache or assert queries fired (e.g. `queryClient.getQueryData(...)`).
 *
 * Future providers (router, theme, auth) layer in here so component
 * tests need not be edited individually when the app shell grows.
 */
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { type RenderOptions, type RenderResult, render } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";
import { I18nextProvider } from "react-i18next";
import { MemoryRouter } from "react-router-dom";

import { i18n } from "../i18n";
import { ThemeProvider } from "../components/theme/ThemeProvider";

export interface ProviderRenderOptions extends Omit<RenderOptions, "wrapper"> {
  /**
   * Override the default QueryClient. Useful for tests that need to seed
   * cache state, share a client across multiple renders, or inspect
   * mutation state after the render returns.
   */
  queryClient?: QueryClient;
  /**
   * Initial route entries for the MemoryRouter wrapper. Defaults to ["/"].
   * Tests that exercise routing — App-level smoke, route-bound pages
   * like FlowDetailPage — set this to drive useParams + Routes.
   */
  routerEntries?: string[];
}

export interface ProviderRenderResult extends RenderResult {
  queryClient: QueryClient;
}

const buildClient = (): QueryClient =>
  new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

export function renderWithProviders(
  ui: ReactElement,
  options: ProviderRenderOptions = {},
): ProviderRenderResult {
  const { queryClient = buildClient(), routerEntries = ["/"], ...rest } = options;

  const Wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <I18nextProvider i18n={i18n}>
          <MemoryRouter initialEntries={routerEntries}>{children}</MemoryRouter>
        </I18nextProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );

  return { ...render(ui, { wrapper: Wrapper, ...rest }), queryClient };
}
