import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { conversationsClient } from "../../api/notifications";
import { selectors } from "../../consts/selectors";
import { renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";
import { TestAppRouter } from "../../app/routes";

vi.mock("../../api/notifications", () => ({
  identityClient: { getSession: vi.fn(() => Promise.resolve({ signedIn: true, subject: "owner-1" })) },
  conversationsClient: { getAsk: vi.fn(), answer: vi.fn(), listAsks: vi.fn() },
  notificationsClient: { get: vi.fn() },
  deliveryClient: { getTimeline: vi.fn(() => Promise.resolve({ notifications: [] })) },
}));

describe("HistoryPage", () => {
  const originalInnerWidth = window.innerWidth;

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
    Object.defineProperty(window, "innerWidth", { configurable: true, value: originalInnerWidth });
  });

  it("renders only returned answered/defaulted/expired decisions and exposes a late-answer default", async () => {
    await setLocale("en");
    vi.mocked(conversationsClient.listAsks).mockResolvedValue({
      asks: [
        { id: "open", question: "Open request", state: "pending", options: [] },
        { id: "done", question: "Closed request", state: "answered", answer: "yes", answerLabel: "Yes", source: "agent-manager", answeredAt: "2026-10-20T14:00:00Z", resolvedAt: "2026-10-20T14:00:00Z", options: [] },
        { id: "default", question: "Defaulted request", state: "defaulted", defaultAnswer: "yes", reversible: true, source: "workflow", defaultAppliedAt: "2026-10-20T14:00:00Z", resolvedAt: "2026-10-20T14:00:00Z", options: [{ key: "yes", label: "Yes" }] },
        { id: "expired", question: "Expired request", state: "expired", source: "scheduler", resolvedAt: "2026-10-20T14:00:00Z", options: [] },
        { id: "withdrawn", question: "Withdrawn request", state: "withdrawn", options: [] },
      ],
    } as never);
    renderWithProviders(<TestAppRouter initialEntries={["/history"]} />, { withoutRouter: true });

    const page = screen.getByTestId(selectors.pages.history);
    await waitFor(() => expect(page).toHaveTextContent("Closed request"));
    expect(page).toHaveTextContent("Defaulted request");
    expect(page).toHaveTextContent("Expired request");
    expect(page).not.toHaveTextContent("Open request");
    expect(page).not.toHaveTextContent("Withdrawn request");
    expect(page).toHaveTextContent("This reversible default can still accept one late answer.");
    expect(page).toHaveTextContent(/cannot be reopened/);
    expect(page).toHaveTextContent("agent-manager");
    expect(page.querySelectorAll("dt")).toHaveLength(6);
    expect(page).toHaveTextContent(/Filters apply only to these loaded results/);
    expect(screen.queryByRole("button", { name: /reopen|answer/i })).not.toBeInTheDocument();
    expect(conversationsClient.listAsks).toHaveBeenCalledWith({ openOnly: false, limit: 200 });
    expect(conversationsClient.answer).not.toHaveBeenCalled();
  });

  it("keeps equal-time returned rows distinct and labels the loaded set without claiming a total", async () => {
    await setLocale("en");
    vi.mocked(conversationsClient.listAsks).mockResolvedValue({
      asks: [
        { id: "a", question: "Same-time A", state: "expired", resolvedAt: "2026-10-20T14:00:00Z", options: [] },
        { id: "b", question: "Same-time B", state: "expired", resolvedAt: "2026-10-20T14:00:00Z", options: [] },
      ],
    } as never);
    renderWithProviders(<TestAppRouter initialEntries={["/history"]} />, { withoutRouter: true });

    const page = screen.getByTestId(selectors.pages.history);
    await waitFor(() => expect(page).toHaveTextContent("Same-time A"));
    expect(page).toHaveTextContent("Same-time B");
    expect(page).toHaveTextContent(/up to 200 returned decisions/);
    expect(screen.queryByText(/total|all decisions/)).not.toBeInTheDocument();
  });

  it("filters only loaded history records by text and clears the local search", async () => {
    await setLocale("en");
    vi.mocked(conversationsClient.listAsks).mockResolvedValue({
      asks: [
        { id: "mobile", question: "Mobile device request", state: "answered", answer: "yes", answerLabel: "Yes", options: [] },
        { id: "desktop", question: "Desktop request", state: "expired", options: [] },
      ],
    } as never);
    const user = userEvent.setup();
    renderWithProviders(<TestAppRouter initialEntries={["/history?q=mobile"]} />, { withoutRouter: true });

    const page = screen.getByTestId(selectors.pages.history);
    const search = screen.getByRole("searchbox", { name: "Filter loaded requests" });
    const outcomeFilters = screen.getByRole("navigation", { name: "History" });
    await waitFor(() => expect(page).toHaveTextContent("Mobile device request"));
    expect(page).not.toHaveTextContent("Desktop request");
    expect(search).toHaveValue("mobile");
    expect(outcomeFilters).toHaveAttribute("data-search-active", "true");
    const clearFooter = screen.getByRole("contentinfo");
    expect(clearFooter).toHaveClass("history-clear-footer", "border-y", "bg-app-surface");
    expect(clearFooter).toContainElement(screen.getByRole("button", { name: "Clear filters" }));

    await user.clear(search);
    await user.type(search, "missing");
    await waitFor(() => expect(page).toHaveTextContent(/Nothing in the latest 200 loaded records matches “missing”\./));
    expect(page).toHaveTextContent(/Older matching records may exist\. This is not a complete account search\./);
    await user.click(screen.getByRole("button", { name: "Clear filters" }));

    await waitFor(() => expect(search).toHaveValue(""));
    expect(outcomeFilters).toHaveAttribute("data-search-active", "false");
    expect(page).toHaveTextContent("Mobile device request");
    expect(page).toHaveTextContent("Desktop request");
    expect(conversationsClient.answer).not.toHaveBeenCalled();
  });

  it.each([
    ["phone", 390],
    ["wide", 1280],
  ])("keeps outcome filters visible, keyboard-operable, and local to loaded results at %s width", async (_viewport, width) => {
    Object.defineProperty(window, "innerWidth", { configurable: true, value: width });
    await setLocale("en");
    vi.mocked(conversationsClient.listAsks).mockResolvedValue({
      asks: [
        { id: "done", question: "Closed request", state: "answered", answer: "yes", answerLabel: "Yes", options: [] },
        { id: "defaulted", question: "Defaulted request", state: "defaulted", defaultAnswer: "later", options: [] },
        { id: "expired", question: "Expired request", state: "expired", options: [] },
      ],
    } as never);
    const user = userEvent.setup();
    renderWithProviders(<TestAppRouter initialEntries={["/history?state=expired"]} />, { withoutRouter: true });

    const page = screen.getByTestId(selectors.pages.history);
    const filters = screen.getByRole("navigation", { name: "History" });
    expect(filters).toHaveClass("flex", "flex-wrap");
    expect(filters).not.toHaveClass("hidden");
    for (const state of ["all", "answered", "defaulted", "expired"] as const) {
      expect(screen.getByTestId(selectors.history.filterLink({ state }))).toHaveClass("min-h-11");
    }
    const expired = screen.getByTestId(selectors.history.filterLink({ state: "expired" }));
    await waitFor(() => expect(page).toHaveTextContent("Expired request"));
    expect(expired).toHaveAttribute("aria-current", "page");
    expect(page).not.toHaveTextContent("Closed request");
    const defaulted = screen.getByTestId(selectors.history.filterLink({ state: "defaulted" }));
    defaulted.focus();
    await user.keyboard("{Enter}");
    await waitFor(() => expect(page).toHaveTextContent("Defaulted request"));
    expect(page).not.toHaveTextContent("Expired request");
    expect(page).not.toHaveTextContent("Closed request");

    await user.click(screen.getByTestId(selectors.history.filterLink({ state: "answered" })));
    await waitFor(() => expect(page).toHaveTextContent("Closed request"));
    expect(page).not.toHaveTextContent("Defaulted request");
    expect(page).not.toHaveTextContent("Expired request");

    await user.click(screen.getByTestId(selectors.history.filterLink({ state: "expired" })));
    await waitFor(() => expect(page).toHaveTextContent("Expired request"));
    expect(page).not.toHaveTextContent("Defaulted request");
    expect(page).not.toHaveTextContent("Closed request");

    await user.click(screen.getByTestId(selectors.history.filterLink({ state: "all" })));
    await waitFor(() => expect(page).toHaveTextContent("Closed request"));
    expect(page).toHaveTextContent("Defaulted request");
    expect(page).toHaveTextContent("Expired request");
  });
});
