import { Check, Copy } from "lucide-react";
import * as React from "react";

import { cn } from "../../lib/utils";

export interface CodeBlockProps {
  code: string;
  language?: string;
  showLineNumbers?: boolean;
  caption?: string;
  className?: string;
  "data-testid"?: string;
  copyLabel: string;
  copiedLabel: string;
  copyShortLabel: string;
}

export function CodeBlock({
  code,
  language,
  showLineNumbers = false,
  caption,
  className,
  "data-testid": testId,
  copyLabel,
  copiedLabel,
  copyShortLabel,
}: CodeBlockProps) {
  const [copied, setCopied] = React.useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1800);
    } catch {
      // ignore — clipboard may be unavailable in sandboxed contexts
    }
  };

  const lines = code.split("\n");

  return (
    <figure
      data-testid={testId}
      className={cn(
        "relative overflow-hidden rounded-panel border border-app-border bg-app-shell text-app-foreground",
        className,
      )}
    >
      <div className="flex items-center justify-between border-b border-app-border bg-app-surface-muted px-3 py-1.5">
        <span className="font-mono text-xs uppercase tracking-wide text-app-muted-foreground">
          {language ?? "text"}
        </span>
        <button
          type="button"
          aria-label={copyLabel}
          onClick={() => { void handleCopy(); }}
          className="inline-flex h-7 items-center gap-1 rounded-control px-2 text-xs text-app-muted-foreground hover:bg-app-surface hover:text-app-foreground"
        >
          {copied ? <Check aria-hidden className="h-3.5 w-3.5" /> : <Copy aria-hidden className="h-3.5 w-3.5" />}
          {copied ? copiedLabel : copyShortLabel}
        </button>
      </div>
      <pre className="overflow-x-auto bg-app-shell px-4 py-3 font-mono text-xs leading-relaxed text-app-primary-foreground">
        <code>
          {showLineNumbers
            ? lines.map((line, i) => (
                <div key={i} className="grid grid-cols-[2.5rem_1fr] gap-2">
                  <span aria-hidden className="select-none text-app-muted-foreground">
                    {i + 1}
                  </span>
                  <span className="whitespace-pre-wrap break-all">{line || " "}</span>
                </div>
              ))
            : <span className="whitespace-pre-wrap break-all">{code}</span>}
        </code>
      </pre>
      {caption ? (
        <figcaption className="border-t border-app-border bg-app-surface-muted px-3 py-1.5 text-xs text-app-muted-foreground">
          {caption}
        </figcaption>
      ) : null}
    </figure>
  );
}
