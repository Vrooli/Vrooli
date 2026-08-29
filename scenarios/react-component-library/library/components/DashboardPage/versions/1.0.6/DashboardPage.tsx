/**
 * @libraryId react-component-library:DashboardPage
 * @displayName DashboardPage
 * @description A metric-first page template with data-source-owned regions and honest asynchronous states.
 * @version 1.0.6
 * @tags ["template","page","data-source","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:DashboardPage */
import { useStrings } from "@vrooli/react-component-library/useLocale/1.1.0";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.2";

import type { ReactNode } from "react";
const panel = {
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, 0.5rem)",
  background: "var(--color-surface, #ffffff)",
  color: "var(--color-foreground, #0f172a)",
  padding: "var(--space-md, 24px)",
  boxShadow: "var(--elev-raised, 0 1px 2px rgba(9, 18, 22, .06), 0 1px 3px rgba(9, 18, 22, .10))",
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
  const strings = useStrings();
  if (state === "loading")
    return (
      <div
        data-testid="templates.dashboard-page"
        role="status"
        style={{ ...panel, textAlign: "center" }}
      >
        {strings("templates.dashboard-page.loading", "Loading…")}
      </div>
    );
  if (state === "refreshing")
    return (
      <div role="status" style={panel}>
        <strong>{strings("templates.dashboard-page.refreshing", "Refreshing")}</strong>
        {children}
      </div>
    );
  if (state === "stale") return <div>Showing stale data{children}</div>;
  if (state === "empty")
    return (
      <div data-state="empty" style={{ ...panel, textAlign: "center" }}>
        {strings("templates.dashboard-page.nothing-here", "Nothing here")}
      </div>
    );
  if (state === "partial-error")
    return <div role="status">Some sections need attention{children}</div>;
  if (state === "fatal-error")
    return (
      <div role="alert" style={{ ...panel, borderColor: "var(--color-danger, #dc2626)" }}>
        {strings("templates.dashboard-page.unable-to-load-this-page", "Unable to load this page")}
      </div>
    );
  if (state === "offline") return <div role="status">Offline{children}</div>;
  return <>{children}</>;
}
export const DashboardPage = withClassName(function DashboardPage({
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
  const strings = useStrings();
  return (
    <StateView state={state}>
      <div style={{ display: "grid", gap: 16 }}>
        <header>
          <h1 style={{ margin: 0, fontSize: 24 }}>
            {strings("templates.dashboard-page.dashboard", "Dashboard")}
          </h1>
          <p style={muted}>
            {strings(
              "templates.dashboard-page.a-clear-view-of-what-needs-attention",
              "A clear view of what needs attention.",
            )}
          </p>
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
});
