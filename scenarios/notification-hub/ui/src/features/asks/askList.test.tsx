import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ListAsksRequest_Order } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { conversationsClient, identityClient } from "../../api/notifications";
import { selectors } from "../../consts/selectors";
import { en } from "../../i18n/catalog";
import { setLocale } from "../../i18n";
import { TestAppRouter } from "../../app/routes";
import { expectNoA11yViolations, listAsksFor, renderWithProviders } from "../../test-utils";
import { makeAsk } from "../packet/mocks/askFactory";

vi.mock("../../api/notifications", () => ({
  identityClient: { getSession: vi.fn(() => Promise.resolve({ signedIn: true, subject: "owner-1" })) },
  conversationsClient: { getAsk: vi.fn(), answer: vi.fn(), listAsks: vi.fn(), listNativeConsents: vi.fn() },
}));
const listAsks = vi.mocked(conversationsClient.listAsks);
const show = (path: string) => renderWithProviders(<TestAppRouter initialEntries={[path]} />, { withoutRouter: true });
// The shell also reads the waiting count (one row); the list reads a page.
const lastRequest = () => listAsks.mock.calls.map(([request]) => request).filter((request) => request.limit === 50).at(-1);
// What the list says about a filtered queue, and the raw source the rows no longer show.
const WAITING_OF = "1 of 2 waiting";
const SOURCE_SLUG = "agent-manager";
const deskAsk = makeAsk({ id: "desk-1", question: "Publish the weekly digest?", options: [], sender: "Research Desk", urgency: "normal", state: "pending" });
const treasuryAsk = makeAsk({ id: "treasury-1", question: "Move funds to cold storage?", options: [], sender: "Treasury steward", urgency: "high", state: "pending" });
const facets = { urgencies: [["high", 1], ["normal", 1]] as [string, number][], senders: [["Research Desk", 1], ["Treasury steward", 1]] as [string, number][] };

describe("sorting and filtering a decision list", () => {
  beforeEach(async () => {
    vi.resetAllMocks();
    vi.mocked(conversationsClient.listNativeConsents).mockResolvedValue({ consents: [], nextCursor: "" } as never);
    vi.mocked(identityClient.getSession).mockResolvedValue({ signedIn: true, subject: "owner-1" } as never);
    Object.defineProperty(HTMLElement.prototype, "scrollTo", { configurable: true, value: vi.fn() });
    await setLocale("en");
  });

  it("says how Needs you is ordered and reorders on the server when the owner picks another sort", async () => {
    listAsks.mockImplementation(listAsksFor(() => ({ asks: [treasuryAsk, deskAsk], totalAwaiting: 2, facets })));
    const user = userEvent.setup();
    show("/asks");
    await screen.findByText(treasuryAsk.question);
    const sort = within(screen.getByTestId(selectors.asks.sort)).getByRole("button", { name: `${en.asks.sort.label}: ${en.asks.sort.attention}` });
    expect(sort).toHaveTextContent(en.asks.sort.attention);
    expect(lastRequest()).toMatchObject({ order: ListAsksRequest_Order.ATTENTION });

    await user.click(sort);
    expect(screen.getAllByRole("option").map((option) => option.textContent.replace("✓", ""))).toEqual([en.asks.sort.attention, en.asks.sort.due, en.asks.sort.newest, en.asks.sort.oldest]);
    await user.click(screen.getByRole("option", { name: en.asks.sort.due }));
    await waitFor(() => expect(lastRequest()).toMatchObject({ order: ListAsksRequest_Order.DUE, cursor: "" }));
    expect(screen.getByTestId(selectors.asks.sort)).toHaveTextContent(en.asks.sort.due);
  });

  it("keeps the sort in the address so a reload or a shared link opens the same view", async () => {
    listAsks.mockImplementation(listAsksFor(() => ({ asks: [deskAsk], totalAwaiting: 1, facets })));
    show("/asks?sort=oldest&urgency=high&sender=Treasury%20steward");
    await screen.findByText(deskAsk.question);
    expect(lastRequest()).toMatchObject({ order: ListAsksRequest_Order.OLDEST, urgencies: ["high"], senders: ["Treasury steward"], stateFilter: "awaiting" });
    expect(screen.getByTestId(selectors.asks.sort)).toHaveTextContent(en.asks.sort.oldest);
  });

  it("opens History on newest first without naming an order, and offers every sort", async () => {
    listAsks.mockImplementation(listAsksFor(() => ({ asks: [deskAsk], facets })));
    const user = userEvent.setup();
    show("/history");
    await screen.findByText(deskAsk.question);
    expect(lastRequest()).not.toHaveProperty("order");
    expect(screen.getByTestId(selectors.asks.sort)).toHaveTextContent(en.asks.sort.newest);
    await user.click(within(screen.getByTestId(selectors.asks.sort)).getByRole("button"));
    await user.click(screen.getByRole("option", { name: en.asks.sort.attention }));
    await waitFor(() => expect(lastRequest()).toMatchObject({ order: ListAsksRequest_Order.ATTENTION }));
  });

  it("filters by urgency and sender from a sheet that shows what each choice would leave, and stays open while choosing", async () => {
    listAsks.mockImplementation(listAsksFor((request) => ({
      asks: request.urgencies?.includes("high") ? [treasuryAsk] : [treasuryAsk, deskAsk],
      totalAwaiting: 2,
      facets: request.urgencies?.includes("high")
        ? { urgencies: [["high", 1], ["normal", 1]], senders: [["Treasury steward", 1]] }
        : facets,
    })));
    const user = userEvent.setup();
    show("/asks");
    await screen.findByText(deskAsk.question);
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent(en.asks.list.title);
    await user.click(await screen.findByTestId(selectors.asks.filtersOpen));
    const sheet = await screen.findByTestId(selectors.asks.filtersSheet);
    const high = within(sheet).getByTestId(selectors.asks.urgencyChip({ urgency: "high" }));
    expect(high).toHaveTextContent(`${en.asks.urgency.high}1`);
    expect(within(sheet).getAllByTestId(selectors.asks.senderChip).map((chip) => chip.textContent)).toEqual(["Research Desk1", "Treasury steward1"]);
    expect(within(sheet).getByTestId(selectors.asks.filtersShow)).toHaveTextContent("Show 2 decisions");

    await user.click(high);
    await waitFor(() => expect(lastRequest()).toMatchObject({ urgencies: ["high"] }));
    // The sheet is still there, now showing the senders of high decisions only.
    await waitFor(() => expect(within(screen.getByTestId(selectors.asks.filtersSheet)).getAllByTestId(selectors.asks.senderChip)).toHaveLength(1));
    expect(screen.getByTestId(selectors.asks.urgencyChip({ urgency: "high" }))).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByTestId(selectors.asks.filtersShow)).toHaveTextContent("Show 1 decision");
    await user.click(screen.getByTestId(selectors.asks.senderChip));
    await waitFor(() => expect(lastRequest()).toMatchObject({ urgencies: ["high"], senders: ["Treasury steward"] }));

    await user.click(screen.getByTestId(selectors.asks.filtersShow));
    await waitFor(() => expect(screen.queryByTestId(selectors.asks.filtersSheet)).not.toBeInTheDocument());
    const active = screen.getByRole("list", { name: en.asks.filters.active });
    expect(within(active).getAllByRole("button").map((chip) => chip.getAttribute("aria-label"))).toEqual([
      `Remove filter: ${en.asks.urgency.high}`, "Remove filter: Treasury steward",
    ]);
    expect(screen.getByTestId(selectors.asks.filtersOpen)).toHaveTextContent("Filters (2)");
    expect(screen.getByText(WAITING_OF)).toBeInTheDocument();
  });

  it("removes one filter with one tap, and every filter with Clear filters", async () => {
    listAsks.mockImplementation(listAsksFor(() => ({ asks: [treasuryAsk], totalAwaiting: 2, facets })));
    const user = userEvent.setup();
    show("/asks?urgency=high&sender=Treasury%20steward&sort=due");
    await screen.findByText(treasuryAsk.question);
    await user.click(screen.getByRole("button", { name: `Remove filter: ${en.asks.urgency.high}` }));
    await waitFor(() => expect(lastRequest()).toMatchObject({ senders: ["Treasury steward"] }));
    expect(lastRequest()).not.toHaveProperty("urgencies");

    await user.click(screen.getByTestId(selectors.asks.filtersOpen));
    await user.click(await screen.findByTestId(selectors.asks.filtersClear));
    await waitFor(() => expect(lastRequest()).not.toHaveProperty("senders"));
    expect(lastRequest()).toMatchObject({ order: ListAsksRequest_Order.DUE });
    expect(screen.queryByRole("list", { name: en.asks.filters.active })).not.toBeInTheDocument();
  });

  it("explains a filter that leaves nothing and offers a way out", async () => {
    listAsks.mockImplementation(listAsksFor((request) => ({ asks: request.urgencies ? [] : [deskAsk], totalAwaiting: 1, facets })));
    const user = userEvent.setup();
    show("/asks?urgency=critical");
    await screen.findByRole("heading", { name: en.pages.history.noMatches });
    await user.click(screen.getByRole("button", { name: en.pages.history.clearFilters }));
    await screen.findByText(deskAsk.question);
    expect(lastRequest()).not.toHaveProperty("urgencies");
  });

  it("groups an attention-ordered queue under headings that say why it is in that order", async () => {
    const escalated = makeAsk({ id: "e1", question: "Escalated one", options: [], state: "escalated", urgency: "low", sender: "Treasury steward" });
    const critical = makeAsk({ id: "c1", question: "Critical one", options: [], state: "pending", urgency: "critical", sender: "Research Desk" });
    listAsks.mockImplementation(listAsksFor(() => ({ asks: [escalated, critical, treasuryAsk, deskAsk], totalAwaiting: 4, facets })));
    show("/asks");
    await screen.findByText(escalated.question);
    const groups = screen.getAllByTestId(selectors.asks.group);
    expect(groups.map((group) => within(group).getByRole("heading", { level: 2 }).textContent)).toEqual([
      en.asks.row.escalated, en.asks.urgency.critical, en.asks.urgency.high, en.asks.urgency.normal,
    ]);
    expect(within(groups[1]!).getByText(critical.question)).toBeInTheDocument();
  });

  it("shows no group headings when the order is not by attention, or when one group holds everything", async () => {
    listAsks.mockImplementation(listAsksFor(() => ({ asks: [treasuryAsk, deskAsk], totalAwaiting: 2, facets })));
    const { unmount } = show("/asks?sort=due");
    await screen.findByText(deskAsk.question);
    expect(screen.queryByTestId(selectors.asks.group)).not.toBeInTheDocument();
    unmount();
    listAsks.mockImplementation(listAsksFor(() => ({ asks: [deskAsk], totalAwaiting: 1, facets })));
    show("/asks");
    await screen.findByText(deskAsk.question);
    expect(screen.queryByTestId(selectors.asks.group)).not.toBeInTheDocument();
  });

  it("names who sent each decision, and says how urgent it is without saying normal", async () => {
    const low = makeAsk({ id: "l1", question: "A low one", options: [], urgency: "low", state: "pending", sender: "Backup manager" });
    listAsks.mockImplementation(listAsksFor(() => ({ asks: [treasuryAsk, deskAsk, low], totalAwaiting: 3, facets })));
    show("/asks?sort=due");
    await screen.findByText(deskAsk.question);
    const rows = screen.getAllByTestId(selectors.asks.item);
    expect(rows[0]).toHaveTextContent("Treasury steward");
    expect(rows[0]).toHaveTextContent(en.asks.urgency.high);
    expect(rows[1]).toHaveTextContent("Research Desk");
    expect(rows[1]).not.toHaveTextContent(en.asks.urgency.normal);
    expect(rows[2]).toHaveTextContent(en.asks.urgency.low);
    expect(screen.queryByText(SOURCE_SLUG)).not.toBeInTheDocument();
  });

  it("has nothing for assistive technology to flag with the sheet open", async () => {
    listAsks.mockImplementation(listAsksFor(() => ({ asks: [treasuryAsk, deskAsk], totalAwaiting: 2, facets })));
    const user = userEvent.setup();
    const { container } = show("/asks?urgency=high");
    await screen.findByText(treasuryAsk.question);
    await expectNoA11yViolations(container);
    await user.click(await screen.findByTestId(selectors.asks.filtersOpen));
    await screen.findByTestId(selectors.asks.filtersSheet);
    await expectNoA11yViolations(document.body);
  });
});
