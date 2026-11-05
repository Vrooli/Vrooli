import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, renderHook, type RenderHookOptions, type RenderOptions } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";

export function createTestQueryClient() {
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

interface QueryClientRenderOptions extends Omit<RenderOptions, "wrapper"> {
  queryClient?: QueryClient;
}

export function renderWithQueryClient(
  ui: ReactElement,
  options: QueryClientRenderOptions = {},
) {
  const queryClient = options.queryClient ?? createTestQueryClient();

  function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  }

  return {
    queryClient,
    ...render(ui, { ...options, wrapper: Wrapper }),
  };
}

type HookOptions<Props> = Omit<RenderHookOptions<Props>, "wrapper"> & {
  queryClient?: QueryClient;
};

export function renderHookWithQueryClient<Result, Props>(
  hook: (initialProps: Props) => Result,
  options: HookOptions<Props> = {},
) {
  const queryClient = options.queryClient ?? createTestQueryClient();

  function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  }

  return {
    queryClient,
    ...renderHook(hook, { ...options, wrapper: Wrapper }),
  };
}
