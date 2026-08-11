import { Icon, type IconName } from "./Icon";

const names: IconName[] = [
  "check",
  "close",
  "chevronDown",
  "chevronRight",
  "menu",
  "search",
  "plus",
  "arrowStart",
  "arrowEnd",
];

const labels: Record<IconName, string> = {
  check: "Check",
  close: "Close",
  chevronDown: "Chevron down",
  chevronRight: "Chevron right",
  menu: "Menu",
  search: "Search",
  plus: "Plus",
  arrowStart: "Arrow start",
  arrowEnd: "Arrow end",
  send: "Send",
};

const shellStyle = {
  display: "grid",
  gap: "var(--space-lg)",
  inlineSize: "100%",
  maxInlineSize: "100%",
  boxSizing: "border-box" as const,
  padding: "var(--space-xl)",
  border: "var(--border-hairline) solid var(--color-border)",
  borderRadius: "var(--radius-panel)",
  background:
    "linear-gradient(145deg, var(--color-surface-raised), color-mix(in srgb, var(--color-primary) 5%, var(--color-surface-raised)))",
  boxShadow: "var(--elev-raised)",
};

export function IconShowcase({ args }: { args?: { name?: IconName } }) {
  const selected = args?.name ?? "check";
  return (
    <section
      aria-label="Icon specimen"
      data-rcl-icon-showcase
      style={shellStyle}
    >
      <header style={{ display: "grid", gap: "var(--space-2xs)" }}>
        <span
          style={{
            color: "var(--color-primary)",
            font: "var(--text-overline)",
            letterSpacing: "0.1em",
            textTransform: "uppercase",
          }}
        >
          Semantic iconography
        </span>
        <h1
          style={{
            margin: 0,
            color: "var(--color-foreground)",
            font: "var(--text-title)",
          }}
        >
          Icons with a point of view
        </h1>
        <p
          style={{
            margin: 0,
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
            textWrap: "balance",
          }}
        >
          A compact registry keeps geometry, optical scale, direction, and
          accessible naming consistent.
        </p>
      </header>
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(auto-fit, minmax(4.5rem, 1fr))",
          gap: "var(--space-sm)",
          paddingBlock: "var(--space-md)",
          borderBlock: "var(--border-hairline) solid var(--color-border)",
        }}
      >
        {names.map((name) => (
          <div
            key={name}
            style={{
              display: "grid",
              justifyItems: "center",
              gap: "var(--space-xs)",
              padding: "var(--space-sm)",
              borderRadius: "var(--radius-md)",
              background:
                name === selected
                  ? "color-mix(in srgb, var(--color-primary) 12%, transparent)"
                  : "transparent",
              color: "var(--color-muted-foreground)",
              font: "var(--text-caption)",
            }}
          >
            <Icon
              name={name}
              label={name === selected ? labels[name] : undefined}
              tone={name === selected ? "accent" : "default"}
              size="lg"
            />
            <span>{labels[name]}</span>
          </div>
        ))}
      </div>
      <footer
        style={{
          display: "flex",
          alignItems: "center",
          gap: "var(--space-sm)",
          flexWrap: "wrap",
          color: "var(--color-muted-foreground)",
          font: "var(--text-caption)",
        }}
      >
        <Icon name={selected} tone="accent" size="md" />
        <span>{labels[selected]} · named for assistive technology</span>
      </footer>
    </section>
  );
}
