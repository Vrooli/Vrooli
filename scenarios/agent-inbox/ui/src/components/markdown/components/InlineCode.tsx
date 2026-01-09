import type { ReactNode } from "react";
import { useMemo } from "react";
import { Check, Copy } from "lucide-react";
import { useCodeCopy } from "../hooks/useCodeCopy";

interface InlineCodeProps {
  children: ReactNode;
}

/**
 * Styled inline code component matching the dark theme.
 */
export function InlineCode({ children }: InlineCodeProps) {
  const textContent = useMemo(() => extractTextContent(children), [children]);
  const { copied, copyCode } = useCodeCopy(textContent);

  return (
    <span className="group inline-flex items-center gap-1.5 rounded-full border border-slate-600/80 bg-slate-700/80 px-2 py-0.5 text-xs font-mono text-slate-200">
      <code className="leading-relaxed">{children}</code>
      {textContent ? (
        <button
          type="button"
          onClick={copyCode}
          className="opacity-0 group-hover:opacity-100 transition-opacity text-slate-400 hover:text-slate-200"
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
