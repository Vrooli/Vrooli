/**
 * @libraryId react-component-library:GlobalCommandSystem
 * @displayName GlobalCommandSystem
 * @description A scoped command system that keeps registration, keyboard invocation, discovery, execution, history, and recovery in one coherent workflow.
 * @version 1.0.2
 * @tags ["pattern","commands","keyboard","recovery","responsive","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

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
[data-rcl-global-command-system] { display: grid; gap: var(--space-md, 1rem); min-inline-size: 0; }
[data-rcl-global-command-trigger] { display: inline-flex; align-items: center; justify-content: space-between; gap: var(--space-md, 1rem); inline-size: fit-content; min-block-size: var(--tap-target-min, 44px); max-inline-size: 100%; padding: var(--space-xs, .625rem) var(--space-sm, .75rem); border: var(--border-hairline, 1px) solid var(--color-border, #cbd5e1); border-radius: var(--radius-control, .625rem); background: var(--color-surface-raised, #fff); color: var(--color-foreground, #0f172a); box-shadow: var(--elev-raised, 0 3px 12px rgb(15 23 42 / .06)); font: var(--text-label, 650 .8125rem/1.2 system-ui, sans-serif); cursor: pointer; transition: border-color var(--dur-quick, 140ms) var(--ease-standard, ease), background var(--dur-quick, 140ms) var(--ease-standard, ease), transform var(--dur-quick, 140ms) var(--ease-standard, ease); }
[data-rcl-global-command-trigger]:hover { border-color: var(--color-primary, #2563eb); background: color-mix(in srgb, var(--color-primary, #2563eb) 7%, var(--color-surface-raised, #fff)); transform: translateY(calc(var(--space-3xs, .25rem) * -1)); }
[data-rcl-global-command-trigger]:active { transform: translateY(0) scale(.985); }
[data-rcl-global-command-trigger]:focus-visible { outline: 3px solid color-mix(in srgb, var(--color-focus, #2563eb) 38%, transparent); outline-offset: 2px; }
[data-rcl-global-command-trigger] kbd { padding: var(--space-3xs, .2rem) var(--space-2xs, .35rem); border: var(--border-hairline, 1px) solid var(--color-border, #cbd5e1); border-radius: var(--radius-control, .375rem); background: var(--color-surface-muted, #f1f5f9); color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 650 .6875rem/1.2 ui-monospace, monospace); white-space: nowrap; }
[data-rcl-global-command-system-status] { position: absolute; inline-size: 1px; block-size: 1px; overflow: hidden; clip-path: inset(50%); white-space: nowrap; }
@media (prefers-reduced-motion: reduce) { [data-rcl-global-command-trigger] { transition: none; } [data-rcl-global-command-trigger]:hover { transform: none; } }
@media (forced-colors: active) { [data-rcl-global-command-trigger], [data-rcl-global-command-trigger] kbd { border-color: CanvasText; background: Canvas; color: CanvasText; box-shadow: none; } }
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
      <style data-rcl-global-command-system-styles dangerouslySetInnerHTML={{ __html: styles }} />
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
