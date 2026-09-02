import { render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
// @ts-expect-error Vitest executes this parity guard in Node; swarm UI omits Node typings.
import { readFileSync } from "node:fs";
// @ts-expect-error Vitest executes this parity guard in Node; swarm UI omits Node typings.
import { dirname, join, resolve } from "node:path";
// @ts-expect-error Vitest executes this parity guard in Node; swarm UI omits Node typings.
import { fileURLToPath } from "node:url";
import type { CommandSuggestion, VoiceRejection } from "@vrooli/audio-capture-browser";
import { AudioPlayerBar } from "../components/AudioPlayerBar";
import { AudioUnavailableBanner } from "../components/AudioUnavailableBanner";
import { EnableAudioBanner } from "../components/EnableAudioBanner";
import { VoiceCommandSuggestion } from "../components/VoiceCommandSuggestion";
import { VoiceRejectionBanner } from "../components/VoiceRejectionBanner";

const AUDIO_ROOT = dirname(fileURLToPath(import.meta.url));
const SURFACES = ["unavailable", "enable-audio", "command-suggestion", "player", "rejection"] as const;

const suggestion: CommandSuggestion = {
  id: "suggestion-1",
  commandId: "open-session",
  description: "Open the session",
  confidence: 0.99,
  rawText: "open the session",
  timestamp: 1,
  args: {},
};

const rejection: VoiceRejection = {
  kind: "retryable",
  cause: "empty-transcript",
  id: "rejection-1",
  blob: new Blob(["audio"]),
  mimeType: "audio/webm",
  durationMs: 1000,
  score: 0,
  threshold: 0,
  createdAt: 1,
  status: "idle",
};

describe("consumer audio state parity", () => {
  it("renders every state surface in swarm-manager", () => {
    render(
      <>
        <AudioUnavailableBanner reason="scenario_not_running" />
        <EnableAudioBanner onEnable={async () => true} onDismiss={vi.fn()} />
        <VoiceCommandSuggestion suggestion={suggestion} onConfirm={vi.fn()} onDismiss={vi.fn()} />
        <AudioPlayerBar isSpeaking={false} loading onStop={vi.fn()} />
        <VoiceRejectionBanner rejection={rejection} onRetry={vi.fn()} onDismiss={vi.fn()} />
      </>,
    );
    expect([...document.querySelectorAll("[data-audio-state]")].map((node) => node.getAttribute("data-audio-state"))).toEqual(SURFACES);
  });

  it("keeps the same state markers in web-console", () => {
    const webComponents = resolve(AUDIO_ROOT, "../../../../web-console/ui/src/components");
    for (const state of SURFACES) {
      const source = {
        unavailable: {
          file: "banners/descriptors.tsx",
          marker: `"data-audio-state": "${state}"`,
        },
        "enable-audio": {
          file: "banners/descriptors.tsx",
          marker: `"data-audio-state": "${state}"`,
        },
        "command-suggestion": {
          file: "VoiceCommandSuggestion.tsx",
          marker: `data-audio-state="${state}"`,
        },
        player: {
          file: "AudioPlayerBar.tsx",
          marker: `data-audio-state="${state}"`,
        },
        rejection: {
          file: "banners/descriptors.tsx",
          marker: `"data-audio-state": "${state}"`,
        },
      }[state];
      expect(readFileSync(join(webComponents, source.file), "utf8"), source.file).toContain(source.marker);
    }
  });
});
