import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it } from "vitest";

import { Tabs } from "./Tabs";

const LABEL_A = "alpha-x";
const LABEL_B = "beta-x";
const LABEL_C = "gamma-x";
const LIST_LABEL = "sections-x";

function Controlled() {
  const [v, setV] = useState<"a" | "b" | "c">("a");
  return (
    <Tabs
      ariaLabel={LIST_LABEL}
      value={v}
      onChange={setV}
      items={[
        { value: "a", label: LABEL_A },
        { value: "b", label: LABEL_B },
        { value: "c", label: LABEL_C },
      ]}
    />
  );
}

describe("Tabs", () => {
  it("renders a tablist with the selected tab", () => {
    render(<Controlled />);
    const list = screen.getByRole("tablist", { name: LIST_LABEL });
    expect(list).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: LABEL_A })).toHaveAttribute("aria-selected", "true");
  });

  it("switches tabs on click", async () => {
    render(<Controlled />);
    await userEvent.click(screen.getByRole("tab", { name: LABEL_B }));
    expect(screen.getByRole("tab", { name: LABEL_B })).toHaveAttribute("aria-selected", "true");
  });

  it("supports arrow-key navigation", async () => {
    render(<Controlled />);
    const a = screen.getByRole("tab", { name: LABEL_A });
    a.focus();
    await userEvent.keyboard("{ArrowRight}");
    expect(screen.getByRole("tab", { name: LABEL_B })).toHaveAttribute("aria-selected", "true");
    await userEvent.keyboard("{End}");
    expect(screen.getByRole("tab", { name: LABEL_C })).toHaveAttribute("aria-selected", "true");
  });
});
