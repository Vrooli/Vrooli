import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ViewModeToggle } from "./ViewModeToggle";

describe("ViewModeToggle", () => {
  it("calls onChange with selected mode", () => {
    const onChange = vi.fn();
    render(<ViewModeToggle mode="code" onChange={onChange} />);
    fireEvent.click(screen.getByRole("button", { name: /Preview/i }));
    expect(onChange).toHaveBeenCalledWith("preview");
  });
});
