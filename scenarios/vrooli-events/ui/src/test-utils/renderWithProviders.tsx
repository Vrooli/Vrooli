// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md
import React from "react";
import { render, type RenderOptions } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

interface CustomRenderOptions extends RenderOptions {
    queryClient?: QueryClient;
}

/**
 * Renders a component wrapped in all required providers (QueryClient, etc.)
 * with test-friendly defaults (no retries, no refetch on window focus).
 */
export function renderWithProviders(
    ui: React.ReactElement,
    options: CustomRenderOptions = {},
) {
    const {
        queryClient = new QueryClient({
            defaultOptions: {
                queries: {
                    retry: false,
                    refetchOnWindowFocus: false,
                },
            },
        }),
        ...renderOptions
    } = options;

    function Wrapper({ children }: { children: React.ReactNode }) {
        return (
            <QueryClientProvider client={queryClient}>
                {children}
            </QueryClientProvider>
        );
    }

    return render(ui, { wrapper: Wrapper, ...renderOptions });
}
