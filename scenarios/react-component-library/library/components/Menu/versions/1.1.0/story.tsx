import { useState } from "react";
import { Menu, MenuParts } from "./Menu";

const shell = {
  display: "grid",
  alignContent: "start",
  gap: "var(--space-md)",
  width: "min(100%, 620px)",
  minHeight: 360,
  padding: "var(--space-xl)",
  border: "1px solid var(--color-border)",
  borderRadius: "var(--radius-panel)",
  background:
    "linear-gradient(145deg, var(--color-surface-raised), color-mix(in srgb, var(--color-primary) 5%, var(--color-surface-raised)))",
  boxShadow: "var(--elev-raised)",
} as const;

function Showcase({
  title,
  detail,
  children,
}: {
  title: string;
  detail: string;
  children: React.ReactNode;
}) {
  return (
    <section style={shell}>
      <div style={{ display: "grid", gap: "var(--space-2xs)" }}>
        <span
          style={{
            color: "var(--color-primary)",
            font: "var(--text-overline)",
            letterSpacing: ".1em",
            textTransform: "uppercase",
          }}
        >
          Command surface
        </span>
        <strong style={{ font: "var(--text-title)" }}>{title}</strong>
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
          }}
        >
          {detail}
        </span>
      </div>
      {children}
    </section>
  );
}

function Items() {
  const [checked, setChecked] = useState(true);
  return (
    <>
      <MenuParts.Label>Workspace</MenuParts.Label>
      <MenuParts.Item shortcut="⌘K" onSelect={() => undefined}>
        Open command center
      </MenuParts.Item>
      <MenuParts.Item shortcut="⌘P" onSelect={() => undefined}>
        Publish changes
      </MenuParts.Item>
      <MenuParts.Separator />
      <MenuParts.CheckboxItem checked={checked} onCheckedChange={setChecked}>
        Show activity
      </MenuParts.CheckboxItem>
      <MenuParts.RadioItem checked onCheckedChange={() => undefined}>
        Use system theme
      </MenuParts.RadioItem>
      <MenuParts.Submenu label="More views">
        <MenuParts.Item onSelect={() => undefined}>
          Activity timeline
        </MenuParts.Item>
        <MenuParts.Item onSelect={() => undefined}>
          Audit history
        </MenuParts.Item>
      </MenuParts.Submenu>
      <MenuParts.Item disabled>Manage billing</MenuParts.Item>
    </>
  );
}

export function Default() {
  return (
    <Showcase
      title="Commands stay close to context"
      detail="Arrow keys move, typeahead jumps, and selection semantics remain visible without color alone."
    >
      <Menu defaultOpen>
        <MenuParts.Trigger>Workspace menu</MenuParts.Trigger>
        <MenuParts.Content aria-label="Workspace commands">
          <Items />
        </MenuParts.Content>
      </Menu>
    </Showcase>
  );
}
export function Interactive() {
  return (
    <Showcase
      title="A menu that respects momentum"
      detail="Open the menu, choose an action, or press Escape to return to the trigger."
    >
      <Menu>
        <MenuParts.Trigger>Workspace menu</MenuParts.Trigger>
        <MenuParts.Content aria-label="Workspace commands">
          <Items />
        </MenuParts.Content>
      </Menu>
    </Showcase>
  );
}
