import { act, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { useAgentVoice, selectAgentVoice } from "../useAgentVoice";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import { useAgentVoiceStore } from "../../stores/useAgentVoiceStore";
import { parseAgentVoiceEvent } from "../../api/agentVoice";

const media = vi.hoisted(() => ({ start: vi.fn(), stop: vi.fn(), ready: vi.fn(), acceptSDP: vi.fn(), acceptAudio: vi.fn(), mute: vi.fn(), callbacks: null as null | { onConnected: () => void; onLevel: (level: number) => void; onError: (error: Error) => void } }));
vi.mock("@vrooli/audio-capture-browser", async (load) => ({ ...await load<object>(), DuplexAudio: class { constructor(callbacks: typeof media.callbacks) { media.callbacks = callbacks; } start = media.start; stop = media.stop; ready = media.ready; acceptSDP = media.acceptSDP; acceptAudio = media.acceptAudio; mute = media.mute; } }));
vi.mock("../../api/sessions", () => ({ buildSessionWsUrl: (id: string) => `ws://localhost/sessions/${id}/ws` }));
vi.mock("../useConversationSession", () => ({ refreshConversationSession: vi.fn(() => Promise.resolve({ ok: true })) }));

class Socket {
  static OPEN = 1;
  static instances: Socket[] = [];
  readyState = 1; bufferedAmount = 0;
  sent: { type: string; transport?: string; sdp?: string }[] = [];
  onmessage: ((event: { data: string }) => void) | null = null;
  onclose: (() => void) | null = null; onerror: (() => void) | null = null;
  constructor(readonly url: string) { Socket.instances.push(this); }
  send(data: string) { this.sent.push(JSON.parse(data) as { type: string }); }
  close() { if (this.readyState === 3) return; this.readyState = 3; this.onclose?.(); }
  emit(value: unknown) { this.onmessage?.({ data: JSON.stringify(value) }); }
}
function Harness({ id = "session" }: { id?: string }) {
  const voice = useAgentVoice(id);
  return <><button onClick={() => { void voice.start(); }}>Start</button><output data-testid="partial" data-role={voice.partialTranscriptRole}>{voice.partialTranscript}</output><output data-testid="checking">{String(voice.isChecking)}</output><output data-testid="available">{String(voice.capability.available)}</output><output data-testid="voice-state">{voice.voiceState}:{voice.audioLevel}:{voice.error}</output><output data-testid="native-active">{String(voice.capability.nativeActive)}</output><button onClick={voice.toggleMute}>Mute</button><button onClick={voice.stop}>End voice</button></>;
}
function socket(): Socket { const value = Socket.instances.at(-1); if (!value) throw new Error("missing socket"); return value; }
beforeEach(() => {
  vi.clearAllMocks(); Socket.instances = []; vi.stubGlobal("WebSocket", Socket);
  media.start.mockResolvedValue("v=offer"); media.acceptSDP.mockResolvedValue(undefined);
  useWorkspaceStore.setState({ voiceEnabled: true, agentVoiceMode: "agent", agentVoiceTransport: "webrtc" });
  useAgentVoiceStore.setState({ connected: {} });
});
afterEach(() => { vi.useRealTimers(); });
describe("native voice routing", () => {
  it("uses native voice in Auto only with an active supported native session", () => {
    const cap = { available: true, agent: "codex", nativeActive: false };
    expect(selectAgentVoice("auto", cap)).toBe(false); expect(selectAgentVoice("agent", cap)).toBe(true);
    expect(selectAgentVoice("auto", { ...cap, nativeActive: true })).toBe(true);
    expect(selectAgentVoice("dictation", { ...cap, nativeActive: true })).toBe(false);
    expect(selectAgentVoice("auto", { ...cap, available: false, nativeActive: true })).toBe(false);
  });
  it("shows listening only after media connects, with live level and mute", async () => {
    renderWithProviders(<Harness />);
    act(() => { socket().emit({ type: "capability", capability: { available: true, agent: "codex", nativeActive: true } }); });
    expect(screen.getByTestId("native-active")).toHaveTextContent("true");
    act(() => { screen.getByText("Start").click(); });
    await waitFor(() => { expect(socket().sent.some((m) => m.type === "start" && m.transport === "webrtc")).toBe(true); });
    expect(screen.getByTestId("voice-state")).toHaveTextContent("preparing");
    act(() => { media.callbacks?.onConnected(); media.callbacks?.onLevel(0.7); });
    expect(screen.getByTestId("voice-state")).toHaveTextContent("listening:0.7");
    act(() => { screen.getByRole("button", { name: "Mute" }).click(); }); expect(media.mute).toHaveBeenCalledWith(true);
    act(() => { screen.getByRole("button", { name: "End voice" }).click(); }); expect(media.stop).toHaveBeenCalled(); expect(useAgentVoiceStore.getState().connected).toEqual({});
  });
  it("cancels late setup when changing panes", async () => {
    let resolve!: (sdp: string) => void; media.start.mockImplementation(() => new Promise<string>((r) => { resolve = r; }));
    const view = renderWithProviders(<Harness />); const first = socket();
    act(() => { first.emit({ type: "capability", capability: { available: true, agent: "codex", nativeActive: false } }); });
    act(() => { screen.getByText("Start").click(); }); view.rerender(<Harness id="other" />);
    await act(() => { resolve("late offer"); return Promise.resolve(); }); expect(first.sent.some((m) => m.type === "start")).toBe(false); expect(media.stop).toHaveBeenCalled();
  });
  it("offers explicit dictation after native failure without replay", async () => {
    renderWithProviders(<Harness />);
    act(() => { socket().emit({ type: "capability", capability: { available: true, agent: "codex", nativeActive: false } }); });
    act(() => { screen.getByText("Start").click(); }); await waitFor(() => { expect(media.start).toHaveBeenCalled(); });
    act(() => { socket().emit({ type: "error", message: "Voice unavailable" }); });
    expect(screen.getByTestId("voice-state")).toHaveTextContent("Voice unavailable");
    act(() => { useWorkspaceStore.getState().setAgentVoiceMode("dictation"); }); expect(useWorkspaceStore.getState().agentVoiceMode).toBe("dictation");
    expect(socket().sent.filter((m) => m.type === "start")).toHaveLength(1);
  });
  it("prefers canonical partials without duplicating flat transcript deltas", () => {
    renderWithProviders(<Harness />);
    const emit = (method: string, params: object) => { act(() => { socket().emit({type:"event",method,params}); }); };
    emit("thread/realtime/transcript/delta",{delta:"Hello"});
    emit("thread/realtime/item/transcript/delta",{itemId:"one",delta:"Hello"});
    emit("thread/realtime/transcript/delta",{delta:" world"});
    emit("thread/realtime/item/transcript/delta",{itemId:"one",delta:" world"});
    expect(screen.getByTestId("partial")).toHaveTextContent("Hello world");
    emit("thread/realtime/item/transcript/delta",{itemId:"two",delta:"Next"});
    expect(screen.getByTestId("partial")).toHaveTextContent("Next");
  });
  it("distinguishes discovery from a confirmed unavailable capability", () => {
    renderWithProviders(<Harness />);
    expect(screen.getByTestId("checking")).toHaveTextContent("true");
    act(() => { socket().emit({type:"capability",capability:{available:false,agent:"codex",nativeActive:false,retryable:true}}); });
    expect(screen.getByTestId("checking")).toHaveTextContent("true");
    act(() => { socket().emit({type:"capability",capability:{available:false,agent:"claude",nativeActive:false}}); });
    expect(screen.getByTestId("checking")).toHaveTextContent("false");
    expect(media.start).not.toHaveBeenCalled();
  });
  it("rediscovers native capability after a new session finishes startup without starting the mic", async () => {
    vi.useFakeTimers(); renderWithProviders(<Harness />);
    act(() => { socket().emit({type:"capability",capability:{available:false,agent:"codex",nativeActive:false,retryable:true}}); socket().close(); });
    await act(async () => { await vi.advanceTimersByTimeAsync(1000); });
    expect(Socket.instances.length).toBeGreaterThan(1);
    act(() => { socket().emit({type:"capability",capability:{available:true,agent:"codex",nativeActive:false}}); });
    expect(screen.getByTestId("available")).toHaveTextContent("true");
    expect(screen.getByTestId("checking")).toHaveTextContent("false");
    expect(media.start).not.toHaveBeenCalled();
    expect(Socket.instances.flatMap(s=>s.sent).some(m=>m.type==="start")).toBe(false);
  });
  it.each([false, true])("bounds retries for rapidly closed sockets (available=%s) and cancels them when leaving", async (available) => {
    vi.useFakeTimers(); const view=renderWithProviders(<Harness />);
    for(let i=0;i<20;i++) {
      act(()=>{socket().emit({type:"capability",capability:{available,agent:"codex",nativeActive:false,retryable:true}});socket().close();});
      await act(async()=>{await vi.advanceTimersByTimeAsync(60_000);});
    }
    expect(Socket.instances.length).toBeLessThanOrEqual(11);
    expect(screen.getByTestId("checking")).toHaveTextContent("false");
    const before=Socket.instances.length; view.unmount();
    await act(async()=>{await vi.advanceTimersByTimeAsync(60_000);});
    expect(Socket.instances).toHaveLength(before); expect(media.start).not.toHaveBeenCalled();
  });
  it("does not retry a known unsupported agent", async () => {
    vi.useFakeTimers(); renderWithProviders(<Harness />);
    act(()=>{socket().emit({type:"capability",capability:{available:false,agent:"claude",nativeActive:false,retryable:false}});socket().close();});
    await act(async()=>{await vi.advanceTimersByTimeAsync(60_000);});
    expect(Socket.instances).toHaveLength(1); expect(media.start).not.toHaveBeenCalled();
  });
  it("discards a pending discovery retry when switching sessions", async () => {
    vi.useFakeTimers(); const view=renderWithProviders(<Harness />);
    act(()=>{socket().emit({type:"capability",capability:{available:false,agent:"codex",nativeActive:false,retryable:true}});socket().close();});
    view.rerender(<Harness id="other" />);
    await act(async()=>{await vi.advanceTimersByTimeAsync(1000);});
    expect(Socket.instances).toHaveLength(2);
    expect(socket().url).toContain("other"); expect(media.start).not.toHaveBeenCalled();
  });
  it("tracks provider speaker identity across canonical and flat caption streams", () => {
    renderWithProviders(<Harness />);
    const emit=(method:string,params:object)=>{act(()=>{socket().emit({type:"event",method,params});});};
    emit("thread/realtime/item/started",{item:{id:"user-1",type:"transcriptSegment",role:"user"}});
    emit("thread/realtime/item/transcript/delta",{itemId:"user-1",delta:"My question"});
    expect(screen.getByTestId("partial")).toHaveAttribute("data-role","user");
    emit("thread/realtime/item/started",{item:{id:"reply-1",type:"transcriptSegment",role:"assistant"}});
    emit("thread/realtime/item/transcript/delta",{itemId:"reply-1",delta:"My answer"});
    expect(screen.getByTestId("partial")).toHaveTextContent("My answer");
    expect(screen.getByTestId("partial")).toHaveAttribute("data-role","assistant");
    emit("thread/realtime/item/completed",{item:{id:"reply-1",type:"transcriptSegment",role:"assistant"}});
    expect(screen.getByTestId("partial")).toHaveTextContent("");
    emit("thread/realtime/transcript/delta",{role:"user",delta:"Next question"});
    expect(screen.getByTestId("partial")).toHaveAttribute("data-role","user");
  });
  it("retains the complete multi-line native response across streamed deltas", () => {
    renderWithProviders(<Harness />);
    const parts = Array.from({ length: 30 }, (_, index) => `Response line ${String(index)}.\n`);
    for (const delta of parts) act(() => { socket().emit({ type: "event", method: "thread/realtime/item/transcript/delta", params: { itemId: "long-reply", delta } }); });
    expect(screen.getByTestId("partial").textContent).toBe(parts.join(""));
  });
  it("times out silent discovery handshakes and ends checking after bounded retries", async () => {
    vi.useFakeTimers(); renderWithProviders(<Harness />);
    for (let attempt = 0; attempt < 12; attempt++) {
      await act(async () => { await vi.advanceTimersByTimeAsync(20_000); });
    }
    expect(Socket.instances).toHaveLength(11);
    expect(screen.getByTestId("checking")).toHaveTextContent("false");
    expect(media.start).not.toHaveBeenCalled();
  });
  it("discards malformed frames", () => { expect(parseAgentVoiceEvent("garbage")).toBeNull(); expect(parseAgentVoiceEvent("null")).toBeNull(); expect(parseAgentVoiceEvent("{}")).toBeNull(); });
});
