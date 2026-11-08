import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Select } from "@vrooli/react-component-library/Select/1.3.21";

const options = [
  { value: "light", label: "Light" },
  { value: "dark", label: "Dark" },
];

describe("Select labelled by a <label for>", () => {
  it("names the trigger after the label, and the label moves focus to it", () => {
    render(
      <>
        <label htmlFor="theme">Theme</label>
        <Select id="theme" options={options} defaultValue="dark" />
      </>,
    );

    const trigger = screen.getByRole("button", { name: "Theme" });
    expect(trigger).toHaveTextContent("Dark");
    // The native select keeps the form value; a label click lands on it and
    // hands focus to the trigger.
    const native = document.querySelector("select") as HTMLSelectElement;
    expect(native.labels?.[0]).toHaveTextContent("Theme");
    expect(native).toHaveValue("dark");
    fireEvent.focus(native);
    expect(trigger).toHaveFocus();
  });

  it("borrows the text of a wrapping label without reading its own value", () => {
    render(
      <label htmlFor="density">
        <span>Density</span>
        <Select id="density" options={options} defaultValue="light" />
      </label>,
    );
    expect(screen.getByRole("button", { name: "Density" })).toBeInTheDocument();
  });

  it("keeps aria-label naming when no label element is used", () => {
    render(<Select aria-label="Density" options={options} />);
    expect(screen.getByRole("button", { name: "Density" })).toBeInTheDocument();
  });
});
