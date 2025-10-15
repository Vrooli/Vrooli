import {
  elements,
  state,
  TAB_COLOR_DEFAULT,
  TAB_COLOR_MAP,
  TAB_COLOR_OPTIONS,
  TAB_LONG_PRESS_DELAY,
  terminalDefaults,
  initialSuppressedState,
  shortcutButtons,
  nextTabSequence
} from './state.js'
import { closeDrawer, isDrawerFloating } from './drawer.js'

const tabColorButtons = new Map()
let terminalHandlers = {
  onResize: null,
  onData: null
}
const RANDOM_LABEL_ALPHABET = 'abcdefghijklmnopqrstuvwxyz'
const RANDOM_LABEL_LENGTH = 3
const RANDOM_LABEL_ATTEMPTS = 10
const TAB_DEFAULT_LABEL_PREFIX = 'Term'
const TAB_SESSION_STATE_META = {
  attached: {
    icon: null,
    label: ''
  },
  starting: {
    icon: 'loader-2',
    label: 'Session starting',
    extraClass: 'tab-status-loading'
  },
  closing: {
    icon: 'power',
    label: 'Stopping session',
    extraClass: 'tab-status-closing'
  },
  detached: {
    icon: 'unlink',
    label: 'Detached from session',
    extraClass: 'tab-status-detached'
  },
  none: {
    icon: 'minus',
    label: 'No session attached',
    extraClass: 'tab-status-empty'
  }
}
let callbacks = {
  onActiveTabChanged: null,
  onTabCloseRequested: null,
  onTabCreated: null,
  onTabMetadataChanged: null,
  onTabCustomizationOpened: null,
  onTabCustomizationClosed: null,
  onShortcut: null
}
let tabMenuCloseTimer = null

function resolveCrypto() {
  const candidate = typeof globalThis !== 'undefined' ? globalThis.crypto : null
  if (candidate && typeof candidate.getRandomValues === 'function') {
    return candidate
  }
  if (typeof window !== 'undefined' && window.crypto && typeof window.crypto.getRandomValues === 'function') {
    return window.crypto
  }
  return null
}

function generateRandomLetters(length = RANDOM_LABEL_LENGTH) {
  const alphabetLength = RANDOM_LABEL_ALPHABET.length
  const buffer = []
  const cryptoObj = resolveCrypto()
  if (cryptoObj) {
    const values = new Uint8Array(length)
    cryptoObj.getRandomValues(values)
    for (let index = 0; index < length; index += 1) {
      buffer.push(RANDOM_LABEL_ALPHABET[values[index] % alphabetLength])
    }
  } else {
    for (let index = 0; index < length; index += 1) {
      const randomIndex = Math.floor(Math.random() * alphabetLength)
      buffer.push(RANDOM_LABEL_ALPHABET[randomIndex])
    }
  }
  return buffer.join('')
}

function generateDefaultTabLabel(existingLabels = new Set()) {
  const labels = existingLabels instanceof Set ? existingLabels : new Set()
  for (let attempt = 0; attempt < RANDOM_LABEL_ATTEMPTS; attempt += 1) {
    const candidate = `${TAB_DEFAULT_LABEL_PREFIX} ${generateRandomLetters()}`
    if (!labels.has(candidate)) {
      return candidate
    }
  }
  return `${TAB_DEFAULT_LABEL_PREFIX} ${generateRandomLetters()}`
}

export function configureTabManager(options = {}) {
  callbacks = {
    ...callbacks,
    ...Object.fromEntries(
      Object.entries(options).map(([key, value]) => [key, typeof value === 'function' ? value : null])
    )
  }
}

export function setTerminalEventHandlers({ onResize, onData } = {}) {
  terminalHandlers = {
    onResize: typeof onResize === 'function' ? onResize : null,
    onData: typeof onData === 'function' ? onData : null
  }
}

export function initializeShortcutButtons() {
  shortcutButtons.forEach((button) => {
    button.addEventListener('click', () => {
      const command = (button.dataset.command || '').trim()
      const id = button.dataset.shortcutId || 'unknown'
      if (callbacks.onShortcut) {
        callbacks.onShortcut({ command, id })
      }
      if (command && state.drawer && state.drawer.open && isDrawerFloating()) {
        closeDrawer()
      }
    })
  })
}

export function initializeTabCustomizationUI() {
  const container = elements.tabContextColors
  if (!container) return
  container.innerHTML = ''
  tabColorButtons.clear()

  TAB_COLOR_OPTIONS.forEach((option) => {
    const button = document.createElement('button')
    button.type = 'button'
    button.className = 'tab-color-option'
    button.dataset.colorId = option.id
    button.style.setProperty('--swatch-color', option.swatch)
    button.setAttribute('role', 'option')
    button.setAttribute('aria-label', `${option.label} tab color`)
    button.addEventListener('click', () => {
      selectTabColor(option.id)
    })
    tabColorButtons.set(option.id, button)
    container.appendChild(button)
  })

  if (elements.tabContextForm) {
    elements.tabContextForm.addEventListener('submit', (event) => {
      event.preventDefault()
      handleTabContextSubmit()
    })
  }
  if (elements.tabContextCancel) {
    elements.tabContextCancel.addEventListener('click', () => closeTabCustomization())
  }
  if (elements.tabContextBackdrop) {
    elements.tabContextBackdrop.addEventListener('click', () => closeTabCustomization())
  }
  if (elements.tabContextReset) {
    elements.tabContextReset.addEventListener('click', handleTabContextReset)
  }

  selectTabColor(TAB_COLOR_DEFAULT)
}

export function createTerminalTab({ focus = false, id = null, label = null, colorId = TAB_COLOR_DEFAULT } = {}) {
  if (!elements.terminalHost) return null
  if (!id) {
    id = `tab-${Date.now()}-${nextTabSequence()}`
  }
  if (!label) {
    const existingLabels = new Set(state.tabs.map((entry) => entry.label))
    label = generateDefaultTabLabel(existingLabels)
  }

  const container = document.createElement('div')
  container.className = 'terminal-screen'
  container.dataset.tabId = id

  elements.terminalHost.appendChild(container)

  const term = new window.Terminal({ ...terminalDefaults })
  if (typeof term.setOption === 'function') {
    try {
      term.setOption('rendererType', terminalDefaults.rendererType || 'canvas')
      if (Number.isFinite(terminalDefaults.scrollback)) {
        term.setOption('scrollback', terminalDefaults.scrollback)
      }
    } catch (error) {
      console.warn('Unable to apply terminal renderer/scrollback options:', error)
    }
  }
  const fitAddon = new window.FitAddon.FitAddon()
  term.loadAddon(fitAddon)
  term.open(container)
  term.write('\r')
  fitAddon.fit()

  const tab = {
    id,
    label,
    defaultLabel: label,
    colorId: colorId || TAB_COLOR_DEFAULT,
    term,
    fitAddon,
    container,
    phase: 'idle',
    socketState: 'disconnected',
    session: null,
    socket: null,
    reconnecting: false,
    hasEverConnected: false,
    hasReceivedLiveOutput: false,
    heartbeatInterval: null,
    inputSeq: 0,
    inputBatch: null,
    inputBatchScheduled: false,
    replayPending: false,
    replayComplete: true,
    lastReplayCount: 0,
    lastReplayTruncated: false,
    transcript: [],
    transcriptByteSize: 0,
    events: [],
    suppressed: initialSuppressedState(),
    pendingWrites: [],
    localEchoBuffer: [],
    errorMessage: '',
    lastSentSize: { cols: 0, rows: 0 },
    telemetry: {
      typed: 0,
      queued: 0,
      sent: 0,
      batches: 0,
      lastBatchSize: 0
    },
    domItem: null,
    domButton: null,
    domClose: null,
    domLabel: null,
    domStatus: null,
    wasDetached: false
  }

  term.onResize(({ cols, rows }) => {
    if (terminalHandlers.onResize) {
      terminalHandlers.onResize(tab, cols, rows)
    }
  })

  term.onData((data) => {
    if (terminalHandlers.onData) {
      terminalHandlers.onData(tab, data)
    }
  })

  container.addEventListener('mousedown', () => {
    setActiveTab(tab.id)
  })

  state.tabs.push(tab)
  renderTabs()

  if (focus || state.activeTabId === null) {
    setActiveTab(tab.id)
  } else if (tab.container) {
    tab.container.classList.remove('active')
  }

  if (callbacks.onTabCreated) {
    callbacks.onTabCreated(tab)
  }

  return tab
}

export function destroyTerminalTab(tab) {
  if (!tab) return
  if (state.tabMenu.open && state.tabMenu.tabId === tab.id) {
    closeTabCustomization()
  }
  if (tab.heartbeatInterval) {
    clearInterval(tab.heartbeatInterval)
    tab.heartbeatInterval = null
  }
  tab.inputBatchScheduled = false
  tab.inputBatch = null
  tab.localEchoBuffer = []
  try {
    tab.term?.dispose()
  } catch (_error) {
    // ignore
  }
  try {
    tab.socket?.close()
  } catch (_error) {
    // ignore
  }
  if (tab.container?.parentElement) {
    tab.container.parentElement.removeChild(tab.container)
  }
  if (tab.domItem?.parentElement) {
    tab.domItem.parentElement.removeChild(tab.domItem)
  }
  tab.domItem = null
  tab.domButton = null
  tab.domClose = null
  tab.domLabel = null
  tab.domStatus = null
}

export function getActiveTab() {
  if (!state.activeTabId) return null
  return state.tabs.find((tab) => tab.id === state.activeTabId) || null
}

export function findTab(tabId) {
  return state.tabs.find((tab) => tab.id === tabId) || null
}

export function setActiveTab(tabId) {
  const tab = findTab(tabId)
  if (!tab) {
    return
  }
  const previousActiveId = state.activeTabId
  state.activeTabId = tabId
  state.tabs.forEach((entry) => {
    if (!entry.container) return
    if (entry.id === tabId) {
      entry.container.classList.add('active')
    } else {
      entry.container.classList.remove('active')
    }
    updateTabButtonState(entry)
  })
  focusTerminal(tab)
  if (callbacks.onActiveTabChanged) {
    callbacks.onActiveTabChanged(tab, previousActiveId)
  }
}

function focusTerminal(tab) {
  if (!tab) return
  requestAnimationFrame(() => {
    tab.term.focus()
    tab.fitAddon?.fit()
    tab.term.write('')
    try {
      tab.term?.refresh(0, tab.term.rows - 1)
    } catch (error) {
      console.warn('Failed to refresh terminal on focus:', error)
    }
  })
}

export function renderTabs() {
  if (!elements.tabList) return

  const knownIds = new Set(state.tabs.map((tab) => tab.id))

  Array.from(elements.tabList.children).forEach((child) => {
    const tabId = child instanceof HTMLElement ? child.dataset.tabId : undefined
    if (tabId && !knownIds.has(tabId)) {
      elements.tabList.removeChild(child)
    }
  })

  state.tabs.forEach((tab) => {
    if (!tab.domItem) {
      const item = document.createElement('div')
      item.className = 'tab-item'
      item.dataset.tabId = tab.id

      const selectBtn = document.createElement('button')
      selectBtn.type = 'button'
      selectBtn.className = 'tab-button'
      selectBtn.dataset.tabId = tab.id
      selectBtn.setAttribute('role', 'tab')
      selectBtn.title = tab.label
      const labelSpan = document.createElement('span')
      labelSpan.className = 'tab-label'
      labelSpan.textContent = tab.label
      selectBtn.appendChild(labelSpan)

      const statusSpan = document.createElement('span')
      statusSpan.className = 'tab-status-icon hidden'
      statusSpan.setAttribute('aria-hidden', 'true')
      selectBtn.appendChild(statusSpan)
      selectBtn.addEventListener('click', () => setActiveTab(tab.id))
      registerTabCustomizationHandlers(selectBtn, tab)

      const closeBtn = document.createElement('button')
      closeBtn.type = 'button'
      closeBtn.className = 'tab-close'
      closeBtn.dataset.tabId = tab.id
      closeBtn.setAttribute('aria-label', `Close ${tab.label}`)
      closeBtn.textContent = '×'
      closeBtn.disabled = false
      closeBtn.addEventListener('click', (event) => {
        event.stopPropagation()
        if (callbacks.onTabCloseRequested) {
          callbacks.onTabCloseRequested(tab)
        }
      })

      item.appendChild(selectBtn)
      item.appendChild(closeBtn)

      const insertionPoint = elements.tabAddSlot || elements.addTabBtn || null
      if (insertionPoint && elements.tabList.contains(insertionPoint)) {
        elements.tabList.insertBefore(item, insertionPoint)
      } else {
        elements.tabList.appendChild(item)
      }

      tab.domItem = item
      tab.domButton = selectBtn
      tab.domClose = closeBtn
      tab.domLabel = labelSpan
      tab.domStatus = statusSpan
    } else {
      const insertionPoint = elements.tabAddSlot || elements.addTabBtn || null
      if (insertionPoint && elements.tabList.contains(insertionPoint)) {
        elements.tabList.insertBefore(tab.domItem, insertionPoint)
      } else {
        elements.tabList.appendChild(tab.domItem)
      }
    }

    if (!tab.domStatus && tab.domButton) {
      tab.domStatus = tab.domButton.querySelector('.tab-status-icon')
    }

    if (tab.domLabel) {
      tab.domLabel.textContent = tab.label
    } else if (tab.domButton) {
      tab.domButton.textContent = tab.label
    }
    if (tab.domButton) {
      tab.domButton.title = tab.label
    }
    if (tab.domClose) {
      tab.domClose.setAttribute('aria-label', `Close ${tab.label}`)
      tab.domClose.disabled = false
      tab.domClose.removeAttribute('disabled')
    }
    applyTabAppearance(tab)
    updateTabButtonState(tab)
  })
}

function registerTabCustomizationHandlers(button, tab) {
  if (!button) return
  button.addEventListener('contextmenu', (event) => {
    event.preventDefault()
    handleTabCustomizationRequest(tab, event)
  })

  button.addEventListener('pointerdown', (event) => {
    if (event.pointerType === 'mouse') return
    let longPressTriggered = false
    const timer = window.setTimeout(() => {
      longPressTriggered = true
      handleTabCustomizationRequest(tab, event)
    }, TAB_LONG_PRESS_DELAY)

    const cancel = () => {
      window.clearTimeout(timer)
      button.removeEventListener('pointerup', cancel)
      button.removeEventListener('pointerleave', cancel)
      button.removeEventListener('pointercancel', cancel)
      if (longPressTriggered) {
        button.addEventListener('click', suppressNextClick, { once: true, capture: true })
      }
    }

    button.addEventListener('pointerup', cancel)
    button.addEventListener('pointerleave', cancel)
    button.addEventListener('pointercancel', cancel)
  })
}

function handleTabCustomizationRequest(tab, triggerEvent) {
  if (!tab) return
  let anchor
  if (triggerEvent && typeof triggerEvent.clientX === 'number' && typeof triggerEvent.clientY === 'number') {
    anchor = { x: triggerEvent.clientX, y: triggerEvent.clientY }
  } else if (tab.domButton) {
    const rect = tab.domButton.getBoundingClientRect()
    anchor = { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 }
  } else {
    anchor = { x: window.innerWidth / 2, y: window.innerHeight / 2 }
  }
  setActiveTab(tab.id)
  openTabCustomization(tab, anchor)
  if (callbacks.onTabCustomizationOpened) {
    callbacks.onTabCustomizationOpened(tab)
  }
}

function handleTabContextSubmit() {
  const tabId = state.tabMenu.tabId
  const tab = tabId ? findTab(tabId) : null
  if (!tab) {
    closeTabCustomization()
    return
  }

  const inputValue = elements.tabContextName?.value?.trim() || ''
  const colorId = state.tabMenu.selectedColor && TAB_COLOR_MAP[state.tabMenu.selectedColor]
    ? state.tabMenu.selectedColor
    : TAB_COLOR_DEFAULT

  tab.label = inputValue || tab.defaultLabel || tab.label
  tab.colorId = colorId

  if (tab.domLabel) {
    tab.domLabel.textContent = tab.label
  }
  if (tab.domButton) {
    tab.domButton.title = tab.label
  }
  if (tab.domClose) {
    tab.domClose.setAttribute('aria-label', `Close ${tab.label}`)
  }

  applyTabAppearance(tab)
  updateTabButtonState(tab)
  renderTabs()

  if (callbacks.onTabMetadataChanged) {
    callbacks.onTabMetadataChanged(tab)
  }

  closeTabCustomization()
}

function handleTabContextReset() {
  if (!state.tabMenu.open) return
  const tabId = state.tabMenu.tabId
  const tab = tabId ? findTab(tabId) : null
  if (!tab) {
    closeTabCustomization()
    return
  }
  const fallbackLabel = tab.defaultLabel || tab.label
  if (elements.tabContextName) {
    elements.tabContextName.value = fallbackLabel
  }
  state.tabMenu.selectedColor = TAB_COLOR_DEFAULT
  selectTabColor(TAB_COLOR_DEFAULT)
  if (callbacks.onTabMetadataChanged) {
    callbacks.onTabMetadataChanged(tab)
  }
}

function suppressNextClick(event) {
  event.preventDefault()
  event.stopImmediatePropagation()
}

export function openTabCustomization(tab, anchor) {
  if (!elements.tabContextMenu || !elements.tabContextBackdrop || !elements.tabContextName) return

  if (tabMenuCloseTimer) {
    window.clearTimeout(tabMenuCloseTimer)
    tabMenuCloseTimer = null
  }

  const option = TAB_COLOR_MAP[tab.colorId] ? tab.colorId : TAB_COLOR_DEFAULT

  state.tabMenu.open = true
  state.tabMenu.tabId = tab.id
  state.tabMenu.selectedColor = option
  state.tabMenu.anchor = anchor || { x: window.innerWidth / 2, y: window.innerHeight / 2 }

  elements.tabContextName.value = tab.label
  selectTabColor(state.tabMenu.selectedColor)

  elements.tabContextBackdrop.classList.remove('hidden')
  elements.tabContextMenu.classList.remove('hidden')
  elements.tabContextMenu.style.left = '0px'
  elements.tabContextMenu.style.top = '0px'

  requestAnimationFrame(() => {
    elements.tabContextBackdrop.classList.add('active')
    elements.tabContextBackdrop.setAttribute('aria-hidden', 'false')
    positionTabContextMenu(state.tabMenu.anchor)
    elements.tabContextMenu.classList.add('active')
    elements.tabContextMenu.setAttribute('aria-hidden', 'false')
    try {
      elements.tabContextName.focus({ preventScroll: true })
    } catch (_error) {
      elements.tabContextName.focus()
    }
    elements.tabContextName.select()
  })
}

export function closeTabCustomization() {
  if (!state.tabMenu.open) return
  state.tabMenu.open = false
  state.tabMenu.tabId = null
  state.tabMenu.anchor = { x: 0, y: 0 }
  state.tabMenu.selectedColor = TAB_COLOR_DEFAULT

  if (!elements.tabContextMenu || !elements.tabContextBackdrop) return

  elements.tabContextMenu.classList.remove('active')
  elements.tabContextMenu.setAttribute('aria-hidden', 'true')
  elements.tabContextBackdrop.classList.remove('active')
  elements.tabContextBackdrop.setAttribute('aria-hidden', 'true')

  tabMenuCloseTimer = window.setTimeout(() => {
    elements.tabContextMenu?.classList.add('hidden')
    elements.tabContextBackdrop?.classList.add('hidden')
    tabMenuCloseTimer = null
    if (callbacks.onTabCustomizationClosed) {
      callbacks.onTabCustomizationClosed()
    }
  }, 180)
}

export function positionTabContextMenu(anchor) {
  if (!elements.tabContextMenu) return
  const menu = elements.tabContextMenu
  const { x, y } = anchor || { x: window.innerWidth / 2, y: window.innerHeight / 2 }

  const padding = 16
  const rect = menu.getBoundingClientRect()

  let left = x
  let top = y

  if (left + rect.width + padding > window.innerWidth) {
    left = window.innerWidth - rect.width - padding
  }
  if (left < padding) {
    left = padding
  }
  if (top + rect.height + padding > window.innerHeight) {
    top = window.innerHeight - rect.height - padding
  }
  if (top < padding) {
    top = padding
  }

  menu.style.left = `${left}px`
  menu.style.top = `${top}px`
}

function selectTabColor(colorId) {
  const option = TAB_COLOR_MAP[colorId] ? colorId : TAB_COLOR_DEFAULT
  state.tabMenu.selectedColor = option
  tabColorButtons.forEach((button, id) => {
    const isSelected = id === option
    button.setAttribute('aria-selected', isSelected ? 'true' : 'false')
    button.classList.toggle('selected', isSelected)
  })
}

export function applyTabAppearance(tab) {
  if (!tab?.domButton) return
  if (!TAB_COLOR_MAP[tab.colorId]) {
    tab.colorId = TAB_COLOR_DEFAULT
  }
  const option = TAB_COLOR_MAP[tab.colorId] || TAB_COLOR_MAP[TAB_COLOR_DEFAULT]
  const styles = option?.styles || TAB_COLOR_MAP[TAB_COLOR_DEFAULT].styles
  if (!styles) return
  const style = tab.domButton.style
  tab.domButton.dataset.tabColor = option?.id || TAB_COLOR_DEFAULT
  style.setProperty('--tab-color-border', styles.border)
  style.setProperty('--tab-color-border-hover', styles.hover)
  style.setProperty('--tab-color-border-active', styles.active)
  style.setProperty('--tab-color-selected-start', styles.selectedStart)
  style.setProperty('--tab-color-selected-end', styles.selectedEnd)
  style.setProperty('--tab-color-glow', styles.glow)
  style.setProperty('--tab-color-label', styles.label)
}

function updateTabSessionIndicator(tab) {
  if (!tab?.domButton) return
  if (!tab.domStatus) {
    tab.domStatus = tab.domButton.querySelector('.tab-status-icon')
  }

  const hasActiveSession = Boolean(tab.session && tab.wasDetached !== true && tab.phase !== 'closed')
  const isStarting = tab.phase === 'creating'
  const isClosing = tab.phase === 'closing'
  const isDetached = !hasActiveSession && (tab.wasDetached || isDetachedTab(tab) || tab.phase === 'closed')

  let sessionState = 'attached'

  if (isStarting) {
    sessionState = 'starting'
  } else if (isClosing) {
    sessionState = 'closing'
  } else if (hasActiveSession) {
    sessionState = 'attached'
  } else if (isDetached) {
    sessionState = 'detached'
  } else {
    sessionState = 'none'
  }

  const stateMeta = TAB_SESSION_STATE_META[sessionState] || TAB_SESSION_STATE_META.attached
  tab.domButton.dataset.sessionState = sessionState
  tab.domButton.dataset.sessionAttached = hasActiveSession ? 'true' : 'false'

  if (tab.domStatus) {
    tab.domStatus.classList.remove(
      'hidden',
      'tab-status-loading',
      'tab-status-closing',
      'tab-status-detached',
      'tab-status-empty'
    )
    tab.domStatus.textContent = ''
    tab.domStatus.innerHTML = ''

    if (stateMeta.icon) {
      const lucide = typeof window !== 'undefined' ? window.lucide : null
      const iconElement = document.createElement('i')
      iconElement.dataset.lucide = stateMeta.icon
      iconElement.setAttribute('aria-hidden', 'true')
      tab.domStatus.appendChild(iconElement)
      if (lucide && typeof lucide.createIcons === 'function') {
        lucide.createIcons({ root: tab.domStatus })
      }

      if (stateMeta.extraClass) {
        tab.domStatus.classList.add(stateMeta.extraClass)
      }
      tab.domStatus.classList.remove('hidden')

      const tooltip = stateMeta.label || ''
      if (tooltip) {
        tab.domStatus.title = tooltip
      } else {
        tab.domStatus.removeAttribute('title')
      }
    } else {
      tab.domStatus.classList.add('hidden')
      tab.domStatus.removeAttribute('title')
    }
  }

  const baseLabel = tab.label || ''
  const stateLabel = stateMeta.label || ''
  if (baseLabel) {
    const titleText = stateLabel ? `${baseLabel} (${stateLabel})` : baseLabel
    tab.domButton.title = titleText
    tab.domButton.setAttribute('aria-label', stateLabel ? `${baseLabel} – ${stateLabel}` : baseLabel)
  } else {
    tab.domButton.title = stateLabel
    if (stateLabel) {
      tab.domButton.setAttribute('aria-label', stateLabel)
    } else {
      tab.domButton.removeAttribute('aria-label')
    }
  }
}

function updateTabButtonState(tab) {
  if (!tab.domButton) return
  tab.domButton.setAttribute('aria-selected', tab.id === state.activeTabId ? 'true' : 'false')
  tab.domButton.dataset.phase = tab.phase
  tab.domButton.dataset.socketState = tab.socketState
  tab.domButton.classList.toggle('tab-running', tab.phase === 'running')
  tab.domButton.classList.toggle('tab-error', tab.socketState === 'error')
  if (tab.domClose) {
    tab.domClose.disabled = false
    tab.domClose.removeAttribute('disabled')
  }
  updateTabSessionIndicator(tab)
  applyTabAppearance(tab)
}

export function refreshTabButton(tab) {
  updateTabButtonState(tab)
}

export function isDetachedTab(tab) {
  if (!tab || tab.session) {
    return false
  }
  if (tab.phase === 'creating' || tab.phase === 'running') {
    return false
  }
  if (tab.wasDetached) {
    return true
  }
  if (tab.phase === 'closed') {
    return true
  }
  if (tab.events && tab.events.length > 0) {
    return true
  }
  if (tab.socketState !== 'open' && tab.socketState !== 'connecting' && tab.transcript && tab.transcript.length > 0) {
    return true
  }
  return false
}

export function getDetachedTabs() {
  return state.tabs.filter((tab) => isDetachedTab(tab))
}

export function getTabs() {
  return state.tabs.slice()
}
