/**
 * @libraryId react-component-library:CommandPalette
 * @displayName CommandPalette
 * @version 1.1.9
 * @tags ["overlay","command","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource overlays.command-palette */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import {
  useEffect,
  useMemo,
  useRef,
  useState,
  useSyncExternalStore,
  type CSSProperties,
  type ReactNode,
} from "react";
import {
  createCommandRegistry,
  type Command,
  type CommandRegistry,
} from "@vrooli/react-component-library/CommandRegistry/1";
import { SearchInput } from "@vrooli/react-component-library/SearchInput/1";
import { Portal } from "@vrooli/react-component-library/Portal/1";
import { useOverlaySurface } from "@vrooli/react-component-library/useOverlaySurface/1";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1";

export type CommandPaletteStatus = "default" | "loading" | "empty" | "request-error" | "retry";

export interface CommandPaletteProps {
  open?: boolean;
  onClose?: () => void;
  registry?: CommandRegistry;
  commands?: Command[];
  status?: CommandPaletteStatus;
  errorMessage?: ReactNode;
  onRetry?: () => void | Promise<void>;
  onExecuted?: (command: Command) => void | Promise<void>;
  title?: string;
  description?: string;
  placeholder?: string;
  className?: string;
  style?: CSSProperties;
}

const styles = `
[data-rcl-command-palette] { position: fixed; inset: 0; z-index: var(--layer-modal, 400); display: grid; align-items: start; justify-items: center; padding: clamp(var(--space-lg, 32px), 10vh, var(--space-4xl, 80px)) var(--space-md, 24px); color: var(--color-foreground, #0f172a); }
[data-rcl-command-palette-backdrop] { position: absolute; inset: 0; border: 0; background: color-mix(in srgb, var(--color-overlay, var(--color-shell)) 48%, transparent); cursor: default; }
[data-rcl-command-palette-panel] { position: relative; display: grid; grid-template-rows: auto auto minmax(0, 1fr) auto; min-block-size: 0; inline-size: min(100%, 42rem); max-block-size: min(calc(100dvh - var(--space-xl, 40px)), 44rem); overflow: hidden; border: var(--border-hairline, 1px) solid var(--color-border-strong, color-mix(in srgb, var(--color-border) 72%, var(--color-foreground))); border-radius: var(--radius-panel, 0.5rem); background: var(--color-surface-raised, #ffffff); box-shadow: var(--elev-overlay, 0 2px 4px rgba(9, 18, 22, .06), 0 4px 12px rgba(9, 18, 22, .10)); animation: rcl-command-palette-enter var(--dur-enter, var(--dur-quick)) var(--ease-standard, cubic-bezier(.2, 0, 0, 1)); }
[data-rcl-command-palette-header] { display: grid; gap: var(--space-2xs, 8px); padding: var(--space-lg, 32px) var(--space-lg, 32px) var(--space-sm, 16px); border-block-end: var(--border-hairline, 1px) solid var(--color-border, #cbd5e1); background: linear-gradient(135deg, color-mix(in srgb, var(--color-primary, #2563eb) 8%, var(--color-surface-raised, #ffffff)), var(--color-surface-raised, #ffffff)); }
[data-rcl-command-palette-eyebrow] { color: var(--color-primary, #2563eb); font: var(--text-overline, 700 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)); letter-spacing: .12em; text-transform: uppercase; }
[data-rcl-command-palette-title] { margin: 0; font: var(--text-title, 700 var(--text-title-size) / var(--text-title-line) var(--font-sans)); letter-spacing: var(--text-title-tracking, -.01em); }
[data-rcl-command-palette-description] { margin: 0; color: var(--color-muted-foreground, #64748b); font: var(--text-body, 400 var(--text-body-size) / var(--text-body-line) var(--font-sans)); }
[data-rcl-command-palette-search] { display: grid; grid-template-columns: minmax(0, 1fr); padding: var(--space-sm, 16px) var(--space-lg, 32px); border-block-end: var(--border-hairline, 1px) solid var(--color-border, #cbd5e1); }
[data-rcl-command-palette-search] label { inline-size: 100%; }
[data-rcl-command-palette-search] label > span { position: absolute; inline-size: 1px; block-size: 1px; overflow: hidden; clip-path: inset(50%); white-space: nowrap; }
[data-rcl-command-palette-search] input { min-block-size: var(--tap-target-min, 44px); border-color: var(--color-border-strong, color-mix(in srgb, var(--color-border) 72%, var(--color-foreground))); box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--color-primary, #2563eb) 8%, transparent); }
[data-rcl-command-palette-list] { display: grid; align-content: start; gap: var(--space-2xs, 8px); min-block-size: 0; overflow: auto; padding: var(--space-sm, 16px); overscroll-behavior: contain; }
[data-rcl-command-palette-group] { display: grid; gap: var(--space-3xs, 4px); }
[data-rcl-command-palette-group-label] { padding: var(--space-2xs, 8px) var(--space-xs, 12px); color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)); letter-spacing: .04em; text-transform: uppercase; }
[data-rcl-command-palette-option] { display: grid; grid-template-columns: minmax(0, 1fr) auto; align-items: center; gap: var(--space-sm, 16px); min-block-size: var(--tap-target-min, 44px); padding: var(--space-xs, 12px) var(--space-sm, 16px); border: var(--border-hairline, 1px) solid transparent; border-radius: var(--radius-control, 0.375rem); background: transparent; color: inherit; text-align: start; cursor: pointer; font: inherit; transition: background var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2, 0, 0, 1)), border-color var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2, 0, 0, 1)), transform var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2, 0, 0, 1)); }
[data-rcl-command-palette-option]:hover, [data-rcl-command-palette-option][aria-selected="true"] { border-color: color-mix(in srgb, var(--color-primary, #2563eb) 36%, var(--color-border, #cbd5e1)); background: color-mix(in srgb, var(--color-primary, #2563eb) 9%, var(--color-surface-raised, #ffffff)); }
[data-rcl-command-palette-option][aria-selected="true"] { transform: translateX(var(--space-3xs, 4px)); }
[data-rcl-command-palette-option]:focus-visible, [data-rcl-command-palette-close]:focus-visible, [data-rcl-command-palette-retry]:focus-visible { outline: 3px solid color-mix(in srgb, var(--color-focus, #2563eb) 38%, transparent); outline-offset: 2px; }
[data-rcl-command-palette-option-copy] { display: grid; gap: var(--space-3xs, 4px); min-inline-size: 0; }
[data-rcl-command-palette-option-label] { overflow-wrap: anywhere; font: var(--text-label, 500 var(--text-label-size) / var(--text-label-line) var(--font-sans)); }
[data-rcl-command-palette-option-description] { overflow-wrap: anywhere; color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)); }
[data-rcl-command-palette-option-meta] { display: inline-flex; align-items: center; gap: var(--space-xs, 12px); color: var(--color-muted-foreground, #64748b); }
[data-rcl-command-palette-option-group] { font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)); }
[data-rcl-command-palette-option-shortcut], [data-rcl-command-palette-key] { padding: var(--space-3xs, 4px) var(--space-2xs, 8px); border: var(--border-hairline, 1px) solid var(--color-border, #cbd5e1); border-radius: var(--radius-control, 0.375rem); background: var(--color-surface-muted, #f1f5f9); color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)); white-space: nowrap; }
[data-rcl-command-palette-state] { display: grid; place-items: center; gap: var(--space-xs, 12px); min-block-size: 12rem; padding: var(--space-xl, 40px); color: var(--color-muted-foreground, #64748b); text-align: center; font: var(--text-body, 400 var(--text-body-size) / var(--text-body-line) var(--font-sans)); }
[data-rcl-command-palette-state] strong { color: var(--color-foreground, #0f172a); font: var(--text-subtitle, 600 var(--text-subheading-size) / var(--text-subheading-line) var(--font-sans)); }
[data-rcl-command-palette-state][data-tone="danger"] { color: var(--color-danger, #dc2626); }
[data-rcl-command-palette-state] button { min-block-size: var(--tap-target-min, 44px); padding-inline: var(--space-sm, 16px); border: var(--border-hairline, 1px) solid var(--color-primary, #2563eb); border-radius: var(--radius-control, 0.375rem); background: var(--color-primary, #2563eb); color: var(--color-primary-foreground, #ffffff); font: var(--text-label, 500 var(--text-label-size) / var(--text-label-line) var(--font-sans)); cursor: pointer; }
[data-rcl-command-palette-footer] { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: var(--space-sm, 16px); padding: var(--space-sm, 16px) var(--space-lg, 32px); border-block-start: var(--border-hairline, 1px) solid var(--color-border, #cbd5e1); color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)); }
[data-rcl-command-palette-footer-hints] { display: inline-flex; flex-wrap: wrap; gap: var(--space-xs, 12px); }
[data-rcl-command-palette-close] { min-block-size: var(--tap-target-min, 44px); padding-inline: var(--space-sm, 16px); border: var(--border-hairline, 1px) solid var(--color-border, #cbd5e1); border-radius: var(--radius-control, 0.375rem); background: transparent; color: inherit; font: var(--text-label, 500 var(--text-label-size) / var(--text-label-line) var(--font-sans)); cursor: pointer; }
@keyframes rcl-command-palette-enter { from { opacity: 0; transform: translateY(calc(var(--space-sm, 16px) * -1)) scale(.985); } to { opacity: 1; transform: translateY(0) scale(1); } }
@media (max-width: 34rem) { [data-rcl-command-palette] { align-items: end; padding: 0; } [data-rcl-command-palette-panel] { inline-size: 100%; max-block-size: min(calc(100dvh - var(--space-sm, 16px)), 48rem); border-block-end: 0; border-radius: var(--radius-panel, 0.5rem) var(--radius-panel, 0.5rem) 0 0; } [data-rcl-command-palette-header], [data-rcl-command-palette-search] { padding-inline: var(--space-md, 24px); } [data-rcl-command-palette-footer] { padding-inline: var(--space-md, 24px); } }
@media (prefers-reduced-motion: reduce) { [data-rcl-command-palette-panel], [data-rcl-command-palette-option] { animation: none; transition: none; } [data-rcl-command-palette-option][aria-selected="true"] { transform: none; } }
@media (forced-colors: active) { [data-rcl-command-palette-panel], [data-rcl-command-palette-option], [data-rcl-command-palette-key], [data-rcl-command-palette-option-shortcut] { border-color: CanvasText; background: Canvas; color: CanvasText; box-shadow: none; } [data-rcl-command-palette-backdrop] { background: CanvasText; opacity: .5; } }
`;

export const CommandPalette = withClassName(function CommandPalette({
  open = false,
  onClose,
  registry,
  commands = [],
  status = "default",
  errorMessage = "Commands could not be refreshed. Your last available actions remain safe to retry.",
  onRetry,
  onExecuted,
  title,
  description,
  placeholder,
  className,
  style,
}: CommandPaletteProps) {
  useLibraryStyleSheet("command-palette-1.1.0", styles);
  const libraryStrings = useStrings();
  placeholder =
    placeholder ?? libraryStrings("overlays.command-palette.search-commands", "Search commands…");
  description =
    description ??
    libraryStrings(
      "overlays.command-palette.search-actions-across-this-workspace-placeholder",
      "Search actions across this workspace.",
    );
  title = title ?? libraryStrings("overlays.command-palette.command-palette", "Command palette");
  const strings = useStrings();
  const localRegistry = useMemo(() => registry ?? createCommandRegistry(), [registry]);
  useEffect(() => {
    if (registry) return undefined;
    const unregister = commands.map((command) => localRegistry.register(command));
    return () => unregister.forEach((remove) => remove());
  }, [commands, localRegistry, registry]);

  useSyncExternalStore(
    localRegistry.subscribe,
    localRegistry.getSnapshot,
    localRegistry.getSnapshot,
  );
  const [query, setQuery] = useState("");
  const [activeId, setActiveId] = useState<string>();
  const [execution, setExecution] = useState<"idle" | "running" | "error">("idle");
  const [executionError, setExecutionError] = useState<ReactNode>();
  const searchRef = useRef<HTMLInputElement>(null);
  const overlay = useOverlaySurface({
    open,
    onOpenChange: (next) => {
      if (!next) onClose?.();
    },
    modal: true,
    kind: "dialog",
    initialFocusRef: searchRef,
  });

  const filtered = (() => {
    const matches = localRegistry.search(query);
    if (query.trim()) return matches;
    const recent = localRegistry.getRecent();
    const recentIds = new Set(recent.map((command) => command.id));
    return [...recent, ...matches.filter((command) => !recentIds.has(command.id))];
  })();
  const grouped = useMemo(() => {
    const groups = new Map<string, Command[]>();
    filtered.forEach((command) => {
      const group = command.group ?? "Actions";
      groups.set(group, [...(groups.get(group) ?? []), command]);
    });
    return [...groups.entries()];
  }, [filtered]);

  useEffect(() => {
    const first = filtered.find((command) => !command.disabled);
    if (!activeId || !filtered.some((command) => command.id === activeId)) {
      setActiveId(first?.id);
    }
  }, [activeId, filtered]);

  if (!overlay.present) return null;

  const orderedCommands = filtered.filter((command) => !command.disabled);
  const moveActive = (direction: 1 | -1) => {
    if (!orderedCommands.length) return;
    const currentIndex = orderedCommands.findIndex((command) => command.id === activeId);
    const nextIndex = (currentIndex + direction + orderedCommands.length) % orderedCommands.length;
    setActiveId(orderedCommands[nextIndex]?.id);
  };
  const execute = async (command: Command) => {
    if (command.disabled || execution === "running") return;
    setExecution("running");
    setExecutionError(undefined);
    try {
      await localRegistry.execute(command.id, { query });
      await onExecuted?.(command);
      setExecution("idle");
    } catch (error) {
      setExecution("error");
      setExecutionError(error instanceof Error ? error.message : "The command could not run.");
    }
  };
  const activeCommand = orderedCommands.find((command) => command.id === activeId);
  const state = execution === "error" ? "request-error" : status;

  return (
    <Portal>
      <div
        data-rcl-command-palette
        className={className}
        style={style}
        data-status={state}
        data-state={overlay.state}
      >
        <button
          data-testid="overlays.command-palette.backdrop"
          type="button"
          data-rcl-command-palette-backdrop
          aria-label={strings(
            "overlays.command-palette.close-command-palette",
            "Close command palette",
          )}
          {...overlay.backdropProps}
        />
        <section
          ref={overlay.surfaceRef}
          data-testid="overlays.command-palette"
          data-rcl-command-palette-panel
          role="dialog"
          aria-modal="true"
          aria-labelledby="rcl-command-palette-title"
          aria-describedby="rcl-command-palette-description"
        >
          <header data-rcl-command-palette-header>
            <span data-rcl-command-palette-eyebrow>
              {strings("overlays.command-palette.command-center", "Command center")}
            </span>
            <h2 id="rcl-command-palette-title" data-rcl-command-palette-title>
              {title}
            </h2>
            <p id="rcl-command-palette-description" data-rcl-command-palette-description>
              {description}
            </p>
          </header>
          <div data-rcl-command-palette-search>
            <SearchInput
              ref={searchRef}
              aria-label={strings(
                "overlays.command-palette.search-commands.search-commands",
                "Search commands",
              )}
              role="combobox"
              aria-controls="rcl-command-palette-list"
              aria-expanded="true"
              aria-activedescendant={activeId ? `rcl-command-${activeId}` : undefined}
              placeholder={placeholder}
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === "ArrowDown") {
                  event.preventDefault();
                  moveActive(1);
                } else if (event.key === "ArrowUp") {
                  event.preventDefault();
                  moveActive(-1);
                } else if (event.key === "Enter" && activeCommand) {
                  event.preventDefault();
                  void execute(activeCommand);
                } else if (event.key === "Escape") {
                  event.preventDefault();
                  overlay.close();
                }
              }}
              style={{ width: "100%" }}
            />
          </div>
          {state === "loading" ? (
            <div data-rcl-command-palette-state role="status">
              <strong>
                {strings("overlays.command-palette.loading-commands", "Loading commands")}
              </strong>
              <span>
                {strings(
                  "overlays.command-palette.preparing-actions-for-this-workspace-span-div-st",
                  "Preparing actions for this workspace…",
                )}
              </span>
            </div>
          ) : state === "request-error" || state === "retry" ? (
            <div data-rcl-command-palette-state data-tone="danger" role="alert">
              <strong>
                {strings("overlays.command-palette.commands-need-a-retry", "Commands need a retry")}
              </strong>
              <span>{executionError ?? errorMessage}</span>
              {onRetry ? (
                <button
                  data-testid="overlays.command-palette.retry"
                  type="button"
                  data-rcl-command-palette-retry
                  onClick={() => void onRetry()}
                >
                  {strings("overlays.command-palette.try-again", "Try again")}
                </button>
              ) : null}
            </div>
          ) : state === "empty" || !grouped.length ? (
            <div data-rcl-command-palette-state role="status">
              <strong>
                {strings("overlays.command-palette.no-matching-commands", "No matching commands")}
              </strong>
              <span>
                {strings(
                  "overlays.command-palette.try-a-shorter-phrase-or-clear-the-search-span-di",
                  "Try a shorter phrase or clear the search.",
                )}
              </span>
            </div>
          ) : (
            <div
              id="rcl-command-palette-list"
              data-rcl-command-palette-list
              role="listbox"
              aria-label={strings(
                "overlays.command-palette.available-commands",
                "Available commands",
              )}
            >
              {grouped.map(([group, groupCommands]) => (
                <div data-rcl-command-palette-group key={group}>
                  <div data-rcl-command-palette-group-label>{group}</div>
                  {groupCommands.map((command) => (
                    <button
                      data-testid={`overlays.command-palette.option.${command.id}`}
                      key={command.id}
                      id={`rcl-command-${command.id}`}
                      type="button"
                      role="option"
                      aria-selected={command.id === activeId}
                      aria-disabled={command.disabled || undefined}
                      disabled={command.disabled}
                      data-rcl-command-palette-option
                      onMouseEnter={() => setActiveId(command.id)}
                      onClick={() => void execute(command)}
                    >
                      <span data-rcl-command-palette-option-copy>
                        <span data-rcl-command-palette-option-label>{command.label}</span>
                        {command.description ? (
                          <span data-rcl-command-palette-option-description>
                            {command.description}
                          </span>
                        ) : null}
                      </span>
                      <span data-rcl-command-palette-option-meta>
                        {command.group ? (
                          <span data-rcl-command-palette-option-group>{command.group}</span>
                        ) : null}
                        {command.shortcut ? (
                          <kbd data-rcl-command-palette-option-shortcut>{command.shortcut}</kbd>
                        ) : null}
                      </span>
                    </button>
                  ))}
                </div>
              ))}
            </div>
          )}
          <footer data-rcl-command-palette-footer>
            <span data-rcl-command-palette-footer-hints>
              <span>
                <kbd data-rcl-command-palette-key>↑↓</kbd>
                {strings("overlays.command-palette.navigate", "Navigate")}
              </span>
              <span>
                <kbd data-rcl-command-palette-key>↵</kbd>
                {strings("overlays.command-palette.run", "Run")}
              </span>
              <span>
                <kbd data-rcl-command-palette-key>
                  {strings("overlays.command-palette.esc", "Esc")}
                </kbd>
                {strings("overlays.command-palette.close", "Close")}
              </span>
            </span>
            <button
              data-testid="overlays.command-palette.close"
              type="button"
              data-rcl-command-palette-close
              onClick={overlay.close}
            >
              {strings("overlays.command-palette.close", "Close")}
            </button>
          </footer>
        </section>
      </div>
    </Portal>
  );
});
