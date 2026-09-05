/**
 * Routing smoke — for each canonical path the matching page selector is in the
 * document. Page-internal behaviour is exercised in per-page tests; this
 * file's job is to assert the router config. Add one case per route you add.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { TestAppRouter } from "./routes";

vi.mock("../api/tokenEconomy", () => ({
  minterClient: {
    listTokenTypes: vi.fn().mockResolvedValue({ tokenTypes: [] }),
    listHolders: vi.fn().mockResolvedValue({ holders: [] }),
    listCatalogEntries: vi.fn().mockResolvedValue({ entries: [] }),
    listPendingRedemptions: vi.fn().mockResolvedValue({ redemptions: [] }),
    listGrants: vi.fn().mockResolvedValue({ grants: [] }),
    listJournalEvents: vi.fn().mockResolvedValue({ events: [] }),
  },
  earningClient: { listEarnings: vi.fn().mockResolvedValue({ submissions: [] }) },
  holderClient: {
    viewEconomy: vi.fn().mockResolvedValue({ holder: undefined, balances: [], events: [], redemptions: [] }),
    browseCatalog: vi.fn().mockResolvedValue({ entries: [] }),
  },
}));

describe("AppRouter", () => {
  afterEach(() => {
    cleanup();
  });

  it.each([
    ["/", selectors.pages.dashboard],
    ["/tokens", selectors.pages.tokens],
    ["/holders", selectors.pages.holders],
    ["/earning", selectors.pages.earning],
    ["/grants", selectors.pages.grants],
    ["/catalog", selectors.pages.catalog],
    ["/approvals", selectors.pages.approvals],
    ["/journal", selectors.pages.journal],
    ["/settings", selectors.pages.settings],
    ["/me", "page-holder-home"],
    ["/me/history", "page-holder-history"],
    ["/me/rewards", "page-holder-rewards"],
  ])("renders %s", (path, selector) => {
    renderWithProviders(<TestAppRouter initialEntries={[path]} />, { withoutRouter: true });
    expect(screen.getByTestId(selector)).toBeInTheDocument();
  });
});
