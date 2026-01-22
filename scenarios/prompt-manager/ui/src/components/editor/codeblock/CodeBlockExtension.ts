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
          const className = element.className || ''
          const classMatch = className.match(/language-(\w+)/)
          return classMatch ? classMatch[1] : ''
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
    }
  },

  addNodeView() {
    return ReactNodeViewRenderer(TipTapCodeBlockView)
  },
})
