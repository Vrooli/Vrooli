import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import VoiceMicButton from "../components/VoiceMicButton";

describe("VoiceMicButton", () => {
  const onStart = vi.fn();
  const onStop = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  const defaults = {
    supported: true,
    isPreparing: false,
    isRecording: false,
    isTranscribing: false,
    error: null,
    onStart,
    onStop,
  };

  // --- Render states ---

  it("returns null when not supported", () => {
    const { container } = render(<VoiceMicButton {...defaults} supported={false} />);
    expect(container.innerHTML).toBe("");
  });

  it("renders mic icon in idle state", () => {
    render(<VoiceMicButton {...defaults} />);
    const btn = screen.getByTestId("voice-mic-btn");
    expect(btn).toBeTruthy();
    // Default border styling (not red, not blue, not amber)
    expect(btn.className).toContain("border-wc-default");
    expect(btn.className).not.toContain("border-red");
    expect(btn.className).not.toContain("border-blue");
  });

  it("shows red styling when recording", () => {
    render(<VoiceMicButton {...defaults} isRecording />);
    const btn = screen.getByTestId("voice-mic-btn");
    expect(btn.className).toContain("border-red-500");
    expect(btn.className).toContain("text-red-400");
  });

  it("shows blue styling and spinner when transcribing", () => {
    render(<VoiceMicButton {...defaults} isTranscribing />);
    const btn = screen.getByTestId("voice-mic-btn");
    expect(btn.className).toContain("border-blue-500");
    expect(btn.className).toContain("text-blue-400");
    // Spinner icon should have animate-spin class
    const spinner = btn.querySelector(".animate-spin");
    expect(spinner).toBeTruthy();
  });

  it("shows amber styling when error and idle", () => {
    render(<VoiceMicButton {...defaults} error="Test error" />);
    const btn = screen.getByTestId("voice-mic-btn");
    expect(btn.className).toContain("border-amber-500");
  });

  it("does not show error styling when recording with error", () => {
    render(<VoiceMicButton {...defaults} isRecording error="Test error" />);
    const btn = screen.getByTestId("voice-mic-btn");
    expect(btn.className).toContain("border-red-500");
    expect(btn.className).not.toContain("border-amber");
  });

  it("does not show error styling when transcribing with error", () => {
    render(<VoiceMicButton {...defaults} isTranscribing error="Test error" />);
    const btn = screen.getByTestId("voice-mic-btn");
    expect(btn.className).toContain("border-blue-500");
    expect(btn.className).not.toContain("border-amber");
  });

  it("shows audio level fill when recording", () => {
    render(<VoiceMicButton {...defaults} isRecording audioLevel={0.5} />);
    const btn = screen.getByTestId("voice-mic-btn");
    const fill = btn.querySelector("span");
    expect(fill).toBeTruthy();
    expect(fill?.style.height).toBe("50%");
  });

  it("hides audio level fill when not recording", () => {
    render(<VoiceMicButton {...defaults} audioLevel={0.5} />);
    const btn = screen.getByTestId("voice-mic-btn");
    const fill = btn.querySelector("span");
    expect(fill).toBeNull();
  });

  it("shows partial transcript when recording (after ref is attached)", () => {
    // On the first render, btnRef.current is null (React attaches refs after
    // commit), so the partial transcript div is hidden. A re-render with the
    // same props causes the ref to be available.
    const { rerender } = render(
      <VoiceMicButton {...defaults} isRecording partialTranscript="hello world" />,
    );
    rerender(<VoiceMicButton {...defaults} isRecording partialTranscript="hello world" />);
    expect(screen.getByText("hello world")).toBeTruthy();
  });

  it("hides partial transcript when not recording", () => {
    const { rerender } = render(
      <VoiceMicButton {...defaults} partialTranscript="hello world" />,
    );
    rerender(<VoiceMicButton {...defaults} partialTranscript="hello world" />);
    expect(screen.queryByText("hello world")).toBeNull();
  });

  // --- Titles ---

  it("shows recording title when recording", () => {
    render(<VoiceMicButton {...defaults} isRecording />);
    expect(screen.getByTestId("voice-mic-btn").title).toBe("Recording... tap to stop");
  });

  it("shows transcribing title when transcribing", () => {
    render(<VoiceMicButton {...defaults} isTranscribing />);
    expect(screen.getByTestId("voice-mic-btn").title).toBe("Transcribing... tap to cancel");
  });

  it("includes backend in idle title", () => {
    render(<VoiceMicButton {...defaults} backend="whisper" />);
    expect(screen.getByTestId("voice-mic-btn").title).toContain("Whisper");
  });

  // --- Pointer interaction (intent-based) ---

  it("calls onStart on pointerDown when idle", () => {
    render(<VoiceMicButton {...defaults} />);
    const btn = screen.getByTestId("voice-mic-btn");
    fireEvent.pointerDown(btn);
    expect(onStart).toHaveBeenCalledWith({ vadEnabled: true });
  });

  it("does not call onStart on pointerDown when transcribing", () => {
    render(<VoiceMicButton {...defaults} isTranscribing />);
    const btn = screen.getByTestId("voice-mic-btn");
    fireEvent.pointerDown(btn);
    expect(onStart).not.toHaveBeenCalled();
  });

  it("does not call onStart on pointerDown when already recording", () => {
    render(<VoiceMicButton {...defaults} isRecording />);
    const btn = screen.getByTestId("voice-mic-btn");
    fireEvent.pointerDown(btn);
    expect(onStart).not.toHaveBeenCalled();
  });

  it("calls onStop on pointerUp when intent is stop (was recording)", () => {
    render(<VoiceMicButton {...defaults} isRecording />);
    const btn = screen.getByTestId("voice-mic-btn");
    fireEvent.pointerDown(btn);
    fireEvent.pointerUp(btn);
    expect(onStop).toHaveBeenCalledTimes(1);
  });

  it("does not call onStop on short tap when starting (tap-to-toggle)", () => {
    render(<VoiceMicButton {...defaults} />);
    const btn = screen.getByTestId("voice-mic-btn");
    fireEvent.pointerDown(btn);
    // Immediate pointerUp — short press, should keep recording
    fireEvent.pointerUp(btn);
    expect(onStop).not.toHaveBeenCalled();
  });

  it("does not call onStop on pointerUp when intent is none (transcribing)", () => {
    render(<VoiceMicButton {...defaults} isTranscribing />);
    const btn = screen.getByTestId("voice-mic-btn");
    fireEvent.pointerDown(btn);
    fireEvent.pointerUp(btn);
    expect(onStop).not.toHaveBeenCalled();
  });

  it("clears intent on pointerCancel", () => {
    render(<VoiceMicButton {...defaults} isRecording />);
    const btn = screen.getByTestId("voice-mic-btn");
    fireEvent.pointerDown(btn); // intent = "stop"
    fireEvent.pointerCancel(btn); // clears intent via handlePointerUp
    // A second pointerUp should not trigger stop again
    fireEvent.pointerUp(btn);
    // onStop called once from pointerCancel (which calls handlePointerUp)
    expect(onStop).toHaveBeenCalledTimes(1);
  });

  // --- Preparing state ---

  it("shows yellow styling and pulsing icon when preparing", () => {
    render(<VoiceMicButton {...defaults} isPreparing />);
    const btn = screen.getByTestId("voice-mic-btn");
    expect(btn.className).toContain("border-yellow-500");
    expect(btn.className).toContain("text-yellow-400");
    const pulse = btn.querySelector(".animate-pulse");
    expect(pulse).toBeTruthy();
  });

  it("shows preparing title when preparing", () => {
    render(<VoiceMicButton {...defaults} isPreparing />);
    expect(screen.getByTestId("voice-mic-btn").title).toBe("Preparing...");
  });

  it("blocks pointer events during preparing", () => {
    render(<VoiceMicButton {...defaults} isPreparing />);
    const btn = screen.getByTestId("voice-mic-btn");
    fireEvent.pointerDown(btn);
    expect(onStart).not.toHaveBeenCalled();
    fireEvent.pointerUp(btn);
    expect(onStop).not.toHaveBeenCalled();
  });

  it("does not show error styling when preparing with error", () => {
    render(<VoiceMicButton {...defaults} isPreparing error="Test error" />);
    const btn = screen.getByTestId("voice-mic-btn");
    expect(btn.className).toContain("border-yellow-500");
    expect(btn.className).not.toContain("border-amber");
  });

  // --- Cancel transcription ---

  it("calls onCancel on tap when transcribing (after grace period)", () => {
    const onCancel = vi.fn();
    // Render as recording first, then switch to transcribing so the grace timestamp is set
    const { rerender } = render(
      <VoiceMicButton {...defaults} isRecording onCancel={onCancel} />,
    );
    rerender(<VoiceMicButton {...defaults} isTranscribing onCancel={onCancel} />);
    // Advance past grace period
    vi.useFakeTimers();
    vi.advanceTimersByTime(500);
    const btn = screen.getByTestId("voice-mic-btn");
    fireEvent.pointerDown(btn);
    fireEvent.pointerUp(btn);
    expect(onCancel).toHaveBeenCalledTimes(1);
    expect(onStop).not.toHaveBeenCalled();
    vi.useRealTimers();
  });

  it("ignores tap during transcribing grace period (race condition guard)", () => {
    const onCancel = vi.fn();
    // Start as recording, then transition to transcribing (simulates VAD auto-stop)
    const { rerender } = render(
      <VoiceMicButton {...defaults} isRecording onCancel={onCancel} />,
    );
    rerender(<VoiceMicButton {...defaults} isTranscribing onCancel={onCancel} />);
    // Tap immediately — within the grace period
    const btn = screen.getByTestId("voice-mic-btn");
    fireEvent.pointerDown(btn);
    fireEvent.pointerUp(btn);
    expect(onCancel).not.toHaveBeenCalled();
    expect(onStop).not.toHaveBeenCalled();
  });

  it("does nothing on tap when transcribing without onCancel", () => {
    render(<VoiceMicButton {...defaults} isTranscribing />);
    const btn = screen.getByTestId("voice-mic-btn");
    fireEvent.pointerDown(btn);
    fireEvent.pointerUp(btn);
    expect(onStop).not.toHaveBeenCalled();
    expect(onStart).not.toHaveBeenCalled();
  });

  it("shows cancel hint in title when transcribing", () => {
    render(<VoiceMicButton {...defaults} isTranscribing />);
    const btn = screen.getByTestId("voice-mic-btn");
    expect(btn.title).toContain("cancel");
  });
});
