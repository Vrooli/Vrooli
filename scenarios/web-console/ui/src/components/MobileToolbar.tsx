// DOC: docs/reference/configuration.md#mobile-toolbar-keys
// DOC: docs/internal/SEAMS.md#axis-2-toolbar-keys-p0-007
import { useCallback } from "react";
import { TOOLBAR_KEYS, type ToolbarKey } from "../consts/toolbar-keys";
import { cn } from "../lib/classnames";
import { slugify } from "../lib/slugify";

// [REQ:P0-007a] Floating Toolbar Component
// [REQ:P0-007b] Terminal Key/Chord Mapping

interface MobileToolbarProps {
  /** Callback to inject input into the active terminal. */
  onInput: (data: string) => void;
  /** Whether the toolbar is visible. */
  visible?: boolean;
}

export default function MobileToolbar({
  onInput,
  visible = true,
}: MobileToolbarProps) {
  const handleKey = useCallback(
    (key: ToolbarKey) => {
      onInput(key.input);
    },
    [onInput],
  );

  if (!visible) return null;

  return (
    <div
      data-testid="mobile-toolbar"
      className="flex shrink-0 items-center gap-1 overflow-x-auto border-t border-wc-default bg-wc-surface-raised px-2 py-1.5 md:hidden"
    >
      {TOOLBAR_KEYS.map((key) => (
        <button
          key={key.label}
          data-testid={`toolbar-key-${slugify(key.label)}`}
          onClick={() => handleKey(key)}
          className={cn(
            "shrink-0 rounded border border-wc-default bg-wc-surface-input px-2 py-1.5 text-xs font-medium text-wc-text-secondary transition active:bg-wc-accent-active",
            key.width === "wide" ? "min-w-[4rem]" : key.width === "narrow" ? "min-w-[2rem]" : "min-w-[2.5rem]",
          )}
        >
          {key.label}
        </button>
      ))}
    </div>
  );
}
