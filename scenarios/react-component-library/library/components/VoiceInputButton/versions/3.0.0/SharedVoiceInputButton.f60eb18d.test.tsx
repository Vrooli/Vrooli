import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { VoiceInputButton } from "./VoiceInputButton.tsx";
import { renderWithProviders } from "../../../../../ui/src/test-utils";

/**
 * The adopted control renders the button and nothing else.
 *
 * The previous version of this suite asserted the opposite: that an error
 * popover, a "Transcribe anyway" link, and a Cancel/Release mic/Stop speech/
 * Export diagnostic strip were all reachable from inside the button's wrapper.
 * Those branches were unreachable in this scenario — no call site supplied the
 * props that armed them — and they were the defect operators kept reporting as
 * "the button shows extra things depending on state". The suite was locking in
 * the implementation instead of the intended behaviour.
 */
describe("adopted VoiceInputButton boundary", () => {
  const STATES = ["idle", "preparing", "recording", "recovering", "transcribing", "unavailable", "error"] as const;

  it("preserves consumer labels", () => {
    renderWithProviders(<VoiceInputButton aria-label="Saisie vocale" state="error" />);
    expect(screen.getByRole("button", { name: "Saisie vocale" })).toBeInTheDocument();
  });

  for (const state of STATES) {
    it(`renders exactly one control in the ${state} state`, () => {
      const { container } = renderWithProviders(<VoiceInputButton state={state} />);
      expect(container.querySelectorAll("button")).toHaveLength(1);
    });
  }

  it("keeps status text and recovery actions out of the control", () => {
    const { container } = renderWithProviders(<VoiceInputButton state="transcribing" />);
    expect(container.querySelector('[role="group"]')).toBeNull();
    expect(screen.queryByRole("button", { name: /cancel/i })).toBeNull();
    expect(screen.queryByRole("button", { name: /release mic/i })).toBeNull();
    expect(screen.queryByRole("button", { name: /stop speech/i })).toBeNull();
    expect(screen.queryByRole("button", { name: /export diagnostic/i })).toBeNull();
    expect(screen.queryByRole("button", { name: /transcribe anyway/i })).toBeNull();
  });

  it("starts and stops through click and pointer gestures", () => {
    const onStart = vi.fn();
    const onStop = vi.fn();
    const onPointerCancel = vi.fn();
    const { rerender } = renderWithProviders(<VoiceInputButton state="idle" onStart={onStart} onStop={onStop} />);
    const idleButton = screen.getByRole("button", { name: "Start voice input" });

    fireEvent.click(idleButton);
    expect(onStart).toHaveBeenCalledOnce();
    fireEvent.pointerDown(idleButton);
    fireEvent.pointerUp(idleButton);

    rerender(<VoiceInputButton state="recording" onStart={onStart} onStop={onStop} onPointerCancel={onPointerCancel} />);
    const recordingButton = screen.getByRole("button", { name: "Stop voice input" });
    fireEvent.pointerDown(recordingButton);
    fireEvent.pointerCancel(recordingButton);
    fireEvent.click(recordingButton);
    expect(onStop).toHaveBeenCalledOnce();
    expect(onPointerCancel).toHaveBeenCalledOnce();
  });

  it("exits passive listening from the control itself", () => {
    const onExitPassive = vi.fn();
    renderWithProviders(<VoiceInputButton state="recovering" onExitPassive={onExitPassive} />);
    const passiveButton = screen.getByRole("button", { name: "Listening for voice input" });
    fireEvent.pointerDown(passiveButton);
    fireEvent.pointerUp(passiveButton);
    expect(onExitPassive).toHaveBeenCalledOnce();
  });
});
