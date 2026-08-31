import { ContextMenu } from "./ContextMenu";
export function Default() {
  return (
    <ContextMenu
      open
      title="Actions"
      closeLabel="Close actions"
      items={[
        { id: "open", label: "Open", shortcut: "Enter", onSelect() {} },
        { id: "rename", label: "Rename", onSelect() {} },
        { id: "delete", label: "Delete", onSelect() {} },
      ]}
    />
  );
}
