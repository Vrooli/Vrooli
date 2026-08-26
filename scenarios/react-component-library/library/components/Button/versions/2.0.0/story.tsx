import type { ReactNode } from "react";
import { Button, type ButtonProps } from "./Button";

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
        gap: "var(--space-lg)",
        width: "min(100%, 560px)",
        minHeight: "220px",
        padding: "var(--space-xl)",
        border: "1px solid var(--color-border)",
        borderRadius: "var(--radius-panel)",
        background:
          "linear-gradient(145deg, var(--color-surface-raised), color-mix(in srgb, var(--color-primary) 5%, var(--color-surface-raised)))",
        boxShadow: "var(--elev-raised)",
      }}
    >
      <div style={{ display: "grid", gap: "var(--space-2xs)" }}>
        <span
          style={{
            color: "var(--color-primary)",
            font: "var(--text-overline)",
            letterSpacing: ".08em",
            textTransform: "uppercase",
          }}
        >
          Control grammar
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
      <div
        style={{
          display: "flex",
          flexWrap: "wrap",
          alignItems: "center",
          gap: "var(--space-sm)",
        }}
      >
        {children}
      </div>
    </section>
  );
}

export function ButtonStory({ args }: StoryHarnessProps) {
  const buttonArgs = args as unknown as ButtonProps;
  const label =
    typeof buttonArgs.children === "string" ? buttonArgs.children : "Action";
  const detail = buttonArgs.disabled
    ? "Disabled states retain the same geometry and clearly communicate that the action is unavailable."
    : "A clear visual hierarchy, a full touch target, and a small amount of responsive motion make the next action feel inevitable.";
  return (
    <Showcase title={label} detail={detail}>
      <Button
        {...buttonArgs}
        aria-label={buttonArgs.size === "icon" ? "Icon action" : buttonArgs["aria-label"]}
      />
    </Showcase>
  );
}
