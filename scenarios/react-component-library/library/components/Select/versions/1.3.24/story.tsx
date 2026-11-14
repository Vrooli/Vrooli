import { Select } from "./Select";

const options = [
  { value: "ready", label: "Ready" },
  { value: "review", label: "Needs review" },
  { value: "blocked", label: "Blocked", disabled: true },
  { value: "published", label: "Published" },
];

/** The field at its normal working size, with a selected option and a disabled choice. */
export function Default() {
  return <Select aria-label="Release status" options={options} defaultValue="ready" />;
}

/** A long selected value remains legible and clips within the field instead of widening the layout. */
export function LongSelectedValue() {
  return (
    <div style={{ inlineSize: 288 }}>
      <Select
        aria-label="Workspace access"
        options={[
          { value: "limited", label: "Limited access" },
          { value: "admin", label: "Awaiting workspace administrator approval" },
        ]}
        defaultValue="admin"
      />
    </div>
  );
}

/** An unselected field keeps the placeholder distinct from a chosen value. */
export function Placeholder() {
  return <Select aria-label="Notification level" placeholder="Choose a level" options={options} />;
}
