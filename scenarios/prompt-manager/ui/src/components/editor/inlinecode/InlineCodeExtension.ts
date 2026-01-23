/**
 * Custom InlineCode extension for TipTap with copy functionality.
 *
 * Extends the default code mark to add a hover-triggered copy button.
 */

import Code from '@tiptap/extension-code'
import { Plugin, PluginKey } from '@tiptap/pm/state'

const INLINE_CODE_COPY_PLUGIN_KEY = new PluginKey('inlineCodeCopy')

// Create the copy button element
function createCopyButton(): HTMLButtonElement {
  const button = document.createElement('button')
  button.type = 'button'
  button.className = 'inline-code-copy-btn'
  button.innerHTML = `
    <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <rect width="14" height="14" x="8" y="8" rx="2" ry="2"/>
      <path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/>
    </svg>
  `
  button.title = 'Copy code'
  return button
}

// Show success feedback
function showCopySuccess(button: HTMLButtonElement): void {
  button.innerHTML = `
    <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-green-400">
      <polyline points="20 6 9 17 4 12"/>
    </svg>
  `
  button.classList.add('copied')
  setTimeout(() => {
    button.innerHTML = `
      <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <rect width="14" height="14" x="8" y="8" rx="2" ry="2"/>
        <path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/>
      </svg>
    `
    button.classList.remove('copied')
  }, 2000)
}

// Copy text to clipboard
async function copyToClipboard(text: string, button: HTMLButtonElement): Promise<void> {
  try {
    await navigator.clipboard.writeText(text)
    showCopySuccess(button)
  } catch (err) {
    console.error('Failed to copy:', err)
  }
}

/**
 * Create the inline code copy plugin.
 */
function createInlineCodeCopyPlugin(): Plugin {
  let currentButton: HTMLButtonElement | null = null
  let currentTarget: HTMLElement | null = null

  const cleanup = () => {
    if (currentButton && currentButton.parentNode) {
      currentButton.parentNode.removeChild(currentButton)
    }
    currentButton = null
    currentTarget = null
  }

  return new Plugin({
    key: INLINE_CODE_COPY_PLUGIN_KEY,
    view() {
      return {
        destroy() {
          cleanup()
        },
      }
    },
    props: {
      handleDOMEvents: {
        mouseover(_view: unknown, event: MouseEvent) {
          const target = event.target as HTMLElement

          // Check if hovering over an inline code element
          if (target.tagName === 'CODE' && !target.closest('pre')) {
            // Don't add another button if already present
            if (target === currentTarget) return false

            // Clean up any existing button
            cleanup()

            // Create and position the copy button
            const button = createCopyButton()
            currentButton = button
            currentTarget = target

            // Position the button relative to the code element
            target.style.position = 'relative'
            target.appendChild(button)

            // Handle click
            button.addEventListener('click', (e) => {
              e.preventDefault()
              e.stopPropagation()
              const code = target.textContent || ''
              void copyToClipboard(code, button)
            })
          }

          return false
        },
        mouseout(_view: unknown, event: MouseEvent) {
          const target = event.target as HTMLElement
          const relatedTarget = event.relatedTarget as HTMLElement | null

          // Check if we're leaving the code element (but not to the button)
          if (target === currentTarget) {
            // If moving to the button, don't clean up
            if (relatedTarget === currentButton || currentButton?.contains(relatedTarget)) {
              return false
            }
            cleanup()
          }

          // If leaving the button, check if going back to code
          if (target === currentButton || currentButton?.contains(target)) {
            if (relatedTarget !== currentTarget && !currentTarget?.contains(relatedTarget)) {
              cleanup()
            }
          }

          return false
        },
      },
    },
  })
}

/**
 * Enhanced InlineCode extension with copy button on hover.
 */
export const InlineCodeExtension = Code.extend({
  addOptions() {
    return {
      // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
      ...this.parent?.(),
      HTMLAttributes: {
        class: 'inline-code-with-copy',
      },
    }
  },

  addProseMirrorPlugins() {
    return [
      ...(this.parent?.() ?? []),
      createInlineCodeCopyPlugin(),
    ]
  },
})
