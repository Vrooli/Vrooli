import { useState } from "react";
import { GeneratedForm, type GeneratedField } from "./GeneratedForm";
import { createFormStore } from "../../../../services/FormStore/versions/1.0.0/FormStore";

type Settings = {
  workspace: string;
  plan: string;
  showBilling: boolean;
  billingEmail: string;
  tags: string[];
  address: { city: string; country: string };
  subtotal: number;
  taxRate: number;
  total: number;
};

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

function settingsFields(): Array<GeneratedField<Settings>> {
  return [
    {
      name: "workspace",
      type: "text",
      label: "Workspace name",
      description: "Appears in navigation and notifications.",
      section: "identity",
      required: true,
    },
    {
      name: "plan",
      type: "select",
      label: "Plan",
      description: "Choose the operating model for this workspace.",
      section: "identity",
      options: [
        { value: "pro", label: "Pro workspace" },
        { value: "team", label: "Team workspace" },
      ],
    },
    {
      name: "showBilling",
      type: "checkbox",
      label: "I manage billing for this workspace",
      section: "identity",
    },
    {
      name: "billingEmail",
      type: "email",
      label: "Billing email",
      description: "Shown only when billing responsibility is enabled.",
      section: "identity",
      when: (values) => values.showBilling,
      conditionalMode: "hide",
    },
    {
      name: "tags",
      type: "array",
      label: "Workspace principles",
      description: "Keep the few ideas that guide decisions close at hand.",
      section: "preferences",
      createItem: () => "New principle",
      renderItem: ({ item, setValue }) => (
        <input
          aria-label="Principle"
          value={String(item)}
          onChange={(event) => setValue(event.target.value)}
          style={inputStyle}
        />
      ),
    },
    {
      name: "address",
      type: "object",
      label: "Mailing address",
      description: "Used for invoices and account correspondence.",
      section: "preferences",
      objectDefaultValue: { city: "", country: "" },
      objectChildren: ({ value, setValue }) => (
        <div
          style={{
            display: "grid",
            gridTemplateColumns:
              "repeat(auto-fit, minmax(min(100%, 13rem), 1fr))",
            gap: "var(--space-md)",
          }}
        >
          <label
            style={{
              display: "grid",
              gap: "var(--space-2xs)",
              color: "var(--color-foreground)",
              font: "var(--text-label)",
            }}
          >
            City
            <input
              aria-label="City"
              value={typeof value.city === "string" ? value.city : ""}
              onChange={(event) => setValue("city", event.target.value)}
              style={inputStyle}
            />
          </label>
          <label
            style={{
              display: "grid",
              gap: "var(--space-2xs)",
              color: "var(--color-foreground)",
              font: "var(--text-label)",
            }}
          >
            Country
            <input
              aria-label="Country"
              value={typeof value.country === "string" ? value.country : ""}
              onChange={(event) => setValue("country", event.target.value)}
              style={inputStyle}
            />
          </label>
        </div>
      ),
    },
    {
      name: "total",
      type: "computed",
      label: "Estimated monthly total",
      description: "Calculated from the workspace plan and tax policy.",
      section: "summary",
      compute: (values) =>
        values.plan === "team"
          ? values.subtotal * (1 + values.taxRate) * 1.25
          : values.subtotal * (1 + values.taxRate),
      format: (value) => `$${Number(value).toFixed(2)}`,
    },
  ];
}

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
        width: "min(100%, 720px)",
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

function createSettingsStore(showBilling = false) {
  return createFormStore<Settings>({
    initialValues: {
      workspace: "Aurora",
      plan: "pro",
      showBilling,
      billingEmail: "",
      tags: ["Design", "Accessibility"],
      address: { city: "Brooklyn", country: "United States" },
      subtotal: 240,
      taxRate: 0.2,
      total: 0,
    },
    validateField: (field, value) =>
      field === "workspace" && !(typeof value === "string" ? value : "").trim()
        ? "Enter a workspace name."
        : field === "billingEmail" &&
            showBilling &&
            !(typeof value === "string" ? value : "").includes("@")
          ? "Enter a reachable billing email."
          : undefined,
  });
}

export function Default() {
  const [store] = useState(() => createSettingsStore());
  return (
    <Showcase
      eyebrow="Schema-driven workflow"
      title="A form that grows with the model"
      detail="Sections, conditional fields, repeating values, nested objects, and derived totals all share one store and one recovery language."
    >
      <GeneratedForm
        store={store}
        title="Workspace settings"
        description="A calm generated form keeps the important choices close while preserving a clear path to save."
        fields={settingsFields()}
        sections={[
          {
            id: "identity",
            title: "Workspace identity",
            description:
              "The details people see and the responsibility they carry.",
          },
          {
            id: "preferences",
            title: "Preferences",
            description:
              "Small defaults that make the workspace feel considered.",
          },
          {
            id: "summary",
            title: "Review",
            description: "A final calculated consequence before you save.",
          },
        ]}
      />
    </Showcase>
  );
}

export function Validation() {
  const [store] = useState(() => createSettingsStore(true));
  return (
    <Showcase
      eyebrow="Generated recovery"
      title="The schema still explains the next step"
      detail="Validation summary, field error, and conditional billing state stay synchronized without a second form implementation."
    >
      <GeneratedForm
        store={store}
        title="Billing profile"
        description="Complete the required details to keep invoices deliverable."
        fields={settingsFields()}
        sections={[
          { id: "identity", title: "Workspace identity" },
          { id: "summary", title: "Review" },
        ]}
      />
    </Showcase>
  );
}
