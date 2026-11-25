import { afterEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { renderWithProviders as render } from "../../test-utils";
import { BrowserWorkspacePane } from "./BrowserWorkspacePane";

class PaneSocket extends EventTarget { static CONNECTING=0; static OPEN=1; readyState=0; binaryType=""; constructor(_url:string){super()} close(){this.readyState=3} send(_data:string){} }
afterEach(()=>vi.unstubAllGlobals());

describe("BrowserWorkspacePane",()=>{
  it("keeps the live pane closed when server-side association is refused",async()=>{
    vi.stubGlobal("fetch",vi.fn(async()=>new Response("refused",{status:403})));
    render(<BrowserWorkspacePane webConsoleSessionID="wc-session" onClose={()=>undefined}/>);
    await waitFor(()=>expect(screen.getByRole("status")).toHaveTextContent(/could not verify/i));
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
  it("does not choose between ambiguous active sessions",async()=>{
    vi.stubGlobal("fetch",vi.fn(async()=>new Response(JSON.stringify({sessions:[{pane_handle:"a".repeat(48),mode:"managed"},{pane_handle:"b".repeat(48),mode:"managed"}]}),{status:200,headers:{"Content-Type":"application/json"}})));
    render(<BrowserWorkspacePane webConsoleSessionID="wc-session" onClose={()=>undefined}/>);
    await waitFor(()=>expect(screen.getByRole("status")).toHaveTextContent(/more than one/i));
    expect(screen.queryByLabelText("Live browser surface")).not.toBeInTheDocument();
  });

  it("explains an empty browser and offers retry plus the enable checklist",async()=>{
    vi.stubGlobal("fetch",vi.fn(async()=>new Response(JSON.stringify({sessions:[]}),{status:200,headers:{"Content-Type":"application/json"}})));
    render(<BrowserWorkspacePane webConsoleSessionID="wc-session" onClose={()=>undefined}/>);
    await waitFor(()=>expect(screen.getByRole("status")).toHaveTextContent(/no browser is attached/i));
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
});
