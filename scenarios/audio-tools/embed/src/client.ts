// Embed Connect client factory + React context.
//
// Consumers wire one of:
//   - <AudioToolsProvider client={createAudioToolsClient({ baseUrl })}>
//   - Nothing — components fall back to a lazy singleton built from
//     window.__AUDIO_TOOLS_URL__ injected by the host scenario at boot.
//
// The host scenario is responsible for populating window.__AUDIO_TOOLS_URL__
// before React mounts (e.g. via the host's discovery endpoint).

import { createClient, type Client, type Transport } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { createContext, createElement, useContext, type ReactNode } from "react";

import { STTService } from "@vrooli/proto-types/audio-tools/v1/stt/stt_pb";
import { SummarizeService } from "@vrooli/proto-types/audio-tools/v1/summarize/summarize_pb";
import { TTSService } from "@vrooli/proto-types/audio-tools/v1/tts/tts_pb";

declare global {
  interface Window {
    __AUDIO_TOOLS_URL__?: string;
  }
}

export interface AudioToolsClient {
  stt: Client<typeof STTService>;
  tts: Client<typeof TTSService>;
  summarize: Client<typeof SummarizeService>;
  baseUrl: string;
}

export interface CreateAudioToolsClientOptions {
  /** Explicit audio-tools base URL. Falls back to window.__AUDIO_TOOLS_URL__. */
  baseUrl?: string;
  /** Inject a custom Connect transport (tests pass a mock here). */
  transport?: Transport;
}

export function createAudioToolsClient(options: CreateAudioToolsClientOptions = {}): AudioToolsClient {
  const resolvedBase = options.baseUrl ?? resolveDefaultBaseUrl();
  const transport = options.transport ?? createConnectTransport({ baseUrl: resolvedBase });
  return {
    stt: createClient(STTService, transport),
    tts: createClient(TTSService, transport),
    summarize: createClient(SummarizeService, transport),
    baseUrl: resolvedBase,
  };
}

function resolveDefaultBaseUrl(): string {
  if (typeof window !== "undefined" && typeof window.__AUDIO_TOOLS_URL__ === "string" && window.__AUDIO_TOOLS_URL__.length > 0) {
    return window.__AUDIO_TOOLS_URL__;
  }
  throw new Error(
    "@audio-tools/embed: no audio-tools base URL configured. Set window.__AUDIO_TOOLS_URL__ before mounting, or pass createAudioToolsClient({ baseUrl }) and wrap your tree in <AudioToolsProvider client={...}>."
  );
}

const AudioToolsContext = createContext<AudioToolsClient | null>(null);

export interface AudioToolsProviderProps {
  client?: AudioToolsClient;
  children: ReactNode;
}

export function AudioToolsProvider(props: AudioToolsProviderProps) {
  const value = props.client ?? defaultClient();
  return createElement(AudioToolsContext.Provider, { value }, props.children);
}

let lazyDefault: AudioToolsClient | null = null;
function defaultClient(): AudioToolsClient {
  if (lazyDefault === null) {
    lazyDefault = createAudioToolsClient();
  }
  return lazyDefault;
}

/**
 * Retrieve the active AudioToolsClient. When no <AudioToolsProvider> is
 * present in the tree, a lazy default client built from
 * window.__AUDIO_TOOLS_URL__ is returned so the zero-config case "just works".
 */
export function useAudioToolsClient(): AudioToolsClient {
  const fromContext = useContext(AudioToolsContext);
  if (fromContext !== null) {
    return fromContext;
  }
  return defaultClient();
}
