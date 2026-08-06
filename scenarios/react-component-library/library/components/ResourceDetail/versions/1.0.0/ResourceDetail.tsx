/** @vrooliComponentSource materialized.resourcedetail */
const muted = { color: "var(--color-muted-foreground, #64748b)" };
export function ResourceDetail({ title = "Resource", entries = [] }: { title?: string; entries?: Array<{ term: string; description: string }> }) { return <article style={{ display: "grid", gap: 16 }}><header><h1 style={{ margin: 0, fontSize: 24 }}>{title}</h1><p style={muted}>Resource details</p></header><dl>{entries.map((entry) => <div key={entry.term}><dt style={muted}>{entry.term}</dt><dd>{entry.description}</dd></div>)}</dl></article>; }
