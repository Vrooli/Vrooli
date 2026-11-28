import { useState } from "react";
import { Button } from "../ui/button";

// lib.dom declares navigator.clipboard as always present, but browsers expose
// it only in secure contexts. Return the honest type so the guard stays real.
function clipboardOrUndefined(): Clipboard | undefined {
  return navigator.clipboard;
}

export function CopyableCommand({ command, copyLabel, copiedLabel }: {
  command: string;
  copyLabel: string;
  copiedLabel: string;
}) {
  const [copied, setCopied] = useState(false);

  const copyCommand = async () => {
    setCopied(false);
    const clipboard = clipboardOrUndefined();
    if (!clipboard) return;
    try {
      await clipboard.writeText(command);
      setCopied(true);
    } catch {
      // Keep the action label honest when the browser rejects clipboard access.
    }
  };

  return (
    <div className="flex items-center gap-2">
      <code className="flex-1 overflow-x-auto rounded-control border border-app-border bg-app-surface-muted px-2 py-1 font-mono text-xs">
        {command}
      </code>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={() => void copyCommand()}
      >
        {copied ? copiedLabel : copyLabel}
      </Button>
    </div>
  );
}
