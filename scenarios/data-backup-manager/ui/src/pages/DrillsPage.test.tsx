import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { DrillsPage } from "./DrillsPage";
import { renderWithProviders } from "../test-utils";
import { strings } from "../consts/strings";
import { selectors } from "../consts/selectors";

vi.mock("../hooks/usePlans", () => ({
  usePlans: () => ({ data: [
    { id: "plan-1", name: "critical", targetIds: ["target-1"], destinationIds: ["dest-1"], recoveryDrillSchedule: "168h" },
    { id: "plan-2", name: "failed", targetIds: ["target-2"], destinationIds: ["dest-2"], recoveryDrillSchedule: "" },
    { id: "plan-3", name: "running", targetIds: ["target-3"], destinationIds: ["dest-3"], recoveryDrillSchedule: "24h" },
    { id: "plan-4", name: "requested", targetIds: ["target-4"], destinationIds: ["dest-4"], recoveryDrillSchedule: "24h" },
  ], isLoading: false, isError: false, refetch: vi.fn() }),
}));
vi.mock("../hooks/useDrills", () => ({
  useDrills: () => ({ data: [
    { id: "drill-1", planId: "plan-1", targetId: "target-1", destinationId: "dest-1", status: 3, requestedAt: undefined },
    { id: "drill-2", planId: "plan-2", targetId: "target-2", destinationId: "dest-2", status: 4, requestedAt: undefined },
    { id: "drill-3", planId: "plan-3", targetId: "target-3", destinationId: "dest-3", status: 2, requestedAt: undefined },
    { id: "drill-4", planId: "plan-4", targetId: "target-4", destinationId: "dest-4", status: 1, requestedAt: undefined },
  ], isLoading: false, isError: false, refetch: vi.fn() }),
  useRunDrill: () => ({ isPending: false, mutate: vi.fn() }),
}));

describe("DrillsPage", () => {
  it("renders durable drill status and its operator action", () => {
    renderWithProviders(<DrillsPage />);
    expect(screen.getByTestId(selectors.pages.drills)).toBeInTheDocument();
    expect(screen.getByText(strings.status.drill.verified)).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: strings.drills.run })).toHaveLength(4);
  });
});
