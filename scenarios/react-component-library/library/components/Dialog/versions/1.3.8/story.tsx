import { useRef, useState } from "react";
import { Dialog } from "./Dialog";

/** Live specimen for focus containment, focus return, naming, and consumer-CSS composition. */
export function Default() {
  const triggerRef = useRef<HTMLButtonElement>(null);
  const [open, setOpen] = useState(true);

  return (
    <main data-rcl-story-background>
      <button ref={triggerRef} type="button" onClick={() => setOpen(true)}>
        Open workspace dialog
      </button>
      <Dialog
        open={open}
        title="Edit workspace"
        description="Update the workspace name without losing your draft."
        closeLabel="Close workspace dialog"
        returnFocusRef={triggerRef}
        onClose={() => setOpen(false)}
        footer={<button type="button">Save workspace</button>}
      >
        <label>
          Workspace name
          <input aria-label="Workspace name" defaultValue="Design system" />
        </label>
      </Dialog>
    </main>
  );
}
