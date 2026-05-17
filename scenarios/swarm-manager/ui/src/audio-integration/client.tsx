// audio-integration Connect client factory + React context.
//
// This folder is the canonical copy-paste reference for adopters of
// audio-tools. Consumers must construct a client explicitly and mount it
// via <AudioToolsProvider client={...}>; there are no window globals and
// no zero-config fallback.
//
// react-refresh/only-export-components is disabled file-wide: this module
// intentionally co-locates the provider component with the create-client
// factory, the context hooks, and a module-level active-client registry
// used by sibling api/*.ts modules. Splitting them into separate files
// would force every adopter (this is the canonical copy-paste reference)
// to wire two imports for one capability. HMR for this file is
// acceptable to break; consumers wrap their tree once at boot.
/* eslint-disable react-refresh/only-export-components */

import { createClient, type Client, type Transport } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { resolveApiBase } from "@vrooli/api-base";
import { createContext, createElement, useContext, type ReactNode } from "react";

import { STTService } from "@vrooli/proto-types/audio-tools/v1/stt/stt_pb";
import { STTAdminService } from "@vrooli/proto-types/audio-tools/v1/stt/stt_admin_pb";
import { SummarizeService } from "@vrooli/proto-types/audio-tools/v1/summarize/summarize_pb";
import { TTSService } from "@vrooli/proto-types/audio-tools/v1/tts/tts_pb";

export interface AudioToolsClient {
  stt: Client<typeof STTService>;
  sttAdmin: Client<typeof STTAdminService>;
  tts: Client<typeof TTSService>;
  summarize: Client<typeof SummarizeService>;
  baseUrl: string;
}

export interface CreateAudioToolsClientOptions {
  /** Required audio-tools base URL (e.g. http://localhost:PORT). */
  baseUrl: string;
  /** Inject a custom Connect transport (tests pass a mock here). */
  transport?: Transport;
}

export function createAudioToolsClient(options: CreateAudioToolsClientOptions): AudioToolsClient {
  const baseUrl = options.baseUrl;
  const transport = options.transport ?? createConnectTransport({ baseUrl });
  return {
    stt: createClient(STTService, transport),
    sttAdmin: createClient(STTAdminService, transport),
    tts: createClient(TTSService, transport),
    summarize: createClient(SummarizeService, transport),
    baseUrl,
  };
}

interface AudioToolsContextValue {
  client: AudioToolsClient;
  unavailableReason?: string;
}

const AudioToolsContext = createContext<AudioToolsContextValue | null>(null);

export interface AudioToolsProviderProps {
  client: AudioToolsClient;
  unavailableReason?: string;
  children: ReactNode;
}

export function AudioToolsProvider(props: AudioToolsProviderProps) {
  registerActiveAudioToolsClient(props.client);
  const value: AudioToolsContextValue = {
    client: props.client,
    unavailableReason: props.unavailableReason,
  };
  return createElement(AudioToolsContext.Provider, { value }, props.children);
}

export function useAudioToolsClient(): AudioToolsClient {
  const fromContext = useContext(AudioToolsContext);
  if (fromContext === null) {
    throw new Error(
      "useAudioToolsClient must be used inside <AudioToolsProvider>. " +
        "Construct a client with createAudioToolsClient({ baseUrl }) at boot.",
    );
  }
  return fromContext.client;
}

export function useAudioToolsUnavailableReason(): string | undefined {
  const fromContext = useContext(AudioToolsContext);
  return fromContext?.unavailableReason;
}

// =============================================================================
// Module-level active client registry
//
// The standalone helpers in api/voice.ts and api/tts.ts (e.g.
// `synthesizeTTS`, `getVoiceStreamConfig`) bind to whichever client is
// currently registered by <AudioToolsProvider>. This lets call sites that
// are not React components — or that pre-date a refactor to hook-style use
// — keep working without threading a client argument everywhere.
//
// Tests can call setActiveAudioToolsClientForTesting() directly.
// =============================================================================

let activeClient: AudioToolsClient | null = null;

function sameOriginBaseUrl(): string {
  return resolveApiBase();
}

function registerActiveAudioToolsClient(client: AudioToolsClient): void {
  activeClient = client;
}

export function getActiveAudioToolsClient(): AudioToolsClient {
  if (activeClient === null) {
    // Fall back to a same-origin client so module-load and pre-provider call
    // sites (e.g. tests that don't mount AudioToolsProvider) don't crash.
    // Production code should always have a real provider mounted at boot.
    activeClient = createAudioToolsClient({ baseUrl: sameOriginBaseUrl() });
  }
  return activeClient;
}

export function setActiveAudioToolsClientForTesting(client: AudioToolsClient | null): void {
  activeClient = client;
}
