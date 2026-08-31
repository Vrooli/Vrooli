import { FormSection } from "./FormSection";

function Input({ label, value }: { label: string; value: string }) {
  const id = label.toLowerCase().replace(/[^a-z0-9]+/g, "-");
  return (
    <label
      htmlFor={id}
      style={{
        display: "grid",
        gap: "var(--space-2xs)",
        color: "var(--color-foreground)",
        font: "var(--text-label)",
      }}
    >
      {label}
      <input
        id={id}
        aria-label={label}
        defaultValue={value}
        style={{
          boxSizing: "border-box",
          minHeight: 44,
          width: "100%",
          border: "1px solid var(--color-border)",
          borderRadius: "var(--radius-control)",
          background: "var(--color-surface)",
          color: "inherit",
          padding: "0 var(--space-sm)",
          font: "inherit",
        }}
      />
    </label>
  );
}

export function Default() {
  return (
    <FormSection
      title="Profile details"
      description="The information teammates see when they work with you."
      summary="All fields are saved to your profile."
      errorCount={0}
    >
      <Input label="Display name" value="Maya Chen" />
      <Input label="Role" value="Product designer" />
    </FormSection>
  );
}

export function Validation() {
  return (
    <FormSection
      title="Workspace access"
      description="Choose who can discover and collaborate in this workspace."
      summary="Two required fields need attention."
      errorCount={2}
    >
      <Input label="Workspace name" value="" />
      <Input label="Visibility" value="Choose visibility" />
    </FormSection>
  );
}

export function Collapsed() {
  return (
    <FormSection
      title="Advanced preferences"
      description="Defaults for notifications and automation."
      summary="Collapsed to keep the primary task focused."
      collapsible
      defaultOpen={false}
    >
      <Input label="Digest cadence" value="Weekly" />
    </FormSection>
  );
}
