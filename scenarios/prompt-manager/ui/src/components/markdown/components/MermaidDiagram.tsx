import { memo, useEffect, useRef, useState } from 'react'
import { Check, Code, Copy, Eye, Loader2 } from 'lucide-react'
import { useCodeCopy } from '../hooks/useCodeCopy'
import { useResolvedTheme } from '@/hooks/use-theme'

interface MermaidDiagramProps {
  /** Raw mermaid source code */
  code: string
}

// Lazy-loaded mermaid instance (singleton, mirrors getHighlighter in CodeBlock)
let mermaidPromise: Promise<typeof import('mermaid')['default']> | null = null
let currentMermaidTheme: string | null = null

async function getMermaid(isDark: boolean) {
  const desiredTheme = isDark ? 'dark' : 'default'

  // Re-initialize if theme changed
  if (mermaidPromise && currentMermaidTheme !== desiredTheme) {
    const mermaid = await mermaidPromise
    mermaid.initialize({
      startOnLoad: false,
      theme: desiredTheme,
      securityLevel: 'strict',
      fontFamily:
        "'JetBrains Mono', 'Fira Code', 'Cascadia Code', ui-monospace, monospace",
      themeVariables: isDark
        ? {
            primaryColor: '#334155',
            primaryTextColor: '#e2e8f0',
            primaryBorderColor: '#475569',
            lineColor: '#6366f1',
            secondaryColor: '#1e293b',
            tertiaryColor: '#0f172a',
            noteBkgColor: '#1e293b',
            noteTextColor: '#e2e8f0',
            noteBorderColor: '#475569',
          }
        : {
            primaryColor: '#e2e8f0',
            primaryTextColor: '#1e293b',
            primaryBorderColor: '#94a3b8',
            lineColor: '#6366f1',
            secondaryColor: '#f1f5f9',
            tertiaryColor: '#f8fafc',
            noteBkgColor: '#f1f5f9',
            noteTextColor: '#1e293b',
            noteBorderColor: '#94a3b8',
          },
    })
    currentMermaidTheme = desiredTheme
    return mermaid
  }

  if (!mermaidPromise) {
    currentMermaidTheme = desiredTheme
    mermaidPromise = import('mermaid')
      .then((mod) => {
        const mermaid = mod.default
        mermaid.initialize({
          startOnLoad: false,
          theme: desiredTheme,
          securityLevel: 'strict',
          fontFamily:
            "'JetBrains Mono', 'Fira Code', 'Cascadia Code', ui-monospace, monospace",
          themeVariables: isDark
            ? {
                primaryColor: '#334155',
                primaryTextColor: '#e2e8f0',
                primaryBorderColor: '#475569',
                lineColor: '#6366f1',
                secondaryColor: '#1e293b',
                tertiaryColor: '#0f172a',
                noteBkgColor: '#1e293b',
                noteTextColor: '#e2e8f0',
                noteBorderColor: '#475569',
              }
            : {
                primaryColor: '#e2e8f0',
                primaryTextColor: '#1e293b',
                primaryBorderColor: '#94a3b8',
                lineColor: '#6366f1',
                secondaryColor: '#f1f5f9',
                tertiaryColor: '#f8fafc',
                noteBkgColor: '#f1f5f9',
                noteTextColor: '#1e293b',
                noteBorderColor: '#94a3b8',
              },
        })
        return mermaid
      })
      .catch((err: unknown) => {
        mermaidPromise = null
        currentMermaidTheme = null
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
  const resolvedTheme = useResolvedTheme()
  const isDark = resolvedTheme === 'dark'

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
          const mermaid = await getMermaid(isDark)
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
  }, [code, isDark])

  // Empty diagram
  if (!code.trim()) {
    return (
      <div className="relative group rounded-lg overflow-hidden my-3">
        <div className="flex items-center justify-between px-4 py-2 bg-muted border-b border-border">
          <span className="text-xs text-muted-foreground font-mono">mermaid</span>
        </div>
        <div className="bg-muted/60 p-8 flex items-center justify-center">
          <span className="text-sm text-muted-foreground italic">Empty diagram</span>
        </div>
      </div>
    )
  }

  return (
    <div className="relative group rounded-lg overflow-hidden my-3">
      {/* Header bar */}
      <div className="flex items-center justify-between px-4 py-2 bg-muted border-b border-border">
        <span className="text-xs text-muted-foreground font-mono">mermaid</span>
        <div className="flex items-center gap-2">
          {/* Source/diagram toggle */}
          <button
            onClick={() => setShowSource((prev) => !prev)}
            className="flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
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
            className="flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
            aria-label={copied ? 'Copied' : 'Copy code'}
            type="button"
          >
            {copied ? (
              <>
                <Check className="h-3.5 w-3.5 text-green-500" />
                <span className="text-green-500">Copied</span>
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
      <div className="bg-muted/60 overflow-x-auto">
        {showSource ? (
          <pre className="p-4 text-sm text-foreground font-mono whitespace-pre overflow-x-auto">
            {code}
          </pre>
        ) : error ? (
          <div>
            <div className="px-4 py-2 bg-red-500/10 border-b border-red-500/30 text-xs text-red-600 dark:text-red-300">
              {error}
            </div>
            <pre className="p-4 text-sm text-foreground font-mono whitespace-pre overflow-x-auto">
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
            <Loader2 className="h-6 w-6 text-muted-foreground animate-spin" />
          </div>
        )}
      </div>
    </div>
  )
})
