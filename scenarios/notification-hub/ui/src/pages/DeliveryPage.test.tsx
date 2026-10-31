import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";

import { notificationsClient, deliveryClient, getDeliveryAttempts } from "../api/notifications";
import { renderWithProviders } from "../test-utils";
import { setLocale } from "../i18n";
import { NotificationState } from "@vrooli/proto-types/notification-hub/v1/shared/types_pb";
import { selectors } from "../consts/selectors";
import { TestAppRouter } from "../app/routes";

vi.mock("../api/notifications", () => ({
  identityClient: { getSession: vi.fn(() => Promise.resolve({ signedIn: true, subject: "owner-1" })) },
  conversationsClient: { getAsk: vi.fn(), answer: vi.fn(), listAsks: vi.fn(() => Promise.resolve({ asks: [] })) },
  notificationsClient: { get: vi.fn() },
  getDeliveryAttempts: vi.fn(() => Promise.resolve({ attempts: [], truncated: false })),
  deliveryClient: { getTimeline: vi.fn(), getAnalytics: vi.fn(() => Promise.resolve({ totalNotifications: 0, channels: [] })) },
}));

const getTimeline = vi.mocked(deliveryClient.getTimeline);
const getNotification = vi.mocked(notificationsClient.get);
const getAnalytics = vi.mocked(deliveryClient.getAnalytics);
const readAttempts = vi.mocked(getDeliveryAttempts);

describe("Delivery views", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows current returned notification state in the overview and links to detail", async () => {
    await setLocale("en");
    getTimeline.mockResolvedValue({ notifications: [
      { id: "failed-1", title: "Adapter failed", reason: "https://provider.example/secret?token=raw-machine-secret", state: NotificationState.FAILED },
      { id: "unknown-1", title: "Future state", state: 99 as NotificationState },
    ] } as never);
    renderWithProviders(<TestAppRouter initialEntries={["/delivery"]} />, { withoutRouter: true });

    const page = await screen.findByTestId(selectors.pages.delivery);
    expect(page).toHaveTextContent("Delivery states");
    expect(page).toHaveTextContent("These describe transport, not whether someone read or answered a request.");
    for (const [state, meaning] of [
      ["Pending", "Accepted for processing"],
      ["Held", "Waiting on a routing policy"],
      ["Routed", "A channel was selected"],
      ["Delivered", "A delivery adapter accepted the payload"],
      ["Failed", "The attempts did not succeed"],
      ["Unroutable", "No eligible target was found"],
      ["Suppressed", "Intentionally not sent"],
    ] as const) {
      expect(page).toHaveTextContent(state);
      expect(page).toHaveTextContent(meaning);
    }
    expect(page).toHaveTextContent("Adapter failed");
    expect(page).toHaveTextContent("Failed");
    expect(page).not.toHaveTextContent("provider.example");
    expect(page).not.toHaveTextContent("raw-machine-secret");
    expect(page).toHaveTextContent("Unknown status");
    expect(screen.getAllByRole("link", { name: "Delivery details" })[0]).toHaveAttribute("href", "/delivery/failed-1");
    expect(screen.getByTestId(selectors.delivery.recordDetailsLink({ id: "failed-1" }))).toHaveAttribute("href", "/delivery/failed-1");
    expect(screen.getByText(/up to 20 recent notifications returned/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /retry|resend/i })).not.toBeInTheDocument();
    expect(screen.queryByText(/metrics couldn’t be loaded/i)).not.toBeInTheDocument();
  });

  it("points analytics recovery directly at the notification record list", async () => {
    await setLocale("en");
    getTimeline.mockResolvedValue({ notifications: [{ id: "record-1", title: "A returned record", state: NotificationState.PENDING }] } as never);
    getAnalytics.mockRejectedValue(new Error("analytics unavailable"));
    renderWithProviders(<TestAppRouter initialEntries={["/delivery"]} />, { withoutRouter: true });

    const control = await screen.findByRole("button", { name: "View delivery records" });
    expect(control).toHaveAttribute("data-testid", selectors.delivery.analyticsRecordsAction);
    expect(control).toHaveAttribute("aria-controls", "delivery-record-list");
    fireEvent.click(control);
    expect(document.querySelector("#delivery-record-list")).toHaveFocus();
    expect(document.querySelectorAll("#delivery-record-list")).toHaveLength(1);
    expect(document.querySelector("#delivery-record-list")).toHaveTextContent("A returned record");
  });

  it("focuses the empty records result after the analytics recovery action", async () => {
    await setLocale("en");
    getTimeline.mockResolvedValue({ notifications: [] } as never);
    getAnalytics.mockRejectedValue(new Error("analytics unavailable"));
    renderWithProviders(<TestAppRouter initialEntries={["/delivery"]} />, { withoutRouter: true });

    expect(await screen.findByRole("button", { name: "View delivery records" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "View delivery records" }));
    expect(await screen.findByText("No notifications were returned within this request’s scope.")).toBeInTheDocument();
    expect(document.querySelector("#delivery-record-list")).toHaveFocus();
  });

  it("shows only the available receipt evidence and never renders provider identifiers", async () => {
    await setLocale("en");
    getNotification.mockResolvedValue({
      notification: { id: "record-1", title: "Report ready", requestedBy: "reports", state: NotificationState.DELIVERED, reason: "https://provider.example/secret?token=raw-machine-secret", createdAt: "2026-10-20T14:00:00Z", updatedAt: "2026-10-20T14:01:00Z", urgency: "normal", sensitivityLabel: "public" },
      receipts: [{ id: "receipt-1", channel: "push", machineId: "phone-1", providerId: "https://secret.example/token", deliveredAt: "2026-10-20T14:01:00Z" }],
    } as never);
    readAttempts.mockResolvedValue({ attempts: [{ channel: "push", attempt: 2, outcome: "retrying", created_at: "2026-10-20T14:00:00Z" }], truncated: true });
    renderWithProviders(<TestAppRouter initialEntries={["/delivery/record-1"]} />, { withoutRouter: true });

    expect(await screen.findByTestId("page-delivery-detail")).toBeInTheDocument();
    const page = screen.getByTestId(selectors.pages.deliveryDetail);
    expect(page).toHaveTextContent("A delivery adapter accepted the payload");
    expect(page).toHaveTextContent("push");
    expect(page).not.toHaveTextContent("phone-1");
    expect(page).not.toHaveTextContent("Machine");
    expect(page).not.toHaveTextContent("provider.example");
    expect(page).not.toHaveTextContent("raw-machine-secret");
    expect(page).toHaveTextContent("Adapter accepted at");
    expect(page).toHaveTextContent(/does not establish device arrival or human reading/);
    expect(await screen.findByText("Recorded outcome")).toBeInTheDocument();
    expect(page).toHaveTextContent("retrying");
    expect(page).toHaveTextContent("Showing a bounded portion of the recorded attempts.");
    expect(page).toHaveTextContent(/snapshot, not a state-transition timeline/);
    expect(page).not.toHaveTextContent("https://secret.example/token");
    expect(screen.queryByRole("button", { name: /retry|resend/i })).not.toBeInTheDocument();
  });

  it("keeps the detail back link at a comfortable target size while the record loads", async () => {
    await setLocale("en");
    getNotification.mockReturnValue(new Promise(() => {}));
    renderWithProviders(<TestAppRouter initialEntries={["/delivery/loading-record"]} />, { withoutRouter: true });

    const page = screen.getByTestId(selectors.pages.deliveryDetail);
    const backLink = screen.getByRole("link", { name: "Back to Delivery" });
    expect(page).toHaveTextContent("Loading decision…");
    expect(backLink.className).toContain("min-h-11");
    expect(backLink.className).toContain("items-center");
  });

  it("keeps reach unknown when there is no receipt, even when current status says Delivered", async () => {
    await setLocale("en");
    getNotification.mockResolvedValue({
      notification: { id: "no-receipt", title: "No receipt", state: NotificationState.DELIVERED },
      receipts: [],
    } as never);
    readAttempts.mockResolvedValue({ attempts: [], truncated: false });
    renderWithProviders(<TestAppRouter initialEntries={["/delivery/no-receipt"]} />, { withoutRouter: true });

    const page = await screen.findByTestId(selectors.pages.deliveryDetail);
    expect(page).toHaveTextContent("No adapter acceptance is recorded for this notification.");
    expect(page).toHaveTextContent("A delivery adapter accepted the payload");
    expect(page).toHaveTextContent("Recorded answer");
    expect(page).toHaveTextContent("Separate request state");
    expect(page).toHaveTextContent("No receipt does not mean unread, offline or a failed answer.");
    await waitFor(() => expect(page).not.toHaveTextContent("Delivery attempts"));
  });

  it("keeps an unknown transport value unknown and refreshes through the existing read", async () => {
    await setLocale("en");
    getNotification.mockResolvedValue({
      notification: { id: "unknown-transport", title: "Future state", state: 99 as NotificationState, updatedAt: "2026-10-20T14:01:00Z" },
      receipts: [],
    } as never);
    renderWithProviders(<TestAppRouter initialEntries={["/delivery/unknown-transport"]} />, { withoutRouter: true });

    const page = await screen.findByTestId(selectors.pages.deliveryDetail);
    expect(page).toHaveTextContent("Delivery status unavailable");
    expect(page).toHaveTextContent("The returned state isn’t recognized");
    expect(page).toHaveTextContent("No delivery success is assumed.");
    expect(page).toHaveTextContent("The returned state isn’t recognized");
    expect(page).toHaveTextContent("Last checked");
    expect(page).not.toHaveTextContent("Time unavailable");
    expect(page).not.toHaveTextContent("A delivery adapter accepted the payload");
    expect(page).not.toHaveTextContent("Adapter receipts");

    fireEvent.click(screen.getByRole("button", { name: "Refresh status" }));
    await waitFor(() => expect(getNotification).toHaveBeenCalledTimes(2));
  });

  it("shows a returned adapter receipt without treating an unknown state as Delivered", async () => {
    await setLocale("en");
    getNotification.mockResolvedValue({
      notification: { id: "unknown-with-receipt", title: "Future state", state: 99 as NotificationState },
      receipts: [{ id: "receipt-unknown", channel: "push", machineId: "phone-2", deliveredAt: "2026-10-20T14:01:00Z" }],
    } as never);
    renderWithProviders(<TestAppRouter initialEntries={["/delivery/unknown-with-receipt"]} />, { withoutRouter: true });

    const page = await screen.findByTestId(selectors.pages.deliveryDetail);
    expect(page).toHaveTextContent("Delivery status unavailable");
    expect(page).toHaveTextContent("The returned state isn’t recognized");
    expect(page).toHaveTextContent("Adapter accepted at");
    expect(page).toHaveTextContent("push");
    expect(page).not.toHaveTextContent("Delivered");
    expect(page).not.toHaveTextContent("A delivery adapter accepted the payload");
  });

  it("reports failed reads without leaking transport details", async () => {
    await setLocale("en");
    getNotification.mockRejectedValue(new Error("secret endpoint token"));
    renderWithProviders(<TestAppRouter initialEntries={["/delivery/missing"]} />, { withoutRouter: true });

    const page = screen.getByTestId(selectors.pages.deliveryDetail);
    await waitFor(() => expect(page).toHaveTextContent("This delivery record could not be loaded."));
    expect(page).not.toHaveTextContent("secret endpoint token");
  });

  it("shows retry when the bounded attempts read fails", async () => {
    await setLocale("en");
    getNotification.mockResolvedValue({ notification: { id: "attempt-error", state: NotificationState.FAILED }, receipts: [] } as never);
    readAttempts.mockRejectedValueOnce(new Error("hidden read failure"));
    renderWithProviders(<TestAppRouter initialEntries={["/delivery/attempt-error"]} />, { withoutRouter: true });
    expect(await screen.findByText("Recorded attempts could not be loaded.")).toBeInTheDocument();
    expect(screen.queryByText("hidden read failure")).not.toBeInTheDocument();
    readAttempts.mockResolvedValue({ attempts: [{ channel: "push", attempt: 1, outcome: "failed", created_at: "2026-10-20T14:00:00Z" }], truncated: false });
    fireEvent.click(screen.getByRole("button", { name: "Retry attempt read" }));
    expect(await screen.findByText("Recorded outcome")).toBeInTheDocument();
  });

  it("offers analytics recovery while leaving notification records readable", async () => {
    await setLocale("en");
    getTimeline.mockResolvedValue({ notifications: [{ id: "record-visible", title: "Visible record", state: NotificationState.FAILED }] } as never);
    getAnalytics.mockRejectedValueOnce(new Error("metrics unavailable"));
    renderWithProviders(<TestAppRouter initialEntries={["/delivery"]} />, { withoutRouter: true });
    expect(await screen.findByText("Attempt metrics couldn’t be loaded")).toBeInTheDocument();
    expect(screen.queryByText("metrics unavailable")).not.toBeInTheDocument();
    expect(screen.queryByText("Visible record")).not.toBeInTheDocument();
    getAnalytics.mockResolvedValue({ totalNotifications: 0, channels: [] } as never);
    expect(getAnalytics).toHaveBeenCalledTimes(1);
    expect(screen.getByText("Attempt metrics couldn’t be loaded")).toBeInTheDocument();
    expect(screen.getByText("You can review individual delivery records while metrics are unavailable.")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "View delivery records" }));
    expect(await screen.findByText("Visible record")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Retry analytics" }));
    await waitFor(() => expect(getAnalytics).toHaveBeenCalledTimes(2));
  });

  it("recovers the delivery timeline independently after its read fails", async () => {
    await setLocale("en");
    getTimeline.mockRejectedValueOnce(new Error("private timeline failure"));
    getAnalytics.mockResolvedValue({ totalNotifications: 0, channels: [] } as never);
    renderWithProviders(<TestAppRouter initialEntries={["/delivery"]} />, { withoutRouter: true });

    expect(await screen.findByText("Delivery records could not be loaded.")).toBeInTheDocument();
    expect(screen.queryByText("private timeline failure")).not.toBeInTheDocument();
    expect(getAnalytics).toHaveBeenCalledTimes(1);
    expect(screen.queryByText("Attempt metrics couldn’t be loaded")).not.toBeInTheDocument();
    getTimeline.mockResolvedValue({ notifications: [{ id: "recovered-record", title: "Recovered record", state: NotificationState.FAILED }] } as never);
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));

    expect(await screen.findByText("Recovered record")).toBeInTheDocument();
    expect(getAnalytics).toHaveBeenCalledTimes(1);
  });

  it("recovers an unavailable notification detail without exposing the read error", async () => {
    await setLocale("en");
    getNotification.mockRejectedValueOnce(new Error("private detail failure"));
    renderWithProviders(<TestAppRouter initialEntries={["/delivery/detail-retry"]} />, { withoutRouter: true });

    expect(await screen.findByText("This delivery record could not be loaded.")).toBeInTheDocument();
    expect(screen.queryByText("private detail failure")).not.toBeInTheDocument();
    getNotification.mockResolvedValue({ notification: { id: "detail-retry", state: NotificationState.PENDING }, receipts: [] } as never);
    readAttempts.mockResolvedValue({ attempts: [], truncated: false });
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));

    await waitFor(() => expect(screen.getByText("Transport state").parentElement).toHaveTextContent("Pending"));
    expect(readAttempts).toHaveBeenCalledWith("detail-retry");
  });
});
