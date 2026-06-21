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
 *   - ThemeProvider with an explicit initial choice so tests don't depend on
 *     localStorage or `prefers-color-scheme`
 *   - MemoryRouter so components that use react-router (NavLink, Outlet,
 *     useNavigate) render without crashing
 *
 * Usage:
 *
 *   import { renderWithProviders } from "@/test-utils";
 *   const { queryClient } = renderWithProviders(<MyComponent />);
 *
 *   // Render a specific route:
 *   renderWithProviders(<App />, { routerEntries: ["/notes"] });
 *
 * The returned `queryClient` is exposed for tests that need to seed the
 * cache or assert queries fired (e.g. `queryClient.getQueryData(...)`).
 */
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { type RenderOptions, type RenderResult, render } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";
import { I18nextProvider } from "react-i18next";
import { MemoryRouter } from "react-router-dom";

import { i18n } from "../i18n";
import { ThemeProvider } from "../theme/ThemeProvider";
import { type ThemeChoice } from "../theme/themeContextValue";
import { ScenarioProvider } from "../features/perf/ScenarioContext";

export interface ProviderRenderOptions extends Omit<RenderOptions, "wrapper"> {
  /**
   * Override the default QueryClient. Useful for tests that need to seed
   * cache state, share a client across multiple renders, or inspect
   * mutation state after the render returns.
   */
  queryClient?: QueryClient;
  /**
   * Initial entries for the in-memory router. Defaults to `["/"]`. Pass a
   * specific path (e.g. `["/notes"]`) to render a particular route.
   */
  routerEntries?: string[];
  /**
   * Initial theme choice for the ThemeProvider. Defaults to `"light"` so tests
   * never read localStorage or `prefers-color-scheme`.
   */
  initialTheme?: ThemeChoice;
  /**
   * When true, skip wrapping in MemoryRouter. Use for tests that already
   * render a `<RouterProvider>` of their own (e.g. routing tests).
   */
  withoutRouter?: boolean;
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
  const {
    queryClient = buildClient(),
    routerEntries = ["/"],
    initialTheme = "light",
    withoutRouter = false,
    ...rest
  } = options;

  const Wrapper = ({ children }: { children: ReactNode }) => {
    const themed = (
      <ThemeProvider initialChoice={initialTheme}>
        {/* ScenarioProvider owns the "current scenario" every per-scenario
            workflow reads; mirror the production composition so components that
            call useScenario render in tests. Its fleet query degrades to the
            selected scenario when unmocked, so it never crashes the render. */}
        <ScenarioProvider>{children}</ScenarioProvider>
      </ThemeProvider>
    );
    const routed = withoutRouter ? themed : (
      <MemoryRouter
        initialEntries={routerEntries}
        future={{ v7_startTransition: true, v7_relativeSplatPath: true }}
      >
        {themed}
      </MemoryRouter>
    );
    return (
      <QueryClientProvider client={queryClient}>
        <I18nextProvider i18n={i18n}>{routed}</I18nextProvider>
      </QueryClientProvider>
    );
  };

  return { ...render(ui, { wrapper: Wrapper, ...rest }), queryClient };
}
