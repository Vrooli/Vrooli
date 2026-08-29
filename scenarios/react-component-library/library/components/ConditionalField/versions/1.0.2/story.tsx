import { useMemo, type ReactNode } from "react";
import { createFormStore } from "@vrooli/react-component-library/FormStore/1.0.0";
import { ConditionalField } from "./ConditionalField";

function TextInput({ label, value }: { label: string; value?: string }) {
  return (
    <label
      style={{
        display: "grid",
        gap: "var(--space-2xs)",
        color: "var(--color-foreground)",
        font: "var(--text-label)",
      }}
    >
      {label}
      <input
        aria-label={label}
        defaultValue={value}
        style={{
          minHeight: 44,
          boxSizing: "border-box",
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

function Showcase({
  children,
  title,
  detail,
}: {
  children: ReactNode;
  title: string;
  detail: string;
}) {
  return (
    <section
      style={{
        display: "grid",
        gap: "var(--space-md)",
        boxSizing: "border-box",
        width: "min(100%, 620px)",
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
          Dependency graph
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

export function Default() {
  const store = useMemo(
    () =>
      createFormStore({
        initialValues: { plan: "team", billingEmail: "billing@acme.test" },
      }),
    [],
  );
  return (
    <Showcase
      title="Only ask for what matters"
      detail="The field follows the current plan without consumer-side conditional markup."
    >
      <select
        aria-label="Plan"
        defaultValue="team"
        onChange={(event) => store.setValue("plan", event.target.value)}
        style={{
          minHeight: 44,
          border: "1px solid var(--color-border)",
          borderRadius: "var(--radius-control)",
          background: "var(--color-surface)",
          color: "inherit",
          padding: "0 var(--space-sm)",
          font: "inherit",
        }}
      >
        <option value="team">Team</option>
        <option value="personal">Personal</option>
      </select>
      <ConditionalField
        store={store}
        field="billingEmail"
        when={(values) => values.plan === "team"}
        fallback={
          <div
            role="status"
            style={{
              color: "var(--color-muted-foreground)",
              font: "var(--text-body)",
            }}
          >
            Billing details are not needed for a personal plan.
          </div>
        }
      >
        <TextInput label="Billing email" value="billing@acme.test" />
      </ConditionalField>
    </Showcase>
  );
}

export function ResetOnHide() {
  const store = useMemo(
    () => createFormStore({ initialValues: { plan: "team", taxID: "TX-2048" } }),
    [],
  );
  return (
    <Showcase
      title="Hidden values do not linger"
      detail="Reset mode returns a dependent value to its declared default when its parent choice changes."
    >
      <select
        aria-label="Plan"
        defaultValue="team"
        onChange={(event) => store.setValue("plan", event.target.value)}
        style={{
          minHeight: 44,
          border: "1px solid var(--color-border)",
          borderRadius: "var(--radius-control)",
          background: "var(--color-surface)",
          color: "inherit",
          padding: "0 var(--space-sm)",
          font: "inherit",
        }}
      >
        <option value="team">Team</option>
        <option value="personal">Personal</option>
      </select>
      <button
        type="button"
        aria-label="Hide tax details"
        onClick={() => store.setValue("plan", "personal")}
        style={{
          minHeight: 44,
          border: "1px solid var(--color-primary)",
          borderRadius: "var(--radius-control)",
          background: "var(--color-primary)",
          color: "var(--color-primary-foreground, #ffffff)",
          padding: "0 var(--space-sm)",
          font: "var(--text-label)",
          cursor: "pointer",
        }}
      >
        Hide tax details
      </button>
      <ConditionalField
        store={store}
        field="taxID"
        mode="reset"
        when={(values) => values.plan === "team"}
        fallback={
          <div
            role="status"
            style={{
              color: "var(--color-muted-foreground)",
              font: "var(--text-body)",
            }}
          >
            Tax details cleared when the plan changes.
          </div>
        }
      >
        <TextInput label="Tax ID" value="TX-2048" />
      </ConditionalField>
    </Showcase>
  );
}
