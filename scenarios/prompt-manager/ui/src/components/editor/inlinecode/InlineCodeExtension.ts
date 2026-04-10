/**
 * Custom InlineCode extension for TipTap with copy functionality.
 *
 * Extends the default code mark to add a hover-triggered copy button.
 * Uses a floating button approach that doesn't modify the DOM structure
 * to avoid conflicts with ProseMirror's state management.
 */

import Code from '@tiptap/extension-code'
import { copyToClipboard } from '@/lib/clipboard'
import { Plugin, PluginKey } from '@tiptap/pm/state'

const INLINE_CODE_COPY_PLUGIN_KEY = new PluginKey('inlineCodeCopy')

// Create a singleton floating button that gets reused
let floatingButton: HTMLButtonElement | null = null

function getOrCreateFloatingButton(): HTMLButtonElement {
  if (floatingButton && document.body.contains(floatingButton)) {
    return floatingButton
  }

  const button = document.createElement('button')
  button.type = 'button'
  button.className = 'inline-code-copy-btn-floating'
  button.innerHTML = `
    <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <rect width="14" height="14" x="8" y="8" rx="2" ry="2"/>
      <path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/>
    </svg>
  `
  button.title = 'Copy code'
  document.body.appendChild(button)
  floatingButton = button
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

// Copy text to clipboard and show button feedback
async function copyCodeToClipboard(text: string, button: HTMLButtonElement): Promise<void> {
  try {
    await copyToClipboard(text)
    showCopySuccess(button)
  } catch (err) {
    console.error('Failed to copy:', err)
  }
}

/**
 * Create the inline code copy plugin.
 * Uses a floating button positioned via getBoundingClientRect().
 * Does NOT modify the DOM structure to avoid ProseMirror state conflicts.
 */
function createInlineCodeCopyPlugin(): Plugin {
  let currentTarget: HTMLElement | null = null
  let clickHandler: ((e: MouseEvent) => void) | null = null

  const hideButton = () => {
    if (floatingButton) {
      floatingButton.style.display = 'none'
      if (clickHandler) {
        floatingButton.removeEventListener('click', clickHandler)
        clickHandler = null
      }
    }
    currentTarget = null
  }

  const showButton = (target: HTMLElement) => {
    const button = getOrCreateFloatingButton()
    const rect = target.getBoundingClientRect()

    // Position button to the right of the code element
    button.style.display = 'flex'
    button.style.position = 'fixed'
    button.style.top = `${rect.top + rect.height / 2}px`
    button.style.left = `${rect.right + 4}px`
    button.style.transform = 'translateY(-50%)'
    button.style.zIndex = '9999'

    // Remove old click handler and add new one
    if (clickHandler) {
      button.removeEventListener('click', clickHandler)
    }
    clickHandler = (e: MouseEvent) => {
      e.preventDefault()
      e.stopPropagation()
      const code = target.textContent || ''
      void copyCodeToClipboard(code, button)
    }
    button.addEventListener('click', clickHandler)

    currentTarget = target
  }

  return new Plugin({
    key: INLINE_CODE_COPY_PLUGIN_KEY,
    view() {
      return {
        destroy() {
          hideButton()
          // Clean up floating button when editor is destroyed
          if (floatingButton && floatingButton.parentNode) {
            floatingButton.parentNode.removeChild(floatingButton)
            floatingButton = null
          }
        },
      }
    },
    props: {
      handleDOMEvents: {
        mouseover(_view: unknown, event: MouseEvent) {
          const target = event.target as HTMLElement

          // Check if hovering over an inline code element (not inside pre/code block)
          if (target.tagName === 'CODE' && !target.closest('pre')) {
            // Don't reposition if already showing for this target
            if (target === currentTarget) return false

            showButton(target)
          }

          return false
        },
        mouseout(_view: unknown, event: MouseEvent) {
          const target = event.target as HTMLElement
          const relatedTarget = event.relatedTarget as HTMLElement | null

          // Only hide if we're leaving the code element and not going to the button
          if (target === currentTarget) {
            // If moving to the floating button, don't hide
            if (relatedTarget === floatingButton || floatingButton?.contains(relatedTarget)) {
              return false
            }
            hideButton()
          }

          // If leaving the button, check if going back to the code element
          if (target === floatingButton || floatingButton?.contains(target)) {
            if (relatedTarget !== currentTarget && !currentTarget?.contains(relatedTarget)) {
              hideButton()
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
