import { useState, type CSSProperties } from "react";
import { Combobox, type ComboboxOption } from "./Combobox";

const shell: CSSProperties = { width: "min(100%, 34rem)", minWidth: 0 };
const options: ComboboxOption[] = [
  {
    value: "design",
    label: "Design systems",
    description: "Tokens, patterns, and visual quality",
  },
  {
    value: "research",
    label: "Research",
    description: "Evidence and customer learning",
  },
  {
    value: "engineering",
    label: "Engineering",
    description: "Build and ship the product",
  },
  {
    value: "operations",
    label: "Operations",
    description: "Keep every workspace healthy",
  },
];

export function Default() {
  return (
    <div style={shell}>
      <Combobox
        label="Workspace team"
        options={options}
        defaultValue="design"
        initialOpen
        description="Choose the team that owns this brief."
      />
    </div>
  );
}
export function Create() {
  const [created, setCreated] = useState("");
  return (
    <div style={shell}>
      <Combobox
        label="Add a capability"
        options={options}
        allowCreate
        initialOpen
        initialQuery="Observability"
        onCreate={setCreated}
        description={
          created
            ? `Created ${created}.`
            : "Search existing capabilities or add a new one."
        }
      />
    </div>
  );
}
export function Empty() {
  return (
    <div style={shell}>
      <Combobox
        label="Workspace team"
        options={options}
        defaultValue=""
        initialOpen
        emptyText="No team matches that search."
        description="Try a different phrase."
      />
    </div>
  );
}
export function ValidationError() {
  return (
    <div style={shell}>
      <Combobox
        label="Workspace team"
        options={options}
        error="Choose the owning team before continuing."
        required
      />
    </div>
  );
}
export function RequestError() {
  return (
    <div style={shell}>
      <Combobox
        label="Remote workspace"
        loadOptions={() => Promise.reject(new Error("Service unavailable"))}
        initialOpen
        debounceMs={0}
        maxAttempts={1}
        description="Remote search is unavailable; retry when the service recovers."
      />
    </div>
  );
}
