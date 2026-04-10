import { useState, useEffect, useCallback, memo } from "react";
import { Copy, Check } from "lucide-react";
import { highlightCode } from "../../lib/highlighter";

interface CodePreviewProps {
  code: string | object | null | undefined;
  language?: string;
  maxHeight?: string;
  className?: string;
}

export const CodePreview = memo(function CodePreview({
  code,
  language = "json",
  maxHeight = "none",
  className = "",
}: CodePreviewProps) {
  const [highlightedHtml, setHighlightedHtml] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const codeString =
    code == null
      ? ""
      : typeof code === "string"
        ? code
        : JSON.stringify(code, null, 2);

  const displayLang = language || "text";

  const copyCode = useCallback(() => {
    const copyFallback = () => {
      const textArea = document.createElement("textarea");
      textArea.value = codeString;
      textArea.setAttribute("readonly", "");
      textArea.style.position = "fixed";
      textArea.style.left = "-9999px";
      document.body.appendChild(textArea);
      textArea.select();
      document.execCommand("copy");
      document.body.removeChild(textArea);
    };

    void (navigator.clipboard?.writeText(codeString).catch(() => {
      copyFallback();
    }) ?? Promise.resolve(copyFallback()))
      .then(() => {
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
      })
      .catch(() => {
        // Ignore copy failures.
      });
  }, [codeString]);

  useEffect(() => {
    let cancelled = false;

    async function highlight(): Promise<void> {
      try {
        const html = await highlightCode(codeString, displayLang);
        if (!cancelled) {
          setHighlightedHtml(html);
        }
      } catch {
        if (!cancelled) {
          setHighlightedHtml(null);
        }
      }
    }

    void highlight();

    return () => {
      cancelled = true;
    };
  }, [codeString, displayLang]);

  return (
    <div className={`relative overflow-hidden rounded-lg border border-border-default/70 bg-surface-base ${className}`}>
      <div className="flex items-center justify-between gap-2 border-b border-border-default/70 bg-surface-overlay/60 px-3 py-2 sm:px-4">
        <span className="truncate font-mono text-xs text-text-muted">{displayLang}</span>
        <button
          onClick={copyCode}
          className="flex shrink-0 items-center gap-1.5 rounded-md px-1.5 py-1 text-xs text-text-muted transition-colors hover:bg-surface-overlay/50 hover:text-text-primary"
          aria-label={copied ? "Copied" : "Copy code"}
          type="button"
        >
          {copied ? (
            <>
              <Check className="h-3.5 w-3.5 text-accent-success" />
              <span className="text-accent-success">Copied</span>
            </>
          ) : (
            <>
              <Copy className="h-3.5 w-3.5" />
              <span>Copy</span>
            </>
          )}
        </button>
      </div>

      <div
        className="overflow-x-auto bg-surface-base"
        style={{ maxHeight, overflowY: maxHeight !== "none" ? "auto" : undefined }}
      >
        {highlightedHtml ? (
          <div
            className="p-4 text-sm [&>pre]:!bg-transparent [&>pre]:!m-0 [&>pre]:!p-0"
            dangerouslySetInnerHTML={{ __html: highlightedHtml }}
          />
        ) : (
          <pre className="overflow-x-auto whitespace-pre-wrap p-4 font-mono text-sm text-text-primary">
            {codeString}
          </pre>
        )}
      </div>
    </div>
  );
});
