import { afterEach, describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { NotificationState } from "@vrooli/proto-types/notification-hub/v1/shared/types_pb";
import { DashboardPage } from "./DashboardPage";

vi.mock("../features/health/HealthCard", () => ({
  HealthCard: () => <div data-testid="health-card" />,
}));

vi.mock("../api/notifications", () => ({
  deliveryClient: {
    getTimeline: vi.fn(),
  },
}));

import { deliveryClient } from "../api/notifications";

const getTimeline = vi.mocked(deliveryClient.getTimeline);

describe("DashboardPage delivery surfaces", () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it("shows empty states when there are no notifications", async () => {
    getTimeline.mockResolvedValue({ notifications: [] } as never);

    renderWithProviders(<DashboardPage />);

    expect(await screen.findByText("No notifications yet. Register this device, then send one from the CLI.")).toBeInTheDocument();
    expect(screen.getByText("--")).toBeInTheDocument();
  });

  it("summarizes delivered notifications and renders the timeline", async () => {
    getTimeline.mockResolvedValue({
      notifications: [
        { id: "delivered", title: "Welcome", body: "Hello", contextUrl: "https://agent-manager.example/runs/run-42", state: NotificationState.DELIVERED },
        { id: "pending", title: "Pending item", body: "Later", contextUrl: "javascript:alert(1)", state: NotificationState.PENDING },
        { id: "held", title: "Held item", body: "Waiting", state: NotificationState.HELD },
        { id: "routed", title: "Routed item", body: "Ready", state: NotificationState.ROUTED },
        { id: "failed", title: "Failed item", body: "Error", state: NotificationState.FAILED },
        { id: "unroutable", title: "Unroutable item", body: "No route", state: NotificationState.UNROUTABLE },
        { id: "suppressed", title: "Suppressed item", body: "Muted", state: NotificationState.SUPPRESSED },
        { id: "unspecified", title: "No context", body: "Default", state: NotificationState.UNSPECIFIED },
        { id: "unknown", title: "Unknown item", body: "Future value", state: 99 as NotificationState },
      ],
    } as never);

    renderWithProviders(<DashboardPage />);

    expect(await screen.findByText("1/9")).toBeInTheDocument();
    expect(screen.getByText("delivered")).toBeInTheDocument();
    expect(screen.getByText("Welcome")).toBeInTheDocument();
    expect(screen.getByText("pending")).toBeInTheDocument();
    expect(screen.getByText("held")).toBeInTheDocument();
    expect(screen.getByText("routed")).toBeInTheDocument();
    expect(screen.getByText("failed")).toBeInTheDocument();
    expect(screen.getByText("unroutable")).toBeInTheDocument();
    expect(screen.getByText("suppressed")).toBeInTheDocument();
    expect(screen.getAllByText("unknown")).toHaveLength(2);
    expect(screen.getByText("No context")).toBeInTheDocument();
    const contextLink = screen.getByRole("link", { name: /Welcome/ });
    expect(contextLink).toHaveAttribute("href", "https://agent-manager.example/runs/run-42");
    expect(screen.queryByRole("link", { name: /Pending item/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /No context/ })).not.toBeInTheDocument();
  });

  it("reports unavailable delivery queries", async () => {
    getTimeline.mockRejectedValue(new Error("offline"));

    renderWithProviders(<DashboardPage />);

    expect(await screen.findByText("Delivery summary unavailable.")).toBeInTheDocument();
    expect(screen.getByText("Timeline unavailable — check the configured recipient and scenario health.")).toBeInTheDocument();
  });
});
