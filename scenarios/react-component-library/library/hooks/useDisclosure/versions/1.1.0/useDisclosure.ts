/**
 * @libraryId react-component-library:useDisclosure
 * @displayName useDisclosure
 * @version 1.1.0
 * @tags ["runtime","accessibility"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource hooks.use-disclosure */
import { useCallback } from "react";
import { useControllableState } from "@vrooli/react-component-library/useControllableState/1";

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
