/**
 * @libraryId react-component-library:DescriptionList
 * @displayName DescriptionList
 * @description A semantic key-value surface for readable resource metadata across narrow and wide layouts.
 * @version 1.0.2
 * @tags ["data-display","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource data-display.description-list */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

const styles = `
[data-rcl-description-list] { display: grid; margin: 0; border: 1px solid var(--color-border, #cbd5e1); border-radius: var(--radius-panel, .75rem); overflow: hidden; container-type: inline-size; }
[data-rcl-description-list-row] { display: grid; grid-template-columns: minmax(8rem, .35fr) minmax(0, 1fr); gap: var(--space-sm, .75rem); min-width: 0; padding: var(--space-sm, 1rem); }
[data-rcl-description-list-row]:nth-child(even) { background: var(--color-surface-muted, #f1f5f9); }
[data-rcl-description-list-term] { color: var(--color-muted-foreground, #64748b); overflow-wrap: anywhere; }
[data-rcl-description-list-value] { min-width: 0; margin: 0; overflow-wrap: anywhere; font-weight: 600; }
@container (max-width: 24rem) { [data-rcl-description-list-row] { grid-template-columns: 1fr; gap: var(--space-3xs, .25rem); } }
@media (forced-colors: active) { [data-rcl-description-list] { border-color: CanvasText; } [data-rcl-description-list-row] { background: Canvas; color: CanvasText; } }
`;

export const DescriptionList = withClassName(function DescriptionList({
  entries = [],
}: {
  entries?: Array<{ term: string; description: string }>;
}) {
  return (
    <dl data-testid="data-display.description-list" data-rcl-description-list>
      <style data-rcl-description-list-styles dangerouslySetInnerHTML={{ __html: styles }} />
      {entries.map((entry, index) => (
        <div key={entry.term} data-rcl-description-list-row data-row-index={index}>
          <dt data-rcl-description-list-term>{entry.term}</dt>
          <dd data-rcl-description-list-value>{entry.description}</dd>
        </div>
      ))}
    </dl>
  );
});
