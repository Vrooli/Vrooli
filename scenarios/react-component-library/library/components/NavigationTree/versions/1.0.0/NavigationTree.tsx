/** @vrooliComponentSource react-component-library:NavigationTree */
const panel = {
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, .75rem)",
  background: "var(--color-surface, #fff)",
  color: "var(--color-foreground, #0f172a)",
  padding: "var(--space-md, 24px)",
  boxShadow: "var(--elev-raised, 0 1px 3px rgb(15 23 42 / .08))",
};
export function NavigationTree({
  items = ["Overview", "Activity"],
}: {
  items?: string[];
}) {
  return (
    <nav
      aria-label="Primary navigation"
      data-component="NavigationTree"
      style={{ ...panel, display: "grid", gap: 12 }}
    >
      <strong
        style={{
          fontSize: 13,
          letterSpacing: ".08em",
          textTransform: "uppercase",
          color: "var(--color-muted-foreground, #64748b)",
        }}
      >
        Workspace
      </strong>
      <ul
        style={{
          display: "grid",
          gap: 4,
          listStyle: "none",
          margin: 0,
          padding: 0,
        }}
      >
        {items.map((item, index) => (
          <li key={`${item}-${String(index)}`}>
            <a
              href={`#${encodeURIComponent(item.toLowerCase())}`}
              style={{
                display: "block",
                borderRadius: 8,
                color: "var(--color-foreground, #0f172a)",
                padding: "10px 12px",
                textDecoration: "none",
              }}
            >
              {item}
            </a>
          </li>
        ))}
      </ul>
    </nav>
  );
}
