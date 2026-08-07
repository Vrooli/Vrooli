import { useState, type CSSProperties } from "react";
import { AlertDialog } from "./AlertDialog";

const frame: CSSProperties = {
  display: "grid",
  gap: "var(--space-sm, 12px)",
  width: "min(100%, 38rem)",
  minWidth: 0,
  boxSizing: "border-box",
  padding: "var(--space-lg, 24px)",
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, 16px)",
  background: "var(--color-surface, #fff)",
  color: "var(--color-foreground, #0f172a)",
  boxShadow: "var(--elev-raised, 0 12px 36px rgb(15 23 42 / .1))",
};
const detail = {
  margin: 0,
  color: "var(--color-muted-foreground, #64748b)",
  fontSize: 13,
  lineHeight: 1.5,
};

export function Default() {
  const [open, setOpen] = useState(true);
  return (
    <div style={frame}>
      <p style={detail}>
        The dialog refuses accidental outside and Escape dismissal. Choose an
        explicit recovery action.
      </p>
      <AlertDialog
        open={open}
        title="Remove this workspace?"
        description="This removes the workspace and its saved views from your account."
        destructive
        confirmLabel="Remove workspace"
        onConfirm={() => setOpen(false)}
        onCancel={() => setOpen(false)}
      >
        <p style={detail}>
          This action cannot be undone. Export anything you may need before
          continuing.
        </p>
      </AlertDialog>
    </div>
  );
}

export function Submitting() {
  return (
    <div style={frame}>
      <AlertDialog
        open
        title="Publish changes?"
        description="Your changes will become visible to everyone with access."
        confirmLabel="Publish"
        busy
        onConfirm={() => undefined}
        onCancel={() => undefined}
      >
        <p style={detail}>
          Publishing takes a moment while we verify the final version.
        </p>
      </AlertDialog>
    </div>
  );
}

export function RequestError() {
  return (
    <div style={frame}>
      <AlertDialog
        open
        title="Publish changes?"
        description="Your changes are still safe, but publishing did not finish."
        status="error"
        errorMessage="The publishing service timed out. Check your connection and try again."
        confirmLabel="Try again"
        onConfirm={() => undefined}
        onCancel={() => undefined}
      >
        <p style={detail}>No partial publish was created.</p>
      </AlertDialog>
    </div>
  );
}
