/**
 * Routing smoke — for each canonical path (`/`, `/flows`, `/evidence`, `/settings`) the
 * matching page selector is in the document. Page-internal behaviour is
 * exercised in per-page tests; this file's job is to assert the router config.
 */
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { TestAppRouter } from "./routes";

describe("AppRouter", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders the dashboard at /", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    await waitFor(() => expect(screen.getByTestId(selectors.pages.dashboard)).toBeInTheDocument());
  });

  it("renders the flows page at /flows", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/flows"]} />, { withoutRouter: true });
    await waitFor(() => expect(screen.getByText(strings.pages.flows.title)).toBeInTheDocument());
  });

  it("renders the evidence page at /evidence", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/evidence"]} />, { withoutRouter: true });
    await waitFor(() => expect(screen.getByText(strings.pages.evidence.title)).toBeInTheDocument());
  });

  it("renders the settings page at /settings", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    await waitFor(() => expect(screen.getByTestId(selectors.pages.settings)).toBeInTheDocument());
  });
});
