import { useState, type ReactNode } from "react";
import { EditableResource } from "./EditableResource";

const frame = {
  display: "grid",
  gap: "var(--space-md)",
  width: "min(100%, 760px)",
  minWidth: 0,
  boxSizing: "border-box",
  padding: "var(--space-xl)",
  border: "var(--border-hairline) solid var(--color-border)",
  borderRadius: "var(--radius-panel)",
  background: "var(--color-surface-raised)",
  boxShadow: "var(--elev-raised)",
} as const;
function Showcase({
  title,
  detail,
  children,
}: {
  title: string;
  detail: string;
  children: ReactNode;
}) {
  return (
    <section style={frame}>
      <div style={{ display: "grid", gap: "var(--space-2xs)" }}>
        <span
          style={{
            color: "var(--color-primary)",
            font: "var(--text-overline)",
            letterSpacing: ".1em",
            textTransform: "uppercase",
          }}
        >
          Editable resource
        </span>
        <strong style={{ font: "var(--text-title)" }}>{title}</strong>
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
          }}
        >
          {detail}
        </span>
      </div>
      {children}
    </section>
  );
}
function Editor({ defaultEditing = false }: { defaultEditing?: boolean }) {
  const [record, setRecord] = useState({
    name: "Release brief",
    owner: "Mara Chen",
  });
  return (
    <EditableResource
      record={record}
      title={record.name}
      description="Save changes without losing the read context."
      entries={[
        { term: "Owner", description: record.owner },
        { term: "State", description: "Needs review" },
      ]}
      renderEditor={({ draft, setDraft }) => (
        <div style={{ display: "grid", gap: "var(--space-sm)" }}>
          <label style={{ display: "grid", gap: "var(--space-2xs)" }}>
            Name
            <input
              aria-label="Name"
              value={draft.name}
              onChange={(event) =>
                setDraft({ ...draft, name: event.target.value })
              }
            />
          </label>
          <label style={{ display: "grid", gap: "var(--space-2xs)" }}>
            Owner
            <input
              aria-label="Owner"
              value={draft.owner}
              onChange={(event) =>
                setDraft({ ...draft, owner: event.target.value })
              }
            />
          </label>
        </div>
      )}
      onSave={async (next) => {
        await new Promise((resolve) => setTimeout(resolve, 80));
        setRecord(next);
      }}
      defaultEditing={defaultEditing}
    />
  );
}
export function Default() {
  return (
    <Showcase
      title="Read first, edit in place"
      detail="The route owns one resource workflow; consumers supply only the record and editor fields."
    >
      <Editor />
    </Showcase>
  );
}

export function Interactive() {
  return (
    <Showcase
      title="Edit, recover, and save"
      detail="A real draft stays local while the editor is open, then returns to the refreshed resource view after save."
    >
      <Editor />
    </Showcase>
  );
}

export function Editing() {
  return (
    <Showcase
      title="A focused editing state"
      detail="The form keeps the resource context visible while exposing comfortable, keyboard-ready fields and a clear save path."
    >
      <Editor defaultEditing />
    </Showcase>
  );
}
