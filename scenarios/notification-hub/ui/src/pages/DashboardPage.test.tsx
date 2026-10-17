import { afterEach, describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
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
        { id: "delivered", title: "Welcome", body: "Hello", state: { toString: () => "NOTIFICATION_STATE_DELIVERED" } },
        { id: "queued", title: "Queued", body: "Later", state: { toString: () => "NOTIFICATION_STATE_QUEUED" } },
      ],
    } as never);

    renderWithProviders(<DashboardPage />);

    expect(await screen.findByText("1/2")).toBeInTheDocument();
    expect(screen.getByText("Welcome")).toBeInTheDocument();
    expect(screen.getByText("queued")).toBeInTheDocument();
  });

  it("reports unavailable delivery queries", async () => {
    getTimeline.mockRejectedValue(new Error("offline"));

    renderWithProviders(<DashboardPage />);

    expect(await screen.findByText("Delivery summary unavailable.")).toBeInTheDocument();
    expect(screen.getByText("Timeline unavailable — check the API identity and scenario health.")).toBeInTheDocument();
  });
});
