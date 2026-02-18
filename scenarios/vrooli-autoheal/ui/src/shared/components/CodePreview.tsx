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
    <div className={`relative group rounded-lg overflow-hidden ${className}`}>
      <div className="flex items-center justify-between px-4 py-2 bg-slate-900 border-b border-slate-700">
        <span className="text-xs text-slate-400 font-mono">{displayLang}</span>
        <button
          onClick={copyCode}
          className="flex items-center gap-1.5 text-xs text-slate-400 hover:text-slate-200 transition-colors"
          aria-label={copied ? "Copied" : "Copy code"}
          type="button"
        >
          {copied ? (
            <>
              <Check className="h-3.5 w-3.5 text-green-400" />
              <span className="text-green-400">Copied</span>
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
        className="bg-slate-800 overflow-x-auto"
        style={{ maxHeight, overflowY: maxHeight !== "none" ? "auto" : undefined }}
      >
        {highlightedHtml ? (
          <div
            className="p-4 text-sm [&>pre]:!bg-transparent [&>pre]:!m-0 [&>pre]:!p-0"
            dangerouslySetInnerHTML={{ __html: highlightedHtml }}
          />
        ) : (
          <pre className="p-4 text-sm text-slate-200 font-mono whitespace-pre-wrap overflow-x-auto">
            {codeString}
          </pre>
        )}
      </div>
    </div>
  );
});
