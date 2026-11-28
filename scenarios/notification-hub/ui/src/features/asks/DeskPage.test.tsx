import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { conversationsClient, identityClient } from "../../api/notifications";
import { TestAppRouter } from "../../app/routes";
import { selectors } from "../../consts/selectors";
import { en } from "../../i18n/catalog";
import { setLocale } from "../../i18n";
import { expectNoA11yViolations, listAsksFor, renderWithProviders } from "../../test-utils";
import { makeAsk } from "../packet/mocks/askFactory";

vi.mock("../../api/notifications", () => ({
  ROUTED_TEST_LEASE_EXPIRED_EVENT: "notification-hub:routed-test-lease-expired",
  identityClient: { getSession: vi.fn(() => Promise.resolve({ signedIn: true, subject: "owner-1" })) },
  conversationsClient: { getAsk: vi.fn(), answer: vi.fn(), listAsks: vi.fn(), listNativeConsents: vi.fn() },
}));

const first = makeAsk({
  id: "ask-1",
  title: "Stop the deletion floors asking twice",
  question: "Restructure both deletion floors?",
  sender: "Steward",
  decisionCause: "Structural change",
  decisionFamily: "guard-friction",
  recommended: "restructure",
  recommendationReason: "Explain the safe route and stop asking the same run twice.",
  options: [
    { key: "restructure", label: "Restructure both floors" },
    { key: "leave", label: "Leave both as they are" },
  ],
  state: "pending",
});
const second = makeAsk({
  id: "ask-2",
  title: "Portal may provision a Memory scope",
  question: "May Portal write its own Memory scope?",
  sender: "Portal",
  recommended: "yes",
  options: [
    { key: "yes", label: "Yes, provision it" },
    { key: "no", label: "No" },
  ],
  state: "pending",
});
const asks = [first, second];

const originalWidth = window.innerWidth;

describe("desktop decision room", () => {
  beforeEach(async () => {
    vi.resetAllMocks();
    Object.defineProperty(window, "innerWidth", { configurable: true, writable: true, value: 1440 });
    vi.mocked(identityClient.getSession).mockResolvedValue({ signedIn: true, subject: "owner-1" } as never);
    vi.mocked(conversationsClient.listAsks).mockImplementation(listAsksFor(() => ({ asks, totalAwaiting: 6 })));
    vi.mocked(conversationsClient.listNativeConsents).mockResolvedValue({ consents: [], nextCursor: "" } as never);
    vi.mocked(conversationsClient.getAsk).mockImplementation(({ askId }) => Promise.resolve({
      ask: asks.find((ask) => ask.id === askId) ?? first,
    }) as never);
    vi.mocked(conversationsClient.answer).mockResolvedValue({} as never);
    await setLocale("en");
  });

  afterEach(() => {
    Object.defineProperty(window, "innerWidth", { configurable: true, writable: true, value: originalWidth });
  });

  it("puts the queue beside the selected decision and reads the centre row", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/asks"]} />, { withoutRouter: true });

    expect(await screen.findByRole("heading", { level: 1, name: en.asks.list.title })).toBeInTheDocument();
    const list = screen.getByTestId(selectors.asks.desk.list);
    const detail = screen.getByTestId(selectors.asks.desk.detail);
    expect(within(list).getByText(first.title)).toBeInTheDocument();
    expect(within(list).getByText(second.title)).toBeInTheDocument();
    expect(await within(detail).findByRole("heading", { level: 2, name: first.title })).toBeInTheDocument();
    expect(within(detail).getAllByText(en.packet.cover.recommended).length).toBeGreaterThan(0);
    expect(within(detail).getByRole("button", { name: en.asks.focus.accept })).toBeInTheDocument();
  });

  it("selects another decision in place without leaving the room", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TestAppRouter initialEntries={["/asks"]} />, { withoutRouter: true });
    await screen.findByRole("heading", { level: 2, name: first.title });

    await user.click(within(screen.getByTestId(selectors.asks.desk.list)).getByRole("button", { name: new RegExp(second.title) }));

    const detail = await screen.findByTestId(selectors.asks.desk.detail);
    expect(await within(detail).findByRole("heading", { level: 2, name: second.title })).toBeInTheDocument();
    expect(screen.getByTestId(selectors.asks.desk.page)).toBeInTheDocument();
  });

  it("accepts the recommendation through the shared answer mutation", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TestAppRouter initialEntries={["/asks"]} />, { withoutRouter: true });
    await screen.findByRole("heading", { level: 2, name: first.title });

    await user.click(screen.getByRole("button", { name: en.asks.focus.accept }));

    await waitFor(() => expect(conversationsClient.answer).toHaveBeenCalledWith(
      expect.objectContaining({ askId: "ask-1", answer: "restructure" }),
    ));
  });

  it("moves to the next decision without answering it", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TestAppRouter initialEntries={["/asks"]} />, { withoutRouter: true });
    await screen.findByRole("heading", { level: 2, name: first.title });

    await user.click(screen.getByRole("button", { name: en.asks.focus.later }));

    const detail = screen.getByTestId(selectors.asks.desk.detail);
    expect(await within(detail).findByRole("heading", { level: 2, name: second.title })).toBeInTheDocument();
    expect(conversationsClient.answer).not.toHaveBeenCalled();
  });

  it("has nothing for assistive technology to flag", async () => {
    const view = renderWithProviders(<TestAppRouter initialEntries={["/asks"]} />, { withoutRouter: true });
    await screen.findByRole("heading", { level: 2, name: first.title });
    await expectNoA11yViolations(view.container);
  });

  it("opens a decision named in the address directly", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/asks?ask=ask-2"]} />, { withoutRouter: true });

    const detail = await screen.findByTestId(selectors.asks.desk.detail);
    expect(await within(detail).findByRole("heading", { level: 2, name: second.title })).toBeInTheDocument();
  });
});
