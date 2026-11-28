import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ListAsksRequest_Order } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { conversationsClient, identityClient } from "../../api/notifications";
import { selectors } from "../../consts/selectors";
import { en } from "../../i18n/catalog";
import { setLocale } from "../../i18n";
import { TestAppRouter } from "../../app/routes";
import { listAsksFor, renderWithProviders } from "../../test-utils";
import { makeAsk, makeAttachment } from "../packet/mocks/askFactory";

vi.mock("../../api/attachments", () => ({
  attachmentUrl: (askId: string, name: string, path = "") => `/files/${askId}/${name}${path ? `?path=${path}` : ""}`,
  fetchAttachment: vi.fn(),
  resolveLinkedArtifact: vi.fn(),
}));

vi.mock("../../api/notifications", () => ({
  ROUTED_TEST_LEASE_EXPIRED_EVENT: "notification-hub:routed-test-lease-expired",
  identityClient: { getSession: vi.fn(() => Promise.resolve({ signedIn: true, subject: "owner-1" })) },
  conversationsClient: { getAsk: vi.fn(), answer: vi.fn(), listAsks: vi.fn(), listNativeConsents: vi.fn() },
}));

const PROGRESS = "1 of 2";
const SECOND_PROGRESS = "2 of 2";
const RECORDED = "Answer recorded: Re-aim at quality";
const first = makeAsk({
  id: "ask-1",
  title: "First decision",
  question: "Re-aim or close?",
  recommended: "re-aim",
  recommendationReason: "Quality gaps remain.",
  state: "pending",
});
const answeredFirst = makeAsk({
  id: "ask-1",
  title: "First decision",
  question: "Re-aim or close?",
  recommended: "re-aim",
  recommendationReason: "Quality gaps remain.",
  state: "answered",
  answer: "re-aim",
  answerLabel: "Re-aim at quality",
});
const second = makeAsk({
  id: "ask-2",
  title: "Second decision",
  question: "Publish or hold?",
  recommended: "hold",
  options: [{ key: "publish", label: "Publish" }, { key: "hold", label: "Hold" }],
  state: "pending",
});

describe("focus review", () => {
  let answered = false;
  beforeEach(async () => {
    vi.resetAllMocks();
    answered = false;
    vi.mocked(identityClient.getSession).mockResolvedValue({ signedIn: true, subject: "owner-1" } as never);
    vi.mocked(conversationsClient.listNativeConsents).mockResolvedValue({ consents: [], nextCursor: "" } as never);
    vi.mocked(conversationsClient.listAsks).mockImplementation(listAsksFor(() => ({ asks: [first, second], totalAwaiting: 2 })));
    vi.mocked(conversationsClient.answer).mockImplementation(async () => { answered = true; return {} as never; });
    vi.mocked(conversationsClient.getAsk).mockImplementation(async ({ askId }) => ({
      ask: askId === "ask-2" ? second : answered ? answeredFirst : first,
    }) as never);
    await setLocale("en");
  });

  it("reads the decision's full detail, not only its queue row", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/focus"]} />, { withoutRouter: true });
    await screen.findByRole("heading", { level: 1, name: first.title });
    await waitFor(() => expect(conversationsClient.getAsk).toHaveBeenCalledWith({ askId: "ask-1" }));
  });

  it("shows the requester's decision page beside the answer", async () => {
    const withPage = makeAsk({
      id: "ask-1",
      title: "First decision",
      question: "Re-aim or close?",
      recommended: "re-aim",
      recommendationReason: "Quality gaps remain.",
      state: "pending",
      attachments: [makeAttachment({ name: "decision.html", explainer: true, mediaType: "text/html" })],
    });
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: withPage } as never);
    const { fetchAttachment } = await import("../../api/attachments");
    vi.mocked(fetchAttachment).mockResolvedValue(new Response("<html><body><h2>A chart lives here</h2></body></html>", { headers: { "content-type": "text/html" } }));

    renderWithProviders(<TestAppRouter initialEntries={["/focus"]} />, { withoutRouter: true });

    expect(await screen.findByTestId(selectors.packet.explainer)).toBeInTheDocument();
    await waitFor(() => expect(fetchAttachment).toHaveBeenCalledWith("ask-1", "decision.html", "", expect.any(AbortSignal)));
  });

  it("presents the attention-order decision with its progress, recommendation and options", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/focus"]} />, { withoutRouter: true });

    expect(await screen.findByRole("heading", { level: 1, name: first.title })).toBeInTheDocument();
    expect(screen.getByText(PROGRESS)).toBeInTheDocument();
    expect(screen.getAllByText(en.packet.cover.recommended).length).toBeGreaterThan(0);
    expect(screen.getByRole("radio", { name: /Re-aim at quality/ })).toBeInTheDocument();
    expect(conversationsClient.listAsks).toHaveBeenCalledWith(
      expect.objectContaining({ stateFilter: "awaiting", order: ListAsksRequest_Order.ATTENTION }),
      expect.anything(),
    );
  });

  it("accepts the recommendation through the shared answer mutation, then advances", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TestAppRouter initialEntries={["/focus"]} />, { withoutRouter: true });
    await screen.findByRole("heading", { level: 1, name: first.title });

    await user.click(screen.getByRole("button", { name: en.asks.focus.accept }));

    await waitFor(() => expect(conversationsClient.answer).toHaveBeenCalledWith(
      expect.objectContaining({ askId: "ask-1", answer: "re-aim" }),
    ));
    expect(await screen.findByText(RECORDED)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: en.asks.focus.next }));
    expect(await screen.findByRole("heading", { level: 1, name: second.title })).toBeInTheDocument();
    expect(screen.getByTestId(selectors.pages.focus)).toHaveTextContent(SECOND_PROGRESS);
  });
});
