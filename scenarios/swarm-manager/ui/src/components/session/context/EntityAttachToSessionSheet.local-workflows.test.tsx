// Real sheet, owner store and read adapter. Local choices are cancelled before any
// attach, create, proposal, staging or navigation operation.
import { useState } from "react";
import { useLocation } from "react-router-dom";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import type { QueryClient } from "@tanstack/react-query";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { getSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { createTestQueryClient, renderWithProviders } from "../../../test-utils";
import { defaultApiClient } from "../../../lib/api-client";
import { useAgentSessionStore } from "../../../stores";
import { resetAgentSessionStoreService } from "../../../stores/agent-session-store";
import { selectors } from "../../../consts/selectors";
import { EntityAttachToSessionSheet } from "./EntityAttachToSessionSheet";
import type { SessionContextOption } from "./session-context-refs";
import { peekStagedContextForSession } from "./pending-session-context";

const option: SessionContextOption = { type: "backlog_item", ref: "fix/fixture-item", title: "Fixture owner", nodeId: "backlog-item/fix/fixture-item" };
const session = (id: string, title: string, kind = "meta_orchestration") => ({ id, title, kind, status: "draft", skill_id: "fixture-skill", run_id: `historical-${id}`, created_at: "2026-10-01T00:00:00Z", updated_at: "2026-10-01T00:00:00Z" });
const sessions = [session("fixture-current", "Current owner"), session("fixture-compatible", "Other owner"), session("fixture-incompatible", "Archived owner", "operating_mode_authoring")];
const views: ReturnType<typeof renderWithProviders>[] = []; const clients: QueryClient[] = [];
let closed: ReturnType<typeof vi.fn>; let unexpected: string[];
function Harness({ initiallyOpen = true, proposalMode = false }: { initiallyOpen?: boolean; proposalMode?: boolean }) {
  const [open, setOpen] = useState(initiallyOpen); const location = useLocation();
  return <><button onClick={() => setOpen(true)}>Open local sheet</button><output data-testid="local-sheet-route">{location.pathname}</output><EntityAttachToSessionSheet isOpen={open} onClose={() => { closed(); setOpen(false); }} option={option} currentSessionId="fixture-current" proposalMode={proposalMode} /></>;
}
function mount(props: Parameters<typeof Harness>[0] = {}) { const queryClient = createTestQueryClient(); clients.push(queryClient); const view = renderWithProviders(<Harness {...props} />, { queryClient }); views.push(view); return view; }
async function existing() { fireEvent.click(screen.getByTestId(selectors.agentSessions.entityAttachModeExisting)); await waitFor(() => expect(useAgentSessionStore.getState().sessions).toHaveLength(3)); }
function list() { return screen.getByTestId(selectors.agentSessions.entityAttachSessionList); }
beforeEach(() => {
  window.localStorage.clear(); useAgentSessionStore.getState().reset(); resetAgentSessionStoreService(); closed = vi.fn(); unexpected = [];
  vi.spyOn(defaultApiClient, "get").mockImplementation(async path => {
    if (path === "/agent-sessions?limit=100") return { sessions };
    unexpected.push(path); throw new Error(`Forbidden sheet read ${path}`);
  });
  for (const method of ["post", "put", "patch", "delete"] as const) vi.spyOn(defaultApiClient, method).mockRejectedValue(new Error(`Forbidden sheet ${method}`));
  vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("Forbidden direct sheet transport")));
});
afterEach(() => {
  views.splice(0).forEach(view => view.unmount()); clients.splice(0).forEach(client => client.clear()); getSpatialNav()?.dispose();
  try {
    expect(unexpected).toEqual([]); for (const method of ["post", "put", "patch", "delete"] as const) expect(defaultApiClient[method]).not.toHaveBeenCalled(); expect(fetch).not.toHaveBeenCalled();
    for (const s of sessions) expect(peekStagedContextForSession(s.id)).toEqual([]);
  } finally { useAgentSessionStore.getState().reset(); resetAgentSessionStoreService(); window.localStorage.clear(); vi.restoreAllMocks(); vi.unstubAllGlobals(); }
});
describe("session sheet local owner choices without attach or creation", () => {
  it("does not acquire sessions while closed", () => { mount({ initiallyOpen: false }); expect(screen.queryByTestId(selectors.agentSessions.entityAttachSheet)).toBeNull(); expect(defaultApiClient.get).not.toHaveBeenCalled(); });
  it("excludes the current owner from existing destinations", async () => { mount(); await existing(); expect(within(list()).queryByText("Current owner")).toBeNull(); expect(within(list()).getByText("Other owner")).toBeVisible(); expect(within(list()).getByText("Archived owner")).toBeVisible(); expect(screen.getByTestId(selectors.agentSessions.entityAttachConfirm)).toBeDisabled(); });
  it.each(["OTHER OWNER", "historical-fixture-compatible"])("filters exact historical session fields case-insensitively (%s)", async query => { mount(); await existing(); fireEvent.change(screen.getByTestId(selectors.agentSessions.entityAttachSearch), { target: { value: ` ${query} ` } }); expect(within(list()).getByText("Other owner")).toBeVisible(); expect(within(list()).queryByText("Archived owner")).toBeNull(); });
  it("retains no-match cancellation without navigation or staging", async () => { mount(); await existing(); fireEvent.change(screen.getByTestId(selectors.agentSessions.entityAttachSearch), { target: { value: "missing owner" } }); expect(list()).toHaveTextContent("No matching sessions."); expect(screen.getByTestId(selectors.agentSessions.entityAttachConfirm)).toBeDisabled(); fireEvent.click(screen.getByRole("button", { name: "Cancel" })); expect(closed).toHaveBeenCalledTimes(1); expect(screen.getByTestId("local-sheet-route")).toHaveTextContent("/"); });
  it("allows a local compatible selection then cancels before Attach", async () => { mount(); await existing(); const checkbox = within(list()).getAllByRole("checkbox").find(node => !node.hasAttribute("disabled")); expect(checkbox).toBeDefined(); fireEvent.click(checkbox!); expect(screen.getByTestId(selectors.agentSessions.entityAttachConfirm)).toBeEnabled(); expect(screen.getByText("Selected Other owner")).toBeVisible(); fireEvent.click(screen.getByRole("button", { name: "Cancel" })); expect(closed).toHaveBeenCalledTimes(1); expect(screen.getByTestId("local-sheet-route")).toHaveTextContent("/"); });
  it("keeps an incompatible destination disabled", async () => { mount(); await existing(); const disabled = within(list()).getAllByRole("checkbox").filter(node => node.hasAttribute("disabled")); expect(disabled).toHaveLength(1); fireEvent.click(disabled[0]!); expect(screen.getByTestId(selectors.agentSessions.entityAttachConfirm)).toBeDisabled(); });
  it("clears search and local destination selection on reopening", async () => { mount(); await existing(); fireEvent.change(screen.getByTestId(selectors.agentSessions.entityAttachSearch), { target: { value: "Other owner" } }); fireEvent.click(screen.getByRole("button", { name: "Cancel" })); fireEvent.click(screen.getByRole("button", { name: "Open local sheet" })); await existing(); expect(screen.getByTestId(selectors.agentSessions.entityAttachSearch)).toHaveValue(""); expect(within(list()).getByText("Archived owner")).toBeVisible(); expect(screen.getByTestId(selectors.agentSessions.entityAttachConfirm)).toBeDisabled(); });
  it("returns to fresh-session choices without creating a draft", async () => { mount(); await existing(); fireEvent.click(screen.getByTestId(selectors.agentSessions.entityAttachModeNew)); expect(screen.queryByTestId(selectors.agentSessions.entityAttachSearch)).toBeNull(); expect(screen.getByTestId(selectors.agentSessions.entityAttachKindSelect)).toBeVisible(); expect(screen.getByTestId(selectors.agentSessions.entityAttachQuickStart)).toHaveTextContent("Draft session"); fireEvent.click(screen.getByRole("button", { name: "Cancel" })); expect(closed).toHaveBeenCalledTimes(1); });
  it("limits proposal framing while leaving existing local destinations available", async () => { mount({ proposalMode: true }); expect(screen.getByTestId(selectors.agentSessions.entityAttachQuickStart)).toHaveTextContent("Start proposal"); expect(screen.queryByTestId(selectors.agentSessions.entityAttachKindSelect)).toBeNull(); expect(screen.getByText("Proposal sessions use managed Swarm Operations and always produce a reviewable mutation list.")).toBeVisible(); await existing(); expect(within(list()).getByText("Other owner")).toBeVisible(); fireEvent.click(screen.getByRole("button", { name: "Cancel" })); expect(closed).toHaveBeenCalledTimes(1); });
});
