import { useState } from "react";
import { StoryPalette, type StoryPaletteItem } from "./StoryPalette";

const stories: StoryPaletteItem[] = [
  { id: "ready", label: "Ready" },
  { id: "loading", label: "Loading" },
  { id: "empty", label: "Empty" },
  { id: "error", label: "Error" },
];

export function Default() {
  const [selectedId, setSelectedId] = useState("ready");
  const selected = stories.find((story) => story.id === selectedId)?.label;
  return (
    <div
      style={{
        display: "grid",
        gap: "var(--space-lg)",
        padding: "var(--space-xl)",
      }}
    >
      <div style={{ display: "grid", gap: "var(--space-2xs)" }}>
        <span
          style={{
            color: "var(--color-primary)",
            font: "var(--text-overline)",
            letterSpacing: ".1em",
            textTransform: "uppercase",
          }}
        >
          Story navigation
        </span>
        <h1 style={{ margin: 0, font: "var(--text-heading)" }}>
          Every state is one decision away
        </h1>
        <p
          style={{
            margin: 0,
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
          }}
        >
          Keep the declared state visible while moving through a specimen.
        </p>
      </div>
      <StoryPalette
        stories={stories}
        selectedId={selectedId}
        onSelect={setSelectedId}
      />
      <p
        role="status"
        style={{
          margin: 0,
          color: "var(--color-muted-foreground)",
          font: "var(--text-body-sm)",
        }}
      >
        Showing: {selected}
      </p>
    </div>
  );
}
