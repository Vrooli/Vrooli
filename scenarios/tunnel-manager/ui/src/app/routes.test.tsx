/**
 * Routing smoke — for each canonical path the matching page selector is in the
 * document. Page-internal behaviour is exercised in per-feature tests; this
 * file's job is to assert the router config wires every surface.
 */
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { TestAppRouter } from "./routes";

const renderAt = (path: string) =>
  renderWithProviders(<TestAppRouter initialEntries={[path]} />, { withoutRouter: true });

describe("AppRouter", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders the overview surface at /", () => {
    renderAt("/");
    expect(screen.getByTestId(selectors.pages.overview)).toBeInTheDocument();
  });

  it("renders the exposure surface at /exposure", () => {
    renderAt("/exposure");
    expect(screen.getByTestId(selectors.pages.exposure)).toBeInTheDocument();
  });

  it("renders the recovery surface at /recovery", () => {
    renderAt("/recovery");
    expect(screen.getByTestId(selectors.pages.recovery)).toBeInTheDocument();
  });

  it("renders the metrics surface at /metrics", () => {
    renderAt("/metrics");
    expect(screen.getByTestId(selectors.pages.metrics)).toBeInTheDocument();
  });

  it("renders the audit surface at /audit", () => {
    renderAt("/audit");
    expect(screen.getByTestId(selectors.pages.audit)).toBeInTheDocument();
  });

  it("renders the drift surface at /drift", () => {
    renderAt("/drift");
    expect(screen.getByTestId(selectors.pages.drift)).toBeInTheDocument();
  });

  it("renders the settings page at /settings", () => {
    renderAt("/settings");
    expect(screen.getByTestId(selectors.pages.settings)).toBeInTheDocument();
  });
});
