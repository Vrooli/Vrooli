import { useState } from "react";
import { useOptimisticAction } from "./useOptimisticAction";

export function Default() {
  const [shouldFail, setShouldFail] = useState(false);
  const optimistic = useOptimisticAction<string, string>({
    value: "Draft",
    action: async (next, signal) => {
      await new Promise((resolve) => setTimeout(resolve, 80));
      if (signal.aborted) throw new Error("Cancelled");
      if (shouldFail) throw new Error("Write failed");
      return next;
    },
  });
  return (
    <section
      data-testid="hooks.use-optimistic-action"
      style={{
        display: "grid",
        gap: "var(--space-sm)",
        width: "min(100%, 520px)",
        padding: "var(--space-lg)",
        border: "var(--border-hairline) solid var(--color-border)",
        borderRadius: "var(--radius-panel)",
        background: "var(--color-surface-raised)",
      }}
    >
      <strong>Optimistic commit with honest rollback</strong>
      <span data-rcl-optimistic-value>{optimistic.value}</span>
      <span role="status">{optimistic.status}</span>
      <div
        style={{ display: "flex", flexWrap: "wrap", gap: "var(--space-2xs)" }}
      >
        <button type="button" onClick={() => void optimistic.run("Saved")}>
          Save
        </button>
        <button
          type="button"
          onClick={() => {
            setShouldFail(true);
            void optimistic.run("Will roll back");
          }}
        >
          Simulate failure
        </button>
      </div>
    </section>
  );
}
