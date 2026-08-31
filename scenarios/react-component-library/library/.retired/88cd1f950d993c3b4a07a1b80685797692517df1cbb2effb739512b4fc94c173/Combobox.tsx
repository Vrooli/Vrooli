/**
 * @libraryId react-component-library:Combobox
 * @displayName Combobox
 * @description The searchable selection field supporting local or remote options, typeahead, creation of new values, result highlighting, virtualization, async cancellation, and accessible active-option reporting.
 * @version 1.0.7
 * @tags ["forms","selection","combobox","async","keyboard","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.2";

/** @vrooliComponentSource forms.combobox */
import { useStrings } from "@vrooli/react-component-library/useLocale/1.1.0";
import {
  useId,
  useMemo,
  useState,
  type CSSProperties,
  type KeyboardEvent,
} from "react";
import {
  AsyncOptionsField,
  type AsyncOptionsFieldProps,
} from "@vrooli/react-component-library/AsyncOptionsField/1.0.0";
import { type SelectOption } from "@vrooli/react-component-library/Select/1";

export interface ComboboxOption extends SelectOption {
  description?: string;
  group?: string;
}
export interface ComboboxProps {
  label: string;
  options?: ComboboxOption[];
  loadOptions?: AsyncOptionsFieldProps["loadOptions"];
  value?: string;
  defaultValue?: string;
  onChange?: (value: string, option?: ComboboxOption) => void;
  onCreate?: (label: string) => void;
  allowCreate?: boolean;
  description?: string;
  error?: string;
  placeholder?: string;
  emptyText?: string;
  disabled?: boolean;
  required?: boolean;
  name?: string;
  id?: string;
  offline?: boolean;
  initialOpen?: boolean;
  initialQuery?: string;
  maxAttempts?: number;
  debounceMs?: number;
  className?: string;
  style?: CSSProperties;
}

const styles = `
  [data-rcl-combobox] { position: relative; display: grid; gap: var(--space-2xs, 8px); min-inline-size: 0; color: var(--color-foreground, #0f172a); }
  [data-rcl-combobox-label] { display: grid; gap: var(--space-3xs, 4px); font: var(--text-label, 500 var(--text-label-size) / var(--text-label-line) var(--font-sans)); }
  [data-rcl-combobox-label] small { color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)); }
  [data-rcl-combobox-control] { position: relative; min-inline-size: 0; }
  [data-rcl-combobox-input] { box-sizing: border-box; inline-size: 100%; min-block-size: var(--tap-target-min, 44px); border: 1px solid var(--color-border, #cbd5e1); border-radius: var(--radius-control, 0.375rem); background: var(--color-surface, #ffffff); color: var(--color-foreground, #0f172a); padding: .625rem 2.75rem .625rem .875rem; font: var(--text-body, 400 var(--text-body-size) / var(--text-body-line) var(--font-sans)); outline: none; transition: border-color var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2, 0, 0, 1)), box-shadow var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2, 0, 0, 1)); }
  [data-rcl-combobox-input]:hover:not(:disabled) { border-color: color-mix(in srgb, var(--color-primary, #2563eb) 48%, var(--color-border, #cbd5e1)); }
  [data-rcl-combobox-input][aria-expanded="true"] { border-end-start-radius: 0; border-end-end-radius: 0; }
  [data-rcl-combobox-input][aria-invalid="true"] { border-color: var(--color-danger, #dc2626); }
  [data-rcl-combobox-input]:disabled { cursor: not-allowed; opacity: .58; background: var(--color-surface-muted, #f1f5f9); }
  [data-rcl-combobox-chevron] { position: absolute; inset-inline-end: .875rem; inset-block-start: 50%; pointer-events: none; color: var(--color-muted-foreground, #64748b); transform: translateY(-50%); transition: transform var(--dur-quick, 180ms) var(--ease-standard, cubic-bezier(.2, 0, 0, 1)); }
  [data-rcl-combobox-input][aria-expanded="true"] + [data-rcl-combobox-chevron] { transform: translateY(-50%) rotate(180deg); }
  [data-rcl-combobox-panel] { position: absolute; z-index: 10; inset-inline: 0; inset-block-start: calc(100% - var(--space-2xs, 8px)); overflow: hidden; border: 1px solid var(--color-border, #cbd5e1); border-block-start: 0; border-radius: 0 0 var(--radius-control, 0.375rem) var(--radius-control, 0.375rem); background: var(--color-surface-raised, #ffffff); box-shadow: var(--elev-overlay, 0 2px 4px rgba(9, 18, 22, .06), 0 4px 12px rgba(9, 18, 22, .10)); }
  [data-rcl-combobox][data-open="true"] [data-rcl-combobox-status] { block-size: 0; min-block-size: 0; overflow: hidden; }
  [data-rcl-combobox-list] { display: grid; max-block-size: min(20rem, 42vh); overflow: auto; padding: var(--space-2xs, 8px); overscroll-behavior: contain; }
  [data-rcl-combobox-option] { display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: .625rem; inline-size: 100%; min-block-size: 2.875rem; box-sizing: border-box; border: 0; border-radius: var(--radius-control, 0.375rem); background: transparent; color: var(--color-foreground, #0f172a); padding: .625rem .75rem; text-align: start; font: inherit; cursor: pointer; }
  [data-rcl-combobox-option]:hover, [data-rcl-combobox-option][data-highlighted="true"] { background: color-mix(in srgb, var(--color-primary, #2563eb) 9%, var(--color-surface-raised, #ffffff)); }
  [data-rcl-combobox-option][aria-selected="true"] { background: color-mix(in srgb, var(--color-primary, #2563eb) 13%, var(--color-surface-raised, #ffffff)); color: var(--color-primary-strong, var(--color-primary)); }
  [data-rcl-combobox-option-copy] { display: grid; gap: .15rem; min-inline-size: 0; }
  [data-rcl-combobox-option-label] { overflow-wrap: anywhere; font-weight: 650; }
  [data-rcl-combobox-option-description] { color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)); }
  [data-rcl-combobox-check] { align-self: center; color: var(--color-primary, #2563eb); font-size: 1.05rem; }
  [data-rcl-combobox-state], [data-rcl-combobox-status] { color: var(--color-muted-foreground, #64748b); font: var(--text-body, 400 var(--text-body-size) / var(--text-body-line) var(--font-sans)); }
  [data-rcl-combobox-state] { padding: .75rem; }
  [data-rcl-combobox-create] { color: var(--color-primary, #2563eb); }
  [data-rcl-combobox-error] { color: var(--color-danger, #dc2626); font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)); }

`;

function optionID(listID: string, value: string) {
  return `${listID}-option-${encodeURIComponent(value)}`;
}

export const Combobox = withClassName(function Combobox({
  label,
  options = [],
  loadOptions,
  value,
  defaultValue = "",
  onChange,
  onCreate,
  allowCreate = false,
  description,
  error,
  placeholder,
  emptyText = "No matches yet",
  disabled = false,
  required = false,
  name,
  id,
  offline = false,
  initialOpen = false,
  initialQuery = "",
  maxAttempts = 2,
  debounceMs = 220,
  className,
  style,
}: ComboboxProps) {
  const libraryStrings = useStrings();
  placeholder =
    placeholder ??
    libraryStrings(
      "forms.combobox.search-or-choose-an-option",
      "Search or choose an option",
    );
  const generatedID = useId().replace(/:/g, "");
  const inputID = id ?? `combobox-${generatedID}`;
  if (loadOptions) {
    return (
      <AsyncOptionsField
        label={label}
        loadOptions={loadOptions}
        value={value}
        defaultValue={defaultValue}
        onChange={(next, option) => onChange?.(next, option)}
        description={description}
        placeholder={placeholder}
        emptyText={emptyText}
        disabled={disabled}
        required={required}
        name={name}
        id={inputID}
        offline={offline}
        initialOpen={initialOpen}
        maxAttempts={maxAttempts}
        debounceMs={debounceMs}
        className={className}
        style={style}
      />
    );
  }
  return (
    <LocalCombobox
      label={label}
      options={options}
      value={value}
      defaultValue={defaultValue}
      onChange={onChange}
      onCreate={onCreate}
      allowCreate={allowCreate}
      description={description}
      error={error}
      placeholder={placeholder}
      emptyText={emptyText}
      disabled={disabled}
      required={required}
      name={name}
      inputID={inputID}
      initialOpen={initialOpen}
      initialQuery={initialQuery}
      className={className}
      style={style}
    />
  );
});

interface LocalProps
  extends Omit<ComboboxProps, "id" | "loadOptions" | "offline"> {
  inputID: string;
  options: ComboboxOption[];
}
function LocalCombobox({
  label,
  options,
  value,
  defaultValue,
  onChange,
  onCreate,
  allowCreate,
  description,
  error,
  placeholder,
  emptyText,
  disabled,
  required,
  name,
  inputID,
  initialOpen,
  initialQuery = "",
  className,
  style,
}: LocalProps) {
  const listID = `${inputID}-list`;
  const [selectedValue, setSelectedValue] = useState(value ?? defaultValue);
  const [query, setQuery] = useState(initialQuery);
  const [open, setOpen] = useState(initialOpen);
  const [highlighted, setHighlighted] = useState(-1);
  const selected = options.find((option) => option.value === selectedValue);
  const filtered = useMemo(
    () =>
      options.filter((option) =>
        `${option.label} ${option.description ?? ""}`
          .toLowerCase()
          .includes(query.toLowerCase()),
      ),
    [options, query],
  );
  const canCreate = Boolean(
    allowCreate &&
      onCreate &&
      query.trim() &&
      !filtered.some(
        (option) => option.label.toLowerCase() === query.trim().toLowerCase(),
      ),
  );
  const choose = (option: ComboboxOption) => {
    setSelectedValue(option.value);
    setQuery("");
    setOpen(false);
    setHighlighted(-1);
    onChange?.(option.value, option);
  };
  const keyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "ArrowDown") {
      event.preventDefault();
      setOpen(true);
      setHighlighted((current) => Math.min(filtered.length - 1, current + 1));
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      setHighlighted((current) =>
        Math.max(0, current < 0 ? filtered.length - 1 : current - 1),
      );
    } else if (event.key === "Home" && open) {
      event.preventDefault();
      setHighlighted(0);
    } else if (event.key === "End" && open) {
      event.preventDefault();
      setHighlighted(filtered.length - 1);
    } else if (
      event.key === "Enter" &&
      open &&
      highlighted >= 0 &&
      filtered[highlighted]
    ) {
      event.preventDefault();
      choose(filtered[highlighted]);
    } else if (event.key === "Escape") {
      setOpen(false);
      setHighlighted(-1);
    }
  };
  return (
    <div
      data-rcl-combobox
      data-open={open || undefined}
      className={className}
      style={style}
    >
      <StyleSheet name="combobox-1-0-6-1" css={styles} />
      <label data-rcl-combobox-label htmlFor={inputID}>
        <span>
          {label}
          {required ? " *" : ""}
        </span>
        {description && <small>{description}</small>}
      </label>
      <div data-rcl-combobox-control>
        <input
          data-testid="forms.combobox"
          id={inputID}
          name={name}
          data-rcl-combobox-input
          role="combobox"
          aria-label={label}
          value={query || selected?.label || ""}
          placeholder={placeholder}
          disabled={disabled}
          required={required}
          autoComplete="off"
          aria-autocomplete="list"
          aria-controls={listID}
          aria-expanded={open}
          aria-activedescendant={
            open && highlighted >= 0 && filtered[highlighted]
              ? optionID(listID, filtered[highlighted].value)
              : undefined
          }
          aria-invalid={error ? true : undefined}
          aria-describedby={error ? `${inputID}-error` : undefined}
          onFocus={() => !disabled && setOpen(true)}
          onChange={(event) => {
            setQuery(event.target.value);
            setOpen(true);
            setHighlighted(-1);
          }}
          onKeyDown={keyDown}
        />
        <span aria-hidden="true" data-rcl-combobox-chevron>
          ⌄
        </span>
      </div>
      {error && (
        <span id={`${inputID}-error`} data-rcl-combobox-error role="alert">
          {error}
        </span>
      )}
      <div data-rcl-combobox-status role="status" aria-live="polite">
        {open && filtered.length ? `${filtered.length} matches` : ""}
      </div>
      {open && (
        <div data-rcl-combobox-panel>
          <div
            id={listID}
            data-rcl-combobox-list
            role="listbox"
            aria-label={`${label} options`}
          >
            {filtered.map((option, index) => (
              <button
                data-testid="forms.combobox"
                key={option.value}
                id={optionID(listID, option.value)}
                type="button"
                role="option"
                data-rcl-combobox-option
                data-highlighted={highlighted === index || undefined}
                aria-selected={option.value === selectedValue}
                onMouseDown={(event) => event.preventDefault()}
                onClick={() => choose(option)}
              >
                <span data-rcl-combobox-option-copy>
                  <span data-rcl-combobox-option-label>{option.label}</span>
                  {option.description && (
                    <span data-rcl-combobox-option-description>
                      {option.description}
                    </span>
                  )}
                </span>
                {option.value === selectedValue && (
                  <span data-rcl-combobox-check aria-hidden="true">
                    ✓
                  </span>
                )}
              </button>
            ))}
            {canCreate && (
              <button
                data-testid="forms.combobox"
                type="button"
                data-rcl-combobox-option
                data-rcl-combobox-create
                onMouseDown={(event) => event.preventDefault()}
                onClick={() => {
                  onCreate?.(query.trim());
                  setQuery("");
                  setOpen(false);
                }}
              >
                Create “{query.trim()}”
              </button>
            )}
            {!filtered.length && !canCreate && (
              <div data-rcl-combobox-state>{emptyText}</div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
