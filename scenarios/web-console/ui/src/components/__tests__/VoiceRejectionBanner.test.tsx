// Tests for VoiceRejectionBanner.

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import VoiceRejectionBanner from "../VoiceRejectionBanner";
import type { VoiceRejection } from "../../hooks/voice/types";
import { strings } from "../../consts/strings";
import { i18n } from "../../i18n";

function retryable(overrides: Partial<Extract<VoiceRejection, { kind: "retryable" }>> = {}): VoiceRejection {
  const base: Extract<VoiceRejection, { kind: "retryable" }> = {
    kind: "retryable",
    id: "rej-1",
    blob: new Blob([new Uint8Array(100)], { type: "audio/webm" }),
    mimeType: "audio/webm",
    durationMs: 5_500,
    score: 0.42,
    threshold: 0.85,
    createdAt: Date.now(),
    status: "idle",
  };
  return { ...base, ...overrides };
}

function explanatory(): VoiceRejection {
  return {
    kind: "explanatory",
    id: "rej-2",
    reason: "This provider does not retain audio; please record again to retry.",
    score: 0.3,
    threshold: 0.85,
    createdAt: Date.now(),
  };
}

describe("VoiceRejectionBanner", () => {
  it("shows Transcribe-anyway and Dismiss for retryable rejection", () => {
    render(
      <VoiceRejectionBanner
        rejection={retryable()}
        onRetry={vi.fn()}
        onDismiss={vi.fn()}
      />,
    );
    expect(screen.getByTestId("voice-rejection-retry")).toHaveTextContent(strings.voiceRejection.transcribeAnyway);
    expect(screen.getByTestId("voice-rejection-dismiss")).toBeInTheDocument();
  });

  it("only shows Dismiss for explanatory rejection", () => {
    render(
      <VoiceRejectionBanner
        rejection={explanatory()}
        onRetry={vi.fn()}
        onDismiss={vi.fn()}
      />,
    );
    expect(screen.queryByTestId("voice-rejection-retry")).toBeNull();
    expect(screen.getByTestId("voice-rejection-dismiss")).toBeInTheDocument();
  });

  it("calls onRetry when Transcribe-anyway is clicked", () => {
    const onRetry = vi.fn();
    render(
      <VoiceRejectionBanner
        rejection={retryable()}
        onRetry={onRetry}
        onDismiss={vi.fn()}
      />,
    );
    fireEvent.click(screen.getByTestId("voice-rejection-retry"));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it("calls onDismiss when Dismiss is clicked", () => {
    const onDismiss = vi.fn();
    render(
      <VoiceRejectionBanner
        rejection={retryable()}
        onRetry={vi.fn()}
        onDismiss={onDismiss}
      />,
    );
    fireEvent.click(screen.getByTestId("voice-rejection-dismiss"));
    expect(onDismiss).toHaveBeenCalledTimes(1);
  });

  it("disables both buttons while retrying", () => {
    render(
      <VoiceRejectionBanner
        rejection={retryable({ status: "retrying" })}
        onRetry={vi.fn()}
        onDismiss={vi.fn()}
      />,
    );
    expect(screen.getByTestId("voice-rejection-retry")).toBeDisabled();
    expect(screen.getByTestId("voice-rejection-dismiss")).toBeDisabled();
    expect(screen.getByTestId("voice-rejection-retry")).toHaveTextContent(strings.voiceRejection.transcribing);
  });

  it("shows Retry + error text on failed status", () => {
    render(
      <VoiceRejectionBanner
        rejection={retryable({ status: "failed", errorMessage: "Network error" })}
        onRetry={vi.fn()}
        onDismiss={vi.fn()}
      />,
    );
    expect(screen.getByTestId("voice-rejection-retry")).toHaveTextContent(strings.voiceRejection.retry);
    expect(screen.getByTestId("voice-rejection-error")).toHaveTextContent("Network error");
    expect(screen.getByTestId("voice-rejection-retry")).not.toBeDisabled();
  });

  // Real-translations test: cimode returns the key without interpolation, so
  // this test opts back to `en` to verify the score / threshold / duration
  // values actually thread through the interpolation tokens.
  describe("interpolation (en locale)", () => {
    beforeEach(async () => {
      await i18n.changeLanguage("en");
    });

    it("includes the score, threshold, and duration in the retryable variant", () => {
      render(
        <VoiceRejectionBanner
          rejection={retryable({ score: 0.42, threshold: 0.85, durationMs: 5_500 })}
          onRetry={vi.fn()}
          onDismiss={vi.fn()}
        />,
      );
      expect(screen.getByText(/0\.42/)).toBeInTheDocument();
      expect(screen.getByText(/0\.85/)).toBeInTheDocument();
      expect(screen.getByText(/5\.5s retained/)).toBeInTheDocument();
    });
  });
});
