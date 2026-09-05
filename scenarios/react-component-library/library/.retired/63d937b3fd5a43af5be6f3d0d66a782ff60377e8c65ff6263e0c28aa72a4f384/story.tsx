import { Popover, PopoverParts } from "./Popover";

/** Live specimen for anchoring, focus return, dismissal, and consumer-CSS composition. */
export function Default() {
  return (
    <main data-rcl-story-background>
      <Popover defaultOpen placement="bottom-start">
        <PopoverParts.Trigger aria-label="Open workspace details">
          Workspace details
        </PopoverParts.Trigger>
        <PopoverParts.Content
          aria-label="Workspace details"
          initialFocus="first"
        >
          <div
            style={{
              display: "grid",
              gap: "var(--space-xs)",
              padding: "var(--space-sm)",
            }}
          >
            <strong>Design system</strong>
            <span>Shared tokens and reusable components.</span>
            <button type="button">Open workspace</button>
          </div>
        </PopoverParts.Content>
      </Popover>
    </main>
  );
}
