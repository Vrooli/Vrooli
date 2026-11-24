/**
 * @libraryId react-component-library:DescriptionList
 * @displayName DescriptionList
 * @description The responsive name-and-value presentation for metadata, adapting between columns, rows, dense summaries, and long values without truncating meaning.
 * @version 1.1.0
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:DescriptionList
 * @vrooliComponentSourceSlot data-display.description-list */
import type { ReactNode } from "react";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

const styles = `
[data-rcl-description-list] { display: grid; min-width: 0; margin: 0; container-type: inline-size; }
[data-rcl-description-list][data-appearance="surface"] { border: 1px solid var(--color-border); border-radius: var(--radius-panel); overflow: hidden; }
[data-rcl-description-list][data-appearance="plain"] { gap: var(--space-xs); }
[data-rcl-description-list-row] { display: grid; grid-template-columns: minmax(8rem, .35fr) minmax(0, 1fr); gap: var(--space-sm); min-width: 0; padding: var(--space-sm); }
[data-rcl-description-list-row]:nth-child(even) { background: var(--color-surface-muted); }
[data-rcl-description-list-term] { color: var(--color-muted-foreground); overflow-wrap: anywhere; }
[data-rcl-description-list-value] { min-width: 0; margin: 0; overflow-wrap: anywhere; font-weight: 600; }
@container (max-width: 24rem) { [data-rcl-description-list-row] { grid-template-columns: 1fr; gap: var(--space-3xs); } }

`;

export const DescriptionList = withClassName(function DescriptionList({
  entries = [],
  children,
  appearance = "surface",
}: {
  entries?: Array<{ term: string; description: string }>;
  children?: ReactNode;
  appearance?: "surface" | "plain";
}) {
  return (
    <dl
      data-testid="data-display.description-list"
      data-rcl-description-list
      data-appearance={appearance}
    >
      <StyleSheet
        libraryId="react-component-library:DescriptionList"
        version="1.1.0"
        css={styles}
      />
      {children !== undefined
        ? children
        : entries.map((entry, index) => (
            <div key={entry.term} data-rcl-description-list-row data-row-index={index}>
              <dt data-rcl-description-list-term>{entry.term}</dt>
              <dd data-rcl-description-list-value>{entry.description}</dd>
            </div>
          ))}
    </dl>
  );
});
