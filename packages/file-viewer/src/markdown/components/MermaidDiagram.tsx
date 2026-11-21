import { memo, useState } from "react";
import { Code, Eye, Loader2, Maximize2 } from "lucide-react";
import { useFileViewerText } from "../../i18n/context.js";
import { CopyIconButton } from "@vrooli/react-component-library/CopyIconButton/1";
import { strings } from "../../i18n/strings.js";
import { copyText } from "../../lib/clipboard.js";
import { useMermaidSvg } from "../hooks/useMermaidSvg.js";

interface MermaidDiagramProps {
  code: string;
  /** Open the diagram in the full-screen zoomable viewer. */
  onOpenFullscreen?: (code: string) => void;
}

/** Renders a mermaid diagram from source code with source/diagram toggle. */
export const MermaidDiagram = memo(function MermaidDiagram({ code, onOpenFullscreen }: MermaidDiagramProps) {
  const { t } = useFileViewerText();
  const { svgHtml, error, unavailable } = useMermaidSvg(code);
  const [showSource, setShowSource] = useState(false);

  if (!code.trim()) {
    return (
      <div className="relative group rounded-lg overflow-hidden my-3">
        <div className="flex items-center justify-between px-4 py-2 bg-fv-surface-base border-b border-fv-default">
          <span className="text-xs text-fv-text-muted font-mono">mermaid</span>
        </div>
        <div className="p-8 flex items-center justify-center">
          <span className="text-sm text-fv-text-faint italic">{t(strings.markdown.diagram.emptyDiagram)}</span>
        </div>
      </div>
    );
  }

  return (
    <div className="relative group rounded-lg overflow-hidden my-3">
      <div className="flex items-center justify-between px-4 py-2 bg-fv-surface-base border-b border-fv-default">
        <span className="text-xs text-fv-text-muted font-mono">mermaid</span>
        <div className="flex items-center gap-2">
          {onOpenFullscreen && (
            <button
              onClick={() => onOpenFullscreen(code)}
              className="flex items-center gap-1.5 text-xs text-fv-text-muted hover:text-fv-text-primary transition-colors"
              aria-label={t(strings.markdown.diagram.openFullscreen)}
              title={t(strings.markdown.diagram.openFullscreen)}
              type="button"
            >
              <Maximize2 className="h-3.5 w-3.5" /><span className="hidden sm:inline">{t(strings.markdown.diagram.expand)}</span>
            </button>
          )}
          <button
            onClick={() => setShowSource((prev) => !prev)}
            className="flex items-center gap-1.5 text-xs text-fv-text-muted hover:text-fv-text-primary transition-colors"
            aria-label={showSource ? t(strings.markdown.diagram.showDiagram) : t(strings.markdown.diagram.showSource)}
            type="button"
          >
            {showSource ? (
              <><Eye className="h-3.5 w-3.5" /><span className="hidden sm:inline">{t(strings.markdown.diagram.diagram)}</span></>
            ) : (
              <><Code className="h-3.5 w-3.5" /><span className="hidden sm:inline">{t(strings.markdown.diagram.source)}</span></>
            )}
          </button>
          <CopyIconButton
            size="xs"
            value={code}
            writeText={copyText}
            aria-label={t(strings.markdown.diagram.copySource)}
            copiedLabel={t(strings.markdown.diagram.copied)}
            failedLabel={t(strings.markdown.copyFailed)}
          />
        </div>
      </div>
      <div className="overflow-x-auto">
        {showSource ? (
          <pre className="p-4 text-sm text-fv-text-primary font-mono whitespace-pre overflow-x-auto">
            {code}
          </pre>
        ) : unavailable ? (
          <div>
            <div role="status" className="px-4 py-2 border-b border-fv-default text-xs text-fv-text-muted">
              {t(strings.markdown.diagram.unavailable)}
            </div>
            <pre className="p-4 text-sm text-fv-text-primary font-mono whitespace-pre overflow-x-auto">
              {code}
            </pre>
          </div>
        ) : error ? (
          <div>
            <div className="px-4 py-2 bg-red-900/40 border-b border-red-700/50 text-xs text-red-300">
              {error}
            </div>
            <pre className="p-4 text-sm text-fv-text-primary font-mono whitespace-pre overflow-x-auto">
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
            <Loader2 className="h-6 w-6 text-fv-text-muted animate-spin" />
          </div>
        )}
      </div>
    </div>
  );
});
