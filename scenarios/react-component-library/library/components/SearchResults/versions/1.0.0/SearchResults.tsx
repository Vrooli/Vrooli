/** @vrooliComponentSource react-component-library:SearchResults */
const panel = {
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, .75rem)",
  background: "var(--color-surface, #fff)",
  color: "var(--color-foreground, #0f172a)",
  padding: "var(--space-md, 24px)",
  boxShadow: "var(--elev-raised, 0 1px 3px rgb(15 23 42 / .08))",
};
const muted = { color: "var(--color-muted-foreground, #64748b)" };
export function SearchResults({
  query = "",
  items = [],
}: {
  query?: string;
  items?: string[];
}) {
  const results = items.filter((item) =>
    item.toLowerCase().includes(query.toLowerCase()),
  );
  return (
    <section aria-label="Search results" style={{ display: "grid", gap: 12 }}>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "baseline",
        }}
      >
        <h2 style={{ margin: 0, fontSize: 18 }}>Results</h2>
        <span style={muted}>{results.length} results</span>
      </div>
      <ul
        style={{
          display: "grid",
          gap: 8,
          listStyle: "none",
          margin: 0,
          padding: 0,
        }}
      >
        {results.length ? (
          results.map((item, index) => (
            <li key={item + String(index)} style={{ ...panel, padding: 16 }}>
              {item}
            </li>
          ))
        ) : (
          <li style={muted}>No matches found</li>
        )}
      </ul>
    </section>
  );
}
