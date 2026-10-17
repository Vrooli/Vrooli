import { useState } from "react";
import { ComposerAttachmentTray, type ComposerAttachment } from "./ComposerAttachmentTray";
export function AttachmentTrayStory({ args }: { args: Record<string, unknown> }) {
  const [items, setItems] = useState(args.items as ComposerAttachment[]);
  return (
    <ComposerAttachmentTray
      items={items}
      onRemove={(id) => setItems((rows) => rows.filter((row) => row.id !== id))}
      onCancel={(id) => setItems((rows) => rows.filter((row) => row.id !== id))}
      onRetry={(id) =>
        setItems((rows) =>
          rows.map((row) => (row.id === id ? { ...row, status: "uploading", progress: 25 } : row)),
        )
      }
      onReorder={(ids) => setItems((rows) => ids.map((id) => rows.find((row) => row.id === id)!))}
    />
  );
}
