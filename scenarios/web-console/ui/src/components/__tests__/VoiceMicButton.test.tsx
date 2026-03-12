import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import VoiceMicButton from "../VoiceMicButton";

describe("VoiceMicButton", () => {
  const defaultProps = {
    supported: true,
    isRecording: false,
    isTranscribing: false,
    error: null as string | null,
    onStart: vi.fn(),
    onStop: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders nothing when not supported", () => {
    const { container } = render(
      <VoiceMicButton {...defaultProps} supported={false} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders mic icon in idle state", () => {
    render(<VoiceMicButton {...defaultProps} />);
    const btn = screen.getByTestId("voice-mic-btn");
    expect(btn).toBeTruthy();
    expect(btn.className).not.toContain("border-red-500");
    expect(btn.querySelector(".animate-spin")).toBeNull();
  });

  it("shows recording state", () => {
    render(<VoiceMicButton {...defaultProps} isRecording={true} audioLevel={0.5} />);
    const btn = screen.getByTestId("voice-mic-btn");
    expect(btn.className).toContain("border-red-500");
    // Audio level fill bar should be present
    const fill = btn.querySelector("span[style]");
    expect(fill).toBeTruthy();
    expect((fill as HTMLElement).style.height).toBe("50%");
  });

  it("shows transcribing state", () => {
    render(<VoiceMicButton {...defaultProps} isTranscribing={true} />);
    const btn = screen.getByTestId("voice-mic-btn");
    expect(btn.querySelector(".animate-spin")).toBeTruthy();
  });

  it("shows error state", () => {
    render(<VoiceMicButton {...defaultProps} error="Mic permission denied" />);
    const btn = screen.getByTestId("voice-mic-btn");
    expect(btn.className).toContain("border-amber-500");
    expect(btn.title).toContain("Mic permission denied");
  });

  it("calls onStart on pointer down when idle", () => {
    const onStart = vi.fn();
    render(<VoiceMicButton {...defaultProps} onStart={onStart} />);
    const btn = screen.getByTestId("voice-mic-btn");
    fireEvent.pointerDown(btn);
    expect(onStart).toHaveBeenCalledOnce();
  });

  it("calls onStop on pointer up when already recording (tap-to-stop)", () => {
    const onStop = vi.fn();
    render(<VoiceMicButton {...defaultProps} isRecording={true} onStop={onStop} />);
    const btn = screen.getByTestId("voice-mic-btn");
    // Simulate press on already-recording button
    fireEvent.pointerDown(btn);
    fireEvent.pointerUp(btn);
    expect(onStop).toHaveBeenCalledOnce();
  });
});
