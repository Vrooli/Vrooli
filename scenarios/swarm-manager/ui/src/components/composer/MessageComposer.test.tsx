/**
 * Tests for MessageComposer's imperative handle.
 *
 * `replaceText` must keep the controlled value in sync even where the
 * undo-preserving execCommand path is unavailable (jsdom exercises the
 * fallback), and must respect a disabled composer.
 */
import { describe, expect, it, vi } from "vitest";
import { createRef } from "react";
import { act, render } from "@testing-library/react";
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
