import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, type RenderOptions } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";

export interface ProviderRenderOptions extends RenderOptions {
  queryClient?: QueryClient;
  routerEntries?: string[];
  withoutRouter?: boolean;
}

export type ProviderRenderResult = ReturnType<typeof render>;

export function renderWithProviders(ui: ReactElement, options: ProviderRenderOptions = {}): ProviderRenderResult {
  const { queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } }), routerEntries = ["/"], withoutRouter = false, wrapper, ...rest } = options;
  const Providers = wrapper ?? (({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      {withoutRouter ? children : <MemoryRouter initialEntries={routerEntries}>{children}</MemoryRouter>}
    </QueryClientProvider>
  ));
  return render(ui, { wrapper: Providers, ...rest });
}
