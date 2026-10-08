/**
 * @libraryId react-component-library:List
 * @displayName List
 * @version 1.0.6
 * @tags ["data-display","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource react-component-library:List */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import type { ReactNode } from "react";

export interface ListItem {
  id?: string;
  title: ReactNode;
  description?: ReactNode;
  meta?: ReactNode;
  tone?: "default" | "success" | "warning" | "danger";
}

export interface ListProps {
  items?: Array<string | ListItem>;
  empty?: ReactNode;
  title?: ReactNode;
  description?: ReactNode;
  label?: string;
  className?: string;
  style?: React.CSSProperties;
}

const styles = `
  [data-rcl-list] { min-inline-size: 0; overflow: hidden; border: 1px solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); color: var(--color-foreground); box-shadow: var(--elev-raised); }
  [data-rcl-list-header] { display: grid; gap: var(--space-3xs); padding: var(--space-md) var(--space-lg); border-block-end: 1px solid var(--color-border); background: color-mix(in srgb, var(--color-primary) 4%, var(--color-surface-raised)); }
  [data-rcl-list-title] { font: var(--text-subtitle); }
  [data-rcl-list-description] { color: var(--color-muted-foreground); font: var(--text-caption); }
  [data-rcl-list-items] { display: grid; margin: 0; padding: 0; list-style: none; }
  [data-rcl-list-item] { display: grid; grid-template-columns: auto minmax(0, 1fr) auto; align-items: center; gap: var(--space-sm); min-inline-size: 0; padding: var(--space-md) var(--space-lg); border-block-end: 1px solid var(--color-border); }
  [data-rcl-list-item]:last-child { border-block-end: 0; }
  [data-rcl-list-mark] { inline-size: .625rem; block-size: .625rem; border-radius: 50%; background: var(--color-primary); box-shadow: 0 0 0 .25rem color-mix(in srgb, var(--color-primary) 13%, transparent); }
  [data-rcl-list-mark][data-tone="success"] { background: var(--color-success); box-shadow: 0 0 0 .25rem color-mix(in srgb, var(--color-success) 13%, transparent); }
  [data-rcl-list-mark][data-tone="warning"] { background: var(--color-warning); box-shadow: 0 0 0 .25rem color-mix(in srgb, var(--color-warning) 13%, transparent); }
  [data-rcl-list-mark][data-tone="danger"] { background: var(--color-danger); box-shadow: 0 0 0 .25rem color-mix(in srgb, var(--color-danger) 13%, transparent); }
  [data-rcl-list-copy] { display: grid; gap: var(--space-3xs); min-inline-size: 0; }
  [data-rcl-list-item-title] { overflow-wrap: anywhere; font: var(--text-label); }
  [data-rcl-list-item-description], [data-rcl-list-empty] { color: var(--color-muted-foreground); font: var(--text-caption); }
  [data-rcl-list-meta] { color: var(--color-muted-foreground); font: var(--text-caption); text-align: end; }
  [data-rcl-list-empty] { display: grid; min-block-size: 8rem; place-items: center; padding: var(--space-lg); text-align: center; }
  @media (max-width: 30rem) { [data-rcl-list-header], [data-rcl-list-item] { padding-inline: var(--space-md); } [data-rcl-list-item] { grid-template-columns: auto minmax(0, 1fr); } [data-rcl-list-meta] { grid-column: 2; text-align: start; } }
`;

function normalizeItem(item: string | ListItem, index: number): ListItem & { key: string } {
  if (typeof item === "string") return { key: `${item}-${index}`, title: item };
  return { ...item, key: item.id ?? `item-${index}` };
}

export const List = withClassName(function List({
  items = [],
  empty = "Nothing here yet.",
  title,
  description,
  label,
  className,
  style,
}: ListProps) {
  const libraryStrings = useStrings();
  label = label ?? libraryStrings("data-display.list.list", "List");
  return (
    <>
      <StyleSheet libraryId="react-component-library:List" version="1.0.6" css={styles} />
      <section
        data-testid="data-display.list"
        className={className}
        style={style}
        data-rcl-list
        aria-label={label}
      >
        {(title || description) && (
          <header data-rcl-list-header>
            {title && <strong data-rcl-list-title>{title}</strong>}
            {description && <span data-rcl-list-description>{description}</span>}
          </header>
        )}
        {items.length ? (
          <ul data-rcl-list-items>
            {items.map((item, index) => {
              const normalized = normalizeItem(item, index);
              return (
                <li key={normalized.key} data-rcl-list-item>
                  <span
                    data-rcl-list-mark
                    data-tone={normalized.tone ?? "default"}
                    aria-hidden="true"
                  />
                  <span data-rcl-list-copy>
                    <span data-rcl-list-item-title>{normalized.title}</span>
                    {normalized.description && (
                      <span data-rcl-list-item-description>{normalized.description}</span>
                    )}
                  </span>
                  {normalized.meta && <span data-rcl-list-meta>{normalized.meta}</span>}
                </li>
              );
            })}
          </ul>
        ) : (
          <div data-rcl-list-empty role="status">
            {empty}
          </div>
        )}
      </section>
    </>
  );
});
