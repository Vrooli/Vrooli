import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Target } from "lucide-react";
import { describe, expect, it, vi } from "vitest";
import { CompactTabBar } from "./compact-tab-bar";

describe("CompactTabBar", () => {
  it("renders optional tab icons and still changes tabs", async () => {
    const user = userEvent.setup();
    const onValueChange = vi.fn();

    render(
      <CompactTabBar
        aria-label="Example tabs"
        activeValue="goals"
        onValueChange={onValueChange}
        tabTestIdPrefix="tab"
        items={[
          { value: "goals", label: "Goals", icon: Target, count: 2 },
          { value: "other", label: "Other" },
        ]}
      />,
    );

    const goalsTab = screen.getByTestId("tab-goals");
    expect(goalsTab.querySelector("svg")).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: /Goals/ })).toHaveAttribute("aria-selected", "true");

    await user.click(screen.getByRole("tab", { name: "Other" }));
    expect(onValueChange).toHaveBeenCalledWith("other");
  });
});
