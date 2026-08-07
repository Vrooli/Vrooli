import { useState, type CSSProperties } from "react";
import { UnsavedChangesFlow } from "./UnsavedChangesFlow";

const shell: CSSProperties = {
  display: "grid",
  gap: "var(--space-md, 16px)",
  width: "min(100%, 34rem)",
  minWidth: 0,
  boxSizing: "border-box",
  padding: "var(--space-lg, 24px)",
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, 16px)",
  background: "var(--color-surface, #fff)",
  color: "var(--color-foreground, #0f172a)",
  boxShadow: "var(--elev-raised, 0 12px 36px rgb(15 23 42 / .12))",
};

function DraftSurface({ status = "Unsaved edits" }: { status?: string }) {
  return (
    <div style={{ display: "grid", gap: "var(--space-md, 16px)" }}>
      <div>
        <span
          style={{
            display: "block",
            color: "var(--color-muted-foreground, #64748b)",
            fontSize: 12,
            fontWeight: 800,
            letterSpacing: ".12em",
            textTransform: "uppercase",
          }}
        >
          Project brief
        </span>
        <strong
          style={{
            display: "block",
            marginTop: 6,
            fontSize: 22,
            letterSpacing: "-.03em",
          }}
        >
          North star workspace
        </strong>
        <span
          style={{
            display: "block",
            marginTop: 6,
            color: "var(--color-muted-foreground, #64748b)",
            fontSize: 13,
            lineHeight: 1.5,
          }}
        >
          A focused draft surface with enough context to make the recovery
          decision feel safe.
        </span>
      </div>
      <label
        style={{
          display: "grid",
          gap: 6,
          color: "var(--color-muted-foreground, #64748b)",
          fontSize: 12,
          fontWeight: 700,
        }}
      >
        Working title
        <input
          defaultValue="North star workspace"
          aria-label="Working title"
          style={{
            minHeight: 46,
            boxSizing: "border-box",
            border: "1px solid var(--color-border, #cbd5e1)",
            borderRadius: "var(--radius-control, 8px)",
            padding: "0 12px",
            background: "var(--color-surface, #fff)",
            color: "inherit",
            font: "inherit",
          }}
        />
      </label>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          gap: 12,
          paddingTop: 12,
          borderTop: "1px solid var(--color-border, #e2e8f0)",
        }}
      >
        <span
          style={{
            color: "var(--color-muted-foreground, #64748b)",
            fontSize: 13,
          }}
        >
          {status}
        </span>
        <span
          style={{
            color: "var(--color-success-foreground, #137333)",
            fontSize: 13,
            fontWeight: 700,
          }}
        >
          Autosave ready
        </span>
      </div>
    </div>
  );
}

export function Default() {
  const [dirty, setDirty] = useState(true);
  const [saved, setSaved] = useState(false);
  return (
    <UnsavedChangesFlow
      defaultOpen
      isDirty={dirty}
      onSave={() => {
        setDirty(false);
        setSaved(true);
      }}
      onDiscard={() => setDirty(false)}
      onLeave={() => setDirty(false)}
      onPreserveDraft={() => setSaved(true)}
    >
      <div style={shell}>
        <DraftSurface
          status={
            saved
              ? "Draft safely preserved"
              : "Unsaved edits · last change 2 min ago"
          }
        />
        <button
          type="button"
          onClick={() => setDirty(true)}
          style={{
            minHeight: 44,
            border: 0,
            borderRadius: "var(--radius-control, 8px)",
            padding: "0 14px",
            background: "var(--color-primary, #2563eb)",
            color: "var(--color-primary-foreground, #fff)",
            font: "inherit",
            fontWeight: 700,
          }}
        >
          Edit this draft
        </button>
      </div>
    </UnsavedChangesFlow>
  );
}

export function Submitting() {
  return (
    <UnsavedChangesFlow
      defaultOpen
      isDirty
      onSave={() =>
        new Promise<void>((resolve) => window.setTimeout(resolve, 1200))
      }
    >
      <div style={shell}>
        <DraftSurface status="Saving the latest revision…" />
      </div>
    </UnsavedChangesFlow>
  );
}

export function RequestError() {
  return (
    <UnsavedChangesFlow
      defaultOpen
      isDirty
      saveState="error"
      onSave={() => {
        throw new Error("The workspace service is temporarily unavailable.");
      }}
      errorMessage="The workspace service is temporarily unavailable. Retry, keep editing, or preserve a private draft."
      onPreserveDraft={() => undefined}
    >
      <div style={shell}>
        <DraftSurface status="Save needs attention" />
      </div>
    </UnsavedChangesFlow>
  );
}

export function Success() {
  return (
    <UnsavedChangesFlow isDirty={false} onSave={() => undefined}>
      <div style={shell}>
        <DraftSurface status="Saved just now · safe to leave" />
      </div>
    </UnsavedChangesFlow>
  );
}

export function Retry() {
  const [attempts, setAttempts] = useState(0);
  return (
    <UnsavedChangesFlow
      defaultOpen
      isDirty
      onSave={() => {
        const next = attempts + 1;
        setAttempts(next);
        if (next < 2)
          throw new Error(
            "Save timed out. Retry when the connection is stable.",
          );
      }}
      errorMessage="Save timed out. Retry when the connection is stable."
      onPreserveDraft={() => undefined}
    >
      <div style={shell}>
        <DraftSurface
          status={attempts === 0 ? "Waiting for a save" : "Retry available"}
        />
      </div>
    </UnsavedChangesFlow>
  );
}
