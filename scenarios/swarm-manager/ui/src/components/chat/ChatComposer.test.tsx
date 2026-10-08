import { fireEvent, screen } from "@testing-library/react";
import { renderWithProviders as renderWithCanonicalProviders } from "../../test-utils/renderWithProviders";
import { describe, expect, it, vi } from "vitest";
import { ChatComposer } from "./ChatComposer";

describe("ChatComposer", () => {
  it("submits on Ctrl+Enter when text is present", () => {
    const onSubmit = vi.fn();
    renderWithCanonicalProviders(<ChatComposer value="Next step" onChange={vi.fn()} onSubmit={onSubmit} testId="composer" />);

    fireEvent.keyDown(screen.getByTestId("composer"), { key: "Enter", ctrlKey: true });

    expect(onSubmit).toHaveBeenCalledTimes(1);
  });

  it("does not submit empty text", () => {
    const onSubmit = vi.fn();
    renderWithCanonicalProviders(<ChatComposer value="  " onChange={vi.fn()} onSubmit={onSubmit} testId="composer" />);

    fireEvent.click(screen.getByTestId("composer-submit"));
    fireEvent.keyDown(screen.getByTestId("composer"), { key: "Enter", metaKey: true });

    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("is controlled by the parent", () => {
    const onChange = vi.fn();
    renderWithCanonicalProviders(<ChatComposer value="" onChange={onChange} onSubmit={vi.fn()} testId="composer" />);

    fireEvent.change(screen.getByTestId("composer"), { target: { value: "Draft" } });

    expect(onChange).toHaveBeenCalledWith("Draft");
  });

  it("disables input and submit while loading", () => {
    renderWithCanonicalProviders(<ChatComposer value="Draft" onChange={vi.fn()} onSubmit={vi.fn()} isSubmitting testId="composer" />);

    expect(screen.getByTestId("composer")).toBeDisabled();
    expect(screen.getByTestId("composer-submit")).toBeDisabled();
  });
});
