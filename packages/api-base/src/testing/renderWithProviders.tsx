import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { type RenderOptions, type RenderResult, render } from "@testing-library/react";
import type { ComponentType, ReactElement, ReactNode } from "react";
import { I18nextProvider } from "react-i18next";
import { createInstance, type i18n as I18nInstance } from "i18next";
import { MemoryRouter } from "react-router-dom";

/** A wrapper hook for scenario-specific providers such as theme or session. */
export type TestProvider = (children: ReactNode) => ReactNode;

/**
 * An initialized i18next-compatible instance supplied by a scenario. The
 * companion deliberately keeps this boundary structural so consumers do not
 * need to duplicate api-base's i18next dependency just to type-check tests.
 */
export type I18nProviderInstance = object;

export interface ProviderRenderOptions extends Omit<RenderOptions, "wrapper"> {
  queryClient?: QueryClient;
  withoutQueryClient?: boolean;
  routerEntries?: string[];
  initialIndex?: number;
  /** Backwards-compatible alias for older scenario test utilities. */
  initialEntries?: string[];
  withoutRouter?: boolean;
  /** Backwards-compatible inverse alias for withoutRouter. */
	withRouter?: boolean;
	/** Backwards-compatible single route alias used by scenario suites. */
	route?: string;
	/** Backwards-compatible scenario wrapper alias. */
	wrapper?: ComponentType<{ children: ReactNode }>;
  withoutI18n?: boolean;
  /** Additional providers owned by the consuming scenario. */
  extraProviders?: TestProvider;
  /** Supply the scenario's initialized i18n instance when it has one. */
  i18n?: I18nProviderInstance;
}

export interface ProviderRenderResult extends RenderResult {
  queryClient: QueryClient;
}

let configuredProviders: TestProvider | undefined;

/**
 * Registers scenario-owned providers for the default render path. Call this
 * once from the UI's Vitest setup file when components require contexts that
 * cannot live in api-base (theme, session, realtime, or domain providers).
 */
export function configureTestProviders(provider: TestProvider | undefined): void {
  configuredProviders = provider;
}

const defaultI18n = createInstance();
void defaultI18n.init({
  initImmediate: false,
  lng: "cimode",
  fallbackLng: false,
  resources: {},
  interpolation: { escapeValue: false },
});

export function createTestQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
        staleTime: 0,
        refetchOnWindowFocus: false,
      },
      mutations: { retry: false },
    },
  });
}

/** Creates a React Query wrapper for hook tests that do not need DOM rendering. */
export function createHookWrapper(queryClient = createTestQueryClient()) {
  const Wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  return { queryClient, wrapper: Wrapper };
}

const buildClient = createTestQueryClient;

export function renderWithProviders(
  ui: ReactElement,
  options: ProviderRenderOptions = {},
): ProviderRenderResult {
  const {
    queryClient = buildClient(),
    withoutQueryClient = false,
    initialEntries,
    routerEntries,
    initialIndex,
    withoutRouter = false,
		withRouter,
		route,
		wrapper: LegacyWrapper,
    withoutI18n = false,
    extraProviders = configuredProviders,
    i18n = defaultI18n,
    ...rest
  } = options;
	const resolvedRouterEntries = routerEntries ?? initialEntries ?? (route ? [route] : ["/"]);
  const resolvedWithoutRouter = withRouter === undefined ? withoutRouter : !withRouter;

  const Wrapper = ({ children }: { children: ReactNode }) => {
		const providerTree = extraProviders ? extraProviders(children) : children;
		const wrapped = LegacyWrapper ? <LegacyWrapper>{providerTree}</LegacyWrapper> : providerTree;
		const routed = resolvedWithoutRouter ? wrapped : (
      <MemoryRouter
        initialEntries={resolvedRouterEntries}
        initialIndex={initialIndex}
        future={{ v7_startTransition: true, v7_relativeSplatPath: true }}
      >
			{wrapped}
      </MemoryRouter>
    );
    const localized = withoutI18n ? routed : (
      <I18nextProvider i18n={i18n as I18nInstance}>{routed}</I18nextProvider>
    );
    return withoutQueryClient ? localized : <QueryClientProvider client={queryClient}>{localized}</QueryClientProvider>;
  };

  return { ...render(ui, { wrapper: Wrapper, ...rest }), queryClient };
}
