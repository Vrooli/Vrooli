import { afterEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import type { DesktopReadiness, DesktopSession } from "@vrooli/proto-types/device-control/v1/desktop/desktop_pb";
import { renderWithProviders as render } from "../../test-utils";
import { DesktopMachineTab } from "./DesktopMachineTab";

type Body = Record<string, unknown>;
type FakeDataChannel = {
  label: string;
  readyState: string;
  onmessage?: (event: { data: string }) => void;
  send: (payload: string) => void;
};

/**
 * SOFTWARE SIMULATION ONLY. This transport-boundary fixture speaks the
 * existing Device Control/Bridge Connect-JSON shapes. It does not emulate a
 * physical desktop, Bridge node, WebRTC network, native input, or hardware.
 */
function installDesktopSoftwareStandIn() {
  const calls: Array<{ path: string; body: Body }> = [];
  const peers: FakePeer[] = [];
  const dataEvents: Array<{ kind: string; pointer?: { x: number; y: number }; clipboardBytes?: number }> = [];
  const simulatedStream = { id: "software-simulation-stream" } as unknown as MediaStream;
  let readiness: Body = { state: "READINESS_STATE_READY", clipboardAllowed: true, selectedDisplayId: "display-sim", displays: [{ id: "display-sim", width: 1920, height: 1080 }] };
  let nextClipboard = "software-simulation clipboard";
  let clipboardInput = "synthetic clipboard input";
  let revokeNextInput = false;
    let revocationSent = false;
  let sessionIndex = 0;
  let leaseEpoch = 0;
  const json = (value: Body) => new Response(JSON.stringify(value), { status: 200, headers: { "content-type": "application/json" } });

  class FakePeer {
    connectionState: RTCPeerConnectionState = "new";
    ontrack: ((event: RTCTrackEvent) => void) | null = null;
    onicecandidate: ((event: RTCPeerConnectionIceEvent) => void) | null = null;
    onconnectionstatechange: ((event: Event) => void) | null = null;
    ondatachannel: ((event: RTCDataChannelEvent) => void) | null = null;
    closed = false;
    localDescription?: RTCSessionDescriptionInit;
    remoteDescription?: RTCSessionDescriptionInit;
    dataChannel?: FakeDataChannel;

    constructor(_configuration?: RTCConfiguration) { peers.push(this); }
    createOffer = vi.fn(async () => ({ type: "offer", sdp: "simulated-offer" } as RTCSessionDescriptionInit));
    setLocalDescription = vi.fn(async (description?: RTCSessionDescriptionInit) => { this.localDescription = description; });
    setRemoteDescription = vi.fn(async (description: RTCSessionDescriptionInit) => {
      this.remoteDescription = description;
      this.dataChannel = {
        label: "data",
        readyState: "open",
        send: (payload) => {
          const message = JSON.parse(payload) as { kind: string; request: Body };
          const commandId = String(message.request.commandId ?? "");
          const action = message.request.action as { pointer?: { x?: number; y?: number } } | undefined;
          dataEvents.push({
            kind: message.kind,
            pointer: action?.pointer ? { x: Number(action.pointer.x), y: Number(action.pointer.y) } : undefined,
            clipboardBytes: message.kind === "clipboard" ? new TextEncoder().encode(String(message.request.text ?? "")).byteLength : undefined,
          });
          if (message.kind === "input" && revokeNextInput) {
            revokeNextInput = false;
            revocationSent = true;
            this.dataChannel?.onmessage?.({ data: JSON.stringify({ error: "controller lease revoked" }) });
            return;
          }
          const receipt: Body = { commandId, outcome: 1 };
          if (message.kind === "clipboard" && message.request.direction === 2) receipt.text = nextClipboard;
          this.dataChannel?.onmessage?.({ data: JSON.stringify({ receipt }) });
        },
      };
      this.ondatachannel?.({ channel: this.dataChannel } as unknown as RTCDataChannelEvent);
      this.ontrack?.({ streams: [simulatedStream] } as unknown as RTCTrackEvent);
      this.connectionState = "connected";
      this.onconnectionstatechange?.(new Event("connectionstatechange"));
    });
    close = vi.fn(() => { this.closed = true; this.connectionState = "closed"; });
    emitConnectionState(state: RTCPeerConnectionState) {
      this.connectionState = state;
      this.onconnectionstatechange?.(new Event("connectionstatechange"));
    }
  }
  vi.stubGlobal("RTCPeerConnection", FakePeer as unknown as typeof RTCPeerConnection);
  vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = String(input);
    const body = init?.body ? JSON.parse(String(init.body)) as Body : {};
    calls.push({ path, body });
    const method = path.split("/").at(-1) ?? "";
    if (method === "GetReadiness") return json(readiness);
    if (method === "OpenSession") {
      sessionIndex += 1;
      const control = Boolean(body.control);
      const epoch = control ? ++leaseEpoch : 0;
      return json({
        ref: { sessionId: `simulation-session-${sessionIndex}` },
        leaseId: control ? `simulation-lease-${sessionIndex}` : "",
        leaseEpoch: String(epoch),
        selectedDisplayId: "display-sim",
        codec: "VP8",
        connectionState: "connecting",
        controller: control,
        geometryRevision: "simulation-geometry",
      });
    }
    if (method === "TakeControl") return json({ ref: body.session, leaseId: `simulation-lease-takeover-${leaseEpoch + 1}`, leaseEpoch: ++leaseEpoch, selectedDisplayId: "display-sim", codec: "VP8", connectionState: "connected", controller: true, geometryRevision: "simulation-geometry" });
    if (method === "ReleaseControl") return json({ ref: body.session, leaseId: "", leaseEpoch: ++leaseEpoch, selectedDisplayId: "display-sim", codec: "VP8", connectionState: "connected", controller: false });
    if (method === "OpenChannel") return json({ grant: { channelId: "simulation-channel" }, routes: [] });
    if (method === "Signal") return json({ payload: btoa(JSON.stringify({ type: "answer", sdp: "simulated-answer" })) });
    if (method === "CloseChannel") return json({ closed: true });
    if (method === "CloseSession") return json({ closed: true });
    if (method === "ReadFrame") return json({ jpeg: "ZmFrZS1qcGVn" });
    if (method === "Input") return json({ outcome: "INPUT_OUTCOME_ACCEPTED" });
    if (method === "Clipboard") {
      if (body.direction === 1) nextClipboard = String(body.text ?? "");
      return json({ outcome: 1, text: nextClipboard });
    }
    return json({});
  }));

  return {
    calls,
    peers,
    dataEvents,
    simulatedStream,
    setReadiness(next: Body) { readiness = next; },
    setClipboardInput(next: string) { clipboardInput = next; },
    revokeNextInput() { revokeNextInput = true; },
    get revocationSent() { return revocationSent; },
    clipboard: {
      readText: vi.fn(async () => clipboardInput),
      writeText: vi.fn(async (text: string) => { nextClipboard = text; }),
    },
  };
}

const ready = { state: 6, selectedDisplayId: "display-sim", displays: [{ id: "display-sim", width: 1920, height: 1080 }] } as unknown as DesktopReadiness;

describe("DesktopMachineTab software stand-in", () => {
  afterEach(() => { vi.unstubAllGlobals(); vi.restoreAllMocks(); });

  it("uses HTTPS frames and types through the selected lease without an ICE connection", async () => {
    const standIn = installDesktopSoftwareStandIn();
    standIn.setReadiness({ state: "READINESS_STATE_READY", httpsFramesAllowed: true, selectedDisplayId: "display-sim" });
    render(<DesktopMachineTab nodeId="software-simulation-node" />);
    fireEvent.click(await screen.findByRole("button", { name: "Open with control" }));
    expect(await screen.findByRole("img", { name: "Selected desktop display" })).toHaveAttribute("src", "data:image/jpeg;base64,ZmFrZS1qcGVn");
    expect(standIn.peers).toHaveLength(0);
    expect(standIn.calls.find((call) => call.path.endsWith("/ReadFrame"))?.body.leaseEpoch).toBe("1");
    const text = screen.getByLabelText("Type into the focused desktop field");
    fireEvent.change(text, { target: { value: "A@" } });
    fireEvent.click(screen.getByRole("button", { name: "Type on desktop" }));
    await waitFor(() => expect(standIn.calls.filter((call) => call.path.endsWith("/Input"))).toHaveLength(2));
    expect(text).toHaveValue("");
    expect(standIn.calls.some((call) => call.path.endsWith("/OpenChannel"))).toBe(false);
  });

  it("presents permission refusal and requires fresh owner readiness before opening", async () => {
    const standIn = installDesktopSoftwareStandIn();
    standIn.setReadiness({ state: 4, reasonCode: "permission_required", recovery: "Grant desktop permission before opening." });
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: standIn.clipboard });
    render(<DesktopMachineTab nodeId="software-simulation-node" />);

    expect(await screen.findByText("Grant desktop permission before opening.")).toBeInTheDocument();
    expect(screen.getByTestId("workspace-pane-state")).toHaveAttribute("data-pane-state", "permission-blocked");
    expect(screen.queryByRole("button", { name: "Open with control" })).not.toBeInTheDocument();
    expect(standIn.calls.some((call) => call.path.endsWith("/OpenSession"))).toBe(false);

    standIn.setReadiness(ready as unknown as Body);
    fireEvent.click(screen.getByRole("button", { name: "Check readiness" }));
    expect(await screen.findByRole("button", { name: "Open with control" })).toBeInTheDocument();
    expect(standIn.calls.filter((call) => call.path.endsWith("/GetReadiness"))).toHaveLength(2);
    expect(standIn.calls.some((call) => call.path.endsWith("/OpenSession"))).toBe(false);
  });

  it.each([375, 1280])("exercises the production DesktopMachineTab caller at %i px (SOFTWARE SIMULATION)", async (width) => {
    const standIn = installDesktopSoftwareStandIn();
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: standIn.clipboard });
    const { container } = render(<div style={{ width }}><DesktopMachineTab nodeId="software-simulation-node" /></div>);

    await screen.findByRole("button", { name: "Open with control" });
    fireEvent.click(screen.getByRole("button", { name: "View only" }));
    await screen.findByRole("button", { name: "Take control" });
    expect(screen.getByTestId("workspace-pane-state")).toHaveAttribute("data-pane-state", "observe");
    fireEvent.click(screen.getByRole("button", { name: "Take control" }));
    const video = await screen.findByLabelText("Selected desktop display");
    await waitFor(() => expect((video as HTMLVideoElement).srcObject).toBe(standIn.simulatedStream));
    expect(screen.getByTestId("workspace-pane-state")).toHaveAttribute("data-pane-state", "controlling");
    const toolbar = screen.getByRole("toolbar", { name: "Desktop controls" });
    expect(toolbar).toHaveAttribute("data-pane-state", "controlling");

    vi.spyOn(video, "getBoundingClientRect").mockReturnValue({ x: 0, y: 0, left: 0, top: 0, right: 320, bottom: 180, width: 320, height: 180, toJSON: () => ({}) });
    fireEvent.pointerDown(video, { pointerId: 4, clientX: 70, clientY: 60, button: 0 });
    await waitFor(() => expect(standIn.calls.some((call) => call.path.endsWith("/Signal"))).toBe(true));
    await waitFor(() => expect(standIn.dataEvents.some((event) => event.kind === "input")).toBe(true));
    const pointerEvent = standIn.dataEvents.find((event) => event.kind === "input" && event.pointer);
    expect(pointerEvent?.pointer?.x).toBeGreaterThanOrEqual(0);
    expect(pointerEvent?.pointer?.x).toBeLessThan(1920);
    expect(pointerEvent?.pointer?.y).toBeGreaterThanOrEqual(0);
    expect(pointerEvent?.pointer?.y).toBeLessThan(1080);
    expect(standIn.calls.some((call) => call.path.endsWith("/Input"))).toBe(false); // synthetic data-channel acknowledgement

    standIn.setClipboardInput("x".repeat(64 * 1024 + 1));
    fireEvent.click(screen.getByRole("button", { name: "Send text" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(/exceeds the 64 KiB remote-surface limit/);
    expect(standIn.dataEvents.some((event) => event.kind === "clipboard" && event.clipboardBytes)).toBe(false);
    standIn.setClipboardInput("synthetic clipboard input");
    fireEvent.click(screen.getByRole("button", { name: "Send text" }));
    await waitFor(() => expect(standIn.dataEvents.some((event) => event.kind === "clipboard" && event.clipboardBytes === 25)).toBe(true));
    await waitFor(() => expect(standIn.peers[0]?.dataChannel).toBeDefined());
    fireEvent.click(screen.getByRole("button", { name: "Read text" }));
    await waitFor(() => expect(standIn.clipboard.writeText).toHaveBeenCalledWith("software-simulation clipboard"));

    fireEvent.click(screen.getByRole("button", { name: "Release control" }));
    await waitFor(() => expect(screen.getByTestId("workspace-pane-state")).toHaveAttribute("data-pane-state", "observe"));
    expect(standIn.calls.some((call) => call.path.endsWith("/ReleaseControl"))).toBe(true);

    fireEvent.click(screen.getByRole("button", { name: "Take control" }));
    await waitFor(() => expect(screen.getByTestId("workspace-pane-state")).toHaveAttribute("data-pane-state", "controlling"));
    standIn.peers.at(-1)?.emitConnectionState("disconnected");
    expect(await screen.findByText("Desktop connection lost. Reconnect to start a new session and lease.")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Reconnect" }));
    await waitFor(() => expect(standIn.peers.length).toBeGreaterThanOrEqual(2));
    await waitFor(() => expect(screen.getByTestId("workspace-pane-state")).toHaveAttribute("data-pane-state", "controlling"));
    expect(standIn.calls.some((call) => call.path.endsWith("/CloseChannel"))).toBe(true);
    expect(standIn.calls.some((call) => call.path.endsWith("/CloseSession"))).toBe(true);
    expect(screen.getByText("Connection details").closest("details")).not.toHaveAttribute("open");
    expect(screen.getByLabelText("Desktop transport statistics")).toHaveTextContent("VP8");

    standIn.revokeNextInput();
    const reconnectedVideo = screen.getByLabelText("Selected desktop display");
    vi.spyOn(reconnectedVideo, "getBoundingClientRect").mockReturnValue({ x: 0, y: 0, left: 0, top: 0, right: 320, bottom: 180, width: 320, height: 180, toJSON: () => ({}) });
    fireEvent.pointerDown(reconnectedVideo, { pointerId: 5, clientX: 70, clientY: 60, button: 0 });
    await waitFor(() => expect(standIn.revocationSent).toBe(true));
    await waitFor(() => expect(screen.getByTestId("workspace-pane-state")).toHaveAttribute("data-pane-state", "revoked"));
    fireEvent.click(screen.getByRole("button", { name: "Close session" }));
    await waitFor(() => expect(standIn.calls.filter((call) => call.path.endsWith("/CloseSession")).length).toBeGreaterThanOrEqual(2));
    expect(standIn.calls.some((call) => call.path.includes("device_control.v1.desktop.DesktopSessionService"))).toBe(true);
    expect(standIn.calls.some((call) => call.path.includes("vrooli_bridge.v1.interactive.InteractiveDesktopService"))).toBe(true);
    expect(container.querySelector("[data-testid=workspace-surface-pane]")).toHaveAttribute("data-surface-owner", "device-control");
    expect(standIn.calls.map(({ path }) => path).join("\n")).not.toMatch(/https?:\/\/|sdp|clipboard payload/i);
  });
});
