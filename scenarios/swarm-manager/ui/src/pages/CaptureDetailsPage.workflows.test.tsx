import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { Route, Routes, useLocation } from "react-router-dom";
import type { QueryClient } from "@tanstack/react-query";
import { getSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { CaptureDetailsPage } from "./CaptureDetailsPage";
import { createTestQueryClient, renderWithProviders } from "../test-utils";
import { defaultApiClient } from "../lib/api-client";
import { DEFAULT_SETTINGS } from "../services/settings-service";
import { useCaptureStore } from "../stores/capture-store";
import type { Capture } from "../types";

// Outside-fixture draft: actual routed page, store, services and Connect client.
// Only the network boundary is substituted. No attach/native actions are invoked.
const initial: Capture = { id: "owner-capture", text: "A bounded captured thought", attachments: [], created: "2026-02-01T12:00:00Z", status: "failed", classification: null, note: "Original personal note" };
const other: Capture = { ...initial, id: "other-capture", text: "Unrelated capture" };
let current: Capture;
let confirmDelete = false;
let getRefused: boolean;
let transitionRefused: boolean;
let completeRead: ((value: unknown) => void) | undefined;
let deferRead: boolean;
let completeTransition: ((value: Response) => void) | undefined;
const clients: QueryClient[] = [];
const unexpected: string[] = [];
const starts: unknown[] = [];
const allowed = new Set<string>();
function response(body: unknown, status = 200) { return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } }); }
function raw(capture: Capture) { return { ...capture, classification: capture.classification && { ...capture.classification, classified_at: capture.classification.classifiedAt } }; }
function Location() { const location = useLocation(); return <output data-testid="capture-workflow-location">{location.pathname + location.search}</output>; }
function mount(path = "/captures/owner-capture") {
  const client = createTestQueryClient(); clients.push(client);
  return { ...renderWithProviders(<><Routes><Route path="/captures/:captureId" element={<CaptureDetailsPage />} /><Route path="/captures" element={<CaptureDetailsPage />} /></Routes><Location /></>, { queryClient: client, initialEntries: [path] }), client };
}
function seed() { useCaptureStore.setState({ captures: [current, other], status: "success" }); }
function noWrites() { for (const method of ["post", "put", "patch", "delete"] as const) expect(defaultApiClient[method]).not.toHaveBeenCalled(); expect(starts).toEqual([]); }
async function openDelete() { await waitFor(() => expect(clients.at(-1)?.getQueryData<{ deleteConfirmation: { capture: string } }>(["settings"])?.deleteConfirmation.capture).toBe("simple")); fireEvent.click(await screen.findByTestId("detail-header-actions")); fireEvent.click(await screen.findByRole("menuitem", { name: "Delete" })); return screen.findByRole("alertdialog", { name: "Delete Capture" }); }
beforeEach(() => {
  confirmDelete = false; current = structuredClone(initial); getRefused = false; transitionRefused = false; deferRead = false; completeRead = undefined; completeTransition = undefined; unexpected.length = 0; starts.length = 0; allowed.clear(); useCaptureStore.getState().reset();
  vi.spyOn(defaultApiClient, "get").mockImplementation(async path => {
    if (path === "/settings") { const { deleteConfirmation: _, ...settings } = DEFAULT_SETTINGS; return { settings: { ...settings, ...(confirmDelete ? { deleteConfirmationLevels: { capture: "DELETE_CONFIRM_LEVEL_SIMPLE" } } : {}) } }; }
    if (path === "/captures/owner-capture") { if (getRefused) throw new Error("Capture owner refused read"); if (deferRead) return new Promise(resolve => { completeRead = resolve; }); return { capture: raw(current) }; }
    unexpected.push(path); throw new Error(`Unexpected capture read ${path}`);
  });
  for (const method of ["post", "put", "patch", "delete"] as const) vi.spyOn(defaultApiClient, method).mockRejectedValue(new Error(`Unexpected capture ${method}`));
  vi.stubGlobal("fetch", vi.fn(async (value: RequestInfo | URL, init?: RequestInit) => {
    const request = value instanceof Request ? new Request(value, init) : new Request(String(value), init); const path = new URL(request.url).pathname;
    if (path.endsWith("/ListTransitions")) return response({ transitions: [{ key: "capture.classify", label: "Classify capture", subject: "capture", kind: "TRANSITION_KIND_WORKFLOW" }] });
    if (path.endsWith("/StartTransition")) { starts.push(JSON.parse(await request.text())); if (transitionRefused) return response({ code: "permission_denied", message: "Capture owner refused classification" }, 403); return new Promise<Response>(resolve => { completeTransition = resolve; }); }
    unexpected.push(path); throw new Error(`Unexpected capture transport ${path}`);
  }));
});
afterEach(() => { cleanup(); getSpatialNav()?.dispose(); expect(unexpected).toEqual([]); for (const method of ["post", "put", "patch", "delete"] as const) if (!allowed.has(method)) expect(defaultApiClient[method]).not.toHaveBeenCalled(); clients.splice(0).forEach(client => client.clear()); vi.restoreAllMocks(); vi.unstubAllGlobals(); useCaptureStore.getState().reset(); });

describe("Capture details exact-owner workflows", () => {
  it("renders the store-selected capture without a redundant detail read", async () => { seed(); mount(); expect(await screen.findByText("Capture Text")).toBeVisible(); expect(screen.getAllByText(initial.text).length).toBeGreaterThan(0); expect(vi.mocked(defaultApiClient.get).mock.calls.some(([path]) => path === "/captures/owner-capture")).toBe(false); noWrites(); });
  it("keeps a deep link loading until its exact owner response populates the query cache", async () => { deferRead = true; const { client } = mount(); expect(await screen.findByText("Loading capture...")).toBeVisible(); await waitFor(() => expect(completeRead).toBeDefined()); expect(client.getQueryData(["capture", "owner-capture"])).toBeUndefined(); await act(async () => completeRead?.({ capture: raw(current) })); expect(await screen.findByText("Capture Text")).toBeVisible(); expect(client.getQueryData<Capture>(["capture", "owner-capture"])?.id).toBe("owner-capture"); noWrites(); });
  it("does not read or mutate an absent capture identifier", async () => { mount("/captures"); expect(await screen.findByText("No capture selected.")).toBeVisible(); expect(vi.mocked(defaultApiClient.get).mock.calls.some(([path]) => path.startsWith("/captures"))).toBe(false); noWrites(); });
  it("reports a refused deep link under the canonical test client without a fallback write", async () => { getRefused = true; mount(); expect(await screen.findByText("Capture not found.", {}, { timeout: 12000 })).toBeVisible(); expect(vi.mocked(defaultApiClient.get).mock.calls.filter(([path]) => path === "/captures/owner-capture")).toHaveLength(1); noWrites(); }, 15000);
  it("disables both classification controls while the exact request is pending, then updates only that capture", async () => { seed(); mount(); fireEvent.click((await screen.findAllByRole("button", { name: "Retry" }))[0]!); await waitFor(() => expect(completeTransition).toBeDefined()); expect(starts).toEqual([{ transitionKey: "capture.classify", subjectRef: { subject: "capture", value: "owner-capture" } }]); for (const button of screen.getAllByRole("button", { name: "Retry" })) { expect(button).toBeDisabled(); fireEvent.click(button); } expect(starts).toHaveLength(1); expect(useCaptureStore.getState().captures[0]?.status).toBe("failed"); await act(async () => completeTransition?.(response({ executionId: "fixture-classification" }))); expect(await screen.findByText("Classification in progress...")).toBeVisible(); expect(useCaptureStore.getState().captures.find(c => c.id === "owner-capture")?.classification).toBeNull(); expect(useCaptureStore.getState().captures.find(c => c.id === "other-capture")).toBe(other); });
  it("retains failed classification and existing store state on owner refusal without alternate writes", async () => { transitionRefused = true; seed(); const before = useCaptureStore.getState().captures; mount(); fireEvent.click((await screen.findAllByRole("button", { name: "Retry" }))[0]!); await waitFor(() => expect(starts).toHaveLength(1)); await waitFor(() => expect(screen.getAllByRole("button", { name: "Retry" })[0]).not.toBeDisabled()); expect(screen.getByText("Classification failed")).toBeVisible(); expect(useCaptureStore.getState().captures).toBe(before); expect(defaultApiClient.post).not.toHaveBeenCalled(); expect(defaultApiClient.patch).not.toHaveBeenCalled(); expect(defaultApiClient.delete).not.toHaveBeenCalled(); });
  it("cancels a personal note draft without writing or changing the owner store", async () => { seed(); mount(); fireEvent.click(await screen.findByRole("button", { name: "Original personal note" })); fireEvent.change(screen.getByRole("textbox"), { target: { value: "Unsubmitted personal draft" } }); fireEvent.click(screen.getByRole("button", { name: "Cancel" })); expect(screen.getByRole("button", { name: "Original personal note" })).toBeVisible(); expect(useCaptureStore.getState().captures[0]?.note).toBe(initial.note); noWrites(); });
  it("saves a note to the exact capture and updates local state only after owner completion", async () => { allowed.add("patch"); let complete!: (value: unknown) => void; vi.mocked(defaultApiClient.patch).mockImplementation(() => new Promise(resolve => { complete = resolve; })); seed(); mount(); fireEvent.click(await screen.findByRole("button", { name: "Original personal note" })); fireEvent.change(screen.getByRole("textbox"), { target: { value: "Reviewed personal note" } }); fireEvent.click(screen.getByRole("button", { name: "Save" })); await waitFor(() => expect(vi.mocked(defaultApiClient.patch).mock.calls).toEqual([["/captures/owner-capture", { note: "Reviewed personal note" }]])); expect(useCaptureStore.getState().captures[0]?.note).toBe(initial.note); await act(async () => complete({ capture: raw({ ...current, note: "Reviewed personal note" }) })); expect(await screen.findByRole("button", { name: "Reviewed personal note" })).toBeVisible(); expect(useCaptureStore.getState().captures.find(c => c.id === "other-capture")).toBe(other); });
  // Desired lifecycle regressions: unqualified against current NoteEditor source.
  // Do not execute these as passing proof until the bounded owner repair is reviewed.
  it("blocks a duplicate note save and cancellation while the owner write is pending", async () => {
    allowed.add("patch"); let complete!: (value: unknown) => void;
    vi.mocked(defaultApiClient.patch).mockImplementation(() => new Promise(resolve => { complete = resolve; }));
    seed(); mount(); fireEvent.click(await screen.findByRole("button", { name: "Original personal note" }));
    fireEvent.change(screen.getByRole("textbox"), { target: { value: "Pending personal note" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(defaultApiClient.patch).toHaveBeenCalledOnce());
    expect(screen.getByRole("button", { name: "Saving note" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(screen.getByRole("textbox")).toHaveValue("Pending personal note");
    fireEvent.click(screen.getByRole("button", { name: "Saving note" })); expect(defaultApiClient.patch).toHaveBeenCalledOnce();
    await act(async () => complete({ capture: raw({ ...current, note: "Pending personal note" }) }));
    expect(await screen.findByRole("button", { name: "Pending personal note" })).toBeVisible();
  });
  it("handles note refusal visibly while retaining the exact draft and original owner state", async () => {
    allowed.add("patch"); vi.mocked(defaultApiClient.patch).mockRejectedValue(new Error("Capture owner refused note"));
    seed(); const before = useCaptureStore.getState().captures; mount();
    fireEvent.click(await screen.findByRole("button", { name: "Original personal note" }));
    fireEvent.change(screen.getByRole("textbox"), { target: { value: "Unaccepted personal note" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Capture owner refused note");
    expect(screen.getByRole("textbox")).toHaveValue("Unaccepted personal note");
    expect(useCaptureStore.getState().captures).toBe(before); expect(defaultApiClient.patch).toHaveBeenCalledOnce();
    expect(defaultApiClient.post).not.toHaveBeenCalled(); expect(defaultApiClient.delete).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(screen.getByRole("button", { name: "Original personal note" })).toBeVisible();
  });
  it.each(["escape", "backdrop", "close"] as const)("opens only the chosen attachment and closes by %s without writes", async mode => { current.attachments = ["/fixture/one.png", "/fixture/two.png"]; seed(); mount(); fireEvent.click(await screen.findByAltText("Attachment 2")); const full = screen.getByAltText("Full resolution attachment"); expect(full).toHaveAttribute("src", "/fixture/two.png"); fireEvent.click(full); expect(full).toBeVisible(); if (mode === "escape") fireEvent.keyDown(window, { key: "Escape" }); else if (mode === "backdrop") fireEvent.click(full.parentElement!); else fireEvent.click(screen.getByRole("button", { name: "Close lightbox" })); await waitFor(() => expect(screen.queryByAltText("Full resolution attachment")).toBeNull()); noWrites(); });
  it.each([false, true])("renders classified metadata and its actual proposal outcome (items=%s)", async withItems => { current.status = "classified"; current.classification = { classifiedAt: "2026-02-02T12:00:00Z", items: withItems ? [{ kind: "idea", title: "Owner proposal", description: "Suggestion", priority: 5, tags: [], confidence: 0.8 }] : [] }; seed(); mount(); expect(await screen.findByText(withItems ? "Proposals sent to the Decide stream." : "Nothing actionable detected")).toBeVisible(); expect(screen.getByText("Classified at")).toBeVisible(); expect(screen.getByText("owner-capture")).toBeVisible(); noWrites(); });
  it("cancels capture deletion without a request or store change", async () => { confirmDelete = true; seed(); const before = useCaptureStore.getState().captures; mount(); const dialog = await openDelete(); fireEvent.click(within(dialog).getByRole("button", { name: "Cancel" })); expect(useCaptureStore.getState().captures).toBe(before); noWrites(); });
  it("deletes only the confirmed capture after owner completion and preserves its unrelated sibling", async () => { confirmDelete = true; allowed.add("delete"); let complete!: (value: unknown) => void; vi.mocked(defaultApiClient.delete).mockImplementation(() => new Promise(resolve => { complete = resolve; })); seed(); mount(); const dialog = await openDelete(); const textbox = within(dialog).queryByRole("textbox"); if (textbox) fireEvent.change(textbox, { target: { value: "owner-capture" } }); fireEvent.click(within(dialog).getByRole("button", { name: "Delete" })); await waitFor(() => expect(vi.mocked(defaultApiClient.delete).mock.calls).toEqual([["/captures/owner-capture"]])); expect(screen.getByTestId("capture-workflow-location")).toHaveTextContent("/captures/owner-capture"); expect(useCaptureStore.getState().captures).toHaveLength(2); expect(within(dialog).getByRole("button", { name: "Processing..." })).toBeDisabled(); await act(async () => complete({})); await waitFor(() => expect(screen.getByTestId("capture-workflow-location")).toHaveTextContent("/plan")); expect(useCaptureStore.getState().captures).toEqual([other]); });
  it("keeps the confirmation open and preserves the capture after a refused delete", async () => { confirmDelete = true; allowed.add("delete"); vi.mocked(defaultApiClient.delete).mockRejectedValue(new Error("Capture owner refused deletion")); seed(); const before = useCaptureStore.getState().captures; mount(); const dialog = await openDelete(); const textbox = within(dialog).queryByRole("textbox"); if (textbox) fireEvent.change(textbox, { target: { value: "owner-capture" } }); fireEvent.click(within(dialog).getByRole("button", { name: "Delete" })); await waitFor(() => expect(defaultApiClient.delete).toHaveBeenCalledOnce()); await waitFor(() => expect(within(dialog).getByRole("button", { name: "Delete" })).not.toBeDisabled()); expect(dialog).toBeVisible(); expect(useCaptureStore.getState().captures).toBe(before); expect(screen.getByTestId("capture-workflow-location")).toHaveTextContent("/captures/owner-capture"); expect(defaultApiClient.post).not.toHaveBeenCalled(); expect(defaultApiClient.patch).not.toHaveBeenCalled(); });
});
