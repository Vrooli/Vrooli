import { Dialog } from "@vrooli/react-component-library";

export function Drawer() {
  return (
    <Dialog
      actions={[
        { id: "cancel", label: "Cancel", onSelect: () => {}, primary: false },
        { id: "save", label: "Save", onSelect: () => {}, primary: true },
      ]}
    />
  );
}
