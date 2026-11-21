/**
 * Canonical render helper for prompt-manager UI tests.
 *
 * Mirrors the option surface of `@vrooli/api-base/testing`'s renderWithProviders
 * but renders through this scenario's own React, react-router and react-query
 * packages. The api-base companion carries its own React 18 runtime under a
 * `file:` dependency; rendering React 19 elements through it throws
 * "Objects are not valid as a React child". Keeping the provider tree local
 * guarantees one React runtime per test process.
 */
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, type RenderOptions, type RenderResult } from '@testing-library/react'
import type { ComponentType, ReactElement, ReactNode } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { createTestQueryClient } from '@/test/query'

export type TestProvider = (children: ReactNode) => ReactNode

export interface ProviderRenderOptions extends Omit<RenderOptions, 'wrapper'> {
  queryClient?: QueryClient
  withoutQueryClient?: boolean
  routerEntries?: string[]
  initialIndex?: number
  initialEntries?: string[]
  withoutRouter?: boolean
  withRouter?: boolean
  route?: string
  wrapper?: ComponentType<{ children: ReactNode }>
  extraProviders?: TestProvider
}

export interface ProviderRenderResult extends RenderResult {
  queryClient: QueryClient
}

let configuredProviders: TestProvider | undefined

/** Registers scenario-owned providers for the default render path. */
export function configureTestProviders(provider: TestProvider | undefined): void {
  configuredProviders = provider
}

export function renderWithProviders(
  ui: ReactElement,
  options: ProviderRenderOptions = {},
): ProviderRenderResult {
  const {
    queryClient = createTestQueryClient(),
    withoutQueryClient = false,
    initialEntries,
    routerEntries,
    initialIndex,
    withoutRouter = false,
    withRouter,
    route,
    wrapper: LegacyWrapper,
    extraProviders = configuredProviders,
    ...rest
  } = options
  const resolvedRouterEntries = routerEntries ?? initialEntries ?? (route ? [route] : ['/'])
  const resolvedWithoutRouter = withRouter === undefined ? withoutRouter : !withRouter

  const Wrapper = ({ children }: { children: ReactNode }) => {
    const providerTree = extraProviders ? extraProviders(children) : children
    const wrapped = LegacyWrapper ? <LegacyWrapper>{providerTree}</LegacyWrapper> : providerTree
    const routed = resolvedWithoutRouter ? wrapped : (
      <MemoryRouter
        initialEntries={resolvedRouterEntries}
        initialIndex={initialIndex}
        future={{ v7_startTransition: true, v7_relativeSplatPath: true }}
      >
        {wrapped}
      </MemoryRouter>
    )
    return withoutQueryClient ? routed : <QueryClientProvider client={queryClient}>{routed}</QueryClientProvider>
  }

  return { ...render(ui, { wrapper: Wrapper, ...rest }), queryClient }
}

export { renderWithProviders as render }
export {
  act,
  cleanup,
  fireEvent,
  renderHook,
  screen,
  waitFor,
  waitForElementToBeRemoved,
  within,
} from '@testing-library/react'
