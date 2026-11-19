import { en } from "../i18n/catalog";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Code, ConnectError } from "@connectrpc/connect";

import { notificationsClient, getDeliveryAttempts, identityClient } from "../api/notifications";
import { renderWithProviders } from "../test-utils";
import { setLocale } from "../i18n";
import { NotificationState } from "@vrooli/proto-types/notification-hub/v1/shared/types_pb";
import { selectors } from "../consts/selectors";
import { TestAppRouter } from "../app/routes";

const HIDDEN_READ_FAILURE = "hidden read failure";
const DETAIL_FAILURE = "private detail failure";

vi.mock("../api/notifications", () => ({
  ROUTED_TEST_LEASE_EXPIRED_EVENT: "notification-hub:routed-test-lease-expired",
  identityClient: { getSession: vi.fn(() => Promise.resolve({ signedIn: true, subject: "owner-1" })), login: vi.fn() },
  conversationsClient: { getAsk: vi.fn(), answer: vi.fn(), listAsks: vi.fn(() => Promise.resolve({ asks: [], totalAwaiting: 0 })), listNativeConsents: vi.fn(() => Promise.resolve({ consents: [] })) },
  notificationsClient: { get: vi.fn() },
  getDeliveryAttempts: vi.fn(() => Promise.resolve({ attempts: [], truncated: false })),
  deliveryClient: { getTimeline: vi.fn(), getAnalytics: vi.fn() },
}));

const getNotification = vi.mocked(notificationsClient.get);
const readAttempts = vi.mocked(getDeliveryAttempts);
const show = (id: string) => renderWithProviders(<TestAppRouter initialEntries={[`/delivery/${id}`]} />, { withoutRouter: true });

describe("Delivery record", () => {
  beforeEach(async () => { await setLocale("en"); });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows only the available receipt evidence and never renders provider identifiers", async () => {
    getNotification.mockResolvedValue({
      notification: { id: "record-1", title: "Report ready", requestedBy: "reports", state: NotificationState.DELIVERED, reason: "https://provider.example/secret?token=raw-machine-secret", createdAt: "2026-10-20T14:00:00Z", updatedAt: "2026-10-20T14:01:00Z", urgency: "normal", sensitivityLabel: "public" },
      receipts: [{ id: "receipt-1", channel: "push", machineId: "phone-1", providerId: "https://secret.example/token", deliveredAt: "2026-10-20T14:01:00Z" }],
    } as never);
    readAttempts.mockResolvedValue({ attempts: [{ channel: "push", attempt: 2, outcome: "retrying", created_at: "2026-10-20T14:00:00Z" }], truncated: true });
    show("record-1");

    expect(await screen.findByRole("heading", { level: 1, name: "Report ready" })).toBeInTheDocument();
    const page = screen.getByTestId(selectors.pages.deliveryDetail);
    expect(page).toHaveTextContent("record-1");
    expect(page).toHaveTextContent("A delivery adapter accepted the payload");
    expect(page).toHaveTextContent("push");
    expect(page).not.toHaveTextContent("phone-1");
    expect(page).not.toHaveTextContent("Machine");
    expect(page).not.toHaveTextContent("provider.example");
    expect(page).not.toHaveTextContent("raw-machine-secret");
    expect(page).toHaveTextContent("Adapter accepted at");
    expect(page).toHaveTextContent(/does not establish device arrival or human reading/);
    expect(await screen.findByText(en.pages.delivery.attemptOutcome)).toBeInTheDocument();
    expect(page).toHaveTextContent("retrying");
    expect(page).toHaveTextContent(en.pages.delivery.attemptsTruncated);
    expect(page).toHaveTextContent(/not a history of changes/);
    expect(page).not.toHaveTextContent("https://secret.example/token");
    expect(screen.queryByRole("button", { name: /resend/i })).not.toBeInTheDocument();
  });

  it("keeps the back link at a comfortable target size while the record loads", () => {
    getNotification.mockReturnValue(new Promise(() => {}));
    show("loading-record");

    const page = screen.getByTestId(selectors.pages.deliveryDetail);
    const backLink = screen.getByRole("link", { name: "Delivery" });
    expect(backLink).toHaveAttribute("href", "/delivery");
    expect(page).toHaveTextContent(en.common.loading);
    expect(backLink.className).toContain("min-h-11");
  });

  it("keeps reach unknown when there is no receipt, even when current status says Delivered", async () => {
    getNotification.mockResolvedValue({
      notification: { id: "no-receipt", title: "No receipt", state: NotificationState.DELIVERED },
      receipts: [],
    } as never);
    readAttempts.mockResolvedValue({ attempts: [], truncated: false });
    show("no-receipt");

    const page = await screen.findByTestId(selectors.pages.deliveryDetail);
    await screen.findByRole("heading", { level: 2, name: en.pages.delivery.reachUnknownTitle });
    expect(page).toHaveTextContent("No adapter acceptance is recorded for this notification.");
    expect(page).toHaveTextContent("A delivery adapter accepted the payload");
    expect(page).toHaveTextContent(en.pages.delivery.noReceipt);
    await waitFor(() => expect(page).not.toHaveTextContent("Delivery attempts"));
  });

  it("keeps an unknown transport value unknown and refreshes through the existing read", async () => {
    getNotification.mockResolvedValue({
      notification: { id: "unknown-transport", title: "Future state", state: 99 as NotificationState, createdAt: "2026-10-20T14:00:00Z", updatedAt: "2026-10-20T14:01:00Z" },
      receipts: [],
    } as never);
    show("unknown-transport");

    const page = await screen.findByTestId(selectors.pages.deliveryDetail);
    await screen.findByRole("heading", { level: 2, name: en.pages.delivery.unknownStatusHeading });
    expect(page).toHaveTextContent("No delivery success is assumed.");
    expect(page).toHaveTextContent("Last checked");
    expect(page).not.toHaveTextContent("Time unavailable");
    expect(page).not.toHaveTextContent("A delivery adapter accepted the payload");
    expect(page).not.toHaveTextContent("Adapter receipts");

    fireEvent.click(screen.getByRole("button", { name: "Refresh status" }));
    await waitFor(() => expect(getNotification).toHaveBeenCalledTimes(2));
  });

  it("shows a returned adapter receipt without treating an unknown state as Delivered", async () => {
    getNotification.mockResolvedValue({
      notification: { id: "unknown-with-receipt", title: "Future state", state: 99 as NotificationState },
      receipts: [{ id: "receipt-unknown", channel: "push", machineId: "phone-2", deliveredAt: "2026-10-20T14:01:00Z" }],
    } as never);
    show("unknown-with-receipt");

    const page = await screen.findByTestId(selectors.pages.deliveryDetail);
    await screen.findByRole("heading", { level: 2, name: en.pages.delivery.unknownStatusHeading });
    expect(page).toHaveTextContent("Adapter accepted at");
    expect(page).toHaveTextContent("push");
    expect(page).not.toHaveTextContent("Delivered");
    expect(page).not.toHaveTextContent("A delivery adapter accepted the payload");
  });

  it("reports failed reads without leaking transport details", async () => {
    getNotification.mockRejectedValue(new Error("secret endpoint token"));
    show("missing");

    const page = screen.getByTestId(selectors.pages.deliveryDetail);
    await waitFor(() => expect(page).toHaveTextContent(en.pages.delivery.detailLoadError));
    expect(page).not.toHaveTextContent("secret endpoint token");
  });

  it("shows retry when the bounded attempts read fails", async () => {
    getNotification.mockResolvedValue({ notification: { id: "attempt-error", state: NotificationState.FAILED }, receipts: [] } as never);
    readAttempts.mockRejectedValueOnce(new Error(HIDDEN_READ_FAILURE));
    show("attempt-error");
    expect(await screen.findByText(en.pages.delivery.attemptsError)).toBeInTheDocument();
    expect(screen.queryByText(HIDDEN_READ_FAILURE)).not.toBeInTheDocument();
    readAttempts.mockResolvedValue({ attempts: [{ channel: "push", attempt: 1, outcome: "failed", created_at: "2026-10-20T14:00:00Z" }], truncated: false });
    fireEvent.click(screen.getByRole("button", { name: en.common.retry }));
    expect(await screen.findByText(en.pages.delivery.attemptOutcome)).toBeInTheDocument();
  });

  it("recovers an unavailable notification without exposing the read error", async () => {
    getNotification.mockRejectedValueOnce(new Error(DETAIL_FAILURE));
    show("detail-retry");

    expect(await screen.findByText(en.pages.delivery.detailLoadError)).toBeInTheDocument();
    expect(screen.queryByText(DETAIL_FAILURE)).not.toBeInTheDocument();
    getNotification.mockResolvedValue({ notification: { id: "detail-retry", state: NotificationState.PENDING }, receipts: [] } as never);
    readAttempts.mockResolvedValue({ attempts: [], truncated: false });
    fireEvent.click(screen.getByRole("button", { name: en.common.retry }));

    expect(await screen.findByText(en.pages.delivery.states.pending)).toBeInTheDocument();
    expect(readAttempts).toHaveBeenCalledWith("detail-retry");
  });
});

describe("Delivery record context link", () => {
  beforeEach(async () => { await setLocale("en"); });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("offers the producer's link as a visible second step, for an external and a relative link alike", async () => {
    for (const [contextUrl, href, name] of [
      ["https://reports.example.test/r/42", "https://reports.example.test/r/42", `${en.pages.dashboard.openSource} · reports.example.test`],
      ["/asks/ask-9", `${window.location.origin}/asks/ask-9`, en.pages.dashboard.openSource],
    ]) {
      getNotification.mockResolvedValue({
        notification: { id: "record-2", title: "Report ready", requestedBy: "reports", state: NotificationState.DELIVERED, createdAt: "2026-10-20T14:00:00Z", updatedAt: "2026-10-20T14:01:00Z", urgency: "normal", sensitivityLabel: "public", contextUrl },
        receipts: [],
      } as never);
      const view = show("record-2");
      const link = await screen.findByRole("link", { name });
      expect(link).toHaveAttribute("href", href);
      view.unmount();
    }
  });

  it("offers no link for a notification without one or with an unsafe one", async () => {
    for (const contextUrl of ["", "javascript:alert(1)"]) {
      getNotification.mockResolvedValue({
        notification: { id: "record-3", title: "Report ready", requestedBy: "reports", state: NotificationState.DELIVERED, createdAt: "2026-10-20T14:00:00Z", updatedAt: "2026-10-20T14:01:00Z", urgency: "normal", sensitivityLabel: "public", contextUrl },
        receipts: [],
      } as never);
      const view = show("record-3");
      await screen.findByRole("heading", { level: 1, name: "Report ready" });
      expect(screen.queryByRole("link", { name: new RegExp(`^${en.pages.dashboard.openSource}`) })).not.toBeInTheDocument();
      view.unmount();
    }
  });
});

describe("Delivery record from a native permission request", () => {
  const authorizationId = "authorization-fixture-1";
  const reviewUrl = `https://agent-manager.example.test/api/v1/authorizations/${authorizationId}/approve`;
  const record = (extra: Record<string, unknown>, contextUrl = reviewUrl) => ({
    notification: { id: "record-4", title: "Authorization requested", requestedBy: "owner-1", state: NotificationState.DELIVERED, createdAt: "2026-10-20T14:00:00Z", updatedAt: "2026-10-20T14:01:00Z", urgency: "normal", sensitivityLabel: "private", dedupeKey: authorizationId, contextUrl },
    receipts: [],
    ...extra,
  });
  beforeEach(async () => { await setLocale("en"); });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("leads with a primary Review in Agent Manager button above every fact, and says where it goes", async () => {
    getNotification.mockResolvedValue(record({ nativeConsentReviewUrl: reviewUrl }) as never);
    show("record-4");

    const button = await screen.findByRole("link", { name: en.nativeConsent.open });
    expect(button).toHaveAttribute("href", reviewUrl);
    expect(button.className).toContain("bg-app-primary");
    expect(button).toHaveAccessibleDescription(new RegExp(`${en.nativeConsent.authorityShort} agent-manager\\.example\\.test`));
    expect(button.parentElement?.querySelector("bdi")).toHaveAttribute("dir", "ltr");
    const firstFact = screen.getByText(en.pages.delivery.created);
    expect(button.compareDocumentPosition(firstFact) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(button.compareDocumentPosition(screen.getByText(en.pages.delivery.states.delivered)) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it("does not repeat the same page as a footer Open source link", async () => {
    getNotification.mockResolvedValue(record({ nativeConsentReviewUrl: reviewUrl }) as never);
    show("record-4");

    await screen.findByRole("link", { name: en.nativeConsent.open });
    expect(screen.queryByRole("link", { name: new RegExp(`^${en.pages.dashboard.openSource}`) })).not.toBeInTheDocument();
  });

  it("offers no review button for a notification the Hub did not name as a permission request", async () => {
    // Same link shape and the same dedupe key, but no Hub verdict: the page never infers one.
    getNotification.mockResolvedValue(record({}) as never);
    show("record-4");

    await screen.findByRole("heading", { level: 1, name: "Authorization requested" });
    expect(screen.queryByRole("link", { name: en.nativeConsent.open })).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: `${en.pages.dashboard.openSource} · agent-manager.example.test` })).toHaveAttribute("href", reviewUrl);
  });
});

describe("Delivery record opened while signed out", () => {
  beforeEach(async () => { await setLocale("en"); });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows the sign-in card, then the record itself once the owner signs in, without a reload", async () => {
    const user = userEvent.setup();
    getNotification.mockRejectedValueOnce(new ConnectError("no session", Code.Unauthenticated));
    getNotification.mockResolvedValue({
      notification: { id: "record-5", title: "Report ready", requestedBy: "owner-1", state: NotificationState.DELIVERED, createdAt: "2026-10-20T14:00:00Z", updatedAt: "2026-10-20T14:01:00Z", urgency: "normal", sensitivityLabel: "public" },
      receipts: [],
    } as never);
    vi.mocked(identityClient.login).mockResolvedValue({} as never);
    show("record-5");

    await user.type(await screen.findByTestId(selectors.session.email), "owner@example.test");
    await user.type(screen.getByTestId(selectors.session.password), "secret");
    await user.click(screen.getByTestId(selectors.session.submit));

    expect(await screen.findByRole("heading", { level: 1, name: "Report ready" })).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.session.form)).not.toBeInTheDocument();
    expect(getNotification).toHaveBeenCalledTimes(2);
  });
});
