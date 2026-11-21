import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { Route, Routes, useLocation } from "react-router-dom";
import type { QueryClient } from "@tanstack/react-query";
import { getSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { ScenarioDetailsPage } from "./ScenarioDetailsPage";
import { createTestQueryClient, renderWithProviders } from "../test-utils";
import { defaultApiClient } from "../lib/api-client";
import { DEFAULT_SETTINGS } from "../services/settings-service";
import { useScenariosStore } from "../stores/scenarios-store";

// Real page, owner services, query cache and canonical dialogs. All write
// transports are forbidden: confirming a local file choice is not archiving.
const owner = "fixture-local-owner";
const preferenceKey = "swarm-manager.archive.preferences.v1";
const clients: QueryClient[] = [];
const views: { unmount: () => void }[] = [];
const unexpected: string[] = [];
let filesRefused = false;
let originalPreference: string | null;
const files = [
  { name: "PRD.md", path: "PRD.md", type: "file", size: "32" },
  { name: "notes.md", path: "docs/notes.md", type: "file", size: "24" },
  { name: "main.ts", path: "src/main.ts", type: "file", size: "48" },
];
function Location() { const location = useLocation(); return <output data-testid="scenario-local-location">{location.pathname + location.search}</output>; }
function mount() {
  const client = createTestQueryClient(); clients.push(client);
  const view = renderWithProviders(<><Routes><Route path="/scenarios/:name" element={<ScenarioDetailsPage />} /></Routes><Location /></>, { queryClient: client, initialEntries: [`/scenarios/${owner}`] });
  views.push(view); return client;
}
async function archive() {
  mount(); await screen.findAllByText("Local Owner");
  await waitFor(() => expect(clients.at(-1)?.getQueryData<{ deleteConfirmation: { scenario: string } }>(["settings"])?.deleteConfirmation.scenario).toBe("strong"));
  fireEvent.click(screen.getAllByTestId("scenario-detail-tab-manage").at(-1)!);
  fireEvent.click(screen.getByTestId("scenario-details-delete"));
  const dialog = await screen.findByTestId("scenario-delete-dialog");
  await waitFor(() => expect(screen.queryByTestId("scenario-archive-file-tree-loading")).toBeNull());
  return dialog;
}
async function customize() {
  fireEvent.click(screen.getByTestId("customize-files-link"));
  const dialog = await screen.findByTestId("file-selection-dialog");
  await waitFor(() => expect(screen.queryByTestId("file-selection-loading-state")).toBeNull());
  return dialog;
}
function cancelArchive() {
  fireEvent.click(screen.getByTestId("scenario-delete-cancel"));
  expect(screen.queryByTestId("scenario-delete-dialog")).toBeNull();
  expect(screen.getByTestId("scenario-local-location")).toHaveTextContent(`/scenarios/${owner}`);
}
beforeEach(() => {
  originalPreference = localStorage.getItem(preferenceKey); localStorage.removeItem(preferenceKey);
  filesRefused = false; unexpected.length = 0; useScenariosStore.getState().reset();
  vi.spyOn(defaultApiClient, "get").mockImplementation(async path => {
    if (path === "/settings") { const { deleteConfirmation: _, ...settings } = DEFAULT_SETTINGS; return { settings: { ...settings, deleteConfirmationLevels: { scenario: "DELETE_CONFIRM_LEVEL_STRONG" } } }; }
    if (path === `/scenarios/${owner}`) return { scenario: { name: owner, display_name: "Local Owner", description: "Prospective local preservation only", status: "stopped", priority: 2, tags: ["fixture"], is_greenfield: false } };
    if (path === `/scenarios/${owner}/context`) return { scenario_name: owner, goals: [], orphan_items: [], rollup: { total: 0 }, fixes: { active: [], archived: [] } };
    if (path === `/scenarios/${owner}/files`) { if (filesRefused) throw new Error("Owner refused prospective files"); return { files }; }
    unexpected.push(path); throw new Error(`Unexpected scenario read ${path}`);
  });
  for (const method of ["post", "put", "patch", "delete"] as const) vi.spyOn(defaultApiClient, method).mockRejectedValue(new Error(`Forbidden scenario ${method}`));
  vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("Forbidden external transport")));
});
afterEach(() => {
  try {
    views.splice(0).forEach(view => view.unmount()); cleanup(); getSpatialNav()?.dispose();
    expect(unexpected).toEqual([]); expect(fetch).not.toHaveBeenCalled();
    for (const method of ["post", "put", "patch", "delete"] as const) expect(defaultApiClient[method]).not.toHaveBeenCalled();
  } finally {
    clients.splice(0).forEach(client => client.clear());
    vi.restoreAllMocks(); vi.unstubAllGlobals(); useScenariosStore.getState().reset();
    if (originalPreference === null) localStorage.removeItem(preferenceKey); else localStorage.setItem(preferenceKey, originalPreference);
  }
});

describe("Scenario prospective preservation and local owner navigation", () => {
  it("reveals owner work from the overview shortcut without creating work", async () => {
    mount(); await screen.findAllByText("Local Owner");
    fireEvent.click(screen.getAllByTestId("scenario-overview-work-link").at(-1)!);
    expect(await screen.findByTestId("scenario-coverage-empty")).toBeVisible();
    expect(screen.getByTestId("scenario-local-location")).toHaveTextContent(`/scenarios/${owner}`);
  });
  it("changes the prospective preset and previews only the owner-returned matching paths", async () => {
    const dialog = await archive();
    fireEvent.change(within(dialog).getByTestId("archive-preset-select"), { target: { value: "documentation" } });
    expect(within(dialog).getByTestId("archive-preview-count")).toHaveTextContent("2 files");
    expect(within(dialog).getByText("docs/notes.md")).toBeVisible();
    expect(within(dialog).queryByText("src/main.ts")).toBeNull();
    cancelArchive();
  });
  it("confirms an exact local all-file choice into the preview then cancels the archive", async () => {
    const parent = await archive(); const selection = await customize();
    fireEvent.click(within(selection).getByTestId("select-all-button"));
    expect(within(selection).getByTestId("confirm-selection-button")).toHaveTextContent("3 files");
    fireEvent.click(within(selection).getByTestId("confirm-selection-button"));
    await waitFor(() => expect(screen.queryByTestId("file-selection-dialog")).toBeNull());
    expect(within(parent).getByText("Custom Selection")).toBeVisible();
    expect(within(parent).getByTestId("archive-preview-count")).toHaveTextContent("3 files");
    expect(within(parent).getByText("src/main.ts")).toBeVisible(); cancelArchive();
  });
  it("cancels a tentative all-file choice without replacing the original planning preview", async () => {
    const parent = await archive(); const before = within(parent).getByTestId("archive-preview-count").textContent;
    const selection = await customize(); fireEvent.click(within(selection).getByTestId("select-all-button"));
    fireEvent.click(within(selection).getByRole("button", { name: "Cancel" }));
    await waitFor(() => expect(screen.queryByTestId("file-selection-dialog")).toBeNull());
    expect(within(parent).getByTestId("archive-preview-count")).toHaveTextContent(before!);
    expect(within(parent).queryByText("Custom Selection")).toBeNull(); cancelArchive();
  });
  it("clears a tentative selection and cancels without persisting an empty custom choice", async () => {
    const parent = await archive(); const selection = await customize();
    fireEvent.click(within(selection).getByTestId("clear-all-button"));
    expect(within(selection).getByTestId("confirm-selection-button")).toHaveTextContent("0 files");
    fireEvent.click(within(selection).getByRole("button", { name: "Cancel" }));
    expect(within(parent).getByTestId("archive-preview-count")).toHaveTextContent("1 files"); cancelArchive();
  });
  it("removes and restores prospective preservation using only the exact file read", async () => {
    const dialog = await archive(); const offered = vi.mocked(defaultApiClient.get).mock.calls.filter(([path]) => path === `/scenarios/${owner}/files`).length;
    fireEvent.click(within(dialog).getByTestId("scenario-delete-archive"));
    expect(screen.queryByTestId("archive-preview-panel")).toBeNull();
    fireEvent.click(within(dialog).getByTestId("scenario-delete-archive"));
    expect(await screen.findByTestId("archive-preview-panel")).toBeVisible();
    await waitFor(() => expect(vi.mocked(defaultApiClient.get).mock.calls.filter(([path]) => path === `/scenarios/${owner}/files`).length).toBe(offered + 1));
    cancelArchive();
  });
  it("allows local spec-sync intent to be deselected and canceled before any sync offer", async () => {
    const dialog = await archive(); const toggle = within(dialog).getByTestId("spec-sync-toggle");
    const input = within(toggle).getByRole("checkbox"); expect(input).not.toBeChecked();
    fireEvent.click(input); expect(input).toBeChecked();
    expect(within(dialog).getByTestId("scenario-delete-confirm")).toHaveTextContent("Sync & Archive");
    fireEvent.click(input); expect(input).not.toBeChecked();
    expect(within(dialog).getByTestId("scenario-delete-confirm")).toHaveTextContent("Delete Scenario"); cancelArchive();
  });
  it("keeps refused prospective file reads cancelable through both nested and parent dialogs", async () => {
    filesRefused = true; await archive(); const selection = await customize();
    expect(within(selection).getByTestId("confirm-selection-button")).toHaveTextContent("0 files");
    fireEvent.click(within(selection).getByRole("button", { name: "Cancel" }));
    await waitFor(() => expect(screen.queryByTestId("file-selection-dialog")).toBeNull()); cancelArchive();
  });
});
