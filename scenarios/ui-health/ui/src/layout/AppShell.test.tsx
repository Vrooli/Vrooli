import { beforeEach, afterEach, describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderWithProviders, makeHealthResponse } from "../test-utils";
import { ThemeProvider } from "../theme/ThemeProvider";
import { AppShell } from "./AppShell";
import { MemoryRouter } from "react-router-dom";
import { selectors } from "../consts/selectors";
import { NAV_ITEMS } from "./navItems";

describe("AppShell product configuration", () => {
  beforeEach(() => vi.stubGlobal("matchMedia", (media: string) => ({ media, matches: media.includes("min-width"), addEventListener() {}, removeEventListener() {} })));
  afterEach(() => vi.unstubAllGlobals());
  it("keeps every destination and the existing preference controls in the library shell", () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } });
    client.setQueryData(["health"], makeHealthResponse());
    const { container } = renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><QueryClientProvider client={client}><ThemeProvider><AppShell /></ThemeProvider></QueryClientProvider></MemoryRouter>, { withoutRouter: true });
    expect(container.querySelectorAll("[data-rcl-app-shell]")).toHaveLength(1);
    for (const item of NAV_ITEMS) {
      expect(Array.from(container.querySelectorAll("a")).some(link => link.getAttribute("href") === item.path)).toBe(true);
    }
    expect(screen.getByRole("group", { name: /language|locale/i })).toBeInTheDocument();
    expect(screen.getByTestId(selectors.theme.toggle)).toBeInTheDocument();
    expect(container.querySelector('[data-rcl-app-shell-utility]')).toBeInTheDocument();
  });
});
