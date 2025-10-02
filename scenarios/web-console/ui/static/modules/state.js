/**
 * Global state and constants for web-console
 */

// DOM elements
export const elements = {
  sessionId: document.getElementById('sessionId'),
  sessionPhase: document.getElementById('sessionPhase'),
  socketState: document.getElementById('socketState'),
  sessionCommand: document.getElementById('sessionCommand'),
  transcriptSize: document.getElementById('transcriptSize'),
  eventFeed: document.getElementById('eventFeed'),
  eventMeta: document.getElementById('eventMeta'),
  errorBanner: document.getElementById('errorBanner'),
  tabList: document.getElementById('tabList'),
  addTabBtn: document.getElementById('addTabBtn'),
  tabAddSlot: document.getElementById('tabAddSlot'),
  terminalHost: document.getElementById('terminalHost'),
  layout: document.getElementById('mainLayout'),
  drawerToggle: document.getElementById('drawerToggle'),
  drawerClose: document.getElementById('drawerClose'),
  drawerIndicator: document.getElementById('drawerIndicator'),
  detailsDrawer: document.getElementById('detailsDrawer'),
  drawerBackdrop: document.getElementById('drawerBackdrop'),
  tabContextMenu: document.getElementById('tabContextMenu'),
  tabContextForm: document.getElementById('tabContextForm'),
  tabContextName: document.getElementById('tabContextName'),
  tabContextColors: document.getElementById('tabContextColors'),
  tabContextCancel: document.getElementById('tabContextCancel'),
  tabContextReset: document.getElementById('tabContextReset'),
  tabContextBackdrop: document.getElementById('tabContextBackdrop'),
  signOutAllSessions: document.getElementById('signOutAllSessions')
}

// Terminal defaults
export const terminalDefaults = {
  convertEol: true,
  cursorBlink: true,
  fontFamily: 'JetBrains Mono, SFMono-Regular, Menlo, monospace',
  fontSize: 14,
  theme: {
    background: '#0f172a',
    foreground: '#e2e8f0',
    cursor: '#38bdf8'
  }
}

// Tab color options
export const TAB_COLOR_OPTIONS = [
  {
    id: 'sky',
    label: 'Sky',
    swatch: '#38bdf8',
    styles: {
      border: 'rgba(56, 189, 248, 0.28)',
      hover: 'rgba(56, 189, 248, 0.45)',
      active: 'rgba(56, 189, 248, 0.65)',
      selectedStart: 'rgba(56, 189, 248, 0.35)',
      selectedEnd: 'rgba(14, 116, 144, 0.38)',
      glow: 'rgba(56, 189, 248, 0.35)',
      label: '#f8fafc'
    }
  },
  {
    id: 'emerald',
    label: 'Emerald',
    swatch: '#34d399',
    styles: {
      border: 'rgba(52, 211, 153, 0.26)',
      hover: 'rgba(16, 185, 129, 0.42)',
      active: 'rgba(16, 185, 129, 0.62)',
      selectedStart: 'rgba(16, 185, 129, 0.32)',
      selectedEnd: 'rgba(5, 150, 105, 0.38)',
      glow: 'rgba(16, 185, 129, 0.32)',
      label: '#ecfeff'
    }
  },
  {
    id: 'amber',
    label: 'Amber',
    swatch: '#f59e0b',
    styles: {
      border: 'rgba(245, 158, 11, 0.32)',
      hover: 'rgba(245, 158, 11, 0.48)',
      active: 'rgba(245, 158, 11, 0.68)',
      selectedStart: 'rgba(251, 191, 36, 0.35)',
      selectedEnd: 'rgba(217, 119, 6, 0.42)',
      glow: 'rgba(245, 158, 11, 0.35)',
      label: '#fffbeb'
    }
  },
  {
    id: 'sunset',
    label: 'Sunset',
    swatch: '#f97316',
    styles: {
      border: 'rgba(249, 115, 22, 0.32)',
      hover: 'rgba(249, 115, 22, 0.48)',
      active: 'rgba(249, 115, 22, 0.7)',
      selectedStart: 'rgba(251, 146, 60, 0.36)',
      selectedEnd: 'rgba(194, 65, 12, 0.38)',
      glow: 'rgba(249, 115, 22, 0.36)',
      label: '#fff7ed'
    }
  },
  {
    id: 'violet',
    label: 'Violet',
    swatch: '#a855f7',
    styles: {
      border: 'rgba(168, 85, 247, 0.32)',
      hover: 'rgba(139, 92, 246, 0.5)',
      active: 'rgba(139, 92, 246, 0.7)',
      selectedStart: 'rgba(196, 181, 253, 0.36)',
      selectedEnd: 'rgba(126, 58, 242, 0.42)',
      glow: 'rgba(168, 85, 247, 0.38)',
      label: '#f5f3ff'
    }
  },
  {
    id: 'slate',
    label: 'Slate',
    swatch: '#64748b',
    styles: {
      border: 'rgba(100, 116, 139, 0.32)',
      hover: 'rgba(100, 116, 139, 0.5)',
      active: 'rgba(100, 116, 139, 0.7)',
      selectedStart: 'rgba(148, 163, 184, 0.32)',
      selectedEnd: 'rgba(71, 85, 105, 0.48)',
      glow: 'rgba(100, 116, 139, 0.35)',
      label: '#f8fafc'
    }
  }
]

export const TAB_COLOR_DEFAULT = TAB_COLOR_OPTIONS[0]?.id || 'sky'
export const TAB_COLOR_MAP = TAB_COLOR_OPTIONS.reduce((map, option) => {
  map[option.id] = option
  return map
}, {})

export const TAB_LONG_PRESS_DELAY = 550

// Event aggregation
export const AGGREGATED_EVENT_TYPES = new Set(['ws-empty-frame'])
export const SUPPRESSED_EVENT_LABELS = {
  'ws-empty-frame': 'Empty frames suppressed'
}

export function initialSuppressedState() {
  return {
    'ws-empty-frame': 0
  }
}

// Application state
export const state = {
  tabs: [],
  activeTabId: null,
  bridge: null,
  workspaceSocket: null,
  workspaceReconnectTimer: null,
  drawer: {
    open: false,
    unreadCount: 0,
    previousFocus: null
  },
  tabMenu: {
    open: false,
    tabId: null,
    selectedColor: TAB_COLOR_DEFAULT,
    anchor: { x: 0, y: 0 }
  }
}

export let tabSequence = 0

export function incrementTabSequence() {
  return ++tabSequence
}

// Shortcut buttons
export const shortcutButtons = Array.from(document.querySelectorAll('[data-shortcut-id]'))
