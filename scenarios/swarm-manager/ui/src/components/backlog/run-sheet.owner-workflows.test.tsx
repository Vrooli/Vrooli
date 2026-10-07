import { useState } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { Route, Routes, useLocation } from "react-router-dom";
import type { QueryClient } from "@tanstack/react-query";
import { getSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { RunSheet, type RunSheetTarget } from "./run-sheet";
import { ApiError, defaultApiClient } from "../../lib/api-client";
import { createTestQueryClient, renderWithProviders } from "../../test-utils";

// Outside-only desired contract. Actual RunSheet/service/protobuf translation.
// All POSTs are strict synthetic offers: even confirm:true never reaches a run.
// Current adapter passes unknown `strategy` instead of generated executionMode;
// desired exact-wire assertions deliberately retain that baseline regression.
const target: RunSheetTarget = { kind: "execute", name: "fixture-run", title: "Fixture reviewed work" };
const queuePath = "/backlog/execute/fixture-run/queue";
const previewBody = { operation: "generator", mode: "yolo", confirm: false };
const clients: QueryClient[] = [];
const unmounts: Array<() => void> = [];
const unexpected: string[] = [];
let expectedPosts: unknown[][];
let blockingReasons: Array<{ code: string; message: string; forceable: boolean }>;
let savedMode: string;
let nativeAvailable: boolean;
let catalogRefused: boolean;
let queueRefused: boolean;
let stalePlan: boolean;
let queueDeferred: boolean;
let releaseQueue: ((value: unknown) => void) | undefined;
const strategies = [
  { id: "sliced", display_name: "Reviewed slices", workflow_key: "fixture/slices", description: "Bounded reviewed slices", when_to_use: "Fixture ready work", cost_band: "fixture", cost_estimate: 1 },
  { id: "goal", display_name: "Reviewed goal session", workflow_key: "fixture/goal", description: "Bounded reviewed goal", when_to_use: "Fixture native objective", cost_band: "fixture", cost_estimate: 2 },
];
const preferences = { preferred_runner: "RUNNER_TYPE_CODEX", model: "fixture-codex-default", effort: "medium" };
function confirmed(overrides: Record<string, unknown> = {}) { return { operation: "generator", mode: "yolo", started_by: "swarm-manager-ui", confirm: true, force: false, execution_mode: "sliced", max_slices: 4, blocker_repair_policy: "in_scope_only", execution_preferences: preferences, ...overrides }; }
function preview() { return { dryRun: true, queued: false, message: "Fixture ready", blockingReasons, item: { kind: "execute", name: "fixture-run", title: "Fixture reviewed work", description: "Fixture specification", status: "ready", priority: 1, created: "2026-01-01T00:00:00Z", updated: "2026-01-01T00:00:00Z", executionMode: savedMode, executionLimits: { maxSlices: 4, maxTokens: "200000", maxWallSeconds: "300", maxTurns: 20, maxChargeMicroUsd: "2000000", maxChildren: 1, maxNodeAttempts: 4, maxRetries: 1 }, continuation: "manual", scopePolicy: "fixed" } }; }
function Location() { const location = useLocation(); return <output data-testid="run-sheet-owner-location">{location.pathname + location.search}</output>; }
function mount(options: { open?: boolean; targets?: RunSheetTarget[] } = {}) {
  const close = vi.fn(); const success = vi.fn(); const client = createTestQueryClient(); clients.push(client);
  function Parent() { const [open, setOpen] = useState(options.open ?? true); return <RunSheet isOpen={open} onClose={() => { close(); setOpen(false); }} target={options.targets ? undefined : target} targets={options.targets} onSuccess={success} />; }
  const view = renderWithProviders(<><Routes><Route path="/plan" element={<Parent />} /><Route path="/backlog/:kind/:name" element={<h1>Exact reviewed item destination</h1>} /></Routes><Location /></>, { queryClient: client, initialEntries: ["/plan?keep=fixture"] });
  unmounts.push(view.unmount); return { ...view, close, success };
}
async function ready() { await screen.findByText("Ready to start"); await waitFor(() => expect(screen.queryByText("Checking readiness…")).toBeNull()); return screen.getByRole("button", { name: "Start run" }); }
beforeEach(() => {
  expectedPosts = [[queuePath, previewBody]]; blockingReasons = []; savedMode = "sliced"; nativeAvailable = true; catalogRefused = false; queueRefused = false; stalePlan = false; queueDeferred = false; releaseQueue = undefined; unexpected.length = 0;
  vi.spyOn(defaultApiClient, "get").mockImplementation(async path => {
    if (path === "/execution/strategies?backlog_kind=execute&backlog_name=fixture-run" || path === "/execution/strategies") { if (catalogRefused) throw new Error("Fixture run catalog refused"); return { items: strategies }; }
    if (path === "/execution-options?role=code.smart") return { options: [
      { runner_type: "RUNNER_TYPE_CODEX", available: true, native_objective: nativeAvailable, default_model: "fixture-codex-default", models: [{ id: "fixture-codex-default", canonical_model: "Fixture Codex default", is_default: true }, { id: "fixture-codex-alt", canonical_model: "Fixture Codex alternate", is_default: false }], effort_levels: ["medium", "high"] },
      { runner_type: "RUNNER_TYPE_OPENCODE", available: true, native_objective: false, default_model: "fixture-open-default", models: [{ id: "fixture-open-default", canonical_model: "Fixture Open default", is_default: true }], effort_levels: ["low"] },
    ] };
    unexpected.push(path); throw new Error(`Unexpected run-sheet owner read ${path}`);
  });
  vi.spyOn(defaultApiClient, "post").mockImplementation(async (path, body) => {
    if (path !== queuePath && path !== "/backlog/execute/fixture-second/queue") { unexpected.push(path); throw new Error(`Unexpected run-sheet POST ${path}`); }
    if ((body as { confirm?: boolean }).confirm === false) { expect(body).toEqual(previewBody); return preview(); }
    if (stalePlan) throw new ApiError("http", "Fixture reviewed paths stale", { status: 409, code: "plan_stale", details: { missingPaths: [{ glob: "fixture/source/**", reason: "Missing synthetic path" }] } });
    if (queueRefused) throw new Error("Fixture queue owner refused");
    if (queueDeferred) return new Promise(resolve => { releaseQueue = resolve; });
    return { dryRun: false, queued: true, taskId: "fixture-task", runId: "fixture-run-id", message: "Fixture accepted transport only", blockingReasons: [] };
  });
  for (const method of ["put", "patch", "delete"] as const) vi.spyOn(defaultApiClient, method).mockRejectedValue(new Error(`Unexpected run-sheet ${method}`));
  vi.stubGlobal("fetch", vi.fn(async () => { throw new Error("Unexpected direct run-sheet transport"); }));
});
afterEach(() => {
  unmounts.splice(0).forEach(unmount => unmount()); cleanup(); getSpatialNav()?.dispose();
  try { expect(unexpected).toEqual([]); expect(vi.mocked(defaultApiClient.post).mock.calls).toEqual(expectedPosts); for (const method of ["put", "patch", "delete"] as const) expect(defaultApiClient[method]).not.toHaveBeenCalled(); expect(fetch).not.toHaveBeenCalled(); }
  finally { clients.splice(0).forEach(client => client.clear()); vi.restoreAllMocks(); vi.unstubAllGlobals(); }
});

describe("RunSheet actual preflight and reviewed controls", () => {
  it("does not acquire a catalog or preflight while closed", () => { expectedPosts = []; mount({ open: false }); expect(defaultApiClient.get).not.toHaveBeenCalled(); expect(screen.queryByTestId("run-sheet")).toBeNull(); });
  it("offers only explicit confirm:false preflight before Cancel, with no synthetic run callback", async () => { const view = mount(); await ready(); fireEvent.click(screen.getByRole("button", { name: "Cancel" })); await waitFor(() => expect(screen.queryByTestId("run-sheet")).toBeNull()); expect(view.close).toHaveBeenCalledOnce(); expect(view.success).not.toHaveBeenCalled(); });
  it("keeps catalog refusal visible and confirmation disabled without an alternate queue", async () => { catalogRefused = true; const view = mount(); expect(await screen.findByText("Fixture run catalog refused")).toBeVisible(); expect(screen.getByRole("button", { name: "Start run" })).toBeDisabled(); fireEvent.click(screen.getByRole("button", { name: "Start run" })); expect(view.success).not.toHaveBeenCalled(); expect(view.close).not.toHaveBeenCalled(); });
  it("shows non-overridable owner blockers with no override or confirmed request", async () => { blockingReasons = [{ code: "fixture_scope", message: "Fixture owner scope unavailable", forceable: false }]; const view = mount(); await screen.findByText("Fixture owner scope unavailable"); expect(screen.getByRole("button", { name: "Start run" })).toBeDisabled(); expect(screen.queryByRole("checkbox", { name: /Override eligible blockers/ })).toBeNull(); fireEvent.click(screen.getByRole("button", { name: "Start run" })); expect(view.success).not.toHaveBeenCalled(); });
  it("does not silently substitute another execution mode when the reviewed saved mode is unavailable", async () => { savedMode = "fixture-unavailable"; const view = mount(); expect(await screen.findByText("The item's execution approach “fixture-unavailable” is unavailable.")).toBeVisible(); expect(screen.getByRole("button", { name: "Start run" })).toBeDisabled(); expect(screen.getByRole("radio", { name: "Reviewed slices" })).not.toBeChecked(); expect(view.success).not.toHaveBeenCalled(); });
  it("switches runner and replaces stale model and effort with that runner's declared defaults", async () => { mount(); await ready(); fireEvent.change(screen.getByRole("combobox", { name: "Model" }), { target: { value: "fixture-codex-alt" } }); fireEvent.change(screen.getByRole("combobox", { name: "Effort" }), { target: { value: "high" } }); fireEvent.change(screen.getByRole("combobox", { name: "Preferred runner" }), { target: { value: "RUNNER_TYPE_OPENCODE" } }); await waitFor(() => expect(screen.getByRole("combobox", { name: "Model" })).toHaveValue("fixture-open-default")); expect(screen.getByRole("combobox", { name: "Effort" })).toHaveValue("low"); });
  it("blocks a reviewed goal mode without a native-objective catalog option and offers no confirmation", async () => { savedMode = "goal"; nativeAvailable = false; const view = mount(); expect(await screen.findByRole("alert")).toHaveTextContent("Goal sessions require an available native-objective runner"); expect(screen.getByRole("button", { name: "Start run" })).toBeDisabled(); fireEvent.click(screen.getByRole("button", { name: "Start run" })); expect(view.success).not.toHaveBeenCalled(); });
});

describe("RunSheet exact synthetic confirmation adapter regressions", () => {
  it("serializes the reviewed execution mode rather than dropping it as an unknown strategy field", async () => { expectedPosts.push([queuePath, confirmed()]); const view = mount(); fireEvent.click(await ready()); await waitFor(() => expect(view.success).toHaveBeenCalledOnce()); expect(view.close).toHaveBeenCalledOnce(); expect(view.success).toHaveBeenCalledWith(expect.objectContaining({ taskId: "fixture-task", runId: "fixture-run-id", queued: true })); });
  it("narrows the reviewed slice cap and forwards exact alternate model and effort without changing owner item", async () => { expectedPosts.push([queuePath, confirmed({ max_slices: 2, execution_preferences: { ...preferences, model: "fixture-codex-alt", effort: "high" } })]); const view = mount(); await ready(); const slider = screen.getByRole("slider", { name: "Maximum slices" }); expect(slider).toHaveAttribute("max", "4"); fireEvent.change(slider, { target: { value: "2" } }); fireEvent.change(screen.getByRole("combobox", { name: "Model" }), { target: { value: "fixture-codex-alt" } }); fireEvent.change(screen.getByRole("combobox", { name: "Effort" }), { target: { value: "high" } }); fireEvent.click(screen.getByRole("button", { name: "Start run" })); await waitFor(() => expect(view.success).toHaveBeenCalledOnce()); });
  it("switches to reviewed native goal mode and excludes non-native runner preferences", async () => { expectedPosts.push([queuePath, confirmed({ execution_mode: "goal" })]); const view = mount(); await ready(); fireEvent.click(screen.getByRole("radio", { name: "Reviewed goal session" })); expect(screen.getByRole("slider", { name: "Maximum sessions" })).toHaveValue("4"); expect(within(screen.getByRole("combobox", { name: "Preferred runner" })).queryByRole("option", { name: "RUNNER_TYPE_OPENCODE" })).toBeNull(); fireEvent.click(screen.getByRole("button", { name: "Start run" })); await waitFor(() => expect(view.success).toHaveBeenCalledOnce()); });
  it("forwards a selected investigation-only blocker policy in the complete request", async () => { expectedPosts.push([queuePath, confirmed({ blocker_repair_policy: "investigate" })]); const view = mount(); await ready(); fireEvent.click(screen.getByRole("radio", { name: /Investigate, then ask/ })); fireEvent.click(screen.getByRole("button", { name: "Start run" })); await waitFor(() => expect(view.success).toHaveBeenCalledOnce()); });
  it("requires the explicit eligible-blocker override and records only that offered boolean", async () => { blockingReasons = [{ code: "fixture_question", message: "Fixture optional decision pending", forceable: true }]; expectedPosts.push([queuePath, confirmed({ force: true })]); const view = mount(); await screen.findByText(/Fixture optional decision pending/); expect(screen.getByRole("button", { name: "Start run" })).toBeDisabled(); fireEvent.click(screen.getByRole("checkbox", { name: /Override eligible blockers/ })); fireEvent.click(screen.getByRole("button", { name: "Start run" })); await waitFor(() => expect(view.success).toHaveBeenCalledOnce()); });
  it("holds confirmation pending with no duplicate request or close callback", async () => { queueDeferred = true; expectedPosts.push([queuePath, confirmed()]); const view = mount(); fireEvent.click(await ready()); await waitFor(() => expect(releaseQueue).toBeDefined()); const pending = screen.getByRole("button", { name: "Starting…" }); expect(pending).toBeDisabled(); fireEvent.click(pending); const cancel = screen.getByRole("button", { name: "Cancel" }); expect(cancel).toBeDisabled(); fireEvent.click(cancel); expect(view.close).not.toHaveBeenCalled(); expect(view.success).not.toHaveBeenCalled(); await act(async () => releaseQueue?.({ dryRun: false, queued: true, taskId: "fixture-task", blockingReasons: [] })); await waitFor(() => expect(view.success).toHaveBeenCalledOnce()); });
  it("retains selected narrowed controls and owner refusal without success, close or alternate write", async () => { queueRefused = true; expectedPosts.push([queuePath, confirmed({ max_slices: 2 })]); const view = mount(); await ready(); fireEvent.change(screen.getByRole("slider", { name: "Maximum slices" }), { target: { value: "2" } }); fireEvent.click(screen.getByRole("button", { name: "Start run" })); expect(await screen.findByText("Fixture queue owner refused")).toBeVisible(); expect(screen.getByRole("slider", { name: "Maximum slices" })).toHaveValue("2"); expect(view.success).not.toHaveBeenCalled(); expect(view.close).not.toHaveBeenCalled(); });
  it("cancels the stale-plan repair panel without starting Plan Workshop or reoffering the queue", async () => { stalePlan = true; expectedPosts.push([queuePath, confirmed()]); const view = mount(); fireEvent.click(await ready()); const panel = await screen.findByTestId("stale-plan-panel"); expect(within(panel).getByTestId("stale-plan-missing-path")).toHaveTextContent("fixture/source/**"); fireEvent.click(within(panel).getByRole("button", { name: "Cancel" })); await waitFor(() => expect(screen.queryByTestId("stale-plan-panel")).toBeNull()); expect(screen.getByTestId("run-sheet")).toBeVisible(); expect(view.close).not.toHaveBeenCalled(); expect(view.success).not.toHaveBeenCalled(); });
});
