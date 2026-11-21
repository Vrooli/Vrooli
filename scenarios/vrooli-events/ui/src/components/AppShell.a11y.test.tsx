import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { expectNoA11yViolations } from "@vrooli/api-base/testing";
import { renderWithProviders } from "../test-utils";
import { Layout } from "./Layout";

vi.mock("../lib/api", () => ({
  fetchHealth: vi.fn().mockResolvedValue({
    status: "healthy",
    service: "vrooli-events",
    readiness: true,
    subscribers: 0,
    store: { totalEvents: 0, totalPayloadBytes: 0 },
  }),
}));

describe("Vrooli Events application shell accessibility", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders navigation and the main landmark without axe violations", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { container } = renderWithProviders(
      <Routes>
        <Route element={<Layout />} path="*" />
      </Routes>,
      {
        withoutI18n: true,
        withoutRouter: true,
        wrapper: ({ children }) => (
          <QueryClientProvider client={queryClient}>
            <MemoryRouter initialEntries={["/stream"]}>{children}</MemoryRouter>
          </QueryClientProvider>
        ),
      },
    );

    await expectNoA11yViolations(container);
    expect(container.querySelector("main")).toBeInTheDocument();
  });
});
