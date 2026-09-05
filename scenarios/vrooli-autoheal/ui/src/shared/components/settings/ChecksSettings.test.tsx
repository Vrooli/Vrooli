import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ChecksSettings } from "./ChecksSettings";
import { renderWithProviders } from "../../../test-utils";

describe("ChecksSettings", () => {
  it("renders categories, toggles checks, and handles collapsed or empty groups", () => {
    const onToggleEnabled = vi.fn();
    const onToggleAutoHeal = vi.fn();
    const onBulkUpdate = vi.fn();
    renderWithProviders(
      <ChecksSettings
        checksByCategory={{
          infrastructure: [{ id: "dns", title: "DNS", description: "Resolves names", importance: "required", category: "infrastructure", intervalSeconds: 30, config: { enabled: true, autoHeal: true } }],
          resource: [],
          system: [{ id: "custom", title: "Custom", description: "Custom check", importance: "optional", category: "system", intervalSeconds: 60, config: { enabled: false, autoHeal: false } }],
        }}
        expandedCategories={{ infrastructure: true, resource: false, system: false }}
        toggleCategory={vi.fn()}
        categoryLabels={{ infrastructure: "Infrastructure", system: "System" }}
        categoryIcons={{}}
        onToggleEnabled={onToggleEnabled}
        onToggleAutoHeal={onToggleAutoHeal}
        onBulkUpdate={onBulkUpdate}
        isUpdating={false}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: /enable all$/i }));
    fireEvent.click(screen.getByRole("button", { name: /infrastructure/i }));
    const switches = screen.getAllByRole("switch");
    expect(switches[0]).toBeDefined();
    expect(switches[1]).toBeDefined();
    const enabledSwitch = switches[0];
    const autoHealSwitch = switches[1];
    if (!enabledSwitch || !autoHealSwitch) throw new Error("expected both check switches");
    fireEvent.click(enabledSwitch);
    fireEvent.click(autoHealSwitch);
    expect(onBulkUpdate).toHaveBeenCalledWith("enableAll");
    expect(onToggleEnabled).toHaveBeenCalledWith("dns", false);
    expect(onToggleAutoHeal).toHaveBeenCalledWith("dns", false);
  });
});
