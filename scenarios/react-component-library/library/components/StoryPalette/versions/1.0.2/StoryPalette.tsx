/**
 * @libraryId react-component-library:StoryPalette
 * @displayName StoryPalette
 * @version 1.0.2
 * @tags ["preview","navigation","responsive"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import type { Dispatch, SetStateAction } from "react";

export type StoryPaletteItem = { id: string; label: string };

export const StoryPalette = withClassName(function StoryPalette({
  stories,
  selectedId,
  onSelect,
}: {
  stories: StoryPaletteItem[];
  selectedId?: string;
  onSelect?: Dispatch<SetStateAction<string>> | ((id: string) => void);
}) {
  return (
    <nav data-testid="preview.story-palette" aria-label="Preview stories" data-story-palette>
      <div
        style={{
          display: "flex",
          gap: "var(--space-3xs)",
          overflowX: "auto",
          paddingBlock: "var(--space-3xs)",
        }}
      >
        {stories.map((story) => {
          const selected = story.id === selectedId;
          return (
            <button
              key={story.id}
              type="button"
              aria-current={selected ? "true" : undefined}
              onClick={() => onSelect?.(story.id)}
              style={{
                minHeight: "var(--tap-target-min)",
                flex: "0 0 auto",
                paddingInline: "var(--space-sm)",
                border: `var(--border-hairline) solid ${selected ? "var(--color-primary)" : "var(--color-border)"}`,
                borderRadius: "var(--radius-control)",
                background: selected ? "var(--color-primary)" : "var(--color-surface)",
                color: selected ? "var(--color-primary-foreground)" : "var(--color-foreground)",
                font: "var(--text-body-sm)",
              }}
            >
              {story.label}
            </button>
          );
        })}
      </div>
    </nav>
  );
});
