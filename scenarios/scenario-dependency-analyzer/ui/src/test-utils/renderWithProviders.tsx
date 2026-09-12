import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, type RenderOptions, type RenderResult } from "@testing-library/react";
import type { ComponentType, ReactElement, ReactNode } from "react";
import { I18nextProvider } from "react-i18next";
import { createInstance, type i18n as I18nInstance } from "i18next";
import { MemoryRouter } from "react-router-dom";

export type TestProvider = (children: ReactNode) => ReactNode;
export interface ProviderRenderOptions extends Omit<RenderOptions, "wrapper"> {
  queryClient?: QueryClient; withoutQueryClient?: boolean; routerEntries?: string[]; initialIndex?: number;
  initialEntries?: string[]; withoutRouter?: boolean; withRouter?: boolean; route?: string;
  wrapper?: ComponentType<{ children: ReactNode }>; withoutI18n?: boolean; extraProviders?: TestProvider;
}
export interface ProviderRenderResult extends RenderResult { queryClient: QueryClient }
const defaultI18n = createInstance();
void defaultI18n.init({ lng: "cimode", fallbackLng: false, resources: {}, interpolation: { escapeValue: false } });

export function createTestQueryClient(): QueryClient {
  return new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0, staleTime: 0, refetchOnWindowFocus: false }, mutations: { retry: false } } });
}
export function createHookWrapper(queryClient = createTestQueryClient()) {
  const Wrapper = ({ children }: { children: ReactNode }) => <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  return { queryClient, wrapper: Wrapper };
}
export function configureTestProviders(_provider: TestProvider | undefined): void {}
export function renderWithProviders(ui: ReactElement, options: ProviderRenderOptions = {}): ProviderRenderResult {
  const { queryClient = createTestQueryClient(), withoutQueryClient = false, initialEntries, routerEntries, initialIndex, withoutRouter = false, withRouter, route, wrapper: LegacyWrapper, withoutI18n = false, extraProviders, ...rest } = options;
  const entries = routerEntries ?? initialEntries ?? (route ? [route] : ["/"]);
  const routed = (children: ReactNode) => {
    const provided = extraProviders ? extraProviders(children) : children;
    const legacy = LegacyWrapper ? <LegacyWrapper>{provided}</LegacyWrapper> : provided;
    const useRouter = withRouter === undefined ? !withoutRouter : withRouter;
    return useRouter ? <MemoryRouter initialEntries={entries} initialIndex={initialIndex}>{legacy}</MemoryRouter> : legacy;
  };
  const Wrapper = ({ children }: { children: ReactNode }) => {
    const localized = withoutI18n ? routed(children) : <I18nextProvider i18n={defaultI18n as I18nInstance}>{routed(children)}</I18nextProvider>;
    return withoutQueryClient ? localized : <QueryClientProvider client={queryClient}>{localized}</QueryClientProvider>;
  };
  return { ...render(ui, { wrapper: Wrapper, ...rest }), queryClient };
}
