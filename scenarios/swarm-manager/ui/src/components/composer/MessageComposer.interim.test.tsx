import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";

vi.mock("./MicButton", () => ({
  MicButton: ({ onPartialTranscript, onTranscript }: {
    onPartialTranscript?: (text: string) => void;
    onTranscript: (text: string) => void;
  }) => (
    <div>
      <button type="button" data-testid="mock-partial" onClick={() => onPartialTranscript?.("unsettled words")}>partial</button>
      <button type="button" data-testid="mock-final" onClick={() => onTranscript("unsettled words")}>final</button>
    </div>
  ),
}));

import { MessageComposer } from "./MessageComposer";

describe("MessageComposer interim transcript", () => {
  it("renders interim text as dotted-underlined content without adding it to the value", () => {
    const onChange = vi.fn();
    render(<MessageComposer value="settled" onChange={onChange} onSubmit={vi.fn()} onTranscript={vi.fn()} />);

    fireEvent.click(screen.getByTestId("mock-partial"));

    const interim = screen.getByTestId("composer-interim");
    expect(interim).toHaveTextContent("unsettled words");
    expect(interim).toHaveClass("border-dotted");
    expect(screen.getByRole("textbox")).toHaveValue("settled");
  });

  it("materializes interim text before a user edit and ignores its duplicate final", () => {
    const onChange = vi.fn();
    const onTranscript = vi.fn();
    render(<MessageComposer value="settled" onChange={onChange} onSubmit={vi.fn()} onTranscript={onTranscript} />);

    fireEvent.click(screen.getByTestId("mock-partial"));
    fireEvent.focus(screen.getByRole("textbox"));
    expect(onChange).toHaveBeenCalledWith("settled unsettled words");
    expect(screen.queryByTestId("composer-interim")).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId("mock-final"));
    expect(onTranscript).not.toHaveBeenCalled();
  });
});

