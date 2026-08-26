import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent } from "@testing-library/react";
import MobileToolbar from "../components/MobileToolbar";

const baseProps = {
  onInput: vi.fn(() => ({ status: "sent" as const, offset: 1 })),
  onFocusTerminal: vi.fn(),
  activeSessionId: "sess-1",
  voice: {
    supported: true,
    preparing: false,
    recording: false,
    transcribing: false,
    error: null,
    level: 0,
    partialTranscript: "",
    backend: "browser",
    onStart: vi.fn(),
    onStop: vi.fn(),
  },
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

  it("keeps input, send, image upload, and mic in messages mode", () => {
    render(<MobileToolbar {...baseProps} viewMode="messages" />);

    expect(screen.getByTestId("mobile-command-input")).toBeInTheDocument();
    expect(screen.getByTestId("mobile-command-submit")).toBeInTheDocument();
    expect(screen.queryByTestId("expand-toggle")).toBeNull();
    expect(screen.getByTestId("toolbar-upload-image")).toBeInTheDocument();
    // Mic button is present
    expect(screen.getByTestId("voice-mic-btn")).toHaveAttribute("data-control-size", "sm");
  });

  it("spreads messages-mode action buttons evenly across the row", () => {
    render(<MobileToolbar {...baseProps} onOpenAi={vi.fn()} viewMode="messages" />);

    const row = screen.getByTestId("messages-toolbar-actions");
    const ai = screen.getByTestId("toolbar-ai");
    const upload = screen.getByTestId("toolbar-upload-image");
    const mic = screen.getByTestId("voice-mic-btn");

    expect(row).toHaveClass("items-stretch");
    expect(ai).toHaveClass("flex-1");
    expect(upload).toHaveClass("flex-1");
    expect(mic.parentElement).toHaveClass("flex-1");
    expect(mic).toHaveClass("w-full");
    expect(mic).toHaveAttribute("data-control-size", "sm");
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

    expect(baseProps.onInput).toHaveBeenCalledWith("hello", "bulk_text");
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

  // --- Full-screen composer entry (corner expand icon) ---

  it("shows the corner expand icon in terminal mode and opens the composer", () => {
    const onExpandComposer = vi.fn();
    render(<MobileToolbar {...baseProps} onExpandComposer={onExpandComposer} viewMode="terminal" />);
    const expand = screen.getByTestId("expand-toggle");
    expect(expand).toBeInTheDocument();
    fireEvent.click(expand);
    expect(onExpandComposer).toHaveBeenCalledTimes(1);
  });

  it("shows the corner expand icon in messages mode and opens the composer", () => {
    const onExpandComposer = vi.fn();
    render(<MobileToolbar {...baseProps} onExpandComposer={onExpandComposer} viewMode="messages" />);
    const expand = screen.getByTestId("expand-toggle");
    expect(expand).toBeInTheDocument();
    fireEvent.click(expand);
    expect(onExpandComposer).toHaveBeenCalledTimes(1);
  });

  it("does not render the expand icon when onExpandComposer is not provided", () => {
    render(<MobileToolbar {...baseProps} viewMode="terminal" />);
    expect(screen.queryByTestId("expand-toggle")).toBeNull();
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

  it("shows voice command suggestions and routes confirm or dismiss", () => {
    const onCommandConfirm = vi.fn();
    const onCommandDismiss = vi.fn();
    const suggestion = { id: "s1", commandId: "list-files", description: "List files", confidence: 0.9, rawText: "list files", timestamp: 1, args: {} };
    const first = render(<MobileToolbar {...baseProps} voice={{ ...baseProps.voice, commandSuggestion: suggestion, onCommandConfirm, onCommandDismiss }} />);
    expect(screen.getByTestId("voice-command-suggestion")).toHaveTextContent("List files");
    fireEvent.click(screen.getByTestId("voice-command-confirm"));
    expect(onCommandConfirm).toHaveBeenCalledWith(suggestion);

    first.unmount();
    render(<MobileToolbar {...baseProps} voice={{ ...baseProps.voice, commandSuggestion: suggestion, onCommandConfirm, onCommandDismiss }} />);
    fireEvent.click(screen.getByTestId("voice-command-dismiss"));
    expect(onCommandDismiss).toHaveBeenCalledWith(suggestion);
  });
});
