import { describe, it, expect, vi } from "vitest";
import { render, act, fireEvent, screen } from "@testing-library/react";
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
  readonly initialOpen?: boolean;
}

function Harness({ interim = "", initialOpen = true }: HarnessProps) {
  const draft = useComposerDraft("sess-interim");
  const [open] = useState(initialOpen);
  return (
    <FullScreenComposer
      open={open}
      onClose={vi.fn()}
      draft={draft}
      onInput={((): GateResult => ({ status: "sent", seq: 1 })) as never}
      onFocusTerminal={vi.fn()}
      interimTranscript={interim}
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

    rerender(<Harness interim="speaking now" />);
    expect(textarea.className).toContain("text-transparent");
  });

  it("keeps the mirror out of the accessibility tree", () => {
    render(<Harness interim="spoken words" />);
    expect(screen.getByTestId("composer-interim-overlay").getAttribute("aria-hidden")).toBe("true");
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
