/**
 * Route-retirement contract: the old Operations Center and Command Post
 * URLs redirect to the Plan board instead of 404ing, and the decisions
 * deep link carries the drawer parameter. Old /operations filter links
 * keep their query string (same param names on the board).
 */

import { QueryClientProvider } from "@tanstack/react-query";
import { render, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import App from "./App";
import { createTestQueryClient } from "./test-utils";

function renderAt(path: string) {
  window.history.pushState({}, "", path);
  return render(
    <QueryClientProvider client={createTestQueryClient()}>
      <App />
    </QueryClientProvider>,
  );
}

describe("retired-route redirects", () => {
  afterEach(() => {
    window.history.pushState({}, "", "/");
  });

  it("root redirects to the Plan lens", async () => {
    renderAt("/");
    await waitFor(() => {
      expect(window.location.pathname).toBe("/graph/plan");
    });
  });

  it("/command-post redirects to the Plan lens", async () => {
    renderAt("/command-post");
    await waitFor(() => {
      expect(window.location.pathname).toBe("/graph/plan");
    });
  });

  it("/command-post/decisions redirects with the decisions drawer open", async () => {
    renderAt("/command-post/decisions");
    await waitFor(() => {
      expect(window.location.pathname).toBe("/graph/plan");
      expect(window.location.search).toContain("drawer=decisions");
    });
  });

  it("/operations redirects preserving filter params", async () => {
    renderAt("/operations?status=running&lane=execute");
    await waitFor(() => {
      expect(window.location.pathname).toBe("/graph/plan");
      expect(window.location.search).toContain("status=running");
      expect(window.location.search).toContain("lane=execute");
    });
  });
});
