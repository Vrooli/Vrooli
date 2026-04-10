/**
 * Custom Link extension for TipTap with hover preview functionality.
 *
 * Extends the default link mark to detect hover and trigger preview display.
 */

import Link from '@tiptap/extension-link'
import { Plugin, PluginKey } from '@tiptap/pm/state'

const LINK_PREVIEW_PLUGIN_KEY = new PluginKey('linkPreview')

// Event types for link hover
export interface LinkHoverEvent {
  type: 'hover' | 'leave'
  url: string
  position: { x: number; y: number }
}

export type LinkHoverCallback = (event: LinkHoverEvent) => void

// Global callback for link hover events
let linkHoverCallback: LinkHoverCallback | null = null

/**
 * Set the callback for link hover events.
 */
export function setLinkHoverCallback(callback: LinkHoverCallback | null): void {
  linkHoverCallback = callback
}

/**
 * Create the link preview plugin.
 */
function createLinkPreviewPlugin(): Plugin {
  let currentLink: HTMLAnchorElement | null = null
  let hoverTimeout: ReturnType<typeof setTimeout> | null = null

  const cleanup = () => {
    if (hoverTimeout) {
      clearTimeout(hoverTimeout)
      hoverTimeout = null
    }
    currentLink = null
  }

  return new Plugin({
    key: LINK_PREVIEW_PLUGIN_KEY,
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

          // Check if hovering over a link
          const link = target.closest('a')
          if (!(link instanceof HTMLAnchorElement)) return false

          // Don't trigger for the same link
          if (link === currentLink) return false

          // Clean up previous
          cleanup()
          currentLink = link

          // Get the URL
          const url = link.href
          if (!url || !url.startsWith('http')) return false

          // Debounce the hover event
          hoverTimeout = setTimeout(() => {
            if (linkHoverCallback && currentLink === link) {
              const rect = link.getBoundingClientRect()
              linkHoverCallback({
                type: 'hover',
                url,
                position: {
                  x: rect.left,
                  y: rect.bottom,
                },
              })
            }
          }, 500) // 500ms delay before showing preview

          return false
        },
        mouseout(_view: unknown, event: MouseEvent) {
          const target = event.target as HTMLElement
          const relatedTarget = event.relatedTarget as HTMLElement | null

          // Check if we're leaving a link
          const link = target.closest('a')
          if (link instanceof HTMLAnchorElement && link === currentLink) {
            // Check if moving to the tooltip (don't close if so)
            // The tooltip handles its own mouseout
            if (!(relatedTarget instanceof HTMLElement) || !relatedTarget.closest('.link-preview-tooltip')) {
              cleanup()
              if (linkHoverCallback) {
                linkHoverCallback({
                  type: 'leave',
                  url: '',
                  position: { x: 0, y: 0 },
                })
              }
            }
          }

          return false
        },
      },
    },
  })
}

/**
 * Enhanced Link extension with preview on hover.
 */
export const LinkPreviewExtension = Link.extend({
  addProseMirrorPlugins() {
    const parentPlugins = this.parent?.() ?? []
    return [...parentPlugins, createLinkPreviewPlugin()]
  },
})
