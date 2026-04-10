import { memo, useEffect, useRef, useState } from 'react';
import { Check, Code, Copy, Eye, Loader2 } from 'lucide-react';
import { useCodeCopy } from '../hooks/useCodeCopy';
import { useTheme } from '@/contexts/ThemeContext';

interface MermaidDiagramProps {
  code: string;
}

let mermaidPromise: Promise<typeof import('mermaid')['default']> | null = null;
let currentMermaidTheme: string | null = null;

async function getMermaid(isDark: boolean) {
  const desiredTheme = isDark ? 'dark' : 'default';

  if (mermaidPromise && currentMermaidTheme !== desiredTheme) {
    const mermaid = await mermaidPromise;
    mermaid.initialize({
      startOnLoad: false,
      theme: desiredTheme,
      securityLevel: 'strict',
      fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', ui-monospace, monospace",
    });
    currentMermaidTheme = desiredTheme;
    return mermaid;
  }

  if (!mermaidPromise) {
    currentMermaidTheme = desiredTheme;
    mermaidPromise = import('mermaid')
      .then((mod) => {
        const mermaid = mod.default;
        mermaid.initialize({
          startOnLoad: false,
          theme: desiredTheme,
          securityLevel: 'strict',
          fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', ui-monospace, monospace",
        });
        return mermaid;
      })
      .catch((error: unknown) => {
        mermaidPromise = null;
        currentMermaidTheme = null;
        throw error;
      });
  }

  return mermaidPromise;
}

export const MermaidDiagram = memo(function MermaidDiagram({ code }: MermaidDiagramProps) {
  const [svgHtml, setSvgHtml] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [showSource, setShowSource] = useState(false);
  const { copied, copyCode } = useCodeCopy(code);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const { resolvedTheme } = useTheme();

  useEffect(() => {
    let cancelled = false;
    setError(null);

    if (debounceRef.current) {
      clearTimeout(debounceRef.current);
    }

    debounceRef.current = setTimeout(() => {
      async function renderDiagram() {
        try {
          const mermaid = await getMermaid(resolvedTheme === 'dark');
          if (cancelled) return;

          const id = `mermaid-${crypto.randomUUID()}`;
          const { svg } = await mermaid.render(id, code);
          if (cancelled) return;

          setSvgHtml(svg);
          setError(null);
        } catch (renderError) {
          if (cancelled) return;
          const message = renderError instanceof Error ? renderError.message : 'Failed to render diagram';
          setError(message);
          setSvgHtml(null);
        }
      }

      void renderDiagram();
    }, 100);

    return () => {
      cancelled = true;
      if (debounceRef.current) {
        clearTimeout(debounceRef.current);
      }
    };
  }, [code, resolvedTheme]);

  if (!code.trim()) {
    return (
      <div className="relative my-3 overflow-hidden rounded-lg border border-white/10">
        <div className="border-b border-white/10 bg-slate-900/80 px-3 py-2 font-mono text-xs text-slate-400">
          mermaid
        </div>
        <div className="flex items-center justify-center bg-slate-950/80 p-8 text-sm italic text-slate-500">
          Empty diagram
        </div>
      </div>
    );
  }

  return (
    <div className="relative my-3 overflow-hidden rounded-lg border border-white/10">
      <div className="flex items-center justify-between border-b border-white/10 bg-slate-900/80 px-3 py-2">
        <span className="font-mono text-xs text-slate-400">mermaid</span>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowSource((prev) => !prev)}
            className="flex items-center gap-1.5 text-xs text-slate-300 transition-colors hover:text-slate-100"
            aria-label={showSource ? 'Show diagram' : 'Show source'}
            type="button"
          >
            {showSource ? (
              <>
                <Eye className="h-3.5 w-3.5" />
                <span>Diagram</span>
              </>
            ) : (
              <>
                <Code className="h-3.5 w-3.5" />
                <span>Source</span>
              </>
            )}
          </button>
          <button
            onClick={copyCode}
            className="flex items-center gap-1.5 text-xs text-slate-300 transition-colors hover:text-slate-100"
            aria-label={copied ? 'Copied' : 'Copy code'}
            type="button"
          >
            {copied ? (
              <>
                <Check className="h-3.5 w-3.5 text-emerald-400" />
                <span className="text-emerald-400">Copied</span>
              </>
            ) : (
              <>
                <Copy className="h-3.5 w-3.5" />
                <span>Copy</span>
              </>
            )}
          </button>
        </div>
      </div>

      <div className="overflow-x-auto bg-slate-950/80">
        {showSource ? (
          <pre className="overflow-x-auto whitespace-pre p-4 font-mono text-sm text-slate-100">{code}</pre>
        ) : error ? (
          <div>
            <div className="border-b border-red-500/40 bg-red-500/10 px-4 py-2 text-xs text-red-200">{error}</div>
            <pre className="overflow-x-auto whitespace-pre p-4 font-mono text-sm text-slate-100">{code}</pre>
          </div>
        ) : svgHtml ? (
          <div
            className="flex items-center justify-center p-4 [&>svg]:max-w-full"
            dangerouslySetInnerHTML={{ __html: svgHtml }}
          />
        ) : (
          <div className="flex items-center justify-center p-8">
            <Loader2 className="h-6 w-6 animate-spin text-slate-400" />
          </div>
        )}
      </div>
    </div>
  );
});
