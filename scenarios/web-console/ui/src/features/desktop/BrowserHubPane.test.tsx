import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { renderWithProviders as render } from "../../test-utils";
import { afterEach, describe, expect, it, vi } from "vitest";
import { BrowserHubPane, decodeBrowserHubBinaryFrame, decodeBrowserHubSessionFrame, MAX_BROWSER_FRAME_PIXELS, validateDecodedBrowserFrame, type HubMessage } from "./BrowserHubPane";

class FakeWebSocket extends EventTarget {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;
  readyState = FakeWebSocket.CONNECTING;
  sent: string[] = [];
  url: string;
  constructor(url: string) { super(); this.url = url; FakeWebSocket.created.push(this); }
  static created: FakeWebSocket[] = [];
  send(data: string) { this.sent.push(data); }
  close() { this.readyState = FakeWebSocket.CLOSED; this.dispatchEvent(new Event("close")); }
  open() { this.readyState = FakeWebSocket.OPEN; this.dispatchEvent(new Event("open")); }
  message(data: string) { const event = new Event("message"); Object.defineProperty(event, "data", { value: data }); this.dispatchEvent(event); }
  binaryMessage(data: ArrayBuffer) { const event = new Event("message"); Object.defineProperty(event, "data", { value: data }); this.dispatchEvent(event); }
}

function frameFor(sessionID: string): ArrayBuffer {
  const header = new TextEncoder().encode(JSON.stringify({ version: 1, session_id: sessionID }));
  const frame = new Uint8Array(4 + header.length + 2);
  new DataView(frame.buffer).setUint32(0, header.length, false);
  frame.set(header, 4);
  frame.set([0xff, 0xd8], 4 + header.length);
  return frame.buffer;
}

afterEach(() => {
  FakeWebSocket.created = [];
  vi.unstubAllGlobals();
});

describe("BrowserHubPane", () => {
  it("preserves the owner revoked state instead of showing an endless connect state", () => {
    const session = {
      sessionId: "opaque-pane-handle-revoked",
      mode: "managed" as const,
      state: "revoked" as const,
      controller: false,
      association: { source: "agent-run" as const, verified: true },
      capabilities: { observe: true, control: false, frames: true },
    };
    render(<BrowserHubPane session={session} onClose={() => undefined} />);
    expect(screen.getByTestId("workspace-pane-state")).toHaveAttribute("data-pane-state", "revoked");
    expect(screen.getByText(/browser owner revoked this stream/i)).toBeInTheDocument();
    expect(FakeWebSocket.created).toHaveLength(0);
  });

  it("accepts only sanitized binary frames for the selected BAS session", () => {
    const header = JSON.stringify({ version: 1, session_id: "bas-session-17", captured_at: "2026-11-19T00:00:00Z" });
    const encodedHeader = new TextEncoder().encode(header);
    const bytes = new Uint8Array(4 + encodedHeader.length + 4);
    new DataView(bytes.buffer).setUint32(0, encodedHeader.length, false);
    bytes.set(encodedHeader, 4);
    bytes.set([0xff, 0xd8, 0xff, 0xd9], 4 + encodedHeader.length);
    expect(decodeBrowserHubBinaryFrame(bytes.buffer, "bas-session-17")?.jpeg).toEqual(new Uint8Array([0xff, 0xd8, 0xff, 0xd9]));
    expect(decodeBrowserHubBinaryFrame(bytes.buffer, "other-session")).toBeUndefined();
    const withPrivateURL = new TextEncoder().encode(JSON.stringify({ version: 1, session_id: "bas-session-17", page_url: "https://example.test/?token=private" }));
    const unsafe = new Uint8Array(4 + withPrivateURL.length + 2);
    new DataView(unsafe.buffer).setUint32(0, withPrivateURL.length, false);
    unsafe.set(withPrivateURL, 4);
    unsafe.set([0xff, 0xd8], 4 + withPrivateURL.length);
    expect(decodeBrowserHubBinaryFrame(unsafe.buffer, "bas-session-17")).toBeUndefined();
  });

  it("preserves identical JPEG fixture bytes across binary and session-frame encodings", () => {
    const sessionID = "bas-session-17";
    const jpeg = new Uint8Array([0xff, 0xd8, 0x01, 0x02, 0xff, 0xd9]);
    const header = new TextEncoder().encode(JSON.stringify({ version: 1, session_id: sessionID }));
    const binary = new Uint8Array(4 + header.length + jpeg.length);
    new DataView(binary.buffer).setUint32(0, header.length, false);
    binary.set(header, 4);
    binary.set(jpeg, 4 + header.length);
    const textFrame = {
      type: "session_frame",
      session_id: sessionID,
      data: btoa(String.fromCharCode(...jpeg)),
      media_type: "image/jpeg",
      width: 2,
      height: 1,
      captured_at: "2026-11-21T00:00:00Z",
    };

    const binaryDecoded = decodeBrowserHubBinaryFrame(binary.buffer, sessionID);
    const textDecoded = decodeBrowserHubSessionFrame(textFrame, sessionID);
    expect(binaryDecoded?.jpeg).toEqual(jpeg);
    expect(binaryDecoded?.jpeg.buffer).toBe(binary.buffer);
    expect(binaryDecoded?.jpeg.byteOffset).toBe(4 + header.length);
    expect(textDecoded?.image).toEqual(jpeg);
    expect(textDecoded?.image).toEqual(binaryDecoded?.jpeg);
    expect(decodeBrowserHubSessionFrame({ ...textFrame, session_id: "other-session" }, sessionID)).toBeUndefined();
    expect(decodeBrowserHubSessionFrame({ ...textFrame, data: "%%%" }, sessionID)).toBeUndefined();

    // Fixture-only representation size: distinct envelopes and stream kinds, not a transport benchmark.
    const binaryBytes = binary.byteLength;
    const textBytes = new TextEncoder().encode(JSON.stringify(textFrame)).byteLength;
    expect(textBytes).toBeGreaterThan(binaryBytes);
  });

  it("bounds decoded canvas dimensions and checks session-frame metadata", () => {
    expect(validateDecodedBrowserFrame(1920, 1080, 1920, 1080)).toBe(true);
    expect(validateDecodedBrowserFrame(1920, 1080, 1080, 1920)).toBe(false);
    expect(validateDecodedBrowserFrame(MAX_BROWSER_FRAME_PIXELS + 1, 1)).toBe(false);
    expect(validateDecodedBrowserFrame(0, 1080)).toBe(false);
  });

  it("rejects oversized or inconsistent JSON metadata before base64 decode", () => {
    const decode = vi.spyOn(globalThis, "atob");
    const frame: HubMessage = {
      type: "session_frame",
      session_id: "bas-session-17",
      data: btoa("jpeg-bytes"),
      media_type: "image/jpeg",
      width: 5000,
      height: 4000,
      captured_at: "2026-11-21T00:00:00Z",
      timestamp: "2026-11-21T00:00:00Z",
    };

    expect(decodeBrowserHubSessionFrame(frame, "bas-session-17")).toBeUndefined();
    expect(decodeBrowserHubSessionFrame({ ...frame, width: 2, height: 1, page_url: "https://private.invalid/" } as HubMessage, "bas-session-17")).toBeUndefined();
    expect(decodeBrowserHubSessionFrame({ ...frame, width: 2, height: 1, captured_at: "x".repeat(65) }, "bas-session-17")).toBeUndefined();
    expect(decode).not.toHaveBeenCalled();
  });

  it("closes and refuses an oversized decoded frame before canvas allocation", async () => {
    vi.stubGlobal("WebSocket", FakeWebSocket);
    const close = vi.fn();
    vi.stubGlobal("createImageBitmap", vi.fn(async () => ({ width: 5000, height: 4000, close })));
    const drawImage = vi.fn();
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue({ drawImage } as unknown as CanvasRenderingContext2D);
    const session = {
      sessionId: "oversized-frame-session",
      mode: "managed" as const,
      state: "viewer" as const,
      controller: false,
      association: { source: "agent-run" as const, verified: true },
      capabilities: { observe: true, control: false, frames: true },
    };
    render(<BrowserHubPane session={session} />);
    await waitFor(() => expect(FakeWebSocket.created).toHaveLength(1));
    const socket = FakeWebSocket.created[0]!;
    act(() => socket.open());
    act(() => socket.binaryMessage(frameFor(session.sessionId)));

    await screen.findByText("The browser owner sent a frame with invalid dimensions.");
    expect(close).toHaveBeenCalledTimes(1);
    expect(drawImage).not.toHaveBeenCalled();
  });

  it("takes control through a fresh owner receipt and releases it by rotating to observe", async () => {
    vi.stubGlobal("WebSocket", FakeWebSocket);
    vi.stubGlobal("createImageBitmap", vi.fn(async () => ({ width: 2, height: 2, close() {} })));
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue({ drawImage() {} } as unknown as CanvasRenderingContext2D);
    const returnRun = vi.fn(async () => {});
    const session = {
      sessionId: "bas-session-17",
      mode: "managed" as const,
      state: "controller" as const,
      controller: true,
      controlReceipt: "opaque-controller-receipt",
      sessionEpoch: 2n,
      association: { source: "agent-run" as const, runId: "run-31", threadId: "thread-9", verified: true },
      capabilities: { observe: true, control: true, frames: true },
      viewport: { width: 100, height: 50 },
      frameSize: { width: 200, height: 100 },
    };
    render(<BrowserHubPane session={{ ...session, state: "viewer", controller: false, controlReceipt: undefined, sessionEpoch: undefined }} onControlReturned={returnRun} />);
    await waitFor(() => expect(FakeWebSocket.created).toHaveLength(1));
    const observer = FakeWebSocket.created[0]!;
    expect(observer.url).toContain("/api/v1/browser/stream");
    expect(observer.url).not.toContain("bas-session-17");
    expect(observer.url).not.toContain("receipt");
    act(()=>{observer.open();});
    expect(observer.sent.map((value) => JSON.parse(value))).toEqual([
      { type: "attach", role: "observe", pane_handle: "bas-session-17" },
      { type: "subscribe_recording", session_id: "bas-session-17" },
    ]);
    fireEvent.click(screen.getByRole("button", { name: "Take control" }));
    await waitFor(() => expect(FakeWebSocket.created).toHaveLength(2));
    const socket = FakeWebSocket.created[1]!;
    act(()=>{socket.open();});
    expect(socket.sent.map((value) => JSON.parse(value))).toEqual([
      { type: "attach", role: "control", pane_handle: "bas-session-17" },
      { type: "subscribe_recording", session_id: "bas-session-17" },
    ]);
    expect(socket.sent.join("\n")).not.toContain("opaque-controller-receipt");
    act(() => socket.binaryMessage(frameFor("bas-session-17")));
    await screen.findByRole("button", { name: "Type text" });
    fireEvent.keyDown(screen.getByLabelText("Live browser surface"), { key: "a" });
    await waitFor(() => expect(socket.sent).toHaveLength(3));
    const sentInput = JSON.parse(socket.sent[2]!) as { input: { input_id?: string } };
    expect(JSON.parse(socket.sent[2]!)).toMatchObject({ type: "recording_input", session_id: "bas-session-17", input: { type: "keyboard", key: "a", modifiers: [] } });
    expect(sentInput.input.input_id).toMatch(/^wc-input-\d+$/);
    act(() => socket.message(JSON.stringify({ type: "recording_input_applied", session_id: "bas-session-17", input_id: sentInput.input.input_id })));
    act(() => socket.binaryMessage(frameFor("bas-session-17")));
    await waitFor(() => expect(screen.getByLabelText("Browser viewer telemetry")).toHaveTextContent(/input to next painted frame .* \(1 samples\)/));
    act(()=>{socket.close();});
    await waitFor(() => expect(FakeWebSocket.created).toHaveLength(3));
    const recoveryObserver = FakeWebSocket.created[2]!;
    act(()=>{recoveryObserver.open();});
    expect(recoveryObserver.sent.map((value) => JSON.parse(value))).toEqual([
      { type: "attach", role: "observe", pane_handle: "bas-session-17" },
      { type: "subscribe_recording", session_id: "bas-session-17" },
    ]);
    const header = new TextEncoder().encode(JSON.stringify({ version: 1, session_id: "bas-session-17" }));
    const frame = new Uint8Array(4 + header.length + 2);
    new DataView(frame.buffer).setUint32(0, header.length, false);
    frame.set(header, 4);
    frame.set([0xff, 0xd8], 4 + header.length);
    act(() => recoveryObserver.binaryMessage(frame.buffer));
    await waitFor(() => expect(returnRun).toHaveBeenCalledTimes(1));
    await act(async () => { await returnRun.mock.results[0]?.value; });
    fireEvent.click(screen.getByRole("button", { name: "Take control" }));
    await waitFor(() => expect(FakeWebSocket.created).toHaveLength(4));
    const reconnected = FakeWebSocket.created[3]!;
    act(() => reconnected.open());
    act(() => reconnected.binaryMessage(frame.buffer));
    await screen.findByRole("button", { name: "Release control" });
    fireEvent.click(screen.getByRole("button", { name: "Release control" }));
    await waitFor(() => expect(FakeWebSocket.created).toHaveLength(5));
    const returned = FakeWebSocket.created[4]!;
    act(()=>{returned.open();});
    expect(returned.sent.map((value) => JSON.parse(value))).toEqual([
      { type: "attach", role: "observe", pane_handle: "bas-session-17" },
      { type: "subscribe_recording", session_id: "bas-session-17" },
    ]);
    expect(returnRun).toHaveBeenCalledTimes(1);
    act(() => returned.binaryMessage(frame.buffer));
    await waitFor(() => expect(returnRun).toHaveBeenCalledTimes(2));
    expect(reconnected.readyState).toBe(FakeWebSocket.CLOSED);
  });

  it("reconnects through the owner with a fresh server-side receipt exchange", async()=>{
    vi.stubGlobal("WebSocket",FakeWebSocket);
    const session={sessionId:"opaque-pane-handle-1",mode:"managed" as const,state:"viewer" as const,controller:false,
      association:{source:"agent-run" as const,verified:true},capabilities:{observe:true,control:false,frames:true}};
    const onRevalidateOwner=vi.fn(async()=>true);
    render(<BrowserHubPane session={session} onRevalidateOwner={onRevalidateOwner}/>);
    await waitFor(()=>expect(FakeWebSocket.created).toHaveLength(1));
    const first=FakeWebSocket.created[0]!; act(()=>{first.open(); first.close();});
    await waitFor(()=>expect(onRevalidateOwner).toHaveBeenCalledOnce());
    const reconnect=await screen.findByRole("button",{name:"Reconnect"}); fireEvent.click(reconnect);
    await waitFor(()=>expect(FakeWebSocket.created).toHaveLength(2));
    const second=FakeWebSocket.created[1]!; act(()=>{second.open();});
    expect(second.url).toContain("/api/v1/browser/stream");
    expect(second.url).not.toContain("opaque-pane-handle-1");
    expect(second.sent.map((value)=>JSON.parse(value))).toEqual([{type:"attach",role:"observe",pane_handle:"opaque-pane-handle-1"},{type:"subscribe_recording",session_id:"opaque-pane-handle-1"}]);
  });

  it("does not expose reconnect when the owner cannot revalidate the run", async()=>{
    vi.stubGlobal("WebSocket", FakeWebSocket);
    const onRevalidateOwner=vi.fn(async()=>false);
    const session={sessionId:"opaque-pane-handle-2",mode:"managed" as const,state:"viewer" as const,controller:false,
      association:{source:"agent-run" as const,verified:true},capabilities:{observe:true,control:false,frames:true}};
    render(<BrowserHubPane session={session} onRevalidateOwner={onRevalidateOwner}/>);
    await waitFor(()=>expect(FakeWebSocket.created).toHaveLength(1));
    act(()=>{FakeWebSocket.created[0]!.open(); FakeWebSocket.created[0]!.close();});
    await waitFor(()=>expect(onRevalidateOwner).toHaveBeenCalledOnce());
    expect(screen.queryByRole("button",{name:"Reconnect"})).not.toBeInTheDocument();
  });

  it("does not retry after BAS refuses a reconnect even if the run lookup succeeds", async()=>{
    vi.stubGlobal("WebSocket", FakeWebSocket);
    const onRevalidateOwner=vi.fn(async()=>true);
    const session={sessionId:"opaque-pane-handle-denied",mode:"managed" as const,state:"viewer" as const,controller:false,
      association:{source:"agent-run" as const,verified:true},capabilities:{observe:true,control:false,frames:true}};
    render(<BrowserHubPane session={session} onRevalidateOwner={onRevalidateOwner}/>);
    await waitFor(()=>expect(FakeWebSocket.created).toHaveLength(1));
    const socket=FakeWebSocket.created[0]!;
    act(()=>{socket.open(); socket.message(JSON.stringify({type:"browser_stream_error"}));});
    await waitFor(()=>expect(onRevalidateOwner).toHaveBeenCalledWith("owner-refused"));
    expect(screen.queryByRole("button",{name:"Reconnect"})).not.toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent(/refused or lost/i);
  });

  it("shows owner revocation as terminal and does not offer stale-handle reconnect", async () => {
    vi.stubGlobal("WebSocket", FakeWebSocket);
    const session = { sessionId: "opaque-pane-handle-ended", mode: "managed" as const, state: "viewer" as const, controller: false,
      association: { source: "agent-run" as const, verified: true }, capabilities: { observe: true, control: false, frames: true } };
    render(<BrowserHubPane session={session} />);
    await waitFor(() => expect(FakeWebSocket.created).toHaveLength(1));
    const socket = FakeWebSocket.created[0]!;
    act(() => socket.open());
    act(() => socket.message(JSON.stringify({ type: "session_access_revoked", session_id: session.sessionId })));
    act(() => socket.close());
    expect(screen.getByTestId("workspace-pane-state")).toHaveAttribute("data-pane-state", "revoked");
    expect(screen.getByText(/close and reopen the pane/i)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Reconnect" })).not.toBeInTheDocument();
  });

  it("waits for observer confirmation before closing a controller pane", async () => {
    vi.stubGlobal("WebSocket", FakeWebSocket);
    vi.stubGlobal("createImageBitmap", vi.fn(async () => ({ width: 2, height: 2, close() {} })));
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue({ drawImage() {} } as unknown as CanvasRenderingContext2D);
    const onClose = vi.fn();
    const onControlReturned = vi.fn(async () => {});
    render(<BrowserHubPane session={{ sessionId: "opaque-pane-handle-7", mode: "managed", state: "viewer", controller: false,
      association: { source: "agent-run", verified: true }, capabilities: { observe: true, control: true, frames: true } }} onClose={onClose} onControlReturned={onControlReturned} />);
    await waitFor(() => expect(FakeWebSocket.created).toHaveLength(1));
    act(() => FakeWebSocket.created[0]!.open());
    fireEvent.click(screen.getByRole("button", { name: "Take control" }));
    await waitFor(() => expect(FakeWebSocket.created).toHaveLength(2));
    act(() => FakeWebSocket.created[1]!.open());
    fireEvent.click(screen.getByRole("button", { name: "Close session" }));
    await waitFor(() => expect(FakeWebSocket.created).toHaveLength(3));
    const observer = FakeWebSocket.created[2]!;
    act(() => observer.open());
    expect(onClose).not.toHaveBeenCalled();
    const header = new TextEncoder().encode(JSON.stringify({ version: 1, session_id: "opaque-pane-handle-7" }));
    const frame = new Uint8Array(4 + header.length + 2);
    new DataView(frame.buffer).setUint32(0, header.length, false);
    frame.set(header, 4);
    frame.set([0xff, 0xd8], 4 + header.length);
    act(() => observer.binaryMessage(frame.buffer));
    await waitFor(() => expect(onControlReturned).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
  });
});
