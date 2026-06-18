import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { SegmentedControl } from "./segmented-control";

describe("SegmentedControl", () => {
  const options = [
    { value: "a" as const, label: "Alpha" },
    { value: "b" as const, label: "Bravo" },
  ];

  it("renders a radiogroup with the checked option reflecting value", () => {
    render(
      <SegmentedControl label="Choice" value="b" options={options} onChange={vi.fn()} />,
    );
    const group = screen.getByRole("radiogroup", { name: "Choice" });
    expect(group).toBeInTheDocument();
    const radios = screen.getAllByRole("radio");
    expect(radios).toHaveLength(2);
    expect(radios[1]).toHaveAttribute("aria-checked", "true");
    expect(radios[0]).toHaveAttribute("aria-checked", "false");
  });

  it("emits the option value on click", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <SegmentedControl
        label="Choice"
        value="a"
        options={options}
        onChange={onChange}
        optionTestId={(v) => `seg-${v}`}
      />,
    );
    await user.click(screen.getByTestId("seg-b"));
    expect(onChange).toHaveBeenCalledWith("b");
  });
});
