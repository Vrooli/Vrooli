import "@testing-library/jest-dom";
import React from "react";
import { vi } from "vitest";

// jsdom has no Web Audio API; audio-integration's sharedAudioContext.ts
// installs a focus/click handler that tries `new AudioContext()` to
// satisfy autoplay policy. Stub a minimal no-op so any consumer that
// pulls in audio-integration (MicButton in MessageComposer, etc.)
// doesn't blow up tests that aren't exercising the audio path.
if (typeof (globalThis as { AudioContext?: unknown }).AudioContext === "undefined") {
  class StubAudioContext {
    state = "running";
    createAnalyser() { return { fftSize: 0, getByteTimeDomainData: () => undefined, disconnect: () => undefined }; }
    createMediaStreamSource() { return { connect: () => undefined, disconnect: () => undefined }; }
    createGain() { return { gain: { value: 0 }, connect: () => undefined, disconnect: () => undefined }; }
    resume() { return Promise.resolve(); }
    close() { return Promise.resolve(); }
  }
  (globalThis as { AudioContext?: unknown }).AudioContext = StubAudioContext;
}

// jsdom's WebSocket opens REAL network connections. The voice stack
// (useVoiceCore mounts → PcmVoiceStreamProvider.preConnect) constructs a
// WebSocket whenever the default capability check reports Whisper healthy,
// which is the hardcoded default. Any test that renders the message composer
// (directly or via a page) therefore opened a real socket to the voice-stream
// URL; the pending connections held the event loop open and hung the whole
// run at process exit. Stub a minimal no-op socket: it never touches the
// network, stays CONNECTING, and close() flips it to CLOSED so provider
// cleanup (dispose → ws.close()) behaves. Tests that genuinely exercise a
// WebSocket install their own fake in beforeEach, overriding this.
class StubWebSocket {
  static readonly CONNECTING = 0;
  static readonly OPEN = 1;
  static readonly CLOSING = 2;
  static readonly CLOSED = 3;
  readonly CONNECTING = 0;
  readonly OPEN = 1;
  readonly CLOSING = 2;
  readonly CLOSED = 3;
  readyState = StubWebSocket.CONNECTING;
  url: string;
  onopen: ((ev: unknown) => void) | null = null;
  onclose: ((ev: unknown) => void) | null = null;
  onerror: ((ev: unknown) => void) | null = null;
  onmessage: ((ev: unknown) => void) | null = null;
  constructor(url: string | URL) {
    this.url = String(url);
  }
  send() {}
  close() {
    this.readyState = StubWebSocket.CLOSED;
    this.onclose?.({});
  }
  addEventListener() {}
  removeEventListener() {}
}
(globalThis as { WebSocket?: unknown }).WebSocket = StubWebSocket;

// jsdom does not implement HTMLMediaElement playback. The TTS stack
// (KokoroProvider) creates a reusable HTMLAudioElement and calls
// play()/pause()/load() during unlock and on dispose; jsdom throws
// "Not implemented", which surfaced (once the WebSocket hang was removed) as
// an unhandled error that crashed the test worker. Stub them as no-ops.
if (typeof window !== "undefined" && typeof window.HTMLMediaElement !== "undefined") {
  window.HTMLMediaElement.prototype.play = function play() {
    return Promise.resolve();
  };
  window.HTMLMediaElement.prototype.pause = function pause() {};
  window.HTMLMediaElement.prototype.load = function load() {};
}

const originalConsoleLog = console.log.bind(console);
console.log = (...args: unknown[]) => {
  if (typeof args[0] === "string" && args[0].startsWith("[api-base]")) {
    return;
  }
  originalConsoleLog(...args);
};

vi.mock("@monaco-editor/react", () => ({
  __esModule: true,
  default: ({
    value = "",
    onChange,
    options,
    "data-testid": testId,
  }: {
    value?: string;
    onChange?: (value?: string) => void;
    options?: { readOnly?: boolean };
    "data-testid"?: string;
  }) =>
    React.createElement("textarea", {
      "data-testid": testId ?? "monaco-editor",
      // Surfaced so tests can assert view-vs-edit separately: a read-only file
      // must still render its content.
      "data-read-only": options?.readOnly ? "true" : "false",
      readOnly: Boolean(options?.readOnly),
      value,
      onChange: (event: React.ChangeEvent<HTMLTextAreaElement>) =>
        onChange?.(event.target.value),
    }),
  DiffEditor: ({
    original,
    modified,
    "data-testid": testId,
  }: {
    original?: string;
    modified?: string;
    "data-testid"?: string;
  }) =>
    React.createElement("div", {
      "data-testid": testId ?? "monaco-diff-editor",
      "data-original": original ?? "",
      "data-modified": modified ?? "",
    }),
}));
