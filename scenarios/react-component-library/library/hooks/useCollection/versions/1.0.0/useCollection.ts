/** @libraryId react-component-library:useCollection */
/** @vrooliComponentSource hooks.use-collection */
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent,
  type PointerEvent,
} from "react";
import {
  createSelectionStore,
  type SelectionMode,
  type SelectionStore,
} from "@vrooli/react-component-library/SelectionStore/2";
import { useLongPress } from "@vrooli/react-component-library/useLongPress/1";

interface CollectionCommand {
  id: string;
  label: string;
  shortcut?: string;
  disabled?: boolean;
  run: () => unknown;
}
interface CommandRegistry {
  register: (command: CollectionCommand) => () => void;
}

export interface RowAction<T> {
  id: string;
  label: string;
  icon?: React.ReactNode;
  tone?: "neutral" | "primary" | "destructive";
  shortcut?: string;
  swipe?: "leading" | "trailing";
  bulk?: boolean;
  undoable?: boolean;
  separatorBefore?: boolean;
  hidden?: (row: T) => boolean;
  disabled?: (row: T) => string | false;
  onSelect: (rows: T[]) => void | Promise<void>;
}
export interface RowSelection {
  selectionMode: boolean;
  selected: boolean;
  disabled?: boolean;
  disabledReason?: string;
  onToggleSelect?: () => void;
}
export interface CollectionRowState {
  key: string;
  index: number;
  isCursor: boolean;
  selection: RowSelection;
}
export interface BulkOutcome {
  id: string;
  status: "success" | "failed" | "skipped";
  error?: string;
}
export interface UseCollectionOptions<T> {
  getKey: (row: T) => string;
  getSearchText?: (row: T) => string;
  actions?: readonly RowAction<T>[];
  primaryActionId?: string;
  search?: (row: T, q: string) => boolean;
  query?: string;
  filter?: (row: T) => boolean;
  sort?: (a: T, b: T) => number;
  selection?: {
    mode: SelectionMode;
    enterOn?: readonly ("long-press" | "checkbox" | "shortcut")[];
    retain?: "prune" | "keep";
    selectable?: (row: T) => string | false;
    selected?: readonly string[];
    onChange?: (keys: string[]) => void;
  };
  onOpen?: (row: T) => void;
  announce?: (message: string) => void;
  registerCommands?: boolean;
  commandRegistry?: CommandRegistry;
}
export interface ResolvedRowAction {
  id: string;
  label: string;
  icon?: React.ReactNode;
  tone: RowAction<unknown>["tone"];
  shortcut?: string;
  swipe?: RowAction<unknown>["swipe"];
  disabledReason?: string;
  run: () => unknown;
}
export interface UseCollectionResult<T> {
  rows: readonly T[];
  rowStateFor: (row: T) => CollectionRowState;
  getContainerProps: () => Record<string, unknown>;
  getRowProps: (row: T) => Record<string, unknown>;
  actionsFor: (row: T) => readonly ResolvedRowAction[];
  bulk: {
    count: number;
    actions: readonly ResolvedRowAction[];
    run: (actionId: string) => Promise<BulkOutcome[]>;
    clear: () => void;
    selectAllVisible: () => void;
  };
  cursorKey: string | null;
  selectionMode: boolean;
  exitSelection: () => void;
}

export function useCollection<T>(
  items: readonly T[],
  options: UseCollectionOptions<T>,
): UseCollectionResult<T> {
  const {
    getKey,
    getSearchText,
    actions = [],
    query = "",
    search,
    filter,
    sort,
    selection = { mode: "none" },
    onOpen,
    announce,
    registerCommands = false,
    commandRegistry,
  } = options;
  const { enterOn, retain = "prune", selectable, onChange, selected } = selection;
  const [, refresh] = useState(0);
  const storeRef = useRef<SelectionStore>();
  if (!storeRef.current)
    storeRef.current = createSelectionStore([], selection.mode);
  const store = storeRef.current;
  const requestedMode = useRef(selection.mode);
  const [cursor, setCursor] = useState<string | null>(null);
  const pendingLongPressKey = useRef<string | null>(null);
  const previousKeys = useRef("");

  const rows = useMemo(() => {
    const result = items.filter(
      (row) =>
        (!query || !search || search(row, query)) && (!filter || filter(row)),
    );
    return sort ? [...result].sort(sort) : result;
  }, [filter, items, query, search, sort]);
  const keys = useMemo(() => rows.map(getKey), [getKey, rows]);
  const selectableRows = useMemo(
    () => rows.filter((row) => !selectable?.(row)),
    [rows, selectable],
  );
  const selectableKeys = useMemo(
    () => selectableRows.map(getKey),
    [getKey, selectableRows],
  );

  useEffect(() => {
    if (requestedMode.current !== selection.mode) {
      requestedMode.current = selection.mode;
      store.setMode(selection.mode);
      refresh((value) => value + 1);
    }
  }, [selection.mode, store]);
  useEffect(() => {
    if (selected && onChange) {
      const current = store.getSnapshot().keys;
      const next = new Set(selected);
      if (
        current.size !== next.size ||
        [...current].some((key) => !next.has(key))
      )
        store.setSelected(selected);
    }
  }, [onChange, selected, store]);
  useEffect(() => {
    const signature = keys.join("\u0000");
    if (signature === previousKeys.current) return;
    previousKeys.current = signature;
    store.retain(keys, retain);
    setCursor((current) =>
      current && keys.includes(current) ? current : (keys[0] ?? null),
    );
    refresh((value) => value + 1);
  }, [keys, retain, store]);

  const announceSelection = useCallback(
    () => announce?.(`${store.size()} selected`),
    [announce, store],
  );
  const mutate = useCallback(
    (fn: () => void) => {
      fn();
      refresh((value) => value + 1);
      onChange?.([...store.getSnapshot().keys]);
      announceSelection();
    },
    [announceSelection, onChange, store],
  );
  const enterSelection = useCallback(
    (key: string) => {
      if (!enterOn?.includes("long-press")) return;
      const row = rows.find((candidate) => getKey(candidate) === key);
      if (!row || selectable?.(row)) return;
      if (store.getSnapshot().mode === "none") store.setMode("multi");
      if (!store.isSelected(key)) store.toggle(key);
      refresh((value) => value + 1);
      onChange?.([...store.getSnapshot().keys]);
      announce?.("Selection mode. 1 selected");
    },
    [announce, enterOn, getKey, onChange, rows, selectable, store],
  );
  const enterSelectionFromKeyboard = useCallback(() => {
    if (store.getSnapshot().mode !== "none") return true;
    if (!enterOn?.includes("shortcut")) return false;
    store.setMode("multi");
    return true;
  }, [enterOn, store]);
  const longPress = useLongPress({
    onLongPress: () => {
      const key = pendingLongPressKey.current;
      if (key) enterSelection(key);
    },
    disabled: !enterOn?.includes("long-press"),
  });

  const reasonFor = useCallback(
    (row: T) => selectable?.(row) || undefined,
    [selectable],
  );
  const actionsFor = useCallback(
    (row: T): ResolvedRowAction[] =>
      actions
        .filter((action) => !action.hidden?.(row))
        .map((action) => {
          const disabledReason = action.disabled?.(row) || undefined;
          return {
            id: action.id,
            label: action.label,
            icon: action.icon,
            tone: action.tone,
            shortcut: action.shortcut,
            swipe: action.swipe,
            disabledReason,
            run: () => (disabledReason ? undefined : action.onSelect([row])),
          };
        }),
    [actions],
  );
  const commands = useMemo<CollectionCommand[]>(() => {
    if (!registerCommands) return [];
    return rows.flatMap((row) =>
      actionsFor(row)
        .filter((action) => Boolean(action.shortcut))
        .map((action) => ({
          id: `collection:${getKey(row)}:${action.id}`,
          label: `${action.label}: ${getSearchText?.(row) ?? getKey(row)}`,
          shortcut: action.shortcut,
          disabled: Boolean(action.disabledReason),
          run: async () => {
            await action.run();
          },
        })),
    );
  }, [actionsFor, getKey, getSearchText, registerCommands, rows]);
  useEffect(() => {
    if (!registerCommands || !commandRegistry) return undefined;
    const unregister = commands.map((command) => commandRegistry.register(command));
    return () => unregister.forEach((remove) => remove());
  }, [commandRegistry, commands, registerCommands]);
  const rowStateFor = useCallback(
    (row: T): CollectionRowState => {
      const key = getKey(row);
      const disabledReason = reasonFor(row);
      return {
        key,
        index: rows.indexOf(row),
        isCursor: key === cursor,
        selection: {
          selectionMode: store.getSnapshot().mode !== "none",
          selected: store.isSelected(key),
          disabled: !!disabledReason,
          disabledReason,
          onToggleSelect: disabledReason
            ? undefined
            : () => mutate(() => store.toggle(key)),
        },
      };
    },
    [cursor, getKey, mutate, reasonFor, rows, store],
  );

  const moveCursor = (delta: number, range: boolean) => {
    const navigationKeys = range ? selectableKeys : keys;
    const index = Math.max(0, navigationKeys.indexOf(cursor ?? navigationKeys[0] ?? ""));
    const next = Math.min(navigationKeys.length - 1, Math.max(0, index + delta));
    const key = navigationKeys[next];
    if (!key) return;
    setCursor(key);
    if (range) mutate(() => store.extendTo(key, selectableKeys));
  };
  const keyboard = (event: KeyboardEvent) => {
    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      moveCursor(event.key === "ArrowDown" ? 1 : -1, event.shiftKey);
    } else if (
      (event.metaKey || event.ctrlKey) &&
      event.key.toLowerCase() === "a"
    ) {
      event.preventDefault();
      if (enterSelectionFromKeyboard()) mutate(() => store.selectAll(selectableKeys));
    } else if (event.key === " ") {
      event.preventDefault();
      const row = rows.find((candidate) => getKey(candidate) === cursor);
      const selectedKey = cursor;
      if (row && selectedKey && !reasonFor(row) && (store.getSnapshot().mode !== "none" || enterSelectionFromKeyboard()))
        mutate(() => store.toggle(selectedKey));
    } else if (event.key === "Escape") {
      mutate(() => store.setMode("none"));
    } else if (event.key === "Enter" && cursor) {
      const row = rows.find((candidate) => getKey(candidate) === cursor);
      const action =
        (row &&
          actionsFor(row).find(
            (entry) => entry.id === options.primaryActionId,
          )) ||
        (row && actionsFor(row)[0]);
      if (action && !action.disabledReason) void action.run();
    } else if (
      event.key.length === 1 &&
      !event.altKey &&
      !event.ctrlKey &&
      !event.metaKey
    ) {
      const found = rows.find((row) =>
        (getSearchText?.(row) ?? String(row)).toLocaleLowerCase().includes(event.key.toLocaleLowerCase()),
      );
      if (found) setCursor(getKey(found));
    }
  };
  const getContainerProps = () => ({
    onKeyDown: keyboard,
    role: store.getSnapshot().mode === "none" ? "list" : "listbox",
    tabIndex: 0,
  });
  const getRowProps = (row: T) => {
    const key = getKey(row);
    const state = rowStateFor(row);
    return {
      role: store.getSnapshot().mode === "none" ? "listitem" : "option",
      tabIndex: state.isCursor ? 0 : -1,
      "aria-selected":
        store.getSnapshot().mode === "none"
          ? undefined
          : state.selection.selected,
      onPointerDown: (event: PointerEvent<HTMLElement>) => {
        pendingLongPressKey.current = key;
        longPress.longPressProps.onPointerDown?.(event);
      },
      onPointerMove: (event: PointerEvent<HTMLElement>) =>
        longPress.longPressProps.onPointerMove?.(event),
      onPointerUp: (event: PointerEvent<HTMLElement>) => {
        longPress.longPressProps.onPointerUp?.(event);
        pendingLongPressKey.current = null;
      },
      onPointerCancel: (event: PointerEvent<HTMLElement>) => {
        longPress.longPressProps.onPointerCancel?.(event);
        pendingLongPressKey.current = null;
      },
      onClick: (event: React.MouseEvent<HTMLElement>) => {
        longPress.longPressProps.onClick?.(event);
        if (event.defaultPrevented) return;
        if (reasonFor(row)) return;
        if (event.shiftKey) mutate(() => store.extendTo(key, selectableKeys));
        else if (event.metaKey || event.ctrlKey || store.getSnapshot().mode !== "none") mutate(() => store.toggle(key));
        else onOpen?.(row);
      },
    };
  };
  const bulkActions = actions
    .filter((action) => action.bulk)
    .flatMap((action) => {
      const selectedRows = rows.filter((row) => store.isSelected(getKey(row)));
      const sample = selectedRows[0] ?? rows[0];
      if (!sample) return [];
      const resolved = actionsFor(sample).find(
        (entry) => entry.id === action.id,
      );
      if (!resolved) return [];
      return {
        ...resolved,
        run: async () => {
          const outcomes: BulkOutcome[] = [];
          for (const row of selectedRows) {
            const disabledReason = action.disabled?.(row);
            if (disabledReason) {
              outcomes.push({
                id: getKey(row),
                status: "skipped",
                error: disabledReason,
              });
              continue;
            }
            try {
              await action.onSelect([row]);
              outcomes.push({ id: getKey(row), status: "success" });
            } catch (error) {
              outcomes.push({
                id: getKey(row),
                status: "failed",
                error: error instanceof Error ? error.message : String(error),
              });
            }
          }
          const failed = outcomes.filter(
            (outcome) => outcome.status === "failed",
          );
          announce?.(
            failed.length
              ? `${failed.length} action${failed.length === 1 ? "" : "s"} failed`
              : `${outcomes.length} action${outcomes.length === 1 ? "" : "s"} completed`,
          );
          return outcomes;
        },
      };
    });
  return {
    rows,
    rowStateFor,
    getContainerProps,
    getRowProps,
    actionsFor,
    bulk: {
      count: store.size(),
      actions: bulkActions,
      run: async (actionId) => {
        const action = bulkActions.find((entry) => entry.id === actionId);
        return action ? action.run() : [];
      },
      clear: () => mutate(() => store.clear()),
      selectAllVisible: () => mutate(() => store.selectAll(selectableKeys)),
    },
    cursorKey: cursor,
    selectionMode: store.getSnapshot().mode !== "none",
    exitSelection: () => mutate(() => store.setMode("none")),
  };
}
