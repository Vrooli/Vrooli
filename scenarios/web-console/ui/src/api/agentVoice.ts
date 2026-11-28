import { buildSessionWsUrl } from "./sessions";
import type { DuplexAudioChunk } from "../audio-integration";

export interface AgentVoiceCapability {
  available: boolean;
  retryable?: boolean;
  agent: string;
  version?: string;
  reason?: string;
  nativeActive: boolean;
}
export interface AgentVoiceEvent {
  type: string;
  capability?: AgentVoiceCapability;
  method?: string;
  message?: string;
  code?: string;
  params?: {
    sdp?: string;
    audio?: DuplexAudioChunk;
    delta?: string;
    text?: string;
    role?: string;
    itemId?: string;
    item?: { type?: string; role?: string; text?: string; id?: string };
  };
}

export function agentVoiceURL(sessionId: string): string {
  const url = new URL(buildSessionWsUrl(sessionId));
  url.pathname = url.pathname.replace(/\/ws$/, "/agent-voice");
  return url.toString();
}

export function parseAgentVoiceEvent(data: string): AgentVoiceEvent | null {
  try {
    const value: unknown = JSON.parse(data);
    if (!isRecord(value) || typeof value.type !== "string") return null;
    const event: AgentVoiceEvent = { type: value.type };
    if (typeof value.message === "string") event.message = value.message;
    if (typeof value.code === "string") event.code = value.code;
    if (typeof value.method === "string") event.method = value.method;
    if (value.type === "capability") {
      const cap = value.capability;
      if (!isRecord(cap) || typeof cap.available !== "boolean" || typeof cap.agent !== "string" || typeof cap.nativeActive !== "boolean") return null;
      event.capability = { ...(typeof cap.retryable === "boolean" ? { retryable: cap.retryable } : {}), available: cap.available, agent: cap.agent, nativeActive: cap.nativeActive, ...(typeof cap.reason === "string" ? { reason: cap.reason } : {}), ...(typeof cap.version === "string" ? { version: cap.version } : {}) };
    }
    if (isRecord(value.params)) {
      const params = value.params;
      event.params = {};
      for (const key of ["sdp", "delta", "text", "role", "itemId"] as const) if (typeof params[key] === "string") event.params[key] = params[key];
      if (isRecord(params.item)) {
        event.params.item = {};
        for (const key of ["type", "role", "text", "id"] as const) if (typeof params.item[key] === "string") event.params.item[key] = params.item[key];
      }
      if (params.audio !== undefined) {
        const audio = params.audio;
        if (!isRecord(audio) || typeof audio.data !== "string" || typeof audio.sampleRate !== "number" || typeof audio.numChannels !== "number") return null;
        event.params.audio = { data: audio.data, sampleRate: audio.sampleRate, numChannels: audio.numChannels, samplesPerChannel: typeof audio.samplesPerChannel === "number" ? audio.samplesPerChannel : null, itemId: typeof audio.itemId === "string" ? audio.itemId : null };
      }
    }
    return event;
  } catch { return null; }
}

function isRecord(value: unknown): value is Record<string, unknown> { return value !== null && typeof value === "object" && !Array.isArray(value); }
