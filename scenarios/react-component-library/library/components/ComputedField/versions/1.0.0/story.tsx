import { useMemo, type ReactNode } from "react";
import { createFormStore } from "@vrooli/react-component-library/FormStore/1.0.0";
import { ComputedField } from "./ComputedField";

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
          Derived value
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

function CurrencyInput({
  store,
  field,
  label,
  value,
}: {
  store: ReturnType<
    typeof createFormStore<{ subtotal: number; taxRate: number }>
  >;
  field: "subtotal" | "taxRate";
  label: string;
  value: number;
}) {
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
        type="number"
        defaultValue={value}
        onChange={(event) => store.setValue(field, Number(event.target.value))}
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

export function Default() {
  const store = useMemo(
    () => createFormStore({ initialValues: { subtotal: 240, taxRate: 0.2 } }),
    [],
  );
  return (
    <Showcase
      title="Always show the consequence"
      detail="The total follows the source values and stays visibly marked as calculated."
    >
      <CurrencyInput
        store={store}
        field="subtotal"
        label="Subtotal"
        value={240}
      />
      <CurrencyInput
        store={store}
        field="taxRate"
        label="Tax rate"
        value={0.2}
      />
      <ComputedField
        store={store}
        field="subtotal"
        label="Estimated total"
        description="Calculated from subtotal and tax rate."
        compute={(values) => values.subtotal * (1 + values.taxRate)}
        format={(value) =>
          `$${typeof value === "number" ? value.toFixed(2) : value}`
        }
      />
    </Showcase>
  );
}
