import { Icon, type IconName, type IconSize, type IconTone } from "./Icon";

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
  "eye",
  "eyeOff",
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
  eye: "Eye",
  eyeOff: "Eye off",
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
    <section aria-label="Icon specimen" data-rcl-icon-showcase style={shellStyle}>
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
          A compact registry keeps geometry, optical scale, direction, and accessible naming
          consistent.
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

export function IconAnatomy() {
  return (
    <section aria-label="Icon anatomy" data-rcl-icon-anatomy style={shellStyle}>
      <h1 style={{ margin: 0, font: "var(--text-title)" }}>
        A named icon with an accessible label
      </h1>
      <Icon name="check" label="Check" size="lg" tone="accent" />
      <p style={{ margin: 0 }}>The anatomy frame proves the root geometry and accessible name.</p>
    </section>
  );
}

export function IconSetMatrix() {
  return (
    <section aria-label="Icon set matrix" data-rcl-icon-set style={shellStyle}>
      <h2 style={{ margin: 0, font: "var(--text-heading)" }}>Nine registered names</h2>
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(3, minmax(0, 1fr))",
          gap: "var(--space-md)",
        }}
      >
        {names.map((name) => (
          <Icon key={name} name={name} label={labels[name]} size="md" />
        ))}
      </div>
    </section>
  );
}

export function IconTreatmentMatrix() {
  const treatments: Array<[IconSize, IconTone]> = [
    ["sm", "default"],
    ["md", "muted"],
    ["lg", "accent"],
    ["lg", "danger"],
  ];
  return (
    <section aria-label="Icon treatment matrix" data-rcl-icon-treatment style={shellStyle}>
      <h2 style={{ margin: 0, font: "var(--text-heading)" }}>Size and tone treatments</h2>
      <div
        style={{ display: "flex", flexWrap: "wrap", gap: "var(--space-lg)", alignItems: "center" }}
      >
        {treatments.map(([size, tone]) => (
          <Icon
            key={`${size}-${tone}`}
            name="plus"
            label={`${size} ${tone}`}
            size={size}
            tone={tone}
          />
        ))}
      </div>
    </section>
  );
}
