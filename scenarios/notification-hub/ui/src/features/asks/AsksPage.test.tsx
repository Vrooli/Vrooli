import { afterEach, describe, expect, it, vi } from "vitest";
import { act, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient } from "@tanstack/react-query";
import { Route, Routes } from "react-router-dom";

import { conversationsClient } from "../../api/notifications";
import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";
import { i18n } from "../../i18n";
import { strings } from "../../consts/strings";
import { selectors } from "../../consts/selectors";
import { AsksPage } from "./AsksPage";
import { TestAppRouter } from "../../app/routes";

vi.mock("../../api/notifications", () => ({
  identityClient: { getSession: vi.fn(() => Promise.resolve({ signedIn: true, subject: "owner-1" })) },
  conversationsClient: { getAsk: vi.fn(), answer: vi.fn(), listAsks: vi.fn() },
}));

describe("AsksPage", () => {
  afterEach(() => vi.clearAllMocks());

  it("shows a bounded recipient queue without login and links to history", async () => {
    await setLocale("en");
    vi.mocked(conversationsClient.listAsks).mockResolvedValue({
      asks: [{ id: "ask-1", question: "Review the remaining delivery gap", options: [], recommended: "", state: "pending" }],
    } as never);

    renderWithProviders(
      <Routes>
        <Route path="/asks" element={<AsksPage />} />
      </Routes>,
      { initialEntries: ["/asks"] },
    );

    expect(await screen.findByTestId(selectors.asks.item)).toHaveTextContent("Review the remaining delivery gap");
    expect(screen.getByTestId(selectors.asks.list)).toBeInTheDocument();
    expect(screen.getByRole("searchbox", { name: i18n.t(strings.asks.list.filterAll) })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: i18n.t(strings.asks.list.historyLink) })).toHaveAttribute("href", "/history");
    expect(conversationsClient.listAsks).toHaveBeenCalledWith({ openOnly: true, limit: 200 });
  });

  it("shows a bounded checked empty result without claiming the full queue is resolved", async () => {
    await setLocale("en");
    vi.mocked(conversationsClient.listAsks).mockResolvedValue({ asks: [] } as never);
    renderWithProviders(<TestAppRouter initialEntries={["/asks"]} />, { withoutRouter: true });

    expect(await screen.findByRole("heading", { name: i18n.t(strings.asks.list.emptyTitle) })).toBeInTheDocument();
    expect(screen.getByText(i18n.t(strings.asks.list.emptyChecked))).toBeInTheDocument();
    expect(screen.getByText(i18n.t(strings.asks.list.latestCap, { count: 200 }))).toBeInTheDocument();
    expect(screen.getByText(i18n.t(strings.asks.list.olderMayExist))).toBeInTheDocument();
    expect(screen.queryByRole("searchbox")).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: i18n.t(strings.asks.list.historyLink) })).toHaveAttribute("href", "/history");
    expect(conversationsClient.listAsks).toHaveBeenCalledTimes(1);
    expect(conversationsClient.listAsks).toHaveBeenCalledWith({ openOnly: true, limit: 200 });
  });

  it("waits for a pending refresh before calling an empty result current", async () => {
    await setLocale("en");
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } });
    queryClient.setQueryData(["asks", "open"], []);
    let finishRead!: () => void;
    vi.mocked(conversationsClient.listAsks).mockImplementation(() => new Promise((resolve) => {
      finishRead = () => resolve({ asks: [] });
    }) as never);
    renderWithProviders(<TestAppRouter initialEntries={["/asks"]} />, { withoutRouter: true, queryClient });

    expect(await screen.findByText(i18n.t(strings.asks.list.checking))).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: i18n.t(strings.asks.list.emptyTitle) })).not.toBeInTheDocument();
    await act(async () => {
      finishRead();
      await Promise.resolve();
    });
    expect(await screen.findByRole("heading", { name: i18n.t(strings.asks.list.emptyTitle) })).toBeInTheDocument();
  });

  it("distinguishes a failed read from an empty result and retries only the read", async () => {
    const user = userEvent.setup();
    await setLocale("en");
    vi.mocked(conversationsClient.listAsks)
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce({ asks: [] } as never);
    renderWithProviders(<TestAppRouter initialEntries={["/asks?vrooli_test_mode=1"]} />, { withoutRouter: true });

    expect(await screen.findByRole("heading", { name: i18n.t(strings.asks.list.loadError) })).toBeInTheDocument();
    expect(screen.getByText(i18n.t(strings.asks.list.connectionError))).toBeInTheDocument();
    expect(screen.getByText(i18n.t(strings.asks.list.notCleared))).toBeInTheDocument();
    expect(screen.queryByText(i18n.t(strings.asks.list.emptyTitle))).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: i18n.t(strings.asks.list.openSettings) })).toHaveAttribute("href", "/settings?vrooli_test_mode=1");
    expect(screen.getAllByRole("link", { name: i18n.t(strings.layout.nav.history) })
      .some((link) => link.getAttribute("href") === "/history?vrooli_test_mode=1")).toBe(true);
    expect(conversationsClient.listAsks).toHaveBeenCalledTimes(1);
    expect(conversationsClient.listAsks).toHaveBeenCalledWith({ openOnly: true, limit: 200 });

    await user.click(screen.getByRole("button", { name: i18n.t(strings.asks.list.retry) }));
    expect(await screen.findByRole("heading", { name: i18n.t(strings.asks.list.emptyTitle) })).toBeInTheDocument();
    expect(conversationsClient.listAsks).toHaveBeenCalledTimes(2);
    expect(conversationsClient.listAsks).toHaveBeenNthCalledWith(2, { openOnly: true, limit: 200 });
    expect(conversationsClient.answer).not.toHaveBeenCalled();
    expect(conversationsClient.getAsk).not.toHaveBeenCalled();
  });

  it("opens Settings from the failed-read recovery state", async () => {
    const user = userEvent.setup();
    await setLocale("en");
    vi.mocked(conversationsClient.listAsks).mockRejectedValue(new Error("offline"));
    renderWithProviders(<TestAppRouter initialEntries={["/asks"]} />, { withoutRouter: true });
    await screen.findByRole("heading", { name: i18n.t(strings.asks.list.loadError) });

    await user.click(screen.getByRole("link", { name: i18n.t(strings.asks.list.openSettings) }));
    expect(await screen.findByTestId(selectors.pages.settings)).toBeInTheDocument();
    expect(conversationsClient.listAsks).toHaveBeenCalledTimes(1);
    expect(conversationsClient.answer).not.toHaveBeenCalled();
  });

  it("keeps previously returned rows visible when a later read fails", async () => {
    await setLocale("en");
    vi.mocked(conversationsClient.listAsks)
      .mockResolvedValueOnce({ asks: [{ id: "ask-1", question: "Existing request", options: [], recommended: "", state: "pending" }] } as never)
      .mockRejectedValueOnce(new Error("offline"))
      .mockRejectedValueOnce(new Error("still offline"));
    const { queryClient } = renderWithProviders(<TestAppRouter initialEntries={["/asks"]} />, { withoutRouter: true });

    expect(await screen.findByTestId(selectors.asks.item)).toHaveTextContent("Existing request");
    await queryClient.invalidateQueries({ queryKey: ["asks", "open"] });
    expect(await screen.findByRole("status")).toHaveTextContent(i18n.t(strings.asks.list.stale));
    expect(screen.getByTestId(selectors.asks.item)).toHaveTextContent("Existing request");
    expect(screen.getByRole("link", { name: i18n.t(strings.asks.list.historyLink) })).toHaveAttribute("href", "/history");

    await userEvent.setup().click(screen.getByRole("button", { name: i18n.t(strings.asks.list.retry) }));
    expect(await screen.findByRole("status")).toHaveTextContent(i18n.t(strings.asks.list.stale));
    expect(screen.getByTestId(selectors.asks.item)).toHaveTextContent("Existing request");
    expect(conversationsClient.listAsks).toHaveBeenCalledTimes(3);
    expect(conversationsClient.listAsks).toHaveBeenNthCalledWith(3, { openOnly: true, limit: 200 });
    expect(conversationsClient.answer).not.toHaveBeenCalled();
    expect(conversationsClient.getAsk).not.toHaveBeenCalled();
  });

  it("keeps the selected bounded empty state accessible", async () => {
    await setLocale("en");
    vi.mocked(conversationsClient.listAsks).mockResolvedValue({ asks: [] } as never);
    const { container } = renderWithProviders(<TestAppRouter initialEntries={["/asks"]} />, { withoutRouter: true });
    await screen.findByRole("heading", { name: i18n.t(strings.asks.list.emptyTitle) });
    await expectNoA11yViolations(container);
  });

  it("keeps the routed-test opt-in on ask and History links", async () => {
    await setLocale("en");
    vi.mocked(conversationsClient.listAsks).mockResolvedValue({
      asks: [{ id: "ask-1", question: "Review the remaining delivery gap", options: [], recommended: "", state: "pending" }],
    } as never);
    renderWithProviders(<TestAppRouter initialEntries={["/asks?vrooli_test_mode=1"]} />, { withoutRouter: true });

    const askRow = await screen.findByTestId(selectors.asks.item);
    expect(askRow).toHaveTextContent("Review the remaining delivery gap");
    expect(askRow.querySelector("a")).toHaveAttribute("href", "/asks/ask-1?vrooli_test_mode=1");
    expect(screen.getByRole("link", { name: i18n.t(strings.asks.list.historyLink) }))
      .toHaveAttribute("href", "/history?vrooli_test_mode=1");
  });

  it("stores the queue filter in the URL and reports when no ask matches", async () => {
    const user = userEvent.setup();
    await setLocale("en");
    vi.mocked(conversationsClient.listAsks).mockResolvedValue({
      asks: [{ id: "ask-1", question: "Review the remaining delivery gap", options: [], recommended: "", state: "pending" }],
    } as never);
    renderWithProviders(<TestAppRouter initialEntries={["/asks?vrooli_test_mode=1"]} />, { withoutRouter: true });

    const filter = await screen.findByRole("searchbox", { name: i18n.t(strings.asks.list.filterAll) });
    await user.type(filter, "missing topic");
    expect(await screen.findByTestId(selectors.asks.empty)).toHaveTextContent(i18n.t(strings.asks.list.emptyFiltered));
    expect(screen.getByRole("searchbox", { name: i18n.t(strings.asks.list.filterAll) })).toHaveValue("missing topic");
    expect(screen.getByRole("link", { name: i18n.t(strings.asks.list.historyLink) })).toHaveAttribute("href", "/history?vrooli_test_mode=1");
  });

  it("returns from an ask with the selected row focused and list scroll restored", async () => {
    const user = userEvent.setup();
    vi.mocked(conversationsClient.listAsks).mockResolvedValue({
      asks: [{ id: "ask-1", question: "Review the remaining delivery gap", options: [], recommended: "", state: "pending" }],
    } as never);
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({
      ask: { id: "ask-1", question: "Review the remaining delivery gap", options: [], state: "pending" },
    } as never);
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    const row = await screen.findByTestId(selectors.asks.item).then((item) => item.querySelector("a")!);
    const main = screen.getByRole("main");
    Object.defineProperty(main, "scrollTop", { configurable: true, value: 417 });
    const scrollTo = vi.fn();
    Object.defineProperty(main, "scrollTo", { configurable: true, value: scrollTo });

    await user.click(row);
    expect(await screen.findByTestId(selectors.asks.question)).toHaveTextContent("Review the remaining delivery gap");
    await user.click(screen.getByRole("link", { name: "asks.detail.back" }));

    const restored = await screen.findByRole("link", { name: "Review the remaining delivery gap" });
    await waitFor(() => expect(restored).toHaveFocus());
    expect(scrollTo).toHaveBeenCalledWith({ top: 417 });
  });
});
