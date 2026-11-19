import { useState } from "react";
import { describe, expect, it } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";

import { Segmented, Tabs } from "./primitives";

function TabsFixture() {
  const [value, setValue] = useState<"one" | "two" | "three">("one");
  return (
    <Tabs
      label="Sections"
      value={value}
      onChange={setValue}
      items={[
        { id: "one", label: "One" },
        { id: "two", label: "Two" },
        { id: "three", label: "Three" },
      ]}
    />
  );
}

describe("Tabs", () => {
  it("uses one tab stop and moves selection with arrow, Home and End keys", () => {
    render(<TabsFixture />);
    const tabs = screen.getAllByRole("tab");
    expect(tabs[0]).toHaveAttribute("tabindex", "0");
    expect(tabs[1]).toHaveAttribute("tabindex", "-1");

    fireEvent.keyDown(tabs[0]!, { key: "ArrowRight" });
    expect(tabs[1]).toHaveAttribute("aria-selected", "true");
    expect(tabs[1]).toHaveAttribute("tabindex", "0");
    expect(document.activeElement).toBe(tabs[1]);

    fireEvent.keyDown(tabs[1]!, { key: "End" });
    expect(tabs[2]).toHaveAttribute("aria-selected", "true");

    fireEvent.keyDown(tabs[2]!, { key: "Home" });
    expect(tabs[0]).toHaveAttribute("aria-selected", "true");

    // Wrapping is part of the roving-tabindex contract.
    fireEvent.keyDown(tabs[0]!, { key: "ArrowLeft" });
    expect(tabs[2]).toHaveAttribute("aria-selected", "true");
  });
});

describe("Segmented", () => {
  it("gives icon-only segments an accessible name", () => {
    render(
      <Segmented
        label="View"
        value="gallery"
        onChange={() => undefined}
        items={[
          { id: "gallery", label: <span aria-hidden>▦</span>, ariaLabel: "Gallery" },
          { id: "list", label: <span aria-hidden>≡</span>, ariaLabel: "List" },
        ]}
      />,
    );
    expect(screen.getByRole("button", { name: "Gallery" })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByRole("button", { name: "List" })).toHaveAttribute("aria-pressed", "false");
  });
});
