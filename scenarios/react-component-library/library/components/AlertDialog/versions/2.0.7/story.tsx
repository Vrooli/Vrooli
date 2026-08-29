import { useStrings } from "@vrooli/react-component-library/useLocale/1.1.0";
import { useState, type CSSProperties } from "react";
import { AlertDialog } from "./AlertDialog";

const frame: CSSProperties = {
  display: "grid",
  gap: "var(--space-sm, 16px)",
  width: "min(100%, 38rem)",
  minWidth: 0,
  boxSizing: "border-box",
  padding: "var(--space-lg, 32px)",
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, 0.5rem)",
  background: "var(--color-surface, #ffffff)",
  color: "var(--color-foreground, #0f172a)",
  boxShadow: "var(--elev-raised, 0 1px 2px rgba(9, 18, 22, .06), 0 1px 3px rgba(9, 18, 22, .10))",
};
const detail = {
  margin: 0,
  color: "var(--color-muted-foreground, #64748b)",
  fontSize: 13,
  lineHeight: 1.5,
};

export function Default() {
  const libraryStrings = useStrings();
  const [open, setOpen] = useState(true);
  return (
    <div style={frame}>
      <p style={detail}>
        The dialog refuses accidental outside and Escape dismissal. Choose an explicit recovery
        action.
      </p>
      <AlertDialog
        open={open}
        title={libraryStrings("overlays.alert-dialog.title", "Remove this workspace?")}
        description={libraryStrings(
          "overlays.alert-dialog.description",
          "This removes the workspace and its saved views from your account.",
        )}
        destructive
        confirmLabel="Remove workspace"
        onConfirm={() => setOpen(false)}
        onCancel={() => setOpen(false)}
      >
        <p style={detail}>
          {libraryStrings(
            "overlays.alert-dialog.this-action-cannot-be-undone-export-anything-you-may-need-before-continuing",
            "This action cannot be undone. Export anything you may need before continuing.",
          )}
        </p>
      </AlertDialog>
    </div>
  );
}

export function Submitting() {
  const libraryStrings = useStrings();
  return (
    <div style={frame}>
      <AlertDialog
        open
        title={libraryStrings("overlays.alert-dialog.title.publish-changes", "Publish changes?")}
        description={libraryStrings(
          "overlays.alert-dialog.description.your-changes-will-become-visible-to-everyone-wit",
          "Your changes will become visible to everyone with access.",
        )}
        confirmLabel="Publish"
        busy
        onConfirm={() => undefined}
        onCancel={() => undefined}
      >
        <p style={detail}>
          {libraryStrings(
            "overlays.alert-dialog.publishing-takes-a-moment-while-we-verify-the-final-version",
            "Publishing takes a moment while we verify the final version.",
          )}
        </p>
      </AlertDialog>
    </div>
  );
}

export function RequestError() {
  const libraryStrings = useStrings();
  return (
    <div style={frame}>
      <AlertDialog
        open
        title={libraryStrings("overlays.alert-dialog.title.publish-changes", "Publish changes?")}
        description={libraryStrings(
          "overlays.alert-dialog.description.your-changes-are-still-safe-but-publishing-did-n",
          "Your changes are still safe, but publishing did not finish.",
        )}
        status="error"
        errorMessage="The publishing service timed out. Check your connection and try again."
        confirmLabel="Try again"
        onConfirm={() => undefined}
        onCancel={() => undefined}
      >
        <p style={detail}>
          {libraryStrings(
            "overlays.alert-dialog.no-partial-publish-was-created",
            "No partial publish was created.",
          )}
        </p>
      </AlertDialog>
    </div>
  );
}
