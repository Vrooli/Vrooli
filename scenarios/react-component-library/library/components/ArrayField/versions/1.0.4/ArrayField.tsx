/**
 * @libraryId react-component-library:ArrayField
 * @displayName ArrayField
 * @version 1.0.4
 * @tags ["form","array","validation","responsive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource forms.array-field */
import {
  useEffect,
  useId,
  useRef,
  useState,
  type CSSProperties,
  type ReactElement,
  type ReactNode,
} from "react";
import { FormField } from "@vrooli/react-component-library/FormField/1";
import type { FormStore } from "@vrooli/react-component-library/FormStore/1";

export interface ArrayItemActions<TItem> {
  index: number;
  canRemove: boolean;
  canMoveUp: boolean;
  canMoveDown: boolean;
  setValue: (value: TItem) => void;
  remove: () => void;
  duplicate: () => void;
  moveUp: () => void;
  moveDown: () => void;
}

export interface ArrayFieldProps<
  TValues extends Record<string, unknown>,
  K extends keyof TValues,
  TItem = TValues[K] extends Array<infer T> ? T : never,
> {
  store: FormStore<TValues>;
  field: K;
  label: ReactNode;
  description?: ReactNode;
  renderItem: (context: {
    item: TItem;
    index: number;
    actions: ArrayItemActions<TItem>;
  }) => ReactElement;
  createItem: () => TItem;
  getItemKey?: (item: TItem, index: number) => string | number;
  itemLabel?: (index: number, item: TItem) => ReactNode;
  itemError?: (item: TItem, index: number) => ReactNode;
  emptyState?: ReactNode;
  addLabel?: ReactNode;
  minItems?: number;
  maxItems?: number;
  disabled?: boolean;
  className?: string;
  style?: CSSProperties;
}

const styles = `
  [data-rcl-array-field] { display: grid; gap: var(--space-md); min-inline-size: 0; color: var(--color-foreground); }
  [data-rcl-array-header] { display: flex; align-items: flex-start; justify-content: space-between; gap: var(--space-md); }
  [data-rcl-array-heading] { display: grid; gap: var(--space-3xs); min-inline-size: 0; }
  [data-rcl-array-label] { color: var(--color-foreground); font: var(--text-subtitle); letter-spacing: var(--text-subtitle-tracking); }
  [data-rcl-array-description] { max-inline-size: 62ch; color: var(--color-muted-foreground); font: var(--text-body); }
  [data-rcl-array-count] { flex: 0 0 auto; color: var(--color-muted-foreground); font: var(--text-caption); }
  [data-rcl-array-add], [data-rcl-array-action] { min-block-size: var(--tap-target-min); border: 1px solid var(--color-border-strong); border-radius: var(--radius-control); background: var(--color-surface); color: var(--color-foreground); font: var(--text-label); cursor: pointer; transition: background var(--dur-quick) var(--ease-standard), border-color var(--dur-quick) var(--ease-standard), color var(--dur-quick) var(--ease-standard); }
  [data-rcl-array-add] { padding-inline: var(--space-sm); border-color: var(--color-primary); background: var(--color-primary); color: var(--color-primary-foreground); box-shadow: var(--elev-raised); }
  [data-rcl-array-add]:hover:not(:disabled), [data-rcl-array-action]:hover:not(:disabled) { border-color: var(--color-primary); background: color-mix(in srgb, var(--color-primary) 9%, var(--color-surface)); color: var(--color-primary); }
  [data-rcl-array-add]:disabled, [data-rcl-array-action]:disabled { cursor: not-allowed; opacity: var(--opacity-disabled); }
  [data-rcl-array-list] { display: grid; gap: var(--space-sm); min-inline-size: 0; }
  [data-rcl-array-item] { display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: var(--space-sm); align-items: start; padding: var(--space-md); border: 1px solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface-raised); box-shadow: var(--elev-subtle); }
  [data-rcl-array-item-actions] { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: var(--space-3xs); }
  [data-rcl-array-action] { inline-size: var(--tap-target-min); padding-inline: 0; }
  [data-rcl-array-empty] { display: grid; justify-items: start; gap: var(--space-2xs); padding: var(--space-lg); border: 1px dashed var(--color-border-strong); border-radius: var(--radius-panel); background: var(--color-surface); color: var(--color-muted-foreground); font: var(--text-body); }
  [data-rcl-array-status] { min-block-size: 1.25rem; color: var(--color-danger); font: var(--text-body); }
  @media (max-width: 34rem) { [data-rcl-array-header] { align-items: stretch; flex-direction: column; } [data-rcl-array-add] { inline-size: 100%; } [data-rcl-array-item] { grid-template-columns: 1fr; } [data-rcl-array-item-actions] { justify-content: stretch; } [data-rcl-array-action] { flex: 1 1 0; } }

`;

function move<T>(items: T[], from: number, to: number) {
  const next = [...items];
  const [item] = next.splice(from, 1);
  if (item !== undefined) next.splice(to, 0, item);
  return next;
}

export const ArrayField = withClassName(function ArrayField<
  TValues extends Record<string, unknown>,
  K extends keyof TValues,
  TItem = TValues[K] extends Array<infer T> ? T : never,
>({
  store,
  field,
  label,
  description,
  renderItem,
  createItem,
  getItemKey,
  itemLabel = (index) => `Item ${index + 1}`,
  itemError,
  emptyState = "Nothing here yet. Add the first item when you are ready.",
  addLabel = "Add item",
  minItems = 0,
  maxItems = Number.POSITIVE_INFINITY,
  disabled = false,
  className,
  style,
}: ArrayFieldProps<TValues, K, TItem>) {
  const [, rerender] = useState(0);
  const generatedId = useId().replace(/:/g, "");
  const lastAdded = useRef<number | undefined>(undefined);
  useEffect(() => store.subscribe(() => rerender((count) => count + 1)), [store]);

  const fieldState = store.getField(field);
  const items = Array.isArray(fieldState.value) ? (fieldState.value as unknown as TItem[]) : [];
  const canAdd = !disabled && items.length < maxItems;
  const canRemove = !disabled && items.length > minItems;
  const update = (next: TItem[]) => store.setValue(field, next as TValues[K]);
  const add = () => {
    if (!canAdd) return;
    lastAdded.current = items.length;
    update([...items, createItem()]);
  };
  const listLabel = typeof label === "string" ? label : "Items";

  return (
    <div className={className} style={style} data-rcl-array-field data-field={String(field)}>
      <StyleSheet libraryId="react-component-library:ArrayField" version="1.0.4" css={styles} />
      <div data-rcl-array-header>
        <div data-rcl-array-heading>
          <div id={`${generatedId}-label`} data-rcl-array-label>
            {label}
          </div>
          {description && (
            <div id={`${generatedId}-description`} data-rcl-array-description>
              {description}
            </div>
          )}
        </div>
        <span data-rcl-array-count aria-label={`${items.length} ${listLabel.toLowerCase()}`}>
          {items.length}
          {Number.isFinite(maxItems) ? ` / ${maxItems}` : ""}
        </span>
      </div>
      <div
        data-rcl-array-list
        role="list"
        aria-labelledby={`${generatedId}-label`}
        aria-describedby={description ? `${generatedId}-description` : undefined}
      >
        {items.length === 0 ? (
          <div data-rcl-array-empty role="status">
            <span>{emptyState}</span>
            <button
              data-testid="forms.array-field"
              type="button"
              data-rcl-array-add
              onClick={add}
              disabled={!canAdd}
            >
              {addLabel}
            </button>
          </div>
        ) : (
          items.map((item, index) => {
            const actions: ArrayItemActions<TItem> = {
              index,
              canRemove,
              canMoveUp: !disabled && index > 0,
              canMoveDown: !disabled && index < items.length - 1,
              setValue: (value) =>
                update(items.map((entry, itemIndex) => (itemIndex === index ? value : entry))),
              remove: () =>
                canRemove && update(items.filter((_, itemIndex) => itemIndex !== index)),
              duplicate: () =>
                canAdd && update([...items.slice(0, index + 1), item, ...items.slice(index + 1)]),
              moveUp: () => index > 0 && update(move(items, index, index - 1)),
              moveDown: () => index < items.length - 1 && update(move(items, index, index + 1)),
            };
            const control = renderItem({ item, index, actions });
            return (
              <div
                data-rcl-array-item
                role="listitem"
                key={getItemKey?.(item, index) ?? `${index}-${String(item)}`}
                data-last-added={lastAdded.current === index || undefined}
              >
                <FormField
                  label={itemLabel(index, item)}
                  error={itemError?.(item, index)}
                  disabled={disabled}
                  control={control}
                />
                <div
                  data-rcl-array-item-actions
                  aria-label={`Actions for ${listLabel} ${index + 1}`}
                >
                  <button
                    data-testid="forms.array-field"
                    type="button"
                    data-rcl-array-action
                    onClick={actions.moveUp}
                    disabled={!actions.canMoveUp}
                    aria-label={`Move ${listLabel.toLowerCase()} ${index + 1} up`}
                  >
                    ↑
                  </button>
                  <button
                    data-testid="forms.array-field"
                    type="button"
                    data-rcl-array-action
                    onClick={actions.moveDown}
                    disabled={!actions.canMoveDown}
                    aria-label={`Move ${listLabel.toLowerCase()} ${index + 1} down`}
                  >
                    ↓
                  </button>
                  <button
                    data-testid="forms.array-field"
                    type="button"
                    data-rcl-array-action
                    onClick={actions.duplicate}
                    disabled={!canAdd}
                    aria-label={`Duplicate ${listLabel.toLowerCase()} ${index + 1}`}
                  >
                    ＋
                  </button>
                  <button
                    data-testid="forms.array-field"
                    type="button"
                    data-rcl-array-action
                    onClick={actions.remove}
                    disabled={!canRemove}
                    aria-label={`Remove ${listLabel.toLowerCase()} ${index + 1}`}
                  >
                    ×
                  </button>
                </div>
              </div>
            );
          })
        )}
      </div>
      {items.length > 0 && (
        <button
          data-testid="forms.array-field"
          type="button"
          data-rcl-array-add
          onClick={add}
          disabled={!canAdd}
        >
          {addLabel}
        </button>
      )}
      <div data-rcl-array-status role="status" aria-live="polite">
        {fieldState.error ??
          (items.length < minItems
            ? `Add at least ${minItems - items.length} more ${listLabel.toLowerCase()}.`
            : "")}
      </div>
    </div>
  );
});
