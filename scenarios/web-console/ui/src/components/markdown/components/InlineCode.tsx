import type { ReactNode } from "react";
import { useMemo } from "react";
import { Check, Copy, FileText } from "lucide-react";
import { useCodeCopy } from "../hooks/useCodeCopy";
import { looksLikeInlineFileReference } from "../../../lib/fileReferences";

interface InlineCodeProps {
  children: ReactNode;
  /**
   * When provided and the inline code text looks like a file path, the chip
   * becomes a button that opens the file in the preview dialog. Lets agents
   * reference files with backticks (e.g. `~/.vrooli/plans/foo.md`) the same
   * way as markdown links.
   */
  onFileReferenceClick?: (path: string) => void;
}

/** Styled inline code with hover-reveal copy button. */
export function InlineCode({ children, onFileReferenceClick }: InlineCodeProps) {
  const textContent = useMemo(() => extractTextContent(children), [children]);
  const { copied, copyCode } = useCodeCopy(textContent);

  const isFileRef =
    onFileReferenceClick !== undefined &&
    textContent.length > 0 &&
    looksLikeInlineFileReference(textContent);

  if (isFileRef) {
    return (
      <span className="group inline-flex max-w-full items-center gap-1 rounded-full border border-wc-default bg-wc-surface-raised/80 px-2 py-0.5 text-xs font-mono align-middle">
        <button
          type="button"
          onClick={() => onFileReferenceClick?.(textContent)}
          className="inline-flex min-w-0 items-center gap-1 text-wc-accent hover:text-wc-accent/80 underline underline-offset-2"
          title={`Open ${textContent}`}
        >
          <FileText className="h-3 w-3 shrink-0" aria-hidden="true" />
          <code className="min-w-0 break-all [overflow-wrap:anywhere] leading-relaxed">{children}</code>
        </button>
        <button
          type="button"
          onClick={copyCode}
          className="opacity-0 group-hover:opacity-100 transition-opacity text-wc-text-muted hover:text-wc-text-primary"
          aria-label={copied ? "Copied" : "Copy path"}
        >
          {copied ? (
            <Check className="h-3.5 w-3.5 text-green-400" />
          ) : (
            <Copy className="h-3.5 w-3.5" />
          )}
        </button>
      </span>
    );
  }

  return (
    <span className="group inline-flex max-w-full items-center gap-1 rounded-full border border-wc-default bg-wc-surface-raised/80 px-2 py-0.5 text-xs font-mono text-wc-text-primary align-middle">
      <code className="leading-relaxed min-w-0 break-all [overflow-wrap:anywhere]">{children}</code>
      {textContent ? (
        <button
          type="button"
          onClick={copyCode}
          className="opacity-0 group-hover:opacity-100 transition-opacity text-wc-text-muted hover:text-wc-text-primary"
          aria-label={copied ? "Copied" : "Copy inline code"}
        >
          {copied ? (
            <Check className="h-3.5 w-3.5 text-green-400" />
          ) : (
            <Copy className="h-3.5 w-3.5" />
          )}
        </button>
      ) : null}
    </span>
  );
}

function extractTextContent(children: ReactNode): string {
  if (typeof children === "string") return children;
  if (Array.isArray(children)) return children.map(extractTextContent).join("");
  if (children && typeof children === "object" && "props" in children) {
    return extractTextContent((children as { props: { children?: ReactNode } }).props.children);
  }
  return "";
}
