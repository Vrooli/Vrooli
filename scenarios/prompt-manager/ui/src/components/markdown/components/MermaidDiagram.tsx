import { memo, useEffect, useRef, useState } from 'react'
import { Check, Code, Copy, Eye, Loader2 } from 'lucide-react'
import { useCodeCopy } from '../hooks/useCodeCopy'

interface MermaidDiagramProps {
  /** Raw mermaid source code */
  code: string
}

// Lazy-loaded mermaid instance (singleton, mirrors getHighlighter in CodeBlock)
let mermaidPromise: Promise<typeof import('mermaid')['default']> | null = null

async function getMermaid() {
  if (!mermaidPromise) {
    mermaidPromise = import('mermaid')
      .then((mod) => {
        const mermaid = mod.default
        mermaid.initialize({
          startOnLoad: false,
          theme: 'dark',
          securityLevel: 'strict',
          fontFamily:
            "'JetBrains Mono', 'Fira Code', 'Cascadia Code', ui-monospace, monospace",
          themeVariables: {
            primaryColor: '#334155', // slate-700
            primaryTextColor: '#e2e8f0', // slate-200
            primaryBorderColor: '#475569', // slate-600
            lineColor: '#6366f1', // indigo-500
            secondaryColor: '#1e293b', // slate-800
            tertiaryColor: '#0f172a', // slate-900
            noteBkgColor: '#1e293b',
            noteTextColor: '#e2e8f0',
            noteBorderColor: '#475569',
          },
        })
        return mermaid
      })
      .catch((err: unknown) => {
        mermaidPromise = null
        throw err
      })
  }
  return mermaidPromise
}

/**
 * Renders a mermaid diagram from source code.
 * Lazy-loads the mermaid library and renders the SVG inline.
 */
export const MermaidDiagram = memo(function MermaidDiagram({
  code,
}: MermaidDiagramProps) {
  const [svgHtml, setSvgHtml] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [showSource, setShowSource] = useState(false)
  const { copied, copyCode } = useCodeCopy(code)
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    let cancelled = false
    setError(null)

    // Debounce rendering by 100ms to avoid thrashing during streaming
    if (debounceRef.current) {
      clearTimeout(debounceRef.current)
    }

    debounceRef.current = setTimeout(() => {
      async function render() {
        try {
          const mermaid = await getMermaid()
          if (cancelled) return

          const id = `mermaid-${crypto.randomUUID()}`
          const { svg } = await mermaid.render(id, code)
          // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- cancelled is mutated asynchronously by cleanup
          if (cancelled) return

          setSvgHtml(svg)
          setError(null)
        } catch (err) {
          if (cancelled) return
          const message =
            err instanceof Error ? err.message : 'Failed to render diagram'
          setError(message)
          setSvgHtml(null)
        }
      }

      void render()
    }, 100)

    return () => {
      cancelled = true
      if (debounceRef.current) {
        clearTimeout(debounceRef.current)
      }
    }
  }, [code])

  // Empty diagram
  if (!code.trim()) {
    return (
      <div className="relative group rounded-lg overflow-hidden my-3">
        <div className="flex items-center justify-between px-4 py-2 bg-slate-900 border-b border-slate-700">
          <span className="text-xs text-slate-400 font-mono">mermaid</span>
        </div>
        <div className="bg-slate-800 p-8 flex items-center justify-center">
          <span className="text-sm text-slate-500 italic">Empty diagram</span>
        </div>
      </div>
    )
  }

  return (
    <div className="relative group rounded-lg overflow-hidden my-3">
      {/* Header bar */}
      <div className="flex items-center justify-between px-4 py-2 bg-slate-900 border-b border-slate-700">
        <span className="text-xs text-slate-400 font-mono">mermaid</span>
        <div className="flex items-center gap-2">
          {/* Source/diagram toggle */}
          <button
            onClick={() => setShowSource((prev) => !prev)}
            className="flex items-center gap-1.5 text-xs text-slate-400 hover:text-slate-200 transition-colors"
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
          {/* Copy button */}
          <button
            onClick={copyCode}
            className="flex items-center gap-1.5 text-xs text-slate-400 hover:text-slate-200 transition-colors"
            aria-label={copied ? 'Copied' : 'Copy code'}
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
      </div>

      {/* Content */}
      <div className="bg-slate-800 overflow-x-auto">
        {showSource ? (
          <pre className="p-4 text-sm text-slate-200 font-mono whitespace-pre overflow-x-auto">
            {code}
          </pre>
        ) : error ? (
          <div>
            <div className="px-4 py-2 bg-red-900/40 border-b border-red-700/50 text-xs text-red-300">
              {error}
            </div>
            <pre className="p-4 text-sm text-slate-200 font-mono whitespace-pre overflow-x-auto">
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
            <Loader2 className="h-6 w-6 text-slate-400 animate-spin" />
          </div>
        )}
      </div>
    </div>
  )
})
