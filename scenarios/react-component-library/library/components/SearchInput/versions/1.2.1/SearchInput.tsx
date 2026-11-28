/**
 * @libraryId react-component-library:SearchInput
 * @displayName SearchInput
 * @description The search field with debouncing, in-flight cancellation, recent queries, clear action, keyboard shortcut binding, loading feedback, and optional suggestions.
 * @version 1.2.1
 * @tags []
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:SearchInput */
import {
  forwardRef,
  useCallback,
  useEffect,
  useId,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
  type CSSProperties,
  type InputHTMLAttributes,
  type KeyboardEvent as ReactKeyboardEvent,
  type MutableRefObject,
  type ReactNode,
  type RefObject,
} from "react";
import { Clock, Search, X } from "lucide-react";

import { cn } from "@vrooli/react-component-library/ClassMerge/1";
import { IconButton } from "@vrooli/react-component-library/IconButton/3";
import { Input } from "@vrooli/react-component-library/Input/1";
import { InputGroup } from "@vrooli/react-component-library/InputGroup/1";
import { Popover, PopoverContent, usePopover } from "@vrooli/react-component-library/Popover/1";
import { Spinner } from "@vrooli/react-component-library/Spinner/1";
import { useAbortableTask } from "@vrooli/react-component-library/useAbortableTask/1";
import { useControllableState } from "@vrooli/react-component-library/useControllableState/1";
import { useElementRect } from "@vrooli/react-component-library/useElementRect/2";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { useStrings } from "@vrooli/react-component-library/useLocale/1";

export const SEARCH_INPUT_APPEARANCES = ["form", "toolbar"] as const;
export const SEARCH_INPUT_SIZES = ["sm", "md", "lg"] as const;
export const SEARCH_INPUT_SHAPES = ["rounded", "pill"] as const;
export const SEARCH_INPUT_LABEL_VISIBILITIES = ["visible", "hidden"] as const;
export const SEARCH_INPUT_STATUSES = ["idle", "loading", "ready", "error"] as const;

export type SearchInputAppearance = (typeof SEARCH_INPUT_APPEARANCES)[number];
export type SearchInputSize = (typeof SEARCH_INPUT_SIZES)[number];
export type SearchInputShape = (typeof SEARCH_INPUT_SHAPES)[number];
export type SearchInputLabelVisibility = (typeof SEARCH_INPUT_LABEL_VISIBILITIES)[number];
export type SearchInputStatus = (typeof SEARCH_INPUT_STATUSES)[number];

/** One selectable suggestion. A bare string is shorthand for `value` + `label`. */
export interface SearchInputSuggestion {
  value: string;
  label?: string;
  description?: string;
}

export interface SearchInputSearchOptions {
  /** Aborted when the query changes, the field closes, or the component unmounts. */
  signal: AbortSignal;
}

/**
 * Resolve suggestions for a query. Providing this hands debouncing, in-flight
 * cancellation, and the loading/error states to the field. A host that already
 * owns its result list can pass `suggestions` instead.
 */
export type SearchInputSearch = (
  query: string,
  options: SearchInputSearchOptions,
) => Promise<readonly (string | SearchInputSuggestion)[]>;

export interface SearchInputProps
  extends Omit<
    InputHTMLAttributes<HTMLInputElement>,
    "children" | "defaultValue" | "onChange" | "onSubmit" | "size" | "type" | "value"
  > {
  /** Controlled query. Pair with `onValueChange` (preferred) or `onChange`. */
  value?: string;
  /** Uncontrolled initial query. */
  defaultValue?: string;
  /** Value-first change contract. This is the contract to adopt. */
  onValueChange?: (value: string) => void;
  /** Native change compatibility for existing input-prop consumers. */
  onChange?: (event: ChangeEvent<HTMLInputElement>) => void;
  /**
   * The field's name. When it is visible it is also the accessible name; when
   * hidden it is rendered visually hidden and still names the input. Omit it
   * and pass `aria-label` when the host already owns the label.
   */
  label?: ReactNode;
  /** `visible` follows the form convention; `toolbar` defaults to hidden. */
  labelVisibility?: SearchInputLabelVisibility;
  /** Helper copy below the field, associated through `aria-describedby`. */
  description?: ReactNode;
  /** Validation message; also sets the invalid tone and `aria-invalid`. */
  error?: ReactNode;
  /**
   * `form` is a labelled field capped to a comfortable reading width.
   * `toolbar` is a full-width pill for a command strip.
   */
  appearance?: SearchInputAppearance;
  size?: SearchInputSize;
  shape?: SearchInputShape;
  /** Replaces the default toolbar magnifier. `null` removes the leading slot. */
  leading?: ReactNode | null;
  /** Trailing controls between the query and the clear action. */
  actions?: ReactNode;
  /** Show the clear action whenever the query is non-empty. Defaults to true. */
  clearable?: boolean;
  onClear?: () => void;
  /** Accessible name for the clear action. */
  clearLabel?: string;
  /** Keyboard hint rendered at the trailing edge. Binding is the host's job. */
  shortcut?: ReactNode;
  /** Native title/description for the shortcut hint. */
  shortcutLabel?: string;
  /** Host-owned loading state. Internal searches derive this automatically. */
  loading?: boolean;
  loadingLabel?: string;
  /** Async suggestion source. Takes precedence over `suggestions`. */
  onSearch?: SearchInputSearch;
  /** Quiet period before an async search starts. */
  debounceMs?: number;
  /** Queries shorter than this do not call `onSearch`. Defaults to 1. */
  minimumQueryLength?: number;
  /** Panel-level search failure. */
  searchError?: ReactNode;
  onSearchError?: (error: unknown) => void;
  /** Retry callback for a failed async search. */
  onRetry?: () => void;
  /** Static suggestions. The field filters them by the current query. */
  suggestions?: readonly (string | SearchInputSuggestion)[];
  /** Queries offered when the field is empty. */
  recentQueries?: readonly string[];
  suggestionsLabel?: string;
  recentLabel?: string;
  /** Replaces the default no-match sentence. */
  emptyLabel?: ReactNode;
  onSuggestionSelect?: (suggestion: SearchInputSuggestion, index: number) => void;
  onRecentSelect?: (query: string, index: number) => void;
  onClearRecent?: () => void;
  clearRecentLabel?: string;
  /** Controlled suggestion panel visibility. */
  open?: boolean;
  defaultOpen?: boolean;
  onOpenChange?: (open: boolean) => void;
  /** Fired on Enter when no suggestion is active. Form submit still occurs. */
  onSubmit?: (value: string) => void;
  /** Defaults to `forms.search-input-panel`. */
  panelTestId?: string;
  /** Defaults to `forms.search-input-option-${index}`. */
  optionTestId?: (suggestion: SearchInputSuggestion, index: number) => string;
  /** Defaults to `forms.search-input`. */
  testId?: string;
  /** Defaults to `${testId}-group`. */
  groupTestId?: string;
  /** Root class. */
  className?: string;
  /** Root style. */
  style?: CSSProperties;
  inputClassName?: string;
  inputStyle?: CSSProperties;
}

const styles = `
[data-rcl-search-input] {
  display: grid;
  gap: var(--space-2xs);
  min-inline-size: 0;
  color: var(--color-foreground);
}
[data-rcl-search-input][data-appearance="form"] {
  inline-size: min(100%, var(--rcl-search-input-form-max, 22.5rem));
}
[data-rcl-search-input][data-appearance="toolbar"] { inline-size: 100%; }

[data-rcl-search-input-label] {
  color: var(--color-foreground);
  font: var(--text-label);
}
[data-rcl-search-input][data-label-visibility="hidden"] [data-rcl-search-input-label] {
  position: absolute;
  inline-size: 1px;
  block-size: 1px;
  overflow: hidden;
  clip-path: inset(50%);
  white-space: nowrap;
}

[data-rcl-search-input-control] { position: relative; min-inline-size: 0; }

/* One clear affordance. Safari and Chrome otherwise paint a second, unstyled
   native cancel button inside type="search". */
[data-rcl-search-input] [data-rcl-input]::-webkit-search-cancel-button,
[data-rcl-search-input] [data-rcl-input]::-webkit-search-decoration {
  -webkit-appearance: none;
  appearance: none;
}

[data-rcl-search-input-description] {
  color: var(--color-muted-foreground);
  font: var(--text-caption);
}
[data-rcl-search-input-error] {
  color: var(--color-danger);
  font: var(--text-caption);
}

[data-rcl-search-input-shortcut] {
  display: inline-flex;
  align-items: center;
  min-block-size: calc(var(--text-label-line) + var(--space-3xs));
  padding-inline: var(--space-2xs);
  border: var(--border-hairline) solid var(--color-border);
  border-radius: var(--radius-control);
  background: color-mix(in srgb, var(--color-surface-muted) 72%, transparent);
  color: var(--color-muted-foreground);
  font: var(--text-caption);
  white-space: nowrap;
}
@media (pointer: coarse) {
  [data-rcl-search-input-shortcut] { display: none; }
}

[data-rcl-search-input-panel] {
  padding: 0;
  overflow: hidden;
}
[data-rcl-search-input-panel] [data-rcl-popover-arrow] { display: none; }
[data-rcl-search-input-list] {
  display: grid;
  gap: var(--space-3xs);
  max-block-size: min(20rem, var(--rcl-popover-available-height, 20rem));
  overflow: auto;
  overscroll-behavior: contain;
  padding: var(--space-2xs);
}
[data-rcl-search-input-section] {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-sm);
  padding: var(--space-2xs) var(--space-xs) var(--space-3xs);
  color: var(--color-muted-foreground);
  font: var(--text-overline);
  letter-spacing: 0.06em;
  text-transform: uppercase;
}
[data-rcl-search-input-section-clear] {
  min-block-size: var(--tap-target-min);
  padding-inline: var(--space-2xs);
  border: 0;
  border-radius: var(--radius-control);
  background: transparent;
  color: var(--color-primary);
  font: var(--text-caption);
  cursor: pointer;
}
[data-rcl-search-input-section-clear]:hover { background: var(--color-surface-muted); }

[data-rcl-search-input-option] {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: start;
  gap: var(--space-xs);
  inline-size: 100%;
  min-block-size: var(--tap-target-min);
  box-sizing: border-box;
  padding: var(--space-xs) var(--space-sm);
  border: 0;
  border-radius: var(--radius-control);
  background: transparent;
  color: var(--color-foreground);
  font: inherit;
  text-align: start;
  cursor: pointer;
}
[data-rcl-search-input-option][data-active="true"],
[data-rcl-search-input-option]:hover {
  background: color-mix(in srgb, var(--color-primary) 10%, var(--color-surface-raised));
}
[data-rcl-search-input-option][data-active="true"] { box-shadow: inset 2px 0 0 var(--color-primary); }
[data-rcl-search-input-option]:focus-visible {
  outline: var(--border-strong) solid var(--color-focus);
  outline-offset: -2px;
}
[data-rcl-search-input-option-copy] { display: grid; gap: var(--space-3xs); min-inline-size: 0; }
[data-rcl-search-input-option-label] { overflow-wrap: anywhere; font: var(--text-body-sm); }
[data-rcl-search-input-option-description] {
  color: var(--color-muted-foreground);
  font: var(--text-caption);
  overflow-wrap: anywhere;
}
[data-rcl-search-input-option-leading] {
  margin-block-start: 1px;
  color: var(--color-muted-foreground);
}
[data-rcl-search-input-match] {
  border-radius: var(--radius-control);
  background: color-mix(in srgb, var(--color-primary) 22%, transparent);
  color: var(--color-foreground);
}

[data-rcl-search-input-state] {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  padding: var(--space-sm);
  color: var(--color-muted-foreground);
  font: var(--text-body-sm);
}
[data-rcl-search-input-state][data-tone="error"] { color: var(--color-danger); }
[data-rcl-search-input-state] strong { color: var(--color-foreground); font: var(--text-label); }
[data-rcl-search-input-state-copy] { display: grid; gap: var(--space-3xs); min-inline-size: 0; }
[data-rcl-search-input-retry] {
  justify-self: start;
  min-block-size: var(--tap-target-min);
  padding-inline: var(--space-sm);
  border: var(--border-hairline) solid var(--color-primary);
  border-radius: var(--radius-control);
  background: var(--color-primary);
  color: var(--color-primary-foreground);
  font: var(--text-label);
  cursor: pointer;
}

@media (prefers-reduced-motion: reduce) {
  [data-rcl-search-input] * { transition: none; animation-duration: 0.01ms; }
}
`;

const SUGGESTION_LIMIT = 12;

function normalizeSuggestion(item: string | SearchInputSuggestion): SearchInputSuggestion {
  return typeof item === "string"
    ? { value: item, label: item }
    : { ...item, label: item.label ?? item.value };
}

function uniqueSuggestions(
  items: readonly (string | SearchInputSuggestion)[],
): SearchInputSuggestion[] {
  const seen = new Set<string>();
  const resolved: SearchInputSuggestion[] = [];
  for (const item of items) {
    const suggestion = normalizeSuggestion(item);
    if (seen.has(suggestion.value)) continue;
    seen.add(suggestion.value);
    resolved.push(suggestion);
    if (resolved.length >= SUGGESTION_LIMIT) break;
  }
  return resolved;
}

/**
 * A native change event for the clear action and suggestion selection.
 *
 * `onChange` is kept because 1.1.x exposed raw input props and existing
 * consumers (FilterBar among them) drive a controlled query through it. A
 * button-driven value change has no native event of its own, so the field
 * supplies the smallest event shape those consumers read: `currentTarget.value`.
 * New code should use `onValueChange`.
 */
function changeEventFor(value: string): ChangeEvent<HTMLInputElement> {
  return {
    target: { value },
    currentTarget: { value },
  } as unknown as ChangeEvent<HTMLInputElement>;
}

function highlightMatch(text: string, query: string): ReactNode {
  const needle = query.trim();
  if (!needle) return text;
  const index = text.toLocaleLowerCase().indexOf(needle.toLocaleLowerCase());
  if (index < 0) return text;
  return (
    <>
      {text.slice(0, index)}
      <mark data-rcl-search-input-match>{text.slice(index, index + needle.length)}</mark>
      {text.slice(index + needle.length)}
    </>
  );
}

/** Points Popover's positioning context at the field it belongs to. */
function SearchAnchor({ controlRef }: { controlRef: RefObject<HTMLElement | null> }) {
  const { triggerRef } = usePopover();
  useLayoutEffect(() => {
    triggerRef.current = controlRef.current;
    return () => {
      if (triggerRef.current === controlRef.current) triggerRef.current = null;
    };
  }, [controlRef, triggerRef]);
  return null;
}

interface AsyncState {
  status: SearchInputStatus;
  items: SearchInputSuggestion[];
  error?: string;
}

const IDLE_ASYNC: AsyncState = { status: "idle", items: [] };

/**
 * The library's one search query field.
 *
 * `form` is the labelled convention: a visible name, a rounded 44px field, and
 * a comfortable reading width. `toolbar` is the command-strip convention: a
 * hidden name, a pill, a magnifier, and an optional shortcut hint.
 *
 * The field owns the query contract, not the result surface. It can filter
 * static `suggestions`, or own a debounced, abortable `onSearch` and render the
 * resulting suggestions in an RCL `Popover` listbox. Choosing a result is
 * reported to the host; the host still decides what selecting a query means.
 */
export const SearchInput = forwardRef<HTMLInputElement, SearchInputProps>(function SearchInput(
  {
    value: valueProp,
    defaultValue = "",
    onValueChange,
    onChange,
    label,
    labelVisibility,
    description,
    error,
    appearance = "form",
    size,
    shape,
    leading,
    actions,
    clearable = true,
    onClear,
    clearLabel,
    shortcut,
    shortcutLabel,
    loading: loadingProp = false,
    loadingLabel,
    onSearch,
    debounceMs = 220,
    minimumQueryLength = 1,
    searchError: searchErrorProp,
    onSearchError,
    onRetry,
    suggestions,
    recentQueries,
    suggestionsLabel,
    recentLabel,
    emptyLabel,
    onSuggestionSelect,
    onRecentSelect,
    onClearRecent,
    clearRecentLabel,
    open: openProp,
    defaultOpen = false,
    onOpenChange,
    onSubmit,
    panelTestId,
    optionTestId,
    testId,
    groupTestId,
    className,
    style,
    inputClassName,
    inputStyle,
    autoFocus,
    disabled = false,
    readOnly = false,
    required = false,
    name,
    maxLength,
    id: idProp,
    onFocus,
    onBlur,
    onKeyDown,
    "aria-label": ariaLabel,
    "aria-describedby": ariaDescribedBy,
    ...inputProps
  },
  ref,
) {
  const strings = useStrings();
  useLibraryStyleSheet("react-component-library:SearchInput", "1.2.1", styles);
  const generatedId = useId().replace(/:/g, "");
  const inputId = idProp ?? `search-input-${generatedId}`;
  const listId = `search-input-list-${generatedId}`;
  const labelId = `search-input-label-${generatedId}`;
  const descriptionId = `search-input-description-${generatedId}`;
  const errorId = `search-input-error-${generatedId}`;
  const inputTestId = testId ?? "forms.search-input";
  const resolvedGroupTestId = groupTestId ?? `${inputTestId}-group`;
  const resolvedPanelTestId = panelTestId ?? "forms.search-input-panel";
  const resolvedSize = size ?? (appearance === "toolbar" ? "md" : "lg");
  const resolvedShape = shape ?? (appearance === "toolbar" ? "pill" : "rounded");
  const resolvedLabelVisibility =
    labelVisibility ?? (appearance === "toolbar" ? "hidden" : "visible");

  const [value, setValue] = useControllableState<string>({
    value: valueProp,
    defaultValue: defaultValue ?? "",
    onChange: onValueChange,
  });
  const [open, setOpen] = useControllableState<boolean>({
    value: openProp,
    defaultValue: defaultOpen,
    onChange: onOpenChange,
  });
  const [activeIndex, setActiveIndex] = useState(-1);
  const [asyncState, setAsyncState] = useState<AsyncState>(IDLE_ASYNC);
  const [retryNonce, setRetryNonce] = useState(0);

  const controlRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement | null>(null);
  const controlRect = useElementRect(controlRef as RefObject<HTMLElement | null>);
  const setInputRef = useCallback(
    (node: HTMLInputElement | null) => {
      inputRef.current = node;
      if (typeof ref === "function") ref(node);
      else if (ref) (ref as MutableRefObject<HTMLInputElement | null>).current = node;
    },
    [ref],
  );

  // Async search reads the latest host callback and query without making the
  // effect restart on every inline function identity.
  const hasOnSearch = onSearch !== undefined;
  const onSearchRef = useRef(onSearch);
  const onSearchErrorRef = useRef(onSearchError);
  const valueRef = useRef(value);
  useEffect(() => {
    onSearchRef.current = onSearch;
    onSearchErrorRef.current = onSearchError;
    valueRef.current = value;
  }, [onSearch, onSearchError, value]);

  const task = useCallback((signal: AbortSignal) => {
    const search = onSearchRef.current;
    return search ? search(valueRef.current.trim(), { signal }) : Promise.resolve([]);
  }, []);
  const { run, abort } = useAbortableTask(task);

  useEffect(() => {
    if (!hasOnSearch) {
      abort();
      setAsyncState((current) =>
        current.status === "idle" && current.items.length === 0 ? current : IDLE_ASYNC,
      );
      return;
    }
    const query = value.trim();
    if (!open || disabled || query.length < minimumQueryLength) {
      abort();
      setAsyncState(IDLE_ASYNC);
      return;
    }
    let cancelled = false;
    setAsyncState({ status: "loading", items: [] });
    const timer = setTimeout(
      () => {
        void run()
          .then((items) => {
            if (cancelled) return;
            setAsyncState({ status: "ready", items: uniqueSuggestions(items) });
          })
          .catch((caught: unknown) => {
            if (cancelled) return;
            if (caught instanceof DOMException && caught.name === "AbortError") return;
            const message = caught instanceof Error && caught.message ? caught.message : undefined;
            setAsyncState({
              status: "error",
              items: [],
              ...(message ? { error: message } : {}),
            });
            onSearchErrorRef.current?.(caught);
          });
      },
      Math.max(0, debounceMs),
    );
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [abort, debounceMs, disabled, hasOnSearch, minimumQueryLength, open, retryNonce, run, value]);

  const normalizedSuggestions = useMemo(() => uniqueSuggestions(suggestions ?? []), [suggestions]);
  const query = value.trim();
  const staticMatches = useMemo(() => {
    if (!query) return normalizedSuggestions;
    const needle = query.toLocaleLowerCase();
    return normalizedSuggestions.filter((suggestion) => {
      const haystack = `${suggestion.label ?? ""} ${suggestion.value} ${suggestion.description ?? ""}`;
      return haystack.toLocaleLowerCase().includes(needle);
    });
  }, [normalizedSuggestions, query]);

  const recentOptions: SearchInputSuggestion[] = useMemo(
    () => (recentQueries ?? []).map((item) => ({ value: item, label: item })),
    [recentQueries],
  );
  const showRecent = query.length === 0 && recentOptions.length > 0;
  const suggestionSource = onSearch !== undefined || suggestions !== undefined;
  const loading = loadingProp || asyncState.status === "loading";
  const resolvedSearchError =
    searchErrorProp ??
    (asyncState.status === "error"
      ? strings("forms.search-input.error", "Search is unavailable")
      : undefined);
  const panelOptions: SearchInputSuggestion[] = showRecent
    ? recentOptions
    : onSearch
      ? asyncState.items
      : staticMatches;
  const showEmpty =
    suggestionSource &&
    !showRecent &&
    query.length >= minimumQueryLength &&
    !loading &&
    !resolvedSearchError &&
    panelOptions.length === 0 &&
    (onSearch ? asyncState.status === "ready" : true);
  const panelVisible =
    open && (panelOptions.length > 0 || loading || Boolean(resolvedSearchError) || showEmpty);
  const optionSignature = `${showRecent ? "recent" : "results"}|${panelOptions
    .map((option) => option.value)
    .join("\u0000")}|${String(panelOptions.length)}`;

  useEffect(() => {
    setActiveIndex(panelOptions.length > 0 ? 0 : -1);
    // The signature is the list's identity; a new object with the same values
    // must not reset the reader's place.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [optionSignature]);

  const optionId = useCallback((index: number) => `${listId}-option-${String(index)}`, [listId]);
  const activeOption = activeIndex >= 0 ? panelOptions[activeIndex] : undefined;
  const activeOptionId = activeOption ? optionId(activeIndex) : undefined;

  const emitValue = useCallback(
    (next: string, nativeEvent?: ChangeEvent<HTMLInputElement>) => {
      setValue(next);
      if (nativeEvent) onChange?.(nativeEvent);
      else onChange?.(changeEventFor(next));
    },
    [onChange, setValue],
  );

  const choose = useCallback(
    (suggestion: SearchInputSuggestion, index: number) => {
      emitValue(suggestion.value);
      if (showRecent) onRecentSelect?.(suggestion.value, index);
      else onSuggestionSelect?.(suggestion, index);
      setOpen(false);
      requestAnimationFrame(() => inputRef.current?.focus());
    },
    [emitValue, onRecentSelect, onSuggestionSelect, setOpen, showRecent],
  );

  const clear = useCallback(() => {
    if (value.length === 0) return;
    emitValue("");
    onClear?.();
    setOpen(recentOptions.length > 0);
    inputRef.current?.focus();
  }, [emitValue, onClear, recentOptions.length, setOpen, value.length]);

  const retry = useCallback(() => {
    onRetry?.();
    setRetryNonce((current) => current + 1);
  }, [onRetry]);

  const handleKeyDown = useCallback(
    (event: ReactKeyboardEvent<HTMLInputElement>) => {
      onKeyDown?.(event);
      if (event.defaultPrevented) return;

      if (event.key === "Escape") {
        if (open) {
          event.preventDefault();
          setOpen(false);
        }
        return;
      }
      if (event.key === "Tab") {
        if (open) setOpen(false);
        return;
      }

      const count = panelOptions.length;
      if (event.key === "ArrowDown" || event.key === "ArrowUp") {
        if (!suggestionSource) return;
        event.preventDefault();
        if (!open) {
          setOpen(true);
          setActiveIndex(event.key === "ArrowDown" ? 0 : Math.max(0, count - 1));
          return;
        }
        if (count === 0) return;
        setActiveIndex((current) => {
          if (current < 0) return event.key === "ArrowDown" ? 0 : count - 1;
          return (current + (event.key === "ArrowDown" ? 1 : -1) + count) % count;
        });
        return;
      }
      if ((event.key === "Home" || event.key === "End") && open && count > 0) {
        event.preventDefault();
        setActiveIndex(event.key === "Home" ? 0 : count - 1);
        return;
      }
      if (event.key === "Enter") {
        if (open && activeOption) {
          event.preventDefault();
          choose(activeOption, activeIndex);
          return;
        }
        onSubmit?.(value);
      }
    },
    [
      activeIndex,
      activeOption,
      choose,
      onKeyDown,
      onSubmit,
      open,
      panelOptions.length,
      setOpen,
      suggestionSource,
      value,
    ],
  );

  const handleChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      emitValue(event.currentTarget.value, event);
      if (suggestionSource) setOpen(true);
    },
    [emitValue, setOpen, suggestionSource],
  );

  const handleFocus = useCallback(
    (event: React.FocusEvent<HTMLInputElement>) => {
      onFocus?.(event);
      if (event.defaultPrevented) return;
      if (suggestionSource) setOpen(true);
    },
    [onFocus, setOpen, suggestionSource],
  );

  const describedBy = [
    ariaDescribedBy,
    description ? descriptionId : undefined,
    error ? errorId : undefined,
  ]
    .filter(Boolean)
    .join(" ");

  const hasLabelElement = label !== undefined || (!ariaLabel && appearance === "form");
  const labelContent = label ?? strings("forms.search-input.search", "Search");
  const inputAriaLabel = hasLabelElement
    ? undefined
    : (ariaLabel ?? strings("forms.search-input.search", "Search"));
  const resolvedLeading =
    leading !== undefined ? leading : appearance === "toolbar" ? <Search aria-hidden /> : null;
  const panelWidth = controlRect?.width ? Math.max(240, Math.min(controlRect.width, 640)) : 320;

  return (
    <Popover
      open={panelVisible}
      onOpenChange={(next) => {
        if (!next) setOpen(false);
      }}
      placement="bottom-start"
      responsive="none"
      restoreFocus={false}
    >
      <div
        data-rcl-search-input="true"
        data-appearance={appearance}
        data-size={resolvedSize}
        data-shape={resolvedShape}
        data-label-visibility={resolvedLabelVisibility}
        data-open={panelVisible ? "true" : undefined}
        data-disabled={disabled ? "true" : undefined}
        className={className}
        style={style}
      >
        {hasLabelElement && (
          <label
            id={labelId}
            htmlFor={inputId}
            data-rcl-search-input-label="true"
            data-visibility={resolvedLabelVisibility}
          >
            {labelContent}
          </label>
        )}
        <div ref={controlRef} data-rcl-search-input-control="true">
          <InputGroup
            testId={resolvedGroupTestId}
            size={resolvedSize}
            shape={resolvedShape}
            tone={error ? "invalid" : "default"}
            disabled={disabled}
          >
            {resolvedLeading ? (
              <InputGroup.Adornment side="leading">{resolvedLeading}</InputGroup.Adornment>
            ) : null}
            <InputGroup.Field>
              <Input
                {...inputProps}
                ref={setInputRef}
                id={inputId}
                type="search"
                name={name}
                value={value}
                disabled={disabled}
                readOnly={readOnly}
                required={required}
                maxLength={maxLength}
                autoFocus={autoFocus}
                autoComplete={inputProps.autoComplete ?? "off"}
                enterKeyHint={inputProps.enterKeyHint ?? "search"}
                data-testid={inputTestId}
                className={cn("rcl-search-input__field", inputClassName)}
                style={inputStyle}
                role={suggestionSource ? "combobox" : undefined}
                aria-label={inputAriaLabel}
                aria-describedby={describedBy || undefined}
                aria-invalid={error ? true : undefined}
                aria-busy={loading || undefined}
                aria-autocomplete={suggestionSource ? "list" : undefined}
                aria-haspopup={suggestionSource ? "listbox" : undefined}
                aria-controls={panelVisible ? listId : undefined}
                aria-expanded={suggestionSource ? panelVisible : undefined}
                aria-activedescendant={panelVisible ? activeOptionId : undefined}
                onChange={handleChange}
                onFocus={handleFocus}
                onBlur={onBlur}
                onKeyDown={handleKeyDown}
              />
            </InputGroup.Field>
            {actions ? (
              <InputGroup.Action testId={`${resolvedGroupTestId}-actions`}>
                {actions}
              </InputGroup.Action>
            ) : null}
            {loading ? (
              <InputGroup.Action testId={`${resolvedGroupTestId}-loading`}>
                <Spinner
                  label={loadingLabel ?? strings("forms.search-input.loading", "Searching")}
                />
              </InputGroup.Action>
            ) : null}
            {clearable && value.length > 0 && !disabled && !readOnly ? (
              <InputGroup.Action testId={`${resolvedGroupTestId}-clear`}>
                <IconButton
                  data-testid={`${inputTestId}-clear`}
                  aria-label={clearLabel ?? strings("forms.search-input.clear", "Clear search")}
                  title={clearLabel ?? strings("forms.search-input.clear", "Clear search")}
                  shape="circle"
                  surface="ghost"
                  size="xs"
                  onClick={clear}
                >
                  <X aria-hidden />
                </IconButton>
              </InputGroup.Action>
            ) : null}
            {shortcut ? (
              <InputGroup.Action testId={`${resolvedGroupTestId}-shortcut`}>
                <kbd data-rcl-search-input-shortcut="true" title={shortcutLabel} aria-hidden="true">
                  {shortcut}
                </kbd>
              </InputGroup.Action>
            ) : null}
          </InputGroup>
        </div>
        {description ? (
          <span id={descriptionId} data-rcl-search-input-description="true">
            {description}
          </span>
        ) : null}
        {error ? (
          <span id={errorId} data-rcl-search-input-error="true" role="alert">
            {error}
          </span>
        ) : null}
      </div>
      <SearchAnchor controlRef={controlRef} />
      {panelVisible ? (
        <PopoverContent
          role="presentation"
          initialFocus="none"
          data-testid={resolvedPanelTestId}
          data-rcl-search-input-panel="true"
          style={{ inlineSize: `min(${String(panelWidth)}px, calc(100vw - var(--space-lg) * 2))` }}
        >
          <div
            id={listId}
            role="listbox"
            aria-label={
              suggestionsLabel ?? strings("forms.search-input.suggestions", "Search suggestions")
            }
            data-rcl-search-input-list="true"
          >
            {loading ? (
              <div data-rcl-search-input-state="true" role="status" aria-live="polite">
                <Spinner
                  label={loadingLabel ?? strings("forms.search-input.loading", "Searching")}
                />
                <span>{loadingLabel ?? strings("forms.search-input.loading", "Searching…")}</span>
              </div>
            ) : resolvedSearchError ? (
              <div data-rcl-search-input-state="true" data-tone="error" role="alert">
                <div data-rcl-search-input-state-copy="true">
                  <strong>
                    {strings("forms.search-input.error-title", "Search is unavailable")}
                  </strong>
                  <span>{resolvedSearchError}</span>
                </div>
                {onSearch || onRetry ? (
                  <button
                    type="button"
                    data-rcl-search-input-retry="true"
                    onMouseDown={(event) => {
                      event.preventDefault();
                    }}
                    onClick={retry}
                  >
                    {strings("forms.search-input.retry", "Try again")}
                  </button>
                ) : null}
              </div>
            ) : showEmpty ? (
              <div data-rcl-search-input-state="true" role="status">
                {emptyLabel ?? (
                  <>
                    {strings("forms.search-input.no-results", "No results for")}{" "}
                    <strong>“{query}”</strong>
                  </>
                )}
              </div>
            ) : (
              <>
                {showRecent ? (
                  <div data-rcl-search-input-section="true">
                    <span>
                      {recentLabel ?? strings("forms.search-input.recent", "Recent searches")}
                    </span>
                    {onClearRecent ? (
                      <button
                        type="button"
                        data-rcl-search-input-section-clear="true"
                        onMouseDown={(event) => {
                          event.preventDefault();
                        }}
                        onClick={onClearRecent}
                      >
                        {clearRecentLabel ?? strings("forms.search-input.clear-recent", "Clear")}
                      </button>
                    ) : null}
                  </div>
                ) : null}
                {panelOptions.map((option, index) => {
                  const active = index === activeIndex;
                  const id = optionId(index);
                  return (
                    <button
                      key={`${option.value}-${String(index)}`}
                      id={id}
                      type="button"
                      role="option"
                      aria-selected={active}
                      data-rcl-search-input-option="true"
                      data-active={active ? "true" : undefined}
                      data-testid={
                        optionTestId
                          ? optionTestId(option, index)
                          : `forms.search-input-option-${String(index)}`
                      }
                      onMouseDown={(event) => {
                        event.preventDefault();
                      }}
                      onMouseEnter={() => {
                        setActiveIndex(index);
                      }}
                      onClick={() => {
                        choose(option, index);
                      }}
                    >
                      <span data-rcl-search-input-option-copy="true">
                        <span data-rcl-search-input-option-label="true">
                          {showRecent
                            ? highlightMatch(option.label ?? option.value, "")
                            : highlightMatch(option.label ?? option.value, query)}
                        </span>
                        {option.description ? (
                          <span data-rcl-search-input-option-description="true">
                            {option.description}
                          </span>
                        ) : null}
                      </span>
                      {showRecent ? (
                        <Clock data-rcl-search-input-option-leading="true" aria-hidden size={14} />
                      ) : null}
                    </button>
                  );
                })}
              </>
            )}
          </div>
        </PopoverContent>
      ) : null}
    </Popover>
  );
});
