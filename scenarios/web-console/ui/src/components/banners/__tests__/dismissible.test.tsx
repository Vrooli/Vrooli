import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { useTranslation } from "react-i18next";
import BannerRegion from "../BannerRegion";
import { INSTANT_DAMPING } from "../damping";
import * as descriptors from "../descriptors";
import type { BannerDescriptor, MaybeBanner } from "../types";

/**
 * Every banner closes.
 *
 * Before this contract, seven of the fourteen notices had no close button —
 * not by design but by accident, because each descriptor decided for itself
 * whether to pass an `onDismiss`. A reader hit an "audio-tools is unavailable"
 * banner they could not remove.
 *
 * A banner is a non-blocking notice. One that cannot be removed is a broken
 * banner: if a condition truly must be acknowledged before work continues, it
 * wants a dialog. Dismissal is safe to make universal because the region owns
 * it — the banner stays hidden only until its condition clears, so nothing is
 * permanently silenced and a genuine recurrence is shown again.
 */

function Host({ banners }: { banners: MaybeBanner[] }) {
  return <BannerRegion banners={banners} damping={INSTANT_DAMPING} />;
}

/** One of every descriptor the app can raise, built with plausible arguments. */
function everyBanner(t: Parameters<typeof descriptors.voiceErrorBanner>[0]): BannerDescriptor[] {
  const noop = vi.fn();
  return [
    descriptors.connectionBanner(t, { retrying: false, onRetry: noop, onDismiss: noop }),
    descriptors.audioUnavailableBanner(t, { reason: "scenario_not_running" }) as BannerDescriptor,
    descriptors.createErrorBanner(t, { message: "boom", retry: true }, { onDismiss: noop, onRetry: noop }),
    descriptors.summarizeErrorBanner(
      t,
      { sessionId: "s", eventId: "e", message: "m", source: "auto", status: "failed" },
      { onRetry: noop, onDismiss: noop },
    ),
    descriptors.enableAudioBanner(t, { enabling: false, onEnable: noop, onDismiss: noop }),
    descriptors.trackingDegradedBanner(t),
    descriptors.voiceFallbackBanner(t, "fallback", noop),
    descriptors.voiceErrorBanner(t, "mic problem"),
    descriptors.voiceTranscribingBanner(t, noop),
    descriptors.voiceStaleMicBanner(t, noop),
    descriptors.ttsSpeakingBanner(t, noop),
    descriptors.voiceRejectionBanner(
      t,
      { kind: "explanatory", id: "r", reason: "why", score: 0, threshold: 1, createdAt: 0 },
      { onRetry: noop, onDismiss: noop },
    ),
    descriptors.sessionRecoveryBanner(t, { inProgress: false, total: 2, recovered: 2, adopted: 0 }),
    descriptors.crashRecoveryBanner(t, 3, noop),
  ];
}

function Collect({ index }: { index: number }) {
  const { t } = useTranslation();
  const all = everyBanner(t);
  return <Host banners={[all[index] ?? null]} />;
}

describe("every banner closes", () => {
  const count = 14;

  for (let index = 0; index < count; index += 1) {
    it(`descriptor #${String(index)} renders a close button`, () => {
      render(<Collect index={index} />);
      const banner = screen.getByTestId(/-banner$|-notice$/);
      const dismiss = screen.getByTestId(`${banner.dataset.testid ?? ""}-dismiss`);
      expect(dismiss).toBeInTheDocument();
      expect(dismiss).not.toBeDisabled();
    });
  }

  it("hides on dismiss even when the condition still holds", () => {
    const { rerender } = render(<Collect index={7} />); // voiceErrorBanner
    const banner = screen.getByTestId("voice-error-banner");
    fireEvent.click(screen.getByTestId("voice-error-banner-dismiss"));
    expect(screen.queryByTestId("voice-error-banner")).toBeNull();
    // Re-rendering with the same condition must not bring it back.
    rerender(<Collect index={7} />);
    expect(screen.queryByTestId("voice-error-banner")).toBeNull();
    expect(banner).toBeTruthy();
  });

  it("shows a later recurrence, so dismissal is per-occurrence not permanent", () => {
    const { rerender } = render(<Collect index={7} />);
    fireEvent.click(screen.getByTestId("voice-error-banner-dismiss"));
    expect(screen.queryByTestId("voice-error-banner")).toBeNull();

    // Condition clears...
    rerender(<Host banners={[]} />);
    expect(screen.queryByTestId("voice-error-banner")).toBeNull();

    // ...and happens again.
    rerender(<Collect index={7} />);
    expect(screen.getByTestId("voice-error-banner")).toBeInTheDocument();
  });
});
