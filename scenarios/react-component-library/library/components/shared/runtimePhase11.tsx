import type { CSSProperties, KeyboardEvent, ReactNode } from "react";

type State = "loading" | "refreshing" | "stale" | "empty" | "partial-error" | "fatal-error" | "offline" | "ready";
type Entry = { term: string; description: string };
type Metric = { label: string; value: string; trend?: string };

const color = {
  background: "var(--color-background, #f8fafc)",
  surface: "var(--color-surface, #ffffff)",
  raised: "var(--color-surface-raised, #ffffff)",
  muted: "var(--color-surface-muted, #f1f5f9)",
  foreground: "var(--color-foreground, #0f172a)",
  quiet: "var(--color-muted-foreground, #64748b)",
  border: "var(--color-border, #cbd5e1)",
  primary: "var(--color-primary, #2563eb)",
  primaryText: "var(--color-primary-foreground, #ffffff)",
  focus: "var(--color-focus, #2563eb)",
};

const panel = (extra: CSSProperties = {}): CSSProperties => ({
  border: `1px solid ${color.border}`,
  borderRadius: "var(--radius-panel, 0.75rem)",
  background: color.surface,
  color: color.foreground,
  boxShadow: "var(--elev-raised, 0 1px 3px rgb(15 23 42 / 0.08))",
  ...extra,
});

const control = (primary = false): CSSProperties => ({
  minHeight: "var(--tap-target-min, 44px)",
  border: `1px solid ${primary ? color.primary : color.border}`,
  borderRadius: "var(--radius-control, 0.5rem)",
  background: primary ? color.primary : color.surface,
  color: primary ? color.primaryText : color.foreground,
  paddingInline: "var(--space-sm, 16px)",
  font: "inherit",
  fontWeight: 650,
  cursor: "pointer",
  transition: "background var(--dur-quick, 180ms) var(--ease-standard, ease), transform var(--dur-quick, 180ms) var(--ease-standard, ease)",
});

const inputStyle: CSSProperties = {
  minHeight: "var(--tap-target-min, 44px)",
  width: "100%",
  boxSizing: "border-box",
  border: `1px solid ${color.border}`,
  borderRadius: "var(--radius-control, 0.5rem)",
  background: color.surface,
  color: color.foreground,
  padding: "0 14px",
  font: "inherit",
  outlineColor: color.focus,
};

const quietText: CSSProperties = { color: color.quiet, fontSize: "var(--text-body-sm-size, 13px)", lineHeight: "var(--text-body-sm-line, 20px)" };
const stack: CSSProperties = { display: "grid", gap: "var(--space-xs, 12px)" };

function SectionLabel({ children }: { children: ReactNode }) {
  return <span style={{ color: color.quiet, fontSize: "var(--text-label-size, 12px)", fontWeight: 700, letterSpacing: "0.06em", textTransform: "uppercase" }}>{children}</span>;
}

export function List({ items = [], empty = "Nothing here" }: { items?: string[]; empty?: ReactNode }) {
  return <ul aria-label="List" style={{ ...stack, listStyle: "none", margin: 0, padding: 0 }}>
    {items.length ? items.map((item) => <li key={item} style={panel({ display: "flex", alignItems: "center", gap: "var(--space-xs, 12px)", minHeight: "52px", padding: "0 var(--space-sm, 16px)" })}>
      <span aria-hidden style={{ width: 8, height: 8, borderRadius: "50%", background: color.primary, boxShadow: `0 0 0 4px color-mix(in srgb, ${color.primary} 15%, transparent)` }} />
      <span style={{ fontWeight: 600 }}>{item}</span>
    </li>) : <li style={panel({ ...quietText, boxShadow: "none", padding: "var(--space-md, 24px)", textAlign: "center" })}>{empty}</li>}
  </ul>;
}

export function Table({ rows = [] }: { rows?: Array<Record<string, string>> }) {
  const columns = Object.keys(rows[0] ?? {});
  return <div style={{ ...panel({ overflow: "auto", padding: 0 }) }}><table role="table" style={{ width: "100%", borderCollapse: "collapse", minWidth: 360 }}>
    <thead><tr>{columns.map((column) => <th key={column} scope="col" style={{ background: color.muted, borderBottom: `1px solid ${color.border}`, color: color.quiet, fontSize: "var(--text-label-size, 12px)", letterSpacing: "0.05em", padding: "13px 16px", textAlign: "left", textTransform: "uppercase" }}>{column}</th>)}</tr></thead>
    <tbody>{rows.map((row, index) => <tr key={index}>{columns.map((column) => <td key={column} style={{ borderBottom: index === rows.length - 1 ? undefined : `1px solid ${color.border}`, padding: "15px 16px", fontWeight: column === columns[0] ? 600 : 400 }}>{row[column]}</td>)}</tr>)}</tbody>
  </table>{rows.length === 0 && <div style={{ ...quietText, padding: "var(--space-lg, 32px)", textAlign: "center" }}>No records to display</div>}</div>;
}

export function DescriptionList({ entries = [] }: { entries?: Entry[] }) {
  return <dl style={{ ...panel({ display: "grid", gap: 0, padding: 0, overflow: "hidden", boxShadow: "none" }) }}>{entries.map((entry, index) => <div key={entry.term} style={{ display: "grid", gridTemplateColumns: "minmax(120px, 0.35fr) 1fr", gap: "var(--space-md, 24px)", padding: "var(--space-sm, 16px)", background: index % 2 ? color.muted : color.surface, borderBottom: index === entries.length - 1 ? undefined : `1px solid ${color.border}` }}><dt style={{ color: color.quiet, fontSize: "var(--text-body-sm-size, 13px)", fontWeight: 700 }}>{entry.term}</dt><dd style={{ margin: 0, fontWeight: 600 }}>{entry.description}</dd></div>)}</dl>;
}

export function TreeView({ nodes = [] }: { nodes?: string[] }) {
  return <ul role="tree" aria-label="Tree" style={{ ...stack, listStyle: "none", margin: 0, padding: 0 }}>{nodes.map((node, index) => <li role="treeitem" aria-level={1} key={node} style={{ display: "flex", alignItems: "center", gap: "var(--space-xs, 12px)", minHeight: 44, borderRadius: "var(--radius-control, 0.5rem)", background: index === 0 ? color.muted : "transparent", padding: "0 var(--space-xs, 12px)", fontWeight: index === 0 ? 700 : 550 }}><span aria-hidden style={{ color: color.primary }}>›</span>{node}</li>)}</ul>;
}

export function Timeline({ events = [] }: { events?: Array<{ label: string; detail?: string }> }) {
  return <ol aria-label="Timeline" style={{ ...stack, listStyle: "none", margin: 0, padding: 0 }}>{events.map((event, index) => <li key={`${event.label}-${index}`} style={{ display: "grid", gridTemplateColumns: "20px 1fr", gap: "var(--space-xs, 12px)" }}><span aria-hidden style={{ position: "relative", display: "grid", placeItems: "center" }}><span style={{ zIndex: 1, width: 10, height: 10, borderRadius: "50%", background: color.primary }} />{index < events.length - 1 && <span style={{ position: "absolute", top: 16, bottom: -16, width: 1, background: color.border }} />}</span><span><strong>{event.label}</strong>{event.detail && <span style={{ ...quietText, display: "block", marginTop: 3 }}>{event.detail}</span>}</span></li>)}</ol>;
}

export function Stat({ label = "Metric", value = "—" }: { label?: string; value?: string }) {
  return <div style={{ ...stack, gap: "var(--space-3xs, 4px)" }}><SectionLabel>{label}</SectionLabel><strong data-stat-value style={{ fontSize: "var(--text-title-size, 24px)", letterSpacing: "-0.02em" }}>{value}</strong></div>;
}

export function StatCard({ label = "Metric", value = "—", trend }: Metric) {
  return <article style={panel({ ...stack, position: "relative", minHeight: 118, boxSizing: "border-box", padding: "var(--space-md, 24px)" })}><div style={{ position: "absolute", insetInlineEnd: 20, top: 20, width: 10, height: 10, borderRadius: "50%", background: color.primary, boxShadow: `0 0 0 6px color-mix(in srgb, ${color.primary} 14%, transparent)` }} /><Stat label={label} value={value} />{trend && <span style={{ color: color.primary, fontSize: "var(--text-body-sm-size, 13px)", fontWeight: 700 }}>{trend}</span>}</article>;
}

export function JsonViewer({ value = {} }: { value?: unknown }) {
  return <pre aria-label="JSON value" style={panel({ margin: 0, overflow: "auto", background: color.muted, fontFamily: "var(--font-mono, ui-monospace)", fontSize: "var(--text-body-sm-size, 13px)", lineHeight: 1.65, padding: "var(--space-md, 24px)", whiteSpace: "pre-wrap" })}>{JSON.stringify(value, null, 2)}</pre>;
}

export function DiffViewer({ before = "", after = "" }: { before?: string; after?: string }) {
  return <div aria-label="Diff" style={{ ...stack, gap: "var(--space-2xs, 8px)" }}><div style={{ borderInlineStart: "3px solid var(--color-danger, #dc2626)", background: "color-mix(in srgb, var(--color-danger, #dc2626) 10%, transparent)", padding: "10px 14px" }}><del>{before}</del></div><div style={{ borderInlineStart: "3px solid var(--color-success, #16a34a)", background: "color-mix(in srgb, var(--color-success, #16a34a) 10%, transparent)", padding: "10px 14px" }}><ins>{after}</ins></div></div>;
}

export function AuditTrail({ entries = [] }: { entries?: Array<{ actor: string; action: string }> }) {
  return <ol aria-label="Audit trail" style={{ ...stack, listStyle: "none", margin: 0, padding: 0 }}>{entries.map((entry, index) => <li key={`${entry.actor}-${index}`} style={panel({ boxShadow: "none", padding: "var(--space-sm, 16px)" })}><strong>{entry.actor}</strong><span style={{ ...quietText, display: "block", marginTop: 4 }}>{entry.action}</span></li>)}</ol>;
}

export function FilterBar({ query = "", onQueryChange }: { query?: string; onQueryChange?: (value: string) => void }) {
  return <form role="search" aria-label="Filter" onSubmit={(event) => event.preventDefault()} style={panel({ display: "flex", alignItems: "end", gap: "var(--space-xs, 12px)", flexWrap: "wrap", padding: "var(--space-sm, 16px)" })}>
    <label style={{ display: "grid", flex: "1 1 240px", gap: 6 }}><span style={{ color: color.quiet, fontSize: "var(--text-label-size, 12px)", fontWeight: 700 }}>Filter results</span><input aria-label="Filter query" value={query} onChange={(event) => onQueryChange?.(event.target.value)} style={inputStyle} placeholder="Search by name or status" /></label>
    <button type="submit" style={control(true)}>Apply filters</button>
  </form>;
}

export function SearchResults({ query = "", items = [] }: { query?: string; items?: string[] }) {
  const results = items.filter((item) => item.toLowerCase().includes(query.toLowerCase()));
  return <section aria-label="Search results" style={stack}><div style={{ display: "flex", alignItems: "baseline", justifyContent: "space-between", gap: "var(--space-xs, 12px)" }}><h2 style={{ margin: 0, fontSize: "var(--text-heading-size, 18px)" }}>Results</h2><span style={quietText}>{results.length} results</span></div><List items={results} empty="No matches found" /></section>;
}

export function Tabs({ items = [], active = "", onChange }: { items?: string[]; active?: string; onChange?: (item: string) => void }) {
  const handleKeyDown = (event: KeyboardEvent<HTMLButtonElement>, index: number) => {
    if (!items.length || !["ArrowRight", "ArrowLeft", "Home", "End"].includes(event.key)) return;
    event.preventDefault();
    const next = event.key === "Home" ? 0 : event.key === "End" ? items.length - 1 : (index + (event.key === "ArrowRight" ? 1 : -1) + items.length) % items.length;
    document.getElementById(`rcl-tab-${next}`)?.focus();
  };
  return <nav aria-label="Tabs" role="tablist" style={{ display: "flex", gap: 4, overflowInline: "auto", borderBottom: `1px solid ${color.border}`, paddingInline: 4 }} onKeyDown={(event) => { if (event.target instanceof HTMLButtonElement) handleKeyDown(event, items.indexOf(event.target.textContent ?? "")); }}>
    {items.map((item, index) => { const selected = item === active; return <button id={`rcl-tab-${index}`} type="button" role="tab" aria-selected={selected} tabIndex={selected ? 0 : -1} key={item} onClick={() => onChange?.(item)} style={{ ...control(false), minHeight: 44, border: "0", borderBottom: `3px solid ${selected ? color.primary : "transparent"}`, borderRadius: 0, background: "transparent", color: selected ? color.primary : color.quiet, paddingInline: "var(--space-xs, 12px)" }}>{item}</button>; })}
  </nav>;
}

export function ScrollableTabs(props: { items?: string[]; active?: string; onChange?: (item: string) => void }) { return <div data-scrollable-tabs style={{ maxWidth: "100%", overflow: "hidden" }}><Tabs {...props} /></div>; }
export function NavLink({ label = "Home", current = false }: { label?: string; current?: boolean }) { return <a href="#" aria-current={current ? "page" : undefined} style={{ display: "flex", alignItems: "center", minHeight: 44, gap: "var(--space-xs, 12px)", borderRadius: "var(--radius-control, 0.5rem)", background: current ? color.primary : "transparent", color: current ? color.primaryText : color.foreground, paddingInline: "var(--space-xs, 12px)", textDecoration: "none", fontWeight: 650 }}><span aria-hidden style={{ width: 7, height: 7, borderRadius: "50%", background: current ? color.primaryText : color.quiet }} />{label}</a>; }
export function NavigationTree({ items = [] }: { items?: string[] }) { return <nav aria-label="Navigation" style={panel({ boxShadow: "none", padding: "var(--space-xs, 12px)" })}><TreeView nodes={items} /></nav>; }
export function PageHeader({ title = "Page", description, actions }: { title?: string; description?: string; actions?: ReactNode }) { return <header style={{ display: "flex", alignItems: "end", justifyContent: "space-between", gap: "var(--space-md, 24px)", flexWrap: "wrap", paddingBottom: "var(--space-sm, 16px)" }}><div style={stack}><h1 style={{ margin: 0, fontSize: "var(--text-title-size, 24px)", lineHeight: 1.2, letterSpacing: "-0.025em" }}>{title}</h1>{description && <p style={{ ...quietText, margin: 0 }}>{description}</p>}</div>{actions && <div style={{ display: "flex", gap: "var(--space-2xs, 8px)", flexWrap: "wrap" }}>{actions}</div>}</header>; }
export function Page({ navigation, children, state = "ready" }: { navigation?: ReactNode; children?: ReactNode; state?: State }) { return <div data-page-state={state} style={{ display: "grid", gridTemplateColumns: navigation ? "minmax(200px, 240px) minmax(0, 1fr)" : "minmax(0, 1fr)", minHeight: "var(--space-2xl, 480px)", background: color.background, color: color.foreground }}>{navigation && <aside style={{ borderInlineEnd: `1px solid ${color.border}`, padding: "var(--space-sm, 16px)" }}>{navigation}</aside>}<main tabIndex={-1} style={{ minWidth: 0, padding: "var(--space-md, 24px)" }}>{children}</main></div>; }
export function AppShell({ navigation, children }: { navigation?: ReactNode; children?: ReactNode }) { return <div data-app-shell style={{ minHeight: "var(--space-2xl, 480px)", background: color.background, color: color.foreground }}>{children ?? navigation}</div>; }
export function AppNavigation({ mode = "desktop", children }: { mode?: "mobile" | "tablet" | "desktop" | "wide"; children?: ReactNode }) {
  const content = children ?? <NavLink label="Home" current />;
  const presentation = mode === "mobile" ? "bottom-navigation" : mode === "tablet" ? "drawer" : "sidebar";
  return <div data-responsive-transformation="sidebar-to-drawer modal-to-bottom-sheet header-to-bottom-navigation" data-viewport-mode={mode} data-presentation={presentation} style={{ minHeight: mode === "mobile" ? 64 : 280, padding: mode === "mobile" ? "var(--space-xs, 12px)" : "var(--space-sm, 16px)", background: mode === "mobile" ? color.surface : color.raised, border: `1px solid ${color.border}`, borderRadius: "var(--radius-panel, 0.75rem)" }}>
    {mode === "mobile" ? <nav aria-label="Application navigation" data-testid="app-navigation-bottom-navigation" style={{ display: "flex", justifyContent: "space-around", gap: 8 }}>{content}</nav> : mode === "tablet" ? <aside aria-label="Application navigation" data-testid="app-navigation-drawer" style={{ maxWidth: 280 }}>{content}</aside> : <aside aria-label="Application navigation" data-testid="app-navigation-sidebar" style={{ width: "min(100%, 280px)" }}>{content}</aside>}
  </div>;
}
export function TopBar({ children }: { children?: ReactNode }) { return <header data-top-bar style={panel({ display: "flex", alignItems: "center", gap: "var(--space-sm, 16px)", minHeight: 64, boxSizing: "border-box", padding: "0 var(--space-md, 24px)" })}>{children ?? <><strong style={{ fontSize: "var(--text-heading-size, 18px)" }}>Application</strong><span style={{ ...quietText, marginInlineStart: "auto" }}>Workspace</span></>}</header>; }
export function ResizableSidebar({ children }: { children?: ReactNode }) { return <aside data-resizable-sidebar style={panel({ minInlineSize: "var(--sidebar-min-width, 260px)", maxInlineSize: "var(--sidebar-max-width, 480px)", padding: "var(--space-sm, 16px)" })}>{children}</aside>; }
export function SplitView({ primary, secondary }: { primary?: ReactNode; secondary?: ReactNode }) { return <div data-split-view style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(min(100%, 280px), 1fr))", gap: "var(--space-md, 24px)" }}><section style={{ minWidth: 0 }}>{primary}</section><section style={{ minWidth: 0 }}>{secondary}</section></div>; }

export function SearchFilterResults({ query = "", items = [] }: { query?: string; items?: string[] }) { return <div style={{ ...stack, gap: "var(--space-md, 24px)" }}><FilterBar query={query} /><SearchResults query={query} items={items} /></div>; }
export function ResourceDetail({ title = "Resource", entries = [] }: { title?: string; entries?: Entry[] }) { return <article style={stack}><PageHeader title={title} description="Resource details" /><DescriptionList entries={entries} /></article>; }

function TemplateState({ state, children }: { state: State; children: ReactNode }) {
  if (state === "loading") return <div role="status" style={panel({ padding: "var(--space-lg, 32px)", textAlign: "center" })}>Loading…</div>;
  if (state === "refreshing") return <div role="status" style={panel({ display: "grid", gap: "var(--space-xs, 12px)", padding: "var(--space-sm, 16px)" })}><strong>Refreshing</strong>{children}</div>;
  if (state === "stale") return <div data-state="stale" style={stack}><div style={{ ...quietText, borderInlineStart: `3px solid ${color.primary}`, paddingInlineStart: 12 }}>Showing stale data</div>{children}</div>;
  if (state === "empty") return <div data-state="empty" style={panel({ ...quietText, padding: "var(--space-lg, 32px)", textAlign: "center" })}>Nothing here</div>;
  if (state === "partial-error") return <div role="status" style={stack}><div style={{ color: "var(--color-warning, #d97706)", fontWeight: 700 }}>Some sections need attention</div>{children}</div>;
  if (state === "fatal-error") return <div role="alert" style={panel({ borderColor: "var(--color-danger, #dc2626)", padding: "var(--space-lg, 32px)", textAlign: "center" })}>Unable to load this page</div>;
  if (state === "offline") return <div role="status" style={stack}><div style={{ color: color.quiet, fontWeight: 700 }}>Offline</div>{children}</div>;
  return <>{children}</>;
}

export function DashboardPage({ state = "ready", data }: { state?: State; data?: { metrics?: Metric[]; detail?: ReactNode; activity?: ReactNode } }) { return <TemplateState state={state}><div style={stack}><section data-region="header" data-source="data-source"><PageHeader title="Dashboard" description="A clear view of what needs attention." /></section><section data-region="primary-metrics" data-source="data-source" style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(180px, 1fr))", gap: "var(--space-sm, 16px)" }}>{(data?.metrics ?? []).map((metric) => <StatCard key={metric.label} {...metric} />)}</section><section data-region="detail" data-source="data-source" style={panel({ padding: "var(--space-md, 24px)" })}>{data?.detail ?? "Supporting detail"}</section><section data-region="activity" data-source="data-source" style={panel({ padding: "var(--space-md, 24px)" })}>{data?.activity ?? "Recent activity"}</section></div></TemplateState>; }
export function CollectionPage({ state = "ready", data }: { state?: State; data?: { items?: string[]; bulkActions?: ReactNode; inspector?: ReactNode } }) { return <TemplateState state={state}><div style={stack}><section data-region="header" data-source="data-source"><PageHeader title="Collection" description="Browse, filter, and act on your resources." /></section><section data-region="filters" data-source="data-source"><FilterBar /></section><section data-region="collection" data-source="data-source"><SearchFilterResults items={data?.items ?? []} /></section><section data-region="bulk-actions" data-source="data-source" style={quietText}>{data?.bulkActions ?? "Bulk actions"}</section><section data-region="inspector" data-source="data-source" style={quietText}>{data?.inspector ?? "Inspector"}</section></div></TemplateState>; }
export function DetailPage({ state = "ready", data }: { state?: State; data?: { title?: string; entries?: Entry[]; primary?: ReactNode; history?: ReactNode; related?: ReactNode } }) { return <TemplateState state={state}><div style={stack}><section data-region="header" data-source="data-source"><PageHeader title={data?.title ?? "Resource"} description="A focused view of this resource." /></section><section data-region="primary" data-source="data-source">{data?.primary ?? <ResourceDetail title={data?.title} entries={data?.entries} />}</section><section data-region="metadata" data-source="data-source"><DescriptionList entries={data?.entries} /></section><section data-region="history" data-source="data-source" style={panel({ padding: "var(--space-md, 24px)" })}>{data?.history ?? "History"}</section><section data-region="related" data-source="data-source" style={panel({ padding: "var(--space-md, 24px)" })}>{data?.related ?? "Related resources"}</section></div></TemplateState>; }
