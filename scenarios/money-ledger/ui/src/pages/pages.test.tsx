import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { AccountsPage } from "./AccountsPage";
import { AdaptersPage } from "./AdaptersPage";
import { StatementsPage } from "./StatementsPage";
import { DashboardPage } from "./DashboardPage";
import { JournalPage } from "./JournalPage";
import { SettingsPage } from "./SettingsPage";
import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";

const api = vi.hoisted(() => ({
  fetchBooks: vi.fn(),
  fetchAccounts: vi.fn(),
  fetchAdapters: vi.fn(),
  fetchGoals: vi.fn(),
  fetchStatement: vi.fn(),
  fetchPosition: vi.fn(),
  fetchPostings: vi.fn(),
  createBook: vi.fn(),
  createAccount: vi.fn(),
  transfer: vi.fn(),
  declareGoal: vi.fn(),
  registerAdapter: vi.fn(),
  runAdapter: vi.fn(),
  importFile: vi.fn(),
  registerManualAdapter: vi.fn(),
  fetchOperatorInputStatus: vi.fn(),
  importOperatorInputsJSON: vi.fn(),
  archiveBook: vi.fn(),
  archiveGoal: vi.fn(),
  reparentGoal: vi.fn(),
  ingestEvent: vi.fn(),
  reversePosting: vi.fn(),
  configuredBookId: vi.fn(() => "book-1"),
}));

vi.mock("../api/ledger", () => api);

afterEach(() => {
  cleanup();
  window.history.replaceState({}, "", "/");
});

beforeEach(() => {
  vi.clearAllMocks();
  api.configuredBookId.mockReturnValue("book-1");
  api.fetchBooks.mockResolvedValue({ books: [{ id: "book-1", name: "Operating", currency: "USD" }] });
  api.fetchAccounts.mockResolvedValue({ accounts: [{ id: "account-1", name: "Cash", kind: "ASSET" }] });
  api.fetchAdapters.mockResolvedValue({ adapters: [{ id: "bank", name: "Bank", kind: 2, enabled: true }] });
  api.registerManualAdapter.mockResolvedValue({ adapter: { id: "manual", enabled: true } });
  api.fetchGoals.mockResolvedValue({ goals: [] });
  api.fetchStatement.mockResolvedValue({ currency: "USD", closingCashMinor: 1200n });
  api.fetchPosition.mockResolvedValue({ runwayMonths: 4.2, runwayAvailable: true, partial: false, availability: [] });
  api.fetchPostings.mockResolvedValue({ postings: [{ id: "posting-1", description: "Sale", audit: [] }] });
  api.ingestEvent.mockResolvedValue({ duplicate: false, posting: { id: "posting-created" } });
  api.reversePosting.mockResolvedValue({ posting: { id: "posting-reversed" } });
  api.createBook.mockResolvedValue({ book: { id: "book-created" } });
  api.createAccount.mockResolvedValue({ account: { id: "account-created" } });
  api.transfer.mockResolvedValue({ transferId: "transfer-created" });
  api.declareGoal.mockResolvedValue({ goal: { id: "goal-created" } });
  api.registerAdapter.mockResolvedValue({ adapter: { id: "file-created", enabled: true } });
  api.runAdapter.mockResolvedValue({});
  api.importFile.mockResolvedValue({});
  api.fetchOperatorInputStatus.mockResolvedValue({ fields: [] });
  api.importOperatorInputsJSON.mockResolvedValue({ fields: [{ path: "cash", status: "current", written: true, reason: "", kind: "monetary" }] });
  api.archiveBook.mockResolvedValue({});
  api.archiveGoal.mockResolvedValue({});
  api.reparentGoal.mockResolvedValue({});
});

describe("ledger page empty and degraded states", () => {
  it("exposes every manual-entry field as a real labelled control [REQ:JRNL-004]", async () => {
    window.history.replaceState({}, "", "/journal");
    renderWithProviders(<JournalPage />);

    expect(await screen.findByLabelText(/date/i)).toBeVisible();
    expect(screen.getByLabelText(/pages\.journal\.accountLabel/)).toBeVisible();
    expect(screen.getByLabelText(/signed amount|signedAmountLabel/i)).toBeVisible();
    expect(screen.getByLabelText(/currency/i)).toBeVisible();
    expect(screen.getByLabelText(/description/i)).toBeVisible();
    expect(screen.getByLabelText(/basis/i)).toBeVisible();
  });

  it("submits the ordinary manual adapter and exposes an idempotent duplicate response", async () => {
    const user = userEvent.setup();
    api.ingestEvent.mockResolvedValueOnce({ duplicate: false, posting: { id: "posting-created" } }).mockResolvedValueOnce({ duplicate: true, posting: { id: "posting-created" } });
    window.history.replaceState({}, "", "/journal");
    renderWithProviders(<JournalPage />);

    const accountSelect = await screen.findByLabelText(/pages\.journal\.accountLabel/);
    await waitFor(() => expect(screen.getByRole("option", { name: /Cash/ })).toBeVisible());
    await user.selectOptions(accountSelect, "account-1");
    await user.type(screen.getByLabelText(/signed amount|signedAmountLabel/i), "1250");
    await user.type(screen.getByLabelText(/description/i), "Cash sale");
    await user.click(screen.getByRole("button", { name: /submitEntry/i }));

    await waitFor(() => expect(api.ingestEvent).toHaveBeenCalledWith(expect.objectContaining({ accountId: "account-1", bookId: "book-1", amountMinor: 1250n, description: "Cash sale" })));
    await user.click(screen.getByRole("button", { name: /submitEntry/i }));
    expect(await screen.findByText(/duplicateNotice/i)).toBeVisible();
  });

  it("requires a reason before reversing a posting and sends the confirmed append-only correction", async () => {
    const user = userEvent.setup();
    window.history.replaceState({}, "", "/journal");
    renderWithProviders(<JournalPage />);

    await user.click(await screen.findByTestId("journal-reverse-posting-1"));
    await user.type(screen.getByLabelText(/reversalReasonLabel/i), "Wrong amount");
    await user.click(screen.getByRole("button", { name: /confirmReversal/i }));

    await waitFor(() => expect(api.reversePosting).toHaveBeenCalledWith("posting-1", "Wrong amount"));
  });

  it("renders one dashboard row for every declared goal [REQ:POS-002]", async () => {
    api.fetchGoals.mockResolvedValue({
      goals: [
        { goal: { id: "goal-revenue", name: "Revenue target", thresholdMinor: 1000n, metric: "revenue", comparator: ">=", sustainPeriods: 2, sustainPeriodUnit: 3 }, met: true, sustainedPeriods: 2, requiredPeriods: 2, periodUnit: 3, explanation: "Revenue is sustained." },
        { goal: { id: "goal-reserve", name: "Reserve target", thresholdMinor: 5000n, metric: "cash", comparator: ">=", sustainPeriods: 3, sustainPeriodUnit: 3 }, met: false, sustainedPeriods: 1, requiredPeriods: 3, periodUnit: 3, explanation: "Reserve needs more periods." },
      ],
    });
    window.history.replaceState({}, "", "/");
    renderWithProviders(<DashboardPage />);

    expect(await screen.findByText(/Revenue target/)).toBeVisible();
    expect(screen.getByText(/Reserve target/)).toBeVisible();
  });

  it("explains an empty account book and keeps its accounting basis visible", async () => {
    api.fetchAccounts.mockResolvedValue({ accounts: [] });
    renderWithProviders(<AccountsPage />);

    await waitFor(() => expect(screen.getByTestId("page-accounts")).toHaveAttribute("data-experience-state", "empty"));
    expect(screen.getByTestId("accounts-empty-guidance")).toBeVisible();
    expect(screen.getByTestId("account-balance-basis")).toBeVisible();
    expect(screen.getByTestId("account-transfer-pair")).toBeVisible();
  });

  it("names an unavailable adapter and its impact [REQ:POS-004]", async () => {
    api.fetchAdapters.mockResolvedValue({ adapters: [{ id: "bank", name: "Bank", enabled: false }] });
    renderWithProviders(<AdaptersPage />);

    await waitFor(() => expect(screen.getByTestId("page-adapters")).toHaveAttribute("data-experience-state", "partial"));
    expect(screen.getByTestId("adapter-availability")).toBeVisible();
    expect(screen.getByTestId("adapter-failure-reason")).toBeVisible();
    expect(screen.getByTestId("adapter-missing-impact")).toBeVisible();
    expect(screen.getByTestId("adapter-credential-gap")).toBeVisible();
  });

  it("distinguishes an empty statement period from a populated read", async () => {
    api.configuredBookId.mockReturnValue("");
    api.fetchBooks.mockResolvedValue({ books: [] });
    renderWithProviders(<StatementsPage />);

    await waitFor(() => expect(screen.getByTestId("page-statements")).toHaveAttribute("data-experience-state", "empty"));
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
    expect(await screen.findByText(/Cash/)).toBeVisible();

    window.history.replaceState({}, "", "/adapters");
    cleanup();
    renderWithProviders(<AdaptersPage />);
    expect(await screen.findByTestId("adapter-availability")).toBeVisible();

    window.history.replaceState({}, "", "/statements");
    cleanup();
    renderWithProviders(<StatementsPage />);
    expect(await screen.findByTestId("statement-figures")).toBeVisible();
  });

  it("searches every adopted ledger table across its visible columns", async () => {
    const accountsView = renderWithProviders(<AccountsPage />);
    await screen.findByTestId(selectors.pages.accountTable);
    fireEvent.change(screen.getByPlaceholderText(/pages\.accounts\.accountLabel/), { target: { value: "no matching account" } });

    accountsView.unmount();
    const journalView = renderWithProviders(<JournalPage />);
    await screen.findByTestId(selectors.pages.eventTable);
    fireEvent.change(screen.getByPlaceholderText(/pages\.journal\.postingAccount/), { target: { value: "no matching posting" } });

    journalView.unmount();
    renderWithProviders(<StatementsPage />);
    await screen.findByTestId(selectors.pages.categoryBreakdown);
    fireEvent.change(screen.getByPlaceholderText(/pages\.statements\.categoryBreakdown/), { target: { value: "no matching category" } });
  });

  it("creates books and accounts, then records a valid transfer", async () => {
    const user = userEvent.setup();
    api.fetchBooks.mockResolvedValue({ books: [
      { id: "book-1", name: "Operating", currency: "USD" },
      { id: "book-2", name: "Reserve", currency: "EUR" },
    ] });
    api.fetchAccounts.mockImplementation((bookId: string) => Promise.resolve({ accounts: bookId === "book-2"
      ? [{ id: "account-3", name: "Reserve cash", kind: "ASSET" }, { id: "account-4", name: "Reserve income", kind: "REVENUE" }]
      : [{ id: "account-1", name: "Cash", kind: "ASSET" }, { id: "account-2", name: "Operating income", kind: "REVENUE" }] }));
    renderWithProviders(<AccountsPage />);

    await screen.findByLabelText(/bookNameLabel/i);
    await user.click(screen.getByRole("button", { name: /createBookAction/i }));
    expect(await screen.findByRole("alert")).toBeVisible();

    await user.type(screen.getByLabelText(/bookNameLabel/i), "New book");
    const currencyInputs = screen.getAllByLabelText(/currencyLabel/i);
    await user.clear(currencyInputs[0]!);
    await user.type(currencyInputs[0]!, "gbp");
    await user.click(screen.getByRole("button", { name: /createBookAction/i }));
    await waitFor(() => expect(api.createBook).toHaveBeenCalledWith("New book", "GBP"));

    await user.type(screen.getByLabelText(/accountNameLabel/i), "Receivables");
    await user.click(screen.getByRole("button", { name: /createAccountAction/i }));
		await waitFor(() => expect(api.createAccount).toHaveBeenCalledWith("book-created", "Receivables", "ASSET"));

    await user.selectOptions(screen.getByLabelText(/fromAccountLabel/i), "account-1");
    await user.selectOptions(screen.getByLabelText(/toAccountLabel/i), "account-2");
    await user.type(screen.getByLabelText(/transferAmountLabel/i), "2500");
    await user.click(screen.getByRole("button", { name: /transferAction/i }));
    await waitFor(() => expect(api.transfer).toHaveBeenCalledWith(expect.objectContaining({
      fromAccountId: "account-1",
      toAccountId: "account-2",
      amountMinor: 2500n,
      currency: "USD",
    })));
  });

  it("declares a goal and exercises adapter registration, run, and import paths", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DashboardPage />);
    await user.type(await screen.findByLabelText(/goalNameLabel/i), "Monthly reserve");
    await user.clear(screen.getByLabelText(/goalThresholdLabel/i));
    await user.type(screen.getByLabelText(/goalThresholdLabel/i), "5000");
    await user.click(screen.getByRole("button", { name: /goalCreateAction/i }));
    await waitFor(() => expect(api.declareGoal).toHaveBeenCalledWith(expect.objectContaining({
      bookId: "book-1",
      name: "Monthly reserve",
      thresholdMinor: 5000n,
    })));

    cleanup();
    renderWithProviders(<AdaptersPage />);
    await screen.findByTestId("page-adapters");
    await user.type(screen.getByRole("textbox", { name: /adapterIdLabel/i }), "file-two");
    await user.type(screen.getByLabelText(/adapterNameLabel/i), "Second file");
    await user.click(screen.getByRole("button", { name: /registerAction/i }));
    await waitFor(() => expect(api.registerAdapter).toHaveBeenCalledWith(expect.objectContaining({ id: "file-two", name: "Second file" })));

    await user.click(screen.getByRole("button", { name: /runAction/i }));
    await waitFor(() => expect(api.runAdapter).toHaveBeenCalledWith("bank"));
  });

  it("imports a CSV file through the selected file adapter", async () => {
    const user = userEvent.setup();
    renderWithProviders(<AdaptersPage />);
    await screen.findByTestId("page-adapters");
    const importAdapter = screen.getByRole("combobox", { name: /adapterIdLabel/i });
    await user.selectOptions(importAdapter, "bank");
    expect(importAdapter).toHaveValue("bank");
    const file = new File(["date,amount\n2026-08-16,2500"], "ledger.csv", { type: "text/csv" });
    Object.defineProperty(file, "arrayBuffer", {
      value: () => new TextEncoder().encode("date,amount\n2026-08-16,2500").buffer,
    });
    const fileInput = screen.getByLabelText(/fileLabel/i);
    await user.upload(fileInput, file);
    expect(fileInput).toHaveProperty("files");
    expect((fileInput as HTMLInputElement).files?.[0]).toBe(file);
    fireEvent.submit(fileInput.closest("form")!);
    await waitFor(() => expect(api.importFile).toHaveBeenCalledWith("bank", expect.any(Uint8Array)));
  });

  it("exports a populated statement and renders source availability", async () => {
    const user = userEvent.setup();
    api.fetchStatement.mockResolvedValue({
      currency: "USD",
      from: "2026-08-01",
      to: "2026-08-16",
      openingCashMinor: 1000n,
      inflowMinor: 2500n,
      outflowMinor: 500n,
      closingCashMinor: 3000n,
      revenueMinor: 2500n,
      expenseMinor: 500n,
      assetsMinor: 3000n,
      liabilitiesMinor: 0n,
      availability: [{ adapterId: "bank", reason: "credential expired" }],
    });
    renderWithProviders(<StatementsPage />);
    expect(await screen.findByText(/credential expired/)).toBeVisible();
    await user.click(screen.getByTestId("statement-export"));
    expect(await screen.findByText(/exported/i)).toBeVisible();
  });

  it("exercises archive, reparent, locale/theme, and operator-input review actions", async () => {
    const user = userEvent.setup();
    api.fetchBooks.mockResolvedValue({
      books: [
        { id: "book-1", name: "Operating", currency: "USD" },
        { id: "book-2", name: "Reserve", currency: "USD" },
      ],
    });
    api.fetchGoals.mockResolvedValue({
      goals: [{ goal: { id: "goal-1", name: "Alive", thresholdMinor: 1000n, comparator: ">=", sustainPeriods: 3 } }],
    });

    renderWithProviders(<SettingsPage />);
    await screen.findByTestId("settings-goal-list");
    await user.click(screen.getByTestId("settings-goal-archive"));
    await user.click(screen.getByTestId("settings-goal-reparent"));
    await user.click(screen.getByTestId("page-settings-theme-dark"));
    await user.click(screen.getByTestId("page-settings-locale-ja"));
    await waitFor(() => {
      expect(api.archiveGoal).toHaveBeenCalledWith("goal-1");
      expect(api.reparentGoal).toHaveBeenCalledWith("goal-1", "book-2");
    });

    cleanup();
    api.fetchAccounts.mockImplementation((bookId?: string) =>
      Promise.resolve({ accounts: [{ id: bookId === "book-2" ? "account-2" : "account-1", name: "Cash", kind: "ASSET" }] }),
    );
    renderWithProviders(<AccountsPage />);
    await user.click(await screen.findByTestId("book-archive-control"));
    await waitFor(() => expect(api.archiveBook).toHaveBeenCalledWith("book-1"));

    cleanup();
    renderWithProviders(<AdaptersPage />);
    await screen.findByTestId("operator-input-surface");
    await user.type(screen.getByLabelText(/^Cash/), "125");
    const previewButton = screen.getAllByRole("button").find((button) => /preview import/i.test(button.textContent ?? ""));
    expect(previewButton).toBeDefined();
    await user.click(previewButton!);
    await waitFor(() => expect(api.importOperatorInputsJSON).toHaveBeenCalledWith(expect.objectContaining({ apply: false })));
    const applyButton = screen.getAllByRole("button").find((button) => /apply reviewed import/i.test(button.textContent ?? ""));
    expect(applyButton).toBeDefined();
    await user.click(applyButton!);
    await waitFor(() => expect(api.importOperatorInputsJSON).toHaveBeenCalledWith(expect.objectContaining({ apply: true })));
  });

  it("drives request errors from mocked transport", async () => {
    api.fetchPosition.mockRejectedValue(new Error("ledger unavailable"));
    renderWithProviders(<DashboardPage />);
    await waitFor(() => expect(screen.getByTestId("page-dashboard")).toHaveAttribute("data-experience-state", "request-error"));

    cleanup();
    api.fetchPosition.mockResolvedValue({ runwayMonths: 4.2, runwayAvailable: true, partial: false, availability: [] });
    api.fetchPostings.mockRejectedValue(new Error("journal unavailable"));
    renderWithProviders(<JournalPage />);
    await waitFor(() => expect(screen.getByTestId("page-journal")).toHaveAttribute("data-experience-state", "request-error"));
  });
});
