import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi } from "vitest";
import { act, fireEvent, screen } from "@testing-library/react";
import { useState } from "react";
import FullScreenComposer from "../components/FullScreenComposer";
import { useComposerDraft } from "../hooks/useComposerDraft";
import type { GateResult } from "../components/terminal/inputGate";

/**
 * Live dictation text must reach the input the operator is looking at.
 *
 * This regressed silently once already: `partialTranscript` was threaded from
 * Workspace through the toolbars into VoiceMicButton, which declared the prop
 * and never read it, so partials were computed, transmitted, and dropped. No
 * test failed, because no test asserted the text was on screen. These do.
 */

interface HarnessProps {
  readonly interim?: string;
  readonly role?: "user" | "assistant";
  readonly endVoice?: () => void;
  readonly liveAgent?: boolean;
  readonly initialOpen?: boolean;
}

function Harness({ interim = "", initialOpen = true, role, liveAgent, endVoice }: HarnessProps) {
  const draft = useComposerDraft("sess-interim");
  const [open] = useState(initialOpen);
  return (
    <FullScreenComposer
      open={open}
      onClose={vi.fn()}
      draft={draft}
      onInput={((): GateResult => ({ status: "sent", offset: 1 })) as never}
      onFocusTerminal={vi.fn()}
      interimTranscript={interim}
      interimTranscriptRole={role}
      agentVoiceActive={liveAgent}
      onEndAgentVoice={endVoice}
    />
  );
}

describe("composer interim transcript", () => {
  it("shows no mirror when nothing is being dictated", () => {
    render(<Harness />);
    expect(screen.queryByTestId("composer-interim-overlay")).toBeNull();
  });

  it("renders the live hypothesis in the input while dictating", () => {
    render(<Harness interim="the quick brown fox" />);
    expect(screen.getByTestId("composer-interim-overlay-text").textContent).toContain("the quick brown fox");
  });

  it("draws settled draft text and the hypothesis in one mirror, in order", () => {
    const { rerender } = render(<Harness interim="" />);
    const textarea = screen.getByTestId("composer-input") as HTMLTextAreaElement;
    act(() => {
      fireEvent.change(textarea, { target: { value: "already typed" } });
    });
    rerender(<Harness interim="and spoken" />);

    const overlay = screen.getByTestId("composer-interim-overlay");
    expect(overlay.textContent).toBe("already typed and spoken");
  });

  it("attaches punctuation to the preceding word instead of spacing it off", () => {
    const { rerender } = render(<Harness interim="" />);
    const textarea = screen.getByTestId("composer-input") as HTMLTextAreaElement;
    act(() => {
      fireEvent.change(textarea, { target: { value: "hello" } });
    });
    rerender(<Harness interim=", world" />);

    expect(screen.getByTestId("composer-interim-overlay").textContent).toBe("hello, world");
  });

  it("hides the textarea's own glyphs while the mirror is drawing them", () => {
    const { rerender } = render(<Harness interim="" />);
    const textarea = screen.getByTestId("composer-input") as HTMLTextAreaElement;
    expect(textarea.className).toContain("text-wc-text-primary");
    expect(textarea.className).not.toContain("text-transparent");
    const placeholder = textarea.placeholder;
    expect(placeholder).not.toBe("");

    rerender(<Harness interim="speaking now" />);
    expect(textarea.className).toContain("text-transparent");
    expect(textarea.placeholder).toBe("");
    rerender(<Harness interim="" />);
    expect(textarea.placeholder).toBe(placeholder);
  });

  it("uses a different preview tone for assistant speech", () => {
    const { rerender } = render(<Harness interim="Question" role="user" />);
    expect(screen.getByTestId("composer-interim-overlay-text")).toHaveClass("text-wc-accent");
    rerender(<Harness interim="Answer" role="assistant" />);
    expect(screen.getByTestId("composer-interim-overlay-text")).toHaveClass("text-wc-text-secondary");
    expect(screen.getByTestId("composer-input")).toHaveValue("");
  });
  it("keeps the mirror out of the accessibility tree", () => {
    render(<Harness interim="spoken words" />);
    expect(screen.getByTestId("composer-interim-overlay").getAttribute("aria-hidden")).toBe("true");
  });

  it("identifies live agent conversation separately and restores the unsent draft", () => {
    const view = render(<Harness />);
    const textarea = screen.getByTestId("composer-input");
    fireEvent.change(textarea, { target: { value: "Unsent typed draft" } });
    view.rerender(<Harness liveAgent interim="A spoken answer" role="assistant" />);
    expect(screen.getByTestId("composer-interim-overlay")).toHaveAttribute("role", "region");
    expect(screen.getByTestId("composer-interim-overlay")).toHaveTextContent("agentVoice.liveAutomatic");
    expect(screen.getByTestId("composer-interim-overlay")).not.toHaveTextContent("Unsent typed draft");
    expect(textarea).toHaveValue("Unsent typed draft");
    expect(textarea).toHaveAttribute("readonly");
    view.rerender(<Harness />);
    expect(textarea).not.toHaveAttribute("readonly");
    expect(textarea).toHaveValue("Unsent typed draft");
  });
  it("uses the send location to end live voice without submitting the preserved draft", () => {
    const endVoice = vi.fn();
    const view = render(<Harness />);
    fireEvent.change(screen.getByTestId("composer-input"), { target: { value: "Unsent draft" } });
    view.rerender(<Harness liveAgent endVoice={endVoice} />);
    expect(screen.getByTestId("composer-send")).toHaveTextContent("agentVoice.end");
    fireEvent.click(screen.getByTestId("composer-send"));
    expect(endVoice).toHaveBeenCalledOnce();
    expect(screen.getByTestId("composer-input")).toHaveValue("Unsent draft");
  });
  it("follows long native captions using their own scroll geometry and respects reading back", () => {
    const view = render(<Harness liveAgent interim="First two lines" role="assistant" />);
    const scroller = screen.getByTestId("agent-voice-caption-scroll");
    let height = 200;
    Object.defineProperties(scroller, { scrollHeight: { configurable: true, get: () => height }, clientHeight: { configurable: true, value: 40 } });
    const longReply = "A complete spoken response. ".repeat(25);
    view.rerender(<Harness liveAgent interim={longReply} role="assistant" />);
    expect(scroller.scrollTop).toBe(200);
    expect(scroller).toHaveTextContent(longReply.trim());
    scroller.scrollTop = 20; fireEvent.scroll(scroller);
    height = 400;
    view.rerender(<Harness liveAgent interim={`${longReply} More words.`} role="assistant" />);
    expect(scroller.scrollTop).toBe(20);
    scroller.scrollTop = 360; fireEvent.scroll(scroller);
    height = 500;
    view.rerender(<Harness liveAgent interim={`${longReply} More words. End.`} role="assistant" />);
    expect(scroller.scrollTop).toBe(500);
    expect(screen.getByTestId("composer-input")).toHaveValue("");
  });
  it("stops previewing once the segment settles into the draft", () => {
    const { rerender } = render(<Harness interim="settling text" />);
    expect(screen.getByTestId("composer-interim-overlay")).toBeTruthy();

    // A segment-final arrives: the host appends it to the draft and
    // TranscriptBuffer clears the hypothesis it superseded.
    const textarea = screen.getByTestId("composer-input") as HTMLTextAreaElement;
    act(() => {
      fireEvent.change(textarea, { target: { value: "settling text" } });
    });
    rerender(<Harness interim="" />);

    expect(screen.queryByTestId("composer-interim-overlay")).toBeNull();
    expect(textarea.value).toBe("settling text");
  });
});
