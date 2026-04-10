import type { ReactNode } from "react";
import { useMemo } from "react";
import { Check, Copy } from "lucide-react";
import { useCodeCopy } from "../hooks/useCodeCopy";

interface InlineCodeProps {
  children: ReactNode;
}

export function InlineCode({ children }: InlineCodeProps) {
  const textContent = useMemo(() => extractTextContent(children), [children]);
  const { status, copyCode } = useCodeCopy(textContent);
  const isCopied = status === "copied";
  const isError = status === "error";

  return (
    <span className="group rounded-md border border-border/60 bg-muted/70 px-1.5 py-0.5 text-xs font-mono text-foreground">
      <code className="leading-relaxed">{children}</code>
      {textContent ? (
        <button
          type="button"
          onClick={copyCode}
          className={`ml-1 inline-flex align-middle opacity-0 group-hover:opacity-100 transition-opacity ${
            isError ? "text-destructive" : "text-muted-foreground hover:text-foreground"
          }`}
          aria-label={isCopied ? "Copied" : "Copy inline code"}
        >
          {isCopied ? (
            <Check className="h-3 w-3 text-success" />
          ) : (
            <Copy className="h-3 w-3" />
          )}
        </button>
      ) : null}
    </span>
  );
}

function extractTextContent(children: ReactNode): string {
  if (typeof children === "string") {
    return children;
  }
  if (Array.isArray(children)) {
    return children.map(extractTextContent).join("");
  }
  if (children && typeof children === "object" && "props" in children) {
    return extractTextContent((children as { props: { children?: ReactNode } }).props.children);
  }
  return "";
}
