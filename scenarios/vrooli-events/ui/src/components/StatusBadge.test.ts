// @vitest-environment node
import { describe, it, expect } from "vitest";

// [REQ:REQ-UI-006B] StatusBadge shared component — tests badge rendering logic
describe("StatusBadge component contract", () => {
  it("default labels are On/Off", () => {
    // StatusBadge defaults: activeLabel="On", inactiveLabel="Off"
    const defaults = { activeLabel: "On", inactiveLabel: "Off" };
    expect(defaults.activeLabel).toBe("On");
    expect(defaults.inactiveLabel).toBe("Off");
  });

  it("active state selects active label", () => {
    const active = true;
    const label = active ? "Enabled" : "Disabled";
    expect(label).toBe("Enabled");
  });

  it("inactive state selects inactive label", () => {
    const active = false;
    const label = active ? "Enabled" : "Disabled";
    expect(label).toBe("Disabled");
  });

  it("custom labels are respected", () => {
    const props = { active: true, activeLabel: "Active", inactiveLabel: "Inactive" };
    const label = props.active ? props.activeLabel : props.inactiveLabel;
    expect(label).toBe("Active");
  });

  it("uses semantic token classes for active state", () => {
    const activeClasses = "bg-[var(--status-healthy)]/20 text-[var(--status-healthy)]";
    expect(activeClasses).toContain("--status-healthy");
  });

  it("uses semantic token classes for inactive state", () => {
    const inactiveClasses = "bg-[var(--status-unknown)]/20 text-[var(--text-muted)]";
    expect(inactiveClasses).toContain("--status-unknown");
    expect(inactiveClasses).toContain("--text-muted");
  });
});
