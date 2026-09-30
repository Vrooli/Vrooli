/**
 * @libraryId react-component-library:useControllableState
 * @displayName useControllableState
 * @description The controlled-or-uncontrolled state primitive that preserves one event contract and prevents a component from maintaining competing internal and external values.
 * @version 1.0.1
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:useControllableState
 * @vrooliComponentSourceSlot hooks.use-controllable-state */
import { useCallback, useState } from "react";

function isUpdater<T>(candidate: T | ((previous: T) => T)): candidate is (previous: T) => T {
  return typeof candidate === "function";
}

export function useControllableState<T>({
  value,
  defaultValue,
  onChange,
}: {
  value?: T;
  defaultValue: T | (() => T);
  onChange?: (next: T) => void;
}) {
  const [uncontrolled, setUncontrolled] = useState<T>(defaultValue);
  const controlled = value !== undefined;
  const current = controlled ? value : uncontrolled;
  const setValue = useCallback(
    (next: T | ((previous: T) => T)) => {
      const resolved = isUpdater(next) ? next(current) : next;
      if (!controlled) setUncontrolled(resolved);
      onChange?.(resolved);
    },
    [controlled, current, onChange],
  );
  return [current, setValue] as const;
}
