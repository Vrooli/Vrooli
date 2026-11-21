import { useRef, useState } from "react";
import { useHoverIntent } from "./useHoverIntent";

type StoryArgs = { openDelay?: number; closeDelay?: number; fuse?: number };
type StoryProps = { args: StoryArgs };

export function HoverIntentStory({ args }: StoryProps) {
  const childRef = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);
  const openDelay = args.openDelay ?? 120;
  const hover = useHoverIntent({
    childRect: () => childRef.current?.getBoundingClientRect() ?? null,
    onOpen: () => setOpen(true),
    onClose: () => setOpen(false),
    openDelay,
    closeDelay: args.closeDelay ?? 120,
    fuse: args.fuse ?? 160,
  });

  return (
    <section
      style={{
        display: "grid",
        gap: 16,
        width: "min(100%, 34rem)",
        padding: 24,
        border: "1px solid var(--color-border)",
        borderRadius: 12,
        background: "var(--color-surface-raised)",
        color: "var(--color-foreground)",
      }}
    >
      <div>
        <h2>Pointer-intent submenu</h2>
        <p role="status" aria-live="polite">
          {open
            ? "The submenu is open."
            : "The submenu is closed; mouse hover opens it after the configured delay."}
        </p>
        <p>Open delay: {openDelay} ms</p>
      </div>
      <button
        type="button"
        aria-label="Preview submenu"
        onPointerEnter={hover.onPointerEnter}
        onPointerMove={hover.onPointerMove}
        onPointerLeave={hover.onPointerLeave}
        style={{ minHeight: 44, padding: "8px 16px" }}
      >
        Preview submenu
      </button>
      {open ? (
        <div
          ref={childRef}
          data-hover-intent-panel
          onPointerEnter={hover.onChildEnter}
          style={{ padding: 16, border: "1px solid var(--color-border)" }}
        >
          Release notes and submenu actions
        </div>
      ) : null}
    </section>
  );
}
