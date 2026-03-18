import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
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
};

describe("MobileToolbar viewMode", () => {
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
});
