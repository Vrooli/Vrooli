import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ChatComposer } from "./ChatComposer";

describe("ChatComposer", () => {
  it("submits on Ctrl+Enter when text is present", () => {
    const onSubmit = vi.fn();
    render(<ChatComposer value="Next step" onChange={vi.fn()} onSubmit={onSubmit} testId="composer" />);

    fireEvent.keyDown(screen.getByTestId("composer"), { key: "Enter", ctrlKey: true });

    expect(onSubmit).toHaveBeenCalledTimes(1);
  });

  it("does not submit empty text", () => {
    const onSubmit = vi.fn();
    render(<ChatComposer value="  " onChange={vi.fn()} onSubmit={onSubmit} testId="composer" />);

    fireEvent.click(screen.getByTestId("composer-submit"));
    fireEvent.keyDown(screen.getByTestId("composer"), { key: "Enter", metaKey: true });

    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("is controlled by the parent", () => {
    const onChange = vi.fn();
    render(<ChatComposer value="" onChange={onChange} onSubmit={vi.fn()} testId="composer" />);

    fireEvent.change(screen.getByTestId("composer"), { target: { value: "Draft" } });

    expect(onChange).toHaveBeenCalledWith("Draft");
  });

  it("disables input and submit while loading", () => {
    render(<ChatComposer value="Draft" onChange={vi.fn()} onSubmit={vi.fn()} isSubmitting testId="composer" />);

    expect(screen.getByTestId("composer")).toBeDisabled();
    expect(screen.getByTestId("composer-submit")).toBeDisabled();
  });
});
