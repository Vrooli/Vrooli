import en from "../../i18n/locales/en.json";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { conversationsClient, identityClient } from "../../api/notifications";
import { selectors } from "../../consts/selectors";
import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";
import { TestAppRouter } from "../../app/routes";

vi.mock("../../api/notifications", () => ({
  identityClient: { getSession: vi.fn(() => Promise.resolve({ signedIn: true, subject: "owner-1" })) },
  conversationsClient: { getAsk: vi.fn(), answer: vi.fn(), listAsks: vi.fn() },
}));
const listAsks = vi.mocked(conversationsClient.listAsks);
const open = { id: "ask-1", question: "Review the remaining delivery gap", state: "pending", options: [], source: "agent-manager" };
const show = (path = "/asks") => renderWithProviders(<TestAppRouter initialEntries={[path]} />, { withoutRouter: true });

describe("Needs you complete search", () => {
  beforeEach(async () => { vi.resetAllMocks(); vi.mocked(identityClient.getSession).mockResolvedValue({ signedIn: true, subject: "owner-1" } as never); Object.defineProperty(HTMLElement.prototype, "scrollTo", { configurable: true, value: vi.fn() }); await setLocale("en"); });

  it("searches pending and escalated requests and offers cross-state search", async () => {
    listAsks.mockResolvedValue({ asks: [open], nextCursor: "" } as never);
    show("/asks?q=delivery&vrooli_test_mode=1");
    const row = await screen.findByTestId(selectors.asks.item);
    expect(row.querySelector("a")).toHaveAttribute("href", "/asks/ask-1?vrooli_test_mode=1");
    expect(listAsks).toHaveBeenCalledWith(expect.objectContaining({ query: "delivery", stateFilter: "awaiting", cursor: "", limit: 50 }), expect.anything());
    expect(screen.getByRole("link", { name: "Search all request statuses" })).toHaveAttribute("href", "/history?q=delivery&vrooli_test_mode=1");
    expect(conversationsClient.answer).not.toHaveBeenCalled();
  });

  it("shows a checked empty queue without hiding search or imposing a result cap", async () => {
    listAsks.mockResolvedValue({ asks: [], nextCursor: "" } as never);
    const { container } = show();
    await screen.findByRole("heading", { name: "Nothing needs an answer" });
    expect(screen.getByRole("searchbox", { name: "Search requests" })).toBeInTheDocument();
    expect(screen.queryByText(/Latest 200|older requests may exist/i)).not.toBeInTheDocument();
    await expectNoA11yViolations(container);
  });

  it("finds unanswered matches through server search rather than the loaded page", async () => {
    listAsks.mockImplementation((request) => Promise.resolve({ asks: request.query === "older" ? [{ ...open, question: "Older escalated request", state: "escalated" }] : [], nextCursor: "" } as never));
    const user = userEvent.setup(); show();
    await screen.findByText(en.pages.history.noAwaiting);
    await user.type(screen.getByRole("searchbox"), "older");
    await screen.findByText(new RegExp("Older\\ escalated\\ request"));
    expect(listAsks).toHaveBeenLastCalledWith(expect.objectContaining({ query: "older", stateFilter: "awaiting" }), expect.anything());
  });

  it("distinguishes an unread queue from an empty queue and supports read-only retry", async () => {
    listAsks.mockRejectedValueOnce(new Error("offline")).mockResolvedValueOnce({ asks: [open], nextCursor: "" } as never);
    const user = userEvent.setup(); show("/asks?vrooli_test_mode=1");
    await screen.findByRole("alert");
    expect(screen.queryByText(en.pages.history.noAwaiting)).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open settings" })).toHaveAttribute("href", "/settings?vrooli_test_mode=1");
    await user.click(screen.getByRole("button", { name: "Try again" }));
    await screen.findByText(open.question);
    expect(conversationsClient.answer).not.toHaveBeenCalled();
  });

  it("continues the actionable queue past a page and retains its context on return", async () => {
    listAsks.mockImplementation((request) => Promise.resolve({ asks: [{ ...open, id: request.cursor ? "second" : "first", question: request.cursor ? "Second request" : "First request" }], nextCursor: request.cursor ? "" : "next" } as never));
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: { ...open, id: "second", question: "Second request" } } as never);
    const user = userEvent.setup(); show("/asks?q=review");
    await screen.findByText(new RegExp("First\\ request"));
    await user.click(screen.getByRole("button", { name: "Load more" }));
    await user.click(await screen.findByRole("link", { name: /Second request/ }));
    await screen.findByTestId(selectors.asks.question);
    await user.click(screen.getByRole("link", { name: "All decisions" }));
    const restored = await screen.findByRole("link", { name: /Second request/ });
    await waitFor(() => expect(restored).toHaveFocus());
    expect(screen.getByRole("searchbox")).toHaveValue("review");
    expect(screen.getByText(new RegExp("First\\ request"))).toBeInTheDocument();
  });
});
