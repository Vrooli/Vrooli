import { renderWithProviders } from "../../test-utils";
import { describe, expect, it, vi } from "vitest";
import { screen, fireEvent } from "@testing-library/react";

import { Checkbox } from "./checkbox";

describe("Checkbox", () => {
  it("renders as a checkbox input", () => {
    renderWithProviders(<Checkbox data-testid="c" />);
    expect(screen.getByTestId<HTMLInputElement>("c").type).toBe("checkbox");
  });

  it("toggles checked state and fires change", () => {
    const onChange = vi.fn();
    renderWithProviders(<Checkbox data-testid="c" onChange={onChange} />);
    fireEvent.click(screen.getByTestId("c"));
    expect(onChange).toHaveBeenCalled();
  });
});
