import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";

vi.mock("../../api/delivery", () => ({
  fetchReadiness: vi.fn(),
  fetchTargets: vi.fn(),
  fetchConformancePlan: vi.fn(),
}));

import { fetchConformancePlan, fetchReadiness, fetchTargets } from "../../api/delivery";
import { DeliveryReadinessCard } from "./DeliveryReadinessCard";

const mockedFetchReadiness = vi.mocked(fetchReadiness);
const mockedFetchTargets = vi.mocked(fetchTargets);
const mockedFetchConformancePlan = vi.mocked(fetchConformancePlan);

describe("DeliveryReadinessCard", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders ready delivery state, targets, and conformance chapter count", async () => {
    mockedFetchReadiness.mockResolvedValue({
      rungs: [{ id: "build", title: "Build", state: "ready", next_action: "Ship" }],
    });
    mockedFetchTargets.mockResolvedValue({
      targets: [{ id: "simulator", label: "iOS simulator", available: true }],
    });
    mockedFetchConformancePlan.mockResolvedValue({
      chapters: [{ id: "proto" }, { id: "ui" }],
    });

    renderWithProviders(<DeliveryReadinessCard />);

    await waitFor(() => expect(screen.getByText(/iOS simulator/)).toBeInTheDocument());
    expect(screen.getByText(/2/)).toBeInTheDocument();
    expect(screen.getAllByText(/ready/).length).toBeGreaterThanOrEqual(2);
  });

  it("keeps blocked readiness and target limitations visible", async () => {
    mockedFetchReadiness.mockResolvedValue({
      rungs: [{
        id: "apple",
        title: "Apple signing",
        state: "blocked",
        next_action: "Use a macOS signing host",
        missing_capability: "Apple hardware",
      }],
    });
    mockedFetchTargets.mockResolvedValue({
      targets: [{ id: "hardware", label: "Apple hardware", available: false, next_action: "Connect a Mac" }],
    });
    mockedFetchConformancePlan.mockResolvedValue({ chapters: [] });

    renderWithProviders(<DeliveryReadinessCard />);

    await waitFor(() => expect(screen.getByText(/Apple signing/)).toBeInTheDocument());
    expect(screen.getByText(/Use a macOS signing host/)).toBeInTheDocument();
    expect(screen.getAllByText(/Apple hardware/).length).toBeGreaterThanOrEqual(2);
    expect(screen.getByText(/Connect a Mac/)).toBeInTheDocument();
  });

  it("reports unavailable delivery reports instead of implying readiness", async () => {
    mockedFetchReadiness.mockRejectedValue(new Error("readiness unavailable"));
    mockedFetchTargets.mockRejectedValue(new Error("targets unavailable"));
    mockedFetchConformancePlan.mockRejectedValue(new Error("plan unavailable"));

    renderWithProviders(<DeliveryReadinessCard />);

    await waitFor(() => {
      expect(screen.getAllByText(strings.delivery.unavailable)).toHaveLength(3);
    });
  });
});
