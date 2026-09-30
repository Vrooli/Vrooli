import { useState } from "react";
import { PromptComposer } from "./PromptComposer";
import type { ComposerAttachment } from "@vrooli/react-component-library/ComposerAttachmentTray/1";
export function ComposerStory({ args }: { args: Record<string, unknown> }) {
  const [value, setValue] = useState(String(args.value ?? ""));
  const [attachments, setAttachments] = useState((args.attachments ?? []) as ComposerAttachment[]);
  const [attempt, setAttempt] = useState(0);
  return (
    <div style={{ maxWidth: "48rem", marginInline: "auto", width: "100%" }}>
      <PromptComposer
        value={value}
        onValueChange={setValue}
        offline={Boolean(args.offline)}
        maxLength={args.maxLength as number | undefined}
        context={args.context as string | undefined}
        attachments={attachments}
        onAttachmentRemove={(id) => setAttachments((rows) => rows.filter((row) => row.id !== id))}
        onAttachmentReorder={(ids) =>
          setAttachments((rows) => ids.map((id) => rows.find((row) => row.id === id)!))
        }
        onAttachmentRetry={(id) =>
          setAttachments((rows) =>
            rows.map((row) => (row.id === id ? { ...row, status: "success" } : row)),
          )
        }
        onFiles={(files) =>
          setAttachments((rows) => [
            ...rows,
            ...files.map((file, index) => ({
              id: `${file.name}-${Date.now()}-${index}`,
              name: file.name,
              mimeType: file.type,
              sizeBytes: file.size,
              status: "success" as const,
            })),
          ])
        }
        onSent={(sent) =>
          setAttachments((rows) => rows.filter((row) => !sent.attachmentIds.includes(row.id)))
        }
        onSend={async () => {
          setAttempt((n) => n + 1);
          await new Promise((resolve) => setTimeout(resolve, args.slow ? 3000 : 500));
          return !(args.fail && attempt === 0);
        }}
      />
    </div>
  );
}
