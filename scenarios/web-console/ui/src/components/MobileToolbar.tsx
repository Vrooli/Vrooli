// DOC: docs/reference/configuration.md#mobile-toolbar-keys
// DOC: docs/internal/SEAMS.md#axis-2-toolbar-keys-p0-007
import { useCallback, useRef, useState } from "react";
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
  const [inputValue, setInputValue] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  const handleKey = useCallback(
    (key: ToolbarKey) => {
      onInput(key.input);
    },
    [onInput],
  );

  const submitCommand = useCallback(() => {
    if (!inputValue) return;
    onInput(inputValue + "\n");
    setInputValue("");
    inputRef.current?.focus();
  }, [inputValue, onInput]);

  if (!visible) return null;

  return (
    <div
      data-testid="mobile-toolbar"
      className="flex shrink-0 flex-col border-t border-wc-default bg-wc-surface-raised md:hidden"
    >
      {/* Command input row */}
      <div className="flex items-center gap-1 px-2 py-1.5">
        <input
          ref={inputRef}
          data-testid="mobile-command-input"
          type="text"
          value={inputValue}
          onChange={(e) => setInputValue(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              e.preventDefault();
              submitCommand();
            }
          }}
          autoComplete="off"
          autoCorrect="on"
          spellCheck={false}
          placeholder="Type command…"
          className="min-w-0 flex-1 rounded border border-wc-default bg-wc-surface-input px-2 py-1.5 text-sm text-wc-text-primary placeholder:text-wc-text-muted outline-none focus:border-wc-accent"
        />
        <button
          data-testid="mobile-command-submit"
          onClick={submitCommand}
          disabled={!inputValue}
          className="shrink-0 rounded border border-wc-default bg-wc-surface-input px-3 py-1.5 text-xs font-medium text-wc-text-secondary transition active:bg-wc-accent-active disabled:opacity-40"
        >
          Send
        </button>
      </div>
      {/* Toolbar keys row */}
      <div className="flex items-center gap-1 overflow-x-auto px-2 py-1.5">
        {TOOLBAR_KEYS.map((key) => (
          <button
            key={key.label}
            data-testid={`toolbar-key-${slugify(key.label)}`}
            onPointerDown={(e) => e.preventDefault()}
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
    </div>
  );
}
