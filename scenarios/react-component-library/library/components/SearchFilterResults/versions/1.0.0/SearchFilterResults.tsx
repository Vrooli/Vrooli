/** @vrooliComponentSource react-component-library:SearchFilterResults */
import { useState } from "react";
const panel = {
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, .75rem)",
  background: "var(--color-surface, #fff)",
  color: "var(--color-foreground, #0f172a)",
  padding: "var(--space-md, 24px)",
  boxShadow: "var(--elev-raised, 0 1px 3px rgb(15 23 42 / .08))",
};
const muted = { color: "var(--color-muted-foreground, #64748b)" };
const button = {
  minHeight: 44,
  border: 0,
  borderRadius: "var(--radius-control, .5rem)",
  background: "var(--color-primary, #2563eb)",
  color: "var(--color-primary-foreground, #fff)",
  paddingInline: 16,
  font: "inherit",
  fontWeight: 700,
};
export function SearchFilterResults({
  query = "",
  items = [],
}: {
  query?: string;
  items?: string[];
}) {
  const [value, setValue] = useState(query);
  const results = items.filter((item) =>
    item.toLowerCase().includes(value.toLowerCase()),
  );
  return (
    <div style={{ display: "grid", gap: 16, minWidth: 0 }}>
      <style>{`.rcl-search-filter{box-sizing:border-box;display:flex;flex-wrap:wrap}.rcl-search-filter input{box-sizing:border-box}.rcl-search-filter button{box-sizing:border-box}@media (max-width:520px){.rcl-search-filter{flex-direction:column}.rcl-search-filter input{flex:0 1 auto !important}.rcl-search-filter button{width:100%}}`}</style>
      <form
        className="rcl-search-filter"
        role="search"
        aria-label="Filter"
        onSubmit={(event) => event.preventDefault()}
        style={{ ...panel, gap: 12, padding: 16 }}
      >
        <input
          aria-label="Filter query"
          value={value}
          onChange={(event) => setValue(event.target.value)}
          style={{
            minHeight: 44,
            flex: "1 1 220px",
            minWidth: 0,
            border: "1px solid var(--color-border, #cbd5e1)",
            borderRadius: 8,
            paddingInline: 12,
            font: "inherit",
          }}
        />
        <button type="submit" style={button}>
          Apply filters
        </button>
      </form>
      <section aria-label="Search results">
        <p style={muted}>{results.length} results</p>
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
              <li
                key={item + String(index)}
                style={{ ...panel, padding: 16, overflowWrap: "anywhere" }}
              >
                {item}
              </li>
            ))
          ) : (
            <li style={muted}>No matches found</li>
          )}
        </ul>
      </section>
    </div>
  );
}
