/** @vrooliComponentSource hooks.use-disclosure */
import { useCallback, useState } from "react";

function isUpdater<T>(
  candidate: T | ((previous: T) => T),
): candidate is (previous: T) => T {
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

export function useDisclosure(
  options: {
    open?: boolean;
    defaultOpen?: boolean;
    onOpenChange?: (open: boolean) => void;
  } = {},
) {
  const [open, setOpen] = useControllableState({
    value: options.open,
    defaultValue: options.defaultOpen ?? false,
    onChange: options.onOpenChange,
  });
  return {
    open,
    setOpen,
    onOpen: useCallback(() => setOpen(true), [setOpen]),
    onClose: useCallback(() => setOpen(false), [setOpen]),
    onToggle: useCallback(() => setOpen((previous) => !previous), [setOpen]),
  };
}
