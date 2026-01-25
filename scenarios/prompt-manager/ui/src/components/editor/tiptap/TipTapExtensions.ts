/**
 * TipTapExtensions - Centralized TipTap extension configuration.
 *
 * Configures all TipTap extensions with consistent settings:
 * - StarterKit (basic formatting, headings, lists, etc.)
 * - Custom code block with syntax highlighting
 * - Custom inline code with copy button
 * - Link with hover preview
 * - Highlight for ==text== syntax
 * - Placeholder text
 * - Typography improvements
 */

import type { AnyExtension } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Placeholder from '@tiptap/extension-placeholder'
import Highlight from '@tiptap/extension-highlight'
import Typography from '@tiptap/extension-typography'
import { CodeBlockExtension } from '../codeblock'
import { InlineCodeExtension } from '../inlinecode'
import { LinkPreviewExtension } from '../linkpreview'

export interface TipTapExtensionsOptions {
  /** Placeholder text shown when editor is empty */
  placeholder?: string
  /** Available languages for code blocks */
  codeBlockLanguages?: string[]
  /** CSS classes for inline code elements */
  inlineCodeClass?: string
  /** CSS classes for link elements */
  linkClass?: string
  /** CSS classes for highlight elements */
  highlightClass?: string
}

const DEFAULT_OPTIONS: Required<TipTapExtensionsOptions> = {
  placeholder: 'Start writing...',
  codeBlockLanguages: [
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
    'rust',
    'java',
    'cpp',
    'ruby',
  ],
  inlineCodeClass: 'bg-muted rounded px-1.5 py-0.5 font-mono text-sm text-primary',
  linkClass: 'text-primary underline hover:text-primary/80 cursor-pointer',
  highlightClass: 'bg-yellow-500/30 rounded px-0.5',
}

/**
 * Create the array of TipTap extensions with consistent configuration.
 *
 * @param options - Configuration options
 * @returns Array of configured TipTap extensions
 */
export function createTipTapExtensions(
  options: TipTapExtensionsOptions = {}
): AnyExtension[] {
  const config = { ...DEFAULT_OPTIONS, ...options }

  // Note: We cast the extensions to Extension[] to satisfy TypeScript
  // The actual extensions are compatible but have more specific types
  return [
    // StarterKit provides basic formatting, headings, lists, blockquotes, etc.
    StarterKit.configure({
      heading: {
        levels: [1, 2, 3],
      },
      // Disable default codeBlock - we use our custom extension
      codeBlock: false,
      // Disable default code mark - we use our custom extension with copy button
      code: false,
    }),

    // Custom inline code with copy button on hover
    InlineCodeExtension.configure({
      HTMLAttributes: {
        class: config.inlineCodeClass,
      },
    }),

    // Custom code block with syntax highlighting and copy button
    CodeBlockExtension.configure({
      languages: config.codeBlockLanguages,
    }),

    // Placeholder text for empty editor
    Placeholder.configure({
      placeholder: config.placeholder,
      emptyEditorClass: 'is-editor-empty',
    }),

    // Highlight for ==text== syntax
    Highlight.configure({
      HTMLAttributes: {
        class: config.highlightClass,
      },
    }),

    // Typography improvements (smart quotes, ellipsis, etc.)
    Typography,

    // Link with hover preview
    LinkPreviewExtension.configure({
      openOnClick: false,
      HTMLAttributes: {
        class: config.linkClass,
      },
    }),
  ] as AnyExtension[]
}

/**
 * Get the default editor prose classes for consistent styling.
 *
 * @returns CSS class string for the editor content
 */
export function getEditorProseClasses(): string {
  return [
    'prose dark:prose-invert prose-sm max-w-none',
    'focus:outline-none min-h-[200px] p-4',
    // Heading styles
    'prose-h1:text-2xl prose-h1:font-bold prose-h1:text-foreground prose-h1:mt-6 prose-h1:mb-4',
    'prose-h2:text-xl prose-h2:font-bold prose-h2:text-foreground prose-h2:mt-5 prose-h2:mb-3',
    'prose-h3:text-lg prose-h3:font-semibold prose-h3:text-foreground prose-h3:mt-4 prose-h3:mb-2',
    // Paragraph and text styles
    'prose-p:text-muted-foreground prose-p:leading-relaxed',
    'prose-a:text-primary prose-a:no-underline hover:prose-a:underline',
    'prose-strong:text-foreground prose-em:text-foreground/90',
    'prose-code:text-primary prose-code:bg-muted',
    'prose-pre:bg-muted prose-pre:rounded-lg',
    'prose-blockquote:border-primary prose-blockquote:text-muted-foreground',
    // List styles
    'prose-ul:text-muted-foreground prose-ol:text-muted-foreground',
    'prose-li:text-muted-foreground',
  ].join(' ')
}
