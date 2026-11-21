import type { ReactNode } from "react";
import { AspectRatio } from "./AspectRatio";

function Showcase({
  children,
  title,
  detail,
}: {
  children: ReactNode;
  title: string;
  detail: string;
}) {
  return (
    <section
      style={{
        boxSizing: "border-box",
        display: "grid",
        gap: "var(--space-md)",
        width: "min(100%, 560px)",
        padding: "var(--space-xl)",
        border: "1px solid var(--color-border)",
        borderRadius: "var(--radius-panel)",
        background: "var(--color-surface-raised)",
        boxShadow: "var(--elev-raised)",
      }}
    >
      <div style={{ display: "grid", gap: "var(--space-2xs)" }}>
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
      title="Space reserved before content arrives"
      detail="A media frame establishes its shape up front, so a slow image cannot push the surrounding product surface around."
    >
      <AspectRatio ratio="4 / 3">
        <div
          style={{
            display: "grid",
            placeItems: "center",
            height: "100%",
            color: "var(--color-muted-foreground)",
            font: "var(--text-label)",
          }}
        >
          4:3 media frame
        </div>
      </AspectRatio>
    </Showcase>
  );
}

export function Wide() {
  return (
    <Showcase
      title="A cinematic frame"
      detail="The same primitive accepts a deliberate ratio without exposing layout math to the surrounding page."
    >
      <AspectRatio ratio="21 / 9">
        <div
          style={{
            display: "grid",
            placeItems: "center",
            height: "100%",
            color: "var(--color-primary)",
            font: "var(--text-label)",
          }}
        >
          21:9 media frame
        </div>
      </AspectRatio>
    </Showcase>
  );
}
