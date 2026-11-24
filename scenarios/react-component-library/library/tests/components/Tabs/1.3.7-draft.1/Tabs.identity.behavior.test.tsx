import { screen, within } from "@testing-library/react";
import { Tabs } from "@vrooli/react-component-library/Tabs/1";
import { renderWithProviders } from "../../../../../ui/src/test-utils";

describe("Tabs instance identity", () => {
  it("keeps tab and panel ids unique and correctly associated across tablists", () => {
    const items = [
      { id: "overview", label: "Overview" },
      { id: "activity", label: "Activity" },
    ];

    renderWithProviders(
      <>
        <Tabs
          ariaLabel="Primary sections"
          items={items}
          defaultActive="overview"
          panels={{ overview: <p>Primary overview panel</p> }}
        />
        <Tabs
          ariaLabel="Dialog sections"
          items={items}
          defaultActive="overview"
          panels={{ overview: <p>Dialog overview panel</p> }}
        />
      </>,
    );

    const tablists = screen.getAllByRole("tablist");
    const tabs = tablists.map((tablist) => within(tablist).getByRole("tab", { name: "Overview" }));
    const panels = screen.getAllByRole("tabpanel");
    const tabIds = tabs.map((tab) => tab.id);
    const panelIds = panels.map((panel) => panel.id);

    expect(new Set(tabIds).size).toBe(2);
    expect(new Set(panelIds).size).toBe(2);
    expect(tabs.map((tab) => tab.getAttribute("aria-controls"))).toEqual(panelIds);
    expect(panels.map((panel) => panel.getAttribute("aria-labelledby"))).toEqual(tabIds);
    expect(panels.map((panel) => panel.textContent)).toEqual([
      "Primary overview panel",
      "Dialog overview panel",
    ]);
  });
});
