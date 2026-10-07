import { useState } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { Route, Routes, useLocation } from "react-router-dom";
import type { QueryClient } from "@tanstack/react-query";
import { getSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { CreateWorkFromPlanDialog } from "./CreateWorkFromPlanDialog";
import { defaultApiClient } from "../../lib/api-client";
import { createTestQueryClient, renderWithProviders } from "../../test-utils";

// Outside-only draft. Actual form, service translation, routing and canonical
// controls; only upstream API client offers are replaced. No run/start/accept.
const clients: QueryClient[] = [];
const unmounts: Array<() => void> = [];
const unexpected: string[] = [];
let plans: Array<Record<string, unknown>>;
let listRefused: boolean;
let listDeferred: boolean;
let releaseList: ((value: unknown) => void) | undefined;
let expectedPosts: Array<unknown[]>;
const blankItems = { plan_id: undefined, source_path: undefined, markdown: undefined, title: undefined, slug: undefined, container: { type: "items", name: undefined, title: undefined, description: undefined } };
const importResult = { slug: "fixture-plan", plan_id: "fixture-plan-id", container: "items", items: [{ kind: "execute", name: "fixture-phase", title: "Fixture phase", action: "created" }], count: 1, created: 1, updated: 0, linked: 0 };
function Location() { const location = useLocation(); return <output data-testid="create-work-owner-location">{location.pathname + location.search}</output>; }
function mount(openInitially = true) {
  const imported = vi.fn(); const close = vi.fn(); const client = createTestQueryClient(); clients.push(client);
  function Parent() {
    const [open, setOpen] = useState(openInitially);
    return <><button onClick={() => setOpen(true)}>Open fixture import</button><CreateWorkFromPlanDialog isOpen={open} onClose={() => { close(); setOpen(false); }} onImported={imported} /></>;
  }
  const view = renderWithProviders(<><Routes><Route path="/plan" element={<Parent />} /><Route path="/backlog/:kind/:name" element={<h1>Exact imported backlog destination</h1>} /></Routes><Location /></>, { queryClient: client, initialEntries: ["/plan?keep=fixture"] });
  unmounts.push(view.unmount); return { ...view, client, imported, close };
}
async function loaded() { return screen.findByTestId("create-work-plan-option-fixture-plan"); }
function expectPost(body: unknown) { expectedPosts = [["/plan-import", body, { signal: undefined }]]; }
function chooseMarkdown() { fireEvent.click(screen.getByRole("button", { name: "Adopt markdown" })); }
beforeEach(() => {
  plans = [
    { id: "fixture-plan-id", slug: "fixture-plan", title: "Fixture canonical plan", updated_at: "2026-01-02T00:00:00Z", phase_count: 2 },
    { id: "fixture-old-id", slug: "older-owner", title: "Older owner title", created_at: "2026-01-01T00:00:00Z", phase_count: 1 },
    { id: "", slug: "invalid-owner", title: "Incomplete owner record", phase_count: 9 },
  ];
  listRefused = false; listDeferred = false; releaseList = undefined; expectedPosts = []; unexpected.length = 0;
  vi.spyOn(defaultApiClient, "get").mockImplementation(async (path, options) => {
    if (path !== "/plan-import/plans") { unexpected.push(path); throw new Error(`Unexpected import owner read ${path}`); }
    expect(options?.signal).toBeInstanceOf(AbortSignal);
    if (listRefused) throw new Error("Fixture plan catalog refused");
    if (listDeferred) return new Promise(resolve => { releaseList = resolve; });
    return { plans };
  });
  for (const method of ["post", "put", "patch", "delete"] as const) vi.spyOn(defaultApiClient, method).mockRejectedValue(new Error(`Unexpected import owner ${method}`));
  vi.stubGlobal("fetch", vi.fn(async () => { throw new Error("Unexpected direct import transport"); }));
});
afterEach(() => {
  unmounts.splice(0).forEach(unmount => unmount()); cleanup(); getSpatialNav()?.dispose();
  try {
    expect(unexpected).toEqual([]); expect(vi.mocked(defaultApiClient.post).mock.calls).toEqual(expectedPosts);
    for (const method of ["put", "patch", "delete"] as const) expect(defaultApiClient[method]).not.toHaveBeenCalled();
    expect(fetch).not.toHaveBeenCalled();
  } finally { clients.splice(0).forEach(client => client.clear()); vi.restoreAllMocks(); vi.unstubAllGlobals(); }
});

describe("Create work real canonical-plan selection and lifecycle", () => {
  it("does not acquire a canonical catalog or offer an import while closed", () => {
    mount(false); expect(defaultApiClient.get).not.toHaveBeenCalled(); expect(screen.queryByTestId("create-work-from-plan-dialog")).toBeNull();
  });
  it("renders pending catalog state without claiming an empty result and imports nothing", async () => {
    listDeferred = true; mount(); expect(await screen.findByText("Loading plans")).toBeVisible(); expect(screen.queryByText("No plans match this search.")).toBeNull(); expect(screen.getByTestId("create-work-from-plan-submit")).toBeDisabled();
    await waitFor(() => expect(releaseList).toBeDefined()); await act(async () => releaseList?.({ plans })); await loaded();
  });
  it("orders complete canonical records by owner date and filters title, slug and id without offering writes", async () => {
    mount(); await loaded(); const list = screen.getByTestId("create-work-plan-list");
    expect(within(list).getAllByRole("button").map(button => button.getAttribute("data-testid"))).toEqual(["create-work-plan-option-fixture-plan", "create-work-plan-option-older-owner"]);
    expect(screen.queryByText("Incomplete owner record")).toBeNull();
    for (const query of [" OLDER OWNER TITLE ", "older-owner", "fixture-old-id"]) { fireEvent.change(screen.getByTestId("create-work-plan-search"), { target: { value: query } }); expect(within(list).getAllByRole("button")).toHaveLength(1); expect(within(list).getByRole("button")).toHaveTextContent("Older owner title"); }
    fireEvent.change(screen.getByTestId("create-work-plan-search"), { target: { value: "no owner matches" } }); expect(screen.getByText("No plans match this search.")).toBeVisible(); expect(screen.getByTestId("create-work-from-plan-submit")).toBeDisabled();
  });
  it("keeps Create work disabled until an exact existing plan is chosen", async () => {
    mount(); await loaded(); fireEvent.click(screen.getByTestId("create-work-from-plan-submit")); expect(defaultApiClient.post).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("create-work-plan-option-older-owner")); expect(screen.getByText("1 phase selected.")).toBeVisible(); expect(screen.getByTestId("create-work-from-plan-submit")).not.toBeDisabled();
  });
  it("shows refused catalog reason and leaves Create work disabled without an alternate source read", async () => {
    listRefused = true; const view = mount(); expect(await screen.findByTestId("create-work-from-plan-error")).toHaveTextContent("Fixture plan catalog refused"); expect(screen.getByTestId("create-work-from-plan-submit")).toBeDisabled(); expect(view.imported).not.toHaveBeenCalled(); expect(defaultApiClient.get).toHaveBeenCalledOnce();
  });
  it("aborts only the original catalog read on parent close and refuses late callback effects", async () => {
    listDeferred = true; const view = mount(); await waitFor(() => expect(releaseList).toBeDefined()); const signal = vi.mocked(defaultApiClient.get).mock.calls[0]?.[1]?.signal; expect(signal?.aborted).toBe(false);
    fireEvent.click(screen.getByRole("button", { name: "Close" })); await waitFor(() => expect(signal?.aborted).toBe(true));
    await act(async () => releaseList?.({ plans })); expect(view.close).toHaveBeenCalledOnce(); expect(view.imported).not.toHaveBeenCalled(); expect(screen.queryByTestId("create-work-from-plan-dialog")).toBeNull();
  });
});

describe("Create work adapter payload, pending and refusal", () => {
  it("requires actual markdown or path rather than optional title/slug before offering import", async () => {
    mount(); await loaded(); chooseMarkdown(); fireEvent.change(screen.getByPlaceholderText("Plan title"), { target: { value: "Optional owner title" } }); fireEvent.change(screen.getByPlaceholderText("stable-slug"), { target: { value: "optional-owner" } });
    fireEvent.change(screen.getByTestId("create-work-markdown"), { target: { value: " \n " } }); fireEvent.change(screen.getByTestId("create-work-source-path"), { target: { value: "  " } });
    expect(screen.getByTestId("create-work-from-plan-submit")).toBeDisabled(); fireEvent.click(screen.getByTestId("create-work-from-plan-submit"));
  });
  it("imports a selected canonical plan with exact items container and publishes normalized result without auto closing", async () => {
    expectPost({ ...blankItems, plan_id: "fixture-plan-id" }); vi.mocked(defaultApiClient.post).mockResolvedValue(importResult);
    const view = mount(); fireEvent.click(await loaded()); fireEvent.click(screen.getByTestId("create-work-from-plan-submit"));
    expect(await screen.findByTestId("create-work-from-plan-success")).toHaveTextContent("1 created, 0 updated, 0 linked.");
    expect(view.imported).toHaveBeenCalledWith({ slug: "fixture-plan", planId: "fixture-plan-id", container: "items", items: importResult.items, milestone: undefined, count: 1, created: 1, updated: 0, linked: 0 }); expect(view.close).not.toHaveBeenCalled();
  });
  it("trims all markdown and milestone metadata into the exact existing import adapter fields", async () => {
    const body = { ...blankItems, source_path: "/fixture/nonsecret-plan.markdown", markdown: "# Fixture pasted plan", title: "Fixture title", slug: "fixture-slug", container: { type: "milestone", name: "fixture-milestone", title: "Fixture milestone title", description: "Fixture description" } }; expectPost(body); vi.mocked(defaultApiClient.post).mockResolvedValue({ ...importResult, container: "milestone", milestone: { name: "fixture-milestone", title: "Fixture milestone title", action: "created" } });
    mount(); await loaded(); chooseMarkdown();
    for (const [element, value] of [[screen.getByTestId("create-work-source-path"), " /fixture/nonsecret-plan.markdown "], [screen.getByTestId("create-work-markdown"), " # Fixture pasted plan "], [screen.getByPlaceholderText("Plan title"), " Fixture title "], [screen.getByPlaceholderText("stable-slug"), " fixture-slug "]] as const) fireEvent.change(element, { target: { value } });
    fireEvent.click(screen.getByRole("button", { name: /Milestone.*Create or update/ }));
    fireEvent.change(screen.getByPlaceholderText("milestone-name"), { target: { value: " fixture-milestone " } }); fireEvent.change(screen.getByPlaceholderText("Milestone title"), { target: { value: " Fixture milestone title " } }); fireEvent.change(screen.getByPlaceholderText("Milestone description"), { target: { value: " Fixture description " } });
    fireEvent.click(screen.getByTestId("create-work-from-plan-submit")); expect(await screen.findByTestId("create-work-result-links")).toHaveTextContent("Milestone: Fixture milestone title");
  });
  it("drops hidden milestone metadata when switched back to the items container", async () => {
    expectPost({ ...blankItems, plan_id: "fixture-plan-id" }); vi.mocked(defaultApiClient.post).mockResolvedValue(importResult); mount(); fireEvent.click(await loaded());
    fireEvent.click(screen.getByRole("button", { name: /Milestone.*Create or update/ })); fireEvent.change(screen.getByPlaceholderText("milestone-name"), { target: { value: "hidden-milestone" } });
    fireEvent.click(screen.getByRole("button", { name: /Backlog items.*Create or link/ })); expect(screen.queryByPlaceholderText("milestone-name")).toBeNull(); fireEvent.click(screen.getByTestId("create-work-from-plan-submit")); await screen.findByTestId("create-work-from-plan-success");
  });
  it("retains selected owner and prevents duplicate import or parent close while pending", async () => {
    expectPost({ ...blankItems, plan_id: "fixture-plan-id" }); let release!: (value: unknown) => void; vi.mocked(defaultApiClient.post).mockImplementation(() => new Promise(resolve => { release = resolve; }));
    const view = mount(); fireEvent.click(await loaded()); const submit = screen.getByTestId("create-work-from-plan-submit"); fireEvent.click(submit); fireEvent.click(submit);
    await waitFor(() => expect(defaultApiClient.post).toHaveBeenCalledOnce()); expect(submit).toBeDisabled(); const close = screen.getByRole("button", { name: "Close" }); expect(close).toBeDisabled(); fireEvent.click(close); expect(view.close).not.toHaveBeenCalled(); expect(view.imported).not.toHaveBeenCalled();
    await act(async () => release(importResult)); expect(await screen.findByTestId("create-work-from-plan-success")).toHaveTextContent("1 created"); expect(view.imported).toHaveBeenCalledOnce();
  });
  it("retains pasted draft and exposes owner refusal without imported callback or alternate write", async () => {
    expectPost({ ...blankItems, markdown: "# Retained fixture draft" }); vi.mocked(defaultApiClient.post).mockRejectedValue(new Error("Fixture import owner refused"));
    const view = mount(); await loaded(); chooseMarkdown(); fireEvent.change(screen.getByTestId("create-work-markdown"), { target: { value: "# Retained fixture draft" } }); fireEvent.click(screen.getByTestId("create-work-from-plan-submit"));
    expect(await screen.findByTestId("create-work-from-plan-error")).toHaveTextContent("Fixture import owner refused"); expect(screen.getByTestId("create-work-markdown")).toHaveValue("# Retained fixture draft"); expect(view.imported).not.toHaveBeenCalled(); expect(view.close).not.toHaveBeenCalled();
  });
  it("closes and reopens with retained draft but fresh catalog and cleared prior refusal", async () => {
    expectPost({ ...blankItems, markdown: "# Reopen fixture draft" }); vi.mocked(defaultApiClient.post).mockRejectedValue(new Error("Fixture previous import refused"));
    const view = mount(); await loaded(); chooseMarkdown(); fireEvent.change(screen.getByTestId("create-work-markdown"), { target: { value: "# Reopen fixture draft" } }); fireEvent.click(screen.getByTestId("create-work-from-plan-submit")); await screen.findByText("Fixture previous import refused");
    fireEvent.click(screen.getByRole("button", { name: "Close" })); await waitFor(() => expect(view.close).toHaveBeenCalledOnce()); fireEvent.click(screen.getByRole("button", { name: "Open fixture import" }));
    await waitFor(() => expect(defaultApiClient.get).toHaveBeenCalledTimes(2)); expect(screen.getByTestId("create-work-markdown")).toHaveValue("# Reopen fixture draft"); expect(screen.queryByText("Fixture previous import refused")).toBeNull(); expect(view.imported).not.toHaveBeenCalled();
  });
  it("limits result links and routes to the exact imported synthetic backlog owner", async () => {
    expectPost({ ...blankItems, plan_id: "fixture-plan-id" }); vi.mocked(defaultApiClient.post).mockResolvedValue({ ...importResult, count: 7, created: 7, items: Array.from({ length: 7 }, (_, index) => ({ kind: "execute", name: `fixture-phase-${index}`, title: `Phase ${index}`, action: "created" })) });
    const view = mount(); fireEvent.click(await loaded()); fireEvent.click(screen.getByTestId("create-work-from-plan-submit")); const links = await screen.findByTestId("create-work-result-links"); expect(within(links).getAllByRole("link")).toHaveLength(5); expect(within(links).getByText("+2 more")).toBeVisible();
    const link = within(links).getByRole("link", { name: "execute/fixture-phase-0" }); expect(link).toHaveAttribute("href", "/backlog/execute/fixture-phase-0"); fireEvent.click(link); expect(await screen.findByText("Exact imported backlog destination")).toBeVisible(); expect(screen.getByTestId("create-work-owner-location")).toHaveTextContent("/backlog/execute/fixture-phase-0"); expect(screen.queryByTestId("create-work-from-plan-dialog")).toBeNull(); expect(view.imported).toHaveBeenCalledOnce();
  });
});
