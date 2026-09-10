/**
 * @libraryId react-component-library:TargetSwitcher
 * @displayName Target Switcher
 * @description A compact target identity control with an anchored fleet menu, presence details, keyboard navigation, and an optional target-linking action.
 * @version 0.1.1
 * @tags ["navigation","target","fleet","overlay"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { Popover, PopoverParts } from "@vrooli/react-component-library/Popover/1";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { useEffect, useRef, useState, type KeyboardEvent } from "react";

export type TargetStatusTone = "local" | "success" | "warning" | "danger" | "neutral";

export interface TargetSwitcherOption {
  id: string;
  label: string;
  meta?: string;
  description?: string;
  status?: string;
  statusTone?: TargetStatusTone;
  badge?: string;
  disabled?: boolean;
}

export interface TargetSwitcherProps {
  value: string;
  options: TargetSwitcherOption[];
  onValueChange: (value: string) => void;
  label?: string;
  eyebrow?: string;
  description?: string;
  statusLabel?: string;
  statusValue?: string;
  statusTone?: TargetStatusTone;
  addTargetLabel?: string;
  onAddTarget?: () => void;
  testId?: string;
  className?: string;
  triggerClassName?: string;
}

const styleSheet = `
[data-rcl-target-switcher] { position: relative; min-inline-size: 0; }
[data-rcl-target-switcher-trigger] { display: inline-flex; min-block-size: var(--tap-target-min); max-inline-size: min(15rem, 100%); min-inline-size: 0; align-items: center; gap: var(--space-xs); border: 1px solid transparent; border-radius: 999px; background: var(--color-surface-muted); color: var(--color-muted-foreground); padding: var(--space-2xs) var(--space-xs) var(--space-2xs) var(--space-sm); font: var(--text-label); cursor: pointer; transition: background-color var(--dur-quick) var(--ease-standard), border-color var(--dur-quick) var(--ease-standard), color var(--dur-quick) var(--ease-standard), box-shadow var(--dur-quick) var(--ease-standard); }
[data-rcl-target-switcher-trigger]:hover, [data-rcl-target-switcher-trigger][aria-expanded="true"] { border-color: var(--color-border); background: var(--color-surface-raised); color: var(--color-foreground); }
[data-rcl-target-switcher-trigger]:focus-visible { outline: none; box-shadow: var(--focus-ring); }
[data-rcl-target-switcher-dot] { inline-size: .45rem; block-size: .45rem; flex: 0 0 auto; border-radius: 50%; background: var(--color-muted); box-shadow: 0 0 0 .18rem color-mix(in srgb, currentColor 12%, transparent); }
[data-rcl-target-switcher-dot][data-tone="local"], [data-rcl-target-switcher-dot][data-tone="success"] { background: var(--color-success); }
[data-rcl-target-switcher-dot][data-tone="warning"] { background: var(--color-warning); }
[data-rcl-target-switcher-dot][data-tone="danger"] { background: var(--color-danger); }
[data-rcl-target-switcher-label] { min-inline-size: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
[data-rcl-target-switcher-chevron] { inline-size: .5rem; block-size: .5rem; flex: 0 0 auto; border-inline-end: var(--border-strong) solid currentColor; border-block-end: var(--border-strong) solid currentColor; transform: translateY(-.12rem) rotate(45deg); transition: transform var(--dur-quick) var(--ease-standard); }
[data-rcl-target-switcher-trigger][aria-expanded="true"] [data-rcl-target-switcher-chevron] { transform: translateY(.12rem) rotate(225deg); }
[data-rcl-target-switcher-menu] { display: grid; gap: var(--space-sm); box-sizing: border-box; inline-size: min(25rem, calc(100vw - (var(--space-lg) * 2))); padding: var(--space-md); }
[data-rcl-target-switcher-summary] { display: grid; min-inline-size: 0; gap: var(--space-3xs); }
[data-rcl-target-switcher-eyebrow] { color: var(--color-muted-foreground); font: var(--text-overline); letter-spacing: .08em; text-transform: uppercase; }
[data-rcl-target-switcher-summary-title] { min-inline-size: 0; overflow-wrap: anywhere; color: var(--color-foreground-strong); font: var(--text-heading-sm); }
[data-rcl-target-switcher-summary-description] { margin: 0; color: var(--color-muted-foreground); font: var(--text-body-sm); line-height: 1.45; }
[data-rcl-target-switcher-status] { display: flex; min-inline-size: 0; align-items: center; justify-content: space-between; gap: var(--space-md); border-block-start: 1px solid var(--color-border-muted); padding-block-start: var(--space-sm); color: var(--color-muted-foreground); font: var(--text-body-sm); }
[data-rcl-target-switcher-status-value] { display: inline-flex; min-inline-size: 0; align-items: center; gap: var(--space-2xs); color: var(--color-foreground); font-weight: 650; text-align: end; }
[data-rcl-target-switcher-options] { display: grid; max-block-size: min(18rem, 42vh); gap: var(--space-3xs); overflow: auto; border-block-start: 1px solid var(--color-border-muted); padding-block-start: var(--space-sm); }
[data-rcl-target-switcher-option] { display: grid; grid-template-columns: auto minmax(0, 1fr) auto; min-block-size: var(--tap-target-min); align-items: center; gap: var(--space-xs); border: 1px solid transparent; border-radius: var(--radius-control); background: transparent; color: var(--color-foreground); padding: var(--space-xs); text-align: start; cursor: pointer; }
[data-rcl-target-switcher-option]:hover, [data-rcl-target-switcher-option][data-active="true"] { border-color: var(--color-border); background: var(--color-surface-muted); }
[data-rcl-target-switcher-option]:focus-visible { outline: none; box-shadow: var(--focus-ring); }
[data-rcl-target-switcher-option][aria-selected="true"] { border-color: color-mix(in srgb, var(--color-primary) 32%, var(--color-border)); background: color-mix(in srgb, var(--color-primary) 10%, var(--color-surface-muted)); }
[data-rcl-target-switcher-option][aria-disabled="true"] { cursor: not-allowed; opacity: var(--opacity-disabled); }
[data-rcl-target-switcher-option-body] { display: grid; min-inline-size: 0; gap: var(--space-3xs); }
[data-rcl-target-switcher-option-name] { min-inline-size: 0; overflow-wrap: anywhere; font-weight: 650; }
[data-rcl-target-switcher-option-meta] { min-inline-size: 0; overflow-wrap: anywhere; color: var(--color-muted-foreground); font: var(--text-caption); line-height: 1.35; }
[data-rcl-target-switcher-option-badge] { max-inline-size: 8rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; border: 1px solid var(--color-border-muted); border-radius: 999px; color: var(--color-muted-foreground); padding: var(--space-3xs) var(--space-2xs); font: var(--text-caption); }
[data-rcl-target-switcher-check] { color: var(--color-primary); font-weight: 800; }
[data-rcl-target-switcher-add] { display: inline-flex; min-block-size: var(--tap-target-min); align-items: center; border-block-start: 1px solid var(--color-border-muted); background: transparent; color: var(--color-primary); padding: var(--space-sm) var(--space-2xs) 0; font: var(--text-label); cursor: pointer; text-align: start; }
[data-rcl-target-switcher-add]:hover { color: var(--color-primary-strong); }
[data-rcl-target-switcher-add]:focus-visible { outline: none; box-shadow: var(--focus-ring); }
@media (max-width: 38rem) { [data-rcl-target-switcher-trigger] { max-inline-size: 8rem; } [data-rcl-target-switcher-menu] { inline-size: min(25rem, calc(100vw - (var(--space-sm) * 2))); padding: var(--space-sm); } }
`;

export function TargetSwitcher({
  value,
  options,
  onValueChange,
  label = "Change target",
  eyebrow = "Viewing",
  description,
  statusLabel = "Status",
  statusValue,
  statusTone,
  addTargetLabel,
  onAddTarget,
  testId = "navigation.target-switcher",
  className,
  triggerClassName,
}: TargetSwitcherProps) {
  const [open, setOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(0);
  const optionRefs = useRef<Array<HTMLButtonElement | null>>([]);
  const selected = options.find((option) => option.id === value) ?? options[0];

  useEffect(() => {
    if (!open) return;
    const index = Math.max(
      0,
      options.findIndex((option) => option.id === selected?.id),
    );
    setActiveIndex(index);
    requestAnimationFrame(() => optionRefs.current[index]?.focus());
  }, [open, options, selected?.id]);

  const choose = (option: TargetSwitcherOption) => {
    if (option.disabled) return;
    onValueChange(option.id);
    setOpen(false);
  };

  const focusOption = (index: number) => {
    setActiveIndex(index);
    requestAnimationFrame(() => optionRefs.current[index]?.focus());
  };

  const handleOptionKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (options.length === 0) return;
    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      const direction = event.key === "ArrowDown" ? 1 : -1;
      setActiveIndex((current) => {
        let next = current;
        for (let count = 0; count < options.length; count += 1) {
          next = (next + direction + options.length) % options.length;
          if (!options[next]?.disabled) {
            requestAnimationFrame(() => optionRefs.current[next]?.focus());
            return next;
          }
        }
        return current;
      });
    } else if (event.key === "Home" || event.key === "End") {
      event.preventDefault();
      const sequence = event.key === "Home" ? options : [...options].reverse();
      const offset = sequence.findIndex((option) => !option.disabled);
      if (offset >= 0) focusOption(event.key === "Home" ? offset : options.length - 1 - offset);
    } else if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      const option = options[activeIndex];
      if (option) choose(option);
    }
  };

  if (!selected) return null;

  return (
    <div data-rcl-target-switcher className={className} data-testid={testId}>
      <StyleSheet
        libraryId="react-component-library:TargetSwitcher"
        version="1.0.0"
        css={styleSheet}
      />
      <Popover open={open} onOpenChange={setOpen} placement="bottom-end" responsive="none">
        <PopoverParts.Trigger asChild>
          <button
            type="button"
            className={triggerClassName}
            data-rcl-target-switcher-trigger
            aria-label={label}
            aria-haspopup="dialog"
            aria-expanded={open}
            data-value={selected.id}
          >
            <span
              data-rcl-target-switcher-dot
              data-tone={selected.statusTone ?? "neutral"}
              aria-hidden="true"
            />
            <span data-rcl-target-switcher-label>{selected.label}</span>
            <span data-rcl-target-switcher-chevron aria-hidden="true" />
          </button>
        </PopoverParts.Trigger>
        <PopoverParts.Content role="dialog" aria-label={label} initialFocus="none">
          <div data-rcl-target-switcher-menu>
            <div data-rcl-target-switcher-summary>
              <span data-rcl-target-switcher-eyebrow>{eyebrow}</span>
              <strong data-rcl-target-switcher-summary-title>{selected.label}</strong>
              {(selected.description ?? description) && (
                <p data-rcl-target-switcher-summary-description>
                  {selected.description ?? description}
                </p>
              )}
            </div>
            {(statusValue ?? selected.status) && (
              <div data-rcl-target-switcher-status>
                <span>{statusLabel}</span>
                <strong data-rcl-target-switcher-status-value>
                  <span
                    data-rcl-target-switcher-dot
                    data-tone={statusTone ?? selected.statusTone ?? "neutral"}
                    aria-hidden="true"
                  />
                  {statusValue ?? selected.status}
                </strong>
              </div>
            )}
            <div
              data-rcl-target-switcher-options
              role="listbox"
              aria-label={label}
              onKeyDown={handleOptionKeyDown}
            >
              {options.map((option, index) => (
                <button
                  key={option.id}
                  ref={(element) => {
                    optionRefs.current[index] = element;
                  }}
                  type="button"
                  role="option"
                  aria-selected={option.id === selected.id}
                  aria-disabled={option.disabled || undefined}
                  tabIndex={index === activeIndex ? 0 : -1}
                  data-active={index === activeIndex}
                  data-rcl-target-switcher-option
                  onFocus={() => setActiveIndex(index)}
                  onClick={() => choose(option)}
                >
                  <span
                    data-rcl-target-switcher-dot
                    data-tone={option.statusTone ?? "neutral"}
                    aria-hidden="true"
                  />
                  <span data-rcl-target-switcher-option-body>
                    <span data-rcl-target-switcher-option-name>{option.label}</span>
                    {(option.meta ?? option.status) && (
                      <span data-rcl-target-switcher-option-meta>
                        {[option.meta, option.status].filter(Boolean).join(" · ")}
                      </span>
                    )}
                  </span>
                  {option.id === selected.id ? (
                    <span data-rcl-target-switcher-check aria-hidden="true">
                      ✓
                    </span>
                  ) : (
                    option.badge && (
                      <span data-rcl-target-switcher-option-badge>{option.badge}</span>
                    )
                  )}
                </button>
              ))}
            </div>
            {onAddTarget && addTargetLabel && (
              <button
                type="button"
                data-rcl-target-switcher-add
                onClick={() => {
                  setOpen(false);
                  onAddTarget();
                }}
              >
                {addTargetLabel}
              </button>
            )}
          </div>
        </PopoverParts.Content>
      </Popover>
    </div>
  );
}
