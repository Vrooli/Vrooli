import { useState } from "react";
import { Badge } from "@vrooli/react-component-library/Badge/1.0.0";
import { Icon } from "@vrooli/react-component-library/Icon/1.1.0";
import { NavLink } from "@vrooli/react-component-library/NavLink/1.0.0";
import { SidebarShell } from "./SidebarShell";

function NavigationSpecimen() {
  return (
    <nav aria-label="Workspace sections" style={{ display: "grid", gap: "var(--space-2xs)" }}>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: "var(--space-xs)", padding: "var(--space-xs)" }}>
        <strong>Northstar workspace</strong>
        <Badge tone="warning">2 alerts</Badge>
      </div>
      <NavLink label="Overview" href="/overview" current icon={<Icon name="check" size="sm" />} />
      <NavLink label="Resources" href="/resources" icon={<Icon name="search" size="sm" />} description="Manage connected resources" />
      <NavLink label="Activity" href="/activity" icon={<Icon name="arrowEnd" size="sm" />} />
      <NavLink label="Settings" href="/settings" icon={<Icon name="menu" size="sm" />} />
    </nav>
  );
}

export function Persistent({ args }: StoryHarnessProps) {
  const props = args as Record<string, unknown>;
  return (
    <SidebarShell mode="persistent" mobileOpen={false} mobileLabel="Workspace navigation" desktopLabel="Workspace" closeLabel="Close navigation" width={Number(props.width) || 280} onMobileClose={() => undefined}>
      <NavigationSpecimen />
    </SidebarShell>
  );
}

export function OverlayOpen() {
  const [open, setOpen] = useState(true);
  return (
    <SidebarShell mode="overlay" mobileOpen={open} mobileLabel="Mobile navigation" closeLabel="Close menu" mobileHeader={<strong>Northstar menu</strong>} onMobileClose={() => setOpen(false)}>
      <NavigationSpecimen />
    </SidebarShell>
  );
}

export function ResponsiveClosed() {
  return (
    <SidebarShell mode="responsive" mobileOpen={false} mobileLabel="Scenario navigation" desktopLabel="Scenario" closeLabel="Close menu" onMobileClose={() => undefined}>
      <NavigationSpecimen />
    </SidebarShell>
  );
}

export function FixtureFailure() {
  return (
    <SidebarShell mode="persistent" mobileOpen={false} mobileLabel="Workspace navigation" desktopLabel="Workspace" closeLabel="Close navigation" onMobileClose={() => undefined}>
      <div role="alert" style={{ padding: "var(--space-md)" }}>Navigation data is unavailable. Retry to restore workspace sections.</div>
    </SidebarShell>
  );
}
