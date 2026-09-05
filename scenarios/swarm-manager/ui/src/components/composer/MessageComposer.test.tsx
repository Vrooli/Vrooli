/**
 * Tests for MessageComposer's imperative handle.
 *
 * `replaceText` must keep the controlled value in sync even where the
 * undo-preserving execCommand path is unavailable (jsdom exercises the
 * fallback), and must respect a disabled composer.
 */
import { describe, expect, it, vi } from "vitest";
import { createRef } from "react";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { MessageComposer, type MessageComposerHandle } from "./MessageComposer";

function renderComposer(props: Partial<React.ComponentProps<typeof MessageComposer>> = {}) {
  const ref = createRef<MessageComposerHandle>();
  const onChange = vi.fn();
  render(
    <MessageComposer
      ref={ref}
      value="typed by hand"
      onChange={onChange}
      onSubmit={vi.fn()}
      {...props}
    />,
  );
  return { ref, onChange };
}

describe("MessageComposer replaceText", () => {
  it("updates the value via onChange when execCommand is unavailable", () => {
    const { ref, onChange } = renderComposer();

    act(() => ref.current?.replaceText("Suggested prompt."));

    expect(onChange).toHaveBeenCalledWith("Suggested prompt.");
  });

  it("still replaces text when the composer is disabled (plain state set)", () => {
    const { ref, onChange } = renderComposer({ disabled: true });

    act(() => ref.current?.replaceText("Suggested prompt."));

    expect(onChange).toHaveBeenCalledWith("Suggested prompt.");
  });
});

describe("MessageComposer layout controls", () => {
  it("starts as a normal two-row, focusable textarea", () => {
    renderComposer({ testId: "composer" });
    const textarea = screen.getByTestId("composer");

    expect(textarea).toHaveAttribute("rows", "2");
    expect(textarea).not.toHaveAttribute("disabled");
    textarea.focus();
    expect(document.activeElement).toBe(textarea);
  });

  it("keeps the painted overlay at the textarea scroll position", () => {
    renderComposer({ testId: "composer" });
    const textarea = screen.getByTestId("composer");
    const overlay = screen.getByTestId("composer-overlay");
    Object.defineProperty(textarea, "scrollTop", { configurable: true, value: 48, writable: true });

    fireEvent.scroll(textarea);

    expect(overlay.scrollTop).toBe(48);
  });
});
