import { screen } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, describe, expect, it } from "vitest";

import {
  configureTestProviders,
  createTestQueryClient,
  renderWithProviders,
} from "./renderWithProviders";

describe("renderWithProviders", () => {
  afterEach(() => configureTestProviders(undefined));

  it("provides a fresh retry-free query client and default router", () => {
    const { queryClient } = renderWithProviders(<div>ready</div>);

    expect(queryClient).toBeInstanceOf(Object);
    expect(queryClient.getDefaultOptions().queries?.retry).toBe(false);
    expect(screen.getByText("ready")).toBeTruthy();
  });

  it("supports explicit clients, route entries, and the legacy route option", () => {
    const queryClient = createTestQueryClient();
    function RouteProbe(): ReactNode {
      return createElement("output", { "data-testid": "route" }, window.location.pathname);
    }

    renderWithProviders(<RouteProbe />, { queryClient, initialEntries: ["/legacy"] });

    expect(screen.getByTestId("route").textContent).toBe("/legacy");
  });

  it("composes scenario-owned providers around the shared defaults", () => {
    configureTestProviders((children) => (
      <section data-testid="scenario-provider">{children}</section>
    ));

    renderWithProviders(<span>child</span>);

    expect(screen.getByTestId("scenario-provider").contains(screen.getByText("child"))).toBe(true);
  });
});
