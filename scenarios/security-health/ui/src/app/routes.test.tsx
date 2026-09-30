/**
 * Routing smoke — for each canonical path (`/`, `/dependencies`, `/secrets`,
 * `/settings`) the matching page selector is in the document. Page-internal
 * behaviour is exercised in per-page tests; this file's job is to assert the
 * router config.
 */
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { TestAppRouter } from "./routes";

describe("AppRouter", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders the posture page at /", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.posture)).toBeInTheDocument();
  });

  it("renders the dependencies page at /dependencies", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/dependencies"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.dependencies)).toBeInTheDocument();
  });

  it("renders the secrets page at /secrets", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/secrets"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.secrets)).toBeInTheDocument();
  });

  it("renders the settings page at /settings", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.settings)).toBeInTheDocument();
  });
});
