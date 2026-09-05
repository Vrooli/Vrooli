import { useState } from "react";
import { FilterBar, type FilterOption } from "./FilterBar";
import { PreviewShowcase } from "../../../../preview-harnesses/showcase/PreviewShowcase";

const options: FilterOption[] = [
  { id: "ready", label: "Ready", count: 24 },
  { id: "review", label: "Needs review", count: 7 },
  { id: "archived", label: "Archived", count: 3 },
];

function FilterBarSubject(props: Record<string, unknown>) {
  return (
    <FilterBar {...(props as unknown as Parameters<typeof FilterBar>[0])} />
  );
}

export function Default() {
  return (
    <PreviewShowcase
      subject={FilterBarSubject}
      args={{ options, defaultActiveFilterIds: ["ready"] }}
      label="Find the signal quickly"
      description="Query, narrow, and reset without losing the context of the collection you are exploring."
    />
  );
}

export function Interactive({ log }: StoryHarnessProps<Record<string, never>>) {
  const [lastApplied, setLastApplied] = useState("No filters applied yet");
  return (
    <PreviewShowcase
      subject={() => (
        <FilterBar
          options={options}
          onApply={({ query, activeFilterIds }) => {
            const summary = `${query || "All records"} · ${activeFilterIds.length} active`;
            setLastApplied(summary);
            log("apply", summary);
          }}
          onReset={() => {
            setLastApplied("Reset to all records");
            log("reset", "all");
          }}
        />
      )}
      label="A filter surface with an honest state"
      description="Every change remains keyboard reachable, while apply and reset stay explicit actions."
    >
      <div role="status">Last applied: {lastApplied}</div>
    </PreviewShowcase>
  );
}
