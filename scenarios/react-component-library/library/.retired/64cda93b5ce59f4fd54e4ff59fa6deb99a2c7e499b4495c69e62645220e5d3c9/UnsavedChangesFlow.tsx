/** @vrooliComponentSource react-component-library:UnsavedChangesFlow */
import { useState, type CSSProperties, type ReactNode } from "react";
import {
  DirtyStateGuard,
  type DirtyStateGuardPromptProps,
} from "../../../DirtyStateGuard/versions/1.0.0/DirtyStateGuard";
import { AlertDialog } from "../../../AlertDialog/versions/1.0.0/AlertDialog";

export type UnsavedChangesSaveState = "idle" | "saving" | "saved" | "error";

export interface UnsavedChangesFlowProps {
  children: ReactNode;
  isDirty: boolean;
  defaultOpen?: boolean;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  onSave: () => void | Promise<void>;
  onDiscard?: () => void;
  onLeave?: () => void;
  onPreserveDraft?: () => void | Promise<void>;
  saveState?: UnsavedChangesSaveState;
  title?: string;
  description?: string;
  saveLabel?: string;
  discardLabel?: string;
  continueLabel?: string;
  preserveDraftLabel?: string;
  errorMessage?: string;
  className?: string;
  style?: CSSProperties;
}

const preserveStyle: CSSProperties = {
  display: "grid",
  gap: "var(--space-2xs, 8px)",
  marginTop: "var(--space-sm, 12px)",
  padding: "var(--space-sm, 12px)",
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-control, 8px)",
  background: "var(--color-surface-muted, #f5f7fb)",
};

const preserveButton: CSSProperties = {
  minBlockSize: "2.75rem",
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-control, 8px)",
  background: "var(--color-surface, #fff)",
  color: "var(--color-foreground, #0f172a)",
  font: "inherit",
  fontWeight: 700,
  cursor: "pointer",
};

export function UnsavedChangesFlow({
  children,
  isDirty,
  defaultOpen = false,
  open,
  onOpenChange,
  onSave,
  onDiscard,
  onLeave,
  onPreserveDraft,
  saveState = "idle",
  title = "Keep your work before leaving?",
  description = "You have edits that are not saved yet. Choose how you want to continue.",
  saveLabel = "Save changes",
  discardLabel = "Discard changes",
  continueLabel = "Keep editing",
  preserveDraftLabel = "Save a private draft",
  errorMessage = "We could not save your changes. You can retry or keep editing while we recover.",
  className,
  style,
}: UnsavedChangesFlowProps) {
  const [localError, setLocalError] = useState<string | null>(null);
  const [preserving, setPreserving] = useState(false);
  const [preserved, setPreserved] = useState(false);
  const resolvedError =
    localError ?? (saveState === "error" ? errorMessage : null);

  const save = async () => {
    setLocalError(null);
    try {
      await onSave();
    } catch (error) {
      setLocalError(
        error instanceof Error && error.message ? error.message : errorMessage,
      );
      throw error;
    }
  };

  const preserve = async () => {
    if (!onPreserveDraft) return;
    setPreserving(true);
    try {
      await onPreserveDraft();
      setPreserved(true);
    } finally {
      setPreserving(false);
    }
  };

  const renderPrompt = (prompt: DirtyStateGuardPromptProps) => (
    <AlertDialog
      open
      title={prompt.title}
      description={prompt.description}
      status={resolvedError ? "error" : "default"}
      errorMessage={resolvedError ?? undefined}
      onConfirm={prompt.onSave}
      onCancel={prompt.onContinue}
      confirmLabel={
        prompt.saving || saveState === "saving" ? "Saving…" : prompt.saveLabel
      }
      cancelLabel={prompt.continueLabel}
      busy={prompt.saving || saveState === "saving"}
      destructive={false}
    >
      <p
        style={{
          margin: 0,
          color: "var(--color-muted-foreground, #64748b)",
          fontSize: 13,
          lineHeight: 1.5,
        }}
      >
        Saving keeps the current version available. Discarding removes these
        edits from this device.
      </p>
      {onPreserveDraft && (
        <div style={preserveStyle}>
          <strong style={{ fontSize: 13 }}>
            {preserved ? "Private draft saved" : "Need more time?"}
          </strong>
          <span
            style={{
              color: "var(--color-muted-foreground, #64748b)",
              fontSize: 13,
              lineHeight: 1.45,
            }}
          >
            Keep a recoverable copy and continue deciding without losing your
            work.
          </span>
          <button
            type="button"
            style={preserveButton}
            disabled={preserving || prompt.saving}
            onClick={() => void preserve()}
          >
            {preserving
              ? "Saving private draft…"
              : preserved
                ? "Draft saved"
                : preserveDraftLabel}
          </button>
        </div>
      )}
      <button
        type="button"
        data-rcl-unsaved-discard
        disabled={prompt.saving || preserving}
        onClick={prompt.onDiscard}
        style={{
          ...preserveButton,
          marginTop: "var(--space-sm, 12px)",
          borderColor: "var(--color-danger-border, #e7a8a8)",
          color: "var(--color-danger-foreground, #b42318)",
        }}
      >
        {prompt.discardLabel}
      </button>
    </AlertDialog>
  );

  return (
    <div data-rcl-unsaved-changes-flow className={className} style={style}>
      <DirtyStateGuard
        isDirty={isDirty}
        defaultOpen={defaultOpen}
        open={open}
        onOpenChange={onOpenChange}
        onSave={save}
        onDiscard={onDiscard}
        onLeave={onLeave}
        title={title}
        description={description}
        saveLabel={saveLabel}
        discardLabel={discardLabel}
        continueLabel={continueLabel}
        renderPrompt={renderPrompt}
      >
        {children}
      </DirtyStateGuard>
    </div>
  );
}
