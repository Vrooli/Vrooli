import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";

import { notificationsClient, deliveryClient } from "../api/notifications";
import { renderWithProviders } from "../test-utils";
import { setLocale } from "../i18n";
import { NotificationState } from "@vrooli/proto-types/notification-hub/v1/shared/types_pb";
import { selectors } from "../consts/selectors";
import { TestAppRouter } from "../app/routes";

vi.mock("../api/notifications", () => ({
  identityClient: { getSession: vi.fn(() => Promise.resolve({ signedIn: true, subject: "owner-1" })) },
  conversationsClient: { getAsk: vi.fn(), answer: vi.fn(), listAsks: vi.fn(() => Promise.resolve({ asks: [] })) },
  notificationsClient: { get: vi.fn() },
  deliveryClient: { getTimeline: vi.fn() },
}));

const getTimeline = vi.mocked(deliveryClient.getTimeline);
const getNotification = vi.mocked(notificationsClient.get);

describe("Delivery views", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows current returned notification state in the overview and links to detail", async () => {
    await setLocale("en");
    getTimeline.mockResolvedValue({ notifications: [
      { id: "failed-1", title: "Adapter failed", reason: "provider timeout", state: NotificationState.FAILED },
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
      ["Delivered", "At least one adapter accepted the payload"],
      ["Failed", "Delivery did not succeed under its retry policy"],
      ["Unroutable", "No eligible target was available"],
      ["Suppressed", "Not sent under a suppression rule"],
    ] as const) {
      expect(page).toHaveTextContent(state);
      expect(page).toHaveTextContent(meaning);
    }
    expect(page).toHaveTextContent("Adapter failed");
    expect(page).toHaveTextContent("Failed");
    expect(page).toHaveTextContent("provider timeout");
    expect(page).toHaveTextContent("Unknown status");
    expect(screen.getAllByRole("link", { name: "Delivery details" })[0]).toHaveAttribute("href", "/delivery/failed-1");
    expect(screen.getByTestId(selectors.delivery.recordDetailsLink({ id: "failed-1" }))).toHaveAttribute("href", "/delivery/failed-1");
    expect(screen.getByText(/up to 20 recent notifications returned/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /retry|resend/i })).not.toBeInTheDocument();
    expect(screen.queryByText(/analytics/i)).not.toBeInTheDocument();
  });

  it("shows only the available receipt evidence and never renders provider identifiers", async () => {
    await setLocale("en");
    getNotification.mockResolvedValue({
      notification: { id: "record-1", title: "Report ready", requestedBy: "reports", state: NotificationState.DELIVERED, reason: "accepted", createdAt: "2026-10-20T14:00:00Z", updatedAt: "2026-10-20T14:01:00Z", urgency: "normal", sensitivityLabel: "public" },
      receipts: [{ id: "receipt-1", channel: "push", machineId: "phone-1", providerId: "https://secret.example/token", deliveredAt: "2026-10-20T14:01:00Z" }],
    } as never);
    renderWithProviders(<TestAppRouter initialEntries={["/delivery/record-1"]} />, { withoutRouter: true });

    expect(await screen.findByTestId("page-delivery-detail")).toBeInTheDocument();
    const page = screen.getByTestId(selectors.pages.deliveryDetail);
    expect(page).toHaveTextContent("At least one adapter accepted the payload.");
    expect(page).toHaveTextContent("push");
    expect(page).toHaveTextContent("phone-1");
    expect(page).toHaveTextContent("Adapter accepted at");
    expect(page).toHaveTextContent(/does not establish device arrival or human reading/);
    expect(page).toHaveTextContent(/Attempt history is not available/);
    expect(page).toHaveTextContent(/snapshot, not a state-transition timeline/);
    expect(page).not.toHaveTextContent("https://secret.example/token");
    expect(screen.queryByRole("button", { name: /retry|resend/i })).not.toBeInTheDocument();
  });

  it("keeps reach unknown when there is no receipt, even when current status says Delivered", async () => {
    await setLocale("en");
    getNotification.mockResolvedValue({
      notification: { id: "no-receipt", title: "No receipt", state: NotificationState.DELIVERED },
      receipts: [],
    } as never);
    renderWithProviders(<TestAppRouter initialEntries={["/delivery/no-receipt"]} />, { withoutRouter: true });

    const page = await screen.findByTestId(selectors.pages.deliveryDetail);
    expect(page).toHaveTextContent("No adapter receipt was returned. Reach remains unknown.");
    expect(page).toHaveTextContent("At least one adapter accepted the payload.");
    expect(page).toHaveTextContent("Ask and history status are separate from delivery receipt evidence.");
    expect(page).toHaveTextContent("A missing receipt does not tell you whether the notification was read, the device was offline, or a separate request was answered.");
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
    expect(page).toHaveTextContent("No delivery success is assumed from this status value.");
    expect(page).toHaveTextContent("Unknown status");
    expect(page).toHaveTextContent("Time unavailable");
    expect(page).not.toHaveTextContent("At least one adapter accepted the payload");
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
    expect(page).toHaveTextContent("Unknown status");
    expect(page).toHaveTextContent("Adapter accepted at");
    expect(page).toHaveTextContent("push");
    expect(page).not.toHaveTextContent("Delivered");
    expect(page).not.toHaveTextContent("At least one adapter accepted the payload");
  });

  it("reports failed reads without leaking transport details", async () => {
    await setLocale("en");
    getNotification.mockRejectedValue(new Error("secret endpoint token"));
    renderWithProviders(<TestAppRouter initialEntries={["/delivery/missing"]} />, { withoutRouter: true });

    const page = screen.getByTestId(selectors.pages.deliveryDetail);
    await waitFor(() => expect(page).toHaveTextContent("This delivery record could not be loaded."));
    expect(page).not.toHaveTextContent("secret endpoint token");
  });
});
