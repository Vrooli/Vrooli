import type { ReactElement, ReactNode } from "react";
import { render, type RenderOptions, type RenderResult } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";

interface ProviderOptions extends Omit<RenderOptions, "wrapper"> {
  route?: string;
  queryClient?: QueryClient;
  wrapper?: RenderOptions["wrapper"];
  withoutRouter?: boolean;
}

export function renderWithProviders(ui: ReactElement, options: ProviderOptions = {}): RenderResult {
  const { route = "/", queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } }), wrapper: customWrapper, withoutRouter = false, ...renderOptions } = options;
  const defaultWrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      {withoutRouter ? children : <MemoryRouter initialEntries={[route]}>{children}</MemoryRouter>}
    </QueryClientProvider>
  );
  return render(ui, { wrapper: customWrapper ?? defaultWrapper, ...renderOptions });
}
