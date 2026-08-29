/**
 * @libraryId react-component-library:GlobalCommandSystem
 * @displayName GlobalCommandSystem
 * @description A scoped command system that keeps registration, keyboard invocation, discovery, execution, history, and recovery in one coherent workflow.
 * @version 1.0.3
 * @tags ["pattern","commands","keyboard","recovery","responsive","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.2";

/** @vrooliComponentSource patterns.global-command-system */
import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type CSSProperties,
  type ReactNode,
} from "react";
import {
  CommandPalette,
  type CommandPaletteStatus,
} from "@vrooli/react-component-library/CommandPalette/1.0.0";
import {
  createCommandRegistry,
  type Command,
  type CommandRegistry,
} from "@vrooli/react-component-library/CommandRegistry/1.0.0";
import {
  createShortcutRegistry,
  eventToShortcut,
  type ShortcutRegistry,
} from "@vrooli/react-component-library/ShortcutRegistry/1.0.0";

export interface GlobalCommandSystemProps {
  commands: Command[];
  registry?: CommandRegistry;
  shortcutRegistry?: ShortcutRegistry;
  open?: boolean;
  defaultOpen?: boolean;
  onOpenChange?: (open: boolean) => void;
  onExecuted?: (command: Command) => void | Promise<void>;
  status?: CommandPaletteStatus;
  errorMessage?: ReactNode;
  onRetry?: () => void | Promise<void>;
  triggerLabel?: string;
  shortcutLabel?: string;
  children?: ReactNode;
  className?: string;
  style?: CSSProperties;
}

const styles = `
[data-rcl-global-command-system] { display: grid; gap: var(--space-md, 24px); min-inline-size: 0; }
[data-rcl-global-command-trigger] { display: inline-flex; align-items: center; justify-content: space-between; gap: var(--space-md, 24px); inline-size: fit-content; min-block-size: var(--tap-target-min, 44px); max-inline-size: 100%; padding: var(--space-xs, 12px) var(--space-sm, 16px); border: var(--border-hairline, 1px) solid var(--color-border, #cbd5e1); border-radius: var(--radius-control, 0.375rem); background: var(--color-surface-raised, #ffffff); color: var(--color-foreground, #0f172a); box-shadow: var(--elev-raised, 0 1px 2px rgba(9, 18, 22, .06), 0 1px 3px rgba(9, 18, 22, .10)); font: var(--text-label, 500 var(--text-label-size) / var(--text-label-line) var(--font-sans)); cursor: pointer; transition: border-color var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2, 0, 0, 1)), background var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2, 0, 0, 1)), transform var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2, 0, 0, 1)); }
[data-rcl-global-command-trigger]:hover { border-color: var(--color-primary, #2563eb); background: color-mix(in srgb, var(--color-primary, #2563eb) 7%, var(--color-surface-raised, #ffffff)); transform: translateY(calc(var(--space-3xs, 4px) * -1)); }
[data-rcl-global-command-trigger]:active { transform: translateY(0) scale(.985); }
[data-rcl-global-command-trigger] kbd { padding: var(--space-3xs, 4px) var(--space-2xs, 8px); border: var(--border-hairline, 1px) solid var(--color-border, #cbd5e1); border-radius: var(--radius-control, 0.375rem); background: var(--color-surface-muted, #f1f5f9); color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)); white-space: nowrap; }



`;

export const GlobalCommandSystem = withClassName(function GlobalCommandSystem({
  commands,
  registry,
  shortcutRegistry,
  open,
  defaultOpen = false,
  onOpenChange,
  onExecuted,
  status = "default",
  errorMessage,
  onRetry,
  triggerLabel = "Open command palette",
  shortcutLabel = "⌘K",
  children,
  className,
  style,
}: GlobalCommandSystemProps) {
  const localRegistry = useMemo(() => registry ?? createCommandRegistry(), [registry]);
  const localShortcutRegistry = useMemo(
    () => shortcutRegistry ?? createShortcutRegistry(),
    [shortcutRegistry],
  );
  const [localOpen, setLocalOpen] = useState(defaultOpen);
  const resolvedOpen = open ?? localOpen;
  const setOpen = useCallback(
    (next: boolean) => {
      if (open === undefined) setLocalOpen(next);
      onOpenChange?.(next);
    },
    [onOpenChange, open],
  );

  useEffect(() => {
    const unregister = commands.map((command) => localRegistry.register(command));
    return () => unregister.forEach((remove) => remove());
  }, [commands, localRegistry]);

  useEffect(() => {
    const removeShortcut = localShortcutRegistry.register({
      id: "global-command-palette",
      keys: "mod+k",
      priority: 100,
      run: () => setOpen(true),
    });
    const handleKeyDown = (event: KeyboardEvent) => {
      const shortcut = localShortcutRegistry.resolve(eventToShortcut(event));
      if (!shortcut) return;
      event.preventDefault();
      void shortcut.run();
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => {
      removeShortcut();
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [localShortcutRegistry, setOpen]);

  return (
    <div data-rcl-global-command-system className={className} style={style}>
      <StyleSheet name="globalcommandsystem-1-0-2-1" css={styles} />
      {children}
      <button
        data-testid="patterns.global-command-system"
        type="button"
        data-rcl-global-command-trigger
        aria-haspopup="dialog"
        aria-expanded={resolvedOpen}
        onClick={() => setOpen(true)}
      >
        <span>{triggerLabel}</span>
        <kbd aria-hidden="true">{shortcutLabel}</kbd>
      </button>
      <span data-rcl-global-command-system-status role="status" aria-live="polite">
        {resolvedOpen ? "Command palette open" : "Command palette closed"}
      </span>
      <CommandPalette
        open={resolvedOpen}
        registry={localRegistry}
        onClose={() => setOpen(false)}
        onExecuted={onExecuted}
        status={status}
        errorMessage={errorMessage}
        onRetry={onRetry}
      />
    </div>
  );
});
