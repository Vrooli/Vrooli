import { useRef, useState } from "react";
import { useOutsideInteraction } from "./useOutsideInteraction";

export function Default({ args }: StoryHarnessProps<{ active: boolean }>) {
  const surfaceRef = useRef<HTMLDivElement>(null);
  const [message, setMessage] = useState("Surface is open");
  useOutsideInteraction({
    active: args.active,
    surfaceRef,
    onPointerDownOutside: () =>
      setMessage("Dismissed from outside pointer interaction"),
    onFocusOutside: () =>
      setMessage("Dismissed from outside focus interaction"),
    onEscape: () => setMessage("Dismissed with Escape"),
  });
  return (
    <div
      style={{
        display: "grid",
        gap: "var(--space-md)",
        minBlockSize: 220,
        padding: "var(--space-lg)",
        background: "var(--color-surface-raised)",
        color: "var(--color-foreground)",
      }}
    >
      <div
        ref={surfaceRef}
        style={{
          display: "grid",
          gap: "var(--space-xs)",
          padding: "var(--space-md)",
          border: "1px solid var(--color-border)",
          borderRadius: "var(--radius-panel)",
        }}
      >
        <strong>Layer surface</strong>
        <span role="status" aria-live="polite">
          {message}
        </span>
        <button type="button">Inside action</button>
      </div>
      <button type="button" aria-label="Outside action">
        Outside action
      </button>
    </div>
  );
}
