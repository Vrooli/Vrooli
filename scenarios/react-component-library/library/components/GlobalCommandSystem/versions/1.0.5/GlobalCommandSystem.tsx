/**
 * @libraryId react-component-library:GlobalCommandSystem
 * @displayName GlobalCommandSystem
 * @version 1.0.5
 * @tags ["pattern","commands","keyboard","recovery","responsive","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

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
} from "@vrooli/react-component-library/CommandPalette/1";
import {
  createCommandRegistry,
  type Command,
  type CommandRegistry,
} from "@vrooli/react-component-library/CommandRegistry/1";
import {
  createShortcutRegistry,
  eventToShortcut,
  type ShortcutRegistry,
} from "@vrooli/react-component-library/ShortcutRegistry/1";

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
[data-rcl-global-command-system] { display: grid; gap: var(--space-md); min-inline-size: 0; }
[data-rcl-global-command-trigger] { display: inline-flex; align-items: center; justify-content: space-between; gap: var(--space-md); inline-size: fit-content; min-block-size: var(--tap-target-min); max-inline-size: 100%; padding: var(--space-xs) var(--space-sm); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-control); background: var(--color-surface-raised); color: var(--color-foreground); box-shadow: var(--elev-raised); font: var(--text-label); cursor: pointer; transition: border-color var(--dur-quick) var(--ease-standard), background var(--dur-quick) var(--ease-standard), transform var(--dur-quick) var(--ease-standard); }
[data-rcl-global-command-trigger]:hover { border-color: var(--color-primary); background: color-mix(in srgb, var(--color-primary) 7%, var(--color-surface-raised)); transform: translateY(calc(var(--space-3xs) * -1)); }
[data-rcl-global-command-trigger]:active { transform: translateY(0) scale(.985); }
[data-rcl-global-command-trigger] kbd { padding: var(--space-3xs) var(--space-2xs); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-control); background: var(--color-surface-muted); color: var(--color-muted-foreground); font: var(--text-caption); white-space: nowrap; }



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
      <StyleSheet libraryId="react-component-library:GlobalCommandSystem" version="1.0.5" css={styles} />
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
