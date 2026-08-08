import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { VoiceInputButton } from "./SharedVoiceInputButton";
import { renderWithProviders } from "../test-utils";

describe("adopted VoiceInputButton boundary", () => {
  it("preserves consumer labels and exposes recovery actions", () => {
    const onDismissError = vi.fn();
    const onTranscribeAnyway = vi.fn();

    renderWithProviders(
      <VoiceInputButton
        aria-label="Saisie vocale"
        state="error"
        error="Microphone busy"
        rejectionReason="Speaker mismatch"
        onDismissError={onDismissError}
        onTranscribeAnyway={onTranscribeAnyway}
      />,
    );

    expect(screen.getByRole("button", { name: "Saisie vocale" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Dismiss voice input error" }));
    fireEvent.click(screen.getByRole("button", { name: "Transcribe anyway" }));
    expect(onDismissError).toHaveBeenCalledOnce();
    expect(onTranscribeAnyway).toHaveBeenCalledOnce();
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

  it("keeps recovery, TTS, and diagnostic actions reachable", () => {
    const onCancel = vi.fn();
    const onReleaseMic = vi.fn();
    const onTtsStop = vi.fn();
    const onExportDiagnostic = vi.fn(() => null);
    const onExitPassive = vi.fn();

    const { rerender } = renderWithProviders(
      <VoiceInputButton
        state="transcribing"
        onCancel={onCancel}
        staleLiveMic
        onReleaseMic={onReleaseMic}
        isTtsSpeaking
        onTtsStop={onTtsStop}
        canExportDiagnostic
        onExportDiagnostic={onExportDiagnostic}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    fireEvent.click(screen.getByRole("button", { name: "Release mic" }));
    fireEvent.click(screen.getByRole("button", { name: "Stop speech" }));
    fireEvent.click(screen.getByRole("button", { name: "Export diagnostic" }));
    expect(onCancel).toHaveBeenCalledOnce();
    expect(onReleaseMic).toHaveBeenCalledOnce();
    expect(onTtsStop).toHaveBeenCalledOnce();
    expect(onExportDiagnostic).toHaveBeenCalledOnce();

    rerender(<VoiceInputButton state="recovering" onExitPassive={onExitPassive} />);
    const passiveButton = screen.getByRole("button", { name: "Listening for voice input" });
    fireEvent.pointerDown(passiveButton);
    fireEvent.pointerUp(passiveButton);
    expect(onExitPassive).toHaveBeenCalledOnce();
  });
});
