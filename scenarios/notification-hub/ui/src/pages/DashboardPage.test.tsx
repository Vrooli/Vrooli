import { afterEach, describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { NotificationState } from "@vrooli/proto-types/notification-hub/v1/shared/types_pb";
import { selectors } from "../consts/selectors";
import { setLocale } from "../i18n";
import { DashboardPage } from "./DashboardPage";

vi.mock("../features/health/HealthCard", () => ({
  HealthCard: () => <div data-testid="health-card" />,
}));

vi.mock("../api/notifications", () => ({
  deliveryClient: { getTimeline: vi.fn() },
  conversationsClient: { listAsks: vi.fn() },
}));

import { conversationsClient, deliveryClient } from "../api/notifications";

const getTimeline = vi.mocked(deliveryClient.getTimeline);
const listAsks = vi.mocked(conversationsClient.listAsks);

describe("Updates page", () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it("scopes an empty state to the returned list and keeps API health separate", async () => {
    await setLocale("en");
    getTimeline.mockResolvedValue({ notifications: [] } as never);
    listAsks.mockResolvedValue({ asks: [] } as never);

    renderWithProviders(<DashboardPage />);

    const page = await screen.findByTestId(selectors.pages.dashboard);
    expect(page).toHaveTextContent(/No notifications were returned in this request’s scope/);
    expect(page).toHaveTextContent(/This does not indicate producer health/);
    expect(screen.getByTestId("health-card")).toBeInTheDocument();
    expect(listAsks).toHaveBeenCalledWith({ openOnly: false, limit: 50 });
  });

  it("shows informational content and the seven native states without read or acknowledgement controls", async () => {
    await setLocale("en");
    getTimeline.mockResolvedValue({
      notifications: [
        { id: "delivered", title: "Welcome", body: "Hello", requestedBy: "agent-manager", createdAt: "2026-10-20T14:00:00Z", urgency: "normal", sensitivityLabel: "public", reason: "accepted by adapter", state: NotificationState.DELIVERED },
        { id: "pending", title: "Pending item", body: "Later", state: NotificationState.PENDING },
        { id: "held", title: "Held item", body: "Waiting", state: NotificationState.HELD },
        { id: "routed", title: "Routed item", body: "Ready", state: NotificationState.ROUTED },
        { id: "failed", title: "Failed item", body: "Error", state: NotificationState.FAILED },
        { id: "unroutable", title: "Unroutable item", body: "No route", state: NotificationState.UNROUTABLE },
        { id: "suppressed", title: "Suppressed item", body: "Muted", state: NotificationState.SUPPRESSED },
        { id: "unspecified", title: "No context", body: "Default", state: NotificationState.UNSPECIFIED },
        { id: "unknown", title: "Unknown item", body: "Future value", state: 99 as NotificationState },
      ],
    } as never);
    listAsks.mockResolvedValue({
      asks: [{ id: "ask-1", notificationId: "delivered", state: "pending" }],
    } as never);

    renderWithProviders(<DashboardPage />);

    const page = await screen.findByTestId(selectors.pages.dashboard);
    for (const value of ["Welcome", "Hello", "agent-manager", "Pending", "Held", "Routed", "Delivered", "Failed", "Unroutable", "Suppressed", "Unknown status", "At least one adapter accepted the payload."]) {
      expect(page).toHaveTextContent(value);
    }
    expect(page).toHaveTextContent(/Unknown status.*Unknown status/s);
    expect(screen.getByRole("link", { name: "Open decision" })).toHaveAttribute("href", "/asks/ask-1");
    expect(screen.getAllByRole("link", { name: "Delivery details" })).toHaveLength(9);
    expect(screen.queryByRole("link", { name: /Open decision/ })).toHaveAttribute("href", "/asks/ask-1");
    expect(screen.queryByRole("button", { name: /read|acknowledge/i })).not.toBeInTheDocument();
    expect(screen.queryByText(/timeline/i)).not.toBeInTheDocument();
  });

  it("does not invent an Open decision link when no returned ask matches the notification", async () => {
    await setLocale("en");
    getTimeline.mockResolvedValue({
      notifications: [{ id: "notification-1", title: "Informational update", body: "Details", state: NotificationState.PENDING }],
    } as never);
    listAsks.mockResolvedValue({ asks: [{ id: "unrelated", notificationId: "different-notification", state: "pending" }] } as never);

    renderWithProviders(<DashboardPage />);

    expect(await screen.findByTestId(selectors.pages.dashboard)).toHaveTextContent("Informational update");
    expect(screen.queryByRole("link", { name: "Open decision" })).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Delivery details" })).toHaveAttribute("href", "/delivery/notification-1");
  });

  it("keeps updates visible when linked-ask lookup fails and does not offer an unverified decision link", async () => {
    await setLocale("en");
    getTimeline.mockResolvedValue({
      notifications: [{ id: "notification-1", title: "Informational update", body: "Details", state: NotificationState.PENDING }],
    } as never);
    listAsks.mockRejectedValue(new Error("offline"));

    renderWithProviders(<DashboardPage />);

    const page = await screen.findByTestId(selectors.pages.dashboard);
    expect(page).toHaveTextContent("Informational update");
    expect(page).toHaveTextContent("Decision links could not be checked, so no decision actions are shown.");
    expect(screen.queryByRole("link", { name: "Open decision" })).not.toBeInTheDocument();
  });

  it("reports an unavailable feed without naming producer health", async () => {
    await setLocale("en");
    getTimeline.mockRejectedValue(new Error("offline"));
    listAsks.mockResolvedValue({ asks: [] } as never);

    renderWithProviders(<DashboardPage />);

    expect(await screen.findByTestId(selectors.pages.dashboard)).toHaveTextContent("Updates could not be loaded.");
  });
});
