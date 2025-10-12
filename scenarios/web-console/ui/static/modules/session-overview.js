import { elements, state } from './state.js'
import { getDetachedTabs, getActiveTab } from './tab-manager.js'
import { proxyToApi, formatRelativeTimestamp, formatAbsoluteTime } from './utils.js'

let updateActionsCallback = null
const relativeTimeFormatter = typeof Intl !== 'undefined' && typeof Intl.RelativeTimeFormat === 'function'
  ? new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' })
  : null

export function configureSessionOverview({ updateSessionActions } = {}) {
  updateActionsCallback = typeof updateSessionActions === 'function' ? updateSessionActions : null
}

export function setSessionOverviewLoading(isLoading) {
  const button = elements.sessionOverviewRefresh
  if (button) {
    button.disabled = Boolean(isLoading)
    button.classList.toggle('loading', Boolean(isLoading))
    if (isLoading) {
      button.setAttribute('aria-busy', 'true')
    } else {
      button.removeAttribute('aria-busy')
    }
  }
}

export function queueSessionOverviewRefresh(delay = 0) {
  if (typeof window === 'undefined') return
  if (state.sessions.refreshTimer) {
    clearTimeout(state.sessions.refreshTimer)
    state.sessions.refreshTimer = null
  }
  state.sessions.refreshTimer = window.setTimeout(() => {
    state.sessions.refreshTimer = null
    refreshSessionOverview({ silent: true })
  }, Math.max(0, delay))
}

export async function refreshSessionOverview(options = {}) {
  if (state.sessions.loading) {
    state.sessions.needsRefresh = true
    return
  }

  state.sessions.loading = true
  state.sessions.error = null
  setSessionOverviewLoading(true)
  renderSessionOverview()

  try {
    const response = await proxyToApi('/api/v1/sessions')
    if (!response.ok) {
      const text = await response.text()
      throw new Error(text || `API error (${response.status})`)
    }
    let data
    try {
      data = await response.json()
    } catch (_error) {
      data = []
    }

    if (Array.isArray(data)) {
      state.sessions.items = data
    } else if (data && typeof data === 'object') {
      const list = Array.isArray(data.sessions) ? data.sessions : []
      state.sessions.items = list
      if (typeof data.capacity === 'number' && Number.isFinite(data.capacity) && data.capacity > 0) {
        state.sessions.capacity = data.capacity
      }
    } else {
      state.sessions.items = []
    }
    state.sessions.lastFetched = Date.now()
    let capacityHeader = response.headers.get('x-session-capacity')
    if (!capacityHeader) {
      capacityHeader = response.headers.get('X-Session-Capacity')
    }
    if (capacityHeader) {
      const parsedCapacity = parseInt(capacityHeader, 10)
      if (Number.isFinite(parsedCapacity) && parsedCapacity > 0) {
        state.sessions.capacity = parsedCapacity
      }
    }
  } catch (error) {
    state.sessions.error = error instanceof Error ? error : new Error(String(error))
    if (options.silent !== true) {
      console.error('Failed to refresh session overview:', error)
    }
  } finally {
    state.sessions.loading = false
    setSessionOverviewLoading(false)
    renderSessionOverview()
    if (state.sessions.needsRefresh) {
      state.sessions.needsRefresh = false
      queueSessionOverviewRefresh(50)
    }
  }
}

export function startSessionOverviewWatcher() {
  if (typeof window === 'undefined') return
  if (state.sessions.pollHandle) {
    clearInterval(state.sessions.pollHandle)
    state.sessions.pollHandle = null
  }
  refreshSessionOverview({ silent: true })
  state.sessions.pollHandle = window.setInterval(() => {
    if (typeof document !== 'undefined' && document.hidden) {
      return
    }
    refreshSessionOverview({ silent: true })
  }, 20000)
}

export function stopSessionOverviewWatcher() {
  if (state.sessions.pollHandle) {
    clearInterval(state.sessions.pollHandle)
    state.sessions.pollHandle = null
  }
  if (state.sessions.refreshTimer) {
    clearTimeout(state.sessions.refreshTimer)
    state.sessions.refreshTimer = null
  }
  state.sessions.needsRefresh = false
}

if (typeof window !== 'undefined') {
  window.addEventListener('beforeunload', () => {
    stopSessionOverviewWatcher()
  })
}

export function renderSessionOverview() {
  const listEl = elements.sessionOverviewList
  if (!listEl) return

  const apiSessions = Array.isArray(state.sessions.items) ? state.sessions.items.slice() : []
  const sessionMap = new Map()
  apiSessions.forEach((session) => {
    if (!session || typeof session.id !== 'string') return
    const id = session.id.trim()
    if (!id) return
    sessionMap.set(id, { ...session })
  })

  state.tabs.forEach((tab) => {
    if (!tab || !tab.session) return
    const tabSession = tab.session
    const id = typeof tabSession.id === 'string' ? tabSession.id.trim() : ''
    if (!id || sessionMap.has(id)) return
    const args = Array.isArray(tabSession.args) ? [...tabSession.args] : []
    sessionMap.set(id, {
      id,
      createdAt: tabSession.createdAt || null,
      expiresAt: tabSession.expiresAt || null,
      lastActivity: tabSession.lastActivity || tabSession.updatedAt || tabSession.createdAt || null,
      state: tab.phase === 'closed' ? 'closed' : 'active',
      command: tabSession.command || '',
      args
    })
  })

  const sessions = Array.from(sessionMap.values())
  const tabBySessionId = new Map()
  state.tabs.forEach((tab) => {
    if (tab.session && tab.session.id) {
      tabBySessionId.set(tab.session.id, tab)
    }
  })
  const detachedTabs = getDetachedTabs()

  const countElement = elements.sessionOverviewCount
  const capacityValue = typeof state.sessions.capacity === 'number' && Number.isFinite(state.sessions.capacity) && state.sessions.capacity > 0
    ? state.sessions.capacity
    : null

  if (countElement) {
    const base = state.sessions.loading
      ? '…'
      : state.sessions.error
        ? '!'
        : String(sessions.length)
    const hasCapacity = capacityValue !== null
    const displayCapacity = hasCapacity ? capacityValue : '—'
    const countText = `${base} of ${displayCapacity}`
    countElement.textContent = countText

    let label
    if (state.sessions.loading) {
      label = hasCapacity
        ? `Loading active sessions (capacity ${capacityValue})`
        : 'Loading active sessions (capacity unknown)'
    } else if (state.sessions.error) {
      label = hasCapacity
        ? `Unable to load active sessions (capacity ${capacityValue})`
        : 'Unable to load active sessions (capacity unknown)'
    } else {
      label = hasCapacity
        ? `${sessions.length} of ${capacityValue} sessions active`
        : `${sessions.length} sessions active (capacity unknown)`
    }
    countElement.setAttribute('aria-label', label)
    countElement.title = label
  }

  const preserveExistingList = (state.sessions.loading || state.sessions.error) && listEl.childElementCount > 0
  if (!preserveExistingList) {
    listEl.innerHTML = ''
  }

  const activeSessionId = getActiveTab()?.session?.id || null
  let orphanCount = 0

  if (!preserveExistingList) {
    const fragment = document.createDocumentFragment()

    sessions.sort((a, b) => {
      const aTime = a && a.lastActivity ? Date.parse(a.lastActivity) : 0
      const bTime = b && b.lastActivity ? Date.parse(b.lastActivity) : 0
      return bTime - aTime
    })

    sessions.forEach((session) => {
      if (!session || !session.id) return
      const li = document.createElement('li')
      li.className = 'session-overview-item'

      const header = document.createElement('div')
      header.className = 'session-overview-row'

      const sessionId = typeof session.id === 'string' ? session.id : ''
      if (!sessionId) {
        return
      }

      const idSpan = document.createElement('span')
      idSpan.className = 'session-overview-id'
      idSpan.textContent = sessionId.slice(0, 8)
      idSpan.title = sessionId
      header.appendChild(idSpan)

      const statusSpan = document.createElement('span')
      statusSpan.className = 'session-overview-status'

      const attachedTab = tabBySessionId.get(sessionId)
      if (attachedTab) {
        li.dataset.sessionState = 'attached'
        const tabLabel = attachedTab.label || attachedTab.defaultLabel || attachedTab.id
        statusSpan.textContent = attachedTab.id === state.activeTabId ? 'Active tab' : `Attached (${tabLabel})`
      } else {
        li.dataset.sessionState = 'orphaned'
        statusSpan.textContent = 'No tab attached'
        orphanCount += 1
      }

      header.appendChild(statusSpan)
      li.appendChild(header)

      if (session.command) {
        const commandLine = document.createElement('div')
        commandLine.className = 'session-overview-command'
        const commandParts = [session.command]
        if (Array.isArray(session.args) && session.args.length > 0) {
          commandParts.push(...session.args)
        }
        commandLine.textContent = commandParts.join(' ')
        li.appendChild(commandLine)
      }

      if (session.lastActivity) {
        const timestampLine = document.createElement('div')
        timestampLine.className = 'session-overview-timestamp'
        const relative = formatRelativeTimestamp(session.lastActivity, relativeTimeFormatter)
        const absolute = formatAbsoluteTime(session.lastActivity)
        if (relative && absolute) {
          timestampLine.textContent = `Last active ${relative} (${absolute})`
        } else if (relative) {
          timestampLine.textContent = `Last active ${relative}`
        } else if (absolute) {
          timestampLine.textContent = `Last active ${absolute}`
        }
        if (timestampLine.textContent) {
          li.appendChild(timestampLine)
        }
      }

      if (sessionId === activeSessionId) {
        li.dataset.active = 'true'
      }

      fragment.appendChild(li)
    })

    listEl.appendChild(fragment)
  } else {
    sessions.forEach((session) => {
      if (!session || !session.id) return
      if (!tabBySessionId.get(session.id)) {
        orphanCount += 1
      }
    })
  }

  const emptyEl = elements.sessionOverviewEmpty
  if (emptyEl) {
    if (!state.sessions.loading && !state.sessions.error && sessions.length === 0) {
      emptyEl.classList.remove('hidden')
    } else {
      emptyEl.classList.add('hidden')
    }
  }

  const alertEl = elements.sessionOverviewAlert
  if (alertEl) {
    const messageSpan = alertEl.querySelector('span')
    if (messageSpan) {
      if (orphanCount <= 0) {
        messageSpan.textContent = 'Unmapped sessions detected. Use “Sign out all sessions” to clean up.'
      } else if (orphanCount === 1) {
        messageSpan.textContent = '1 session is not mapped to a tab. Use “Sign out all sessions” to clean up.'
      } else {
        messageSpan.textContent = `${orphanCount} sessions are not mapped to tabs. Use “Sign out all sessions” to clean up.`
      }
    }
    if (!state.sessions.loading && !state.sessions.error && orphanCount > 0) {
      alertEl.classList.remove('hidden')
    } else {
      alertEl.classList.add('hidden')
    }
  }

  const detachedMetaEl = elements.detachedTabsMeta
  if (detachedMetaEl) {
    if (detachedTabs.length > 0) {
      const labels = detachedTabs.map((tab) => tab.label || tab.defaultLabel || tab.id)
      const joined = labels.slice(0, 3).join(', ')
      const remaining = labels.length > 3 ? `, +${labels.length - 3} more` : ''
      const label = detachedTabs.length === 1
        ? `1 tab is open without an active session (${labels[0]}).`
        : `${detachedTabs.length} tabs are open without active sessions (${joined}${remaining}).`
      detachedMetaEl.textContent = `${label} Use “Close detached tabs” to clean up.`
      detachedMetaEl.classList.remove('hidden')
    } else {
      detachedMetaEl.textContent = ''
      detachedMetaEl.classList.add('hidden')
    }
  }

  const metaEl = elements.sessionOverviewMeta
  if (metaEl) {
    metaEl.classList.toggle('error', Boolean(state.sessions.error))
    const capacityText = capacityValue ? `Capacity ${capacityValue}` : null
    if (state.sessions.error) {
      const message = state.sessions.error instanceof Error ? state.sessions.error.message : String(state.sessions.error)
      const segments = [`Unable to load sessions: ${message}`]
      segments.push(capacityText || 'Capacity unknown')
      metaEl.textContent = segments.join(' • ')
    } else if (state.sessions.loading) {
      metaEl.textContent = `Loading sessions… • ${capacityText || 'Capacity unknown'}`
    } else if (state.sessions.lastFetched) {
      const relative = formatRelativeTimestamp(state.sessions.lastFetched, relativeTimeFormatter)
      const absolute = formatAbsoluteTime(state.sessions.lastFetched)
      const segments = []
      if (relative && absolute) {
        segments.push(`Updated ${relative} (${absolute})`)
      } else if (relative) {
        segments.push(`Updated ${relative}`)
      } else if (absolute) {
        segments.push(`Updated ${absolute}`)
      }
      segments.push(capacityText || 'Capacity unknown')
      metaEl.textContent = segments.join(' • ')
    } else {
      metaEl.textContent = capacityText || 'Capacity unknown'
    }
    metaEl.classList.toggle('hidden', metaEl.textContent === '')
  }

  updateActionsCallback?.()
}
