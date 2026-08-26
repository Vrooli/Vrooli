import { useState } from "react";
import { ArrayField } from "./ArrayField";
import { createFormStore } from "@vrooli/react-component-library/FormStore/1.0.0";

const inputStyle = {
  boxSizing: "border-box",
  width: "100%",
  minHeight: 44,
  border: "1px solid var(--color-border)",
  borderRadius: "var(--radius-control)",
  background: "var(--color-surface)",
  color: "var(--color-foreground)",
  paddingInline: "var(--space-sm)",
  font: "inherit",
} as const;

function Showcase({
  children,
  eyebrow,
  title,
  detail,
}: {
  children: React.ReactNode;
  eyebrow: string;
  title: string;
  detail: string;
}) {
  return (
    <section
      style={{
        boxSizing: "border-box",
        display: "grid",
        gap: "var(--space-lg)",
        width: "min(100%, 640px)",
        padding: "var(--space-xl)",
        border: "1px solid var(--color-border)",
        borderRadius: "var(--radius-panel)",
        background: "var(--color-surface-raised)",
        boxShadow: "var(--elev-raised)",
      }}
    >
      <div style={{ display: "grid", gap: "var(--space-2xs)" }}>
        <span
          style={{
            color: "var(--color-primary)",
            font: "var(--text-overline)",
            letterSpacing: ".08em",
            textTransform: "uppercase",
          }}
        >
          {eyebrow}
        </span>
        <strong
          style={{
            font: "var(--text-title)",
            color: "var(--color-foreground)",
          }}
        >
          {title}
        </strong>
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

export function Default() {
  const [store] = useState(() =>
    createFormStore({
      initialValues: { tags: ["Design systems", "Accessibility"] },
    }),
  );
  return (
    <Showcase
      eyebrow="Ordered collection"
      title="Keep the list in your language"
      detail="Items stay editable in place, with reorder and duplicate actions that remain reachable by keyboard and touch."
    >
      <ArrayField
        store={store}
        field="tags"
        label="Workspace principles"
        description="Arrange the principles in the order your team should read them."
        createItem={() => "New principle"}
        getItemKey={(item, index) => `${item}-${index}`}
        renderItem={({ item, actions }) => (
          <input
            aria-label="Principle"
            value={item}
            onChange={(event) => actions.setValue(event.target.value)}
            style={inputStyle}
          />
        )}
      />
    </Showcase>
  );
}

export function Empty() {
  const [store] = useState(() =>
    createFormStore({ initialValues: { tags: [] as string[] } }),
  );
  return (
    <Showcase
      eyebrow="Start small"
      title="An empty state with a clear invitation"
      detail="The first action is visible without making an empty collection feel broken."
    >
      <ArrayField
        store={store}
        field="tags"
        label="Notification channels"
        description="Choose where important workspace updates should appear."
        minItems={1}
        createItem={() => "Email"}
        addLabel="Add channel"
        renderItem={({ item, actions }) => (
          <input
            aria-label="Channel"
            value={item}
            onChange={(event) => actions.setValue(event.target.value)}
            style={inputStyle}
          />
        )}
      />
    </Showcase>
  );
}
