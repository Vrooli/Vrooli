import { ResponsiveDialog } from "./ResponsiveDialog";
export function Default() {
  return (
    <ResponsiveDialog
      open
      title="Responsive dialog"
      closeLabel="Close dialog"
      headerActions={<button>Save</button>}
      footer={<button>Continue</button>}
    >
      <input aria-label="Preserved draft" defaultValue="Persistent value" />
      <div style={{ minHeight: 480 }}>Resize while this surface remains open.</div>
    </ResponsiveDialog>
  );
}
