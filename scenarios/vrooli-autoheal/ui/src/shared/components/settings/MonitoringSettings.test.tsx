import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { MonitoringSettings } from "./MonitoringSettings";
import { renderWithProviders } from "../../../test-utils";

describe("MonitoringSettings", () => {
  it("renders loading and empty states", () => {
    renderWithProviders(<MonitoringSettings monitoring={undefined} isLoading onAddScenario={vi.fn()} onRemoveScenario={vi.fn()} onSetCritical={vi.fn()} onAddResource={vi.fn()} onRemoveResource={vi.fn()} isUpdating={false} />);
    expect(document.querySelector("svg.lucide-loader-circle")).toBeInTheDocument();
    renderWithProviders(<MonitoringSettings monitoring={{ scenarios: {}, resources: [] }} isLoading={false} onAddScenario={vi.fn()} onRemoveScenario={vi.fn()} onSetCritical={vi.fn()} onAddResource={vi.fn()} onRemoveResource={vi.fn()} isUpdating={false} />);
    expect(screen.getByText(/no scenarios configured/i)).toBeInTheDocument();
    expect(screen.getByText(/no resources configured/i)).toBeInTheDocument();
  });

  it("adds, sorts, toggles, and removes monitored items", () => {
    const onAddScenario = vi.fn();
    const onRemoveScenario = vi.fn();
    const onSetCritical = vi.fn();
    const onAddResource = vi.fn();
    const onRemoveResource = vi.fn();
    renderWithProviders(
      <MonitoringSettings
        monitoring={{ scenarios: { zeta: { critical: false }, alpha: { critical: true } }, resources: ["redis", "postgres"] }}
        isLoading={false}
        onAddScenario={onAddScenario}
        onRemoveScenario={onRemoveScenario}
        onSetCritical={onSetCritical}
        onAddResource={onAddResource}
        onRemoveResource={onRemoveResource}
        isUpdating={false}
      />,
    );

    const textboxes = screen.getAllByRole("textbox");
    const [scenarioInput, resourceInput] = textboxes;
    const criticalCheckbox = screen.getAllByRole("checkbox")[0];
    const addButtons = screen.getAllByRole("button", { name: /^add$/i });
    if (!scenarioInput || !resourceInput || !criticalCheckbox || !addButtons[1]) throw new Error("Monitoring controls missing");
    fireEvent.change(scenarioInput, { target: { value: "new-app" } });
    fireEvent.click(criticalCheckbox);
    fireEvent.keyDown(scenarioInput, { key: "Enter" });
    fireEvent.change(resourceInput, { target: { value: "qdrant" } });
    fireEvent.keyDown(resourceInput, { key: "Enter" });
    fireEvent.click(screen.getByTitle("Make non-critical"));
    const removeScenario = screen.getAllByTitle("Remove scenario")[0];
    const removeResource = screen.getAllByTitle("Remove resource")[0];
    if (!removeScenario || !removeResource) throw new Error("Remove controls missing");
    fireEvent.click(removeScenario);
    fireEvent.click(removeResource);

    expect(onAddScenario).toHaveBeenCalledWith("new-app", true);
    expect(onAddResource).toHaveBeenCalledWith("qdrant");
    expect(onSetCritical).toHaveBeenCalledWith("alpha", false);
    expect(onRemoveScenario).toHaveBeenCalledWith("alpha");
    expect(onRemoveResource).toHaveBeenCalledWith("postgres");
  });
});
