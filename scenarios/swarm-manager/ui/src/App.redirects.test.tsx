/**
 * Route-retirement contract: /plan is the first-class board route, the legacy
 * graph paths and the retired Operations Center / Command Post / list-page URLs
 * redirect to it instead of 404ing, deep-link query state (drawer, filters) is
 * preserved, and legacy graph lens URLs redirect into /graph query state.
 */

import { QueryClientProvider } from "@tanstack/react-query";
import { render, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import App from "./App";
import { createTestQueryClient } from "./test-utils";
import { DEFAULT_SETTINGS } from "./services/settings-service";

// Exercise routing and the real shared shell without starting the unrelated
// graph, voice capture, or settings-editor surfaces.
vi.mock("./surfaces/graph/components/Sidebar", () => ({ Sidebar: () => null }));
vi.mock("./surfaces/graph/components/CapturePanel", () => ({ CapturePanel: () => null }));
vi.mock("./surfaces/graph/components/SettingsDrawer", () => ({ SettingsDrawer: () => null }));
vi.mock("./surfaces/graph/components/GraphWorkspace", () => ({ GraphWorkspace: () => null }));
vi.mock("./pages/BacklogDetailsPage", () => ({ BacklogDetailsPage: () => null }));

function renderAt(path: string, theme?: "dark" | "light") {
  window.history.pushState({}, "", path);
  const client = createTestQueryClient();
  if (theme) client.setQueryData(["settings"], { ...DEFAULT_SETTINGS, theme });
  return render(
    <QueryClientProvider client={client}>
      <App />
    </QueryClientProvider>,
  );
}

describe("retired-route redirects", () => {
  afterEach(() => {
    window.history.pushState({}, "", "/");
    delete document.documentElement.dataset.theme;
    delete document.documentElement.dataset.resolvedTheme;
    document.documentElement.style.colorScheme = "";
  });

  it.each(["dark", "light"] as const)("applies the saved %s theme on a direct backlog review route", async (theme) => {
    renderAt("/backlog/execute/approval-review?tab=prompt", theme);

    await waitFor(() => {
      expect(document.documentElement.dataset.resolvedTheme).toBe(theme);
      expect(document.documentElement.style.colorScheme).toBe(theme);
    });
    expect(window.location.pathname).toBe("/backlog/execute/approval-review");
  });

  it("root redirects to the first-class Plan route", async () => {
    renderAt("/");
    await waitFor(() => {
      expect(window.location.pathname).toBe("/plan");
    });
  });

  it("legacy /graph/plan redirects to /plan preserving query state", async () => {
    renderAt("/graph/plan?drawer=decisions");
    await waitFor(() => {
      expect(window.location.pathname).toBe("/plan");
      expect(window.location.search).toContain("drawer=decisions");
    });
  });

  it("bare /graph resolves as the single Graph surface", async () => {
    renderAt("/graph");
    await waitFor(() => {
      expect(window.location.pathname).toBe("/graph");
    });
  });

  it("legacy /graph/focus redirects to /graph focus mode preserving query state", async () => {
    renderAt("/graph/focus?select=backlog-item%2Ffix%2Fbug");
    await waitFor(() => {
      expect(window.location.pathname).toBe("/graph");
      expect(window.location.search).toContain("mode=focus");
      expect(window.location.search).toContain("select=backlog-item%2Ffix%2Fbug");
    });
  });

  it("legacy /graph/topology redirects to /graph preserving query state", async () => {
    renderAt("/graph/topology");
    await waitFor(() => {
      expect(window.location.pathname).toBe("/graph");
      expect(window.location.search).not.toContain("mode=");
    });
  });

  it("/command-post redirects to the Plan route", async () => {
    renderAt("/command-post");
    await waitFor(() => {
      expect(window.location.pathname).toBe("/plan");
    });
  });

  it("/command-post/decisions redirects with the decisions drawer open", async () => {
    renderAt("/command-post/decisions");
    await waitFor(() => {
      expect(window.location.pathname).toBe("/plan");
      expect(window.location.search).toContain("drawer=decisions");
    });
  });

  it("/operations redirects preserving filter params", async () => {
    renderAt("/operations?status=running&lane=execute");
    await waitFor(() => {
      expect(window.location.pathname).toBe("/plan");
      expect(window.location.search).toContain("status=running");
      expect(window.location.search).toContain("lane=execute");
    });
  });

  it("retired /executions list page redirects to /plan (detail routes unaffected)", async () => {
    renderAt("/executions");
    await waitFor(() => {
      expect(window.location.pathname).toBe("/plan");
    });
  });

  it("retired /scenarios list page redirects to /plan (detail routes unaffected)", async () => {
    renderAt("/scenarios");
    await waitFor(() => {
      expect(window.location.pathname).toBe("/plan");
    });
  });
});
