/**
 * @libraryId react-component-library:Select
 * @displayName Select
 * @description The single-selection field with native-like semantics, keyboard navigation, option groups, descriptions, async option loading, and responsive popover or sheet presentation.
 * @version 1.3.10
 * @tags []
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { Popover, PopoverParts } from "@vrooli/react-component-library/Popover/1";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import {
  forwardRef,
  useEffect,
  useId,
  useRef,
  useState,
  type ChangeEvent,
  type HTMLAttributes,
  type KeyboardEvent,
  type SelectHTMLAttributes,
} from "react";

export interface SelectOption {
  value: string;
  label: string;
  disabled?: boolean;
}

export interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
  options: SelectOption[];
  placeholder?: string;
  /** Value-only change callback for composed controls and typed adapters. */
  onValueChange?: (value: string) => void;
}

const styleSheet = `
[data-rcl-select-field] { position: relative; display: block; min-inline-size: 0; }
[data-rcl-select-trigger] { box-sizing: border-box; display: flex; inline-size: 100%; min-block-size: var(--tap-target-min); align-items: center; justify-content: space-between; gap: var(--space-sm); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-control); background: var(--color-field); color: var(--color-foreground); padding: var(--space-2xs) var(--space-sm); font: inherit; text-align: start; cursor: pointer; transition: border-color var(--dur-quick) var(--ease-standard), box-shadow var(--dur-quick) var(--ease-standard), background-color var(--dur-quick) var(--ease-standard); }
[data-rcl-select-trigger]:hover:not(:disabled), [data-rcl-select-trigger][aria-expanded="true"] { border-color: var(--color-primary); background: var(--color-surface-raised); }
[data-rcl-select-trigger]:focus-visible { outline: none; box-shadow: var(--focus-ring); }
[data-rcl-select-trigger][aria-invalid="true"] { border-color: var(--color-danger); }
[data-rcl-select-trigger][aria-disabled="true"] { cursor: not-allowed; opacity: var(--opacity-disabled); }
[data-rcl-select-value] { min-inline-size: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
[data-rcl-select-value][data-placeholder="true"] { color: var(--color-muted-foreground); }
[data-rcl-select-chevron] { inline-size: .5rem; block-size: .5rem; flex: 0 0 auto; border-inline-end: var(--border-strong) solid currentColor; border-block-end: var(--border-strong) solid currentColor; color: var(--color-muted-foreground); transform: translateY(-.12rem) rotate(45deg); transition: transform var(--dur-quick) var(--ease-standard); }
[data-rcl-select-trigger][aria-expanded="true"] [data-rcl-select-chevron] { transform: translateY(.12rem) rotate(225deg); }
[data-rcl-select-listbox] { display: grid; gap: var(--space-3xs); min-inline-size: 13rem; padding: var(--space-2xs); }
[data-rcl-select-option] { display: flex; min-block-size: var(--tap-target-min); align-items: center; gap: var(--space-xs); border: 0; border-radius: var(--radius-control); background: transparent; color: var(--color-foreground); padding: var(--space-xs) var(--space-sm); font: var(--text-body-sm); text-align: start; cursor: pointer; }
[data-rcl-select-option]:hover, [data-rcl-select-option][data-highlighted="true"] { background: var(--color-surface-muted); outline: none; }
[data-rcl-select-option][aria-selected="true"] { background: color-mix(in srgb, var(--color-primary) 11%, var(--color-surface-muted)); color: var(--color-foreground-strong); font-weight: 650; }
[data-rcl-select-option][data-disabled="true"] { cursor: not-allowed; opacity: var(--opacity-disabled); }
[data-rcl-select-check] { inline-size: 1rem; flex: 0 0 1rem; color: var(--color-primary); font-weight: 700; }
[data-rcl-select-label] { position: absolute; inline-size: 1px; block-size: 1px; overflow: hidden; clip: rect(0 0 0 0); clip-path: inset(50%); white-space: nowrap; }
[data-rcl-select-native] { position: absolute; inline-size: 1px; block-size: 1px; margin: -1px; overflow: hidden; clip: rect(0 0 0 0); clip-path: inset(50%); white-space: nowrap; }
@media (pointer: coarse) { [data-rcl-select-trigger] { font-size: max(16px, 1em); } }
`;

function stringValue(value: SelectHTMLAttributes<HTMLSelectElement>["value"]): string {
  if (Array.isArray(value)) return value.length > 0 ? String(value[0]) : "";
  return value === undefined || value === null ? "" : String(value);
}

export const Select = forwardRef<HTMLSelectElement, SelectProps>(function Select(
  {
    className,
    id,
    name,
    form,
    required,
    disabled = false,
    options,
    placeholder,
    value,
    defaultValue,
    children,
    onChange,
    onValueChange,
    style,
    onKeyDown,
    "aria-label": ariaLabel,
    "aria-labelledby": ariaLabelledBy,
    ...triggerProps
  },
  ref,
) {
  const generatedId = useId().replace(/:/g, "");
  const inputId = id ?? `select-${generatedId}`;
  const listId = `${inputId}-list`;
  const [uncontrolledValue, setUncontrolledValue] = useState(() => stringValue(defaultValue));
  const selectedValue = value === undefined ? uncontrolledValue : stringValue(value);
  const selectedOption = options.find((option) => option.value === selectedValue);
  const [open, setOpen] = useState(false);
  const [highlightedIndex, setHighlightedIndex] = useState(-1);
  const nativeRef = useRef<HTMLSelectElement | null>(null);
  const triggerRef = useRef<HTMLDivElement | null>(null);
  const dispatchingChange = useRef(false);
  const typeahead = useRef("");
  const typeaheadTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const buttonProps = triggerProps as unknown as HTMLAttributes<HTMLDivElement>;

  const setNativeRef = (element: HTMLSelectElement | null) => {
    nativeRef.current = element;
    if (typeof ref === "function") ref(element);
    else if (ref) ref.current = element;
  };

  useEffect(() => {
    if (!open) {
      setHighlightedIndex(-1);
      return;
    }
    const selectedIndex = options.findIndex(
      (option) => option.value === selectedValue && !option.disabled,
    );
    setHighlightedIndex(
      selectedIndex >= 0 ? selectedIndex : options.findIndex((option) => !option.disabled),
    );
  }, [open, options, selectedValue]);

  useEffect(() => () => clearTimeout(typeaheadTimer.current), []);

  const choose = (option: SelectOption) => {
    if (disabled || option.disabled) return;
    if (value === undefined) setUncontrolledValue(option.value);
    onValueChange?.(option.value);
    const native = nativeRef.current;
    if (native && native.value !== option.value) {
      const setter = Object.getOwnPropertyDescriptor(HTMLSelectElement.prototype, "value")?.set;
      setter?.call(native, option.value);
      dispatchingChange.current = true;
      native.dispatchEvent(new Event("change", { bubbles: true }));
      dispatchingChange.current = false;
    }
    setOpen(false);
    triggerRef.current?.focus();
  };

  const moveHighlight = (delta: number) => {
    if (options.length === 0) return;
    setHighlightedIndex((current) => {
      let next = current < 0 ? (delta > 0 ? -1 : options.length) : current;
      for (let count = 0; count < options.length; count += 1) {
        next = (next + delta + options.length) % options.length;
        if (!options[next]?.disabled) return next;
      }
      return current;
    });
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      if (!open) setOpen(true);
      moveHighlight(event.key === "ArrowDown" ? 1 : -1);
    } else if (event.key === "Home" && open) {
      event.preventDefault();
      setHighlightedIndex(options.findIndex((option) => !option.disabled));
    } else if (event.key === "End" && open) {
      event.preventDefault();
      let lastEnabled = -1;
      options.forEach((option, index) => {
        if (!option.disabled) lastEnabled = index;
      });
      setHighlightedIndex(lastEnabled);
    } else if ((event.key === "Enter" || event.key === " ") && open) {
      event.preventDefault();
      const option = options[highlightedIndex];
      if (option) choose(option);
    } else if (event.key === "Escape" && open) {
      event.preventDefault();
      setOpen(false);
    } else if (event.key.length === 1 && !event.ctrlKey && !event.metaKey && !event.altKey) {
      typeahead.current += event.key.toLocaleLowerCase();
      const match = options.findIndex(
        (option) =>
          !option.disabled && option.label.toLocaleLowerCase().startsWith(typeahead.current),
      );
      if (match >= 0) {
        if (!open) choose(options[match]);
        else setHighlightedIndex(match);
      }
      clearTimeout(typeaheadTimer.current);
      typeaheadTimer.current = setTimeout(() => {
        typeahead.current = "";
      }, 500);
    }
    onKeyDown?.(event as unknown as KeyboardEvent<HTMLSelectElement>);
  };

  return (
    <>
      <StyleSheet
        libraryId="react-component-library:Select"
        version="1.3.7-draft.1"
        css={styleSheet}
      />
      <Popover open={open} onOpenChange={setOpen} placement="bottom-start" responsive="none">
        <span data-rcl-select-field>
          {ariaLabel && (
            <span id={`${inputId}-label`} data-rcl-select-label>
              {ariaLabel}
            </span>
          )}
          <select
            ref={setNativeRef}
            id={inputId}
            name={name}
            form={form}
            required={required}
            disabled={disabled}
            value={selectedValue}
            data-rcl-select-native
            tabIndex={-1}
            aria-hidden="true"
            aria-label={ariaLabel}
            onChange={(event: ChangeEvent<HTMLSelectElement>) => {
              if (value === undefined) setUncontrolledValue(event.currentTarget.value);
              if (!dispatchingChange.current) onValueChange?.(event.currentTarget.value);
              onChange?.(event);
            }}
          >
            {placeholder && <option value="">{placeholder}</option>}
            {options.map((option) => (
              <option key={option.value} value={option.value} disabled={option.disabled}>
                {option.label}
              </option>
            ))}
          </select>
          <PopoverParts.Trigger asChild aria-haspopup="listbox">
            <div
              {...buttonProps}
              ref={triggerRef}
              data-testid={triggerProps["data-testid"] ?? "forms.select"}
              data-rcl-select="true"
              data-rcl-select-trigger
              data-open={open || undefined}
              className={className}
              style={style}
              role="button"
              tabIndex={disabled ? -1 : 0}
              aria-disabled={disabled || undefined}
              aria-haspopup="listbox"
              aria-labelledby={ariaLabelledBy ?? (ariaLabel ? `${inputId}-label` : undefined)}
              aria-expanded={open}
              aria-controls={open ? listId : undefined}
              aria-activedescendant={
                open && highlightedIndex >= 0 ? `${listId}-option-${highlightedIndex}` : undefined
              }
              onKeyDown={handleKeyDown}
            >
              <span data-rcl-select-value data-placeholder={!selectedOption || undefined}>
                {selectedOption?.label ?? placeholder ?? "Choose an option"}
              </span>
              <span data-rcl-select-chevron aria-hidden="true" />
            </div>
          </PopoverParts.Trigger>
        </span>
        <PopoverParts.Content
          role="listbox"
          aria-label={ariaLabel}
          aria-labelledby={ariaLabelledBy}
          initialFocus="none"
          data-rcl-select-listbox
        >
          {options.map((option, index) => (
            <div
              key={option.value}
              id={`${listId}-option-${index}`}
              role="option"
              aria-selected={option.value === selectedValue}
              aria-disabled={option.disabled || undefined}
              data-rcl-select-option
              data-highlighted={index === highlightedIndex || undefined}
              data-disabled={option.disabled || undefined}
              onMouseDown={(event) => event.preventDefault()}
              onClick={() => choose(option)}
            >
              <span data-rcl-select-check aria-hidden="true">
                {option.value === selectedValue ? "✓" : ""}
              </span>
              <span>{option.label}</span>
            </div>
          ))}
        </PopoverParts.Content>
      </Popover>
    </>
  );
});
