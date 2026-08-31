import { useState } from "react";
import { CommandPalette } from "./CommandPalette";

const commands = [
  {
    id: "open-workspace",
    label: "Open workspace",
    description: "Navigate to the active workspace.",
    group: "Workspace",
    shortcut: "Enter",
    run: () => undefined,
  },
  {
    id: "invite-collaborator",
    label: "Invite collaborator",
    description: "Share access without leaving the keyboard.",
    group: "Workspace",
    shortcut: "I",
    run: () => undefined,
  },
];

/** Live specimen for modal keyboard behavior and consumer-CSS composition. */
export function Default() {
  const [open, setOpen] = useState(true);

  return (
    <main data-rcl-story-background>
      <button type="button" onClick={() => setOpen(true)}>
        Open command palette
      </button>
      <CommandPalette
        open={open}
        onClose={() => setOpen(false)}
        title="Command palette"
        description="Search workspace actions."
        commands={commands}
      />
    </main>
  );
}
