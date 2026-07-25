import { useState } from "react";
import { Button } from "../../../components/ui/button";

export function CopyableCommand({ command, copyLabel, copiedLabel }: {
  command: string;
  copyLabel: string;
  copiedLabel: string;
}) {
  const [copied, setCopied] = useState(false);
  return (
    <div className="flex items-center gap-2">
      <code className="flex-1 overflow-x-auto rounded-control border border-app-border bg-app-surface-muted px-2 py-1 font-mono text-xs">
        {command}
      </code>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={() => {
          if (navigator.clipboard) void navigator.clipboard.writeText(command);
          setCopied(true);
        }}
      >
        {copied ? copiedLabel : copyLabel}
      </Button>
    </div>
  );
}
