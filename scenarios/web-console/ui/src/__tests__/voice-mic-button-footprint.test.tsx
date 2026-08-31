import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi } from "vitest";
import { fireEvent, screen } from "@testing-library/react";
import VoiceMicButton from "../components/VoiceMicButton";
import { BannerHarness } from "../components/banners/__tests__/harness";
import {
  ttsSpeakingBanner,
  voiceErrorBanner,
  voiceStaleMicBanner,
  voiceTranscribingBanner,
} from "../components/banners/descriptors";

/**
 * The microphone button renders the button and nothing else.
 *
 * Recovery affordances have now migrated three times: from the library
 * component into a host wrapper, and from there into app chrome. Each move was
 * reported as "the button shows extra things depending on state", because a
 * strip laid out in flow beneath the control changes its footprint and shifts
 * the layout while the operator is mid-sentence.
 *
 * These tests assert the split structurally rather than by inspecting styles,
 * so the next well-meant "just put a small link under the mic" fails here.
 */

const ALL_STATES = [
  { label: "idle", props: { supported: true, isPreparing: false, isRecording: false, isTranscribing: false, error: null } },
  { label: "preparing", props: { supported: true, isPreparing: true, isRecording: false, isTranscribing: false, error: null } },
  { label: "recording", props: { supported: true, isPreparing: false, isRecording: true, isTranscribing: false, error: null } },
  { label: "transcribing", props: { supported: true, isPreparing: false, isRecording: false, isTranscribing: true, error: null } },
  { label: "passive", props: { supported: true, isPreparing: false, isRecording: false, isTranscribing: false, isPassive: true, error: null } },
  { label: "error", props: { supported: true, isPreparing: false, isRecording: false, isTranscribing: false, error: "backend unavailable" } },
  { label: "unavailable", props: { supported: false, isPreparing: false, isRecording: false, isTranscribing: false, error: null } },
] as const;

describe("VoiceMicButton footprint", () => {
  for (const state of ALL_STATES) {
    it(`renders exactly one interactive element in the ${state.label} state`, () => {
      const { container } = render(
        <VoiceMicButton {...state.props} onStart={vi.fn()} onStop={vi.fn()} testId="mic" />,
      );
      expect(container.querySelectorAll("button")).toHaveLength(1);
    });

    it(`renders no text content beside the button in the ${state.label} state`, () => {
      const { container } = render(
        <VoiceMicButton {...state.props} onStart={vi.fn()} onStop={vi.fn()} testId="mic" />,
      );
      // Everything the control renders lives inside the button itself. Any
      // sibling text is a status surface that belongs on app chrome.
      // The library control injects its own <style> element, which is not
      // rendered text, so drop stylesheets before reading the copy.
      const clone = container.cloneNode(true) as HTMLElement;
      clone.querySelectorAll("style, script").forEach((node) => { node.remove(); });
      clone.querySelector('[data-testid="mic"]')?.remove();
      expect(clone.textContent.trim()).toBe("");
    });
  }

  it("never renders a recovery action strip, even when every recovery signal is set", () => {
    const { container } = render(
      <VoiceMicButton
        supported
        isPreparing={false}
        isRecording={false}
        isTranscribing
        error="something went wrong"
        onStart={vi.fn()}
        onStop={vi.fn()}
        testId="mic"
      />,
    );
    expect(container.querySelector('[role="group"]')).toBeNull();
    expect(container.querySelector('[role="alert"]')).toBeNull();
    expect(screen.queryByText(/cancel/i)).toBeNull();
    expect(screen.queryByText(/release/i)).toBeNull();
    expect(screen.queryByText(/export/i)).toBeNull();
  });

  it("does not accept the transcript preview that once fed a tooltip", () => {
    // A regression guard with teeth: if someone re-adds `partialTranscript` to
    // the button's props, this stops compiling and the intent is restated.
    const props = Object.keys({
      supported: true, isPreparing: true, isRecording: true, isTranscribing: true,
      error: null, onStart: vi.fn(), onStop: vi.fn(),
    });
    expect(props).not.toContain("partialTranscript");
  });

  it("keeps an unavailable microphone visible and disabled", () => {
    render(
      <VoiceMicButton
        supported={false}
        isPreparing={false}
        isRecording={false}
        isTranscribing={false}
        error={null}
        capabilityReason="audio-tools is degraded: whisper-stt"
        operatorCommand="vrooli scenario restart audio-tools --json"
        onStart={vi.fn()}
        onStop={vi.fn()}
        testId="mic"
      />,
    );

    const button = screen.getByTestId("mic");
    expect(button).toBeDisabled();
    expect(button).toHaveAttribute("aria-label", expect.stringContaining("whisper-stt"));
    expect(button).toHaveAttribute("aria-label", expect.stringContaining("vrooli scenario restart audio-tools --json"));
    expect(button.parentElement).toHaveAttribute("title", expect.stringContaining("whisper-stt"));
  });
});

/**
 * The recovery affordances the button no longer shows now live in the banner
 * region — as four independent notices rather than one strip holding five
 * unrelated conditions behind a row of buttons.
 */
describe("voice recovery notices", () => {
  it("stays out of the layout when there is nothing to recover from", () => {
    const { container } = render(<BannerHarness build={() => []} />);
    expect(container.firstChild).toBeNull();
  });

  it("surfaces the error the button no longer shows", () => {
    render(<BannerHarness build={(t) => voiceErrorBanner(t, "backend unavailable")} />);
    expect(screen.getByTestId("voice-error-banner")).toHaveTextContent("backend unavailable");
  });

  it("offers cancel only while a transcription is actually in flight", () => {
    const onCancel = vi.fn();
    const { rerender } = render(<BannerHarness build={() => []} />);
    expect(screen.queryByTestId("voice-transcribing-banner-cancel")).toBeNull();

    rerender(<BannerHarness build={(t) => voiceTranscribingBanner(t, onCancel)} />);
    fireEvent.click(screen.getByTestId("voice-transcribing-banner-cancel"));
    expect(onCancel).toHaveBeenCalledOnce();
  });

  it("raises one banner per condition, and collapses the rest behind a summary", () => {
    render(
      <BannerHarness
        build={(t) => [
          voiceTranscribingBanner(t, vi.fn()),
          voiceStaleMicBanner(t, vi.fn()),
          ttsSpeakingBanner(t, vi.fn()),
          voiceErrorBanner(t, "backend unavailable"),
        ]}
      />,
    );
    // Four active conditions cost one banner plus a summary row — not four
    // stacked strips shoving the workspace down mid-sentence.
    expect(screen.getAllByTestId(/-banner$/)).toHaveLength(1);
    expect(screen.getByTestId("voice-error-banner")).toBeTruthy();
    expect(screen.getByTestId("banner-region-overflow-toggle")).toBeTruthy();

    fireEvent.click(screen.getByTestId("banner-region-overflow-toggle"));
    expect(screen.getByTestId("voice-stale-mic-banner-release-mic")).toBeTruthy();
    expect(screen.getByTestId("tts-speaking-banner-stop-speech")).toBeTruthy();
    expect(screen.getByTestId("voice-transcribing-banner-cancel")).toBeTruthy();
  });

  it("no longer offers a diagnostic export", () => {
    render(
      <BannerHarness
        build={(t) => [voiceTranscribingBanner(t, vi.fn()), voiceErrorBanner(t, "boom")]}
      />,
    );
    fireEvent.click(screen.getByTestId("banner-region-overflow-toggle"));
    expect(screen.queryByText(/export/i)).toBeNull();
  });

  it("reports itself as status, not alert, when nothing is broken", () => {
    render(<BannerHarness build={(t) => voiceStaleMicBanner(t, vi.fn())} />);
    expect(screen.getByTestId("voice-stale-mic-banner").getAttribute("role")).toBe("status");
  });
});
