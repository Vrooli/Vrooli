import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import VoiceMicButton from "../VoiceMicButton";

describe("VoiceMicButton", () => {
  const defaultProps = {
    supported: true,
    isRecording: false,
    isTranscribing: false,
    error: null as string | null,
    onToggle: vi.fn(),
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
    render(<VoiceMicButton {...defaultProps} isRecording={true} />);
    const btn = screen.getByTestId("voice-mic-btn");
    expect(btn.className).toContain("border-red-500");
    expect(btn.querySelector(".animate-pulse")).toBeTruthy();
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

  it("calls onToggle on click", () => {
    const onToggle = vi.fn();
    render(<VoiceMicButton {...defaultProps} onToggle={onToggle} />);
    const btn = screen.getByTestId("voice-mic-btn");
    fireEvent.click(btn);
    expect(onToggle).toHaveBeenCalledOnce();
  });
});
