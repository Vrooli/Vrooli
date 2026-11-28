import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, type RenderOptions, type RenderResult } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";
export { expectNoA11yViolations } from "./a11y";

export type ProviderRenderOptions = Omit<RenderOptions, "wrapper"> & {
  withoutQueryClient?: boolean;
  withoutRouter?: boolean;
  initialEntries?: string[];
};

export type ProviderRenderResult = RenderResult & { queryClient: QueryClient };

export function renderWithProviders(ui: ReactElement, options: ProviderRenderOptions = {}): ProviderRenderResult {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 }, mutations: { retry: false } }
  });
  const { withoutRouter, withoutQueryClient, initialEntries, ...renderOptions } = options;
  const Wrapper = ({ children }: { children: ReactNode }) => {
    let tree = children;
    if (!withoutRouter && initialEntries) tree = <MemoryRouter initialEntries={initialEntries}>{tree}</MemoryRouter>;
    if (!withoutQueryClient) tree = <QueryClientProvider client={queryClient}>{tree}</QueryClientProvider>;
    return tree;
  };
  return { ...render(ui, { ...renderOptions, wrapper: Wrapper }), queryClient };
}
