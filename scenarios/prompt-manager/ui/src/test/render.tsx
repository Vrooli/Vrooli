import React from 'react'
import { renderHook, type RenderHookOptions } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { ThemeProvider } from '@/hooks/use-theme'
import { Toaster } from '@/components/ui/toaster'
import { createTestQueryClient } from './query'

interface ProviderOptions {
  queryClient?: QueryClient
  route?: string
  withRouter?: boolean
  withQueryClient?: boolean
  withTheme?: boolean
  withToaster?: boolean
}

export function createTestWrapper(options: ProviderOptions = {}) {
  const {
    queryClient = createTestQueryClient(),
    route = '/',
    withRouter = true,
    withQueryClient = true,
    withTheme = true,
    withToaster = false,
  } = options

  return function TestWrapper({ children }: { children: React.ReactNode }) {
    let tree = <>{children}</>

    if (withTheme) {
      tree = <ThemeProvider>{tree}</ThemeProvider>
    }

    if (withRouter) {
      tree = (
        <MemoryRouter
          initialEntries={[route]}
          future={{ v7_startTransition: true, v7_relativeSplatPath: true }}
        >
          {tree}
        </MemoryRouter>
      )
    }

    if (withQueryClient) {
      tree = <QueryClientProvider client={queryClient}>{tree}</QueryClientProvider>
    }

    if (withToaster) {
      tree = (
        <>
          {tree}
          <Toaster />
        </>
      )
    }

    return tree
  }
}

type RenderHookWithProvidersOptions<Props> =
  ProviderOptions & Omit<RenderHookOptions<Props>, 'wrapper'>

export function renderHookWithProviders<Result, Props>(
  callback: (props: Props) => Result,
  options: RenderHookWithProvidersOptions<Props> = {}
) {
  const { queryClient = createTestQueryClient(), ...renderOptions } = options
  const wrapper = createTestWrapper({ ...options, queryClient })
  const result = renderHook(callback, { ...renderOptions, wrapper })

  return {
    ...result,
    queryClient,
  }
}
