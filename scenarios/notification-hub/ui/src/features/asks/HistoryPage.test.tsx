import en from "../../i18n/locales/en.json";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import { AskSchema, type Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { conversationsClient, identityClient } from "../../api/notifications";
import { selectors } from "../../consts/selectors";
import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";
import { TestAppRouter } from "../../app/routes";

vi.mock("../../api/notifications", () => ({
  identityClient: { getSession: vi.fn(() => Promise.resolve({ signedIn: true, subject: "owner-1" })) },
  conversationsClient: { getAsk: vi.fn(), answer: vi.fn(), listAsks: vi.fn() },
  notificationsClient: { get: vi.fn() },
  deliveryClient: { getTimeline: vi.fn(() => Promise.resolve({ notifications: [] })) },
}));
const ask = (id: string, state = "answered", extra: Partial<Omit<Ask, "$typeName" | "$unknown">> = {}) => create(AskSchema, {
  id, question: `Request ${id}`, state, source: "agent-manager", createdAt: "2026-10-20T14:00:00Z", ...extra,
});
const listAsks = vi.mocked(conversationsClient.listAsks);
const show = (path = "/history") => renderWithProviders(<TestAppRouter initialEntries={[path]} />, { withoutRouter: true });

describe("History request browser", () => {
  beforeEach(async () => { vi.resetAllMocks(); vi.mocked(identityClient.getSession).mockResolvedValue({ signedIn: true, subject: "owner-1" } as never); Object.defineProperty(HTMLElement.prototype, "scrollTo", { configurable: true, value: vi.fn() }); await setLocale("en"); });
  afterEach(() => vi.clearAllMocks());

  it("includes outstanding and resolved lifecycle states without a fixed-inventory warning", async () => {
    listAsks.mockResolvedValue({ asks: [ask("open", "pending"), ask("escalated", "escalated"), ask("done"), ask("expired", "expired"), ask("withdrawn", "withdrawn")], nextCursor: "" } as never);
    show();
    await screen.findByText(ask("open").question);
    expect(screen.getByText(ask("withdrawn").question)).toBeInTheDocument();
    expect(screen.getAllByText(en.pages.history.awaiting)).toHaveLength(3);
    expect(screen.getByRole("link", { name: "Awaiting answer" })).toHaveAttribute("href", "/history?state=awaiting");
    expect(screen.queryByText(/latest 200|older records are not available/i)).not.toBeInTheDocument();
    expect(listAsks).toHaveBeenCalledWith({ limit: 50, cursor: "", query: "", stateFilter: "all" }, expect.objectContaining({ signal: expect.any(AbortSignal) }));
    expect(conversationsClient.answer).not.toHaveBeenCalled();
  });

  it("sends search to the server and preserves it while changing lifecycle filters", async () => {
    listAsks.mockImplementation((request) => Promise.resolve({ asks: request.query === "old" ? [ask("older-match", request.stateFilter === "awaiting" ? "pending" : "answered")] : [ask("recent")], nextCursor: "" } as never));
    const user = userEvent.setup();
    show("/history?vrooli_test_mode=1");
    await screen.findByText(ask("recent").question);
    await user.type(screen.getByRole("searchbox", { name: "Search requests" }), "old");
    await screen.findByText(ask("older-match").question);
    expect(screen.queryByText(ask("recent").question)).not.toBeInTheDocument();
    await user.click(screen.getByRole("link", { name: "Awaiting answer" }));
    await waitFor(() => expect(listAsks).toHaveBeenLastCalledWith(expect.objectContaining({ query: "old", stateFilter: "awaiting", cursor: "" }), expect.anything()));
    expect(screen.getByRole("searchbox")).toHaveValue("old");
    expect(screen.getByRole("link", { name: /Request older-match/ })).toHaveAttribute("href", "/asks/older-match?vrooli_test_mode=1");
    await user.click(screen.getByRole("button", { name: "Clear filters" }));
    await screen.findByText(ask("recent").question);
    expect(screen.getByRole("searchbox")).toHaveFocus();
    expect(screen.getByRole("link", { name: "All" })).toHaveAttribute("aria-current", "page");
  });

  it("keeps loaded rows when continuation fails and retries the same cursor", async () => {
    listAsks.mockResolvedValueOnce({ asks: [ask("first")], nextCursor: "next-page" } as never)
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce({ asks: [ask("second")], nextCursor: "" } as never);
    const user = userEvent.setup(); show();
    await screen.findByText(ask("first").question);
    await user.click(screen.getByRole("button", { name: "Load more" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Your loaded results are still here");
    expect(screen.getByText(ask("first").question)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Try loading more again" }));
    await screen.findByText(ask("second").question);
    expect(screen.getByText(ask("first").question)).toBeInTheDocument();
    expect(listAsks).toHaveBeenNthCalledWith(2, expect.objectContaining({ cursor: "next-page" }), expect.anything());
    expect(listAsks).toHaveBeenNthCalledWith(3, expect.objectContaining({ cursor: "next-page" }), expect.anything());
    expect(screen.queryByRole("button", { name: "Load more" })).not.toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("All 2 matching requests shown");
  });

  it("never announces an empty inventory while the initial search is pending or failed", async () => {
    listAsks.mockRejectedValueOnce(new Error("offline")).mockResolvedValueOnce({ asks: [], nextCursor: "" } as never);
    const user = userEvent.setup(); show();
    expect(screen.getByRole("status")).toHaveTextContent("Searching requests");
    expect(await screen.findByRole("alert")).toHaveTextContent("History could not be loaded");
    expect(screen.queryByText(en.pages.history.empty)).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Try again" }));
    await screen.findByRole("heading", { name: "No requests yet" });
    expect(screen.getByRole("searchbox")).toHaveFocus();
  });

  it("keeps the page title and reports a genuine server-query no-match", async () => {
    listAsks.mockResolvedValue({ asks: [], nextCursor: "" } as never);
    show("/history?q=missing&state=awaiting");
    await screen.findByRole("heading", { name: "No matching requests" });
    expect(screen.getByRole("heading", { name: "History" })).toBeInTheDocument();
    expect(listAsks).toHaveBeenCalledWith(expect.objectContaining({ query: "missing", stateFilter: "awaiting" }), expect.anything());
    expect(screen.queryByText(/older matching records may exist/i)).not.toBeInTheDocument();
  });

  it("preserves late-answer recovery and full request text while composing a concise summary", async () => {
    listAsks.mockResolvedValue({ asks: [ask("default", "defaulted", { title: "Review deletion request", question: "Delete /home/owner/important-folder after review?", defaultAnswer: "no", reversible: true }), ask("answered", "answered", { answer: "no", answerLabel: "Keep the files", answeredAt: "2026-10-20T15:30:00Z" })], nextCursor: "" } as never);
    const { container } = show();
    await screen.findByText(new RegExp("Review\\ deletion\\ request"));
    expect(screen.getByText(new RegExp("Delete\\ /home/owner/important\\-folder\\ after\\ review\\?"))).toBeInTheDocument();
    expect(screen.getByText(en.pages.history.lateAnswerAvailable)).toBeInTheDocument();
    expect(screen.getByText(/Answer: Keep the files/)).toBeInTheDocument();
    expect(container.querySelector('time[datetime="2026-10-20T15:30:00Z"]')).toHaveTextContent(/2026/);
    await expectNoA11yViolations(container);
  });

  it("returns from a second-page request with search, pages, focus and scroll retained", async () => {
    listAsks.mockImplementation((request) => Promise.resolve({ asks: [ask(request.cursor ? "second" : "first")], nextCursor: request.cursor ? "" : "next" } as never));
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: ask("second") } as never);
    const user = userEvent.setup(); show("/history?q=retained&state=answered");
    await screen.findByText(ask("first").question);
    await user.click(screen.getByRole("button", { name: "Load more" }));
    const row = await screen.findByRole("link", { name: /Request second/ });
    const main = screen.getByRole("main");
    Object.defineProperty(main, "scrollTop", { configurable: true, value: 417 });
    const scrollTo = vi.fn(); Object.defineProperty(main, "scrollTo", { configurable: true, value: scrollTo });
    await user.click(row);
    await screen.findByTestId(selectors.asks.question);
    await user.click(screen.getByRole("link", { name: "Back to history" }));
    const restored = await screen.findByRole("link", { name: /Request second/ });
    await waitFor(() => expect(restored).toHaveFocus());
    expect(scrollTo).toHaveBeenCalledWith({ top: 417 });
    expect(screen.getByRole("searchbox")).toHaveValue("retained");
    expect(screen.getByRole("link", { name: "Answered" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByText(ask("first").question)).toBeInTheDocument();
  });

  it("retains stale results on a refresh failure and retries only the read", async () => {
    listAsks.mockResolvedValueOnce({ asks: [ask("saved")], nextCursor: "" } as never).mockRejectedValueOnce(new Error("offline")).mockResolvedValueOnce({ asks: [ask("fresh")], nextCursor: "" } as never);
    const user = userEvent.setup(); const { queryClient } = show();
    await screen.findByText(ask("saved").question);
    await queryClient.invalidateQueries({ queryKey: ["asks", "history"] });
    expect(await screen.findByRole("alert")).toHaveTextContent("History could not be loaded");
    expect(screen.getByText(ask("saved").question)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Try again" }));
    await screen.findByText(ask("fresh").question);
    expect(conversationsClient.answer).not.toHaveBeenCalled();
  });
});
