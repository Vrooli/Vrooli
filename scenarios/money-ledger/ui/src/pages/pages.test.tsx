import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { AccountsPage } from "./AccountsPage";
import { AdaptersPage } from "./AdaptersPage";
import { StatementsPage } from "./StatementsPage";
import { DashboardPage } from "./DashboardPage";
import { JournalPage } from "./JournalPage";
import { renderWithProviders } from "../test-utils";

const api = vi.hoisted(() => ({
  fetchBooks: vi.fn(),
  fetchAccounts: vi.fn(),
  fetchAdapters: vi.fn(),
  fetchStatement: vi.fn(),
  fetchPosition: vi.fn(),
  fetchPostings: vi.fn(),
  configuredBookId: vi.fn(() => "book-1"),
}));

vi.mock("../api/ledger", () => api);

afterEach(() => {
  cleanup();
  window.history.replaceState({}, "", "/");
});

beforeEach(() => {
  vi.clearAllMocks();
  api.fetchBooks.mockResolvedValue({ books: [{ id: "book-1", name: "Operating" }] });
  api.fetchAccounts.mockResolvedValue({ accounts: [{ id: "account-1", name: "Cash", kind: "ASSET" }] });
  api.fetchAdapters.mockResolvedValue({ adapters: [{ id: "bank", name: "Bank", enabled: true }] });
  api.fetchStatement.mockResolvedValue({ currency: "USD", closingCashMinor: 1200n });
  api.fetchPosition.mockResolvedValue({ runwayMonths: 4.2, runwayAvailable: true, partial: false, availability: [] });
  api.fetchPostings.mockResolvedValue({ postings: [{ id: "posting-1", description: "Sale" }] });
});

function renderFixture(path: string, fixture: string) {
  window.history.replaceState({}, "", `${path}?fixture=${fixture}`);
}

describe("ledger page empty and degraded states", () => {
  it("explains an empty account book and keeps its accounting basis visible", () => {
    renderFixture("/accounts", "empty");
    renderWithProviders(<AccountsPage />);

    expect(screen.getByTestId("page-accounts")).toHaveAttribute("data-experience-state", "empty");
    expect(screen.getByTestId("accounts-empty-guidance")).toBeVisible();
    expect(screen.getByTestId("account-balance-basis")).toBeVisible();
    expect(screen.getByTestId("account-transfer-pair")).toBeVisible();
  });

  it("names an unavailable adapter and its impact [REQ:POS-004]", () => {
    renderFixture("/adapters", "adapter-unavailable");
    renderWithProviders(<AdaptersPage />);

    expect(screen.getByTestId("page-adapters")).toHaveAttribute("data-experience-state", "partial");
    expect(screen.getByTestId("adapter-availability")).toBeVisible();
    expect(screen.getByTestId("adapter-failure-reason")).toBeVisible();
    expect(screen.getByTestId("adapter-missing-impact")).toBeVisible();
    expect(screen.getByTestId("adapter-credential-gap")).toBeVisible();
  });

  it("distinguishes an empty statement period from a populated read", () => {
    renderFixture("/statements", "empty");
    renderWithProviders(<StatementsPage />);

    expect(screen.getByTestId("page-statements")).toHaveAttribute("data-experience-state", "empty");
    expect(screen.getByTestId("statement-empty-guidance")).toBeVisible();
    expect(screen.getByTestId("statement-coverage")).toBeVisible();
    expect(screen.getByTestId("statement-export")).toBeEnabled();
  });

  it("renders live books, position, and statement data", async () => {
    window.history.replaceState({}, "", "/");
    renderWithProviders(<DashboardPage />);
    expect(await screen.findByTestId("position-runway")).toBeVisible();

    window.history.replaceState({}, "", "/accounts");
    cleanup();
    renderWithProviders(<AccountsPage />);
    expect(await screen.findByText(/Cash · ASSET/)).toBeVisible();

    window.history.replaceState({}, "", "/adapters");
    cleanup();
    renderWithProviders(<AdaptersPage />);
    expect(await screen.findByTestId("adapter-availability")).toBeVisible();

    window.history.replaceState({}, "", "/statements");
    cleanup();
    renderWithProviders(<StatementsPage />);
    expect(await screen.findByTestId("statement-figures")).toBeVisible();
  });

  it("exercises every authored dashboard and journal state [REQ:UI-001] [REQ:JRNL-004]", () => {
    for (const state of ["empty", "request-error", "loading", "pending", "partial", "stale", "position-partial", "position-complete", "goal-unmet"]) {
      renderFixture("/", state);
      renderWithProviders(<DashboardPage />);
      expect(screen.getByTestId("page-dashboard")).toHaveAttribute("data-experience-state");
      cleanup();
    }
    for (const state of ["empty", "request-error", "loading", "pending", "partial", "stale", "reversed"]) {
      renderFixture("/journal", state);
      renderWithProviders(<JournalPage />);
      expect(screen.getByTestId("page-journal")).toHaveAttribute("data-experience-state");
      cleanup();
    }
  });
});
