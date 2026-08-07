import { useRef, useState, type CSSProperties } from "react";
import { DirtyStateGuard, type DirtyStateGuardHandle } from "./DirtyStateGuard";

const shell: CSSProperties = {
  display: "grid",
  gap: "var(--space-md, 16px)",
  width: "min(100%, 30rem)",
  minWidth: 0,
  boxSizing: "border-box",
  padding: "var(--space-lg, 24px)",
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, 16px)",
  background: "var(--color-surface, #fff)",
  color: "var(--color-foreground, #0f172a)",
  boxShadow: "var(--elev-raised, 0 12px 36px rgb(15 23 42 / .12))",
};

function DraftForm({ onSave }: { onSave: () => void }) {
  return (
    <div style={{ display: "grid", gap: "12px" }}>
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
            fontSize: 20,
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
          }}
        >
          A quiet place to refine the decision before it ships.
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
            minHeight: 44,
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
          Draft · saved a moment ago
        </span>
        <button
          type="button"
          onClick={onSave}
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
          Save draft
        </button>
      </div>
    </div>
  );
}

export function Default() {
  const [dirty, setDirty] = useState(false);
  const guard = useRef<DirtyStateGuardHandle>(null);
  return (
    <DirtyStateGuard
      ref={guard}
      isDirty={dirty}
      onSave={() => setDirty(false)}
      onLeave={() => setDirty(false)}
    >
      <div style={shell}>
        <DraftForm onSave={() => setDirty(false)} />
        <button
          type="button"
          onClick={() => {
            setDirty(true);
            window.setTimeout(() => guard.current?.requestLeave(), 0);
          }}
          style={{
            minHeight: 40,
            border: "1px solid var(--color-border, #cbd5e1)",
            borderRadius: "var(--radius-control, 8px)",
            background: "transparent",
            color: "inherit",
            font: "inherit",
            fontWeight: 700,
          }}
        >
          Preview leave protection
        </button>
      </div>
    </DirtyStateGuard>
  );
}

export function PromptOpen() {
  const [saved, setSaved] = useState(false);
  return (
    <DirtyStateGuard
      defaultOpen
      isDirty={!saved}
      onSave={() => setSaved(true)}
      onLeave={() => setSaved(true)}
    >
      <div style={shell}>
        <DraftForm onSave={() => setSaved(true)} />
        <p
          style={{
            margin: 0,
            color: "var(--color-muted-foreground, #64748b)",
            fontSize: 13,
          }}
        >
          {saved
            ? "Your changes are safe."
            : "This draft has meaningful unsaved edits."}
        </p>
      </div>
    </DirtyStateGuard>
  );
}
