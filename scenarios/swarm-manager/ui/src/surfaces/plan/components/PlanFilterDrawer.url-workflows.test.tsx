import { useState } from "react";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { useLocation, useNavigate } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils";
import { defaultApiClient } from "../../../lib/api-client";
import { operationsStoreInitialFilters, useOperationsStore } from "../../../stores/operations-store";
import { createPlanDataInitialState, resetPlanRequestState, usePlanDataStore } from "../stores/plan-data-store";
import { usePlanUrlState } from "../hooks/usePlanUrlState";
import { PlanFilterDrawer } from "./PlanFilterDrawer";

function Owner() {
  const state = usePlanUrlState(); const location = useLocation(); const navigate = useNavigate();
  const [open, setOpen] = useState(true);
  return <><PlanFilterDrawer isOpen={open} onClose={() => setOpen(false)} filters={state.filters}
    viewMode={state.viewMode} showSnoozed={state.showSnoozed} hasActiveFilters={state.hasFilters}
    onFiltersChange={state.setFilters} onViewModeChange={state.setViewMode}
    onShowSnoozedChange={state.setShowSnoozed} onReset={state.resetFilters} />
    <output data-testid="location">{location.pathname}{location.search}</output>
    <button onClick={() => setOpen(true)}>Reopen filters</button>
    <button onClick={() => navigate("/plan?keep=second&status=failed&lane=review&owner_type=session&q=next&view=by-phase&show_snoozed=1&goal=fixture-next")}>Open other share</button>
  </>;
}
function params() { return new URLSearchParams(screen.getByTestId("location").textContent?.split("?")[1]); }
const initialOperations = useOperationsStore.getState(); const initialPlan = usePlanDataStore.getState();
beforeEach(() => {
  resetPlanRequestState();
  useOperationsStore.setState({filters: {...operationsStoreInitialFilters}, viewMode: "by-milestone"});
  usePlanDataStore.setState(createPlanDataInitialState());
  vi.spyOn(defaultApiClient, "get").mockImplementation(async (path) => {
    expect(["/plan?window_seconds=21600", "/plan?window_seconds=21600&goal=fixture-goal", "/plan?window_seconds=86400&goal=fixture-goal", "/plan?window_seconds=86400&goal=fixture-next"]).toContain(path);
    return {} as never;
  });
  for (const method of ["post", "put", "patch", "delete"] as const)
    vi.spyOn(defaultApiClient, method).mockImplementation(async () => { throw new Error("Unexpected owner request " + method); });
  vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("Unexpected transport")));
});
afterEach(() => {
  cleanup(); resetPlanRequestState();
  try {
    for (const method of ["post", "put", "patch", "delete"] as const) expect(defaultApiClient[method]).not.toHaveBeenCalled();
    for (const [path] of vi.mocked(defaultApiClient.get).mock.calls)
      expect(["/plan?window_seconds=21600", "/plan?window_seconds=21600&goal=fixture-goal", "/plan?window_seconds=86400&goal=fixture-goal", "/plan?window_seconds=86400&goal=fixture-next"]).toContain(path);
    expect(fetch).not.toHaveBeenCalled();
  } finally {
    useOperationsStore.setState(initialOperations, true); usePlanDataStore.setState(initialPlan, true);
    vi.restoreAllMocks(); vi.unstubAllGlobals();
  }
});

function mount(url = "/plan?keep=fixture&goal=fixture-goal&window_seconds=21600") { renderWithProviders(<Owner />, {initialEntries:[url]}); }
describe("actual Plan filter drawer and shareable owner URL", () => {
  it("hydrates real controls from a shared URL while keeping goal/window and unrelated keys", async () => {
    mount("/plan?keep=fixture&goal=fixture-goal&window_seconds=21600&status=needs_review&lane=review&owner_type=backlog&q=owner&view=by-phase&show_snoozed=1");
    await waitFor(() => expect(screen.getByTestId("plan-filter-search")).toHaveValue("owner"));
    expect(screen.getByTestId("plan-filter-status")).toHaveValue("needs_review"); expect(screen.getByTestId("plan-filter-lane")).toHaveValue("review");
    expect(screen.getByTestId("plan-filter-owner-type")).toHaveValue("backlog"); expect(screen.getByTestId("plan-filter-group-by")).toHaveValue("by-phase");
    expect(screen.getByTestId("plan-filter-show-snoozed")).toBeChecked(); expect(params().get("keep")).toBe("fixture");
    expect(usePlanDataStore.getState().goal).toBe("fixture-goal"); expect(usePlanDataStore.getState().windowSeconds).toBe(21600);
  });
  it.each([["status","status","running"],["lane","lane","execute"],["owner-type","owner_type","scenario"]])("changes and clears %s through real stores and URL", async (control, key, value) => {
    mount(); await waitFor(() => expect(usePlanDataStore.getState().goal).toBe("fixture-goal"));
    fireEvent.change(screen.getByTestId("plan-filter-"+control), {target:{value}});
    await waitFor(() => expect(params().get(key)).toBe(value)); expect(params().get("goal")).toBe("fixture-goal"); expect(params().get("keep")).toBe("fixture");
    fireEvent.change(screen.getByTestId("plan-filter-"+control), {target:{value:""}});
    await waitFor(() => expect(params().has(key)).toBe(false)); expect(params().get("window_seconds")).toBe("21600");
  });
  it("persists escaped search text and clears it without losing owner scope", async () => {
    mount(); await waitFor(() => expect(usePlanDataStore.getState().goal).toBe("fixture-goal"));
    fireEvent.change(screen.getByTestId("plan-filter-search"), {target:{value:"owner & context"}});
    await waitFor(() => expect(params().get("q")).toBe("owner & context"));
    expect(useOperationsStore.getState().filters.q).toBe("owner & context");
    fireEvent.change(screen.getByTestId("plan-filter-search"), {target:{value:""}});
    await waitFor(() => expect(params().has("q")).toBe(false)); expect(params().get("goal")).toBe("fixture-goal");
  });
  it("changes grouping and snoozed visibility then resets only the offered filter fields", async () => {
    mount(); await waitFor(() => expect(usePlanDataStore.getState().goal).toBe("fixture-goal"));
    fireEvent.change(screen.getByTestId("plan-filter-group-by"), {target:{value:"by-phase"}});
    await waitFor(() => expect(params().get("view")).toBe("by-phase"));
    fireEvent.click(screen.getByTestId("plan-filter-show-snoozed")); await waitFor(() => expect(params().get("show_snoozed")).toBe("1"));
    fireEvent.change(screen.getByTestId("plan-filter-search"), {target:{value:"filter"}});
    fireEvent.click(await screen.findByTestId("plan-filter-reset"));
    await waitFor(() => { expect(params().has("q")).toBe(false); expect(params().has("view")).toBe(false); expect(params().has("show_snoozed")).toBe(false); });
    expect(params().get("goal")).toBe("fixture-goal"); expect(params().get("window_seconds")).toBe("21600"); expect(params().get("keep")).toBe("fixture");
    expect(screen.queryByTestId("plan-filter-reset")).not.toBeInTheDocument();
  });
  it("closes and reopens the actual panel without resetting the shared filter", async () => {
    mount("/plan?keep=fixture&q=preserved"); await waitFor(() => expect(screen.getByTestId("plan-filter-search")).toHaveValue("preserved"));
    fireEvent.click(screen.getByRole("button", {name:"Close panel"})); await waitFor(() => expect(screen.queryByTestId("plan-filter-drawer")).not.toBeInTheDocument());
    expect(params().get("q")).toBe("preserved"); fireEvent.click(screen.getByRole("button", {name:"Reopen filters"}));
    expect(await screen.findByTestId("plan-filter-search")).toHaveValue("preserved");
  });
  it("loads a second share into mounted controls without writing stale first-owner state", async () => {
    mount(); await waitFor(() => expect(usePlanDataStore.getState().goal).toBe("fixture-goal"));
    fireEvent.click(screen.getByRole("button", {name:"Open other share"}));
    await waitFor(() => expect(screen.getByTestId("plan-filter-search")).toHaveValue("next"));
    expect(screen.getByTestId("plan-filter-status")).toHaveValue("failed"); expect(screen.getByTestId("plan-filter-owner-type")).toHaveValue("session");
    expect(params().get("keep")).toBe("second"); expect(params().get("goal")).toBe("fixture-next"); expect(params().get("q")).toBe("next");
  });
  it("normalizes invalid shared filters to visible defaults and preserves unrelated parameters", async () => {
    mount("/plan?keep=fixture&status=invalid&lane=invalid&owner_type=invalid&view=invalid&window_seconds=9&show_snoozed=yes");
    await waitFor(() => expect(params().has("status")).toBe(false));
    for (const id of ["status","lane","owner-type"]) expect(screen.getByTestId("plan-filter-"+id)).toHaveValue("");
    expect(screen.getByTestId("plan-filter-group-by")).toHaveValue("by-milestone"); expect(screen.getByTestId("plan-filter-show-snoozed")).not.toBeChecked();
    expect(params().get("keep")).toBe("fixture"); expect(usePlanDataStore.getState().windowSeconds).toBe(86400);
  });
});
