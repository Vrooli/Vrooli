import { describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";

import { Select } from "./select";

describe("Select", () => {
  it("renders options and fires change", () => {
    const onChange = vi.fn();
    render(
      <Select data-testid="s" onChange={onChange}>
        <option value="a">A</option>
        <option value="b">B</option>
      </Select>,
    );
    const sel = screen.getByTestId<HTMLSelectElement>("s");
    expect(sel.tagName).toBe("SELECT");
    fireEvent.change(sel, { target: { value: "b" } });
    expect(onChange).toHaveBeenCalled();
  });

  it("applies the base classes and merges custom className", () => {
    render(
      <Select data-testid="s" className="custom">
        <option>A</option>
      </Select>,
    );
    const el = screen.getByTestId("s");
    expect(el.className).toMatch(/rounded-control/);
    expect(el.className).toMatch(/custom/);
  });
});
