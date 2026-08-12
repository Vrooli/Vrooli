import { jsx as _jsx } from "react/jsx-runtime";
import { render } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { I18nextProvider } from "react-i18next";
import { i18n } from "../i18n";
export function createTestQueryClient() {
    return new QueryClient({
        defaultOptions: {
            queries: { retry: false },
            mutations: { retry: false },
        },
    });
}
export function renderWithProviders(ui, { queryClient = createTestQueryClient(), ...options } = {}) {
    function Wrapper({ children }) {
        return (_jsx(QueryClientProvider, { client: queryClient, children: _jsx(I18nextProvider, { i18n: i18n, children: children }) }));
    }
    return render(ui, { wrapper: Wrapper, ...options });
}
