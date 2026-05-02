import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

vi.mock("./lib/api", () => ({
  fetchHealth: vi.fn().mockResolvedValue({
    status: "ok",
    service: "test-service",
    timestamp: "2026-05-01T00:00:00Z",
  }),
}));

import App from "./App";
import { setLocale } from "./i18n";
import en from "./i18n/locales/en.json";
import ja from "./i18n/locales/ja.json";

const renderApp = () => {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <App />
    </QueryClientProvider>,
  );
};

describe("App locale wiring", () => {
  beforeEach(async () => {
    window.localStorage.clear();
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
  });

  it("renders English copy by default and reflects it on <html>", async () => {
    renderApp();
    expect(await screen.findByText(en.app.eyebrow)).toBeInTheDocument();
    expect(screen.getByText(en.app.description)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: en.health.refresh }),
    ).toBeInTheDocument();
    expect(document.documentElement.lang).toBe("en");
    expect(document.documentElement.dir).toBe("ltr");
  });

  it("switches to Japanese when the 日本語 toggle is clicked", async () => {
    const user = userEvent.setup();
    renderApp();
    await user.click(screen.getByRole("button", { name: "日本語" }));

    await waitFor(() => {
      expect(screen.getByText(ja.app.eyebrow)).toBeInTheDocument();
    });
    expect(screen.getByText(ja.app.description)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: ja.health.refresh }),
    ).toBeInTheDocument();
    expect(document.documentElement.lang).toBe("ja");
  });

  it("persists the chosen locale to localStorage so returning visits restore it", async () => {
    const user = userEvent.setup();
    renderApp();
    await user.click(screen.getByRole("button", { name: "日本語" }));

    await waitFor(() => {
      expect(window.localStorage.getItem("vrooli.locale")).toBe("ja");
    });
  });

  it("marks the active locale's toggle as pressed", async () => {
    const user = userEvent.setup();
    renderApp();

    expect(screen.getByRole("button", { name: en.locale.english })).toHaveAttribute(
      "aria-pressed",
      "true",
    );

    await user.click(screen.getByRole("button", { name: en.locale.japanese }));

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: ja.locale.japanese }),
      ).toHaveAttribute("aria-pressed", "true");
    });
  });
});
