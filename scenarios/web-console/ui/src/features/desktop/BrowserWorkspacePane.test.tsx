import { afterEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { renderWithProviders as render } from "../../test-utils";
import { BrowserWorkspacePane, projectBrowserDiscoveryState } from "./BrowserWorkspacePane";

const auth = vi.hoisted(() => ({ getAccessToken: vi.fn().mockResolvedValue(null) }));
vi.mock("../../lib/auth", () => ({ getAccessToken: auth.getAccessToken }));

class PaneSocket extends EventTarget { static CONNECTING=0; static OPEN=1; static urls:string[]=[]; static sent:string[]=[]; static instances:PaneSocket[]=[]; readyState=0; binaryType=""; constructor(url:string){super();PaneSocket.urls.push(url);PaneSocket.instances.push(this);queueMicrotask(()=>{this.readyState=1;this.dispatchEvent(new Event("open"))})} close(){this.readyState=3;this.dispatchEvent(new Event("close"))} send(data:string){PaneSocket.sent.push(data)} message(data:string){const event=new Event("message");Object.defineProperty(event,"data",{value:data});this.dispatchEvent(event)} }
afterEach(()=>{vi.unstubAllGlobals();PaneSocket.instances=[];PaneSocket.urls=[];PaneSocket.sent=[];});

describe("BrowserWorkspacePane",()=>{
  it("uses the shared discovery states at desktop and phone widths", async()=>{
    expect(projectBrowserDiscoveryState({kind:"checking"})).toBe("loading");
    expect(projectBrowserDiscoveryState({kind:"no-session"})).toBe("closed");
    expect(projectBrowserDiscoveryState({kind:"multiple",count:2})).toBe("permission-blocked");
    expect(projectBrowserDiscoveryState({kind:"unauthorized"})).toBe("permission-blocked");
    expect(projectBrowserDiscoveryState({kind:"owner-offline"})).toBe("offline");
    for(const width of [375,1280]) {
      vi.stubGlobal("fetch",vi.fn(async()=>new Response(JSON.stringify({sessions:[]}),{status:200,headers:{"Content-Type":"application/json"}})));
      const {unmount}=render(<div style={{width}}><BrowserWorkspacePane webConsoleSessionID="wc-session" onClose={()=>undefined}/></div>);
      await waitFor(()=>expect(screen.getByTestId("workspace-surface-pane")).toHaveAttribute("data-state","closed"));
      expect(screen.getByRole("status")).toHaveTextContent(/Surface is closed/i);
      expect(screen.getByRole("toolbar",{name:"Browser pane controls"})).toHaveAttribute("aria-orientation","horizontal");
      unmount();
    }
  });
  it("keeps the live pane closed when server-side association is refused",async()=>{
    vi.stubGlobal("fetch",vi.fn(async()=>new Response("refused",{status:403})));
    render(<BrowserWorkspacePane webConsoleSessionID="wc-session" onClose={()=>undefined}/>);
    await waitFor(()=>expect(screen.getByTestId("browser-pane-empty")).toHaveTextContent(/could not verify/i));
    expect(screen.queryByLabelText("Live browser surface")).not.toBeInTheDocument();
  });
  it("mounts the observer only for one owner-attested active session",async()=>{
    vi.stubGlobal("WebSocket",PaneSocket);
    vi.stubGlobal("fetch",vi.fn(async()=>new Response(JSON.stringify({sessions:[{pane_handle:"a".repeat(48),mode:"managed",can_control:true}]}),{status:200,headers:{"Content-Type":"application/json"}})));
    render(<BrowserWorkspacePane webConsoleSessionID="wc-session" onClose={()=>undefined}/>);
    await waitFor(()=>expect(screen.getByLabelText("Live browser surface")).toBeInTheDocument());
    expect(screen.getByText(/Observer/)).toBeInTheDocument();
    expect(screen.queryByText(/run-123|thread-456/)).not.toBeInTheDocument();
  });
  it("keeps browser selectors and secret payloads out of URLs, history, storage and logs",async()=>{
    PaneSocket.urls=[];PaneSocket.sent=[];
    const handle="c".repeat(48);
    const secrets=["https://node.internal/private", "grant-secret-value", "v=0\r\no=secret-sdp", "private clipboard payload", "typed password payload"];
    const storageWrites=vi.spyOn(Storage.prototype,"setItem");
    const consoleSinks=[vi.spyOn(console,"log"),vi.spyOn(console,"warn"),vi.spyOn(console,"error")];
    vi.stubGlobal("WebSocket",PaneSocket);
    vi.stubGlobal("fetch",vi.fn(async()=>new Response(JSON.stringify({sessions:[{pane_handle:handle,mode:"managed",can_control:true,node_url:secrets[0],grant:secrets[1],sdp:secrets[2],clipboard:secrets[3],input:secrets[4]}]}),{status:200,headers:{"Content-Type":"application/json"}})));
    const initialURL=window.location.href;
    render(<BrowserWorkspacePane webConsoleSessionID="wc-session" onClose={()=>undefined}/>);
    await waitFor(()=>expect(screen.getByLabelText("Live browser surface")).toBeInTheDocument());
    await waitFor(()=>expect(PaneSocket.sent.some((data)=>data.includes('"type":"attach"'))).toBe(true));
    expect(PaneSocket.urls).toEqual([expect.stringMatching(/\/api\/v1\/browser\/stream$/)]);
    expect(PaneSocket.urls.join(" ")).not.toContain(handle);
    expect(PaneSocket.sent[0]).toContain(handle); // ephemeral owner selector travels only in the socket frame
    expect(window.location.href).toBe(initialURL);
    const persisted=[...storageWrites.mock.calls.flatMap((call)=>call.map(String)),localStorage.getItem("browser-session"),sessionStorage.getItem("browser-session")].join(" ");
    const logged=consoleSinks.flatMap((sink)=>sink.mock.calls.flatMap((call)=>call.map(String))).join(" ");
    for(const secret of secrets) { expect(persisted).not.toContain(secret); expect(logged).not.toContain(secret); }
    for(const secret of secrets) { expect(PaneSocket.urls.join(" ")).not.toContain(secret); }
  });
  it("keeps attach, recovery, terminal denial, and close values out of durable sinks",async()=>{
    vi.stubGlobal("WebSocket",PaneSocket);
    const first="d".repeat(48), refreshed="e".repeat(48);
    const secrets=[first,refreshed,"wc-session","https://private-node.invalid/path","synthetic-grant","v=0\r\no=private-sdp","clipboard-secret","typed-secret"];
    const fetchMock=vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({sessions:[{pane_handle:first,mode:"managed",can_control:true,node_url:secrets[3],grant:secrets[4],sdp:secrets[5],clipboard:secrets[6],input:secrets[7]}]}),{status:200,headers:{"Content-Type":"application/json"}}))
      .mockResolvedValueOnce(new Response(JSON.stringify({sessions:[{pane_handle:refreshed,mode:"managed",can_control:true}]}),{status:200,headers:{"Content-Type":"application/json"}}))
      .mockResolvedValueOnce(new Response(JSON.stringify({sessions:[],run_finished:true}),{status:200,headers:{"Content-Type":"application/json"}}));
    vi.stubGlobal("fetch",fetchMock);
    const storageWrites=vi.spyOn(Storage.prototype,"setItem");
    const historyPush=vi.spyOn(window.history,"pushState");
    const historyReplace=vi.spyOn(window.history,"replaceState");
    const idbOpen=vi.fn();
    vi.stubGlobal("indexedDB",{open:idbOpen});
    const analytics=vi.fn();
    vi.stubGlobal("gtag",analytics);
    const dataLayerPush=vi.fn();
    const dataLayer=Object.assign([] as unknown[],{push:dataLayerPush});
    vi.stubGlobal("dataLayer",dataLayer);
    const beacon=vi.fn();
    const fakeNavigator=Object.create(navigator) as Navigator;
    Object.defineProperty(fakeNavigator,"sendBeacon",{value:beacon});
    vi.stubGlobal("navigator",fakeNavigator);
    const consoleSinks=[vi.spyOn(console,"log"),vi.spyOn(console,"warn"),vi.spyOn(console,"error")];
    const initialURL=window.location.href;
    const onClose=vi.fn();
    const {unmount}=render(<BrowserWorkspacePane webConsoleSessionID="wc-session" onClose={onClose}/>);
    await waitFor(()=>expect(screen.getByLabelText("Live browser surface")).toBeInTheDocument());
    const telemetryAtAttach=screen.getByLabelText("Browser viewer telemetry").textContent ?? "";
    await waitFor(()=>expect(PaneSocket.instances).toHaveLength(1));
    act(()=>PaneSocket.instances[0]!.close());
    await waitFor(()=>expect(PaneSocket.instances).toHaveLength(2));
    await waitFor(()=>expect(PaneSocket.sent.filter((data)=>data.includes(refreshed))).toHaveLength(2));
    act(()=>PaneSocket.instances[1]!.message(JSON.stringify({type:"browser_stream_error"})));
    await waitFor(()=>expect(screen.getByTestId("browser-pane-empty")).toHaveTextContent(/agent run has finished/i));
    expect(screen.queryByRole("button",{name:"Reconnect"})).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button",{name:"Close"}));
    expect(onClose).toHaveBeenCalledOnce();
    unmount();

    const stored=[...storageWrites.mock.calls.flatMap((call)=>call.map(String))];
    for(const storage of [localStorage,sessionStorage]) for(const key of Object.keys(storage)) stored.push(key,storage.getItem(key) ?? "");
    const logged=consoleSinks.flatMap((sink)=>sink.mock.calls.flatMap((call)=>call.map(String))).join(" ");
    const telemetry=`${telemetryAtAttach} ${screen.queryByLabelText("Browser viewer telemetry")?.textContent ?? ""}`;
    const durable=[stored.join(" "),window.location.href,logged,telemetry,PaneSocket.urls.join(" ")].join(" ");
    for(const secret of secrets) expect(durable).not.toContain(secret);
    expect(window.location.href).toBe(initialURL);
    expect(historyPush).not.toHaveBeenCalled();
    expect(historyReplace).not.toHaveBeenCalled();
    expect(idbOpen).not.toHaveBeenCalled();
    expect(analytics).not.toHaveBeenCalled();
    expect(dataLayerPush).not.toHaveBeenCalled();
    expect(beacon).not.toHaveBeenCalled();
    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(fetchMock.mock.calls.map(([url])=>String(url)).every((url)=>url.includes("/browser/pane/sessions"))).toBe(true);
  });
  it("does not choose between ambiguous active sessions",async()=>{
    vi.stubGlobal("fetch",vi.fn(async()=>new Response(JSON.stringify({sessions:[{pane_handle:"a".repeat(48),mode:"managed"},{pane_handle:"b".repeat(48),mode:"managed"}]}),{status:200,headers:{"Content-Type":"application/json"}})));
    render(<BrowserWorkspacePane webConsoleSessionID="wc-session" onClose={()=>undefined}/>);
    await waitFor(()=>expect(screen.getByTestId("browser-pane-empty")).toHaveTextContent(/more than one/i));
    expect(screen.queryByLabelText("Live browser surface")).not.toBeInTheDocument();
  });

  it("explains an empty browser and offers retry plus the enable checklist",async()=>{
    vi.stubGlobal("fetch",vi.fn(async()=>new Response(JSON.stringify({sessions:[]}),{status:200,headers:{"Content-Type":"application/json"}})));
    render(<BrowserWorkspacePane webConsoleSessionID="wc-session" onClose={()=>undefined}/>);
    await waitFor(()=>expect(screen.getByTestId("browser-pane-empty")).toHaveTextContent(/no browser is attached/i));
    expect(screen.getByTestId("browser-pane-retry")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("browser-pane-help"));
    expect(await screen.findByTestId("browser-help-dialog")).toBeInTheDocument();
    expect(screen.getByText(/agent run that owns a browser/i)).toBeInTheDocument();
  });

  it("retries the owner attestation from the empty state",async()=>{
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({sessions:[]}),{status:200,headers:{"Content-Type":"application/json"}}))
      .mockResolvedValueOnce(new Response(JSON.stringify({sessions:[{pane_handle:"a".repeat(48),mode:"managed",can_control:true}]}),{status:200,headers:{"Content-Type":"application/json"}}));
    vi.stubGlobal("fetch",fetchMock);
    vi.stubGlobal("WebSocket",PaneSocket);
    render(<BrowserWorkspacePane webConsoleSessionID="wc-session" onClose={()=>undefined}/>);
    await waitFor(()=>expect(screen.getByTestId("browser-pane-retry")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("browser-pane-retry"));
    await waitFor(()=>expect(screen.getByLabelText("Live browser surface")).toBeInTheDocument());
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("replaces cached reconnect with a terminal run state after owner confirmation",async()=>{
    vi.stubGlobal("WebSocket",PaneSocket);
    const fetchMock=vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({sessions:[{pane_handle:"a".repeat(48),mode:"managed",can_control:true}]}),{status:200,headers:{"Content-Type":"application/json"}}))
      .mockResolvedValueOnce(new Response(JSON.stringify({sessions:[],run_finished:true}),{status:200,headers:{"Content-Type":"application/json"}}));
    vi.stubGlobal("fetch",fetchMock);
    render(<BrowserWorkspacePane webConsoleSessionID="wc-session" onClose={()=>undefined}/>);
    await waitFor(()=>expect(screen.getByLabelText("Live browser surface")).toBeInTheDocument());
    await waitFor(()=>expect(PaneSocket.instances).toHaveLength(1));
    act(()=>PaneSocket.instances[0]!.close());
    await waitFor(()=>expect(screen.getByTestId("browser-pane-empty")).toHaveTextContent(/agent run has finished/i));
    expect(screen.queryByRole("button",{name:"Reconnect"})).not.toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("keeps recoverable network loss on the active owner path with a fresh opaque handle",async()=>{
    vi.stubGlobal("WebSocket",PaneSocket);
    const first="b".repeat(48), refreshed="c".repeat(48);
    const fetchMock=vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({sessions:[{pane_handle:first,mode:"managed",can_control:true}]}),{status:200,headers:{"Content-Type":"application/json"}}))
      .mockResolvedValueOnce(new Response(JSON.stringify({sessions:[{pane_handle:refreshed,mode:"managed",can_control:true}]}),{status:200,headers:{"Content-Type":"application/json"}}));
    vi.stubGlobal("fetch",fetchMock);
    render(<BrowserWorkspacePane webConsoleSessionID="wc-session" onClose={()=>undefined}/>);
    await waitFor(()=>expect(screen.getByLabelText("Live browser surface")).toBeInTheDocument());
    await waitFor(()=>expect(PaneSocket.instances).toHaveLength(1));
    act(()=>PaneSocket.instances[0]!.close());
    await waitFor(()=>expect(PaneSocket.instances).toHaveLength(2));
    await waitFor(()=>expect(PaneSocket.sent.some((data)=>data.includes(refreshed))).toBe(true));
    expect(PaneSocket.sent.some((data)=>data.includes(first))).toBe(true);
    expect(PaneSocket.sent.filter((data)=>data.includes(refreshed))).toHaveLength(2);
    expect(screen.queryByTestId("browser-pane-empty")).not.toBeInTheDocument();
  });
});
