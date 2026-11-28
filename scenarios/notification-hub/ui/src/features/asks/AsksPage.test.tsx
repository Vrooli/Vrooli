import { en } from "../../i18n/catalog";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ListAsksRequest_Order } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";
import { conversationsClient, identityClient } from "../../api/notifications";
import { selectors } from "../../consts/selectors";
import { asksInOrder, expectNoA11yViolations, listAsksFor, renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";
import { TestAppRouter } from "../../app/routes";
import { makeAsk } from "../packet/mocks/askFactory";

vi.mock("../../api/notifications", () => ({
  ROUTED_TEST_LEASE_EXPIRED_EVENT: "notification-hub:routed-test-lease-expired",
  identityClient: { getSession: vi.fn(() => Promise.resolve({ signedIn: true, subject: "owner-1" })) },
  conversationsClient: { getAsk: vi.fn(), answer: vi.fn(), listAsks: vi.fn(), listNativeConsents: vi.fn() },
}));
const listAsks = vi.mocked(conversationsClient.listAsks);
const open = makeAsk({ question: "Review the remaining delivery gap", options: [], source: "agent-manager" });
const show = (path = "/asks", state?: unknown) => renderWithProviders(<TestAppRouter initialEntries={[{ pathname: path.split("?")[0] ?? "/asks", search: path.includes("?") ? `?${path.split("?")[1]}` : "", state }]} />, { withoutRouter: true });

describe("Needs you complete search", () => {
  beforeEach(async () => { vi.resetAllMocks(); vi.mocked(conversationsClient.listNativeConsents).mockResolvedValue({ consents: [], nextCursor: "" } as never); vi.mocked(identityClient.getSession).mockResolvedValue({ signedIn: true, subject: "owner-1" } as never); Object.defineProperty(HTMLElement.prototype, "scrollTo", { configurable: true, value: vi.fn() }); await setLocale("en"); });

  it("searches pending and escalated requests and offers cross-state search", async () => {
    listAsks.mockImplementation(listAsksFor(() => ({ asks: [open], nextCursor: "" })));
    show("/asks?q=delivery&vrooli_test_mode=1");
    const row = await screen.findByTestId(selectors.asks.item);
    expect(row.querySelector("a")).toHaveAttribute("href", "/asks/ask-1?vrooli_test_mode=1");
    expect(listAsks).toHaveBeenCalledWith(expect.objectContaining({ query: "delivery", stateFilter: "awaiting", cursor: "", limit: 50, order: ListAsksRequest_Order.ATTENTION }), expect.anything());
    expect(screen.getByRole("link", { name: "Search every status" })).toHaveAttribute("href", "/history?q=delivery&vrooli_test_mode=1");
    expect(conversationsClient.answer).not.toHaveBeenCalled();
  });

  it("shows a checked empty queue without hiding search or imposing a result cap", async () => {
    listAsks.mockImplementation(listAsksFor(() => ({ asks: [], nextCursor: "" })));
    const { container } = show();
    await screen.findByRole("heading", { name: "All caught up" });
    expect(screen.getByRole("button", { name: "Search decisions" })).toBeInTheDocument();
    expect(screen.queryByText(/Latest 200|older requests may exist/i)).not.toBeInTheDocument();
    await expectNoA11yViolations(container);
  });

  it("finds unanswered matches through server search rather than the loaded page", async () => {
    listAsks.mockImplementation(listAsksFor((request) => ({ asks: request.query === "older" ? [makeAsk({ ...open, question: "Older escalated request", state: "escalated" })] : [], nextCursor: "" })));
    const user = userEvent.setup(); show();
    await screen.findByRole("heading", { name: en.pages.history.noAwaiting });
    await user.click(screen.getByRole("button", { name: "Search decisions" }));
    await user.type(screen.getByRole("searchbox"), "older");
    await screen.findByText(new RegExp("Older\\ escalated\\ request"));
    expect(listAsks).toHaveBeenLastCalledWith(expect.objectContaining({ query: "older", stateFilter: "awaiting" }), expect.anything());
  });

  it("distinguishes an unread queue from an empty queue and supports read-only retry", async () => {
    listAsks.mockImplementation(asksInOrder(new Error("offline"), { asks: [open], nextCursor: "" }));
    const user = userEvent.setup(); show("/asks?vrooli_test_mode=1");
    await screen.findByRole("alert");
    expect(screen.queryByText(en.pages.history.noAwaiting)).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Try again" }));
    await screen.findByText(open.question);
    expect(conversationsClient.answer).not.toHaveBeenCalled();
  });

  it("continues the actionable queue past a page and retains its context on return", async () => {
    listAsks.mockImplementation(listAsksFor((request) => ({ asks: [makeAsk({ ...open, id: request.cursor ? "second" : "first", question: request.cursor ? "Second request" : "First request" })], nextCursor: request.cursor ? "" : "next" })));
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: makeAsk({ ...open, id: "second", question: "Second request" }) } as never);
    const user = userEvent.setup(); show("/asks?q=review");
    await screen.findByText(new RegExp("First\\ request"));
    await user.click(screen.getByRole("button", { name: "Load more" }));
    await user.click(await screen.findByRole("link", { name: /Second request/ }));
    await screen.findByTestId(selectors.asks.question);
    // The tab bar has a link of the same name; the way back is the one inside the page.
    await user.click(within(screen.getByRole("main")).getByRole("link", { name: en.layout.nav.asks }));
    const restored = await screen.findByRole("link", { name: /Second request/ });
    await waitFor(() => expect(restored).toHaveFocus());
    expect(screen.getByRole("searchbox")).toHaveValue("review");
    expect(screen.getByText(new RegExp("First\\ request"))).toBeInTheDocument();
  });

  it("puts focus back on the decision the owner just left when the decision page asks for it", async () => {
    listAsks.mockImplementation(listAsksFor(() => ({ asks: [makeAsk({ ...open, id: "first", question: "First request" }), makeAsk({ ...open, id: "second", question: "Second request" })], nextCursor: "" })));
    show("/asks", { restoreAskId: "second" });
    const restored = await screen.findByRole("link", { name: /Second request/ });
    await waitFor(() => expect(restored).toHaveFocus());
    expect(screen.getByRole("link", { name: /First request/ })).not.toHaveFocus();
  });

  describe("when a read fails", () => {
    const listNativeConsents = () => vi.mocked(conversationsClient.listNativeConsents);
    // The shell's own count also reads permission requests, without a cursor; only the page's read has one.
    const pageReads = () => listNativeConsents().mock.calls.filter(([request]) => request.cursor !== undefined).length;

    it("says so once, naming both, when decisions and permission requests both fail, and one retry reads both again", async () => {
      listAsks.mockRejectedValueOnce(new Error("offline"));
      listNativeConsents().mockRejectedValueOnce(new Error("offline"));
      const user = userEvent.setup(); show();
      const alert = await screen.findByRole("alert");
      expect(alert).toHaveTextContent(en.asks.list.loadErrorBoth);
      expect(screen.getAllByRole("alert")).toHaveLength(1);
      listAsks.mockImplementation(listAsksFor(() => ({ asks: [open], nextCursor: "" })));
      await user.click(screen.getByTestId(selectors.asks.loadRetry));
      expect(await screen.findByTestId(selectors.asks.item)).toBeInTheDocument();
      expect(screen.queryByRole("alert")).not.toBeInTheDocument();
      expect(pageReads()).toBe(2);
    });

    it("names only the permission requests when only they fail, and keeps the decisions on screen", async () => {
      listAsks.mockImplementation(listAsksFor(() => ({ asks: [open], nextCursor: "" })));
      listNativeConsents().mockRejectedValue(new Error("offline"));
      const user = userEvent.setup(); show();
      expect(await screen.findByTestId(selectors.asks.item)).toBeInTheDocument();
      expect(await screen.findByRole("alert")).toHaveTextContent(en.nativeConsent.unavailable);
      expect(screen.getAllByRole("alert")).toHaveLength(1);
      const decisionReads = listAsks.mock.calls.length;
      await user.click(screen.getByTestId(selectors.asks.loadRetry));
      await waitFor(() => expect(pageReads()).toBe(2));
      expect(listAsks.mock.calls.length).toBe(decisionReads);
    });

    it("names only the decisions when only they fail", async () => {
      listAsks.mockRejectedValue(new Error("offline"));
      show();
      expect(await screen.findByRole("alert")).toHaveTextContent(en.asks.list.loadError);
      expect(screen.getAllByRole("alert")).toHaveLength(1);
      expect(screen.queryByText(en.asks.list.loadErrorBoth)).not.toBeInTheDocument();
    });
  });
});
