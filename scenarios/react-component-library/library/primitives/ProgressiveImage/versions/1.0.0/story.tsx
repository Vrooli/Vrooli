import type { ReactNode } from "react";
import { ProgressiveImage } from "./ProgressiveImage";

const image = (background: string, accent: string, label: string) =>
  `data:image/svg+xml,${encodeURIComponent(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 960 540"><defs><linearGradient id="g" x1="0" x2="1"><stop stop-color="${background}"/><stop offset="1" stop-color="${accent}"/></linearGradient></defs><rect width="960" height="540" fill="url(#g)"/><circle cx="770" cy="130" r="90" fill="white" fill-opacity=".16"/><path d="M0 430 250 250l160 120 190-210 360 370H0Z" fill="white" fill-opacity=".2"/><text x="56" y="468" fill="white" font-family="system-ui" font-size="30" font-weight="700">${label}</text></svg>`)}`;

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
        <span
          style={{
            color: "var(--color-primary)",
            font: "var(--text-overline)",
            letterSpacing: ".08em",
            textTransform: "uppercase",
          }}
        >
          Media primitive
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
      title="Arrival without the pop"
      detail="The frame is reserved immediately, the image reveals itself softly, and the surrounding surface never jumps."
    >
      <ProgressiveImage
        src={image("#1d4ed8", "#0f766e", "Aurora workspace")}
        alt="Abstract blue-green Aurora workspace landscape"
        ratio="16 / 9"
      />
    </Showcase>
  );
}

export function ResponsiveSources() {
  return (
    <Showcase
      title="Responsive by intent"
      detail="Art direction stays in the source contract while the primitive owns loading, naming, and layout behavior."
    >
      <ProgressiveImage
        src={image("#7c3aed", "#be185d", "Responsive source")}
        sources={[
          {
            media: "(min-width: 60rem)",
            srcSet: image("#0f766e", "#1d4ed8", "Wide art direction"),
          },
        ]}
        alt="Abstract responsive art-directed landscape"
        ratio="3 / 2"
      />
    </Showcase>
  );
}

export function ErrorState() {
  return (
    <Showcase
      title="A useful failure"
      detail="A broken source becomes a legible recovery surface instead of a browser icon or an empty rectangle."
    >
      <ProgressiveImage
        src="/assets/rcl-progressive-image-missing.svg"
        alt="Unavailable workspace preview"
        ratio="16 / 9"
      />
    </Showcase>
  );
}
