/**
 * @libraryId react-component-library:List
 * @displayName List
 * @description The semantic collection with sections, item actions, selection, keyboard navigation, loading placeholders, separators, and responsive density.
 * @version 1.2.1-draft.1
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource react-component-library:List */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import type { HTMLAttributes, ReactNode } from "react";

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
  variant?: "surface" | "plain";
}

const styles = `
  [data-rcl-list] { min-inline-size: 0; color: var(--color-foreground); }
  [data-rcl-list][data-variant="surface"] { overflow: hidden; border: 1px solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); box-shadow: var(--elev-raised); }
  [data-rcl-list][data-variant="plain"] { display: grid; gap: var(--space-3xs); overflow: visible; }
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
  [data-rcl-list][data-variant="plain"] [data-rcl-list-items] { gap: var(--space-3xs); padding: 0; list-style: none; }
  [data-rcl-list][data-variant="plain"] [data-rcl-list-item] { grid-template-columns: minmax(0, 1fr); gap: var(--space-3xs); padding: 0; border: 0; }
  [data-rcl-list][data-variant="plain"] [data-rcl-list-mark] { display: none; }
  [data-rcl-list][data-variant="plain"] [data-rcl-list-copy] { gap: var(--space-3xs); }
  [data-rcl-list][data-variant="plain"] [data-rcl-list-item-title], [data-rcl-list][data-variant="plain"] [data-rcl-list-item-description] { color: inherit; font: var(--text-body-sm); text-align: start; }
  @media (max-width: 30rem) { [data-rcl-list-header], [data-rcl-list-item] { padding-inline: var(--space-md); } [data-rcl-list-item] { grid-template-columns: auto minmax(0, 1fr); } [data-rcl-list-meta] { grid-column: 2; text-align: start; } }
  [data-rcl-list-item-row] { display: flex; flex-direction: column; gap: var(--space-2xs); min-inline-size: 0; padding: var(--space-sm); }
  [data-rcl-list-item-row] + [data-rcl-list-item-row] { border-block-start: 1px solid var(--color-border); }
  [data-rcl-list-item-row-head] { display: flex; flex-wrap: wrap; align-items: flex-start; justify-content: space-between; gap: var(--space-2xs) var(--space-xs); min-inline-size: 0; }
  [data-rcl-list-item-row-primary] { flex: 1 1 15rem; min-inline-size: 0; }
  [data-rcl-list-item-row-content], [data-rcl-list-item-row-footer] { min-inline-size: 0; }
  [data-rcl-list-item-row-footer] { display: flex; flex-direction: column; gap: var(--space-2xs); }
  @media (max-width: 30rem) { [data-rcl-list-item-row] { padding-inline: var(--space-md); } }
`;

export interface ListItemRowProps extends Omit<HTMLAttributes<HTMLLIElement>, "content"> {
  primary: ReactNode;
  aside?: ReactNode;
  content?: ReactNode;
  footer?: ReactNode;
}

/** A semantic, unframed list row with composable primary, aside, content and footer slots. */
export function ListItemRow({ primary, aside, content, footer, ...props }: ListItemRowProps) {
  return (
    <>
      <StyleSheet libraryId="react-component-library:List" version="1.2.1-draft.1" css={styles} />
      <li {...props} data-rcl-list-item-row="">
        <div data-rcl-list-item-row-head="">
          <div data-rcl-list-item-row-primary="">{primary}</div>
          {aside ? <div data-rcl-list-item-row-aside="">{aside}</div> : null}
        </div>
        {content ? <div data-rcl-list-item-row-content="">{content}</div> : null}
        {footer ? <div data-rcl-list-item-row-footer="">{footer}</div> : null}
      </li>
    </>
  );
}

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
  variant = "surface",
}: ListProps) {
  const libraryStrings = useStrings();
  label = label ?? libraryStrings("data-display.list.list", "List");
  const Root = variant === "plain" ? "div" : "section";
  return (
    <>
      <StyleSheet libraryId="react-component-library:List" version="1.2.1-draft.1" css={styles} />
      <Root
        data-testid="data-display.list"
        className={className}
        style={style}
        data-rcl-list
        data-variant={variant}
        aria-label={label}
        role={variant === "plain" ? "list" : undefined}
      >
        {(title || description) && (
          <header data-rcl-list-header>
            {title && <strong data-rcl-list-title>{title}</strong>}
            {description && <span data-rcl-list-description>{description}</span>}
          </header>
        )}
        {items.length ? (
          variant === "plain" ? (
            <div data-rcl-list-items>
              {items.map((item, index) => {
                const normalized = normalizeItem(item, index);
                return (
                  <div key={normalized.key} data-rcl-list-item role="listitem">
                    <span data-rcl-list-copy>
                      <span data-rcl-list-item-title>{normalized.title}</span>
                      {normalized.description && (
                        <span data-rcl-list-item-description>{normalized.description}</span>
                      )}
                    </span>
                    {normalized.meta && <span data-rcl-list-meta>{normalized.meta}</span>}
                  </div>
                );
              })}
            </div>
          ) : (
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
          )
        ) : (
          <div data-rcl-list-empty role="status">
            {empty}
          </div>
        )}
      </Root>
    </>
  );
});
