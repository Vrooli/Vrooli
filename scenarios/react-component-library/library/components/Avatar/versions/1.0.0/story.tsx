import { Avatar, AvatarGroup } from "./Avatar";

const image =
  "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 120 120'%3E%3Cdefs%3E%3ClinearGradient id='g' x1='0' x2='1'%3E%3Cstop stop-color='%2338bdf8'/%3E%3Cstop offset='1' stop-color='%237c3aed'/%3E%3C/linearGradient%3E%3C/defs%3E%3Crect width='120' height='120' fill='url(%23g)'/%3E%3Ccircle cx='60' cy='47' r='23' fill='white' fill-opacity='.8'/%3E%3Cpath d='M18 116c4-29 20-42 42-42s38 13 42 42' fill='white' fill-opacity='.8'/%3E%3C/svg%3E";

const shell = {
  display: "grid",
  gap: "var(--space-md)",
  width: "min(100%, 560px)",
  minWidth: 0,
  minHeight: 280,
  padding: "var(--space-xl)",
  boxSizing: "border-box",
  border: "var(--border-hairline) solid var(--color-border)",
  borderRadius: "var(--radius-panel)",
  background: "var(--color-surface-raised)",
  boxShadow: "var(--elev-raised)",
} as const;

function Showcase({
  title,
  detail,
  children,
}: {
  title: string;
  detail: string;
  children: React.ReactNode;
}) {
  return (
    <section style={shell}>
      <div style={{ display: "grid", gap: "var(--space-2xs)" }}>
        <span
          style={{
            color: "var(--color-primary)",
            font: "var(--text-overline)",
            letterSpacing: ".1em",
            textTransform: "uppercase",
          }}
        >
          Identity primitive
        </span>
        <strong style={{ font: "var(--text-title)" }}>{title}</strong>
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
          }}
        >
          {detail}
        </span>
      </div>
      {children}
    </section>
  );
}

export function Default() {
  return (
    <Showcase
      title="Identity without ambiguity"
      detail="A named avatar keeps the person accessible while presence remains an additional, non-color-only signal."
    >
      <Avatar name="Maya Chen" src={image} presence="online" />
    </Showcase>
  );
}

export function Loading() {
  return (
    <Showcase
      title="Space reserved before arrival"
      detail="Loading imagery does not move the surrounding transcript or hide the identity fallback contract."
    >
      <Avatar
        name="Maya Chen"
        src="/assets/rcl-avatar-loading.svg"
        presence="away"
      />
    </Showcase>
  );
}

export function RequestError() {
  return (
    <Showcase
      title="A useful image failure"
      detail="When the image cannot load, deterministic initials preserve identity instead of leaving a broken rectangle."
    >
      <Avatar name="Maya Chen" presence="busy" />
    </Showcase>
  );
}

export function Group() {
  return (
    <Showcase
      title="Groups preserve the people count"
      detail="The overflow affordance names how many additional people are present and remains keyboard-readable."
    >
      <AvatarGroup maxVisible={3} label="Reviewers">
        <Avatar name="Maya Chen" />
        <Avatar name="Ravi Shah" />
        <Avatar name="Ada Lovelace" />
        <Avatar name="Noah Williams" />
      </AvatarGroup>
    </Showcase>
  );
}
