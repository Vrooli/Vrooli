/**
 * Self-test for secrets-manager's local renderWithProviders.
 *
 * This scenario does not use `@vrooli/api-base/testing`; it composes its own
 * QueryClient and an optional MemoryRouter. What is pinned here is that local
 * shape, including one condition that is easy to trip over.
 */
import { describe, expect, it } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { useQuery } from "@tanstack/react-query";
import { useLocation } from "react-router-dom";

import { renderWithProviders } from "./renderWithProviders";

describe("secrets-manager renderWithProviders", () => {
  it("returns the QueryClient the tree used, with retries disabled", async () => {
    const Probe = () => {
      useQuery({ queryKey: ["k"], queryFn: () => Promise.resolve("v") });
      return <span data-testid="probe">ok</span>;
    };

    const { queryClient } = renderWithProviders(<Probe />);
    expect(queryClient.getDefaultOptions().queries?.retry).toBe(false);
    expect(queryClient.getDefaultOptions().mutations?.retry).toBe(false);
    await waitFor(() => expect(queryClient.getQueryData(["k"])).toBe("v"));
  });

  it("omits the QueryClientProvider when asked", () => {
    // With no provider the tree still renders; only query-using components
    // would throw, which is the point of the escape hatch.
    renderWithProviders(<span data-testid="bare">bare</span>, { withoutQueryClient: true });
    expect(screen.getByTestId("bare").textContent).toBe("bare");
  });

  // The router is mounted only when `initialEntries` is supplied — see the
  // `!withoutRouter && initialEntries` condition in the helper. `withoutRouter`
  // alone does not turn it on, so a component that needs routing context must
  // pass entries explicitly. This is a real trap: omitting `initialEntries`
  // gives no router even though nothing opted out of one.
  it("mounts the router only when initialEntries is supplied", () => {
    const Probe = () => {
      const location = useLocation();
      return <output data-testid="route">{location.pathname}</output>;
    };

    renderWithProviders(<Probe />, { initialEntries: ["/vaults"] });
    expect(screen.getByTestId("route").textContent).toBe("/vaults");
  });

  it("renders without a router when initialEntries is omitted", () => {
    // No routing context here, so the probe must not use useLocation.
    renderWithProviders(<span data-testid="norouter">no router</span>);
    expect(screen.getByTestId("norouter").textContent).toBe("no router");
  });
});
