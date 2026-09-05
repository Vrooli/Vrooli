import { Menu, MenuParts } from "./Menu";

/** Live specimen for roving focus, typeahead, dismissal, and consumer-CSS composition. */
export function Default() {
  return (
    <main data-rcl-story-background>
      <Menu defaultOpen>
        <MenuParts.Trigger aria-label="Open workspace actions">
          Workspace actions
        </MenuParts.Trigger>
        <MenuParts.Content aria-label="Workspace actions">
          <MenuParts.Label>Workspace</MenuParts.Label>
          <MenuParts.Item shortcut="Enter">Open workspace</MenuParts.Item>
          <MenuParts.Item shortcut="I">Invite collaborator</MenuParts.Item>
          <MenuParts.Separator />
          <MenuParts.CheckboxItem checked>
            Keep sidebar open
          </MenuParts.CheckboxItem>
        </MenuParts.Content>
      </Menu>
    </main>
  );
}
