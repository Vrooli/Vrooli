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
export function Overflow() {
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

/** An empty result still exposes the named field and an empty, keyboard-operable list. */
export function EmptyResults() {
  return (
    <Select aria-label="Available projects" placeholder="No projects available" options={[]} />
  );
}

/** A single result remains a real choice instead of becoming a dead-end field. */
export function OneResult() {
  return (
    <Select aria-label="Workspace" options={[{ value: "design", label: "Design workspace" }]} />
  );
}

/** A larger result set keeps its list in the scrollable select surface. */
export function ManyResults() {
  return (
    <Select
      aria-label="Release channel"
      options={[
        { value: "alpha", label: "Alpha" },
        { value: "beta", label: "Beta" },
        { value: "preview", label: "Preview" },
        { value: "canary", label: "Canary" },
        { value: "stable", label: "Stable" },
        { value: "long-term", label: "Long-term support" },
        { value: "archived", label: "Archived" },
        { value: "maintenance", label: "Maintenance" },
        { value: "retired", label: "Retired" },
        { value: "internal", label: "Internal testing" },
      ]}
    />
  );
}

/** While remote options resolve, the field is unavailable and the pending state is announced. */
export function LoadingOptions() {
  return (
    <div>
      <Select aria-label="Team" options={[]} disabled />
      <p role="status">Loading teams…</p>
    </div>
  );
}

/** A failed option request keeps recovery guidance beside the unavailable field. */
export function RequestError() {
  return (
    <div>
      <Select aria-label="Team" options={[]} disabled aria-describedby="team-load-error" />
      <p id="team-load-error" role="alert">
        Teams could not be loaded. Try again.
      </p>
    </div>
  );
}
