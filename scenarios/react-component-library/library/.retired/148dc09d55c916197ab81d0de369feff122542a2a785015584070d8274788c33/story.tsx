import { useRef } from "react";
import { ContextMenu } from "./ContextMenu";

const basicItems = [
  { id: "open", label: "Open", shortcut: "Enter", onSelect() {} },
  { id: "rename", label: "Rename", onSelect() {} },
];

/** The default rendered anatomy: an open menu with its items. */
export function Default() {
  return (
    <ContextMenu
      open
      title="Actions"
      closeLabel="Close actions"
      items={basicItems}
    />
  );
}

/** Placed at a pointer position rather than against an anchor. */
export function AtPointerPosition() {
  return (
    <ContextMenu
      open
      title="Actions"
      closeLabel="Close actions"
      position={{ x: 120, y: 80 }}
      items={basicItems}
    />
  );
}

/** Separators, icons, and shortcut hints in one grouped menu. */
export function GroupedWithShortcuts() {
  return (
    <ContextMenu
      open
      title="Actions"
      closeLabel="Close actions"
      items={[
        {
          id: "copy",
          label: "Copy",
          shortcut: "⌘C",
          icon: (
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth={2}
            >
              <rect x="9" y="9" width="11" height="11" rx="2" />
            </svg>
          ),
          onSelect() {},
        },
        { id: "paste", label: "Paste", shortcut: "⌘V", onSelect() {} },
        {
          id: "select-all",
          label: "Select all",
          separatorBefore: true,
          onSelect() {},
        },
      ]}
    />
  );
}

/** A checkable item reports its state through menuitemcheckbox rather than a label change. */
export function WithCheckableItem() {
  return (
    <ContextMenu
      open
      title="Actions"
      closeLabel="Close actions"
      items={[
        {
          id: "mouse-mode",
          label: "Mouse reporting",
          pressed: true,
          onSelect() {},
        },
        { id: "rename", label: "Rename", onSelect() {} },
      ]}
    />
  );
}

/** A destructive item is marked so it reads as consequential before it is chosen. */
export function WithDestructiveItem() {
  const anchor = useRef<HTMLButtonElement>(null);
  return (
    <>
      <button ref={anchor} type="button">
        Anchor
      </button>
      <ContextMenu
        open
        title="Actions"
        closeLabel="Close actions"
        anchorRef={anchor}
        items={[
          { id: "rename", label: "Rename", onSelect() {} },
          {
            id: "delete",
            label: "Delete",
            destructive: true,
            separatorBefore: true,
            onSelect() {},
          },
        ]}
      />
    </>
  );
}
