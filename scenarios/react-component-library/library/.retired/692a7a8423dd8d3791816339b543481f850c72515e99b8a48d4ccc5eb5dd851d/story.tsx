import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import type { ReactNode } from "react";
import {
  ProgressiveImage,
  type ProgressiveImageProps,
} from "./ProgressiveImage";

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
  const libraryStrings = useStrings();
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
          {libraryStrings(
            "primitives.progressive-image.media-primitive",
            "Media primitive",
          )}
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

type Args = Pick<ProgressiveImageProps, "displayState">;

export function Default({ args }: StoryHarnessProps<Args>) {
  const libraryStrings = useStrings();
  return (
    <Showcase
      title={libraryStrings(
        "primitives.progressive-image.title",
        "Arrival without the pop",
      )}
      detail="The frame is reserved immediately and the image reveals itself softly."
    >
      <ProgressiveImage
        {...args}
        src={image("#1d4ed8", "#0f766e", "Aurora workspace")}
        alt={libraryStrings(
          "primitives.progressive-image.alt",
          "Abstract blue-green Aurora workspace landscape",
        )}
        ratio="16 / 9"
      />
    </Showcase>
  );
}

export function Loading({ args }: StoryHarnessProps<Args>) {
  const libraryStrings = useStrings();
  return (
    <Showcase
      title={libraryStrings(
        "primitives.progressive-image.title.inspect-the-reserved-frame",
        "Inspect the reserved frame",
      )}
      detail="A controlled loading state makes the skeleton and layout reservation observable."
    >
      <ProgressiveImage
        {...args}
        src={image("#475569", "#334155", "Loading workspace")}
        alt={libraryStrings(
          "primitives.progressive-image.alt.abstract-loading-workspace-preview",
          "Abstract loading workspace preview",
        )}
        ratio="16 / 9"
      />
    </Showcase>
  );
}

export function ResponsiveSources({ args }: StoryHarnessProps<Args>) {
  const libraryStrings = useStrings();
  return (
    <Showcase
      title={libraryStrings(
        "primitives.progressive-image.title.responsive-by-intent",
        "Responsive by intent",
      )}
      detail="Art direction stays in the source contract while the primitive owns loading, naming, and layout behavior."
    >
      <ProgressiveImage
        {...args}
        src={image("#7c3aed", "#be185d", "Responsive source")}
        sources={[
          {
            media: "(min-width: 60rem)",
            srcSet: image("#0f766e", "#1d4ed8", "Wide art direction"),
          },
        ]}
        alt={libraryStrings(
          "primitives.progressive-image.alt.abstract-responsive-art-directed-landscape-ratio",
          "Abstract responsive art-directed landscape",
        )}
        ratio="3 / 2"
      />
    </Showcase>
  );
}

export function ErrorState({ args }: StoryHarnessProps<Args>) {
  const libraryStrings = useStrings();
  return (
    <Showcase
      title={libraryStrings(
        "primitives.progressive-image.title.a-useful-failure",
        "A useful failure",
      )}
      detail="A broken source becomes a legible recovery surface instead of an empty rectangle."
    >
      <ProgressiveImage
        {...args}
        src={image("#7f1d1d", "#991b1b", "Unavailable workspace")}
        alt={libraryStrings(
          "primitives.progressive-image.alt.unavailable-workspace-preview",
          "Unavailable workspace preview",
        )}
        ratio="16 / 9"
      />
    </Showcase>
  );
}
