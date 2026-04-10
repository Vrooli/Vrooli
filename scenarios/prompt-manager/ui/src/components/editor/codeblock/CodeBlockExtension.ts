/**
 * Custom CodeBlock extension for TipTap with enhanced features.
 *
 * Extends the default code block with:
 * - Language attribute support
 * - Custom NodeView rendering
 */

import CodeBlockBase from '@tiptap/extension-code-block'
import { ReactNodeViewRenderer } from '@tiptap/react'
import { TipTapCodeBlockView } from './TipTapCodeBlock'

export interface CodeBlockOptions {
  /**
   * Available languages for the language selector.
   */
  languages: string[]
  /**
   * HTML attributes for the code block element.
   */
  HTMLAttributes: Record<string, unknown>
}

/**
 * Enhanced CodeBlock extension with syntax highlighting and UI features.
 */
export const CodeBlockExtension = CodeBlockBase.extend<CodeBlockOptions>({
  addOptions() {
    // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
    const parentOptions = this.parent ? this.parent() : {}
    return {
      ...parentOptions,
      languages: [
        '',
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
        'rust',
        'java',
        'c',
        'cpp',
        'ruby',
        'php',
      ],
      HTMLAttributes: {},
    }
  },

  addAttributes() {
    const parentAttrs = this.parent ? this.parent() : {}
    return {
      ...parentAttrs,
      language: {
        default: '',
        parseHTML: (element: HTMLElement) => {
          // Try to get language from class (e.g., "language-typescript")
          // Check both <pre> element and its <code> child (marked puts class on <code>)
          const preClassName = element.className || ''
          const preMatch = preClassName.match(/language-(\w+)/)
          if (preMatch) return preMatch[1]

          // Check the <code> child element (this is where marked puts the language class)
          const codeChild = element.firstElementChild
          if (codeChild) {
            const codeClassName = codeChild.className || ''
            const codeMatch = codeClassName.match(/language-(\w+)/)
            if (codeMatch) return codeMatch[1]
          }

          return ''
        },
        renderHTML: (attributes: Record<string, unknown>) => {
          const lang = attributes.language
          if (!lang || typeof lang !== 'string') {
            return {}
          }
          return {
            class: `language-${lang}`,
          }
        },
      },
      fenceCount: {
        default: 3,
        parseHTML: (element: HTMLElement) => {
          // Get fence count from data attribute (set by MarkedParser for 4+ backtick fences)
          const count = element.getAttribute('data-fence-count')
          return count ? parseInt(count, 10) : 3
        },
        renderHTML: (attributes: Record<string, unknown>) => {
          const count = attributes.fenceCount
          // Only output attribute for extended fences (4+)
          if (!count || count === 3) {
            return {}
          }
          return {
            'data-fence-count': typeof count === 'number' ? String(count) : '3',
          }
        },
      },
    }
  },

  addNodeView() {
    return ReactNodeViewRenderer(TipTapCodeBlockView)
  },
})
