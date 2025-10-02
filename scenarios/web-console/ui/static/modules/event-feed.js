/**
 * Event feed management - logging and display
 */

import { elements, AGGREGATED_EVENT_TYPES, SUPPRESSED_EVENT_LABELS, state } from './state.js'
import { formatEventPayload } from './utils.js'

/**
 * Append event to tab's event log
 */
export function appendEvent(tab, type, payload) {
  if (!tab) return
  const normalized = payload instanceof Error ? { message: payload.message, stack: payload.stack } : payload
  if (AGGREGATED_EVENT_TYPES.has(type)) {
    recordSuppressedEvent(tab, type)
    return
  }
  tab.events.push({ type, payload: normalized, timestamp: Date.now() })
  if (tab.events.length > 500) {
    tab.events.splice(0, tab.events.length - 500)
  }
  if (!state.drawer.open && tab.id === state.activeTabId) {
    const nextCount = (state.drawer.unreadCount || 0) + 1
    state.drawer.unreadCount = nextCount > 999 ? 999 : nextCount
  }
}

/**
 * Record suppressed event
 */
export function recordSuppressedEvent(tab, type) {
  if (!tab) return
  const current = (tab.suppressed[type] || 0) + 1
  tab.suppressed[type] = current
  if (tab.id === state.activeTabId && (current === 1 || current % 25 === 0)) {
    renderEventMeta(tab)
  }
}

/**
 * Record transcript entry
 */
export function recordTranscript(tab, entry) {
  if (!tab) return
  tab.transcript.push(entry)
  if (tab.transcript.length > 5000) {
    tab.transcript.splice(0, tab.transcript.length - 5000)
  }
}

/**
 * Show error message
 */
export function showError(tab, message) {
  if (!tab) return
  tab.errorMessage = message || ''
}

/**
 * Render events in sidebar
 */
export function renderEvents(tab) {
  if (!elements.eventFeed) return
  elements.eventFeed.innerHTML = ''
  if (!tab) return
  const recent = tab.events.slice(-50).reverse()
  recent.forEach((event) => {
    const li = document.createElement('li')
    const time = document.createElement('time')
    time.textContent = new Date(event.timestamp).toLocaleTimeString()
    const label = document.createElement('div')
    label.textContent = event.type
    label.style.fontWeight = '600'
    li.appendChild(time)
    li.appendChild(label)
    if (event.payload !== undefined) {
      const detail = document.createElement('pre')
      detail.className = 'event-detail'
      detail.textContent = formatEventPayload(event.payload)
      li.appendChild(detail)
    }
    elements.eventFeed.appendChild(li)
  })
}

/**
 * Render event metadata (suppressed events)
 */
export function renderEventMeta(tab) {
  if (!elements.eventMeta) return
  if (!tab) {
    elements.eventMeta.textContent = ''
    elements.eventMeta.classList.add('hidden')
    return
  }
  const summaries = Object.entries(tab.suppressed)
    .filter(([, count]) => count > 0)
    .map(([type, count]) => `${SUPPRESSED_EVENT_LABELS[type] || type}: ${count}`)

  if (summaries.length === 0) {
    elements.eventMeta.textContent = ''
    elements.eventMeta.classList.add('hidden')
  } else {
    elements.eventMeta.textContent = summaries.join(' • ')
    elements.eventMeta.classList.remove('hidden')
  }
}

/**
 * Render error banner
 */
export function renderError(tab) {
  if (!elements.errorBanner) return
  const message = tab ? tab.errorMessage : ''
  if (message) {
    elements.errorBanner.textContent = message
    elements.errorBanner.classList.remove('hidden')
  } else {
    elements.errorBanner.textContent = ''
    elements.errorBanner.classList.add('hidden')
  }
}
