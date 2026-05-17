import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ChatMessageBubble, type ChatMessageSpeakController } from "./ChatMessageBubble";
import type { ChatMessageView } from "./chat-types";

function makeMessage(role: ChatMessageView["role"], content = "hello"): ChatMessageView {
  return { id: `m-${role}`, role, content };
}

function makeController(overrides: Partial<ChatMessageSpeakController> = {}): ChatMessageSpeakController {
  return {
    speakingMessageId: null,
    loadingMessageId: null,
    unavailable: false,
    speak: vi.fn(),
    stop: vi.fn(),
    ...overrides,
  };
}

describe("ChatMessageBubble speak button", () => {
  it("renders Speak button only on assistant role", () => {
    const ctrl = makeController();
    const { rerender } = render(<ChatMessageBubble message={makeMessage("assistant")} speak={ctrl} />);
    expect(screen.queryByTestId("chat-bubble-speak-m-assistant")).toBeInTheDocument();

    rerender(<ChatMessageBubble message={makeMessage("user")} speak={ctrl} />);
    expect(screen.queryByTestId("chat-bubble-speak-m-user")).not.toBeInTheDocument();

    rerender(<ChatMessageBubble message={makeMessage("system")} speak={ctrl} />);
    expect(screen.queryByTestId("chat-bubble-speak-m-system")).not.toBeInTheDocument();
  });

  it("clicking Speak invokes speak(message.id, content)", () => {
    const ctrl = makeController();
    render(<ChatMessageBubble message={makeMessage("assistant", "hi there")} speak={ctrl} />);
    fireEvent.click(screen.getByTestId("chat-bubble-speak-m-assistant"));
    expect(ctrl.speak).toHaveBeenCalledWith("m-assistant", "hi there");
  });

  it("clicking again while speaking invokes stop()", () => {
    const ctrl = makeController({ speakingMessageId: "m-assistant" });
    render(<ChatMessageBubble message={makeMessage("assistant")} speak={ctrl} />);
    fireEvent.click(screen.getByTestId("chat-bubble-speak-m-assistant"));
    expect(ctrl.stop).toHaveBeenCalled();
    expect(ctrl.speak).not.toHaveBeenCalled();
  });

  it("shows a loading state while audio is being prepared", () => {
    const ctrl = makeController({ speakingMessageId: "m-assistant", loadingMessageId: "m-assistant" });
    render(<ChatMessageBubble message={makeMessage("assistant")} speak={ctrl} />);
    const button = screen.getByTestId("chat-bubble-speak-m-assistant");

    expect(button).toBeDisabled();
    expect(button.getAttribute("data-loading")).toBe("true");
    expect(screen.getByTestId("chat-bubble-audio-loading-m-assistant")).toBeInTheDocument();

    fireEvent.click(button);
    expect(ctrl.stop).not.toHaveBeenCalled();
    expect(ctrl.speak).not.toHaveBeenCalled();
  });

  it("hides the button when audio-tools is unavailable", () => {
    const ctrl = makeController({ unavailable: true });
    render(<ChatMessageBubble message={makeMessage("assistant")} speak={ctrl} />);
    expect(screen.queryByTestId("chat-bubble-speak-m-assistant")).not.toBeInTheDocument();
  });

  it("hides the button on empty content", () => {
    const ctrl = makeController();
    render(<ChatMessageBubble message={makeMessage("assistant", "   ")} speak={ctrl} />);
    expect(screen.queryByTestId("chat-bubble-speak-m-assistant")).not.toBeInTheDocument();
  });
});
