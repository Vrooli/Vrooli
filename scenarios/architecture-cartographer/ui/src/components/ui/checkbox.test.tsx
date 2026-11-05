import { describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";

import { Checkbox } from "./checkbox";

describe("Checkbox", () => {
  it("renders as a checkbox input", () => {
    render(<Checkbox data-testid="c" />);
    expect(screen.getByTestId<HTMLInputElement>("c").type).toBe("checkbox");
  });

  it("toggles checked state and fires change", () => {
    const onChange = vi.fn();
    render(<Checkbox data-testid="c" onChange={onChange} />);
    fireEvent.click(screen.getByTestId("c"));
    expect(onChange).toHaveBeenCalled();
  });
});
