// audio-integration React context.
//
// Mirrors web-console's approach: the UI talks same-origin to
// swarm-manager's own AudioAdminService + AudioRuntimeService, and the
// server owns the inter-scenario hop to audio-tools. Module-level
// Connect clients live in api/voice.ts; this file carries the
// "unavailable reason" surface so components can render a degraded
// state when audio-tools is down behind the server.
/* eslint-disable react-refresh/only-export-components */

import { createClient, type Client, type Transport } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { createContext, createElement, useContext, type ReactNode } from "react";

import { AudioAdminService } from "@vrooli/proto-types/swarm-manager/v1/audio_admin/audio_admin_pb";
import { AudioRuntimeService } from "@vrooli/proto-types/swarm-manager/v1/audio_runtime/audio_runtime_pb";

import { API_BASE } from "../lib/api-client";

export interface AudioToolsClient {
  audioAdmin: Client<typeof AudioAdminService>;
  audioRuntime: Client<typeof AudioRuntimeService>;
  baseUrl: string;
}

export interface CreateAudioToolsClientOptions {
  /** Base URL for swarm-manager's API. Defaults to same-origin. */
  baseUrl?: string;
  /** Inject a custom Connect transport (tests pass a mock here). */
  transport?: Transport;
}

export function createAudioToolsClient(options: CreateAudioToolsClientOptions = {}): AudioToolsClient {
  const baseUrl = options.baseUrl ?? API_BASE;
  const transport = options.transport ?? createConnectTransport({ baseUrl });
  return {
    audioAdmin: createClient(AudioAdminService, transport),
    audioRuntime: createClient(AudioRuntimeService, transport),
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
    return getActiveAudioToolsClient();
  }
  return fromContext.client;
}

export function useAudioToolsUnavailableReason(): string | undefined {
  const fromContext = useContext(AudioToolsContext);
  return fromContext?.unavailableReason;
}

// =============================================================================
// Module-level active client registry — used by api/*.ts modules that
// can't easily thread a client argument through every call site.
// =============================================================================

let activeClient: AudioToolsClient | null = null;

function registerActiveAudioToolsClient(client: AudioToolsClient): void {
  activeClient = client;
}

export function getActiveAudioToolsClient(): AudioToolsClient {
  if (activeClient === null) {
    activeClient = createAudioToolsClient();
  }
  return activeClient;
}

export function setActiveAudioToolsClientForTesting(client: AudioToolsClient | null): void {
  activeClient = client;
}
