import { Popover, PopoverParts } from "./Popover";

/** Live specimen for anchoring, focus return, dismissal, and consumer-CSS composition. */
export function Default() {
  return (
    <main data-rcl-story-background>
      <Popover defaultOpen placement="bottom-start">
        <PopoverParts.Trigger aria-label="Open workspace details">
          Workspace details
        </PopoverParts.Trigger>
        <PopoverParts.Content aria-label="Workspace details" initialFocus="first">
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

/** An inline anchor keeps its trigger inside the sentence instead of breaking the line. */
export function InlineInProse() {
  return (
    <main data-rcl-story-background>
      <p data-testid="popover-prose" style={{ maxInlineSize: "28rem" }}>
        Keep the original{" "}
        <Popover inline placement="bottom-start">
          <PopoverParts.Trigger asChild>
            <button type="button">08:53 UTC</button>
          </PopoverParts.Trigger>
          <PopoverParts.Content aria-label="Cutoff details">
            <div style={{ padding: "var(--space-sm)" }}>4:53 AM your time</div>
          </PopoverParts.Content>
        </Popover>{" "}
        trial cutoff and continue the remaining checks.
      </p>
    </main>
  );
}
