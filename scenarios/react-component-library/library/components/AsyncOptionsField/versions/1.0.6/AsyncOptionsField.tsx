/**
 * @libraryId react-component-library:AsyncOptionsField
 * @displayName AsyncOptionsField
 * @description
 * @version 1.0.6
 * @tags ["forms","async","combobox","accessibility","responsive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

/** @vrooliComponentSource forms.async-options-field */
import { useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import {
  useCallback,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
  type KeyboardEvent,
} from "react";
import { useAbortableTask } from "@vrooli/react-component-library/useAbortableTask/1.0.0";
import { useRetry } from "@vrooli/react-component-library/useRetry/1.0.0";

export interface AsyncOption {
  value: string;
  label: string;
  description?: string;
  group?: string;
  disabled?: boolean;
}

export interface AsyncOptionsResult {
  options: AsyncOption[];
  nextPage?: number;
}

export interface AsyncOptionsRequest {
  signal: AbortSignal;
  page: number;
  pageSize: number;
}

export interface AsyncOptionsFieldProps {
  label: string;
  loadOptions: (
    query: string,
    request: AsyncOptionsRequest,
  ) => Promise<AsyncOptionsResult | AsyncOption[]>;
  value?: string;
  defaultValue?: string;
  onChange?: (value: string, option: AsyncOption) => void;
  description?: string;
  placeholder?: string;
  emptyText?: string;
  loadingText?: string;
  errorText?: string;
  offlineText?: string;
  retryLabel?: string;
  pageSize?: number;
  debounceMs?: number;
  maxAttempts?: number;
  initialOptions?: AsyncOption[];
  initialOpen?: boolean;
  disabled?: boolean;
  required?: boolean;
  name?: string;
  id?: string;
  className?: string;
  style?: CSSProperties;
  offline?: boolean;
}

type RequestStatus = "idle" | "loading" | "success" | "error" | "offline";

const styles = `
  [data-rcl-async-options] { position: relative; display: grid; gap: var(--space-2xs, .5rem); min-inline-size: 0; color: var(--color-foreground, #0f172a); }
  [data-rcl-async-options-label] { display: grid; gap: var(--space-3xs, .25rem); font: var(--text-label, 650 .8125rem/1.25rem system-ui, sans-serif); }
  [data-rcl-async-options-label] small { color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 400 .75rem/1rem system-ui, sans-serif); }
  [data-rcl-async-options-control] { position: relative; display: flex; align-items: center; min-inline-size: 0; }
  [data-rcl-async-options-input] { box-sizing: border-box; inline-size: 100%; min-block-size: var(--tap-target-min, 44px); border: 1px solid var(--color-border, #cbd5e1); border-radius: var(--radius-control, .625rem); background: var(--color-surface, #fff); color: var(--color-foreground, #0f172a); padding: .625rem 2.75rem .625rem .875rem; font: var(--text-body, 400 .9375rem/1.4 system-ui, sans-serif); outline: none; transition: border-color var(--dur-quick, 160ms) var(--ease-standard, ease), box-shadow var(--dur-quick, 160ms) var(--ease-standard, ease), background var(--dur-quick, 160ms) var(--ease-standard, ease); }
  [data-rcl-async-options-input]::placeholder { color: var(--color-muted-foreground, #64748b); opacity: .86; }
  [data-rcl-async-options-input]:hover:not(:disabled) { border-color: color-mix(in srgb, var(--color-primary, #2563eb) 48%, var(--color-border, #cbd5e1)); }
  [data-rcl-async-options-input]:focus-visible { border-color: var(--color-primary, #2563eb); box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-primary, #2563eb) 22%, transparent); }
  [data-rcl-async-options-input][aria-expanded="true"] { border-end-start-radius: 0; border-end-end-radius: 0; }
  [data-rcl-async-options-input]:disabled { cursor: not-allowed; opacity: .58; background: var(--color-surface-muted, #f1f5f9); }
  [data-rcl-async-options-chevron] { position: absolute; inset-inline-end: .875rem; pointer-events: none; color: var(--color-muted-foreground, #64748b); font-size: 1rem; line-height: 1; transition: transform var(--dur-quick, 160ms) var(--ease-standard, ease); }
  [data-rcl-async-options-input][aria-expanded="true"] + [data-rcl-async-options-chevron] { transform: rotate(180deg); }
  [data-rcl-async-options-panel] { position: absolute; z-index: 10; inset-inline: 0; inset-block-start: calc(100% - var(--space-2xs, .5rem)); overflow: hidden; border: 1px solid var(--color-border, #cbd5e1); border-block-start: 0; border-radius: 0 0 var(--radius-control, .625rem) var(--radius-control, .625rem); background: var(--color-surface-raised, #fff); box-shadow: var(--elev-overlay, 0 16px 36px rgb(15 23 42 / .16)); }
  [data-rcl-async-options-panel]::before { content: ""; display: block; block-size: var(--space-2xs, .5rem); background: var(--color-surface-raised, #fff); }
  [data-rcl-async-options-list] { display: grid; max-block-size: min(20rem, 42vh); overflow: auto; padding: 0 var(--space-2xs, .5rem) var(--space-2xs, .5rem); overscroll-behavior: contain; }
  [data-rcl-async-options-group] { padding: .625rem .625rem .35rem; color: var(--color-muted-foreground, #64748b); font: var(--text-overline, 700 .6875rem/1rem system-ui, sans-serif); letter-spacing: .08em; text-transform: uppercase; }
  [data-rcl-async-options-option] { display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: .625rem; inline-size: 100%; min-block-size: 2.875rem; box-sizing: border-box; border: 0; border-radius: var(--radius-control, .5rem); background: transparent; color: var(--color-foreground, #0f172a); padding: .625rem .75rem; text-align: start; font: inherit; cursor: pointer; }
  [data-rcl-async-options-option]:hover, [data-rcl-async-options-option][data-highlighted="true"] { background: color-mix(in srgb, var(--color-primary, #2563eb) 9%, var(--color-surface-raised, #fff)); }
  [data-rcl-async-options-option][aria-selected="true"] { background: color-mix(in srgb, var(--color-primary, #2563eb) 13%, var(--color-surface-raised, #fff)); color: var(--color-primary-strong, var(--color-primary, #2563eb)); }
  [data-rcl-async-options-option]:focus-visible { outline: 3px solid color-mix(in srgb, var(--color-primary, #2563eb) 30%, transparent); outline-offset: -2px; }
  [data-rcl-async-options-option]:disabled { cursor: not-allowed; opacity: .46; }
  [data-rcl-async-options-option-copy] { display: grid; gap: .15rem; min-inline-size: 0; }
  [data-rcl-async-options-option-label] { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-weight: 650; }
  [data-rcl-async-options-option-description] { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 400 .75rem/1rem system-ui, sans-serif); }
  [data-rcl-async-options-check] { align-self: center; color: var(--color-primary, #2563eb); font-size: 1.05rem; }
  [data-rcl-async-options-state] { display: grid; justify-items: start; gap: .5rem; padding: .75rem; color: var(--color-muted-foreground, #64748b); font: var(--text-body, 400 .875rem/1.35rem system-ui, sans-serif); }
  [data-rcl-async-options-state][data-tone="error"] { color: var(--color-danger, #b91c1c); }
  [data-rcl-async-options-state][data-tone="offline"] { color: var(--color-warning, #a16207); }
  [data-rcl-async-options-state] button, [data-rcl-async-options-more] { min-block-size: 2.25rem; border: 1px solid currentColor; border-radius: var(--radius-control, .5rem); background: transparent; color: inherit; padding-inline: .75rem; font: var(--text-label, 650 .8125rem/1rem system-ui, sans-serif); cursor: pointer; }
  [data-rcl-async-options-more] { justify-self: stretch; margin: .25rem .5rem .125rem; color: var(--color-primary, #2563eb); }
  [data-rcl-async-options-status] { min-block-size: 1rem; color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 400 .75rem/1rem system-ui, sans-serif); }
  [data-rcl-async-options-spinner] { display: inline-block; inline-size: .9rem; block-size: .9rem; border: 2px solid color-mix(in srgb, currentColor 24%, transparent); border-block-start-color: currentColor; border-radius: 50%; animation: rcl-async-options-spin .75s linear infinite; vertical-align: -.15rem; }
  @keyframes rcl-async-options-spin { to { transform: rotate(360deg); } }
  @media (prefers-reduced-motion: reduce) { [data-rcl-async-options-spinner] { animation: none; } [data-rcl-async-options-input], [data-rcl-async-options-chevron] { transition: none; } }
  @media (max-width: 30rem) { [data-rcl-async-options-list] { max-block-size: min(17rem, 36vh); } }
`;

function normalizedResult(result: AsyncOptionsResult | AsyncOption[]): AsyncOptionsResult {
  return Array.isArray(result) ? { options: result } : result;
}

function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === "AbortError";
}

export const AsyncOptionsField = withClassName(function AsyncOptionsField({
  label,
  loadOptions,
  value,
  defaultValue = "",
  onChange,
  description,
  placeholder,
  emptyText = "No matches yet",
  loadingText = "Finding matches…",
  errorText = "We couldn’t load these options.",
  offlineText = "You’re offline. Reconnect to search again.",
  retryLabel = "Try again",
  pageSize = 20,
  debounceMs = 220,
  maxAttempts = 2,
  initialOptions = [],
  initialOpen = false,
  disabled = false,
  required = false,
  name,
  id,
  className,
  style,
  offline = false,
}: AsyncOptionsFieldProps) {
  const libraryStrings = useStrings();
  placeholder =
    placeholder ??
    libraryStrings(
      "forms.async-options-field.search-or-choose-an-option",
      "Search or choose an option",
    );
  const strings = useStrings();
  const generatedID = useId();
  const inputID = id ?? `async-options-${generatedID.replace(/:/g, "")}`;
  const listID = `${inputID}-list`;
  const descriptionID = `${inputID}-description`;
  const rootRef = useRef<HTMLDivElement>(null);
  const queryRef = useRef("");
  const pageRef = useRef(1);
  const requestRef = useRef(0);
  const [query, setQuery] = useState("");
  const [selectedValue, setSelectedValue] = useState(value ?? defaultValue);
  const [options, setOptions] = useState<AsyncOption[]>(initialOptions);
  const [nextPage, setNextPage] = useState<number | undefined>();
  const [status, setStatus] = useState<RequestStatus>("idle");
  const [open, setOpen] = useState(initialOpen);
  const [highlighted, setHighlighted] = useState(-1);
  const { run: runRetry } = useRetry<AsyncOptionsResult>({ maxAttempts });

  useEffect(() => {
    if (value !== undefined) setSelectedValue(value);
  }, [value]);

  const task = useCallback(
    (signal: AbortSignal) =>
      loadOptions(queryRef.current, {
        signal,
        page: pageRef.current,
        pageSize,
      }),
    [loadOptions, pageSize],
  );
  const { run: runLatest, abort: abortLatest } = useAbortableTask(task);

  const request = useCallback(
    (page: number, replace: boolean) => {
      const requestID = requestRef.current + 1;
      requestRef.current = requestID;
      pageRef.current = page;
      setStatus(offline ? "offline" : "loading");
      if (offline) return;
      abortLatest();
      void runRetry(() => runLatest().then(normalizedResult))
        .then((result) => {
          if (requestRef.current !== requestID) return;
          setOptions((previous) => (replace ? result.options : [...previous, ...result.options]));
          setNextPage(result.nextPage);
          setStatus("success");
          setHighlighted((current) =>
            current >= 0 ? current : result.options.findIndex((option) => !option.disabled),
          );
        })
        .catch((caught: unknown) => {
          if (requestRef.current !== requestID || isAbortError(caught)) return;
          setStatus("error");
        });
    },
    [abortLatest, offline, runLatest, runRetry],
  );

  useEffect(() => {
    if (!open) return;
    const timer = globalThis.setTimeout(() => request(1, true), debounceMs);
    return () => globalThis.clearTimeout(timer);
  }, [debounceMs, open, query, request]);

  useEffect(() => {
    const closeIfOutside = (event: PointerEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) setOpen(false);
    };
    document.addEventListener("pointerdown", closeIfOutside);
    return () => document.removeEventListener("pointerdown", closeIfOutside);
  }, []);

  const selected = options.find((option) => option.value === selectedValue);
  const availableOptions = useMemo(() => options.filter((option) => !option.disabled), [options]);
  const selectOption = useCallback(
    (option: AsyncOption) => {
      if (option.disabled) return;
      setSelectedValue(option.value);
      setQuery(option.label);
      setOpen(false);
      setHighlighted(-1);
      onChange?.(option.value, option);
    },
    [onChange],
  );

  const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "ArrowDown") {
      event.preventDefault();
      setOpen(true);
      setHighlighted((current) => Math.min(availableOptions.length - 1, current + 1));
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      setHighlighted((current) =>
        Math.max(0, current < 0 ? availableOptions.length - 1 : current - 1),
      );
    } else if (event.key === "Home" && open && availableOptions.length > 0) {
      event.preventDefault();
      setHighlighted(0);
    } else if (event.key === "End" && open && availableOptions.length > 0) {
      event.preventDefault();
      setHighlighted(availableOptions.length - 1);
    } else if (event.key === "Enter" && open && highlighted >= 0) {
      event.preventDefault();
      const option = availableOptions[highlighted];
      if (option) selectOption(option);
    } else if (event.key === "Escape") {
      setOpen(false);
      setHighlighted(-1);
    }
  };

  const openField = () => {
    if (disabled) return;
    setOpen(true);
  };

  return (
    <div ref={rootRef} data-rcl-async-options className={className} style={style}>
      <style>{styles}</style>
      <label data-rcl-async-options-label htmlFor={inputID}>
        <span>
          {label}
          {required ? " *" : ""}
        </span>
        {description && <small id={descriptionID}>{description}</small>}
      </label>
      <div data-rcl-async-options-control>
        <input
          data-testid="forms.async-options-field"
          id={inputID}
          name={name}
          data-rcl-async-options-input
          role="combobox"
          type="text"
          value={query || selected?.label || ""}
          placeholder={placeholder}
          disabled={disabled}
          required={required}
          autoComplete="off"
          aria-label={label}
          aria-autocomplete="list"
          aria-controls={listID}
          aria-describedby={description ? descriptionID : undefined}
          aria-expanded={open}
          aria-activedescendant={
            open && highlighted >= 0
              ? `${listID}-option-${availableOptions[highlighted]?.value}`
              : undefined
          }
          aria-busy={status === "loading"}
          onFocus={openField}
          onChange={(event) => {
            setQuery(event.target.value);
            setOpen(true);
            setHighlighted(-1);
          }}
          onKeyDown={handleKeyDown}
        />
        <span aria-hidden="true" data-rcl-async-options-chevron>
          ⌄
        </span>
      </div>
      <div data-rcl-async-options-status role="status" aria-live="polite">
        {status === "loading" ? (
          <>
            <span data-rcl-async-options-spinner aria-hidden="true" /> {loadingText}
          </>
        ) : status === "offline" ? (
          offlineText
        ) : (
          ""
        )}
      </div>
      {open && (
        <div data-rcl-async-options-panel>
          {status === "error" ? (
            <div data-rcl-async-options-state data-tone="error">
              <span>{errorText}</span>
              <button
                data-testid="forms.async-options-field"
                type="button"
                onClick={() => request(pageRef.current, pageRef.current !== 1)}
              >
                {retryLabel}
              </button>
            </div>
          ) : status === "offline" ? (
            <div data-rcl-async-options-state data-tone="offline">
              <span>{offlineText}</span>
            </div>
          ) : options.length === 0 && status !== "loading" ? (
            <div data-rcl-async-options-state>
              <span>{emptyText}</span>
            </div>
          ) : (
            <div
              id={listID}
              data-rcl-async-options-list
              role="listbox"
              aria-label={`${label} options`}
            >
              {options.map((option) => {
                const position = availableOptions.indexOf(option);
                return (
                  <div key={option.value}>
                    {option.group &&
                      (position === 0 || options[position - 1]?.group !== option.group) && (
                        <div data-rcl-async-options-group>{option.group}</div>
                      )}
                    <button
                      data-testid="forms.async-options-field"
                      id={`${listID}-option-${option.value}`}
                      type="button"
                      role="option"
                      data-rcl-async-options-option
                      data-highlighted={position === highlighted}
                      aria-selected={option.value === selectedValue}
                      disabled={option.disabled}
                      onMouseDown={(event) => event.preventDefault()}
                      onClick={() => selectOption(option)}
                    >
                      <span data-rcl-async-options-option-copy>
                        <span data-rcl-async-options-option-label>{option.label}</span>
                        {option.description && (
                          <span data-rcl-async-options-option-description>
                            {option.description}
                          </span>
                        )}
                      </span>
                      {option.value === selectedValue && (
                        <span data-rcl-async-options-check aria-hidden="true">
                          ✓
                        </span>
                      )}
                    </button>
                  </div>
                );
              })}
              {nextPage !== undefined && (
                <button
                  data-testid="forms.async-options-field"
                  type="button"
                  data-rcl-async-options-more
                  onClick={() => request(nextPage, false)}
                >
                  {strings("forms.async-options-field.load-more", "Load more")}
                </button>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
});
