import { render, screen, fireEvent } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Tabs } from "@vrooli/react-component-library/Tabs/1.3.1";

describe("Tabs stylesheet ownership", () => {
  it("mounts tab rules with the base rules and retains compact keyboard selection", () => {
    render(<Tabs density="compact" items={["All", "Staged"]} />);
    const tab = screen.getByRole("tab", { name: "All" });
    expect(tab.closest("[data-rcl-tabs]")).toHaveAttribute("data-rcl-tabs-density", "compact");
    const sheet = document.querySelector('style[data-rcl-sheet="react-component-library-tabs-1\\.3\\.1"]');
    expect(sheet?.textContent).toContain('[data-rcl-tabs-density="compact"] [data-rcl-tab]');
    expect(sheet?.textContent).toContain('[data-rcl-tab][aria-selected="true"]');
    fireEvent.keyDown(tab, { key: "ArrowRight" });
    expect(screen.getByRole("tab", { name: "Staged" })).toHaveAttribute("aria-selected", "true");
  });
});
