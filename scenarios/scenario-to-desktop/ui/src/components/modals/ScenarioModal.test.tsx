import { fireEvent, render, screen } from "@/test-utils";
import type React from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ScenarioModal } from "./ScenarioModal";
import type { ScenarioDesktopStatus } from "../scenario-inventory/types";

const scenarios: ScenarioDesktopStatus[] = [
  { name: "canvas-lab", display_name: "Canvas Lab", has_desktop: true },
  { name: "bridge", display_name: "Bridge", has_desktop: false },
];

function renderModal(overrides: Partial<React.ComponentProps<typeof ScenarioModal>> = {}) {
  const props: React.ComponentProps<typeof ScenarioModal> = {
    open: true,
    loading: false,
    scenarios,
    selectedScenarioName: "",
    onSelect: vi.fn(),
    onClose: vi.fn(),
    ...overrides,
  };
  render(<ScenarioModal {...props} />);
  return props;
}

describe("ScenarioModal", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("does not render a portal while closed", () => {
    renderModal({ open: false });
    expect(screen.queryByText("Choose a scenario")).not.toBeInTheDocument();
  });

  it("filters by display name and selects an available scenario", () => {
    const props = renderModal();

    fireEvent.change(screen.getByRole("textbox", { name: "Search scenarios" }), {
      target: { value: "canvas" },
    });
    expect(screen.getByText("Canvas Lab")).toBeInTheDocument();
    expect(screen.queryByText("Bridge")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /Canvas Lab/ }));
    expect(props.onSelect).toHaveBeenCalledWith("canvas-lab");
    expect(window.localStorage.getItem("scenario-to-desktop:recents")).toBe(
      '["canvas-lab"]',
    );
  });

  it("allows an exact custom slug and stores it as a recent choice", () => {
    const props = renderModal();

    fireEvent.change(screen.getByRole("textbox", { name: "Search scenarios" }), {
      target: { value: "new-desktop-scenario" },
    });
    fireEvent.click(screen.getByRole("button", { name: /Use "new-desktop-scenario" slug/ }));

    expect(props.onSelect).toHaveBeenCalledWith("new-desktop-scenario");
    expect(window.localStorage.getItem("scenario-to-desktop:recents")).toBe(
      '["new-desktop-scenario"]',
    );
  });

  it("shows stored recents, including removed scenarios", () => {
    window.localStorage.setItem(
      "scenario-to-desktop:recents",
      '["canvas-lab","retired-app"]',
    );
    renderModal();

    expect(screen.getByText("Recents")).toBeInTheDocument();
    expect(screen.getByText("retired-app")).toBeInTheDocument();
    expect(screen.getByText("Recent")).toBeInTheDocument();
  });

  it("shows loading and closes through its named control", () => {
    const loading = renderModal({ loading: true });
    expect(screen.getByText("Loading scenarios...")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Close scenario chooser" }));
    expect(loading.onClose).toHaveBeenCalledTimes(1);
  });

  it("explains when a search has no matching scenario", () => {
    renderModal();
    fireEvent.change(screen.getByRole("textbox", { name: "Search scenarios" }), {
      target: { value: "not-found" },
    });
    expect(screen.getByText("No scenarios match that search.")).toBeInTheDocument();
  });
});
