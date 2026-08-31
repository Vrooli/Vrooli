import { BottomSheet } from "./BottomSheet";
export function Default() {
  return (
    <BottomSheet
      open
      title="Sheet title"
      closeLabel="Close sheet"
      headerActions={<button>Save</button>}
      footer={<button>Continue</button>}
    >
      <input aria-label="Draft value" defaultValue="Preserved value" />
      <div style={{ minHeight: 480 }}>Scrollable sheet content</div>
    </BottomSheet>
  );
}
