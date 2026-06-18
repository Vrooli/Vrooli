/**
 * Routing smoke — for each canonical path (`/`, `/workspace`, `/library`,
 * `/activity`, `/models`, `/settings`) the matching page selector is in the
 * document. Page-internal behaviour is exercised in per-page tests; this file's
 * job is to assert the router config.
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

  it("renders the home page at /", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.home)).toBeInTheDocument();
  });

  it("renders the workspace page at /workspace", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/workspace"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.workspace)).toBeInTheDocument();
  });

  it("renders the library page at /library", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/library"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.library)).toBeInTheDocument();
  });

  it("renders the activity page at /activity", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/activity"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.activity)).toBeInTheDocument();
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
