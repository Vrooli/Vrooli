/**
 * renderWithProviders — the canonical render helper for component tests.
 *
 * Wraps the rendered tree in the same providers a real `<App />` mount needs,
 * so provider setup stays centralized instead of drifting between test files:
 *
 *   - QueryClientProvider with retry disabled (tests should not retry on
 *     simulated 5xx; they should fail fast) — mirrors `main.tsx`.
 *   - MemoryRouter so components that use react-router (NavLink, Outlet,
 *     useNavigate, useParams) render without crashing. Future flags match
 *     `App.tsx` so tests never emit the v7 upgrade warnings.
 *   - AdminAuthProvider / LandingVariantProvider — the app-level context
 *     providers. They are opt-in (`withAuth`, `withVariant`) because most
 *     presentational tests do not need them, and admin tests typically mock
 *     the provider modules to seed a specific auth/variant state.
 *
 * Usage:
 *
 *   import { renderWithProviders } from '@/test-utils';
 *
 *   // Plain presentational component (QueryClient + router only):
 *   renderWithProviders(<Button>Save</Button>);
 *
 *   // Admin route that reads auth + variant context, mounted at a route:
 *   renderWithProviders(<AdminHome />, {
 *     routerEntries: ['/admin'],
 *     withAuth: true,
 *     withVariant: true,
 *   });
 *
 * The returned `queryClient` is exposed for tests that need to seed the cache
 * or assert queries fired (e.g. `queryClient.getQueryData(...)`).
 */
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { type RenderOptions, type RenderResult, render } from '@testing-library/react';
import type { ReactElement, ReactNode } from 'react';
import { MemoryRouter } from 'react-router-dom';

import { AdminAuthProvider } from '../app/providers/AdminAuthProvider';
import { LandingVariantProvider } from '../app/providers/LandingVariantProvider';

export interface ProviderRenderOptions extends Omit<RenderOptions, 'wrapper'> {
  /**
   * Override the default QueryClient. Useful for tests that need to seed cache
   * state, share a client across renders, or inspect mutation state after the
   * render returns.
   */
  queryClient?: QueryClient;
  /**
   * Initial entries for the in-memory router. Defaults to `['/']`. Pass a
   * specific path (e.g. `['/admin/analytics']`) to render a particular route
   * or to satisfy `useParams`.
   */
  routerEntries?: string[];
  /**
   * When true, skip wrapping in MemoryRouter. Use for tests that already
   * render a router of their own.
   */
  withoutRouter?: boolean;
  /**
   * Wrap the tree in AdminAuthProvider so components can read `useAdminAuth`.
   * Defaults to false. Mock `../app/providers/AdminAuthProvider` to seed a
   * specific authenticated state.
   */
  withAuth?: boolean;
  /**
   * Wrap the tree in LandingVariantProvider so components can read
   * `useLandingVariant`. Defaults to false. Mock
   * `../app/providers/LandingVariantProvider` to seed a specific variant.
   */
  withVariant?: boolean;
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
    routerEntries = ['/'],
    withoutRouter = false,
    withAuth = false,
    withVariant = false,
    ...rest
  } = options;

  const Wrapper = ({ children }: { children: ReactNode }) => {
    let tree: ReactNode = children;
    if (withVariant) {
      tree = <LandingVariantProvider>{tree}</LandingVariantProvider>;
    }
    if (withAuth) {
      tree = <AdminAuthProvider>{tree}</AdminAuthProvider>;
    }
    if (!withoutRouter) {
      tree = (
        <MemoryRouter
          initialEntries={routerEntries}
          future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
        >
          {tree}
        </MemoryRouter>
      );
    }
    return <QueryClientProvider client={queryClient}>{tree}</QueryClientProvider>;
  };

  return { ...render(ui, { wrapper: Wrapper, ...rest }), queryClient };
}
