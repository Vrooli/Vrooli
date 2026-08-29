import { useState, type ReactNode } from "react";
import { Form } from "./Form";
import { FormActions } from "@vrooli/react-component-library/FormActions/1.0.0";
import { FormField } from "@vrooli/react-component-library/FormField/1.0.1";
import { createFormStore } from "@vrooli/react-component-library/FormStore/1.0.0";

function Showcase({ children, eyebrow }: { children: ReactNode; eyebrow: string }) {
  return (
    <section
      style={{
        boxSizing: "border-box",
        display: "grid",
        gap: "var(--space-md)",
        width: "min(100%, 560px)",
        padding: "var(--space-xl)",
        border: "1px solid var(--color-border)",
        borderRadius: "var(--radius-panel)",
        background: "var(--color-surface-raised)",
        boxShadow: "var(--elev-raised)",
      }}
    >
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
      {children}
    </section>
  );
}

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

function createSettingsStore() {
  return createFormStore({
    initialValues: { workspace: "Aurora", timezone: "Eastern Time" },
    validateField: (field, value) =>
      field === "workspace" && !value.trim() ? "Enter a workspace name." : undefined,
  });
}

export function Default() {
  const [store] = useState(createSettingsStore);
  return (
    <Showcase eyebrow="Workspace settings">
      <Form
        store={store}
        aria-label="Workspace settings"
        title="Make this space yours"
        description="A calm, deliberate form shell keeps related settings together and tells you exactly what happens next."
        onSubmit={() => undefined}
        footer={<FormActions store={store} resetLabel="Reset" />}
      >
        <FormField
          label="Workspace name"
          description="This name appears in navigation and notifications."
          control={<input style={inputStyle} defaultValue="Aurora" />}
        />
        <FormField
          label="Time zone"
          description="Used for scheduled tasks and activity timestamps."
          control={<input style={inputStyle} defaultValue="Eastern Time" />}
        />
      </Form>
    </Showcase>
  );
}

export function Error() {
  const [store] = useState(() => {
    const nextStore = createFormStore({
      initialValues: { workspace: "" },
      validateField: () => "Enter a workspace name before saving.",
    });
    nextStore.setError("workspace", "Enter a workspace name before saving.");
    return nextStore;
  });
  return (
    <Showcase eyebrow="Recovery state">
      <Form
        store={store}
        aria-label="Workspace settings"
        title="One clear next step"
        description="Validation is specific, associated with the field, and leaves the rest of the workflow intact."
        onSubmit={() => undefined}
        footer={<FormActions store={store} />}
      >
        <FormField
          label="Workspace name"
          required
          error={store.getField("workspace").error}
          control={<input style={inputStyle} />}
        />
      </Form>
    </Showcase>
  );
}

export function Offline() {
  const [store] = useState(() => {
    const nextStore = createFormStore({
      initialValues: { workspace: "Aurora" },
    });
    nextStore.setPhase("offline");
    return nextStore;
  });
  return (
    <Showcase eyebrow="Resilient workflow">
      <Form
        store={store}
        aria-label="Offline workspace settings"
        title="Your work is safe"
        description="The form stays usable when the connection drops, with a status that explains what will happen next."
        footer={<FormActions store={store} />}
      >
        <FormField
          label="Workspace name"
          control={<input style={inputStyle} defaultValue="Aurora" />}
        />
      </Form>
    </Showcase>
  );
}
