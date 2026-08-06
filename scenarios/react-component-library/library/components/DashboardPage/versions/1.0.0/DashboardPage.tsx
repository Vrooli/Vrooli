/** @vrooliComponentSource react-component-library:DashboardPage */
import type { ReactNode } from "react";
const panel = {
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, .75rem)",
  background: "var(--color-surface, #fff)",
  color: "var(--color-foreground, #0f172a)",
  padding: "var(--space-md, 24px)",
  boxShadow: "var(--elev-raised, 0 1px 3px rgb(15 23 42 / .08))",
};
const muted = { color: "var(--color-muted-foreground, #64748b)" };
type State =
  | "loading"
  | "refreshing"
  | "stale"
  | "empty"
  | "partial-error"
  | "fatal-error"
  | "offline"
  | "ready";
function StateView({ state, children }: { state: State; children: ReactNode }) {
  if (state === "loading")
    return (
      <div role="status" style={{ ...panel, textAlign: "center" }}>
        Loading…
      </div>
    );
  if (state === "refreshing")
    return (
      <div role="status" style={panel}>
        <strong>Refreshing</strong>
        {children}
      </div>
    );
  if (state === "stale") return <div>Showing stale data{children}</div>;
  if (state === "empty")
    return (
      <div data-state="empty" style={{ ...panel, textAlign: "center" }}>
        Nothing here
      </div>
    );
  if (state === "partial-error")
    return <div role="status">Some sections need attention{children}</div>;
  if (state === "fatal-error")
    return (
      <div
        role="alert"
        style={{ ...panel, borderColor: "var(--color-danger, #dc2626)" }}
      >
        Unable to load this page
      </div>
    );
  if (state === "offline") return <div role="status">Offline{children}</div>;
  return <>{children}</>;
}
export function DashboardPage({
  state = "ready",
  data,
}: {
  state?: State;
  data?: {
    metrics?: Array<{ label: string; value: string }>;
    detail?: ReactNode;
    activity?: ReactNode;
  };
}) {
  return (
    <StateView state={state}>
      <div style={{ display: "grid", gap: 16 }}>
        <header>
          <h1 style={{ margin: 0, fontSize: 24 }}>Dashboard</h1>
          <p style={muted}>A clear view of what needs attention.</p>
        </header>
        <section
          data-region="primary-metrics"
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(auto-fit, minmax(180px, 1fr))",
            gap: 12,
          }}
        >
          {(data?.metrics ?? []).map((metric) => (
            <article key={metric.label} style={panel}>
              <span
                style={{
                  ...muted,
                  fontSize: 12,
                  fontWeight: 700,
                  textTransform: "uppercase",
                }}
              >
                {metric.label}
              </span>
              <strong style={{ display: "block", marginTop: 8, fontSize: 28 }}>
                {metric.value}
              </strong>
            </article>
          ))}
        </section>
        <section data-region="detail" style={panel}>
          {data?.detail ?? "Supporting detail"}
        </section>
        <section data-region="activity" style={panel}>
          {data?.activity ?? "Recent activity"}
        </section>
      </div>
    </StateView>
  );
}
