/**
 * TipTapCodeBlock - Custom code block node view for TipTap.
 *
 * Features:
 * - Shiki syntax highlighting (lazy-loaded)
 * - Language label in header
 * - Copy button with feedback
 * - Clean, dark-themed styling
 */

import { NodeViewContent, NodeViewWrapper, type NodeViewProps } from '@tiptap/react'
import { memo, useCallback, useEffect, useMemo, useState, useSyncExternalStore } from 'react'
import { useResolvedTheme } from '@/hooks/use-theme'
import { Check, Copy } from 'lucide-react'
import { detectLanguage, normalizeLanguage } from './languageDetection'

/**
 * Generate line numbers for code.
 */
function generateLineNumbers(code: string): string[] {
  const lines = code.split('\n')
  return lines.map((_, index) => String(index + 1))
}

// Shiki highlighter type
interface ShikiHighlighter {
  getLoadedLanguages(): string[]
  codeToHtml(code: string, options: { lang: string; theme: string }): string
}

// Lazy-loaded shiki highlighter
let highlighterPromise: Promise<ShikiHighlighter> | null = null

async function getHighlighter(): Promise<ShikiHighlighter> {
  if (!highlighterPromise) {
    highlighterPromise = import('shiki').then((shiki) =>
      (shiki.createHighlighter as (options: {
        themes: string[]
        langs: string[]
      }) => Promise<ShikiHighlighter>)({
        themes: ['github-dark', 'github-light'],
        langs: [
          'typescript',
          'javascript',
          'python',
          'go',
          'json',
          'bash',
          'sql',
          'html',
          'css',
          'yaml',
          'markdown',
          'jsx',
          'tsx',
          'rust',
          'java',
          'c',
          'cpp',
          'ruby',
          'php',
          'swift',
          'kotlin',
        ],
      })
    )
  }
  return highlighterPromise
}

/**
 * Hook for copying code to clipboard with visual feedback.
 */
function useCodeCopy(code: string) {
  const [copied, setCopied] = useState(false)

  const copyCode = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(code)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch (err) {
      console.error('Failed to copy code:', err)
    }
  }, [code])

  return { copied, copyCode }
}

/**
 * Custom TipTap code block node view with syntax highlighting.
 */
export const TipTapCodeBlockView = memo(function TipTapCodeBlockView({
  node,
  updateAttributes,
  extension,
  editor,
}: NodeViewProps) {
  const [highlightedHtml, setHighlightedHtml] = useState<string | null>(null)
  const resolvedTheme = useResolvedTheme()

  // Track editor focus state using React's useSyncExternalStore for reliable updates
  const isEditorFocused = useSyncExternalStore(
    (callback) => {
      editor.on('focus', callback)
      editor.on('blur', callback)
      return () => {
        editor.off('focus', callback)
        editor.off('blur', callback)
      }
    },
    () => editor.isFocused,
    () => false // SSR fallback
  )

  // Get the code content from the node
  const code = node.textContent || ''
  const language = (node.attrs.language as string) || ''

  const { copied, copyCode } = useCodeCopy(code)

  // Generate line numbers
  const lineNumbers = useMemo(() => generateLineNumbers(code), [code])

  // Determine the language (from attribute or auto-detect)
  const normalizedLang = language
    ? normalizeLanguage(language)
    : detectLanguage(code)

  // Syntax highlight the code
  useEffect(() => {
    let cancelled = false

    async function highlight() {
      if (!code.trim()) {
        setHighlightedHtml(null)
        return
      }

      try {
        const highlighter = await getHighlighter()
        if (cancelled) return

        const langs = highlighter.getLoadedLanguages()
        const langToUse = langs.includes(normalizedLang) ? normalizedLang : 'text'

        const html = highlighter.codeToHtml(code, {
          lang: langToUse,
          theme: resolvedTheme === 'dark' ? 'github-dark' : 'github-light',
        })

        // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
        if (!cancelled) {
          setHighlightedHtml(html)
        }
      } catch (err) {
        console.warn('Syntax highlighting failed:', err)
      }
    }

    void highlight()

    return () => {
      cancelled = true
    }
  }, [code, normalizedLang, resolvedTheme])

  // Handle language selection
  const handleLanguageChange = useCallback(
    (e: React.ChangeEvent<HTMLSelectElement>) => {
      updateAttributes({ language: e.target.value })
    },
    [updateAttributes]
  )

  // Get available languages from extension config
  const extensionOptions = extension.options as { languages?: string[] } | undefined
  const availableLanguages: string[] = extensionOptions?.languages ?? [
    'text',
    'typescript',
    'javascript',
    'python',
    'go',
    'json',
    'bash',
    'sql',
    'html',
    'css',
    'yaml',
    'rust',
    'java',
    'cpp',
    'ruby',
  ]

  return (
    <NodeViewWrapper className="relative group rounded-lg overflow-hidden my-3 not-prose">
      {/* Header with language selector and copy button */}
      <div className="flex items-center justify-between px-3 py-2 bg-card border-b border-border">
        <select
          value={language || normalizedLang}
          onChange={handleLanguageChange}
          className="text-xs text-muted-foreground font-mono bg-transparent border-none outline-none cursor-pointer hover:text-foreground transition-colors"
          contentEditable={false}
        >
          {availableLanguages.map((lang) => (
            <option key={lang} value={lang} className="bg-muted">
              {lang || 'auto'}
            </option>
          ))}
        </select>
        <button
          type="button"
          onClick={() => void copyCode()}
          className="flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
          contentEditable={false}
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

      {/* Code content - layered approach for smooth transitions */}
      <div className="bg-muted/30 overflow-x-auto relative flex">
        {/* Line numbers gutter */}
        <div
          className="flex-shrink-0 py-4 pl-3 pr-2 text-sm font-mono text-muted-foreground select-none border-r border-border/50 text-right"
          contentEditable={false}
          aria-hidden="true"
        >
          {lineNumbers.map((num, i) => (
            <div key={i} className="leading-relaxed">
              {num}
            </div>
          ))}
        </div>

        {/* Code content area */}
        <div className="flex-1 relative min-w-0">
          {/* Highlighted view - shown when not editing */}
          <div
            className={`p-4 text-sm [&>pre]:!bg-transparent [&>pre]:!m-0 [&>pre]:!p-0 [&_code]:!leading-relaxed font-mono transition-opacity duration-150 ${
              isEditorFocused ? 'opacity-0 pointer-events-none' : 'opacity-100'
            }`}
            dangerouslySetInnerHTML={{ __html: highlightedHtml || '' }}
            contentEditable={false}
          />
          {/* Editable content area - always in DOM, visible when editing */}
          <div
            className={`absolute inset-0 transition-opacity duration-150 ${
              isEditorFocused ? 'opacity-100' : 'opacity-0 pointer-events-none'
            }`}
          >
            <NodeViewContent
              as="pre"
              className="p-4 text-sm text-foreground font-mono whitespace-pre overflow-x-auto !bg-transparent !m-0 leading-relaxed"
            />
          </div>
        </div>
      </div>
    </NodeViewWrapper>
  )
})
