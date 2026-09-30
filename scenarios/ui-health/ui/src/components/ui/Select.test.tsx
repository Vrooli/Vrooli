import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { Select } from "./Select";

const ARIA = "tone-x";

describe("Select", () => {
  it("calls onChange when selection changes", async () => {
    const onChange = vi.fn();
    render(
      <Select
        ariaLabel={ARIA}
        value="a"
        onChange={onChange}
        options={[
          { value: "a", label: "alpha-x" },
          { value: "b", label: "beta-x" },
        ]}
      />,
    );
    await userEvent.selectOptions(screen.getByRole("combobox", { name: ARIA }), "b");
    expect(onChange).toHaveBeenCalledWith("b");
  });
});
