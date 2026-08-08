export interface VoiceTransport {
  buildStreamUrl(language: string, sessionId: string, resumeToken: string): string;
  transcribeRetained(blob: Blob, language: string): Promise<string>;
  getStreamConfig?(): Promise<unknown>;
}

export interface VoiceTransportStatus {
  readonly code: string;
  readonly message: string;
}

let registeredVoiceTransport: VoiceTransport | null = null;

/** Register the consuming scenario's transport adapter before the shared hook runs. */
export function registerVoiceTransport(transport: VoiceTransport): void {
  registeredVoiceTransport = transport;
}

export function getDefaultVoiceTransport(): VoiceTransport | null {
  return registeredVoiceTransport;
}

export function requireVoiceTransport(): VoiceTransport {
  if (!registeredVoiceTransport) {
    throw new Error("Voice transport is not registered. Call the scenario's registerVoiceTransport() during application bootstrap.");
  }
  return registeredVoiceTransport;
}
