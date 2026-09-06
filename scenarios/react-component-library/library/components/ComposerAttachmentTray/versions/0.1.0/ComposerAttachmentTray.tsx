/**
 * @libraryId react-component-library:ComposerAttachmentTray
 * @displayName ComposerAttachmentTray
 * @description A responsive attachment tray with per-file progress, recovery, removal and keyboard reordering.
 * @version 0.1.0
 * @tags ["ai","attachments","composer","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource ai.composer-attachment-tray */
import { useRef, useState, type CSSProperties } from "react";
import { AttachmentPreview, type AttachmentPreviewProps } from "@vrooli/react-component-library/AttachmentPreview/1";
import { Cluster } from "@vrooli/react-component-library/Cluster/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";

export interface ComposerAttachment extends Omit<AttachmentPreviewProps, "onRemove" | "onRetry" | "onCancel"> { id: string }
export interface ComposerAttachmentTrayProps {
  items: ComposerAttachment[];
  label?: string;
  disabled?: boolean;
  onRemove?: (id: string) => void;
  onRetry?: (id: string) => void;
  onCancel?: (id: string) => void;
  onReorder?: (ids: string[]) => void;
  className?: string;
  style?: CSSProperties;
}
const css = `
[data-rcl-composer-attachments] { min-inline-size: 0; }
[data-rcl-composer-attachment] { flex: 1 1 16rem; min-inline-size: 0; max-inline-size: 100%; padding: var(--space-2xs); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-surface); background: var(--color-surface); transition: border-color var(--dur-quick) var(--ease-standard), background-color var(--dur-quick) var(--ease-standard); }
[data-rcl-composer-attachment]:focus-within { border-color: var(--color-primary); }
[data-rcl-composer-attachment-order] { display: flex; justify-content: flex-end; gap: var(--space-3xs); margin-block-start: var(--space-3xs); }
[data-rcl-composer-attachment-summary] { margin: 0 0 var(--space-2xs); color: var(--color-muted-foreground); font-size: var(--text-caption-size); }
@media (prefers-reduced-motion: reduce) { [data-rcl-composer-attachment] { transition: none; } }
`;
/** Upload ownership stays with the caller; failed items never discard healthy attachments. */
export function ComposerAttachmentTray({ items, label = "Attachments", disabled, onRemove, onRetry, onCancel, onReorder, className, style }: ComposerAttachmentTrayProps) {
  const root = useRef<HTMLDivElement>(null);
  const [announcement, setAnnouncement] = useState("");
  const move = (index: number, offset: number) => {
    if (!onReorder || disabled) return;
    const ids = items.map(item => item.id);
    [ids[index], ids[index + offset]] = [ids[index + offset], ids[index]];
    onReorder(ids);
    setAnnouncement(`${items[index].name} moved to position ${index + offset + 1} of ${items.length}.`);
  };
  const remove = (item: ComposerAttachment, index: number) => {
    onRemove?.(item.id);
    setAnnouncement(`${item.name} removed.`);
    requestAnimationFrame(() => {
      const cards = root.current?.querySelectorAll<HTMLElement>("[data-rcl-composer-attachment]");
      (cards?.[Math.min(index, (cards?.length ?? 0) - 1)]?.querySelector<HTMLElement>("button:not(:disabled)") ?? root.current)?.focus();
    });
  };
  return <div ref={root} tabIndex={-1} role="group" aria-label={label} data-rcl-composer-attachments className={className} style={style}>
    <StyleSheet name="composer-attachment-tray-1" css={css} />
    <p data-rcl-composer-attachment-summary role="status">{announcement || (items.length ? `${items.length} ${items.length === 1 ? "attachment" : "attachments"}` : "No attachments")}</p>
    <Cluster gap="2xs">
      {items.map(({ id, ...item }, index) => <div key={id} data-rcl-composer-attachment>
        <AttachmentPreview {...item}
          onRemove={!disabled && onRemove ? () => remove({ id, ...item }, index) : undefined}
          onRetry={!disabled && onRetry ? () => onRetry(id) : undefined}
          onCancel={!disabled && onCancel ? () => onCancel(id) : undefined} />
        {onReorder && items.length > 1 && <div data-rcl-composer-attachment-order>
          <Button type="button" variant="ghost" size="sm" disabled={disabled || index === 0} aria-label={`Move ${item.name} earlier`} onClick={() => move(index, -1)}>Move earlier</Button>
          <Button type="button" variant="ghost" size="sm" disabled={disabled || index === items.length - 1} aria-label={`Move ${item.name} later`} onClick={() => move(index, 1)}>Move later</Button>
        </div>}
      </div>)}
    </Cluster>
  </div>;
}