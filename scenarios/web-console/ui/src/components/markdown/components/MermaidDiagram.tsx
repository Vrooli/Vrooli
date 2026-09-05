import { memo, useState } from "react";
import { Check, Code, Copy, Eye, Loader2, Maximize2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../../../consts/strings";
import { useCodeCopy } from "../hooks/useCodeCopy";
import { useMermaidSvg } from "../hooks/useMermaidSvg";

interface MermaidDiagramProps {
  code: string;
  /** Open the diagram in the full-screen zoomable viewer. */
  onOpenFullscreen?: (code: string) => void;
}

/** Renders a mermaid diagram from source code with source/diagram toggle. */
export const MermaidDiagram = memo(function MermaidDiagram({ code, onOpenFullscreen }: MermaidDiagramProps) {
  const { t } = useTranslation();
  const { svgHtml, error } = useMermaidSvg(code);
  const [showSource, setShowSource] = useState(false);
  const { copied, copyCode } = useCodeCopy(code);

  if (!code.trim()) {
    return (
      <div className="relative group rounded-lg overflow-hidden my-3">
        <div className="flex items-center justify-between px-4 py-2 bg-wc-surface-base border-b border-wc-default">
          <span className="text-xs text-wc-text-muted font-mono">mermaid</span>
        </div>
        <div className="bg-wc-surface p-8 flex items-center justify-center">
          <span className="text-sm text-wc-text-faint italic">{t(strings.mermaid.emptyDiagram)}</span>
        </div>
      </div>
    );
  }

  return (
    <div className="relative group rounded-lg overflow-hidden my-3">
      <div className="flex items-center justify-between px-4 py-2 bg-wc-surface-base border-b border-wc-default">
        <span className="text-xs text-wc-text-muted font-mono">mermaid</span>
        <div className="flex items-center gap-2">
          {onOpenFullscreen && (
            <button
              onClick={() => onOpenFullscreen(code)}
              className="flex items-center gap-1.5 text-xs text-wc-text-muted hover:text-wc-text-primary transition-colors"
              aria-label={t(strings.mermaid.openFullscreen)}
              title={t(strings.mermaid.openFullscreen)}
              type="button"
            >
              <Maximize2 className="h-3.5 w-3.5" /><span className="hidden sm:inline">{t(strings.mermaid.expand)}</span>
            </button>
          )}
          <button
            onClick={() => setShowSource((prev) => !prev)}
            className="flex items-center gap-1.5 text-xs text-wc-text-muted hover:text-wc-text-primary transition-colors"
            aria-label={showSource ? t(strings.mermaid.showDiagram) : t(strings.mermaid.showSource)}
            type="button"
          >
            {showSource ? (
              <><Eye className="h-3.5 w-3.5" /><span className="hidden sm:inline">{t(strings.mermaid.diagram)}</span></>
            ) : (
              <><Code className="h-3.5 w-3.5" /><span className="hidden sm:inline">{t(strings.mermaid.source)}</span></>
            )}
          </button>
          <button
            onClick={copyCode}
            className="flex items-center gap-1.5 text-xs text-wc-text-muted hover:text-wc-text-primary transition-colors"
            aria-label={copied ? t(strings.mermaid.copied) : t(strings.mermaid.copy)}
            type="button"
          >
            {copied ? (
              <><Check className="h-3.5 w-3.5 text-green-400" /><span className="hidden text-green-400 sm:inline">{t(strings.mermaid.copied)}</span></>
            ) : (
              <><Copy className="h-3.5 w-3.5" /><span className="hidden sm:inline">{t(strings.mermaid.copy)}</span></>
            )}
          </button>
        </div>
      </div>
      <div className="bg-wc-surface overflow-x-auto">
        {showSource ? (
          <pre className="p-4 text-sm text-wc-text-primary font-mono whitespace-pre overflow-x-auto">
            {code}
          </pre>
        ) : error ? (
          <div>
            <div className="px-4 py-2 bg-red-900/40 border-b border-red-700/50 text-xs text-red-300">
              {error}
            </div>
            <pre className="p-4 text-sm text-wc-text-primary font-mono whitespace-pre overflow-x-auto">
              {code}
            </pre>
          </div>
        ) : svgHtml ? (
          <div
            className="p-4 flex items-center justify-center [&>svg]:max-w-full"
            dangerouslySetInnerHTML={{ __html: svgHtml }}
          />
        ) : (
          <div className="p-8 flex items-center justify-center">
            <Loader2 className="h-6 w-6 text-wc-text-muted animate-spin" />
          </div>
        )}
      </div>
    </div>
  );
});
