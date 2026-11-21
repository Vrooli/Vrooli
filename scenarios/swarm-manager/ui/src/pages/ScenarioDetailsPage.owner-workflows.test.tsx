import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { Route, Routes, useLocation } from "react-router-dom";
import type { QueryClient } from "@tanstack/react-query";
import { getSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { ScenarioDetailsPage } from "./ScenarioDetailsPage";
import { createTestQueryClient, renderWithProviders } from "../test-utils";
import { defaultApiClient } from "../lib/api-client";
import { DEFAULT_SETTINGS } from "../services/settings-service";
import { useScenariosStore } from "../stores/scenarios-store";
import type { Scenario } from "../types";

// Outside-only draft. Real page/router/hooks/store/services; only upstream client
// methods are substituted. No attach/start/stop/restart/delete/spec-sync actions.
const initial: Scenario = { name: "fixture-owner", displayName: "Fixture Owner", description: "Bounded owner fixture", status: "stopped", priority: 2, tags: ["fixture"], isGreenfield: false };
const unrelated: Scenario = { ...initial, name: "unrelated-owner", displayName: "Unrelated Owner" };
const clients: QueryClient[] = [];
const mountedViews:{unmount:()=>void}[]=[];
const unexpected: string[] = [];
let current: Scenario;
let deferRead: boolean;
let readRefused: boolean;
let filesRefused: boolean;
let contextRefused: boolean;
let releaseRead: ((value: unknown) => void) | undefined;
let context: unknown;
let allowPatch: boolean;
function raw(value: Scenario) { return { name: value.name, display_name: value.displayName, description: value.description, status: value.status, priority: value.priority, tags: value.tags, is_greenfield: value.isGreenfield }; }
function Location() { const location = useLocation(); return <output data-testid="scenario-owner-location">{location.pathname + location.search}</output>; }
function mount(path = "/scenarios/fixture-owner") {
  const client = createTestQueryClient(); clients.push(client);
  client.setQueryDefaults(["scenarios", "unrelated-owner"], { gcTime: Infinity }); client.setQueryData(["scenarios", "unrelated-owner"], unrelated);
  const view=renderWithProviders(<><Routes><Route path="/scenarios/:name" element={<ScenarioDetailsPage />} /><Route path="/scenarios" element={<ScenarioDetailsPage />} /><Route path="/goals/:name" element={<h1>Selected goal destination</h1>} /></Routes><Location /></>, { queryClient: client, initialEntries: [path] });mountedViews.push(view);return{client,...view};
}
async function manage() { await screen.findAllByText("Fixture Owner"); fireEvent.click(screen.getAllByTestId("scenario-detail-tab-manage").at(-1)!); return screen.findByTestId("scenario-greenfield-toggle"); }
function noWrites() { for (const method of ["post", "put", "patch", "delete"] as const) expect(defaultApiClient[method]).not.toHaveBeenCalled(); }
async function openArchive() {
  await screen.findAllByText("Fixture Owner");
  await waitFor(() => expect(clients.at(-1)?.getQueryData<{ deleteConfirmation: { scenario: string } }>(["settings"])?.deleteConfirmation.scenario).toBe("strong"));
  fireEvent.click(screen.getAllByTestId("scenario-detail-tab-manage").at(-1)!);
  fireEvent.click(screen.getByTestId("scenario-details-delete"));
  return screen.findByTestId("scenario-delete-dialog");
}
beforeEach(() => {
  current = { ...initial }; deferRead = false; readRefused = false; filesRefused = false; contextRefused = false; releaseRead = undefined; allowPatch = false; unexpected.length = 0;
  context = { scenario_name: initial.name, goals: [], orphan_items: [], rollup: { total: 0 }, fixes: { active: [], archived: [] } };
  useScenariosStore.getState().reset();
  vi.spyOn(defaultApiClient, "get").mockImplementation(async path => {
    if (path === "/settings") { const { deleteConfirmation: _, ...settings } = DEFAULT_SETTINGS; return { settings: { ...settings, deleteConfirmationLevels: { scenario: "DELETE_CONFIRM_LEVEL_STRONG" } } }; }
    if (path === "/scenarios/fixture-owner") { if (readRefused) throw new Error("Scenario owner refused read"); if (deferRead) return new Promise(resolve => { releaseRead = resolve; }); return { scenario: raw(current) }; }
    if (path === "/scenarios/fixture-owner/context") { if (contextRefused) throw new Error("Scenario owner refused context"); return context; }
    if (path === "/scenarios/fixture-owner/files") { if (filesRefused) throw new Error("Scenario owner refused files"); return { files: [{ name: "PRD.md", path: "PRD.md", type: "file", size: "32" }, { name: "notes.md", path: "docs/notes.md", type: "file", size: "24" }] }; }
    unexpected.push(path); throw new Error(`Unexpected scenario owner read ${path}`);
  });
  for (const method of ["post", "put", "patch", "delete"] as const) vi.spyOn(defaultApiClient, method).mockRejectedValue(new Error(`Unexpected scenario owner ${method}`));
  vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("Unexpected scenario external transport")));
});
afterEach(() => {
  mountedViews.splice(0).forEach(view=>view.unmount());cleanup(); getSpatialNav()?.dispose(); expect(unexpected).toEqual([]); expect(fetch).not.toHaveBeenCalled();
  for (const method of ["post", "put", "delete"] as const) expect(defaultApiClient[method]).not.toHaveBeenCalled(); if (!allowPatch) expect(defaultApiClient.patch).not.toHaveBeenCalled();
  for (const client of clients.splice(0)) { expect(client.getQueryData(["scenarios", "unrelated-owner"])).toEqual(unrelated); client.clear(); }
  vi.restoreAllMocks(); vi.unstubAllGlobals(); useScenariosStore.getState().reset();
});
describe("Scenario details actual owner workflows", () => {
  it("waits for the exact deep-link owner response before populating the scenario cache", async () => { deferRead = true; const { client } = mount(); expect(await screen.findByTestId("scenario-details-loading-state")).toBeVisible(); await waitFor(() => expect(releaseRead).toBeDefined()); expect(client.getQueryData(["scenarios", initial.name])).toBeUndefined(); await act(async () => releaseRead?.({ scenario: raw(current) })); expect((await screen.findAllByText("Fixture Owner"))[0]).toBeVisible(); expect(client.getQueryData<Scenario>(["scenarios", initial.name])?.status).toBe("stopped"); noWrites(); });
  it("rejects a missing route identifier without an owner read or mutation", async () => { mount("/scenarios"); expect(await screen.findByText("Invalid URL")).toBeVisible(); expect(vi.mocked(defaultApiClient.get).mock.calls.some(([path]) => path.startsWith("/scenarios"))).toBe(false); noWrites(); });
  it("preserves a refused owner read without an alternate lifecycle write", async () => { readRefused = true; mount(); expect(await screen.findByText("Unable to load scenario", {}, { timeout: 12000 })).toBeVisible(); expect(screen.getByTestId("scenario-owner-location")).toHaveTextContent("/scenarios/fixture-owner"); noWrites(); }, 15000);
  it("holds metadata cache/store changes until exact owner completion and blocks a duplicate toggle", async () => { allowPatch = true; let finish!: (value: unknown) => void; vi.mocked(defaultApiClient.patch).mockImplementation(() => new Promise(resolve => { finish = resolve; })); useScenariosStore.setState({ scenarios: [initial, unrelated], status: "success" }); const { client } = mount(); const toggle = await manage(); fireEvent.click(toggle); await waitFor(() => expect(defaultApiClient.patch).toHaveBeenCalledWith("/scenarios/fixture-owner", { is_greenfield: true })); expect(toggle).toBeDisabled(); fireEvent.click(toggle); expect(defaultApiClient.patch).toHaveBeenCalledOnce(); expect(client.getQueryData<Scenario>(["scenarios", initial.name])?.isGreenfield).toBe(false); expect(useScenariosStore.getState().scenarios.find(s => s.name === initial.name)?.isGreenfield).toBe(false); await act(async () => finish({ scenario: raw({ ...current, isGreenfield: true }) })); await waitFor(() => expect(client.getQueryData<Scenario>(["scenarios", initial.name])?.isGreenfield).toBe(true)); expect(useScenariosStore.getState().scenarios.find(s => s.name === initial.name)?.isGreenfield).toBe(true); expect(useScenariosStore.getState().scenarios.find(s => s.name === unrelated.name)).toBe(unrelated); });
  it("rolls back refused metadata intent while preserving the owner cache and unrelated row", async () => { allowPatch = true; vi.mocked(defaultApiClient.patch).mockRejectedValue(new Error("Scenario owner refused metadata")); useScenariosStore.setState({ scenarios: [initial, unrelated], status: "success" }); const { client } = mount(); const toggle = await manage(); fireEvent.click(toggle); expect((await screen.findAllByText("Failed to update settings. Please try again."))[0]).toBeVisible(); expect(toggle).not.toBeDisabled(); expect(client.getQueryData<Scenario>(["scenarios", initial.name])?.isGreenfield).toBe(false); expect(useScenariosStore.getState().scenarios.find(s => s.name === initial.name)?.isGreenfield).toBe(false); expect(defaultApiClient.patch).toHaveBeenCalledOnce(); });
  it("routes the owner-returned associated goal without creating or launching work", async () => { context = { scenario_name: initial.name, goals: [{ name: "fixture-goal", title: "Owner returned goal", status: "active", priority: 1, scope: { total: 3, completed: 1, in_progress: 1, pending: 1 } }], rollup: { total: 3, completed: 1, in_progress: 1, pending: 1 } }; mount(); await screen.findAllByText("Fixture Owner"); fireEvent.click(screen.getAllByTestId("scenario-detail-tab-work").at(-1)!); fireEvent.click(await screen.findByTestId("scenario-coverage-goal-fixture-goal")); expect(await screen.findByText("Selected goal destination")).toBeVisible(); expect(screen.getByTestId("scenario-owner-location")).toHaveTextContent("/goals/fixture-goal"); noWrites(); });
  it("renders resolved empty coverage as absence of work rather than creating a tracker", async () => { mount(); await screen.findAllByText("Fixture Owner"); fireEvent.click(screen.getAllByTestId("scenario-detail-tab-work").at(-1)!); expect(await screen.findByTestId("scenario-coverage-empty")).toBeVisible(); noWrites(); });
  it("shows refused work-context evidence without dispatching remediation", async () => { contextRefused = true; mount(); await screen.findAllByText("Fixture Owner"); fireEvent.click(screen.getAllByTestId("scenario-detail-tab-work").at(-1)!); expect(await screen.findByText("Failed to load scenario coverage.", {}, { timeout: 12000 })).toBeVisible(); noWrites(); }, 15000);
  it("loads a prospective archive file tree and cancels without deleting or spec-syncing", async () => { mount(); const dialog = await openArchive(); await waitFor(() => expect(vi.mocked(defaultApiClient.get).mock.calls.some(([path]) => path === "/scenarios/fixture-owner/files")).toBe(true)); await waitFor(() => expect(screen.queryByTestId("scenario-archive-file-tree-loading")).toBeNull()); expect(within(dialog).getByTestId("archive-preview-count")).toHaveTextContent(/files/); fireEvent.click(screen.getByTestId("scenario-delete-cancel")); expect(screen.queryByTestId("scenario-delete-dialog")).toBeNull(); expect(screen.getByTestId("scenario-owner-location")).toHaveTextContent("/scenarios/fixture-owner"); noWrites(); });
  it("keeps file-list refusal cancelable without an alternate preservation write", async () => { filesRefused = true; mount(); await openArchive(); await waitFor(() => expect(screen.queryByTestId("scenario-archive-file-tree-loading")).toBeNull()); fireEvent.click(screen.getByTestId("scenario-delete-cancel")); expect(screen.queryByTestId("scenario-delete-dialog")).toBeNull(); noWrites(); });
});
