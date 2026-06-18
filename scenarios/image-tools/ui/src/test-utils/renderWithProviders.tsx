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
 *   renderWithProviders(<App />, { routerEntries: ["/jobs"] });
 *
 * The returned `queryClient` is exposed for tests that need to seed the
 * cache or assert queries fired (e.g. `queryClient.getQueryData(...)`).
 */
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { type RenderOptions, type RenderResult, render } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";
import { I18nextProvider } from "react-i18next";
import { MemoryRouter } from "react-router-dom";

import { routerFutureFlags } from "../app/routerFuture";
import { SettingsProvider } from "../features/settings/SettingsProvider";
import { i18n } from "../i18n";
import { ThemeProvider, type ThemeChoice } from "../theme/ThemeProvider";
import { DEFAULT_SETTINGS, type SettingsState } from "../features/settings/useSettings";

export interface ProviderRenderOptions extends Omit<RenderOptions, "wrapper"> {
  /**
   * Override the default QueryClient. Useful for tests that need to seed
   * cache state, share a client across multiple renders, or inspect
   * mutation state after the render returns.
   */
  queryClient?: QueryClient;
  /**
   * Initial entries for the in-memory router. Defaults to `["/"]`. Pass a
   * specific path (e.g. `["/jobs"]`) to render a particular route.
   */
  routerEntries?: string[];
  /**
   * Initial theme choice for the ThemeProvider. Defaults to `"light"` so tests
   * never read localStorage or `prefers-color-scheme`.
   */
  initialTheme?: ThemeChoice;
  /**
   * Initial display/accessibility settings for the SettingsProvider. Defaults
   * to `DEFAULT_SETTINGS` so tests never read localStorage.
   */
  initialSettings?: SettingsState;
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
    initialSettings = DEFAULT_SETTINGS,
    withoutRouter = false,
    ...rest
  } = options;

  const Wrapper = ({ children }: { children: ReactNode }) => {
    const themed = (
      <ThemeProvider initialChoice={initialTheme}>
        <SettingsProvider initialSettings={initialSettings}>{children}</SettingsProvider>
      </ThemeProvider>
    );
    const routed = withoutRouter ? themed : (
      <MemoryRouter initialEntries={routerEntries} future={routerFutureFlags}>
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
