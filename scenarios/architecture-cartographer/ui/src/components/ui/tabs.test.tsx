import { describe, expect, it } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";

import { selectors } from "../../consts/selectors";
import { Tabs, TabsList, TabsTrigger, TabsPanel } from "./tabs";

describe("Tabs", () => {
  const renderTabs = (defaultValue = "a") =>
    render(
      <Tabs defaultValue={defaultValue} ariaLabel="t">
        <TabsList>
          <TabsTrigger value="a">A</TabsTrigger>
          <TabsTrigger value="b">B</TabsTrigger>
        </TabsList>
        <TabsPanel value="a">panel-a</TabsPanel>
        <TabsPanel value="b">panel-b</TabsPanel>
      </Tabs>,
    );

  it("renders the default panel and triggers", () => {
    renderTabs();
    expect(screen.getByTestId(selectors.ui.tabs.panel({ value: "a" }))).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.ui.tabs.panel({ value: "b" }))).toBeNull();
    expect(
      screen.getByTestId(selectors.ui.tabs.trigger({ value: "a" })),
    ).toHaveAttribute("aria-selected", "true");
  });

  it("switches panel when a different trigger is clicked", () => {
    renderTabs();
    fireEvent.click(screen.getByTestId(selectors.ui.tabs.trigger({ value: "b" })));
    expect(screen.getByTestId(selectors.ui.tabs.panel({ value: "b" }))).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.ui.tabs.trigger({ value: "b" })),
    ).toHaveAttribute("aria-selected", "true");
  });
});
