import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import MobileToolbar from "../components/MobileToolbar";

const baseProps = {
  onInput: vi.fn(() => true),
  onFocusTerminal: vi.fn(),
  activeSessionId: "sess-1",
  voiceSupported: true,
  voicePreparing: false,
  voiceRecording: false,
  voiceTranscribing: false,
  voiceError: null,
  voiceLevel: 0,
  voicePartialTranscript: "",
  voiceBackend: "browser",
  onVoiceStart: vi.fn(),
  onVoiceStop: vi.fn(),
  onVoiceCancel: vi.fn(),
  onUploadImage: vi.fn(),
  isTtsSpeaking: false,
  onTtsStop: vi.fn(),
  onSwitchToTerminal: vi.fn(),
};

describe("MobileToolbar viewMode", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("hides terminal-specific keys in messages mode", () => {
    render(<MobileToolbar {...baseProps} viewMode="messages" />);

    // Terminal-specific controls should not be present
    expect(screen.queryByTestId("toolbar-mod-ctrl")).toBeNull();
    expect(screen.queryByTestId("toolbar-mod-alt")).toBeNull();
    expect(screen.queryByTestId("toolbar-mod-shift")).toBeNull();
    expect(screen.queryByTestId(/toolbar-key-/)).toBeNull();
  });

  it("keeps input, send, expand, image upload, and mic in messages mode", () => {
    render(<MobileToolbar {...baseProps} viewMode="messages" />);

    expect(screen.getByTestId("mobile-command-input")).toBeInTheDocument();
    expect(screen.getByTestId("mobile-command-submit")).toBeInTheDocument();
    expect(screen.getByTestId("expand-toggle")).toBeInTheDocument();
    expect(screen.getByTestId("toolbar-upload-image")).toBeInTheDocument();
    // Mic button is present
    expect(screen.getByTestId("voice-mic-btn")).toBeInTheDocument();
  });

  it("shows full toolbar in terminal mode (default)", () => {
    render(<MobileToolbar {...baseProps} />);

    // Terminal-specific controls should be present
    expect(screen.getByTestId("toolbar-mod-ctrl")).toBeInTheDocument();
    expect(screen.getByTestId("toolbar-mod-alt")).toBeInTheDocument();
    expect(screen.getByTestId("toolbar-mod-shift")).toBeInTheDocument();
  });

  it("shows full toolbar when viewMode is explicitly terminal", () => {
    render(<MobileToolbar {...baseProps} viewMode="terminal" />);

    expect(screen.getByTestId("toolbar-mod-ctrl")).toBeInTheDocument();
  });

  // --- Feature 3: Auto-switch to terminal on send ---

  it("calls onSwitchToTerminal when submitting text in messages mode", () => {
    render(<MobileToolbar {...baseProps} viewMode="messages" />);

    const input = screen.getByTestId("mobile-command-input");
    fireEvent.change(input, { target: { value: "hello" } });
    fireEvent.click(screen.getByTestId("mobile-command-submit"));

    expect(baseProps.onInput).toHaveBeenCalledWith("hello");
    expect(baseProps.onSwitchToTerminal).toHaveBeenCalledTimes(1);
  });

  it("calls onSwitchToTerminal when submitting empty input (Enter) in messages mode", () => {
    render(<MobileToolbar {...baseProps} viewMode="messages" />);

    // Empty input — acts as Enter key
    fireEvent.click(screen.getByTestId("mobile-command-submit"));

    expect(baseProps.onSwitchToTerminal).toHaveBeenCalledTimes(1);
  });

  it("does NOT call onSwitchToTerminal when submitting in terminal mode", () => {
    render(<MobileToolbar {...baseProps} viewMode="terminal" />);

    const input = screen.getByTestId("mobile-command-input");
    fireEvent.change(input, { target: { value: "hello" } });
    fireEvent.click(screen.getByTestId("mobile-command-submit"));

    expect(baseProps.onInput).toHaveBeenCalled();
    expect(baseProps.onSwitchToTerminal).not.toHaveBeenCalled();
  });

  it("does not error when onSwitchToTerminal is undefined in messages mode", () => {
    const propsWithoutSwitch = { ...baseProps, onSwitchToTerminal: undefined };
    render(<MobileToolbar {...propsWithoutSwitch} viewMode="messages" />);

    const input = screen.getByTestId("mobile-command-input");
    fireEvent.change(input, { target: { value: "hello" } });

    // Should not throw
    expect(() => {
      fireEvent.click(screen.getByTestId("mobile-command-submit"));
    }).not.toThrow();
  });
});
