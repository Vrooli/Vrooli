/**
 * @libraryId react-component-library:DetailPage
 * @displayName DetailPage
 * @description A record detail template with identity, metadata, and data-source-owned content.
 * @version 1.0.4
 * @tags ["template","page","data-source","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:DetailPage */
import { useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

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
  const strings = useStrings();
  if (state === "loading")
    return (
      <div data-testid="templates.detail-page" role="status" style={{ ...panel, textAlign: "center" }}>
        {strings("templates.detail-page.loading", "Loading…")}
      </div>
    );
  if (state === "refreshing")
    return (
      <div role="status" style={panel}>
        <strong>{strings("templates.detail-page.refreshing", "Refreshing")}</strong>
        {children}
      </div>
    );
  if (state === "stale") return <div>Showing stale data{children}</div>;
  if (state === "empty")
    return (
      <div data-state="empty" style={{ ...panel, textAlign: "center" }}>
        {strings("templates.detail-page.nothing-here", "Nothing here")}
      </div>
    );
  if (state === "partial-error")
    return <div role="status">Some sections need attention{children}</div>;
  if (state === "fatal-error")
    return (
      <div role="alert" style={{ ...panel, borderColor: "var(--color-danger, #dc2626)" }}>
        {strings("templates.detail-page.unable-to-load-this-page", "Unable to load this page")}
      </div>
    );
  if (state === "offline") return <div role="status">Offline{children}</div>;
  return <>{children}</>;
}
export const DetailPage = withClassName(function DetailPage({
  state = "ready",
  data,
}: {
  state?: State;
  data?: {
    title?: string;
    entries?: Array<{ term: string; description: string }>;
    primary?: ReactNode;
    history?: ReactNode;
    related?: ReactNode;
  };
}) {
  const strings = useStrings();
  return (
    <StateView state={state}>
      <div style={{ display: "grid", gap: 16 }}>
        <header>
          <h1 style={{ margin: 0, fontSize: 24 }}>{data?.title ?? "Resource"}</h1>
          <p style={muted}>
            {strings("templates.detail-page.a-focused-view-of-this-resource", "A focused view of this resource.")}
          </p>
        </header>
        <section data-region="primary" style={panel}>
          {data?.primary ?? "Primary resource content"}
        </section>
        <section data-region="metadata" style={panel}>
          <dl>
            {(data?.entries ?? []).map((entry) => (
              <div key={entry.term}>
                <dt style={muted}>{entry.term}</dt>
                <dd style={{ margin: "4px 0 12px", fontWeight: 600 }}>{entry.description}</dd>
              </div>
            ))}
          </dl>
        </section>
        <section data-region="history" style={panel}>
          {data?.history ?? "History"}
        </section>
        <section data-region="related" style={panel}>
          {data?.related ?? "Related resources"}
        </section>
      </div>
    </StateView>
  );
});
