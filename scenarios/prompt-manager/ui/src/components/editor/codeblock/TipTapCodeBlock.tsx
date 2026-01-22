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
import { memo, useCallback, useEffect, useState, useSyncExternalStore } from 'react'
import { Check, Copy } from 'lucide-react'
import { detectLanguage, normalizeLanguage } from './languageDetection'

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
        themes: ['github-dark'],
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
          theme: 'github-dark',
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
  }, [code, normalizedLang])

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
      <div className="flex items-center justify-between px-3 py-2 bg-slate-900 border-b border-slate-700">
        <select
          value={language || normalizedLang}
          onChange={handleLanguageChange}
          className="text-xs text-slate-400 font-mono bg-transparent border-none outline-none cursor-pointer hover:text-slate-200 transition-colors"
          contentEditable={false}
        >
          {availableLanguages.map((lang) => (
            <option key={lang} value={lang} className="bg-slate-800">
              {lang || 'auto'}
            </option>
          ))}
        </select>
        <button
          type="button"
          onClick={() => void copyCode()}
          className="flex items-center gap-1.5 text-xs text-slate-400 hover:text-slate-200 transition-colors"
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

      {/* Code content - show highlighted or editable content */}
      <div className="bg-slate-800 overflow-x-auto">
        {highlightedHtml && !isEditorFocused ? (
          // Show highlighted HTML when not editing
          <div
            className="p-4 text-sm [&>pre]:!bg-transparent [&>pre]:!m-0 [&>pre]:!p-0 font-mono"
            dangerouslySetInnerHTML={{ __html: highlightedHtml }}
            contentEditable={false}
          />
        ) : null}
        {/* Editable content area - always rendered but may be hidden */}
        <div className={highlightedHtml && !isEditorFocused ? 'hidden' : ''}>
          <NodeViewContent
            as="pre"
            className="p-4 text-sm text-slate-200 font-mono whitespace-pre overflow-x-auto !bg-transparent !m-0"
          />
        </div>
      </div>
    </NodeViewWrapper>
  )
})
