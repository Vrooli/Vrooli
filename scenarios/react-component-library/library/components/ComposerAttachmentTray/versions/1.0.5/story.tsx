import { useState } from "react";
import { ComposerAttachmentTray, type ComposerAttachment } from "./ComposerAttachmentTray";
export function AttachmentTrayStory({ args }: { args: Record<string, unknown> }) {
  const [items, setItems] = useState(args.items as ComposerAttachment[]);
  const disabled = args.disabled === true;
  return (
    <ComposerAttachmentTray
      items={items}
      disabled={disabled}
      onRemove={(id) => setItems((rows) => rows.filter((row) => row.id !== id))}
      onCancel={(id) => setItems((rows) => rows.filter((row) => row.id !== id))}
      onRetry={(id) =>
        setItems((rows) =>
          rows.map((row) => (row.id === id ? { ...row, status: "uploading", progress: 25 } : row)),
        )
      }
      onReorder={(ids) =>
        setItems((rows) => {
          const rowsById = new Map(rows.map((row) => [row.id, row]));
          const reordered: ComposerAttachment[] = [];
          for (const id of ids) {
            const row = rowsById.get(id);
            if (!row) return rows;
            reordered.push(row);
          }
          return reordered.length === rows.length ? reordered : rows;
        })
      }
    />
  );
}
