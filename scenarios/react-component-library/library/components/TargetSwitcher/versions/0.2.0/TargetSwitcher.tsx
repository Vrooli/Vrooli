/**
 * @libraryId react-component-library:TargetSwitcher
 * @displayName Target Switcher
 * @description A compact target identity control with an anchored fleet menu, presence details, keyboard navigation, and an optional target-linking action.
 * @version 0.2.0
 * @tags ["navigation","target","fleet","overlay"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { Popover, PopoverParts } from "@vrooli/react-component-library/Popover/1";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { useEffect, useRef, useState, type CSSProperties, type KeyboardEvent } from "react";

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
[data-rcl-target-switcher-trigger] { display: inline-flex; min-block-size: var(--tap-target-min); max-inline-size: min(14rem, 100%); min-inline-size: 0; align-items: center; gap: .45rem; border: 1px solid var(--color-border); border-radius: var(--radius-control); background: var(--color-surface); color: var(--color-foreground); padding: .4rem .6rem; font: var(--text-label); cursor: pointer; transition: background-color var(--dur-quick) var(--ease-standard), border-color var(--dur-quick) var(--ease-standard), color var(--dur-quick) var(--ease-standard), box-shadow var(--dur-quick) var(--ease-standard); }
[data-rcl-target-switcher-trigger]:hover, [data-rcl-target-switcher-trigger][aria-expanded="true"] { border-color: var(--color-border-strong); background: var(--color-surface-muted); color: var(--color-foreground); }
[data-rcl-target-switcher-trigger]:focus-visible { outline: none; box-shadow: var(--focus-ring); }
[data-rcl-target-switcher-dot] { inline-size: .45rem; block-size: .45rem; flex: 0 0 auto; border-radius: 50%; background: var(--color-muted-foreground); box-shadow: 0 0 0 .18rem color-mix(in srgb, currentColor 12%, transparent); }
[data-rcl-target-switcher-dot][data-tone="local"], [data-rcl-target-switcher-dot][data-tone="success"] { background: var(--color-success); }
[data-rcl-target-switcher-dot][data-tone="warning"] { background: var(--color-warning); }
[data-rcl-target-switcher-dot][data-tone="danger"] { background: var(--color-danger); }
[data-rcl-target-switcher-label] { min-inline-size: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
[data-rcl-target-switcher-chevron] { inline-size: .5rem; block-size: .5rem; flex: 0 0 auto; border-inline-end: var(--border-strong) solid currentColor; border-block-end: var(--border-strong) solid currentColor; transform: translateY(-.12rem) rotate(45deg); transition: transform var(--dur-quick) var(--ease-standard); }
[data-rcl-target-switcher-trigger][aria-expanded="true"] [data-rcl-target-switcher-chevron] { transform: translateY(.12rem) rotate(225deg); }
[data-rcl-popover-content][data-rcl-target-switcher-content] { inline-size: min(max(var(--rcl-target-switcher-width, 14rem), 18rem), calc(100vw - 1rem)); max-block-size: none; overflow: visible; }
[data-rcl-target-switcher-menu] { display: grid; box-sizing: border-box; inline-size: 100%; padding: 0; }
[data-rcl-target-switcher-summary] { display: block; min-inline-size: 0; }
[data-rcl-target-switcher-eyebrow] { display: block; border-block-end: 1px solid var(--color-border); color: var(--color-muted-foreground); padding: .55rem .8rem; font-family: var(--font-mono); font-size: .63rem; letter-spacing: .14em; text-transform: uppercase; }
[data-rcl-target-switcher-summary-title], [data-rcl-target-switcher-summary-description], [data-rcl-target-switcher-status] { display: none; }
[data-rcl-target-switcher-status-value] { display: inline-flex; min-inline-size: 0; align-items: center; gap: var(--space-2xs); color: var(--color-foreground); font-weight: 650; text-align: end; }
[data-rcl-target-switcher-options] { display: grid; }
[data-rcl-target-switcher-options][data-scrollable="true"] { max-block-size: min(60vh, 22rem); overflow-y: auto; }
[data-rcl-target-switcher-option] { display: flex; width: 100%; min-block-size: var(--tap-target-min); align-items: center; gap: .55rem; border: 0; border-radius: 0; background: transparent; color: var(--color-foreground); padding: .45rem .65rem; text-align: start; cursor: pointer; }
[data-rcl-target-switcher-option]:hover, [data-rcl-target-switcher-option][data-active="true"] { border-color: transparent; background: var(--color-surface-muted); }
[data-rcl-target-switcher-option]:focus-visible { outline: none; box-shadow: var(--focus-ring); }
[data-rcl-target-switcher-option][aria-selected="true"] { border-color: color-mix(in srgb, var(--color-primary) 32%, var(--color-border)); background: color-mix(in srgb, var(--color-primary) 10%, var(--color-surface-muted)); }
[data-rcl-target-switcher-option][aria-disabled="true"] { cursor: not-allowed; opacity: var(--opacity-disabled); }
[data-rcl-target-switcher-option-body] { display: flex; min-inline-size: 0; flex: 1; flex-direction: column; gap: .1rem; }
[data-rcl-target-switcher-option-name] { min-inline-size: 0; overflow: hidden; font-size: var(--text-body-sm-size); font-weight: 600; text-overflow: ellipsis; white-space: nowrap; }
[data-rcl-target-switcher-option-meta] { min-inline-size: 0; overflow: hidden; color: var(--color-muted-foreground); font-family: var(--font-mono); font-size: .68rem; text-overflow: ellipsis; white-space: nowrap; }
[data-rcl-target-switcher-option-badge] { max-inline-size: 8rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; border: 1px solid var(--color-border-strong); border-radius: var(--radius-control); color: var(--color-muted-foreground); padding: .1rem .4rem; font: var(--text-caption); }
[data-rcl-target-switcher-check] { color: var(--color-primary); font-weight: 800; }
[data-rcl-target-switcher-add] { display: inline-flex; min-block-size: var(--tap-target-min); align-items: center; gap: .45rem; border-block-start: 1px solid var(--color-border); background: transparent; color: var(--color-primary); padding: .6rem .8rem; font: var(--text-label); cursor: pointer; text-align: start; }
[data-rcl-target-switcher-add]:hover { color: var(--color-primary-strong); }
[data-rcl-target-switcher-add]:focus-visible { outline: none; box-shadow: var(--focus-ring); }
@media (max-width: 38rem) { [data-rcl-target-switcher-trigger] { max-inline-size: 8rem; } [data-rcl-popover-content][data-rcl-target-switcher-content] { inline-size: min(max(var(--rcl-target-switcher-width, 8rem), 16rem), calc(100vw - 1rem)); } }
`;

export function TargetSwitcher({
  value,
  options,
  onValueChange,
  label = "Change target",
  eyebrow = "Viewing",
  description: _description,
  statusLabel: _statusLabel = "Status",
  statusValue: _statusValue,
  statusTone: _statusTone,
  addTargetLabel,
  onAddTarget,
  testId = "navigation.target-switcher",
  className,
  triggerClassName,
}: TargetSwitcherProps) {
  const [open, setOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(0);
  const [triggerWidth, setTriggerWidth] = useState<number>();
  const triggerRef = useRef<HTMLButtonElement | null>(null);
  const optionRefs = useRef<Array<HTMLButtonElement | null>>([]);
  const selected = options.find((option) => option.id === value) ?? options[0];

  useEffect(() => {
    const trigger = triggerRef.current;
    if (!trigger) return;
    const updateWidth = () => setTriggerWidth(Math.round(trigger.getBoundingClientRect().width));
    updateWidth();
    if (typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(updateWidth);
    observer.observe(trigger);
    return () => observer.disconnect();
  }, []);

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
    <div data-rcl-target-switcher className={className}>
      <StyleSheet
        libraryId="react-component-library:TargetSwitcher"
        version="1.0.0"
        css={styleSheet}
      />
      <Popover open={open} onOpenChange={setOpen} placement="bottom-end" responsive="none">
        <PopoverParts.Trigger asChild>
          <button
            ref={triggerRef}
            type="button"
            className={triggerClassName}
            data-testid={testId}
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
        <PopoverParts.Content
          role="dialog"
          aria-label={label}
          initialFocus="none"
          data-rcl-target-switcher-content
          style={
            (triggerWidth ? { "--rcl-target-switcher-width": `${triggerWidth}px` } : undefined) as
              | CSSProperties
              | undefined
          }
        >
          <div data-rcl-target-switcher-menu>
            <div data-rcl-target-switcher-summary>
              <span data-rcl-target-switcher-eyebrow>{eyebrow}</span>
            </div>
            <div
              data-rcl-target-switcher-options
              data-scrollable={options.length > 8 ? "true" : "false"}
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
