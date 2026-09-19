// provider-free-exception: The test uses a provider-free or feature-specific harness to isolate its boundary.
import { DesktopSessionsProvider } from "./useDesktopSession";
import { create } from "@bufbuild/protobuf";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, cleanup } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { ListResponseSchema } from "@vrooli/proto-types/portal/v1/surfaces/surfaces_pb";
import { SurfaceKind, SurfaceCapabilityState, SurfaceCapabilityFactSchema } from "@vrooli/proto-types/common/v1/surface_pb";
import { listSurfaces } from "../../api/surfaces";
import { SurfaceCatalog } from "./SurfaceCatalog";
import { strings } from "../../consts/strings";

const desktopTitle = "Office PC";
vi.mock("../../api/surfaces", async importOriginal => ({ ...await importOriginal<typeof import("../../api/surfaces")>(), listSurfaces: vi.fn() }));
const fixture = () => create(ListResponseSchema, {
 surfaces: ["node-a", "node-b"].map(id => ({
  ref: { target: { ownerScenario: "vrooli-bridge", resourceId: id, hostNodeId: id }, ownerScenario: "web-console", surfaceId: id },
  displayLabel: "Office PC", kind: SurfaceKind.TERMINAL, protocolVersions: ["vrooli.terminal.v1"],
  capabilities: [{ capability: "terminal.session", state: SurfaceCapabilityState.READY, evidenceId: "probe", observedAt: { seconds: BigInt(Math.floor(Date.now()/1000)-60) }, expiresAt: { seconds: BigInt(Math.floor(Date.now()/1000)-1) } }],
 })),
 sources: [{ ownerScenario: "web-console", state: "ready" }, { ownerScenario: "device-control", state: "unavailable" }],
});
function mount() {
 const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } });
 return render(<QueryClientProvider client={client}><DesktopSessionsProvider><SurfaceCatalog /></DesktopSessionsProvider></QueryClientProvider>);
}
beforeEach(() => { sessionStorage.clear(); vi.mocked(listSurfaces).mockReset(); });
afterEach(cleanup);
it("distinguishes duplicate names and retains exact selection when a source disappears", async () => { // [REQ:PORTAL-EVERYWHERE-CAT-03]
 const data = fixture(); vi.mocked(listSurfaces).mockResolvedValueOnce(data).mockResolvedValueOnce(create(ListResponseSchema));
 mount(); const choice = await screen.findByRole("option", { name: /Office PC.*node-b/ });
 expect(screen.getByRole("option", { name: /Office PC.*node-a/ })).toBeInTheDocument();
 fireEvent.change(screen.getByRole("combobox"), { target: { value: (choice as HTMLOptionElement).value } });
 expect(screen.getByText(/terminal.session: surfaces.stale/)).toBeInTheDocument();
 fireEvent.click(screen.getByRole("button", { name: strings.surfaces.refresh }));
 await screen.findByText(strings.surfaces.selectionMissing);
 expect(screen.getByRole("combobox")).toHaveValue((choice as HTMLOptionElement).value);
 expect(sessionStorage.getItem("portal.selected-surface.v1")).toContain("node-b");
});
it("retains loaded entries and reports a failed refresh without raw provider errors", async () => { // [REQ:PORTAL-EVERYWHERE-CAT-02]
 vi.mocked(listSurfaces).mockResolvedValueOnce(fixture()).mockRejectedValueOnce(new Error("private provider detail"));
 mount(); await screen.findByRole("option", { name: /node-a/ });
 fireEvent.click(screen.getByRole("button", { name: strings.surfaces.refresh }));
 await screen.findByRole("alert");
 expect(screen.getByRole("option", { name: /node-b/ })).toBeInTheDocument();
 expect(screen.queryByText(/private provider detail/)).not.toBeInTheDocument();
});
it("keeps the catalog usable without browser storage or providers", async () => { // [REQ:PORTAL-EVERYWHERE-OPT-01]
 const storage = vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => { throw new Error("storage denied"); });
 try {
  vi.mocked(listSurfaces).mockResolvedValue(create(ListResponseSchema)); mount();
  await screen.findByText(strings.surfaces.empty);
  expect(screen.getByRole("combobox")).toHaveAccessibleName(strings.surfaces.select);
 } finally { storage.mockRestore(); }
});

it("renders a screenless device panel from buttons and properties", async () => { // UI-07
 const data = fixture();
 data.surfaces[0]!.kind = SurfaceKind.DEVICE_PANEL;
 data.surfaces[0]!.capabilities.push(
  create(SurfaceCapabilityFactSchema, { capability: "device.button.power", state: SurfaceCapabilityState.READY, evidenceId: "button", observedAt: { seconds: BigInt(Math.floor(Date.now()/1000)) }, expiresAt: { seconds: BigInt(Math.floor(Date.now()/1000)+30) } }),
  create(SurfaceCapabilityFactSchema, { capability: "device.property.volume", state: SurfaceCapabilityState.READY, evidenceId: "property", observedAt: { seconds: BigInt(Math.floor(Date.now()/1000)) }, expiresAt: { seconds: BigInt(Math.floor(Date.now()/1000)+30) } }),
 );
 vi.mocked(listSurfaces).mockResolvedValue(data);
 mount();
 const choice = await screen.findByRole("option", { name: /Office PC.*node-a/ });
 fireEvent.change(screen.getByRole("combobox"), { target: { value: (choice as HTMLOptionElement).value } });
 expect(await screen.findByRole("region", { name: strings.surfaces.devicePanelTitle })).toBeInTheDocument();
 expect(screen.getByText(/^power$/)).toBeInTheDocument();
 expect(screen.getByText(/^volume$/)).toBeInTheDocument();
 expect(screen.getByText(strings.surfaces.devicePanelDescription)).toBeInTheDocument();
 expect(screen.queryByText(strings.surfaces.inspectOnly)).not.toBeInTheDocument();
});

it("embeds only provider-opted-in scenario surfaces through the same-origin proxy", async () => { // DEL-06
 const data = fixture();
 data.surfaces[0]!.kind = SurfaceKind.SCENARIO;
 data.surfaces[0]!.ref!.ownerScenario = "demo-scenario";
 data.surfaces[0]!.ref!.surfaceId = "workspace";
 data.surfaces[0]!.protocolVersions = ["vrooli.scenario.embed.v1"];
 vi.mocked(listSurfaces).mockResolvedValue(data);
 mount();
 const choice = await screen.findByRole("option", { name: /Office PC.*node-a/ });
 fireEvent.change(screen.getByRole("combobox"), { target: { value: (choice as HTMLOptionElement).value } });
 const frame = await screen.findByTitle(desktopTitle);
 expect(frame).toHaveAttribute("src", "/embedded/demo-scenario/");
 expect(frame).toHaveAttribute("sandbox", "allow-forms allow-scripts allow-same-origin");
 fireEvent.error(frame);
 expect(await screen.findByTestId("embedded-scenario-unavailable")).toBeInTheDocument();
});

it("does not invent an embed URL for a scenario without the provider contract", async () => { // DEL-06
 const data = fixture();
 data.surfaces[0]!.kind = SurfaceKind.SCENARIO;
 data.surfaces[0]!.ref!.ownerScenario = "demo-scenario";
 data.surfaces[0]!.protocolVersions = ["vrooli.scenario.v1"];
 vi.mocked(listSurfaces).mockResolvedValue(data);
 mount();
 const choice = await screen.findByRole("option", { name: /Office PC.*node-a/ });
 fireEvent.change(screen.getByRole("combobox"), { target: { value: (choice as HTMLOptionElement).value } });
 expect(await screen.findByText(strings.surfaces.inspectOnly)).toBeInTheDocument();
 expect(screen.queryByTitle(desktopTitle)).not.toBeInTheDocument();
});
