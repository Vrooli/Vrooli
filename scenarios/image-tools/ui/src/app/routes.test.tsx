/**
 * Routing smoke — for each canonical path (`/`, `/editor`, `/jobs`,
 * `/models`, `/settings`) the matching page selector is in the document.
 * Page-internal
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

  it("renders the dashboard at /", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.dashboard)).toBeInTheDocument();
  });

  it("renders the editor page at /editor", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/editor"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.editor)).toBeInTheDocument();
  });

  it("renders the jobs page at /jobs", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/jobs"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.jobs)).toBeInTheDocument();
  });

  it("renders the models page at /models", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/models"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.models)).toBeInTheDocument();
  });

  it("renders the settings page at /settings", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.settings)).toBeInTheDocument();
  });
});
