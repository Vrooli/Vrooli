import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";

import { DrillsPage } from "./DrillsPage";
import { renderWithProviders } from "../test-utils";
import { strings } from "../consts/strings";
import { selectors } from "../consts/selectors";

vi.mock("../hooks/usePlans", () => ({
  usePlans: vi.fn(() => ({ data: [
    { id: "plan-1", name: "critical", enabled: true, targetIds: ["target-1", "target-1b"], destinationIds: ["dest-1", "dest-1b"], recoveryDrillSchedule: "168h" },
    { id: "plan-2", name: "failed", enabled: true, targetIds: ["target-2"], destinationIds: ["dest-2"], recoveryDrillSchedule: "24h" },
    { id: "plan-3", name: "running", enabled: true, targetIds: ["target-3"], destinationIds: ["dest-3"], recoveryDrillSchedule: "24h" },
    { id: "plan-4", name: "requested", enabled: true, targetIds: ["target-4"], destinationIds: ["dest-4"], recoveryDrillSchedule: "24h" },
  ], isLoading: false, isError: false, refetch: vi.fn() })),
}));
vi.mock("../hooks/useTargets", () => ({
  useTargets: vi.fn(() => ({ data: [
    { id: "target-1", name: "source-one" },
    { id: "target-1b", name: "source-two" },
    { id: "target-2", name: "source-three" },
    { id: "target-3", name: "source-four" },
    { id: "target-4", name: "source-five" },
  ], isLoading: false, isError: false, refetch: vi.fn() })),
}));
vi.mock("../hooks/useDestinations", () => ({
  useDestinations: vi.fn(() => ({ data: [
    { id: "dest-1", name: "vault-one" },
    { id: "dest-1b", name: "vault-two" },
    { id: "dest-2", name: "vault-three" },
    { id: "dest-3", name: "vault-four" },
    { id: "dest-4", name: "vault-five" },
  ], isLoading: false, isError: false, refetch: vi.fn() })),
}));
vi.mock("../hooks/useDrills", () => ({
  useDrills: vi.fn(() => ({ data: [
    { id: "drill-1", planId: "plan-1", targetId: "target-1", destinationId: "dest-1", status: 3, scheduled: true, requestedAt: timestampFromDate(new Date()), finishedAt: timestampFromDate(new Date()) },
    { id: "drill-2", planId: "plan-2", targetId: "target-2", destinationId: "dest-2", status: 4, scheduled: true, requestedAt: timestampFromDate(new Date()) },
    { id: "drill-3", planId: "plan-3", targetId: "target-3", destinationId: "dest-3", status: 2, scheduled: true, requestedAt: timestampFromDate(new Date()) },
    { id: "drill-4", planId: "plan-4", targetId: "target-4", destinationId: "dest-4", status: 1, scheduled: true, requestedAt: timestampFromDate(new Date()) },
  ], isLoading: false, isError: false, refetch: vi.fn() })),
  useRunDrill: vi.fn(() => ({ isPending: false, mutate: vi.fn() })),
}));

import * as planHooks from "../hooks/usePlans";
import * as targetHooks from "../hooks/useTargets";
import * as destinationHooks from "../hooks/useDestinations";
import * as drillHooks from "../hooks/useDrills";

describe("DrillsPage", () => {
  it("renders durable drill status and its operator action", () => {
    renderWithProviders(<DrillsPage />);
    expect(screen.getByTestId(selectors.pages.drills)).toBeInTheDocument();
    expect(screen.getByText(strings.status.drill.verified)).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: strings.drills.weeklyTitle })).toBeInTheDocument();
    expect(screen.getByText(strings.drills.weeklyIncomplete)).toBeInTheDocument();
    expect(screen.getByText(strings.drills.weeklyVerified)).toBeInTheDocument();
    expect(screen.getByText(strings.drills.weeklyFailed)).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: strings.drills.run })).toHaveLength(7);
    expect(screen.getByText(/source-two \[target-1b\] → vault-two \[dest-1b\]/)).toBeInTheDocument();
  });

  it("renders incomplete pairs and keeps manual or disabled pairs out of the weekly denominator", () => {
    vi.mocked(planHooks.usePlans).mockReturnValue({
      data: [
        { id: "manual", name: "manual", enabled: true, targetIds: ["t1"], destinationIds: ["d1"], recoveryDrillSchedule: "" },
        { id: "disabled", name: "disabled", enabled: false, targetIds: ["t2"], destinationIds: ["d2"], recoveryDrillSchedule: "24h" },
        { id: "empty", name: "empty", enabled: true, targetIds: [], destinationIds: [], recoveryDrillSchedule: "24h" },
      ], isLoading: false, isError: false, refetch: vi.fn(),
    } as never);
    vi.mocked(drillHooks.useDrills).mockReturnValue({ data: [], isLoading: false, isError: false, refetch: vi.fn() } as never);
    vi.mocked(targetHooks.useTargets).mockReturnValue({ data: [], isLoading: false, isError: false, refetch: vi.fn() } as never);
    vi.mocked(destinationHooks.useDestinations).mockReturnValue({ data: [], isLoading: false, isError: false, refetch: vi.fn() } as never);

    renderWithProviders(<DrillsPage />);
    expect(screen.getByText(strings.drills.weeklyNoPairs)).toBeInTheDocument();
    expect(screen.getByText(strings.drills.noUnits)).toBeInTheDocument();
    expect(screen.getByText(/t1 \[t1\].*d1 \[d1\]/)).toBeInTheDocument();
    expect(screen.getByText(/t2 \[t2\].*d2 \[d2\]/)).toBeInTheDocument();
    expect(screen.getByText(strings.drills.weeklyManual)).toBeInTheDocument();
    expect(screen.getByText(strings.drills.weeklyDisabled)).toBeInTheDocument();
  });

  it("renders notification state variants and disables drill actions while a run is pending", () => {
    const now = timestampFromDate(new Date());
    vi.mocked(planHooks.usePlans).mockReturnValue({
      data: [{ id: "notifications", name: "notifications", enabled: true, targetIds: ["t1", "t2", "t3"], destinationIds: ["d1"], recoveryDrillSchedule: "24h" }],
      isLoading: false, isError: false, refetch: vi.fn(),
    } as never);
    vi.mocked(drillHooks.useDrills).mockReturnValue({
      data: [
        { id: "pending", planId: "notifications", targetId: "t1", destinationId: "d1", status: 4, scheduled: true, requestedAt: now, notificationState: "pending" },
        { id: "sent", planId: "notifications", targetId: "t2", destinationId: "d1", status: 4, scheduled: true, requestedAt: now, notificationState: "sent" },
        { id: "untracked", planId: "notifications", targetId: "t3", destinationId: "d1", status: 4, scheduled: true, requestedAt: now, notificationState: "unknown" },
      ], isLoading: false, isError: false, refetch: vi.fn(),
    } as never);
    vi.mocked(targetHooks.useTargets).mockReturnValue({ data: [], isLoading: false, isError: false, refetch: vi.fn() } as never);
    vi.mocked(destinationHooks.useDestinations).mockReturnValue({ data: [], isLoading: false, isError: false, refetch: vi.fn() } as never);
    vi.mocked(drillHooks.useRunDrill).mockReturnValue({ isPending: true, mutate: vi.fn() } as never);

    renderWithProviders(<DrillsPage />);
    expect(screen.getByText(strings.drills.notificationPending)).toBeInTheDocument();
    expect(screen.getByText(strings.drills.notificationSent)).toBeInTheDocument();
    expect(screen.getByText(strings.drills.notificationUntracked)).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: strings.drills.run }).every((button) => (button as HTMLButtonElement).disabled)).toBe(true);
  });

  it("shows a green week only when every scheduled pair has a recent verified receipt", () => {
    const now = timestampFromDate(new Date());
    vi.mocked(planHooks.usePlans).mockReturnValue({
      data: [{ id: "green", name: "green", enabled: true, targetIds: ["target-green"], destinationIds: ["dest-green"], recoveryDrillSchedule: "168h" }],
      isLoading: false, isError: false, refetch: vi.fn(),
    } as never);
    vi.mocked(drillHooks.useDrills).mockReturnValue({
      data: [{ id: "verified", planId: "green", targetId: "target-green", destinationId: "dest-green", status: 3, scheduled: true, requestedAt: now, finishedAt: now }],
      isLoading: false, isError: false, refetch: vi.fn(),
    } as never);
    vi.mocked(targetHooks.useTargets).mockReturnValue({ data: undefined, isLoading: false, isError: false, refetch: vi.fn() } as never);
    vi.mocked(destinationHooks.useDestinations).mockReturnValue({ data: undefined, isLoading: false, isError: false, refetch: vi.fn() } as never);

    renderWithProviders(<DrillsPage />);
    expect(screen.getByText(strings.drills.weeklyAllGreen)).toBeInTheDocument();
    expect(screen.getByText(strings.drills.weeklyVerified)).toBeInTheDocument();
    expect(screen.getByText(/target-green \[target-green\].*dest-green \[dest-green\]/)).toBeInTheDocument();
  });

  it("shows loading and error states and retries each failed data source", () => {
    const refetch = vi.fn();
    vi.mocked(planHooks.usePlans).mockReturnValue({ data: undefined, isLoading: true, isError: false, refetch } as never);
    vi.mocked(drillHooks.useDrills).mockReturnValue({ data: undefined, isLoading: false, isError: false, refetch } as never);
    vi.mocked(targetHooks.useTargets).mockReturnValue({ data: undefined, isLoading: false, isError: false, refetch } as never);
    vi.mocked(destinationHooks.useDestinations).mockReturnValue({ data: undefined, isLoading: false, isError: false, refetch } as never);
    const { rerender } = renderWithProviders(<DrillsPage />);
    expect(screen.getByTestId(selectors.async.loading)).toBeInTheDocument();

    vi.mocked(planHooks.usePlans).mockReturnValue({ data: undefined, isLoading: false, isError: true, refetch } as never);
    vi.mocked(drillHooks.useDrills).mockReturnValue({ data: undefined, isLoading: false, isError: false, refetch } as never);
    vi.mocked(targetHooks.useTargets).mockReturnValue({ data: undefined, isLoading: false, isError: false, refetch } as never);
    vi.mocked(destinationHooks.useDestinations).mockReturnValue({ data: undefined, isLoading: false, isError: false, refetch } as never);
    rerender(<DrillsPage />);
    fireEvent.click(screen.getByTestId(selectors.async.retry));
    expect(refetch).toHaveBeenCalledTimes(4);
  });
});
