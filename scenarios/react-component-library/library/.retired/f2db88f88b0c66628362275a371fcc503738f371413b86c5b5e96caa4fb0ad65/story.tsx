import { resolveStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import { useState } from "react";
import { ObjectField } from "./ObjectField";
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

function AddressFields({
  context,
}: {
  context: {
    value: { city: string; country: string };
    setValue: (key: "city" | "country", value: string) => void;
    getError: (key: "city" | "country") => string | undefined;
  };
}) {
  return (
    <div
      style={{
        display: "grid",
        gridTemplateColumns: "repeat(auto-fit, minmax(min(100%, 13rem), 1fr))",
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
          aria-label={resolveStrings("forms.object-field.aria-label", "City")}
          value={context.value.city}
          onChange={(event) => context.setValue("city", event.target.value)}
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
          aria-label={resolveStrings(
            "forms.object-field.aria-label.country",
            "Country",
          )}
          value={context.value.country}
          onChange={(event) => context.setValue("country", event.target.value)}
          style={inputStyle}
        />
      </label>
    </div>
  );
}

export function Default() {
  const [store] = useState(() =>
    createFormStore({
      initialValues: {
        address: { city: "Brooklyn", country: "United States" },
      },
    }),
  );
  return (
    <Showcase
      eyebrow="Nested object"
      title={resolveStrings(
        "forms.object-field.title",
        "Group related details without losing focus",
      )}
      detail="The group carries its own hierarchy and disclosure while child controls remain ordinary, addressable inputs."
    >
      <ObjectField
        store={store}
        field="address"
        title={resolveStrings(
          "forms.object-field.title.mailing-address",
          "Mailing address",
        )}
        description={resolveStrings(
          "forms.object-field.description",
          "Used for invoices and account correspondence.",
        )}
        collapsible
      >
        {(context) => <AddressFields context={context} />}
      </ObjectField>
    </Showcase>
  );
}

export function Validation() {
  const [store] = useState(() => {
    const next = createFormStore({
      initialValues: { address: { city: "", country: "United States" } },
    });
    next.setError("address", "Add a city before saving.");
    return next;
  });
  return (
    <Showcase
      eyebrow="Validation context"
      title={resolveStrings(
        "forms.object-field.title.errors-stay-with-the-group",
        "Errors stay with the group",
      )}
      detail="A nested error is announced at the section level without hiding the fields needed to recover."
    >
      <ObjectField
        store={store}
        field="address"
        title={resolveStrings(
          "forms.object-field.title.mailing-address",
          "Mailing address",
        )}
        description={resolveStrings(
          "forms.object-field.description.the-city-is-required-for-this-workspace",
          "The city is required for this workspace.",
        )}
        defaultValue={{ city: "", country: "United States" }}
      >
        {(context) => <AddressFields context={context} />}
      </ObjectField>
    </Showcase>
  );
}
