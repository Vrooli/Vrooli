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
    "data-testid": testId,
  }: {
    value?: string;
    onChange?: (value?: string) => void;
    "data-testid"?: string;
  }) =>
    React.createElement("textarea", {
      "data-testid": testId ?? "monaco-editor",
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
