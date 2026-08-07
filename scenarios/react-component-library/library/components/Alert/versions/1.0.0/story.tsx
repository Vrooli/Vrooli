import { useState } from "react";
import { Alert } from "./Alert";

const actionStyle = {
  minHeight: 44,
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-control, 8px)",
  padding: "0 var(--space-sm, 12px)",
  background: "var(--color-surface, #fff)",
  color: "var(--color-foreground, #0f172a)",
  font: "inherit",
  fontWeight: 700,
};

export function Default() {
  const [visible, setVisible] = useState(true);
  if (!visible) return <p role="status">Message dismissed.</p>;
  return (
    <Alert
      title="Workspace synced"
      description="Your latest changes are available on every device."
      actions={
        <button type="button" style={actionStyle}>
          View activity
        </button>
      }
      dismissible
      onDismiss={() => setVisible(false)}
    />
  );
}

export function Success() {
  return (
    <Alert
      tone="success"
      title="Export ready"
      description="The report is ready to download and has been saved to your workspace."
    />
  );
}

export function Warning() {
  return (
    <Alert
      tone="warning"
      title="Connection is unstable"
      description="We are keeping your edits locally and will retry when the connection improves."
      actions={
        <button type="button" style={actionStyle}>
          Retry now
        </button>
      }
    />
  );
}

export function Danger() {
  return (
    <Alert
      tone="danger"
      title="Could not save changes"
      description="The server rejected this update. Review the highlighted fields and try again."
      actions={
        <button
          type="button"
          style={{
            ...actionStyle,
            border: 0,
            background: "var(--color-danger, #dc2626)",
            color: "var(--color-danger-foreground-inverse, #fff)",
          }}
        >
          Review fields
        </button>
      }
    />
  );
}
