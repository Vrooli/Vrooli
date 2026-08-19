// Tests for the voice-rejection banner descriptor, rendered through
// BannerRegion the way Workspace renders it.

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { BannerHarness } from "../banners/__tests__/harness";
import { voiceRejectionBanner } from "../banners/descriptors";
import type { VoiceRejection } from "../../audio-integration";
import { strings } from "../../consts/strings";
import { i18n } from "../../i18n";

function retryable(overrides: Partial<Extract<VoiceRejection, { kind: "retryable" }>> = {}): VoiceRejection {
  const base: Extract<VoiceRejection, { kind: "retryable" }> = {
    kind: "retryable",
    cause: "speaker-rejected",
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

function renderRejection(
  rejection: VoiceRejection,
  handlers: { onRetry?: () => void; onDismiss?: () => void } = {},
) {
  return render(
    <BannerHarness
      build={(t) =>
        voiceRejectionBanner(t, rejection, {
          onRetry: handlers.onRetry ?? vi.fn(),
          onDismiss: handlers.onDismiss ?? vi.fn(),
        })
      }
    />,
  );
}

const RETRY = "voice-rejection-banner-retry";
const DISMISS = "voice-rejection-banner-dismiss";

describe("voice rejection banner", () => {
  it("shows Transcribe-anyway and Dismiss for retryable rejection", () => {
    renderRejection(retryable());
    expect(screen.getByTestId(RETRY)).toHaveTextContent(strings.voiceRejection.transcribeAnyway);
    expect(screen.getByTestId(DISMISS)).toBeInTheDocument();
  });

  it("renders through the shared banner base rather than bespoke markup", () => {
    renderRejection(retryable());
    const banner = screen.getByTestId("voice-rejection-banner");
    // Padding, safe-area insets and colour now come from `[data-wc-banner]` in
    // styles.css. The invariant a unit test can hold is that this notice opts
    // into that base at all — before the refactor each banner hand-rolled its
    // own `ps-[max(0.75rem,var(--wc-safe-left,0px))]` and its own palette.
    expect(banner).toHaveAttribute("data-wc-banner");
    expect(banner).toHaveAttribute("data-tone", "warning");
  });

  it("only shows Dismiss for explanatory rejection", () => {
    renderRejection(explanatory());
    expect(screen.queryByTestId(RETRY)).toBeNull();
    expect(screen.getByTestId(DISMISS)).toBeInTheDocument();
  });

  it("calls onRetry when Transcribe-anyway is clicked", () => {
    const onRetry = vi.fn();
    renderRejection(retryable(), { onRetry });
    fireEvent.click(screen.getByTestId(RETRY));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it("calls onDismiss when Dismiss is clicked", () => {
    const onDismiss = vi.fn();
    renderRejection(retryable(), { onDismiss });
    fireEvent.click(screen.getByTestId(DISMISS));
    expect(onDismiss).toHaveBeenCalledTimes(1);
  });

  it("disables retry and dismiss while retrying", () => {
    renderRejection(retryable({ status: "retrying" }));
    expect(screen.getByTestId(RETRY)).toBeDisabled();
    expect(screen.getByTestId(RETRY)).toHaveTextContent(strings.voiceRejection.transcribing);
    // Dismissing mid-retry would drop the retained audio the retry is using.
    // The button stays in place and goes disabled, so the banner does not
    // change size while the reader is looking at it.
    expect(screen.getByTestId(DISMISS)).toBeDisabled();
  });

  it("shows Retry + error text on failed status", () => {
    renderRejection(retryable({ status: "failed", errorMessage: "Network error" }));
    expect(screen.getByTestId(RETRY)).toHaveTextContent(strings.voiceRejection.retry);
    expect(screen.getByTestId("voice-rejection-banner-detail")).toHaveTextContent("Network error");
    expect(screen.getByTestId(RETRY)).not.toBeDisabled();
  });

  it("shows the empty-transcript copy and a Retry action for cause=empty-transcript", () => {
    renderRejection(retryable({ cause: "empty-transcript", score: 0, threshold: 0 }));
    expect(screen.getByTestId("voice-rejection-banner")).toHaveAttribute("data-cause", "empty-transcript");
    // Empty-transcript turns are framed as "Retry", never "Transcribe anyway".
    expect(screen.getByTestId(RETRY)).toHaveTextContent(strings.voiceRejection.retry);
    expect(screen.getByTestId(RETRY)).not.toHaveTextContent(strings.voiceRejection.transcribeAnyway);
    expect(screen.getByTestId(DISMISS)).toBeInTheDocument();
  });

  it("renders empty-transcript title/detail in en locale", async () => {
    await i18n.changeLanguage("en");
    renderRejection(retryable({ cause: "empty-transcript", durationMs: 8_200 }));
    expect(screen.getByText(/Couldn't transcribe your speech/)).toBeInTheDocument();
    expect(screen.getByText(/8\.2s kept/)).toBeInTheDocument();
  });

  // Real-translations test: cimode returns the key without interpolation, so
  // this test opts back to `en` to verify the score / threshold / duration
  // values actually thread through the interpolation tokens.
  describe("interpolation (en locale)", () => {
    beforeEach(async () => {
      await i18n.changeLanguage("en");
    });

    it("includes the score, threshold, and duration in the retryable variant", () => {
      renderRejection(retryable({ score: 0.42, threshold: 0.85, durationMs: 5_500 }));
      expect(screen.getByText(/0\.42/)).toBeInTheDocument();
      expect(screen.getByText(/0\.85/)).toBeInTheDocument();
      expect(screen.getByText(/5\.5s retained/)).toBeInTheDocument();
    });
  });
});
