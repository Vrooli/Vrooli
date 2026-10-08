// provider-free-exception: The test uses a provider-free or feature-specific harness to isolate its boundary.
/* eslint-disable @typescript-eslint/require-await, no-restricted-syntax */
import { desktopRecoveryKey } from "./desktopRecoveryMarker";
import { DesktopSessionsProvider, useDesktopSession } from "./useDesktopSession";
import { OwnerCleanupResponseSchema, OwnerListAdmissionsResponseSchema, OwnerOpenDispositionSchema } from "@vrooli/proto-types/device-control/v1/desktop/desktop_pb";
import { create } from "@bufbuild/protobuf";
import { fireEvent, render as renderUI, screen, waitFor, cleanup, act } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { LoginResponseSchema, RefreshResponseSchema } from "@vrooli/proto-types/scenario-authenticator/v1/accounts/accounts_pb";
import { ApplicationsResponseSchema, OwnerOpenResponseSchema, ObserveResponseSchema, ActResponseSchema, StopResponseSchema } from "@vrooli/proto-types/device-control/v1/desktop/desktop_pb";
import { SurfaceRefSchema } from "@vrooli/proto-types/common/v1/surface_pb";
import { DesktopSessionPanel, DesktopRecoveryPanel } from "./DesktopSessionPanel";
import { desktopClient, operatorClient } from "../../api/desktop";
import { strings } from "../../consts/strings";
vi.mock("../../api/desktop", () => ({ desktopClient: { reconcileOpen: vi.fn(), listAdmissions: vi.fn(), readCleanup: vi.fn(), open: vi.fn(), applications: vi.fn(), observe: vi.fn(), act: vi.fn(), stop: vi.fn() }, operatorClient: { refresh: vi.fn(), login: vi.fn(), register: vi.fn(), logout: vi.fn() }, operatorHeaders: (token: string) => ({ Authorization: `Bearer ${token}` }) }));
const render = (ui: React.ReactNode) => renderUI(ui, { wrapper: DesktopSessionsProvider });
const surface = create(SurfaceRefSchema, { ownerScenario: "device-control", surfaceId: "desktop", target: { ownerScenario: "vrooli-bridge", resourceId: "host", hostNodeId: "host" } });
const opened = (control = false) => create(OwnerOpenResponseSchema, { session: { surface, sessionId: "lease", desktopSessionId: "2" }, control, expiresAt: { seconds: BigInt(Math.floor(Date.now() / 1000) + 60) } });
const frame = () => create(ObserveResponseSchema, { png: new Uint8Array([1, 2]), width: 1000, height: 500, displayId: "display", geometryRevision: "revision" });
const catalog = () => create(ApplicationsResponseSchema, { revision: "catalog", expiresAt: { seconds: BigInt(Math.floor(Date.now()/1000)+30) }, applications: [{ applicationId: "app-1", name: "Editor", processId: 42 }, { applicationId: "app-2", name: "Editor", processId: 43 }] });
beforeEach(() => {
 vi.clearAllMocks(); localStorage.clear(); sessionStorage.clear();
 vi.stubGlobal("URL", Object.assign(URL, { createObjectURL: vi.fn(() => "blob:frame"), revokeObjectURL: vi.fn() }));
 vi.mocked(operatorClient.login).mockResolvedValue(create(LoginResponseSchema, { account: { id: "owner-id", realm: "default", email: "owner@example.test" }, tokens: { accessToken: "account-token", refreshToken: "refresh-token", accessTokenExpiresAt: { seconds: BigInt(Math.floor(Date.now() / 1000) + 120) } } }));
 vi.mocked(desktopClient.reconcileOpen).mockRejectedValue(new Error("reconciliation unavailable"));
 vi.mocked(desktopClient.listAdmissions).mockResolvedValue(create(OwnerListAdmissionsResponseSchema));
 vi.mocked(desktopClient.open).mockResolvedValue(opened());
 vi.mocked(desktopClient.observe).mockResolvedValue(frame());
 vi.mocked(desktopClient.applications).mockResolvedValue(catalog());
 vi.mocked(desktopClient.stop).mockResolvedValue(create(StopResponseSchema));
});
afterEach(() => { cleanup(); vi.unstubAllGlobals(); });
async function login() {
 fireEvent.change(screen.getByLabelText(strings.desktop.email), { target: { value: "owner@example.test" } });
 fireEvent.change(screen.getByLabelText(strings.desktop.password), { target: { value: "password" } });
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.signIn }));
 await waitFor(() => expect(screen.getByRole("button", { name: strings.desktop.observe })).toBeEnabled());
}
it("keeps account tokens in memory and observation cannot inject input", async () => {
 const onActive = vi.fn(); render(<DesktopSessionPanel surface={surface} onActive={onActive} />);
 await login();
 expect(localStorage.length).toBe(0); expect(sessionStorage.length).toBe(0);
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.observe }));
 const image = await screen.findByRole("img");
 expect(desktopClient.open).toHaveBeenCalledWith(expect.objectContaining({ surface, ttlSeconds: 120, control: false, requestId: expect.any(String) }), { headers: { Authorization: "Bearer account-token" } });
 fireEvent.click(image); expect(desktopClient.act).not.toHaveBeenCalled();
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.stop }));
 await waitFor(() => expect(screen.queryByRole("img")).not.toBeInTheDocument());
 expect(desktopClient.stop).toHaveBeenCalled();
});
it("uses observed geometry and suppresses more clicks after an uncertain outcome", async () => {
 vi.mocked(desktopClient.open).mockResolvedValue(opened(true));
 vi.mocked(desktopClient.act).mockResolvedValue(create(ActResponseSchema, { receipt: { outcome: "outcome_unknown" } }));
 render(<DesktopSessionPanel surface={surface} onActive={vi.fn()} />); await login();
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.control }));
 const image = await screen.findByRole("img");
 vi.spyOn(image.closest("button")!, "getBoundingClientRect").mockReturnValue({ left: 10, top: 20, width: 200, height: 100 } as DOMRect);
 fireEvent.click(image, { clientX: 60, clientY: 40, detail: 1 });
 await screen.findByText(strings.desktop.uncertain);
 expect(desktopClient.act).toHaveBeenCalledWith(expect.objectContaining({ geometryRevision: "revision", action: { action: { case: "pointer", value: expect.objectContaining({ displayId: "display", x: 250, y: 100 }) } } }), expect.anything());
 fireEvent.click(image, { clientX: 60, clientY: 40, detail: 1 }); expect(desktopClient.act).toHaveBeenCalledTimes(1);
});
it("retains a session if Open completes after the routed panel unmounts", async () => {
 let resolve!: (value: ReturnType<typeof opened>) => void;
 vi.mocked(desktopClient.open).mockReturnValue(new Promise(r => { resolve = r; }));
 const view = render(<DesktopSessionPanel surface={surface} onActive={vi.fn()} />); await login();
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.observe })); view.rerender(<div>Another route</div>);
 await act(async () => { resolve(opened()); await Promise.resolve(); });
 expect(desktopClient.stop).not.toHaveBeenCalled();
 expect(desktopClient.observe).not.toHaveBeenCalled();
 view.rerender(<DesktopSessionPanel surface={surface} onActive={vi.fn()} />);
 expect(screen.getByRole("button", { name: strings.desktop.stop })).toBeEnabled();
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.stop }));
 await waitFor(() => expect(desktopClient.stop).toHaveBeenCalledTimes(1));
});
it("scrolls the observed center and blocks further input after an uncertain wheel result", async () => {
 vi.mocked(desktopClient.open).mockResolvedValue(opened(true));
 vi.mocked(desktopClient.act).mockResolvedValue(create(ActResponseSchema, { receipt: { outcome: "outcome_unknown" } }));
 render(<DesktopSessionPanel surface={surface} onActive={vi.fn()} />); await login();
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.control }));
 const down = await screen.findByRole("button", { name: strings.desktop.scrollDown });
 fireEvent.click(down);
 await screen.findByText(strings.desktop.uncertain);
 expect(desktopClient.act).toHaveBeenCalledWith(expect.objectContaining({ geometryRevision: "revision", action: { action: { case: "wheel", value: { displayId: "display", x: 500, y: 250, horizontalTicks: 0, verticalTicks: 3 } } } }), expect.anything());
 expect(down).toBeDisabled();
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.scrollUp }));
 expect(desktopClient.act).toHaveBeenCalledTimes(1);
});
it("inserts prepared Unicode into the exact observed field", async () => {
 vi.mocked(desktopClient.open).mockResolvedValue(opened(true));
 vi.mocked(desktopClient.act).mockResolvedValue(create(ActResponseSchema, { receipt: { outcome: "applied" } }));
 render(<DesktopSessionPanel surface={surface} onActive={vi.fn()} />); await login();
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.control }));
 await screen.findByRole("img");
 fireEvent.click(screen.getByText(strings.desktop.applicationFields));
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.discoverApplications }));
 await screen.findByRole("option", { name: "Editor · 2" });
 fireEvent.change(screen.getByLabelText(strings.desktop.application), { target: { value: "app-2" } });
 fireEvent.change(screen.getByLabelText(strings.desktop.insertDraft), { target: { value: "日本語 🧪" } });
 vi.mocked(desktopClient.observe).mockResolvedValue(create(ObserveResponseSchema, { png: new Uint8Array([1, 2]), width: 1000, height: 500, displayId: "display", geometryRevision: "revision", semantic: { revision: "semantic-revision", processId: 42, expiresAt: { seconds: BigInt(Math.floor(Date.now()/1000)+5) }, elements: [{ elementId: "window-1", name: "Editor", windowId: "window-1" }, { elementId: "window-2", name: "Editor", windowId: "window-2" }, { elementId: "first", name: "Entry", editable: true, parentId: "window-1", windowId: "window-1" }, { elementId: "second", name: "Entry", editable: true, parentId: "window-2", windowId: "window-2" }] } }));
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.inspectApplication }));
 await screen.findByLabelText(strings.desktop.window);
 expect(screen.getByLabelText(strings.desktop.editableField)).toBeDisabled();
 fireEvent.change(screen.getByLabelText(strings.desktop.window), { target: { value: "window-1" } });
 fireEvent.change(screen.getByLabelText(strings.desktop.editableField), { target: { value: "first" } });
 fireEvent.change(screen.getByLabelText(strings.desktop.window), { target: { value: "window-2" } });
 expect(screen.getByLabelText(strings.desktop.editableField)).toHaveValue("");
 expect(screen.getByRole("button", { name: strings.desktop.insertText })).toBeDisabled();
 expect(screen.getAllByRole("option", { name: "Entry · 1" })).toHaveLength(1);
 expect(desktopClient.observe).toHaveBeenLastCalledWith({ session: opened(true).session, applicationId: "app-2", applicationRevision: "catalog" }, expect.anything());
 fireEvent.change(screen.getByLabelText(strings.desktop.editableField), { target: { value: "second" } });
 fireEvent.change(screen.getByLabelText(strings.desktop.characterPosition), { target: { value: "2" } });
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.insertText }));
 await waitFor(() => expect(desktopClient.act).toHaveBeenCalled());
 expect(desktopClient.act).toHaveBeenCalledWith(expect.objectContaining({ action: { action: { case: "text", value: { text: "日本語 🧪", elementId: "second", observationRevision: "semantic-revision", position: 2 } } } }), expect.anything());
 await waitFor(() => expect(screen.getByLabelText(strings.desktop.insertDraft)).toHaveValue(""));
 expect(desktopClient.observe).toHaveBeenLastCalledWith({ session: opened(true).session }, expect.anything());
 fireEvent.change(screen.getByLabelText(strings.desktop.application), { target: { value: "app-1" } });
 expect(screen.queryByLabelText(strings.desktop.editableField)).not.toBeInTheDocument();

});
it("refuses insertion through expired semantic observations", async () => {
 vi.mocked(desktopClient.open).mockResolvedValue(opened(true));
 render(<DesktopSessionPanel surface={surface} onActive={vi.fn()} />); await login();
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.control })); await screen.findByRole("img");
 fireEvent.click(screen.getByText(strings.desktop.applicationFields));
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.discoverApplications }));
 await screen.findByRole("option", { name: "Editor · 2" });
 fireEvent.change(screen.getByLabelText(strings.desktop.application), { target: { value: "app-2" } });
 vi.mocked(desktopClient.observe).mockResolvedValue(create(ObserveResponseSchema, { png: new Uint8Array([1, 2]), width: 1000, height: 500, displayId: "display", geometryRevision: "revision", semantic: { revision: "expired", processId: 42, expiresAt: { seconds: 1n }, elements: [{ elementId: "field", editable: true }] } }));
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.inspectApplication }));
 await screen.findByText(strings.desktop.fieldsExpired);
 expect(screen.getByRole("button", { name: strings.desktop.insertText })).toBeDisabled();
 expect(desktopClient.act).not.toHaveBeenCalled();
});

it("requires rediscovery after catalog expiry and never guesses an application", async () => {
 vi.mocked(desktopClient.applications).mockResolvedValue(create(ApplicationsResponseSchema,{revision:"expired",expiresAt:{seconds:1n},applications:[{applicationId:"old",name:"Editor",processId:42}]}));
 render(<DesktopSessionPanel surface={surface} onActive={vi.fn()} />); await login();
 fireEvent.click(screen.getByRole("button",{name:strings.desktop.observe})); await screen.findByRole("img");
 fireEvent.click(screen.getByText(strings.desktop.applicationFields));
 fireEvent.click(screen.getByRole("button",{name:strings.desktop.discoverApplications}));
 await screen.findByText(strings.desktop.applicationsExpired);
 expect(screen.getByLabelText(strings.desktop.application)).toBeDisabled();
 expect(screen.getByRole("button",{name:strings.desktop.inspectApplication})).toBeDisabled();
 expect(desktopClient.observe).toHaveBeenCalledTimes(1);
});
it("discards a catalog returned after Stop", async () => {
 let resolve!: (value: ReturnType<typeof catalog>) => void;
 vi.mocked(desktopClient.applications).mockReturnValue(new Promise(r=>{resolve=r;}));
 render(<DesktopSessionPanel surface={surface} onActive={vi.fn()} />); await login();
 fireEvent.click(screen.getByRole("button",{name:strings.desktop.observe})); await screen.findByRole("img");
 fireEvent.click(screen.getByText(strings.desktop.applicationFields));
 fireEvent.click(screen.getByRole("button",{name:strings.desktop.discoverApplications}));
 fireEvent.click(screen.getByRole("button",{name:strings.desktop.stop}));
 await act(async()=>{resolve(catalog());await Promise.resolve();});
 expect(screen.queryByLabelText(strings.desktop.application)).not.toBeInTheDocument();
 expect(screen.queryByRole("img")).not.toBeInTheDocument();
});

it("reads the exact lease after a lost Stop response without repeating Stop", async () => {
 vi.mocked(desktopClient.stop).mockRejectedValueOnce(new Error("lost response"));
 render(<DesktopSessionPanel surface={surface} onActive={vi.fn()} />); await login();
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.control }));
 await screen.findByRole("img");
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.stop }));
 await screen.findByText(strings.desktop.uncertain);
 vi.mocked(desktopClient.readCleanup).mockResolvedValue(create(OwnerCleanupResponseSchema,{session:opened().session,released:true,observedAt:{seconds:1n}}));
 fireEvent.click(screen.getByRole("button", { name: strings.companion.checkStatus }));
 await waitFor(() => expect(screen.queryByRole("button", { name: strings.companion.checkStatus })).not.toBeInTheDocument());
 expect(desktopClient.stop).toHaveBeenCalledTimes(1);
 expect(vi.mocked(desktopClient.readCleanup).mock.calls[0]?.[0]).toEqual(vi.mocked(desktopClient.stop).mock.calls[0]?.[0]);
});
it("keeps an expired lease unresolved instead of treating the local clock as Stop confirmation", async () => {
 const onActive=vi.fn();render(<DesktopSessionPanel surface={surface} onActive={onActive}/>);await login();
 fireEvent.click(screen.getByRole("button",{name:strings.desktop.control}));await screen.findByRole("img");
 const clock=vi.spyOn(Date,"now").mockReturnValue(Date.now()+61_000);
 try {
  await screen.findByText(strings.desktop.uncertain);
  expect(screen.getByRole("button",{name:strings.companion.checkStatus})).toBeInTheDocument();
  expect(screen.queryByRole("img")).not.toBeInTheDocument();
  expect(onActive).toHaveBeenLastCalledWith(true);
  expect(desktopClient.stop).not.toHaveBeenCalled();
 } finally {clock.mockRestore();}
});
it("serializes Stop and discards an observation returned while Stop is pending", async () => {
 let finishObserve!:(value:ReturnType<typeof frame>)=>void;
 let failStop!:(error:Error)=>void;
 render(<DesktopSessionPanel surface={surface} onActive={vi.fn()}/>);await login();
 fireEvent.click(screen.getByRole("button",{name:strings.desktop.control}));await screen.findByRole("img");
 vi.mocked(desktopClient.observe).mockReturnValueOnce(new Promise(resolve=>{finishObserve=resolve;}));
 vi.mocked(desktopClient.stop).mockReturnValueOnce(new Promise((_resolve,reject)=>{failStop=reject;}));
 fireEvent.click(screen.getByRole("button",{name:strings.desktop.refresh}));
 const stop=screen.getByRole("button",{name:strings.desktop.stop});fireEvent.click(stop);fireEvent.click(stop);
 expect(desktopClient.stop).toHaveBeenCalledTimes(1);
 await act(async()=>{finishObserve(frame());await Promise.resolve();});
 expect(screen.queryByRole("img")).not.toBeInTheDocument();
 await act(async()=>{failStop(new Error("lost reply"));await Promise.resolve();});
 expect(screen.getByRole("button",{name:strings.companion.checkStatus})).not.toBeDisabled();
 expect(screen.getByText(strings.desktop.uncertain)).toBeInTheDocument();
 expect(screen.queryByRole("img")).not.toBeInTheDocument();
});
it.each(["pending","mismatch","missing","unavailable"])("retains the lease after %s cleanup readback and never repeats Stop on unmount",async outcome=>{
 vi.mocked(desktopClient.stop).mockRejectedValueOnce(new Error("lost reply"));
 if(outcome==="unavailable")vi.mocked(desktopClient.readCleanup).mockRejectedValueOnce(new Error("offline"));
 else vi.mocked(desktopClient.readCleanup).mockResolvedValueOnce(create(OwnerCleanupResponseSchema,{session:outcome==="missing"?undefined:{...opened().session!,sessionId:outcome==="mismatch"?"other":"lease"},released:outcome!=="pending",observedAt:{seconds:1n}}));
 const view=render(<DesktopSessionPanel surface={surface} onActive={vi.fn()}/>);await login();
 fireEvent.click(screen.getByRole("button",{name:strings.desktop.control}));await screen.findByRole("img");
 fireEvent.click(screen.getByRole("button",{name:strings.desktop.stop}));
 await waitFor(()=>expect(screen.getByRole("button",{name:strings.companion.checkStatus})).not.toBeDisabled());
 fireEvent.click(screen.getByRole("button",{name:strings.companion.checkStatus}));
 await waitFor(()=>expect(desktopClient.readCleanup).toHaveBeenCalledTimes(1));
 await waitFor(()=>expect(screen.getByRole("button",{name:strings.companion.checkStatus})).not.toBeDisabled());
 expect(screen.queryByRole("img")).not.toBeInTheDocument();
 expect(desktopClient.stop).toHaveBeenCalledTimes(1);
 view.unmount();expect(desktopClient.stop).toHaveBeenCalledTimes(1);
});

it("retains uncertain cleanup across route navigation and reads without a second Stop", async () => {
 vi.mocked(desktopClient.stop).mockRejectedValueOnce(new Error("lost Stop reply"));
 const view = render(<DesktopSessionPanel surface={surface} onActive={vi.fn()} />); await login();
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.observe })); await screen.findByRole("img");
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.stop }));
 await screen.findByText(strings.desktop.uncertain);
 view.rerender(<div>Another route</div>);
 view.rerender(<DesktopSessionPanel surface={surface} onActive={vi.fn()} />);
 expect(screen.queryByRole("img")).not.toBeInTheDocument();
 expect(screen.getByText(strings.desktop.uncertain)).toBeInTheDocument();
 vi.mocked(desktopClient.readCleanup).mockResolvedValueOnce(create(OwnerCleanupResponseSchema, { session: opened().session, released: true, observedAt: { seconds: 1n } }));
 fireEvent.click(screen.getByRole("button", { name: strings.companion.checkStatus }));
 await screen.findByRole("button", { name: strings.desktop.observe });
 expect(desktopClient.stop).toHaveBeenCalledTimes(1);
 expect(desktopClient.readCleanup).toHaveBeenCalledTimes(1);
 expect(desktopClient.observe).toHaveBeenCalledTimes(1);
});

it("ignores a departed panel's late input outcome after a successor session opens", async () => {
 vi.mocked(desktopClient.open).mockResolvedValueOnce(opened(true));
 let finish!: (value: ReturnType<typeof create<typeof ActResponseSchema>>) => void;
 vi.mocked(desktopClient.act).mockReturnValueOnce(new Promise(resolve => { finish = resolve; }));
 const view = render(<DesktopSessionPanel surface={surface} onActive={vi.fn()} />); await login();
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.control }));
 const image = await screen.findByRole("img");
 vi.spyOn(image.closest("button")!, "getBoundingClientRect").mockReturnValue({ left: 0, top: 0, width: 200, height: 100 } as DOMRect);
 fireEvent.click(image, { detail: 0 });
 await waitFor(() => expect(desktopClient.act).toHaveBeenCalledTimes(1));
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.stop }));
 await waitFor(() => expect(screen.queryByRole("img")).not.toBeInTheDocument());
 view.rerender(<div>Another route</div>);
 view.rerender(<DesktopSessionPanel surface={surface} onActive={vi.fn()} />);
 const successor = opened(true); successor.session!.sessionId = "successor";
 vi.mocked(desktopClient.open).mockResolvedValueOnce(successor);
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.control })); await screen.findByRole("img");
 await act(async () => { finish(create(ActResponseSchema, { receipt: { outcome: "outcome_unknown" } })); });
 expect(screen.queryByText(strings.desktop.uncertain)).not.toBeInTheDocument();
 expect(screen.getByRole("button", { name: strings.desktop.clickImage })).toBeEnabled();
});


it("allows shell controls to stop a lease while its routed panel is absent", async () => {
 function ShellStop() {
  const session = useDesktopSession();
  return <button onClick={() => void session.store.stop()} disabled={!session.active}>Shell stop</button>;
 }
 const view = render(<><ShellStop /><DesktopSessionPanel surface={surface} onActive={vi.fn()} /></>); await login();
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.observe })); await screen.findByRole("img");
 view.rerender(<><ShellStop /><div>Another route</div></>);
 expect(desktopClient.stop).not.toHaveBeenCalled();
 fireEvent.click(screen.getByRole("button", { name: "Shell stop" }));
 await waitFor(() => expect(screen.getByRole("button", { name: "Shell stop" })).toBeDisabled());
 expect(desktopClient.stop).toHaveBeenCalledTimes(1);
 view.rerender(<><ShellStop /><DesktopSessionPanel surface={surface} onActive={vi.fn()} /></>);
 expect(screen.getByRole("button", { name: strings.desktop.observe })).toBeEnabled();
});

async function signInForRecovery() {
 fireEvent.change(screen.getByLabelText(strings.desktop.email), { target: { value: "owner@example.test" } });
 fireEvent.change(screen.getByLabelText(strings.desktop.password), { target: { value: "password" } });
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.signIn }));
 await screen.findByText(strings.desktop.signedIn);
}

it("gates admission until discovery completes and retains every unresolved historical session", async () => {
 const second = opened(); second.session!.sessionId = "second";
 vi.mocked(desktopClient.listAdmissions).mockResolvedValueOnce(create(OwnerListAdmissionsResponseSchema, { admissions: [opened(), second].map(value => ({ session: value.session, control: value.control, expiresAt: value.expiresAt })) }));
 vi.mocked(desktopClient.readCleanup).mockImplementation(async request => create(OwnerCleanupResponseSchema, { session: request.session, released: request.session?.sessionId === "lease", observedAt: { seconds: 1n } }));
 render(<DesktopSessionPanel surface={surface} onActive={vi.fn()} />); await signInForRecovery();
 await waitFor(() => expect(desktopClient.readCleanup).toHaveBeenCalledTimes(2));
 expect(screen.getByRole("button", { name: strings.desktop.control })).toBeDisabled();
 expect(desktopClient.open).not.toHaveBeenCalled(); expect(desktopClient.stop).not.toHaveBeenCalled();
 // A later empty list must not erase the still-unconfirmed second identity.
 vi.mocked(desktopClient.readCleanup).mockImplementation(async request => create(OwnerCleanupResponseSchema, { session: request.session, released: true, observedAt: { seconds: 1n } }));
 fireEvent.click(screen.getByRole("button", { name: strings.companion.checkStatus }));
 await waitFor(() => expect(screen.getByRole("button", { name: strings.desktop.control })).toBeEnabled());
 expect(desktopClient.readCleanup).toHaveBeenLastCalledWith({ session: second.session }, expect.anything());
 expect(desktopClient.stop).not.toHaveBeenCalled();
});

it("retains partial discovery and rejects repeated pagination cursors", async () => {
 vi.mocked(desktopClient.listAdmissions).mockResolvedValueOnce(create(OwnerListAdmissionsResponseSchema, { admissions: [{ session: opened().session, control: false, expiresAt: opened().expiresAt }], nextPageToken: "1" })).mockResolvedValueOnce(create(OwnerListAdmissionsResponseSchema, { nextPageToken: "1" }));
 render(<DesktopSessionPanel surface={surface} onActive={vi.fn()} />); await signInForRecovery();
 await waitFor(() => expect(screen.getByRole("button", { name: strings.companion.checkStatus })).toBeEnabled());
 expect(desktopClient.listAdmissions).toHaveBeenCalledTimes(2);
 expect(screen.getByRole("button", { name: strings.desktop.control })).toBeDisabled();
 vi.mocked(desktopClient.readCleanup).mockResolvedValueOnce(create(OwnerCleanupResponseSchema, { session: opened().session, released: true, observedAt: { seconds: 1n } }));
 fireEvent.click(screen.getByRole("button", { name: strings.companion.checkStatus }));
 await waitFor(() => expect(screen.getByRole("button", { name: strings.desktop.control })).toBeEnabled());
 expect(desktopClient.readCleanup).toHaveBeenCalledWith({ session: opened().session }, expect.anything());
 expect(desktopClient.open).not.toHaveBeenCalled(); expect(desktopClient.stop).not.toHaveBeenCalled();
});

it("keeps unknown Open unresolved when request reconciliation is unavailable", async () => {
 vi.mocked(desktopClient.open).mockRejectedValueOnce(new Error("lost Open reply"));
 render(<DesktopSessionPanel surface={surface} onActive={vi.fn()} />); await login();
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.control }));
 await screen.findByText(strings.desktop.uncertain);
 fireEvent.click(screen.getByRole("button", { name: strings.companion.checkStatus }));
 await waitFor(() => expect(desktopClient.reconcileOpen).toHaveBeenCalledTimes(1));
 expect(screen.getByRole("button", { name: strings.desktop.control })).toBeDisabled();
 expect(desktopClient.open).toHaveBeenCalledTimes(1); expect(desktopClient.stop).not.toHaveBeenCalled();
});


function RefreshAccount() {
 const session = useDesktopSession();
 return <button onClick={() => void session.store.refresh()}>Refresh account</button>;
}
it("serializes refresh token rotation and preserves the live lease", async () => {
 let finish!: (value: ReturnType<typeof create<typeof RefreshResponseSchema>>) => void;
 vi.mocked(operatorClient.refresh).mockReturnValueOnce(new Promise(resolve => { finish = resolve; }));
 render(<><RefreshAccount /><DesktopSessionPanel surface={surface} onActive={vi.fn()} /></>); await login();
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.observe })); await screen.findByRole("img");
 const refresh = screen.getByRole("button", { name: "Refresh account" }); fireEvent.click(refresh); fireEvent.click(refresh);
 expect(operatorClient.refresh).toHaveBeenCalledTimes(1);
 expect(operatorClient.refresh).toHaveBeenCalledWith({ refreshToken: "refresh-token" });
 await act(async () => { finish(create(RefreshResponseSchema, { tokens: { accessToken: "rotated-access", refreshToken: "rotated-refresh", accessTokenExpiresAt: { seconds: BigInt(Math.floor(Date.now()/1000)+120) } } })); });
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.stop }));
 await screen.findByRole("button", { name: strings.desktop.observe });
 expect(desktopClient.stop).toHaveBeenCalledWith({ session: opened().session }, { headers: { Authorization: "Bearer rotated-access" } });
 expect(localStorage.length).toBe(0); expect(sessionStorage.length).toBe(0);
});

it("requires the original actor after refresh fails without losing uncertain cleanup", async () => {
 vi.mocked(operatorClient.refresh).mockRejectedValueOnce(new Error("lost refresh reply"));
 vi.mocked(desktopClient.stop).mockRejectedValueOnce(new Error("lost Stop reply"));
 render(<><RefreshAccount /><DesktopSessionPanel surface={surface} onActive={vi.fn()} /></>); await login();
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.observe })); await screen.findByRole("img");
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.stop })); await screen.findByText(strings.desktop.uncertain);
 fireEvent.click(screen.getByRole("button", { name: "Refresh account" })); await screen.findByRole("button", { name: strings.desktop.signIn });
 const replacement = (id: string) => create(LoginResponseSchema, { account: { id, realm: "default", email: "owner@example.test" }, tokens: { accessToken: "new-access", refreshToken: "new-refresh", accessTokenExpiresAt: { seconds: BigInt(Math.floor(Date.now()/1000)+120) } } });
 vi.mocked(operatorClient.login).mockResolvedValueOnce(replacement("another-actor"));
 fireEvent.change(screen.getByLabelText(strings.desktop.password), { target: { value: "password" } });
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.signIn }));
 await screen.findByText(strings.desktop.failed);
 expect(screen.getByRole("button", { name: strings.desktop.signIn })).toBeInTheDocument();
 vi.mocked(operatorClient.login).mockResolvedValueOnce(replacement("owner-id"));
 fireEvent.change(screen.getByLabelText(strings.desktop.password), { target: { value: "password" } });
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.signIn }));
 await screen.findByRole("button", { name: strings.companion.checkStatus });
 vi.mocked(desktopClient.readCleanup).mockResolvedValueOnce(create(OwnerCleanupResponseSchema, { session: opened().session, released: true, observedAt: { seconds: 1n } }));
 fireEvent.click(screen.getByRole("button", { name: strings.companion.checkStatus }));
 await waitFor(() => expect(screen.getByRole("button", { name: strings.desktop.observe })).toBeEnabled());
 expect(desktopClient.stop).toHaveBeenCalledTimes(1);
 expect(desktopClient.readCleanup).toHaveBeenLastCalledWith({ session: opened().session }, { headers: { Authorization: "Bearer new-access" } });
});


function RecoveryState() {
 const state = useDesktopSession();
 return <p>{state.recoveryRequired ? "Recovery unresolved" : "Recovery clear"}</p>;
}
it("restores a credential-free Quit gate and reconciles exact identities after a full remount", async () => {
 const first = render(<DesktopSessionPanel surface={surface} onActive={vi.fn()} />); await login();
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.observe })); await screen.findByRole("img");
 const marker = localStorage.getItem(desktopRecoveryKey) ?? "";
 expect(marker).toContain("owner-id"); expect(marker).toContain("lease");
 for (const secret of ["account-token", "refresh-token", "password", "owner@example.test"]) expect(marker).not.toContain(secret);
 first.unmount();
 expect(desktopClient.stop).not.toHaveBeenCalled();
 const second = render(<><RecoveryState /><DesktopRecoveryPanel /></>);
 expect(screen.getByText("Recovery unresolved")).toBeInTheDocument();
 expect(screen.getByText(strings.desktop.recoverSignIn)).toBeInTheDocument();
 expect(desktopClient.listAdmissions).toHaveBeenCalledTimes(1);
 vi.mocked(operatorClient.login).mockResolvedValueOnce(create(LoginResponseSchema, { account: { id: "wrong-actor", realm: "default", email: "owner@example.test" }, tokens: { accessToken: "wrong-token", refreshToken: "wrong-refresh", accessTokenExpiresAt: { seconds: BigInt(Math.floor(Date.now()/1000)+120) } } }));
 fireEvent.change(screen.getByLabelText(strings.desktop.email), { target: { value: "owner@example.test" } });
 fireEvent.change(screen.getByLabelText(strings.desktop.password), { target: { value: "password" } });
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.signIn })); await screen.findByText(strings.desktop.failed);
 expect(desktopClient.listAdmissions).toHaveBeenCalledTimes(1);
 vi.mocked(desktopClient.readCleanup).mockResolvedValueOnce(create(OwnerCleanupResponseSchema, { session: opened().session, released: true, observedAt: { seconds: 1n } }));
 fireEvent.change(screen.getByLabelText(strings.desktop.password), { target: { value: "password" } });
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.signIn }));
 await screen.findByText("Recovery clear");
 expect(desktopClient.readCleanup).toHaveBeenCalledWith({ session: opened().session }, expect.anything());
 expect(desktopClient.stop).not.toHaveBeenCalled(); expect(desktopClient.open).toHaveBeenCalledTimes(1);
 expect(localStorage.getItem(desktopRecoveryKey)).toBeNull();
 second.unmount();
});

it("preserves unknown Open across reload even when history is empty", async () => {
 vi.mocked(desktopClient.open).mockRejectedValueOnce(new Error("lost reply"));
 const first = render(<DesktopSessionPanel surface={surface} onActive={vi.fn()} />); await login();
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.control })); await screen.findByText(strings.desktop.uncertain);
 first.unmount();
 render(<><RecoveryState /><DesktopRecoveryPanel /></>);
 await signInForRecovery();
 await waitFor(() => expect(desktopClient.reconcileOpen).toHaveBeenCalledTimes(1));
 expect(screen.getByText("Recovery unresolved")).toBeInTheDocument();
 expect(localStorage.getItem(desktopRecoveryKey)).toContain('"pendingOpen":');
 expect(desktopClient.open).toHaveBeenCalledTimes(1); expect(desktopClient.stop).not.toHaveBeenCalled();
});

it("does not send Open when the recovery marker cannot be written", async () => {
 render(<DesktopSessionPanel surface={surface} onActive={vi.fn()} />); await login();
 const write = vi.spyOn(Storage.prototype, "setItem").mockImplementationOnce(() => { throw new Error("Storage full"); });
 try {
  fireEvent.click(screen.getByRole("button", { name: strings.desktop.control }));
  await screen.findByText(strings.desktop.failed);
  expect(desktopClient.open).not.toHaveBeenCalled();
  expect(desktopClient.stop).not.toHaveBeenCalled();
 } finally { write.mockRestore(); }
});


it("reconciles the persisted Open identity after reload without sending Open again", async () => {
 vi.mocked(desktopClient.open).mockRejectedValueOnce(new Error("lost Open reply"));
 const first = render(<DesktopSessionPanel surface={surface} onActive={vi.fn()} />); await login();
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.control })); await screen.findByText(strings.desktop.uncertain);
 const request = vi.mocked(desktopClient.open).mock.calls[0]?.[0];
 expect(request?.requestId).toMatch(/^[0-9a-f-]{36}$/);
 expect(localStorage.getItem(desktopRecoveryKey)).toContain(request?.requestId ?? "missing");
 first.unmount();
 vi.mocked(desktopClient.reconcileOpen).mockImplementation(async value => create(OwnerOpenDispositionSchema, { requestId: value.requestId, state: "not_admitted" }));
 render(<DesktopSessionPanel surface={surface} onActive={vi.fn()} />); await login();
 expect(desktopClient.reconcileOpen).toHaveBeenCalledWith(request, { headers: { Authorization: "Bearer account-token" } });
 expect(desktopClient.open).toHaveBeenCalledTimes(1);
 expect(desktopClient.stop).not.toHaveBeenCalled();
 expect(localStorage.getItem(desktopRecoveryKey)).toBeNull();
});

it("does not accept a cancellation for a different request ID", async () => {
 vi.mocked(desktopClient.open).mockRejectedValueOnce(new Error("lost reply"));
 vi.mocked(desktopClient.reconcileOpen).mockResolvedValueOnce(create(OwnerOpenDispositionSchema, { requestId: "different", state: "not_admitted" }));
 render(<DesktopSessionPanel surface={surface} onActive={vi.fn()} />); await login();
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.control })); await screen.findByText(strings.desktop.uncertain);
 fireEvent.click(screen.getByRole("button", { name: strings.companion.checkStatus }));
 await waitFor(() => expect(desktopClient.reconcileOpen).toHaveBeenCalledTimes(1));
 expect(screen.getByRole("button", { name: strings.desktop.control })).toBeDisabled();
 expect(localStorage.getItem(desktopRecoveryKey)).toContain('"pendingOpen":');
});

it("requires exact cleanup after reconciliation returns a forwarding binding", async () => {
 vi.mocked(desktopClient.open).mockRejectedValueOnce(new Error("lost reply"));
 vi.mocked(desktopClient.reconcileOpen).mockImplementation(async request => create(OwnerOpenDispositionSchema, { requestId: request.requestId, state: "forwarding", session: opened(true).session, control: true, expiresAt: opened(true).expiresAt }));
 vi.mocked(desktopClient.readCleanup).mockResolvedValueOnce(create(OwnerCleanupResponseSchema, { session: opened(true).session, released: false, observedAt: { seconds: 1n } }));
 render(<DesktopSessionPanel surface={surface} onActive={vi.fn()} />); await login();
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.control })); await screen.findByText(strings.desktop.uncertain);
 fireEvent.click(screen.getByRole("button", { name: strings.companion.checkStatus }));
 await waitFor(() => expect(desktopClient.readCleanup).toHaveBeenCalledTimes(1));
 expect(screen.getByRole("button", { name: strings.desktop.control })).toBeDisabled();
 vi.mocked(desktopClient.readCleanup).mockResolvedValueOnce(create(OwnerCleanupResponseSchema, { session: opened(true).session, released: true, observedAt: { seconds: 1n } }));
 fireEvent.click(screen.getByRole("button", { name: strings.companion.checkStatus }));
 await waitFor(() => expect(screen.getByRole("button", { name: strings.desktop.control })).toBeEnabled());
 expect(desktopClient.reconcileOpen).toHaveBeenCalledTimes(1);
 expect(desktopClient.open).toHaveBeenCalledTimes(1); expect(desktopClient.stop).not.toHaveBeenCalled();
});


it("retains legacy unknown Open markers without inventing request identity", async () => {
 localStorage.setItem(desktopRecoveryKey, JSON.stringify({ version: 1, actor: { id: "owner-id", realm: "default" }, unknownOpen: true, sessions: [] }));
 render(<DesktopSessionPanel surface={surface} onActive={vi.fn()} />); await signInForRecovery();
 await waitFor(() => expect(desktopClient.listAdmissions).toHaveBeenCalledTimes(1));
 expect(screen.getByRole("button", { name: strings.desktop.control })).toBeDisabled();
 expect(desktopClient.reconcileOpen).not.toHaveBeenCalled();
 expect(localStorage.getItem(desktopRecoveryKey)).toContain('"unknownOpen":true');
});

it("honors shell Stop requested while Open is pending", async () => {
 let finish!: (value: ReturnType<typeof opened>) => void;
 vi.mocked(desktopClient.open).mockReturnValueOnce(new Promise(resolve => { finish = resolve; }));
 function ShellStopOpening() {
  const state = useDesktopSession();
  return <button onClick={() => void state.store.stop()}>Stop pending admission</button>;
 }
 render(<><ShellStopOpening /><DesktopSessionPanel surface={surface} onActive={vi.fn()} /></>); await login();
 fireEvent.click(screen.getByRole("button", { name: strings.desktop.control }));
 fireEvent.click(screen.getByRole("button", { name: "Stop pending admission" }));
 await act(async () => { finish(opened(true)); });
 await waitFor(() => expect(screen.getByRole("button", { name: strings.desktop.control })).toBeEnabled());
 expect(desktopClient.open).toHaveBeenCalledTimes(1); expect(desktopClient.stop).toHaveBeenCalledTimes(1);
 expect(desktopClient.observe).not.toHaveBeenCalled();
});
