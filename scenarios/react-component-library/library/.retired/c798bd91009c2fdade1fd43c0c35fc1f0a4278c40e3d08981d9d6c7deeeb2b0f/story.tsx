import { FullPageDrawer } from "./FullPageDrawer";
export function Default() {
  return (
    <FullPageDrawer
      open
      title="Drawer title"
      closeLabel="Close drawer"
      headerActions={<button>Save</button>}
      headerExtra={<small>Workspace</small>}
      footer={<button>Continue</button>}
    >
      <input aria-label="Draft value" defaultValue="Preserved value" />
      <div style={{ minHeight: 640 }}>Scrollable drawer content</div>
    </FullPageDrawer>
  );
}
