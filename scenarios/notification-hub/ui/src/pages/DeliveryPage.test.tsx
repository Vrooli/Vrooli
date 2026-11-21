import { en } from "../i18n/catalog";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";

import { deliveryClient } from "../api/notifications";
import { renderWithProviders } from "../test-utils";
import { setLocale } from "../i18n";
import { NotificationState } from "@vrooli/proto-types/notification-hub/v1/shared/types_pb";
import { selectors } from "../consts/selectors";
import { TestAppRouter } from "../app/routes";

const FAILED_RECORD = "Adapter failed";
const CHANNEL = "web_push";
const METRICS_UNAVAILABLE = "metrics unavailable";
const VISIBLE_RECORD = "Visible record";
const TIMELINE_FAILURE = "private timeline failure";
const RECOVERED_RECORD = "Recovered record";

vi.mock("../api/notifications", () => ({
  ROUTED_TEST_LEASE_EXPIRED_EVENT: "notification-hub:routed-test-lease-expired",
  identityClient: { getSession: vi.fn(() => Promise.resolve({ signedIn: true, subject: "owner-1" })) },
  conversationsClient: { getAsk: vi.fn(), answer: vi.fn(), listAsks: vi.fn(() => Promise.resolve({ asks: [], totalAwaiting: 0 })), listNativeConsents: vi.fn(() => Promise.resolve({ consents: [] })) },
  notificationsClient: { get: vi.fn() },
  getDeliveryAttempts: vi.fn(() => Promise.resolve({ attempts: [], truncated: false })),
  deliveryClient: { getTimeline: vi.fn(), getAnalytics: vi.fn() },
}));
vi.mock("../api/health", () => ({
  fetchHealth: vi.fn(() => Promise.resolve({ status: "healthy", service: "notification-hub", timestamp: "2026-10-20T14:00:00Z" })),
}));

const getTimeline = vi.mocked(deliveryClient.getTimeline);
const getAnalytics = vi.mocked(deliveryClient.getAnalytics);
const emptyAnalytics = { since: "", totalNotifications: 0n, channels: [] } as never;
const show = () => renderWithProviders(<TestAppRouter initialEntries={["/delivery"]} />, { withoutRouter: true });

describe("Delivery overview", () => {
  beforeEach(async () => {
    await setLocale("en");
    getAnalytics.mockResolvedValue(emptyAnalytics);
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows each notification with how it was delivered and links to its record", async () => {
    getTimeline.mockResolvedValue({ notifications: [
      { id: "failed-1", title: FAILED_RECORD, reason: "https://provider.example/secret?token=raw-machine-secret", state: NotificationState.FAILED },
      { id: "unknown-1", title: "Future state", state: 99 as NotificationState },
      { id: "ok-1", title: "Reached", state: NotificationState.DELIVERED },
    ] } as never);
    show();

    const page = await screen.findByTestId(selectors.pages.delivery);
    await screen.findByText(FAILED_RECORD);
    const failed = screen.getByTestId(selectors.delivery.recordDetailsLink({ id: "failed-1" }));
    expect(failed).toHaveAttribute("href", "/delivery/failed-1");
    expect(within(failed).getByText(en.pages.delivery.states.failed)).toBeInTheDocument();
    expect(failed).toHaveTextContent("The attempts did not succeed");
    expect(within(screen.getByTestId(selectors.delivery.recordDetailsLink({ id: "unknown-1" }))).getByText(en.pages.delivery.states.unknown)).toBeInTheDocument();
    expect(within(screen.getByTestId(selectors.delivery.recordDetailsLink({ id: "ok-1" }))).getByText(en.pages.delivery.states.delivered)).toBeInTheDocument();
    expect(page).not.toHaveTextContent("provider.example");
    expect(page).not.toHaveTextContent("raw-machine-secret");
    expect(page).toHaveTextContent("Showing the latest 20.");
    expect(screen.queryByRole("button", { name: /resend/i })).not.toBeInTheDocument();
  });

  it("explains all seven delivery states without claiming anyone read the notification", async () => {
    getTimeline.mockResolvedValue({ notifications: [] } as never);
    show();

    const page = await screen.findByTestId(selectors.pages.delivery);
    expect(page).toHaveTextContent(en.pages.delivery.stateGuideDescription);
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
    expect(await screen.findByText(en.pages.delivery.emptyReturned)).toBeInTheDocument();
  });

  it("reports how each channel is doing and whether the Hub itself is running", async () => {
    getTimeline.mockResolvedValue({ notifications: [] } as never);
    getAnalytics.mockResolvedValue({
      since: "2026-10-19T14:00:00Z", totalNotifications: 42n,
      channels: [{ channel: CHANNEL, delivered: 36n, failed: 3n, attempts: 40n, failureRate: 0.075, averageLatencyMs: 820 }],
    } as never);
    show();

    const channel = await screen.findByText(CHANNEL);
    const row = channel.closest("li");
    expect(row).toHaveTextContent("36 delivered · 3 failed");
    expect(row).toHaveTextContent("7.5% failed");
    expect(row).toHaveTextContent("820 ms average");
    expect(await screen.findByText(en.health.running)).toBeInTheDocument();
  });

  it("keeps notification records readable and retries only metrics when channel reliability fails", async () => {
    getTimeline.mockResolvedValue({ notifications: [{ id: "record-visible", title: VISIBLE_RECORD, state: NotificationState.FAILED }] } as never);
    getAnalytics.mockRejectedValueOnce(new Error(METRICS_UNAVAILABLE));
    show();

    expect(await screen.findByText(en.pages.delivery.analyticsError)).toBeInTheDocument();
    expect(await screen.findByText(VISIBLE_RECORD)).toBeInTheDocument();
    expect(screen.queryByText(METRICS_UNAVAILABLE)).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId(selectors.delivery.analyticsRetry));
    await waitFor(() => expect(getAnalytics).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(screen.queryByText(en.pages.delivery.analyticsError)).not.toBeInTheDocument());
    expect(getTimeline).toHaveBeenCalledTimes(1);
  });

  it("recovers the notification list independently after its read fails", async () => {
    getTimeline.mockRejectedValueOnce(new Error(TIMELINE_FAILURE));
    show();

    expect(await screen.findByText(en.pages.delivery.loadError)).toBeInTheDocument();
    expect(screen.queryByText(TIMELINE_FAILURE)).not.toBeInTheDocument();
    expect(getAnalytics).toHaveBeenCalledTimes(1);
    getTimeline.mockResolvedValue({ notifications: [{ id: "recovered-record", title: RECOVERED_RECORD, state: NotificationState.FAILED }] } as never);
    fireEvent.click(screen.getByRole("button", { name: en.common.retry }));

    expect(await screen.findByText(RECOVERED_RECORD)).toBeInTheDocument();
    expect(getAnalytics).toHaveBeenCalledTimes(1);
  });

  it("switches between Updates and Delivery as two views of what the Hub sent", async () => {
    getTimeline.mockResolvedValue({ notifications: [] } as never);
    show();

    const tabs = await screen.findByRole("navigation", { name: en.pages.dashboard.sectionsLabel });
    expect(within(tabs).getByRole("link", { name: "Delivery" })).toHaveAttribute("aria-current", "page");
    expect(within(tabs).getByRole("link", { name: "Updates" })).toHaveAttribute("href", "/updates");
  });
});
