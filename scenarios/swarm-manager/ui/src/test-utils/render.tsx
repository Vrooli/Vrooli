import {
  render,
  renderHook,
  type RenderHookOptions,
  type RenderOptions,
} from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import type { ReactElement, ReactNode } from "react";
import { createTestQueryClient } from "./query";

type ProviderOptions = {
  queryClient?: QueryClient;
  initialEntries?: string[];
  initialIndex?: number;
  withRouter?: boolean;
};

type RenderWithProvidersOptions = Omit<RenderOptions, "wrapper"> & ProviderOptions;

function createProviderWrapper({
  queryClient = createTestQueryClient(),
  initialEntries = ["/"],
  initialIndex,
  withRouter = true,
}: ProviderOptions = {}) {
  return function ProviderWrapper({ children }: { children: ReactNode }) {
    const content = (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
    if (!withRouter) {
      return content;
    }
    return (
      <MemoryRouter
        initialEntries={initialEntries}
        initialIndex={initialIndex}
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
      >
        {content}
      </MemoryRouter>
    );
  };
}

export function renderWithProviders(
  ui: ReactElement,
  options: RenderWithProvidersOptions = {},
) {
  const { queryClient, initialEntries, initialIndex, withRouter, ...renderOptions } = options;
  return render(ui, {
    wrapper: createProviderWrapper({ queryClient, initialEntries, initialIndex, withRouter }),
    ...renderOptions,
  });
}

export function renderHookWithProviders<Result, Props>(
  callback: (props: Props) => Result,
  options: Omit<RenderHookOptions<Props>, "wrapper"> & ProviderOptions = {},
) {
  const { queryClient, initialEntries, initialIndex, withRouter, ...renderOptions } = options;
  return renderHook(callback, {
    wrapper: createProviderWrapper({ queryClient, initialEntries, initialIndex, withRouter }),
    ...renderOptions,
  });
}

export function createRouterWrapper(initialEntries: string[] = ["/"], initialIndex?: number) {
  return function RouterWrapper({ children }: { children: ReactNode }) {
    return (
      <MemoryRouter
        initialEntries={initialEntries}
        initialIndex={initialIndex}
        future={{ v7_relativeSplatPath: true, v7_startTransition: true }}
      >
        {children}
      </MemoryRouter>
    );
  };
}
