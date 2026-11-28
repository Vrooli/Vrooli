import { fromJson, type JsonValue } from "@bufbuild/protobuf";
import { DesktopReadinessSchema, DesktopSessionSchema } from "@vrooli/proto-types/device-control/v1/desktop/desktop_pb";
import type { DesktopReadiness, DesktopSession } from "@vrooli/proto-types/device-control/v1/desktop/desktop_pb";

// The plain Connect-JSON transport must serialize uint64 as decimal strings
// and omit message-runtime metadata when a decoded session is echoed.
export function desktopJSON(value: unknown): string {
  return JSON.stringify(value, (key, item) => key === "$typeName" ? undefined : typeof item === "bigint" ? item.toString() : item);
}

export interface DesktopSessionTransport {
  readiness(request: unknown): Promise<DesktopReadiness>;
  readFrame?(request: unknown, signal?: AbortSignal): Promise<{ jpeg: string }>;
  open(request: unknown): Promise<DesktopSession>;
  attachViewer(request: unknown): Promise<DesktopSession>;
  takeControl(request: unknown): Promise<DesktopSession>;
  releaseControl(request: unknown): Promise<DesktopSession>;
  signal(request: unknown): Promise<unknown>;
  input(request: unknown): Promise<unknown>;
  clipboard(request: unknown): Promise<unknown>;
  close(request: unknown): Promise<unknown>;
}

export interface BridgeInteractiveTransport {
  open(request: unknown): Promise<unknown>;
  signal(request: unknown): Promise<unknown>;
  close(request: unknown): Promise<unknown>;
}

// Same-origin Connect-JSON adapter. The browser only talks to the control
// plane; node URLs, credentials, SDP secrets, and clipboard contents are not
// placed in durable client state or query strings.
export function createDesktopSessionTransport(baseUrl = ""): DesktopSessionTransport {
  const call = async <T>(method: string, request: unknown, signal?: AbortSignal): Promise<T> => {
    const response = await fetch(`${baseUrl}/vrooli.device_control.v1.desktop.DesktopSessionService/${method}`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      credentials: "same-origin",
      signal,
      body: desktopJSON(request),
    });
    if (!response.ok) {
      if (method === "GetReadiness" && response.status === 503) {
        throw new Error("Web Console could not reach this machine’s desktop service through Bridge.");
      }
      throw new Error(`Desktop ${method} failed (${String(response.status)}).`);
    }
    return (await response.json()) as T;
  };
  const session = async (method: string, request: unknown) => fromJson(DesktopSessionSchema, await call<JsonValue>(method, request), { ignoreUnknownFields: true });
  return {
    readiness: async (request) => fromJson(DesktopReadinessSchema, await call<JsonValue>("GetReadiness", request), { ignoreUnknownFields: true }),
    readFrame: (request, signal) => call("ReadFrame", request, signal),
    open: (request) => session("OpenSession", request),
    attachViewer: (request) => session("AttachViewer", request),
    takeControl: (request) => session("TakeControl", request),
    releaseControl: (request) => session("ReleaseControl", request),
    signal: (request) => call("Signal", request),
    input: (request) => call("Input", request),
    clipboard: (request) => call("Clipboard", request),
    close: (request) => call("CloseSession", request),
  };
}

export function createBridgeInteractiveTransport(baseUrl = ""): BridgeInteractiveTransport {
  const call = async <T>(method: string, request: unknown): Promise<T> => {
    const response = await fetch(`${baseUrl}/vrooli.vrooli_bridge.v1.interactive.InteractiveDesktopService/${method}`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      credentials: "same-origin",
      body: desktopJSON(request),
    });
    if (!response.ok) throw new Error(`desktop channel ${method} failed (${String(response.status)})`);
    return (await response.json()) as T;
  };
  return {
    open: (request) => call("OpenChannel", request),
    signal: (request) => call("Signal", request),
    close: (request) => call("CloseChannel", request),
  };
}

export interface DesktopPeerSignal {
  sendOffer(description: RTCSessionDescriptionInit): Promise<RTCSessionDescriptionInit>;
  sendIce(candidate: RTCIceCandidateInit): Promise<void>;
}

export interface DesktopPeer {
  connection: RTCPeerConnection;
  connect(signal: DesktopPeerSignal): Promise<void>;
  sendData(payload: string): Promise<boolean>;
  close(): void;
}

/** Keep transitional WebRTC states distinct from a lost transport in the pane. */
export function projectDesktopPeerState(state: RTCPeerConnectionState): "connecting" | "connected" | "disconnected" | "failed" | "closed" {
  switch (state) {
    case "new":
    case "connecting": return "connecting";
    case "connected": return "connected";
    case "disconnected": return "disconnected";
    case "failed": return "failed";
    case "closed": return "closed";
  }
}

export interface DesktopIceRoute {
  url?: string;
  username?: string;
  credential?: string;
}

export function createDesktopPeer(onStream: (stream: MediaStream) => void, onIce?: (candidate: RTCIceCandidate) => void, routes: DesktopIceRoute[] = [], onState?: (state: RTCPeerConnectionState) => void, onData?: (payload: string) => void): DesktopPeer {
  const iceServers = routes
    .filter((route) => typeof route.url === "string" && route.url.length > 0)
    .map((route) => ({ urls: route.url as string, username: route.username, credential: route.credential }));
  const connection = new RTCPeerConnection(iceServers.length > 0 ? { iceServers } : undefined);
  let dataChannel: RTCDataChannel | undefined;
  connection.ontrack = (event) => {
    const [stream] = event.streams;
    if (stream) onStream(stream);
  };
  connection.onconnectionstatechange = () => onState?.(projectDesktopPeerState(connection.connectionState));
  connection.ondatachannel = (event) => {
    if (event.channel.label !== "data") return;
    dataChannel = event.channel;
    event.channel.onmessage = (message) => onData?.(typeof message.data === "string" ? message.data : "");
  };
    connection.onicecandidate = (event) => {
      if (event.candidate) onIce?.(event.candidate);
    };
  return {
    connection,
    async connect(signal) {
      const previousIceHandler = connection.onicecandidate;
      connection.onicecandidate = (event) => {
        previousIceHandler?.call(connection, event);
        if (event.candidate) void signal.sendIce(event.candidate.toJSON());
      };
      const offer = await connection.createOffer({ offerToReceiveVideo: true, offerToReceiveAudio: false });
      await connection.setLocalDescription(offer);
      const answer = await signal.sendOffer(offer);
      await connection.setRemoteDescription(answer);
    },
    async sendData(payload) {
      const deadline = Date.now() + 2000;
      while (!dataChannel || dataChannel.readyState !== "open") {
        if (Date.now() >= deadline || dataChannel?.readyState === "closed") return false;
        await new Promise((resolve) => setTimeout(resolve, 25));
      }
      dataChannel.send(payload);
      return true;
    },
    close() {
      connection.ontrack = null;
      connection.onicecandidate = null;
      connection.onconnectionstatechange = null;
      connection.ondatachannel = null;
      dataChannel = undefined;
      connection.close();
    },
  };
}
