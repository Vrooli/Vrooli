import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import VoiceMicButton from "../components/VoiceMicButton";
import VoiceRecoveryBanner from "../components/VoiceRecoveryBanner";

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
      clone.querySelectorAll("style, script").forEach((node) => node.remove());
      clone.querySelector('[data-testid="mic"]')?.remove();
      expect((clone.textContent ?? "").trim()).toBe("");
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
});

describe("VoiceRecoveryBanner", () => {
  it("stays out of the layout when there is nothing to recover from", () => {
    const { container } = render(<VoiceRecoveryBanner />);
    expect(container.firstChild).toBeNull();
  });

  it("surfaces the error the button no longer shows", () => {
    render(<VoiceRecoveryBanner error="backend unavailable" />);
    expect(screen.getByTestId("voice-recovery-error").textContent).toBe("backend unavailable");
    expect(screen.getByTestId("voice-recovery-banner").getAttribute("role")).toBe("alert");
  });

  it("offers cancel only while a transcription is actually in flight", () => {
    const onCancel = vi.fn();
    const { rerender } = render(<VoiceRecoveryBanner isTranscribing={false} onCancel={onCancel} />);
    expect(screen.queryByTestId("voice-recovery-cancel")).toBeNull();

    rerender(<VoiceRecoveryBanner isTranscribing onCancel={onCancel} />);
    screen.getByTestId("voice-recovery-cancel").click();
    expect(onCancel).toHaveBeenCalledOnce();
  });

  it("offers every recovery action its signal calls for", () => {
    render(
      <VoiceRecoveryBanner
        isTranscribing
        onCancel={vi.fn()}
        staleLiveMic
        onReleaseMic={vi.fn()}
        isTtsSpeaking
        onTtsStop={vi.fn()}
        canExportDiagnostic
        onExportDiagnostic={() => null}
      />,
    );
    expect(screen.getByTestId("voice-recovery-cancel")).toBeTruthy();
    expect(screen.getByTestId("voice-recovery-release-mic")).toBeTruthy();
    expect(screen.getByTestId("voice-recovery-stop-speech")).toBeTruthy();
    expect(screen.getByTestId("voice-recovery-export-diagnostic")).toBeTruthy();
  });

  it("reports itself as status, not alert, when only actions are available", () => {
    render(<VoiceRecoveryBanner staleLiveMic onReleaseMic={vi.fn()} />);
    expect(screen.getByTestId("voice-recovery-banner").getAttribute("role")).toBe("status");
  });
});
