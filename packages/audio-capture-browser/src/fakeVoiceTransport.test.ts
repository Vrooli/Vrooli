import { vi } from "vitest";
import type { VoiceAudioChunk, VoiceStreamConnection, VoiceStreamHandlers, VoiceStreamStart, VoiceTransport } from "./transport";
import type { StreamMessage } from "./streamMessages";

/** Typed in-memory transport for provider behavior tests. */
export class FakeVoiceTransport implements VoiceTransport {
  static OPEN = 1;
  static CLOSED = 3;
  static CONNECTING = 0;
  static instances: FakeVoiceConnection[] = [];
  static autoOpen = true;

  readonly transcribeRetainedWithIdentity?: NonNullable<VoiceTransport["transcribeRetainedWithIdentity"]>;

  constructor(
    private readonly retained = async (_blob: Blob, _language: string) => "",
    retainedWithIdentity?: NonNullable<VoiceTransport["transcribeRetainedWithIdentity"]>,
  ) {
    if (retainedWithIdentity) this.transcribeRetainedWithIdentity = retainedWithIdentity;
  }

  openStream(start: VoiceStreamStart, handlers: VoiceStreamHandlers): VoiceStreamConnection {
    const connection = new FakeVoiceConnection(start, handlers);
    FakeVoiceTransport.instances.push(connection);
    if (FakeVoiceTransport.autoOpen) queueMicrotask(() => connection.onopen?.());
    return connection;
  }

  transcribeRetained(blob: Blob, language: string): Promise<string> {
    return this.retained(blob, language);
  }

}

/** Keep wire hashing deterministic so journal tests do not race browser WebCrypto's task queue. */
export function stubCryptoWithoutSubtle(): void {
  const crypto = globalThis.crypto;
  vi.stubGlobal("crypto", {
    randomUUID: crypto.randomUUID?.bind(crypto),
    getRandomValues: crypto.getRandomValues.bind(crypto),
  } as unknown as Crypto);
}

export class FakeVoiceConnection implements VoiceStreamConnection {
  static OPEN = FakeVoiceTransport.OPEN;
  static CLOSED = FakeVoiceTransport.CLOSED;
  static CONNECTING = FakeVoiceTransport.CONNECTING;
  readyState = FakeVoiceConnection.CONNECTING;
  readonly send = vi.fn((_value: VoiceAudioChunk | { type: "end" }) => {});
  readonly sendAudio = vi.fn((chunk: VoiceAudioChunk) => {
    this.send(chunk);
  });
  readonly end = vi.fn(() => { this.send({ type: "end" }); });
  readonly close = vi.fn(() => { this.readyState = FakeVoiceConnection.CLOSED; });
  onopen: (() => void) | null;
  onclose: (() => void) | null;
  onmessage: ((event: { data: string }) => void) | null;

  constructor(readonly start: VoiceStreamStart, private readonly handlers: VoiceStreamHandlers) {
    this.onopen = () => {
      this.readyState = FakeVoiceConnection.OPEN;
      this.handlers.onReady();
    };
    this.onclose = () => {
      this.readyState = FakeVoiceConnection.CLOSED;
      this.handlers.onClose();
    };
    this.onmessage = (event) => {
      try {
        this.handlers.onEvent(JSON.parse(event.data) as StreamMessage);
      } catch {
        // Malformed fake frames match the production decoder's ignore behavior.
      }
    };
  }

  open(): void { this.onopen?.(); }
  isOpen(): boolean { return this.readyState === FakeVoiceConnection.OPEN; }
}
