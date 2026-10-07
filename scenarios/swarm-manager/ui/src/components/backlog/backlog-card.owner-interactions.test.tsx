import { useState } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Link, useLocation } from "react-router-dom";
import { getSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { renderWithProviders } from "../../test-utils";
import { defaultApiClient } from "../../lib/api-client";
import { BacklogCard, type BacklogCardProps } from "./backlog-card";
import type { BacklogItem, BacklogStatus } from "../../types";
import type { ItemActions } from "../../lib/backlog-queue-utils";

// Controlled parent callback contract, real card/widgets/router only. No queue,
// status, snooze or selection response constitutes owner admission or a write.
const item: BacklogItem = { name: "fixture-card", title: "Owner card", description: "Local intent fixture", kind: "idea", status: "ready", priority: 2, tags: ["fixture"], suggestedSkills: [], created: "2026-03-20T00:00:00Z", updated: "2026-03-20T00:00:00Z" };
const actions: ItemActions = { locked: false, terminal: false, blocked: false, blockingDepKeys: [], primaryCta: null, canRun: false, runDisabled: false, canFollowUp: false, canRetry: false, canArchive: false, showDecisionStepper: false, agentRunning: false, notQueueableReason: null, disabledReason: null };
const projected: NonNullable<BacklogCardProps["nextAction"]> = { id: "author_plan", compactLabel: "Plan", expandedLabel: "Author owner plan", enabled: true, blockers: [] };
let views: { unmount: () => void }[] = [];
let next: ReturnType<typeof vi.fn>;
let status: ReturnType<typeof vi.fn>;
let toggle: ReturnType<typeof vi.fn>;
let snooze: ReturnType<typeof vi.fn>;
let parentClick: ReturnType<typeof vi.fn>;
let expectedNext: unknown[][];
let expectedStatus: unknown[][];
let expectedToggle: unknown[][];
let expectedSnooze: unknown[][];
function Location() { const location = useLocation(); return <output data-testid="card-local-location">{location.pathname}</output>; }
function Harness({ overrides = {} }: { overrides?: Partial<BacklogCardProps> }) {
  const [selected, setSelected] = useState(false);
  const [currentStatus, setCurrentStatus] = useState<BacklogStatus>(item.status);
  return <><Link to="/unintended-card-navigation" onClick={parentClick}><BacklogCard item={{ ...item, status: currentStatus }} itemActions={actions} nextAction={projected} onNextAction={next} onStatusChange={value => { status(value); setCurrentStatus(value); }} batchMode isSelected={selected} onToggleSelection={() => { toggle(); setSelected(value => !value); }} onSnooze={snooze} {...overrides} /></Link><Location /></>;
}
function mount(overrides?: Partial<BacklogCardProps>) { const view = renderWithProviders(<Harness overrides={overrides} />, { initialEntries: ["/fixture-cards"] }); views.push(view); return view; }
function locationUnchanged() { expect(screen.getByTestId("card-local-location")).toHaveTextContent("/fixture-cards"); }
beforeEach(() => {
  next = vi.fn(); status = vi.fn(); toggle = vi.fn(); snooze = vi.fn(); parentClick = vi.fn();
  expectedNext = []; expectedStatus = []; expectedToggle = []; expectedSnooze = [];
  for (const method of ["get", "post", "put", "patch", "delete"] as const) vi.spyOn(defaultApiClient, method).mockRejectedValue(new Error(`Forbidden local card ${method}`));
  vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("Forbidden card transport")));
});
afterEach(() => {
  try {
    views.splice(0).forEach(view => view.unmount()); cleanup(); getSpatialNav()?.dispose();
    expect(next.mock.calls).toEqual(expectedNext); expect(status.mock.calls).toEqual(expectedStatus);
    expect(toggle.mock.calls).toEqual(expectedToggle); expect(snooze.mock.calls).toEqual(expectedSnooze);
    expect(parentClick).not.toHaveBeenCalled(); expect(fetch).not.toHaveBeenCalled();
    for (const method of ["get", "post", "put", "patch", "delete"] as const) expect(defaultApiClient[method]).not.toHaveBeenCalled();
  } finally { vi.restoreAllMocks(); vi.unstubAllGlobals(); }
});
describe("Backlog card local owner interactions", () => {
  it("changes batch selection exactly once per checkbox click without following its parent link", async () => {
    mount(); const checkbox = screen.getByRole("checkbox", { name: "Select backlog item Owner card" });
    expect(checkbox).not.toBeChecked(); expectedToggle = [[], []];
    await userEvent.click(checkbox); expect(checkbox).toBeChecked(); locationUnchanged();
    await userEvent.click(checkbox); expect(checkbox).not.toBeChecked(); locationUnchanged();
  });
  it("offers only the projected local primary intent without following the parent link", async () => {
    mount({ batchMode: false }); expectedNext = [[]];
    await userEvent.click(screen.getByRole("button", { name: "Author owner plan" }));
    expect(next).toHaveBeenCalledOnce(); locationUnchanged();
  });
  it("preserves a disabled projection reason without any action or parent navigation", async () => {
    mount({ batchMode: false, nextAction: { ...projected, enabled: false, reason: "Owner admission held" } });
    const button = screen.getByRole("button", { name: "Author owner plan" });
    expect(button).toBeDisabled(); expect(button).toHaveAttribute("title", "Owner admission held");
    await userEvent.click(button); locationUnchanged();
  });
  it("dismisses a current-status selection without emitting a redundant status intent", async () => {
    mount({ batchMode: false }); await userEvent.click(screen.getByTestId("status-chip-trigger"));
    await userEvent.click(screen.getByTestId("status-option-ready"));
    expect(screen.queryByTestId("status-chip-popover")).toBeNull(); locationUnchanged();
  });
  it("reflects the exact selected status in its controlled parent without a transport write", async () => {
    mount({ batchMode: false }); expectedStatus = [["backlog"]];
    await userEvent.click(screen.getByTestId("status-chip-trigger"));
    await userEvent.click(screen.getByTestId("status-option-backlog"));
    expect(screen.getByTestId("status-chip-trigger")).toHaveTextContent("Backlog");
    expect(screen.queryByTestId("status-chip-popover")).toBeNull(); locationUnchanged();
  });
  it("holds the status menu closed while the parent status request is pending", async () => {
    mount({ batchMode: false, statusChangePending: true });
    await userEvent.click(screen.getByTestId("status-chip-trigger"));
    expect(screen.queryByTestId("status-chip-popover")).toBeNull(); locationUnchanged();
  });
  it("dismisses the status menu with Escape without changing status", async () => {
    mount({ batchMode: false }); await userEvent.click(screen.getByTestId("status-chip-trigger"));
    expect(screen.getByTestId("status-chip-popover")).toBeVisible();
    await userEvent.keyboard("{Escape}"); expect(screen.queryByTestId("status-chip-popover")).toBeNull(); locationUnchanged();
  });
  it("offers an exact one-hour snooze key and expiry from the primary action row", async () => {
    vi.spyOn(Date, "now").mockReturnValue(1_800_000_000_000);
    mount({ batchMode: false, showSnooze: true }); expectedSnooze = [["backlog:idea/fixture-card", 1_800_003_600_000]];
    await userEvent.click(screen.getByTestId("snooze-trigger"));
    await userEvent.click(screen.getByTestId("snooze-preset-1-hour"));
    expect(screen.queryByTestId("snooze-popover")).toBeNull(); locationUnchanged();
  });
  it("offers the four-hour snooze independently when the owner projects no primary action", async () => {
    vi.spyOn(Date, "now").mockReturnValue(1_800_000_000_000);
    mount({ batchMode: false, showSnooze: true, nextAction: { id: "none", compactLabel: "", expandedLabel: "", enabled: false, blockers: [] } });
    expect(screen.queryByTestId("backlog-card-actions")).toBeNull(); expectedSnooze = [["backlog:idea/fixture-card", 1_800_014_400_000]];
    await userEvent.click(screen.getByTestId("snooze-trigger"));
    await userEvent.click(screen.getByTestId("snooze-preset-4-hours")); locationUnchanged();
  });
  it("cancels snooze with Escape without creating a local expiry intent", async () => {
    mount({ batchMode: false, showSnooze: true });
    await userEvent.click(screen.getByTestId("snooze-trigger")); expect(screen.getByTestId("snooze-popover")).toBeVisible();
    await userEvent.keyboard("{Escape}"); expect(screen.queryByTestId("snooze-popover")).toBeNull(); locationUnchanged();
  });
});
