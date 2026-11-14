import { renderWithProviders } from "../../test-utils";
import { describe, expect, it, vi } from "vitest";
import { screen, fireEvent } from "@testing-library/react";

import { Radio } from "./radio";

describe("Radio", () => {
  it("renders as a radio input", () => {
    renderWithProviders(<Radio data-testid="r" name="g" value="a" />);
    expect(screen.getByTestId<HTMLInputElement>("r").type).toBe("radio");
  });

  it("fires change when selected", () => {
    const onChange = vi.fn();
    renderWithProviders(<Radio data-testid="r" name="g" value="a" onChange={onChange} />);
    fireEvent.click(screen.getByTestId("r"));
    expect(onChange).toHaveBeenCalled();
  });
});
