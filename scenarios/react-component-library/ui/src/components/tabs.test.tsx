import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { vi } from "vitest";

import { Tabs } from "@vrooli/react-component-library/Tabs/1";
import { renderWithProviders } from "../test-utils";

describe("Tabs", () => {
  it("supports labeled items, badges, and roving keyboard selection", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <Tabs
        items={[
          { id: "overview", label: "Overview", badge: 3 },
          { id: "files", label: "Files" },
        ]}
        defaultActive="overview"
        ariaLabel="Asset information"
      />,
    );

    const overview = screen.getByRole("tab", { name: "Overview" });
    const files = screen.getByRole("tab", { name: "Files" });
    expect(overview).toHaveAttribute("aria-selected", "true");
    expect(overview).toHaveAttribute("tabindex", "0");
    expect(files).toHaveAttribute("tabindex", "-1");

    await user.click(files);
    expect(files).toHaveAttribute("aria-selected", "true");
    expect(overview).toHaveAttribute("aria-selected", "false");

    await user.keyboard("{ArrowLeft}");
    expect(overview).toHaveFocus();
    expect(overview).toHaveAttribute("aria-selected", "true");
  });

  it("supports string items, panels, custom ids, and Home/End navigation", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    renderWithProviders(
      <Tabs
        items={["one", { id: "two", label: "Two", badge: "new" }, "three"]}
        defaultActive="two"
        panels={{ two: <p>Two panel</p>, three: <p>Three panel</p> }}
        onChange={onChange}
        itemTestId={(id) => `tab-${id}`}
      />,
    );

    expect(screen.getByTestId("tab-two")).toHaveAttribute("aria-controls", "rcl-tab-panel-1");
    expect(screen.getByRole("tabpanel")).toHaveTextContent("Two panel");
    screen.getByTestId("tab-two").focus();
    await user.keyboard("{Home}");
    expect(screen.getByTestId("tab-one")).toHaveFocus();
    await user.keyboard("{End}");
    expect(screen.getByTestId("tab-three")).toHaveFocus();
    expect(onChange).toHaveBeenCalledWith("one");
    expect(onChange).toHaveBeenCalledWith("three");
  });

  it("selects the requested tab and announces the active tab", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    renderWithProviders(
      <Tabs
        items={[
          { id: "one", label: "One" },
          { id: "two", label: "Two" },
          { id: "three", label: "Three" },
        ]}
        defaultActive="one"
        onChange={onChange}
        itemTestId={(id) => `tab-${id}`}
      />,
    );

    const second = screen.getByTestId("tab-two");
    await user.click(second);
    expect(onChange).toHaveBeenCalledWith("two");
    expect(second).toHaveAttribute("aria-selected", "true");
  });

  it("keeps controlled selection controlled while reporting keyboard changes", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    renderWithProviders(
      <Tabs items={["one", "two"]} active="one" mode="controlled" onChange={onChange} />,
    );
    await user.click(screen.getByRole("tab", { name: "two" }));
    expect(onChange).toHaveBeenCalledWith("two");
    expect(screen.getByRole("tab", { name: "one" })).toHaveAttribute("aria-selected", "true");
  });
});
