import { afterEach, describe, expect, it, vi } from "vitest";
import { createBridgeInteractiveTransport, createDesktopPeer, createDesktopSessionTransport } from "./desktopTransport";
import { cleanupDesktopOpenFailure } from "./DesktopMachineTab";

describe("desktop transport", () => {
  it("uses same-origin typed control-plane paths without node URLs", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ state: 6 }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    await createDesktopSessionTransport().readiness({ surface: {} });
    expect(fetchMock).toHaveBeenCalledWith(
      "/vrooli.device_control.v1.desktop.DesktopSessionService/GetReadiness",
      expect.objectContaining({ credentials: "same-origin" }),
    );
    expect(JSON.stringify(fetchMock.mock.calls[0])).not.toContain("ssh");
    vi.unstubAllGlobals();
  });

  it("routes SDP signaling through the same-origin Device Control service", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ accepted: true }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    await createDesktopSessionTransport().signal({ payload: "offer" });
    expect(fetchMock).toHaveBeenCalledWith(
      "/vrooli.device_control.v1.desktop.DesktopSessionService/Signal",
      expect.objectContaining({ credentials: "same-origin" }),
    );
    vi.unstubAllGlobals();
  });

  it("routes channel admission and signaling through the same-origin Bridge service", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ accepted: true }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    await createBridgeInteractiveTransport("/api/v1/desktop-bridge").signal({ requestId: "r1" });
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/desktop-bridge/vrooli.vrooli_bridge.v1.interactive.InteractiveDesktopService/Signal",
      expect.objectContaining({ credentials: "same-origin" }),
    );
    vi.unstubAllGlobals();
  });

  it("cleans up a session and Bridge grant when peer setup fails", async () => {
    const bridgeClose = vi.fn().mockResolvedValue(undefined);
    const sessionClose = vi.fn().mockResolvedValue(undefined);
    const session = { ref: { sessionId: "session-1" } } as never;
    await cleanupDesktopOpenFailure({ close: bridgeClose }, { close: sessionClose }, { channelId: "channel-1" }, session);
    expect(bridgeClose).toHaveBeenCalledWith({ grant: { channelId: "channel-1" }, reason: "web_console_open_failed" });
    expect(sessionClose).toHaveBeenCalledWith({ session: { sessionId: "session-1" }, reason: "web_console_open_failed" });
  });

  it("maps every DesktopSessionService method to its typed path", async () => {
    const fetchMock = vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify({ ok: true }), { status: 200 })));
    vi.stubGlobal("fetch", fetchMock);
    const t = createDesktopSessionTransport();
    await t.open({});
    await t.attachViewer({});
    await t.takeControl({});
    await t.input({});
    await t.clipboard({});
    await t.close({});
    const paths = fetchMock.mock.calls.map((c) => c[0] as string);
    const base = "/vrooli.device_control.v1.desktop.DesktopSessionService/";
    expect(paths).toEqual([
      `${base}OpenSession`,
      `${base}AttachViewer`,
      `${base}TakeControl`,
      `${base}Input`,
      `${base}Clipboard`,
      `${base}CloseSession`,
    ]);
    vi.unstubAllGlobals();
  });

  it("maps Bridge channel open and close to their typed paths", async () => {
    const fetchMock = vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify({ ok: true }), { status: 200 })));
    vi.stubGlobal("fetch", fetchMock);
    const t = createBridgeInteractiveTransport();
    await t.open({});
    await t.close({});
    const paths = fetchMock.mock.calls.map((c) => c[0] as string);
    expect(paths).toEqual([
      "/vrooli.vrooli_bridge.v1.interactive.InteractiveDesktopService/OpenChannel",
      "/vrooli.vrooli_bridge.v1.interactive.InteractiveDesktopService/CloseChannel",
    ]);
    vi.unstubAllGlobals();
  });

  it("throws a labeled error when a desktop call is not ok", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("nope", { status: 503 })));
    await expect(createDesktopSessionTransport().readiness({})).rejects.toThrow("desktop GetReadiness failed (503)");
    await expect(createBridgeInteractiveTransport().open({})).rejects.toThrow("desktop channel OpenChannel failed (503)");
    vi.unstubAllGlobals();
  });
});

describe("createDesktopPeer", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  function installMockPeerConnection() {
    const instances: FakePeerConnection[] = [];
    class FakePeerConnection {
      config: unknown;
      connectionState: RTCPeerConnectionState = "new";
      ontrack: ((event: unknown) => void) | null = null;
      onicecandidate: ((event: unknown) => void) | null = null;
      onconnectionstatechange: ((event: unknown) => void) | null = null;
      ondatachannel: ((event: unknown) => void) | null = null;
      localDescription: unknown;
      remoteDescription: unknown;
      closed = false;
      constructor(config?: unknown) {
        this.config = config;
        instances.push(this);
      }
      createOffer = vi.fn().mockResolvedValue({ type: "offer", sdp: "x" });
      setLocalDescription = vi.fn(async (d: unknown) => { this.localDescription = d; });
      setRemoteDescription = vi.fn(async (d: unknown) => { this.remoteDescription = d; });
      close = vi.fn(() => { this.closed = true; });
    }
    vi.stubGlobal("RTCPeerConnection", FakePeerConnection as unknown as typeof RTCPeerConnection);
    return { instances };
  }

  it("configures ICE servers only for routes with a url and forwards track streams", () => {
    const { instances } = installMockPeerConnection();
    const onStream = vi.fn();
    const onState = vi.fn();
    createDesktopPeer(onStream, undefined, [{ url: "turn:a" }, { url: "" }, { username: "u" }], onState);
    const pc = instances[0]!;
    expect((pc.config as { iceServers: unknown[] }).iceServers).toEqual([{ urls: "turn:a", username: undefined, credential: undefined }]);

    const stream = { id: "s" } as unknown as MediaStream;
    pc.ontrack?.({ streams: [stream] });
    expect(onStream).toHaveBeenCalledWith(stream);

    pc.connectionState = "connected";
    pc.onconnectionstatechange?.({});
    expect(onState).toHaveBeenCalledWith("connected");
  });

  it("omits the config entirely when no route carries a url", () => {
    const { instances } = installMockPeerConnection();
    createDesktopPeer(vi.fn(), undefined, []);
    expect(instances[0]!.config).toBeUndefined();
  });

  it("delivers string data-channel messages and ignores non-data channels", () => {
    const { instances } = installMockPeerConnection();
    const onData = vi.fn();
    createDesktopPeer(vi.fn(), undefined, [], undefined, onData);
    const pc = instances[0]!;

    pc.ondatachannel?.({ channel: { label: "other" } });
    const dataChannel: { label: string; onmessage?: (m: unknown) => void } = { label: "data" };
    pc.ondatachannel?.({ channel: dataChannel });
    dataChannel.onmessage?.({ data: "hello" });
    dataChannel.onmessage?.({ data: new ArrayBuffer(2) });
    expect(onData).toHaveBeenNthCalledWith(1, "hello");
    expect(onData).toHaveBeenNthCalledWith(2, "");
  });

  it("negotiates an offer/answer and relays ICE candidates through the signal", async () => {
    const { instances } = installMockPeerConnection();
    const onIce = vi.fn();
    const peer = createDesktopPeer(vi.fn(), onIce, []);
    const pc = instances[0]!;

    const sendIce = vi.fn().mockResolvedValue(undefined);
    const sendOffer = vi.fn().mockResolvedValue({ type: "answer", sdp: "y" });
    await peer.connect({ sendOffer, sendIce });
    expect(pc.createOffer).toHaveBeenCalledWith({ offerToReceiveVideo: true, offerToReceiveAudio: false });
    expect(pc.localDescription).toEqual({ type: "offer", sdp: "x" });
    expect(pc.remoteDescription).toEqual({ type: "answer", sdp: "y" });

    pc.onicecandidate?.({ candidate: { toJSON: () => ({ candidate: "c" }) } });
    expect(onIce).toHaveBeenCalled();
    expect(sendIce).toHaveBeenCalledWith({ candidate: "c" });
  });

  it("sendData waits for an open channel then writes the payload", async () => {
    const { instances } = installMockPeerConnection();
    const peer = createDesktopPeer(vi.fn(), undefined, []);
    const pc = instances[0]!;
    const send = vi.fn();
    const channel: { label: string; readyState: string; send: typeof send; onmessage?: unknown } = { label: "data", readyState: "connecting", send };
    pc.ondatachannel?.({ channel });
    const promise = peer.sendData("payload");
    channel.readyState = "open";
    await expect(promise).resolves.toBe(true);
    expect(send).toHaveBeenCalledWith("payload");
  });

  it("sendData returns false when the channel reports closed", async () => {
    const { instances } = installMockPeerConnection();
    const peer = createDesktopPeer(vi.fn(), undefined, []);
    const pc = instances[0]!;
    const channel = { label: "data", readyState: "closed", send: vi.fn() };
    pc.ondatachannel?.({ channel });
    await expect(peer.sendData("x")).resolves.toBe(false);
  });

  it("close detaches handlers and closes the connection", () => {
    const { instances } = installMockPeerConnection();
    const peer = createDesktopPeer(vi.fn(), undefined, []);
    const pc = instances[0]!;
    peer.close();
    expect(pc.closed).toBe(true);
    expect(pc.ontrack).toBeNull();
    expect(pc.onicecandidate).toBeNull();
    expect(pc.ondatachannel).toBeNull();
  });
});
