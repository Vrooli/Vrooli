import { useCallback, useEffect, useRef, useState } from "react";
import { DuplexAudio, type VoiceState } from "../audio-integration";
import { agentVoiceURL, parseAgentVoiceEvent, type AgentVoiceCapability } from "../api/agentVoice";
import { useAgentVoiceStore } from "../stores/useAgentVoiceStore";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { refreshConversationSession } from "./useConversationSession";

const DISCOVERY_RETRY_DELAYS_MS = [250, 500, 1000, 2000, 4000, 5000, 5000, 5000, 5000, 5000];
type TranscriptRole = "user" | "assistant";
const transcriptRole = (role: unknown): TranscriptRole | null => role === "user" || role === "assistant" ? role : null;

const UNAVAILABLE: AgentVoiceCapability = { available: false, agent: "", nativeActive: false };

export function selectAgentVoice(mode: "auto" | "agent" | "dictation", capability: AgentVoiceCapability): boolean {
  return mode === "agent" || (mode === "auto" && capability.available && capability.nativeActive);
}

export function useAgentVoice(sessionId: string | null) {
  const [capability, setCapability] = useState(UNAVAILABLE);
  const [isChecking, setIsChecking] = useState(Boolean(sessionId));
  const [voiceState, setVoiceState] = useState<VoiceState>("idle");
  const [audioLevel, setAudioLevel] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const [partialTranscript, setPartialTranscript] = useState("");
  const [partialTranscriptRole, setPartialTranscriptRole] = useState<TranscriptRole | null>(null);
  const transcriptRoles = useRef(new Map<string, TranscriptRole>());
  const discoveryRetryCount = useRef(0);
  const [muted, setMuted] = useState(false);
  const [connectionAttempt, setConnectionAttempt] = useState(0);
  const socketRef = useRef<WebSocket | null>(null);
  const mediaRef = useRef<DuplexAudio | null>(null);
  const startedRef = useRef(false);
  const transcriptItem = useRef<string | null>(null);
  const generation = useRef(0);
  const timeoutRef = useRef<ReturnType<typeof setTimeout>>();
  const mode = useWorkspaceStore((s) => s.agentVoiceMode);
  const transport = useWorkspaceStore((s) => s.agentVoiceTransport);
  const enabled = useWorkspaceStore((s) => s.voiceEnabled);
  const selected = selectAgentVoice(mode, capability);

  const stop = useCallback(() => {
    generation.current++;
    if (timeoutRef.current) clearTimeout(timeoutRef.current);
    const media = mediaRef.current; mediaRef.current = null; media?.stop();
    if (startedRef.current && socketRef.current?.readyState === WebSocket.OPEN) socketRef.current.send(JSON.stringify({ type: "stop" }));
    startedRef.current = false;
    transcriptItem.current = null;
    transcriptRoles.current.clear();
    setPartialTranscriptRole(null);
    if (sessionId) useAgentVoiceStore.getState().setConnected(sessionId, false);
    setVoiceState("idle"); setAudioLevel(0); setMuted(false); setPartialTranscript("");
  }, [sessionId]);

  useEffect(() => { discoveryRetryCount.current = 0; }, [sessionId, transport]);

  useEffect(() => {
    setCapability(UNAVAILABLE); setError(null); setIsChecking(Boolean(sessionId));
    if (!sessionId) return;
    let disposed = false;
    let observed: AgentVoiceCapability | undefined;
    let readyAt: number | undefined;
    let retryTimer: ReturnType<typeof setTimeout> | undefined;
    const socket = new WebSocket(agentVoiceURL(sessionId)); socketRef.current = socket;
    // A silent handshake must not leave discovery pending indefinitely.
    const discoveryTimeout = setTimeout(() => { if (!disposed && !observed) socket.close(); }, 10_000);
    const fail = (message: string) => { if (disposed) return; stop(); setError(message); };
    socket.onmessage = (message: MessageEvent<string>) => {
      const event = parseAgentVoiceEvent(message.data); if (!event || disposed) return;
      if (event.type === "capability" && event.capability) { observed = event.capability; if (observed.available && readyAt === undefined) readyAt = Date.now(); setCapability(observed); clearTimeout(discoveryTimeout);
        const settings = useWorkspaceStore.getState();
        setIsChecking(!observed.available && Boolean(observed.retryable) && settings.voiceEnabled && settings.agentVoiceMode !== "dictation" && discoveryRetryCount.current < DISCOVERY_RETRY_DELAYS_MS.length);
        return; }
      if (event.type === "error") { fail(event.message ?? "Agent voice failed. Use Dictation or retry."); return; }
      if (event.type === "accepted" && transport === "websocket") mediaRef.current?.ready();
      if (event.type === "stopped") { startedRef.current = false; stop(); return; }
      if (event.type === "replaced") { startedRef.current = false; stop(); setError("Voice moved to another controller. Tap the microphone to reconnect."); return; }
      if (event.type !== "event") return;
      const params = event.params;
      if (event.method === "thread/realtime/started") setCapability((s) => ({ ...s, nativeActive: true }));
      if (event.method === "thread/realtime/closed") {
        // Connecting explicitly replaces a terminal-owned voice transport. Its
        // old closed notification must not cancel the pending browser offer.
        if (useAgentVoiceStore.getState().connected[sessionId]) { startedRef.current = false; stop(); }
        setCapability((s) => ({ ...s, nativeActive: false }));
      }
      if (event.method === "thread/realtime/error") fail("Codex voice encountered an error. Use Dictation or retry.");
      if (event.method === "thread/realtime/sdp" && params?.sdp) void mediaRef.current?.acceptSDP(params.sdp).catch(() => { fail("Voice negotiation failed. Use Dictation or retry."); });
      if (event.method === "thread/realtime/outputAudio/delta" && params?.audio) {
        try { mediaRef.current?.acceptAudio(params.audio); } catch { fail("Voice playback could not keep up. Reconnect to continue."); }
      }
      if (params?.item?.type === "input_audio_buffer.speech_started") mediaRef.current?.interrupt();
      if (event.method === "thread/realtime/item/started" && params?.item?.type === "transcriptSegment" && params.item.id) {
        const role = transcriptRole(params.item.role);
        if (role) {
          if (transcriptRoles.current.size >= 64) {
            const oldest = transcriptRoles.current.keys().next().value;
            if (oldest) transcriptRoles.current.delete(oldest);
          }
          transcriptRoles.current.set(params.item.id, role);
        }
      }
      // Current Codex emits both flat and canonical item deltas. Prefer the
      // item stream once present, so a spoken phrase never appears twice.
      if (event.method === "thread/realtime/item/transcript/delta" && params?.itemId) {
        const sameItem = transcriptItem.current === params.itemId;
        transcriptItem.current = params.itemId;
        setPartialTranscriptRole(transcriptRoles.current.get(params.itemId) ?? null);
        setPartialTranscript((s) => ((sameItem ? s : "") + (params.delta ?? "")).slice(-8_192));
      } else if (event.method === "thread/realtime/transcript/delta" && transcriptItem.current === null) {
        setPartialTranscriptRole(transcriptRole(params?.role));
        setPartialTranscript((s) => (s + (params?.delta ?? "")).slice(-8_192));
      }
      if (event.method === "thread/realtime/item/completed" && params?.item?.type === "transcriptSegment") {
        if (params.item.id) transcriptRoles.current.delete(params.item.id);
        if (transcriptItem.current === null || transcriptItem.current === params.item.id) {
          transcriptItem.current = null;
          setPartialTranscript(""); setPartialTranscriptRole(null);
        }
        void refreshConversationSession(sessionId);
      }
    };
    socket.onerror = () => { if (mediaRef.current) fail("Agent voice connection failed. Use Dictation or retry."); };
    socket.onclose = () => {
      if (disposed) return;
      clearTimeout(discoveryTimeout);
      setCapability((s) => ({ ...s, available: false, nativeActive: false }));
      if (mediaRef.current) fail("Agent voice disconnected. Use Dictation or retry.");
      if (readyAt !== undefined && Date.now() - readyAt >= 10_000) discoveryRetryCount.current = 0;
      const settings = useWorkspaceStore.getState();
      const delay = DISCOVERY_RETRY_DELAYS_MS[discoveryRetryCount.current];
      const willRetry = settings.voiceEnabled && settings.agentVoiceMode !== "dictation" && (!observed || observed.available || observed.retryable === true) && delay !== undefined;
      setIsChecking(willRetry);
      if (!retryTimer && willRetry) {
        discoveryRetryCount.current++;
        retryTimer = setTimeout(() => { if (!disposed) setConnectionAttempt((value) => value + 1); }, delay);
      }
    };
    const ping = setInterval(() => { if (socket.readyState === WebSocket.OPEN) socket.send(JSON.stringify({ type: "ping" })); }, 15_000);
    return () => { disposed = true; clearTimeout(discoveryTimeout); if (retryTimer) clearTimeout(retryTimer); clearInterval(ping); stop(); socket.onmessage = null; socket.onclose = null; socket.onerror = null; socket.close(); if (socketRef.current === socket) socketRef.current = null; };
  }, [sessionId, stop, transport, connectionAttempt]);

  useEffect(() => {
    if (!enabled || mode === "dictation") { stop(); setError(null); }
  }, [enabled, mode, stop]);
  useEffect(() => {
    const hidden = () => { if (document.visibilityState === "hidden") stop(); };
    const end = () => { stop(); };
    document.addEventListener("visibilitychange", hidden); window.addEventListener("pagehide", end); document.addEventListener("freeze", end);
    return () => { document.removeEventListener("visibilitychange", hidden); window.removeEventListener("pagehide", end); document.removeEventListener("freeze", end); };
  }, [stop]);

  const start = useCallback(async () => {
    if (!enabled || mediaRef.current) return;
    const socket = socketRef.current;
    if (!sessionId || !capability.available || socket?.readyState !== WebSocket.OPEN) { setError(capability.reason ?? "Agent voice is unavailable. Select Dictation to continue."); return; }
    setError(null); setVoiceState("preparing"); setMuted(false);
    const current = ++generation.current;
    const fail = (failure: Error) => { if (current === generation.current) { stop(); setError(failure.message); } };
    const media = new DuplexAudio({
      onAudio: (audio) => {
        if (socket.readyState !== WebSocket.OPEN || socket.bufferedAmount > 256 * 1024) { fail(new Error("Voice network fell behind. Reconnect to continue.")); return; }
        socket.send(JSON.stringify({ type: "audio", audio }));
      },
      onLevel: setAudioLevel,
      onConnected: () => { if (current !== generation.current) return; if (timeoutRef.current) clearTimeout(timeoutRef.current); setVoiceState("listening"); useAgentVoiceStore.getState().setConnected(sessionId, true); },
      onError: fail,
    });
    mediaRef.current = media;
    timeoutRef.current = setTimeout(() => { fail(new Error("Agent voice connection timed out. Use Dictation or retry.")); }, 35_000);
    try {
      const sdp = await media.start(transport);
      if (current !== generation.current) { media.stop(); return; }
      startedRef.current = true;
      socket.send(JSON.stringify({ type: "start", transport, sdp }));
    } catch (failure) { fail(failure instanceof Error ? failure : new Error("Microphone could not start")); }
  }, [capability, enabled, sessionId, stop, transport]);

  const toggleMute = useCallback(() => { setMuted((value) => { mediaRef.current?.mute(!value); return !value; }); }, []);
  const reconnect = useCallback(() => { discoveryRetryCount.current = 0; stop(); setIsChecking(Boolean(sessionId)); setConnectionAttempt((value) => value + 1); }, [sessionId, stop]);
  return { selected, capability, isChecking, voiceState, audioLevel, error, partialTranscript, partialTranscriptRole, muted, toggleMute, start, stop, reconnect };
}
